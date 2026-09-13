//go:build integration

package admission

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/eguilde/egueducation/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestAdmissionWORMIngestionIntentNOBYPASSRLS proves 0155 isolation through
// an ordinary LOGIN NOINHERIT NOBYPASSRLS role, not the fixture superuser.
func TestAdmissionWORMIngestionIntentNOBYPASSRLS(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("requires TEST_DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	admin := newAdmissionExpiryDatabase(t, ctx, dsn)
	if err := db.Migrate(ctx, admin); err != nil {
		t.Fatal(err)
	}
	sessions := db.NewSessionPool(admin)
	adminCtx, releaseAdmin, err := db.AcquireRequestConn(ctx, admin, db.SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: "expiry-regression-actor", IsSuperAdmin: true})
	if err != nil {
		t.Fatal(err)
	}
	defer releaseAdmin()
	applicationID, _, evaluationID := seedAdmissionExpiryGraph(t, adminCtx, sessions)
	seedExpiryRetentionAuthority(t, ctx, admin, sessions)
	authority := loadPreparationRetentionAuthority(t, adminCtx, sessions)
	preparationID := insertRetentionPreparation(t, adminCtx, sessions, applicationID, evaluationID, authority, "")
	var deadline time.Time
	if err = sessions.QueryRow(adminCtx, `select required_retention_until from school_admission_legal_preparations where id=$1::uuid`, preparationID).Scan(&deadline); err != nil {
		t.Fatal(err)
	}
	intentID := uuid.NewString()
	if _, err = sessions.Exec(adminCtx, `insert into archive_ingestion_intents(id,tenant_code,institution_id,actor_subject,idempotency_key,request_fingerprint,purpose,preparation_id,artifact_slot,expected_sha256,expected_size_bytes,reserved_document_id,reserved_version_id,bucket_name,object_key,retention_until) values($1::uuid,'tenant-egueducation','inst-001','expiry-regression-actor','rls-intent',repeat('a',64),'admission_legal_preparation',$2::uuid,'primary',repeat('b',64),10,$3::uuid,$4::uuid,'test','rls/object.pdf',$5)`, intentID, preparationID, uuid.NewString(), uuid.NewString(), deadline); err != nil {
		t.Fatal(err)
	}

	role := "admission_worm_rls_" + strings.ReplaceAll(uuid.NewString()[:12], "-", "")
	password := "pw_" + uuid.NewString()
	quoted := pgx.Identifier{role}.Sanitize()
	if _, err = admin.Exec(ctx, fmt.Sprintf("create role %s login noinherit nobypassrls password '%s'", quoted, password)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if _, cleanupErr := admin.Exec(cleanupCtx, "drop owned by "+quoted); cleanupErr != nil {
			t.Logf("drop owned by %s: %v", role, cleanupErr)
		}
		if _, cleanupErr := admin.Exec(cleanupCtx, "drop role if exists "+quoted); cleanupErr != nil {
			t.Logf("drop role %s: %v", role, cleanupErr)
		}
	})
	for _, statement := range []string{"grant usage on schema public to " + quoted, "grant select,insert,update on archive_ingestion_intents to " + quoted} {
		if _, err = admin.Exec(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	config := admin.Config().Copy()
	config.ConnConfig.User, config.ConnConfig.Password, config.MaxConns = role, password, 1
	app, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Close)
	appSessions := db.NewSessionPool(app)

	for _, scope := range []struct{ name, tenant, institution string }{
		{name: "foreign tenant", tenant: "tenant-balotesti", institution: "inst-001"},
		{name: "foreign institution", tenant: "tenant-egueducation", institution: "inst-002"},
		{name: "foreign tenant and institution", tenant: "tenant-balotesti", institution: "inst-002"},
	} {
		t.Run(scope.name+" cannot read or transition intent", func(t *testing.T) {
			foreignCtx, release, err := db.AcquireRequestConn(ctx, app, db.SessionConfig{TenantID: scope.tenant, InstitutionID: scope.institution, ActorSubject: "foreign-app-user"})
			if err != nil {
				t.Fatal(err)
			}
			defer release()
			var count int
			if err = appSessions.QueryRow(foreignCtx, `select count(*) from archive_ingestion_intents where id=$1::uuid`, intentID).Scan(&count); err != nil || count != 0 {
				t.Fatalf("foreign read count=%d err=%v", count, err)
			}
			if tag, err := appSessions.Exec(foreignCtx, `update archive_ingestion_intents set status='stored',stored_version_id='forged',stored_etag='forged',stored_size_bytes=10,stored_retention_until=retention_until where id=$1::uuid`, intentID); err != nil || tag.RowsAffected() != 0 {
				t.Fatalf("foreign intent transition rows=%d err=%v", tag.RowsAffected(), err)
			}
			_, err = appSessions.Exec(foreignCtx, `insert into archive_ingestion_intents(id,tenant_code,institution_id,actor_subject,idempotency_key,request_fingerprint,purpose,preparation_id,artifact_slot,expected_sha256,expected_size_bytes,reserved_document_id,reserved_version_id,bucket_name,object_key,retention_until) values($1::uuid,'tenant-egueducation','inst-001','expiry-regression-actor',$2,repeat('a',64),'admission_legal_preparation',$3::uuid,'primary',repeat('b',64),10,$4::uuid,$5::uuid,'test','foreign/insert.pdf',$6)`, uuid.NewString(), "cross-scope-"+uuid.NewString(), preparationID, uuid.NewString(), uuid.NewString(), deadline)
			if err == nil || (!strings.Contains(strings.ToLower(err.Error()), "row-level security") && !strings.Contains(strings.ToLower(err.Error()), "context mismatch")) {
				t.Fatalf("cross-scope insert err=%v, want RLS/context denial", err)
			}
		})
	}
	t.Run("own tenant can read", func(t *testing.T) {
		ownCtx, release, err := db.AcquireRequestConn(ctx, app, db.SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: "expiry-regression-actor"})
		if err != nil {
			t.Fatal(err)
		}
		defer release()
		var currentUser string
		var superuser, bypassRLS bool
		if err = appSessions.QueryRow(ownCtx, `select current_user,rolsuper,rolbypassrls from pg_roles where rolname=current_user`).Scan(&currentUser, &superuser, &bypassRLS); err != nil || currentUser != role || superuser || bypassRLS {
			t.Fatalf("application role=%q super=%t bypassrls=%t err=%v", currentUser, superuser, bypassRLS, err)
		}
		var count int
		if err = appSessions.QueryRow(ownCtx, `select count(*) from archive_ingestion_intents where id=$1::uuid`, intentID).Scan(&count); err != nil || count != 1 {
			t.Fatalf("own read count=%d err=%v", count, err)
		}
	})
}
