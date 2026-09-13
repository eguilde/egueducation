//go:build integration

package education

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eguilde/egueducation/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPortfolioOwnAppliedProcedureReadsOnlyBoundPublishedOrSupersededVersion(t *testing.T) {
	it := newGovernanceIntegrationDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	adminPool := openGovernanceIntegrationPool(t, ctx, it.databaseConfig)
	defer adminPool.Close()
	if err := db.Migrate(ctx, adminPool); err != nil {
		t.Fatalf("migrate disposable own-procedure database: %v", err)
	}
	grantGovernanceIntegrationAccess(t, ctx, adminPool, it.roleName)
	fixture := seedGovernanceAuthorizationFixture(t, ctx, adminPool)
	grantPortfolioOwnProcedureReadPermission(t, ctx, adminPool, fixture)

	service := NewService(db.NewSessionPool(it.readerPool))
	ownerCtx, ownerRelease := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	defer ownerRelease()

	legacy := httptest.NewRecorder()
	service.PortfolioOwnAppliedProcedure(legacy, legalWorkflowRequest(ownerCtx, fixture, http.MethodGet, "", map[string]string{"recordID": fixture.portfolioID}))
	assertHandlerCode(t, legacy, http.StatusNotFound, "education_own_portfolio_procedure_not_applied")

	appliedProcedureID := seedPublishedPortfolioProcedure(t, ctx, adminPool, fixture)
	bindPortfolioAppliedProcedure(t, ctx, adminPool, fixture, fixture.portfolioID, appliedProcedureID)
	unrelatedProcedureID := seedAdditionalPublishedPortfolioProcedure(t, ctx, adminPool, fixture, "unrelated-current-procedure", "Procedură mai nouă, neaplicată")

	response := httptest.NewRecorder()
	service.PortfolioOwnAppliedProcedure(response, legalWorkflowRequest(ownerCtx, fixture, http.MethodGet, "", map[string]string{"recordID": fixture.portfolioID}))
	if response.Code != http.StatusOK {
		t.Fatalf("read bound procedure: status=%d body=%s", response.Code, response.Body.String())
	}
	var body OwnPortfolioAppliedProcedureResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode applied procedure response: %v", err)
	}
	if body.Procedure.ID != appliedProcedureID || body.Procedure.ID == unrelatedProcedureID {
		t.Fatalf("response must use the exact applied version, got=%q applied=%q unrelated=%q", body.Procedure.ID, appliedProcedureID, unrelatedProcedureID)
	}
	if body.Procedure.LifecycleStatus != "published" || len(body.Rules) == 0 {
		t.Fatalf("published bound procedure response=%#v", body)
	}
	for _, rule := range body.Rules {
		if rule.ProcedureID != appliedProcedureID {
			t.Fatalf("rule %q belongs to %q, want %q", rule.ID, rule.ProcedureID, appliedProcedureID)
		}
	}
	ownerRelease()

	foreignSubject := subjectForUser(t, ctx, adminPool, fixture.foreignMemberUserID)
	foreignCtx, foreignRelease := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, foreignSubject)
	defer foreignRelease()
	foreignFixture := fixture
	foreignFixture.memberSubject = foreignSubject
	foreign := httptest.NewRecorder()
	service.PortfolioOwnAppliedProcedure(foreign, legalWorkflowRequest(foreignCtx, foreignFixture, http.MethodGet, "", map[string]string{"recordID": fixture.portfolioID}))
	assertHandlerCode(t, foreign, http.StatusForbidden, "education_portfolio_access_denied")
	foreignRelease()

	crossTenantCtx, crossTenantRelease := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantB, fixture.institutionB, fixture.memberSubject)
	defer crossTenantRelease()
	crossTenantFixture := fixture
	crossTenantFixture.tenantA = fixture.tenantB
	crossTenantFixture.institutionA = fixture.institutionB
	crossTenant := httptest.NewRecorder()
	service.PortfolioOwnAppliedProcedure(crossTenant, legalWorkflowRequest(crossTenantCtx, crossTenantFixture, http.MethodGet, "", map[string]string{"recordID": fixture.portfolioID}))
	assertHandlerCode(t, crossTenant, http.StatusForbidden, "education_portfolio_access_denied")
	crossTenantRelease()

	supersedePortfolioProcedure(t, ctx, adminPool, fixture, appliedProcedureID)
	ownerCtx, ownerRelease = governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	defer ownerRelease()
	superseded := httptest.NewRecorder()
	service.PortfolioOwnAppliedProcedure(superseded, legalWorkflowRequest(ownerCtx, fixture, http.MethodGet, "", map[string]string{"recordID": fixture.portfolioID}))
	if superseded.Code != http.StatusOK {
		t.Fatalf("read superseded bound procedure: status=%d body=%s", superseded.Code, superseded.Body.String())
	}
	if err := json.Unmarshal(superseded.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode superseded applied procedure response: %v", err)
	}
	if body.Procedure.ID != appliedProcedureID || body.Procedure.LifecycleStatus != "superseded" {
		t.Fatalf("superseded bound procedure response=%#v", body.Procedure)
	}
}

func grantPortfolioOwnProcedureReadPermission(t *testing.T, ctx context.Context, pool *pgxpool.Pool, fixture governanceAuthorizationFixture) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
		insert into app_user_permissions (user_id, permission_code, tenant_code)
		values
			($1::uuid, 'education.portfolios.read_own', $2),
			($3::uuid, 'education.portfolios.read_own', $2)
		on conflict do nothing
	`, fixture.memberUserID, fixture.tenantA, fixture.foreignMemberUserID); err != nil {
		t.Fatalf("grant own procedure read permissions: %v", err)
	}
}

func bindPortfolioAppliedProcedure(t *testing.T, ctx context.Context, pool *pgxpool.Pool, fixture governanceAuthorizationFixture, portfolioID, procedureID string) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
		update education_portfolios
		set applied_procedure_id = $1::uuid
		where id = $2::uuid and institution_id = $3
	`, procedureID, portfolioID, fixture.institutionA); err != nil {
		t.Fatalf("bind procedure to portfolio: %v", err)
	}
}

func seedAdditionalPublishedPortfolioProcedure(t *testing.T, ctx context.Context, pool *pgxpool.Pool, fixture governanceAuthorizationFixture, procedureCode, title string) string {
	t.Helper()
	procedureID := uuid.NewString()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin additional procedure fixture: %v", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `select set_config('app.is_super_admin','true',true), set_config('app.tenant_id',$1,true), set_config('app.institution_id',$2,true)`, fixture.tenantA, fixture.institutionA); err != nil {
		t.Fatalf("bind additional procedure fixture: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		insert into education_portfolio_procedure_versions (
			id, institution_id, tenant_code, procedure_code, version_no, title, source_ref,
			lifecycle_status, effective_from, created_by_user_id
		) values ($1::uuid, $2, $3, $4, 1, $5, 'Ordinul nr. 3.858/2026',
			'draft', current_date - 1, $6::uuid)
	`, procedureID, fixture.institutionA, fixture.tenantA, procedureCode, title, fixture.memberUserID); err != nil {
		t.Fatalf("seed additional procedure: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		insert into education_portfolio_procedure_section_rules (
			procedure_id, institution_id, tenant_code, section_code, label_ro, label_en,
			source_catalog_version, required, sort_order, active
		)
		select $1::uuid, $2, $3, section_code, label_ro, label_en, catalog_version, true, sort_order, true
		from education_portfolio_sections
		where active and required
		order by sort_order
	`, procedureID, fixture.institutionA, fixture.tenantA); err != nil {
		t.Fatalf("seed additional procedure rules: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		update education_portfolio_procedure_versions
		set lifecycle_status='approved', approved_at=now(), approved_by_user_id=$2::uuid,
			approval_evidence='{"approved":true}'::jsonb
		where id=$1::uuid
	`, procedureID, fixture.memberUserID); err != nil {
		t.Fatalf("approve additional procedure: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		update education_portfolio_procedure_versions
		set lifecycle_status='published', published_at=now(), published_by_user_id=$2::uuid,
			publication_evidence='{"published":true}'::jsonb
		where id=$1::uuid
	`, procedureID, fixture.memberUserID); err != nil {
		t.Fatalf("publish additional procedure: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit additional procedure fixture: %v", err)
	}
	return procedureID
}

func supersedePortfolioProcedure(t *testing.T, ctx context.Context, pool *pgxpool.Pool, fixture governanceAuthorizationFixture, procedureID string) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
		update education_portfolio_procedure_versions
		set lifecycle_status='superseded', superseded_at=now()
		where id=$1::uuid and institution_id=$2
	`, procedureID, fixture.institutionA); err != nil {
		t.Fatalf("supersede applied procedure: %v", err)
	}
}
