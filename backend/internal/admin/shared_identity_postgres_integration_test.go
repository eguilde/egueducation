package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/eguilde/egueducation/internal/config"
	appdb "github.com/eguilde/egueducation/internal/db"
	"github.com/eguilde/egueducation/internal/tenant"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestTenantAdminCannotMutateSharedGlobalIdentityRegression is intentionally
// executed through the HTTP handler with a NOINHERIT/NOBYPASSRLS PostgreSQL
// role.  An app-user who is active in tenant A and tenant B owns one global
// profile; a tenant-A administrator may manage A's grants, but must never
// change its global email, phone, or status.  This catches the historical bug
// where the shared-user check read app_memberships through tenant-A RLS and
// therefore could not see the B membership.
func TestTenantAdminCannotMutateSharedGlobalIdentityRegression(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("EGUEDUCATION_TEST_DATABASE_URL"))
	if databaseURL == "" {
		databaseURL = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	}
	if databaseURL == "" {
		t.Skip("shared identity PostgreSQL integration requires TEST_DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	adminPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect integration administrator database: %v", err)
	}
	t.Cleanup(adminPool.Close)
	if err := appdb.Migrate(ctx, adminPool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	roleName := "shared_identity_app_" + strings.ReplaceAll(uuid.NewString()[:12], "-", "")
	rolePassword := uuid.NewString()
	if _, err := adminPool.Exec(ctx, "create role "+sharedIdentityQuoteIdentifier(roleName)+" login nosuperuser nocreatedb nocreaterole noinherit nobypassrls password "+sharedIdentityQuoteLiteral(rolePassword)); err != nil {
		t.Fatalf("create non-bypass application role: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		if _, err := adminPool.Exec(cleanupCtx, "drop role if exists "+sharedIdentityQuoteIdentifier(roleName)); err != nil {
			t.Errorf("remove non-bypass application role: %v", err)
		}
	})
	quotedRole := sharedIdentityQuoteIdentifier(roleName)
	for _, statement := range []string{
		"grant usage on schema public to " + quotedRole,
		"grant select, insert, update, delete on all tables in schema public to " + quotedRole,
		"grant usage, select on all sequences in schema public to " + quotedRole,
		"grant execute on all functions in schema public to " + quotedRole,
	} {
		if _, err := adminPool.Exec(ctx, statement); err != nil {
			t.Fatalf("grant non-bypass application role: %v", err)
		}
	}

	appConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse application integration database URL: %v", err)
	}
	appConfig.ConnConfig.User = roleName
	appConfig.ConnConfig.Password = rolePassword
	appConfig.MaxConns = 1
	appPool, err := pgxpool.NewWithConfig(ctx, appConfig)
	if err != nil {
		t.Fatalf("connect non-bypass application database role: %v", err)
	}
	t.Cleanup(appPool.Close)
	if err := appPool.Ping(ctx); err != nil {
		t.Fatalf("ping non-bypass application database role: %v", err)
	}

	userID := uuid.New()
	marker := strings.ReplaceAll(userID.String(), "-", "")
	originalEmail := "shared-" + marker + "@example.test"
	originalPhone := "+407" + fmt.Sprintf("%08d", time.Now().UnixNano()%100000000)
	seedSharedIdentity(t, ctx, adminPool, userID, "shared-identity-"+marker, originalEmail, originalPhone)
	t.Cleanup(func() { cleanupSharedIdentity(t, adminPool, userID) })

	// The resolver is part of the production request path.  It runs its own
	// directory query, while the handler's actual mutations use appPool below.
	tenant.ConfigureResolver(adminPool, tenant.ResolverOptions{Environment: "test", BaseDomain: "egueducation.test"})
	service := NewService(config.Config{CustomerName: "EguEducation"}, appdb.NewSessionPool(appPool))
	requestContext, release, err := appdb.AcquireRequestConn(ctx, appPool, appdb.SessionConfig{
		TenantID: "tenant-egueducation", InstitutionID: "inst-001", InstitutionName: "EguEducation", ActorSubject: "tenant-a-admin", IsSuperAdmin: false,
	})
	if err != nil {
		t.Fatalf("bind non-bypass tenant-A request connection: %v", err)
	}
	defer release()

	payload := UpsertUserRequest{
		ID: userID.String(), Name: "Hijacked by tenant A", Email: "hijacked-" + marker + "@example.test",
		Phone: "+407" + fmt.Sprintf("%08d", (time.Now().UnixNano()+1)%100000000), Locale: "ro", Status: "suspended", PreferredOTPChannel: "sms",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal tenant-A mutation request: %v", err)
	}
	request := httptest.NewRequest(http.MethodPut, "https://egueducation.egueducation.test/api/admin/users/"+userID.String(), bytes.NewReader(body)).WithContext(requestContext)
	request.Host = "egueducation.egueducation.test"
	recorder := httptest.NewRecorder()
	service.UpsertUser(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("tenant-A mutation of shared global identity returned %d body=%s, want 403", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"shared_identity_platform_admin_required"`) {
		t.Fatalf("tenant-A mutation response = %s, want shared_identity_platform_admin_required", recorder.Body.String())
	}

	var email, phone, status string
	if err := adminPool.QueryRow(ctx, `select email, phone_number, status from app_users where id=$1`, userID).Scan(&email, &phone, &status); err != nil {
		t.Fatalf("read shared global profile after rejected mutation: %v", err)
	}
	if email != originalEmail || phone != originalPhone || status != "active" {
		t.Fatalf("tenant-A mutation changed global profile email=%q phone=%q status=%q", email, phone, status)
	}
}

func seedSharedIdentity(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, subject, email, phone string) {
	t.Helper()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin shared identity fixture: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, `select set_config('app.tenant_id','tenant-egueducation',true), set_config('app.institution_id','inst-001',true), set_config('app.is_super_admin','true',true)`); err != nil {
		t.Fatalf("scope shared identity fixture: %v", err)
	}
	if _, err := tx.Exec(ctx, `insert into app_users(id,sub,name,email,phone_number,phone_number_verified,status,preferred_otp_channel) values($1,$2,'Shared identity fixture',$3,'',false,'active','sms')`, userID, subject, email); err != nil {
		t.Fatalf("insert shared identity user: %v", err)
	}
	for _, identity := range []struct{ kind, value string }{{"email", email}, {"phone", phone}} {
		if _, err := tx.Exec(ctx, `insert into app_user_identities(user_id,identity_type,normalized_value,display_value,is_primary) values($1,$2,$3,$3,true)`, userID, identity.kind, identity.value); err != nil {
			t.Fatalf("insert shared %s identity: %v", identity.kind, err)
		}
	}
	if _, err := tx.Exec(ctx, `update app_users set phone_number=$2 where id=$1`, userID, phone); err != nil {
		t.Fatalf("project shared identity phone: %v", err)
	}
	for _, membership := range []struct{ tenantCode, orgUnit string }{{"tenant-egueducation", "unit-root"}, {"tenant-balotesti", "unit-balotesti-root"}} {
		if _, err := tx.Exec(ctx, `insert into app_memberships(user_id,tenant_code,position_code,organization_name,org_unit_code,is_primary,active,start_date) values($1,$2,'registrator','Shared identity integration',$3,false,true,current_date)`, userID, membership.tenantCode, membership.orgUnit); err != nil {
			t.Fatalf("insert %s shared membership: %v", membership.tenantCode, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit shared identity fixture: %v", err)
	}
}

func cleanupSharedIdentity(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Errorf("begin shared identity cleanup: %v", err)
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, `select set_config('app.tenant_id','tenant-egueducation',true), set_config('app.institution_id','inst-001',true), set_config('app.is_super_admin','true',true)`); err != nil {
		t.Errorf("scope shared identity cleanup: %v", err)
		return
	}
	if _, err := tx.Exec(ctx, `delete from app_audit_log where details->>'user_id'=$1::text`, userID); err != nil {
		t.Errorf("remove shared identity audits: %v", err)
		return
	}
	if _, err := tx.Exec(ctx, `delete from app_users where id=$1`, userID); err != nil {
		t.Errorf("remove shared identity user: %v", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		t.Errorf("commit shared identity cleanup: %v", err)
	}
}

func sharedIdentityQuoteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}
func sharedIdentityQuoteLiteral(value string) string {
	return `'` + strings.ReplaceAll(value, `'`, `''`) + `'`
}
