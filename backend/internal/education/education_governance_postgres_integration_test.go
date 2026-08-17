//go:build integration

package education

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	appdb "github.com/eguilde/egueducation/internal/db"
)

// TestGovernanceImmutableActorIdentityIntegration uses a database created only
// for this test. It proves real subject-to-UUID lookup, tenant RLS, and the
// fail-closed handling of legacy display-name-only memberships.
func TestGovernanceImmutableActorIdentityIntegration(t *testing.T) {
	it := newGovernanceIntegrationDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	adminPool := openGovernanceIntegrationPool(t, ctx, it.databaseConfig)
	defer adminPool.Close()
	if err := appdb.Migrate(ctx, adminPool); err != nil {
		t.Fatalf("migrate disposable governance database: %v", err)
	}
	if err := appdb.ValidateSchemaContract(ctx, adminPool); err != nil {
		t.Fatalf("validate disposable governance schema: %v", err)
	}
	grantGovernanceIntegrationAccess(t, ctx, adminPool, it.roleName)
	fixture := seedGovernanceAuthorizationFixture(t, ctx, adminPool)
	service := NewService(appdb.NewSessionPool(it.readerPool))

	ctxA, releaseA := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	releasedA := false
	defer func() {
		if !releasedA {
			releaseA()
		}
	}()
	requestA := requestWithContext(ctxA)
	actorID, err := service.currentActorUserID(requestA, fixture.memberSubject)
	if err != nil {
		t.Fatalf("resolve actor UUID in own tenant: %v", err)
	}
	if actorID != fixture.memberUserID || actorID == fixture.memberSubject {
		t.Fatalf("subject %q resolved to %q, want immutable UUID %q", fixture.memberSubject, actorID, fixture.memberUserID)
	}
	allowed, err := service.currentSubjectHasPermission(requestA, fixture.memberSubject, "education.governance.meeting.vote")
	if err != nil || !allowed {
		t.Fatalf("tenant-A contextual permission = %t, %v; want true, nil", allowed, err)
	}
	meeting, err := service.loadGovernanceMeetingAccessContext(requestA, fixture.meetingAID)
	if err != nil {
		t.Fatalf("load own governance meeting: %v", err)
	}
	memberships, err := service.governanceMembershipAccess(requestA, meeting)
	if err != nil {
		t.Fatalf("load own governance memberships: %v", err)
	}
	if !governanceActorAllowed(actorID, meeting, memberships, governanceMeetingActorRule{RequireVotingRight: true}) {
		t.Fatal("active UUID-bound voting membership must authorize voting")
	}
	if governanceActorAllowed(actorID, meeting, memberships, governanceMeetingActorRule{MembershipRoleHints: []string{"presedinte"}}) {
		t.Fatal("legacy display-name-only president membership must not authorize")
	}
	allowed, err = service.authorizeGovernanceMeetingActionForSubject(requestA, fixture.memberSubject, fixture.meetingAID, "education.governance.meeting.vote", governanceMeetingActorRule{RequireVotingRight: true})
	if err != nil || !allowed {
		t.Fatalf("wired governance authorizer for UUID-bound voter = %t, %v; want true, nil", allowed, err)
	}
	allowed, err = service.authorizeGovernanceMeetingActionForSubject(requestA, fixture.memberSubject, fixture.meetingAID, "education.governance.meeting.vote", governanceMeetingActorRule{MembershipRoleHints: []string{"presedinte"}})
	if err != nil || allowed {
		t.Fatalf("wired governance authorizer for legacy name-only chair = %t, %v; want false, nil", allowed, err)
	}
	// The restricted pool deliberately has one connection so the next tenant
	// must reuse the same physical connection after session cleanup.
	releaseA()
	releasedA = true

	ctxB, releaseB := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantB, fixture.institutionB, fixture.memberSubject)
	defer releaseB()
	requestB := requestWithContext(ctxB)
	foreignActorID, err := service.currentActorUserID(requestB, fixture.memberSubject)
	if err != nil {
		t.Fatalf("resolve cross-tenant actor: %v", err)
	}
	if foreignActorID != "" {
		t.Fatalf("tenant B resolved tenant-A-only subject to %q", foreignActorID)
	}
	allowed, err = service.currentSubjectHasPermission(requestB, fixture.memberSubject, "education.governance.meeting.vote")
	if err != nil || allowed {
		t.Fatalf("tenant-B contextual permission = %t, %v; want false, nil", allowed, err)
	}
	if _, err := service.loadGovernanceMeetingAccessContext(requestB, fixture.meetingAID); !errors.Is(err, errGovernanceMeetingNotFound) {
		t.Fatalf("tenant B must not load tenant-A meeting: %v", err)
	}
	allowed, err = service.authorizeGovernanceMeetingActionForSubject(requestB, fixture.memberSubject, fixture.meetingAID, "education.governance.meeting.vote", governanceMeetingActorRule{RequireVotingRight: true})
	if err != nil || allowed {
		t.Fatalf("wired authorizer cross-tenant result = %t, %v; want false, nil", allowed, err)
	}
}

type governanceIntegrationDatabase struct {
	databaseConfig *pgxpool.Config
	readerPool     *pgxpool.Pool
	roleName       string
}

func newGovernanceIntegrationDatabase(t *testing.T) governanceIntegrationDatabase {
	t.Helper()
	baseURL := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if baseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set; governance PostgreSQL integration test is intentionally skipped")
	}
	ctx := context.Background()
	baseConfig, err := pgx.ParseConfig(baseURL)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}
	admin, err := pgx.ConnectConfig(ctx, baseConfig)
	if err != nil {
		t.Fatalf("connect TEST_DATABASE_URL: %v", err)
	}
	databaseName := "education_governance_it_" + strings.ReplaceAll(uuid.NewString()[:12], "-", "")
	roleName := "education_governance_it_" + strings.ReplaceAll(uuid.NewString()[:12], "-", "")
	rolePassword := uuid.NewString()
	if _, err := admin.Exec(ctx, "create database "+quoteGovernanceIdentifier(databaseName)+" template template0"); err != nil {
		admin.Close(ctx)
		t.Fatalf("create disposable governance database: %v", err)
	}
	if _, err := admin.Exec(ctx, "create role "+quoteGovernanceIdentifier(roleName)+" login nosuperuser nobypassrls password "+quoteGovernanceLiteral(rolePassword)); err != nil {
		_, _ = admin.Exec(ctx, "drop database "+quoteGovernanceIdentifier(databaseName))
		admin.Close(ctx)
		t.Fatalf("create restricted governance role: %v", err)
	}
	admin.Close(ctx)
	targetConfig, err := pgxpool.ParseConfig(baseURL)
	if err != nil {
		t.Fatalf("parse target integration config: %v", err)
	}
	targetConfig.ConnConfig.Database = databaseName
	readerConfig := targetConfig.Copy()
	readerConfig.ConnConfig.User = roleName
	readerConfig.ConnConfig.Password = rolePassword
	readerConfig.MaxConns = 1
	readerPool, err := pgxpool.NewWithConfig(ctx, readerConfig)
	if err != nil {
		t.Fatalf("open restricted governance integration pool: %v", err)
	}
	t.Cleanup(func() {
		readerPool.Close()
		cleanup, cleanupErr := pgx.ConnectConfig(context.Background(), baseConfig)
		if cleanupErr != nil {
			t.Errorf("connect to remove governance integration database: %v", cleanupErr)
			return
		}
		defer cleanup.Close(context.Background())
		if _, err := cleanup.Exec(context.Background(), "drop database if exists "+quoteGovernanceIdentifier(databaseName)+" with (force)"); err != nil {
			t.Errorf("drop disposable governance database: %v", err)
		}
		if _, err := cleanup.Exec(context.Background(), "drop role if exists "+quoteGovernanceIdentifier(roleName)); err != nil {
			t.Errorf("drop restricted governance role: %v", err)
		}
	})
	return governanceIntegrationDatabase{databaseConfig: targetConfig, readerPool: readerPool, roleName: roleName}
}

func openGovernanceIntegrationPool(t *testing.T, ctx context.Context, databaseConfig *pgxpool.Config) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.NewWithConfig(ctx, databaseConfig.Copy())
	if err != nil {
		t.Fatalf("open governance integration database: %v", err)
	}
	return pool
}

func grantGovernanceIntegrationAccess(t *testing.T, ctx context.Context, pool *pgxpool.Pool, roleName string) {
	t.Helper()
	for _, statement := range []string{
		"grant usage on schema public to " + quoteGovernanceIdentifier(roleName),
		"grant select, insert, update, delete on all tables in schema public to " + quoteGovernanceIdentifier(roleName),
		"grant usage, select on all sequences in schema public to " + quoteGovernanceIdentifier(roleName),
		"grant execute on all functions in schema public to " + quoteGovernanceIdentifier(roleName),
	} {
		if _, err := pool.Exec(ctx, statement); err != nil {
			t.Fatalf("grant restricted governance integration access: %v", err)
		}
	}
}

type governanceAuthorizationFixture struct {
	tenantA, institutionA       string
	tenantB, institutionB       string
	memberSubject, memberUserID string
	meetingAID                  string
}

func seedGovernanceAuthorizationFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool) governanceAuthorizationFixture {
	t.Helper()
	const tenantA, institutionA = "tenant-egueducation", "inst-001"
	const tenantB, institutionB = "tenant-balotesti", "inst-balotesti"
	memberID := uuid.NewString()
	memberSubject := "governance-member-" + uuid.NewString()
	meetingID := uuid.NewString()
	if _, err := pool.Exec(ctx, `insert into app_users(id, sub, name, email, phone_number, locale, status) values ($1::uuid, $2, 'Governance Integration Member', $3, '+40000000000', 'ro', 'active')`, memberID, memberSubject, memberSubject+"@example.test"); err != nil {
		t.Fatalf("seed UUID-bound governance user: %v", err)
	}
	if _, err := pool.Exec(ctx, `insert into app_memberships(user_id, tenant_code, position_code, org_unit_code, organization_name, is_primary, active, start_date) values ($1::uuid, $2, 'profesor', 'unit-root', 'Governance Integration School', true, true, current_date)`, memberID, tenantA); err != nil {
		t.Fatalf("seed tenant-A membership: %v", err)
	}
	if _, err := pool.Exec(ctx, `insert into app_user_permissions(user_id, permission_code, tenant_code) values ($1::uuid, 'education.governance.meeting.vote', $2)`, memberID, tenantA); err != nil {
		t.Fatalf("seed tenant-A contextual permission: %v", err)
	}
	if _, err := pool.Exec(ctx, `insert into education_meetings (id, school_year, organism, title, meeting_type, status, quorum_required, participants_count, meeting_date, institution_id, chairperson, secretary_name, summary) values ($1::uuid, '2026-2027', 'ca', 'Integration meeting', 'ordinary', 'scheduled', 1, 1, current_date, $2, 'Legacy Chair', 'Legacy Secretary', 'RLS fixture')`, meetingID, institutionA); err != nil {
		t.Fatalf("seed tenant-A governance meeting: %v", err)
	}
	if _, err := pool.Exec(ctx, `insert into education_governance_memberships (school_year, organism, full_name, role_name, mandate_from, mandate_to, voting_right, status, institution_id, app_user_id) values ('2026-2027', 'ca', 'Governance Integration Member', 'Membru CA', current_date - 1, current_date + 1, true, 'activ', $1, $2::uuid), ('2026-2027', 'ca', 'Legacy Chair', 'Președinte CA', current_date - 1, current_date + 1, true, 'activ', $1, null)`, institutionA, memberID); err != nil {
		t.Fatalf("seed UUID and legacy governance memberships: %v", err)
	}
	return governanceAuthorizationFixture{tenantA, institutionA, tenantB, institutionB, memberSubject, memberID, meetingID}
}

func governanceTenantContext(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, institutionID, actorSubject string) (context.Context, func()) {
	t.Helper()
	bound, release, err := appdb.AcquireRequestConn(ctx, pool, appdb.SessionConfig{TenantID: tenantID, InstitutionID: institutionID, ActorSubject: actorSubject})
	if err != nil {
		t.Fatalf("bind restricted governance tenant session: %v", err)
	}
	return bound, release
}

func requestWithContext(ctx context.Context) *http.Request {
	return httptest.NewRequest(http.MethodGet, "http://education.test", nil).WithContext(ctx)
}
func quoteGovernanceIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}
func quoteGovernanceLiteral(value string) string {
	return `'` + strings.ReplaceAll(value, `'`, `''`) + `'`
}
