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

	authruntime "github.com/eguilde/egueducation/internal/auth"
	appdb "github.com/eguilde/egueducation/internal/db"
	"github.com/eguilde/egueducation/internal/httpx"
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
	// Migration 0098 rejects any newly-created ownerless portfolio and accepts
	// an explicit identity binding only when that user belongs to the same
	// tenant/institution. This is the database backstop for the admin create
	// command, independent of a caller's JSON payload.
	_, err := adminPool.Exec(ctx, `
		insert into education_portfolios (
			portfolio_code, owner_name, owner_role, school_year, status, section_count,
			last_updated_on, retention_until, transfer_status, institution_id
		) values ('IT-PORT-OWNERLESS', 'Invalid', 'Profesor', '2030-2031', 'draft', 0, current_date, current_date + 365, 'none', $1)
	`, fixture.institutionA)
	if err == nil {
		t.Fatal("new ownerless portfolio must be rejected by the database trigger")
	}
	if _, err := adminPool.Exec(ctx, `
		insert into education_portfolios (
			portfolio_code, owner_user_id, owner_name, owner_role, school_year, status, section_count,
			last_updated_on, retention_until, transfer_status, institution_id
		) values ('IT-PORT-OWNER-BOUND', $1::uuid, 'Governance Integration Member', 'Profesor', '2030-2031', 'draft', 0, current_date, current_date + 365, 'none', $2)
	`, fixture.memberUserID, fixture.institutionA); err != nil {
		t.Fatalf("new explicitly owner-bound portfolio must be accepted: %v", err)
	}
	storedArchiveID := seedGovernancePortfolioArchiveAttachments(t, ctx, adminPool, fixture.institutionA, fixture.memberUserID)
	service := NewService(appdb.NewSessionPool(it.readerPool))

	ctxA, releaseA := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	releasedA := false
	defer func() {
		if !releasedA {
			releaseA()
		}
	}()
	requestA := requestWithContext(ctxA, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	attachments, total, err := service.listPortfolioArchiveAttachments(requestA, fixture.institutionA, fixture.memberUserID, httpx.PageQuery{Page: 1, PageSize: 25, Sort: "title", Direction: "asc"})
	if err != nil {
		t.Fatalf("list own portfolio archive attachments in tenant A: %v", err)
	}
	if total != 1 || len(attachments) != 1 || attachments[0].ID != storedArchiveID || attachments[0].Title != "Eligible portfolio evidence" || attachments[0].CurrentVersionNo != 1 {
		t.Fatalf("attachment picker must return only current stored tenant evidence: total=%d items=%#v", total, attachments)
	}
	requestA.URL.RawQuery = "page=1&pageSize=25&sort=document_title&direction=asc"
	grantListResponse := httptest.NewRecorder()
	service.PortfolioArchiveAttachmentGrants(grantListResponse, requestA)
	requestA.URL.RawQuery = ""
	if grantListResponse.Code != http.StatusOK {
		t.Fatalf("tenant-scoped portfolio attachment grant list failed: status=%d body=%s", grantListResponse.Code, grantListResponse.Body.String())
	}
	attachments, total, err = service.listPortfolioArchiveAttachments(requestA, fixture.institutionA, fixture.foreignMemberUserID, httpx.PageQuery{Page: 1, PageSize: 25, Sort: "title", Direction: "asc"})
	if err != nil || total != 0 || len(attachments) != 0 {
		t.Fatalf("same-tenant user without explicit attachment grant must see no archive evidence: total=%d items=%#v err=%v", total, attachments, err)
	}
	eligibleUsers, eligibleTotal, err := service.listPortfolioArchiveEligibleUsers(requestA, fixture.institutionA, httpx.PageQuery{Page: 1, PageSize: 25, Sort: "name", Direction: "asc", Filters: map[string]string{"name": "Governance Integration"}})
	if err != nil || eligibleTotal != 1 || len(eligibleUsers) != 1 || eligibleUsers[0].ID != fixture.memberUserID {
		t.Fatalf("eligible users must be limited to active tenant-A memberships: total=%d users=%#v err=%v", eligibleTotal, eligibleUsers, err)
	}
	actorID, err := service.currentActorUserID(requestA, fixture.memberSubject)
	if err != nil {
		t.Fatalf("resolve actor UUID in own tenant: %v", err)
	}
	if actorID != fixture.memberUserID || actorID == fixture.memberSubject {
		t.Fatalf("subject %q resolved to %q, want immutable UUID %q", fixture.memberSubject, actorID, fixture.memberUserID)
	}
	// A portfolio owned by the same immutable user is accessible through the
	// own-portfolio policy. The following checks exercise the real RLS-bound
	// query rather than a handler-only permission shortcut.
	if _, allowed, err := service.requireOwnPortfolio(requestA, fixture.portfolioID, portfolioReadOwnPermission); err != nil || !allowed {
		t.Fatalf("owner must read own portfolio: allowed=%t err=%v", allowed, err)
	}
	if _, allowed, err := service.requireOwnPortfolio(requestA, fixture.foreignPortfolioID, portfolioReadOwnPermission); err != nil || allowed {
		t.Fatalf("teacher must not read another teacher portfolio: allowed=%t err=%v", allowed, err)
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
	var cockpitGovernance DirectorCockpitGovernance
	if err := service.loadDirectorCockpitGovernance(requestA, fixture.institutionA, "2026-2027", &cockpitGovernance); err != nil {
		t.Fatalf("load director cockpit governance metrics: %v", err)
	}
	var cockpitPortfolios DirectorCockpitPortfolios
	if err := service.loadDirectorCockpitPortfolios(requestA, fixture.institutionA, "2026-2027", &cockpitPortfolios); err != nil {
		t.Fatalf("load director cockpit portfolio metrics: %v", err)
	}
	var cockpitEvaluations DirectorCockpitEvaluations
	if err := service.loadDirectorCockpitEvaluations(requestA, fixture.institutionA, "2026-2027", &cockpitEvaluations); err != nil {
		t.Fatalf("load director cockpit evaluation metrics: %v", err)
	}
	var cockpitManagerial DirectorCockpitManagerial
	if err := service.loadDirectorCockpitManagerial(requestA, fixture.institutionA, "2026-2027", &cockpitManagerial); err != nil {
		t.Fatalf("load director cockpit managerial metrics: %v", err)
	}
	var cockpitPersonnel DirectorCockpitPersonnel
	if err := service.loadDirectorCockpitPersonnel(requestA, fixture.institutionA, "2026-2027", &cockpitPersonnel); err != nil {
		t.Fatalf("load director cockpit personnel metrics: %v", err)
	}
	var cockpitCompliance DirectorCockpitCompliance
	if err := service.loadDirectorCockpitCompliance(requestA, fixture.institutionA, &cockpitCompliance); err != nil {
		t.Fatalf("load director cockpit compliance metrics: %v", err)
	}
	// The restricted pool deliberately has one connection so the next tenant
	// must reuse the same physical connection after session cleanup.
	releaseA()
	releasedA = true

	ctxB, releaseB := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantB, fixture.institutionB, fixture.memberSubject)
	defer releaseB()
	requestB := requestWithContext(ctxB, fixture.tenantB, fixture.institutionB, fixture.memberSubject)
	attachments, total, err = service.listPortfolioArchiveAttachments(requestB, fixture.institutionB, fixture.memberUserID, httpx.PageQuery{Page: 1, PageSize: 25, Sort: "title", Direction: "asc"})
	if err != nil || total != 0 || len(attachments) != 0 {
		t.Fatalf("tenant B must not enumerate tenant-A archive attachments: total=%d items=%#v err=%v", total, attachments, err)
	}
	eligibleUsers, eligibleTotal, err = service.listPortfolioArchiveEligibleUsers(requestB, fixture.institutionB, httpx.PageQuery{Page: 1, PageSize: 25, Sort: "name", Direction: "asc", Filters: map[string]string{"name": "Governance Integration"}})
	if err != nil || eligibleTotal != 0 || len(eligibleUsers) != 0 {
		t.Fatalf("tenant B must not enumerate tenant-A grant users: total=%d users=%#v err=%v", eligibleTotal, eligibleUsers, err)
	}
	foreignActorID, err := service.currentActorUserID(requestB, fixture.memberSubject)
	if err != nil {
		t.Fatalf("resolve cross-tenant actor: %v", err)
	}
	if foreignActorID != "" {
		t.Fatalf("tenant B resolved tenant-A-only subject to %q", foreignActorID)
	}
	if _, allowed, err := service.requireOwnPortfolio(requestB, fixture.portfolioID, portfolioReadOwnPermission); err != nil || allowed {
		t.Fatalf("tenant B must not access tenant-A portfolio: allowed=%t err=%v", allowed, err)
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
	foreignMemberUserID         string
	meetingAID                  string
	portfolioID                 string
	foreignPortfolioID          string
}

func seedGovernanceAuthorizationFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool) governanceAuthorizationFixture {
	t.Helper()
	const tenantA, institutionA = "tenant-egueducation", "inst-001"
	const tenantB, institutionB = "tenant-balotesti", "inst-balotesti"
	memberID := uuid.NewString()
	memberSubject := "governance-member-" + uuid.NewString()
	foreignMemberID := uuid.NewString()
	foreignMemberSubject := "governance-foreign-member-" + uuid.NewString()
	meetingID := uuid.NewString()
	portfolioID := uuid.NewString()
	foreignPortfolioID := uuid.NewString()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin governance authorization fixture: %v", err)
	}
	defer tx.Rollback(ctx)
	// The profile-phone constraint is deferred, so create the profile and its
	// primary global identity in the same transaction. This is the same
	// identity-first contract used by the real OTP flow, not a fixture bypass.
	if _, err := tx.Exec(ctx, `select set_config('app.is_super_admin','true',true), set_config('app.tenant_id',$1,true), set_config('app.institution_id',$2,true)`, tenantA, institutionA); err != nil {
		t.Fatalf("bind governance fixture tenant session: %v", err)
	}
	if _, err := tx.Exec(ctx, `insert into app_users(id, sub, name, email, phone_number, locale, status) values ($1::uuid, $2, 'Governance Integration Member', $3, '+40000000000', 'ro', 'active')`, memberID, memberSubject, memberSubject+"@example.test"); err != nil {
		t.Fatalf("seed UUID-bound governance user: %v", err)
	}
	if _, err := tx.Exec(ctx, `insert into app_user_identities(user_id, identity_type, normalized_value, display_value, is_primary) values ($1::uuid, 'phone', '+40000000000', '+40000000000', true)`, memberID); err != nil {
		t.Fatalf("seed primary phone identity for governance user: %v", err)
	}
	if _, err := tx.Exec(ctx, `insert into app_users(id, sub, name, email, phone_number, locale, status) values ($1::uuid, $2, 'Other Integration Member', $3, '+40000000001', 'ro', 'active')`, foreignMemberID, foreignMemberSubject, foreignMemberSubject+"@example.test"); err != nil {
		t.Fatalf("seed foreign UUID-bound user: %v", err)
	}
	if _, err := tx.Exec(ctx, `insert into app_user_identities(user_id, identity_type, normalized_value, display_value, is_primary) values ($1::uuid, 'phone', '+40000000001', '+40000000001', true)`, foreignMemberID); err != nil {
		t.Fatalf("seed foreign primary phone identity: %v", err)
	}
	if _, err := tx.Exec(ctx, `insert into app_memberships(user_id, tenant_code, position_code, org_unit_code, organization_name, is_primary, active, start_date) values ($1::uuid, $2, 'profesor', 'unit-root', 'Governance Integration School', true, true, current_date)`, memberID, tenantA); err != nil {
		t.Fatalf("seed tenant-A membership: %v", err)
	}
	if _, err := tx.Exec(ctx, `insert into app_memberships(user_id, tenant_code, position_code, org_unit_code, organization_name, is_primary, active, start_date) values ($1::uuid, $2, 'profesor', 'unit-root', 'Governance Integration School', true, true, current_date)`, foreignMemberID, tenantA); err != nil {
		t.Fatalf("seed foreign tenant-A membership: %v", err)
	}
	if _, err := tx.Exec(ctx, `insert into app_user_permissions(user_id, permission_code, tenant_code) values ($1::uuid, 'education.governance.meeting.vote', $2)`, memberID, tenantA); err != nil {
		t.Fatalf("seed tenant-A contextual permission: %v", err)
	}
	if _, err := tx.Exec(ctx, `insert into education_portfolios (id, portfolio_code, owner_user_id, owner_name, owner_role, school_year, status, section_count, last_updated_on, retention_until, transfer_status, institution_id) values ($1::uuid, 'IT-PORT-OWNER', $2::uuid, 'Governance Integration Member', 'Profesor', '2026-2027', 'draft', 0, current_date, current_date + 365, 'none', $3), ($4::uuid, 'IT-PORT-FOREIGN', $5::uuid, 'Other Integration Member', 'Profesor', '2026-2027', 'draft', 0, current_date, current_date + 365, 'none', $3)`, portfolioID, memberID, institutionA, foreignPortfolioID, foreignMemberID); err != nil {
		t.Fatalf("seed identity-bound professional portfolios: %v", err)
	}
	if _, err := tx.Exec(ctx, `insert into education_meetings (id, school_year, organism, title, meeting_type, status, quorum_required, participants_count, meeting_date, institution_id, chairperson, secretary_name, summary) values ($1::uuid, '2026-2027', 'ca', 'Integration meeting', 'ordinary', 'scheduled', 1, 1, current_date, $2, 'Legacy Chair', 'Legacy Secretary', 'RLS fixture')`, meetingID, institutionA); err != nil {
		t.Fatalf("seed tenant-A governance meeting: %v", err)
	}
	if _, err := tx.Exec(ctx, `insert into education_governance_memberships (school_year, organism, full_name, role_name, mandate_from, mandate_to, voting_right, status, institution_id, app_user_id) values ('2026-2027', 'ca', 'Governance Integration Member', 'Membru CA', current_date - 1, current_date + 1, true, 'activ', $1, $2::uuid), ('2026-2027', 'ca', 'Legacy Chair', 'Președinte CA', current_date - 1, current_date + 1, true, 'activ', $1, null)`, institutionA, memberID); err != nil {
		t.Fatalf("seed UUID and legacy governance memberships: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit governance authorization fixture: %v", err)
	}
	return governanceAuthorizationFixture{tenantA: tenantA, institutionA: institutionA, tenantB: tenantB, institutionB: institutionB, memberSubject: memberSubject, memberUserID: memberID, foreignMemberUserID: foreignMemberID, meetingAID: meetingID, portfolioID: portfolioID, foreignPortfolioID: foreignPortfolioID}
}

func seedGovernancePortfolioArchiveAttachments(t *testing.T, ctx context.Context, pool *pgxpool.Pool, institutionID, granteeUserID string) string {
	t.Helper()
	storedID := uuid.NewString()
	noVersionID := uuid.NewString()
	noStorageID := uuid.NewString()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin archive attachment fixture: %v", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `select set_config('app.is_super_admin','true',true), set_config('app.institution_id',$1,true)`, institutionID); err != nil {
		t.Fatalf("bind archive attachment fixture session: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		insert into archive_documents (id, institution_id, title, original_file_name, mime_type, source_kind, status, current_version_no)
		values
			($1::uuid, $4, 'Eligible portfolio evidence', 'eligible.pdf', 'application/pdf', 'upload', 'ready', 1),
			($2::uuid, $4, 'No version', 'no-version.pdf', 'application/pdf', 'upload', 'ready', 1),
			($3::uuid, $4, 'No stored source', 'no-storage.pdf', 'application/pdf', 'upload', 'ready', 1)
	`, storedID, noVersionID, noStorageID, institutionID); err != nil {
		t.Fatalf("seed archive attachment documents: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		insert into archive_document_versions (
			document_id, institution_id, version_no, mime_type, title, bucket_name, object_key, hash_sha256, status,
			source_bucket, source_object_key, source_sha256
		) values
			($1::uuid, $3, 1, 'application/pdf', 'Eligible portfolio evidence', 'archive', 'evidence.pdf', 'hash-evidence', 'active', 'archive', 'evidence.pdf', 'hash-evidence'),
			($2::uuid, $3, 1, 'application/pdf', 'No stored source', '', '', 'hash-empty', 'active', '', '', 'hash-empty')
		`, storedID, noStorageID, institutionID); err != nil {
		t.Fatalf("seed archive attachment versions: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		insert into education_portfolio_archive_attachment_grants (institution_id, archive_document_id, grantee_user_id)
		values ($1, $2::uuid, $3::uuid)
	`, institutionID, storedID, granteeUserID); err != nil {
		t.Fatalf("grant only eligible archive attachment: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit archive attachment fixture: %v", err)
	}
	return storedID
}

func governanceTenantContext(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, institutionID, actorSubject string) (context.Context, func()) {
	t.Helper()
	bound, release, err := appdb.AcquireRequestConn(ctx, pool, appdb.SessionConfig{TenantID: tenantID, InstitutionID: institutionID, ActorSubject: actorSubject})
	if err != nil {
		t.Fatalf("bind restricted governance tenant session: %v", err)
	}
	return bound, release
}

func requestWithContext(ctx context.Context, tenantCode, institutionID, actorSubject string) *http.Request {
	requestContext := authruntime.WithSessionContextForIntegration(ctx, authruntime.SessionContext{
		TenantCode:    tenantCode,
		InstitutionID: institutionID,
		User: authruntime.SessionUser{
			Sub: actorSubject,
		},
	})
	return httptest.NewRequest(http.MethodGet, "http://education.test", nil).WithContext(requestContext)
}
func quoteGovernanceIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}
func quoteGovernanceLiteral(value string) string {
	return `'` + strings.ReplaceAll(value, `'`, `''`) + `'`
}
