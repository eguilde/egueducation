//go:build integration

package education

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authruntime "github.com/eguilde/egueducation/internal/auth"
	appdb "github.com/eguilde/egueducation/internal/db"
	"github.com/go-chi/chi/v5"
)

func TestPortfolioDeclarationAcknowledgementsAreOwnerBoundAndServerIssued(t *testing.T) {
	it := newGovernanceIntegrationDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	adminPool := openGovernanceIntegrationPool(t, ctx, it.databaseConfig)
	defer adminPool.Close()
	if err := appdb.Migrate(ctx, adminPool); err != nil {
		t.Fatalf("migrate disposable declaration database: %v", err)
	}
	grantGovernanceIntegrationAccess(t, ctx, adminPool, it.roleName)
	fixture := seedGovernanceAuthorizationFixture(t, ctx, adminPool)

	var foreignSubject string
	tx, err := adminPool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin permission fixture transaction: %v", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `select set_config('app.is_super_admin','true',true), set_config('app.tenant_id',$1,true), set_config('app.institution_id',$2,true)`, fixture.tenantA, fixture.institutionA); err != nil {
		t.Fatalf("bind permission fixture session: %v", err)
	}
	for _, userID := range []string{fixture.memberUserID, fixture.foreignMemberUserID} {
		if _, err := tx.Exec(ctx, `
			insert into app_user_permissions (user_id, permission_code, tenant_code)
			values ($1::uuid, 'education.portfolios.read_own', $2), ($1::uuid, 'education.portfolios.manage_own', $2)
		`, userID, fixture.tenantA); err != nil {
			t.Fatalf("grant own portfolio permissions: %v", err)
		}
	}
	if err := tx.QueryRow(ctx, `select sub from app_users where id = $1::uuid`, fixture.foreignMemberUserID).Scan(&foreignSubject); err != nil {
		t.Fatalf("load foreign subject: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit permission fixture: %v", err)
	}

	service := NewService(appdb.NewSessionPool(it.readerPool))
	ownerContext, ownerRelease := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	defer ownerRelease()
	ownerRequest := requestWithContext(ownerContext, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	ownerRequest = withChiParams(ownerRequest, map[string]string{"recordID": fixture.portfolioID})
	initial := httptest.NewRecorder()
	service.PortfolioOwnDeclarationEvidence(initial, ownerRequest)
	if initial.Code != http.StatusOK {
		t.Fatalf("owner reads declaration templates: status=%d body=%s", initial.Code, initial.Body.String())
	}
	var initialBody PortfolioDeclarationEvidenceResponse
	if err := json.Unmarshal(initial.Body.Bytes(), &initialBody); err != nil {
		t.Fatalf("decode templates response: %v", err)
	}
	if len(initialBody.Templates) != 2 || len(initialBody.Acknowledgements) != 0 {
		t.Fatalf("expected two server-issued active templates and no acknowledgements, got %#v", initialBody)
	}

	ackRequest := httptest.NewRequest(http.MethodPost, "http://education.test", bytes.NewBufferString(`{"confirmed":true}`)).WithContext(ownerRequest.Context())
	ackRequest = withSession(ackRequest, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	ackRequest = withChiParams(ackRequest, map[string]string{"recordID": fixture.portfolioID, "declarationType": "gdpr_information"})
	acknowledged := httptest.NewRecorder()
	service.AcknowledgePortfolioOwnDeclaration(acknowledged, ackRequest)
	if acknowledged.Code != http.StatusOK {
		t.Fatalf("owner acknowledgement: status=%d body=%s", acknowledged.Code, acknowledged.Body.String())
	}
	var evidence PortfolioDeclarationAcknowledgement
	if err := json.Unmarshal(acknowledged.Body.Bytes(), &evidence); err != nil {
		t.Fatalf("decode acknowledgement: %v", err)
	}
	if evidence.DeclarationVersion == "" || evidence.DeclarationText == "" || evidence.AttestationMethod != portfolioDeclarationAcknowledgementMethod || evidence.AcceptedByUserID != fixture.memberUserID {
		t.Fatalf("acknowledgement must preserve server-issued legal evidence, got %#v", evidence)
	}

	foreignContext, foreignRelease := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, foreignSubject)
	defer foreignRelease()
	foreignRequest := requestWithContext(foreignContext, fixture.tenantA, fixture.institutionA, foreignSubject)
	foreignRequest = withChiParams(foreignRequest, map[string]string{"recordID": fixture.portfolioID})
	denied := httptest.NewRecorder()
	service.PortfolioOwnDeclarationEvidence(denied, foreignRequest)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("cross-owner declaration evidence must be denied: status=%d body=%s", denied.Code, denied.Body.String())
	}
}

func withChiParams(request *http.Request, values map[string]string) *http.Request {
	routeContext := chi.NewRouteContext()
	for key, value := range values {
		routeContext.URLParams.Add(key, value)
	}
	return request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
}

func withSession(request *http.Request, tenantCode, institutionID, subject string) *http.Request {
	return request.WithContext(authruntime.WithSessionContextForIntegration(request.Context(), authruntime.SessionContext{
		TenantCode:    tenantCode,
		InstitutionID: institutionID,
		User:          authruntime.SessionUser{Sub: subject},
	}))
}
