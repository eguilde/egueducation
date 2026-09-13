//go:build integration

package education

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	appdb "github.com/eguilde/egueducation/internal/db"
)

// TestSchoolClassesDelegationParity proves that the HTTP helper and the
// FORCE-RLS predicate make the same institution-scoped delegation decision.
// Revoked and expired ledger rows must immediately fail closed without a new
// token or a process restart.
func TestSchoolClassesDelegationParity(t *testing.T) {
	it := newGovernanceIntegrationDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	admin := openGovernanceIntegrationPool(t, ctx, it.databaseConfig)
	defer admin.Close()
	if err := appdb.Migrate(ctx, admin); err != nil {
		t.Fatalf("migrate classes delegation database: %v", err)
	}
	grantGovernanceIntegrationAccess(t, ctx, admin, it.roleName)
	fixture := seedGovernanceAuthorizationFixture(t, ctx, admin)
	director, adjunct := seedDirectorAdjunctDelegationFixture(t, ctx, admin, fixture.tenantA, fixture.institutionA)
	if _, err := admin.Exec(ctx, `
		insert into app_user_permissions(user_id, permission_code, tenant_code)
		values ($1::uuid, 'education.classes.manage', $2), ($1::uuid, 'education.classes.read', $2)
		on conflict do nothing
	`, director.userID, fixture.tenantA); err != nil {
		t.Fatalf("grant director class permissions: %v", err)
	}

	pool := appdb.NewSessionPool(it.readerPool)
	offer := func(permission string, validUntil string) string {
		t.Helper()
		directorCtx, release := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, director.subject)
		defer release()
		var id string
		if err := pool.QueryRow(directorCtx, `
			insert into education_role_delegations(tenant_code, institution_id, delegator_user_id, delegate_user_id, permission_code, resource_type, valid_from, valid_until, offered_by_user_id)
			values($1,$2,$3::uuid,$4::uuid,$5,'institution',current_date,$6::date,$3::uuid)
			returning id::text
		`, fixture.tenantA, fixture.institutionA, director.userID, adjunct.userID, permission, validUntil).Scan(&id); err != nil {
			t.Fatalf("offer %s class delegation: %v", permission, err)
		}
		return id
	}
	accept := func(id string) {
		t.Helper()
		adjunctCtx, release := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, adjunct.subject)
		defer release()
		if _, err := pool.Exec(adjunctCtx, `update education_role_delegations set status='accepted' where id=$1::uuid`, id); err != nil {
			t.Fatalf("accept class delegation: %v", err)
		}
	}
	assertAccess := func(permission string, want bool) {
		t.Helper()
		adjunctCtx, release := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, adjunct.subject)
		defer release()
		service := NewService(pool)
		request := requestWithContext(adjunctCtx, fixture.tenantA, fixture.institutionA, adjunct.subject)
		recorder := httptest.NewRecorder()
		_, all, ok := service.schoolClassesAccessWithWriter(recorder, request, permission == classesManagePermission)
		if ok != want || all != want {
			t.Fatalf("handler delegated %s access: ok=%t all=%t want=%t body=%s", permission, ok, all, want, recorder.Body.String())
		}
		var rlsAllowed bool
		if err := pool.QueryRow(adjunctCtx, `select public.education_classes_actor_has_permission($1)`, permission).Scan(&rlsAllowed); err != nil || rlsAllowed != want {
			t.Fatalf("RLS delegated %s access: allowed=%t want=%t err=%v", permission, rlsAllowed, want, err)
		}
	}

	delegationID := offer(classesManagePermission, "2099-01-01")
	accept(delegationID)
	assertAccess(classesManagePermission, true)

	directorCtx, release := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, director.subject)
	if _, err := pool.Exec(directorCtx, `update education_role_delegations set status='revoked' where id=$1::uuid`, delegationID); err != nil {
		release()
		t.Fatalf("revoke class delegation: %v", err)
	}
	release()
	assertAccess(classesManagePermission, false)

	// The trigger rightly refuses accepting a stale offer. Seed an historical
	// accepted evidence row through the migration owner solely to prove the
	// runtime RLS predicate denies it by validity date.
	expiredID := uuid.NewString()
	seedTx, err := admin.Begin(ctx)
	if err != nil {
		t.Fatalf("begin historical delegation fixture: %v", err)
	}
	defer seedTx.Rollback(ctx)
	// Keep historical-fixture trigger suppression transaction-local so an
	// error cannot leave a pooled connection with replication mode enabled.
	if _, err := seedTx.Exec(ctx, `set local session_replication_role = replica`); err != nil {
		t.Fatalf("configure historical delegation fixture: %v", err)
	}
	if _, err := seedTx.Exec(ctx, `
		insert into education_role_delegations(id,tenant_code,institution_id,delegator_user_id,delegate_user_id,permission_code,resource_type,status,valid_from,valid_until,offered_by_user_id,offered_at,accepted_by_user_id,accepted_at)
		values($1::uuid,$2,$3,$4::uuid,$5::uuid,'education.classes.read','institution','accepted',current_date-10,current_date-1,$4::uuid,now()-interval '10 days',$5::uuid,now()-interval '9 days')
	`, expiredID, fixture.tenantA, fixture.institutionA, director.userID, adjunct.userID); err != nil {
		t.Fatalf("seed expired class delegation evidence: %v", err)
	}
	if err := seedTx.Commit(ctx); err != nil {
		t.Fatalf("commit historical delegation fixture: %v", err)
	}
	assertAccess(classesReadPermission, false)
}
