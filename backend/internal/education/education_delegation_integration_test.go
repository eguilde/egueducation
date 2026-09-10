//go:build integration

package education

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	appdb "github.com/eguilde/egueducation/internal/db"
)

// TestDirectorAdjunctDelegationIntegration exercises the DB trigger and the
// request-time evaluator through a NOBYPASSRLS connection.  It is deliberately
// independent of HTTP route wiring: a future route cannot weaken these DB
// tenant, membership, state, and evidence constraints.
func TestDirectorAdjunctDelegationIntegration(t *testing.T) {
	it := newGovernanceIntegrationDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	adminPool := openGovernanceIntegrationPool(t, ctx, it.databaseConfig)
	defer adminPool.Close()
	if err := appdb.Migrate(ctx, adminPool); err != nil {
		t.Fatalf("migrate disposable delegation database: %v", err)
	}
	grantGovernanceIntegrationAccess(t, ctx, adminPool, it.roleName)
	fixture := seedGovernanceAuthorizationFixture(t, ctx, adminPool)
	director, adjunct := seedDirectorAdjunctDelegationFixture(t, ctx, adminPool, fixture.tenantA, fixture.institutionA)
	pool := appdb.NewSessionPool(it.readerPool)

	directorCtx, releaseDirector := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, director.subject)
	if _, err := adminPool.Exec(ctx, `update app_memberships set start_date=current_date+1 where user_id=$1::uuid and tenant_code=$2`, adjunct.userID, fixture.tenantA); err != nil {
		releaseDirector()
		t.Fatalf("move adjunct membership into the future: %v", err)
	}
	if _, err := pool.Exec(directorCtx, `
		insert into education_role_delegations (
			tenant_code, institution_id, delegator_user_id, delegate_user_id, permission_code,
			resource_type, offered_by_user_id
		) values ($1, $2, $3::uuid, $4::uuid, 'education.portfolios.transfer', 'institution', $3::uuid)
	`, fixture.tenantA, fixture.institutionA, director.userID, adjunct.userID); err == nil {
		releaseDirector()
		t.Fatal("future adjunct membership must not be eligible for a delegation offer")
	}
	if _, err := adminPool.Exec(ctx, `update app_memberships set start_date=current_date-1 where user_id=$1::uuid and tenant_code=$2`, adjunct.userID, fixture.tenantA); err != nil {
		releaseDirector()
		t.Fatalf("restore adjunct membership start date: %v", err)
	}
	var delegationID string
	err := pool.QueryRow(directorCtx, `
		insert into education_role_delegations (
			tenant_code, institution_id, delegator_user_id, delegate_user_id, permission_code,
			resource_type, valid_from, valid_until, offered_by_user_id, notes
		) values ($1, $2, $3::uuid, $4::uuid, 'education.portfolios.transfer',
			'institution', current_date, current_date + 30, $4::uuid, 'client-forged offerer')
		returning id::text
	`, fixture.tenantA, fixture.institutionA, director.userID, adjunct.userID).Scan(&delegationID)
	if err != nil {
		releaseDirector()
		t.Fatalf("director offers own portfolio-transfer permission: %v", err)
	}
	var offeredBy, status string
	if err := pool.QueryRow(directorCtx, `select offered_by_user_id::text, status from education_role_delegations where id=$1::uuid`, delegationID).Scan(&offeredBy, &status); err != nil {
		releaseDirector()
		t.Fatalf("read offered delegation: %v", err)
	}
	if offeredBy != director.userID || status != "offered" {
		releaseDirector()
		t.Fatalf("offer provenance must be server-authoritative: offered_by=%q status=%q", offeredBy, status)
	}

	// A director cannot escape RLS by naming another tenant/institution, cannot
	// self-delegate, and cannot delegate a permission the director does not own.
	if _, err := pool.Exec(directorCtx, `
		insert into education_role_delegations (
			tenant_code, institution_id, delegator_user_id, delegate_user_id, permission_code,
			resource_type, offered_by_user_id
		) values ($1, $2, $3::uuid, $4::uuid, 'education.portfolios.transfer', 'institution', $3::uuid)
	`, fixture.tenantB, fixture.institutionB, director.userID, adjunct.userID); err == nil {
		releaseDirector()
		t.Fatal("NOBYPASSRLS director must not insert a delegation in a foreign tenant")
	}
	if _, err := pool.Exec(directorCtx, `
		insert into education_role_delegations (
			tenant_code, institution_id, delegator_user_id, delegate_user_id, permission_code,
			resource_type, offered_by_user_id
		) values ($1, $2, $3::uuid, $3::uuid, 'education.portfolios.transfer', 'institution', $3::uuid)
	`, fixture.tenantA, fixture.institutionA, director.userID); err == nil {
		releaseDirector()
		t.Fatal("self-delegation must be rejected")
	}
	if _, err := pool.Exec(directorCtx, `
		insert into education_role_delegations (
			tenant_code, institution_id, delegator_user_id, delegate_user_id, permission_code,
			resource_type, offered_by_user_id
		) values ($1, $2, $3::uuid, $4::uuid, 'education.delegation.it.not_owned', 'institution', $3::uuid)
	`, fixture.tenantA, fixture.institutionA, director.userID, adjunct.userID); err == nil {
		releaseDirector()
		t.Fatal("director must not delegate a permission the director does not hold")
	}
	if _, err := pool.Exec(directorCtx, `
		insert into education_role_delegations (
			tenant_code, institution_id, delegator_user_id, delegate_user_id, permission_code,
			resource_type, resource_id, offered_by_user_id
		) values ($1, $2, $3::uuid, $4::uuid, 'education.portfolios.transfer', 'portfolio', $5::uuid, $3::uuid)
	`, fixture.tenantA, fixture.institutionA, director.userID, adjunct.userID, uuid.NewString()); err == nil {
		releaseDirector()
		t.Fatal("delegation for a nonexistent resource must be rejected by the database contract")
	}
	releaseDirector()

	adjunctCtx, releaseAdjunct := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, adjunct.subject)
	if _, err := pool.Exec(adjunctCtx, `update education_role_delegations set status='accepted' where id=$1::uuid`, delegationID); err != nil {
		releaseAdjunct()
		t.Fatalf("designated adjunct accepts offered delegation: %v", err)
	}
	adjunctRequest := requestWithContext(adjunctCtx, fixture.tenantA, fixture.institutionA, adjunct.subject)
	service := NewService(pool)
	allowed, err := service.authorizeEducationPermission(adjunctRequest, EducationDelegationScope{PermissionCode: "education.portfolios.transfer", ResourceType: "portfolio", ResourceID: uuid.NewString()})
	if err != nil || !allowed {
		releaseAdjunct()
		t.Fatalf("accepted institution delegation must authorize at request time: allowed=%t err=%v", allowed, err)
	}
	releaseAdjunct()
	if _, err := adminPool.Exec(ctx, `update app_memberships set end_date=current_date-1 where user_id=$1::uuid and tenant_code=$2`, adjunct.userID, fixture.tenantA); err != nil {
		t.Fatalf("expire adjunct membership: %v", err)
	}
	expiredMembershipCtx, releaseExpiredMembership := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, adjunct.subject)
	allowed, err = service.authorizeEducationPermission(requestWithContext(expiredMembershipCtx, fixture.tenantA, fixture.institutionA, adjunct.subject), EducationDelegationScope{PermissionCode: "education.portfolios.transfer", ResourceType: "institution"})
	releaseExpiredMembership()
	if err != nil || allowed {
		t.Fatalf("expired membership must disable delegated authorization immediately: allowed=%t err=%v", allowed, err)
	}
	if _, err := adminPool.Exec(ctx, `update app_memberships set end_date=null where user_id=$1::uuid and tenant_code=$2`, adjunct.userID, fixture.tenantA); err != nil {
		t.Fatalf("restore adjunct membership for revocation evidence test: %v", err)
	}
	assertDelegationAllowed := func(label string, want bool) {
		t.Helper()
		requestCtx, release := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, adjunct.subject)
		allowedNow, authorizeErr := service.authorizeEducationPermission(requestWithContext(requestCtx, fixture.tenantA, fixture.institutionA, adjunct.subject), EducationDelegationScope{PermissionCode: "education.portfolios.transfer", ResourceType: "institution"})
		release()
		if authorizeErr != nil || allowedNow != want {
			t.Fatalf("%s delegated authorization: allowed=%t want=%t err=%v", label, allowedNow, want, authorizeErr)
		}
	}

	if _, err := adminPool.Exec(ctx, `update app_memberships set position_code='profesor' where user_id=$1::uuid and tenant_code=$2`, adjunct.userID, fixture.tenantA); err != nil {
		t.Fatalf("remove adjunct position after acceptance: %v", err)
	}
	assertDelegationAllowed("delegate no longer adjunct", false)
	if _, err := adminPool.Exec(ctx, `update app_memberships set position_code='director_adjunct' where user_id=$1::uuid and tenant_code=$2`, adjunct.userID, fixture.tenantA); err != nil {
		t.Fatalf("restore adjunct position: %v", err)
	}

	if _, err := adminPool.Exec(ctx, `update app_memberships set position_code='profesor' where user_id=$1::uuid and tenant_code=$2`, director.userID, fixture.tenantA); err != nil {
		t.Fatalf("remove delegator director position after acceptance: %v", err)
	}
	assertDelegationAllowed("delegator no longer director", false)
	if _, err := adminPool.Exec(ctx, `update app_memberships set position_code='director' where user_id=$1::uuid and tenant_code=$2`, director.userID, fixture.tenantA); err != nil {
		t.Fatalf("restore delegator director position: %v", err)
	}

	if _, err := adminPool.Exec(ctx, `
		delete from app_position_permissions
		where position_code='director' and permission_code='education.portfolios.transfer';
		delete from app_role_permissions
		where permission_code='education.portfolios.transfer'
			and role_code in (select role_code from app_position_roles where position_code='director')
	`); err != nil {
		t.Fatalf("remove delegator effective permission after acceptance: %v", err)
	}
	assertDelegationAllowed("delegator lost delegated permission", false)
	if _, err := adminPool.Exec(ctx, `
		insert into app_position_permissions(position_code, permission_code)
		values ('director','education.portfolios.transfer') on conflict do nothing;
		insert into app_role_permissions(role_code, permission_code)
		select role_code, 'education.portfolios.transfer'
		from app_position_roles where position_code='director'
		on conflict do nothing
	`); err != nil {
		t.Fatalf("restore delegator effective permission: %v", err)
	}
	assertDelegationAllowed("restored delegation prerequisites", true)

	directorCtx, releaseDirector = governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, director.subject)
	if _, err := pool.Exec(directorCtx, `
		update education_role_delegations
		set status='revoked', accepted_by_user_id=$2::uuid, accepted_at='2001-01-01'
		where id=$1::uuid
	`, delegationID, director.userID); err != nil {
		releaseDirector()
		t.Fatalf("delegating director revokes accepted delegation: %v", err)
	}
	var acceptedBy string
	if err := pool.QueryRow(directorCtx, `select accepted_by_user_id::text from education_role_delegations where id=$1::uuid`, delegationID).Scan(&acceptedBy); err != nil || acceptedBy != adjunct.userID {
		releaseDirector()
		t.Fatalf("revocation must preserve acceptance provenance: accepted_by=%q err=%v", acceptedBy, err)
	}
	if _, err := pool.Exec(directorCtx, `delete from education_role_delegations where id=$1::uuid`, delegationID); err == nil {
		releaseDirector()
		t.Fatal("delegation evidence must never be hard-deleted")
	}
	releaseDirector()

	adjunctCtx, releaseAdjunct = governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, adjunct.subject)
	adjunctRequest = requestWithContext(adjunctCtx, fixture.tenantA, fixture.institutionA, adjunct.subject)
	allowed, err = service.authorizeEducationPermission(adjunctRequest, EducationDelegationScope{PermissionCode: "education.portfolios.transfer", ResourceType: "institution"})
	if err != nil || allowed {
		releaseAdjunct()
		t.Fatalf("revoked delegation must stop request-time authorization: allowed=%t err=%v", allowed, err)
	}
	releaseAdjunct()
}

type directorAdjunctDelegationActor struct {
	userID  string
	subject string
	phone   string
}

func seedDirectorAdjunctDelegationFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantCode, institutionID string) (directorAdjunctDelegationActor, directorAdjunctDelegationActor) {
	t.Helper()
	director := directorAdjunctDelegationActor{userID: uuid.NewString(), subject: "delegation-director-" + uuid.NewString(), phone: "+40900000001"}
	adjunct := directorAdjunctDelegationActor{userID: uuid.NewString(), subject: "delegation-adjunct-" + uuid.NewString(), phone: "+40900000002"}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin delegation fixture: %v", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `select set_config('app.is_super_admin','true',true), set_config('app.tenant_id',$1,true), set_config('app.institution_id',$2,true)`, tenantCode, institutionID); err != nil {
		t.Fatalf("bind delegation fixture context: %v", err)
	}
	for _, actor := range []directorAdjunctDelegationActor{director, adjunct} {
		if _, err := tx.Exec(ctx, `
			insert into app_users(id, sub, name, email, phone_number, locale, status)
			values ($1::uuid, $2, $2, $2 || '@example.test', $3, 'ro', 'active')
		`, actor.userID, actor.subject, actor.phone); err != nil {
			t.Fatalf("seed delegation user %q: %v", actor.subject, err)
		}
		if _, err := tx.Exec(ctx, `
			insert into app_user_identities(user_id, identity_type, normalized_value, display_value, is_primary)
			values ($1::uuid, 'phone', $2, $2, true)
		`, actor.userID, actor.phone); err != nil {
			t.Fatalf("seed delegation identity %q: %v", actor.subject, err)
		}
	}
	if _, err := tx.Exec(ctx, `
		insert into app_memberships(user_id, tenant_code, position_code, org_unit_code, organization_name, is_primary, active, start_date)
		values ($1::uuid, $3, 'director', 'unit-root', 'Delegation Integration School', true, true, current_date),
			($2::uuid, $3, 'director_adjunct', 'unit-root', 'Delegation Integration School', true, true, current_date)
	`, director.userID, adjunct.userID, tenantCode); err != nil {
		t.Fatalf("seed director and adjunct active memberships: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		insert into app_permissions(code, label) values ('education.delegation.it.not_owned', 'Delegation integration non-owned permission')
		on conflict (code) do nothing
	`); err != nil {
		t.Fatalf("seed non-owned delegation permission: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit delegation fixture: %v", err)
	}
	return director, adjunct
}
