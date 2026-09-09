//go:build integration

package education

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eguilde/egueducation/internal/db"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestPortfolioDocumentMutationRollsBackWhenOpisSyncFails proves the document
// command and its derived OPIS projection are one database unit. The trigger
// intentionally makes the derived write fail after the evidence write has
// happened; create, update, and delete must each be rolled back.
func TestPortfolioDocumentMutationRollsBackWhenOpisSyncFails(t *testing.T) {
	it := newGovernanceIntegrationDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	adminPool := openGovernanceIntegrationPool(t, ctx, it.databaseConfig)
	defer adminPool.Close()
	if err := db.Migrate(ctx, adminPool); err != nil {
		t.Fatalf("migrate disposable database: %v", err)
	}
	if err := db.ValidateSchemaContract(ctx, adminPool); err != nil {
		t.Fatalf("validate schema contract: %v", err)
	}
	grantGovernanceIntegrationAccess(t, ctx, adminPool, it.roleName)
	fixture := seedGovernanceAuthorizationFixture(t, ctx, adminPool)
	service := NewService(db.NewSessionPool(it.readerPool))
	ctxA, release := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	defer release()

	payload := `{"section_code":"A","component_code":"A.1","document_title":"Evidence before rollback","source_scope":"portofoliu","evidence_type":"document","issued_on":"2026-09-01","added_on":"2026-09-02","chronological_index":1,"sensitive_data":false,"authenticity_status":"declarat","file_reference":"archive://00000000-0000-0000-0000-000000000001","notes":"atomic test"}`

	installOpisFailureTrigger(t, ctx, adminPool)
	createResponse := httptest.NewRecorder()
	service.CreatePortfolioDocument(createResponse, portfolioDocumentMutationRequest(ctxA, fixture, http.MethodPost, payload, ""))
	if createResponse.Code != http.StatusInternalServerError {
		t.Fatalf("create with failed OPIS sync status=%d body=%s", createResponse.Code, createResponse.Body.String())
	}
	assertPortfolioDocumentCount(t, ctxA, service, fixture.portfolioID, fixture.institutionA, 0)

	dropOpisFailureTrigger(t, ctx, adminPool)
	createResponse = httptest.NewRecorder()
	service.CreatePortfolioDocument(createResponse, portfolioDocumentMutationRequest(ctxA, fixture, http.MethodPost, payload, ""))
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("create baseline document status=%d body=%s", createResponse.Code, createResponse.Body.String())
	}
	var documentID string
	if err := service.pool.QueryRow(ctxA, `select id::text from education_portfolio_documents where portfolio_id = $1::uuid and institution_id = $2`, fixture.portfolioID, fixture.institutionA).Scan(&documentID); err != nil {
		t.Fatalf("load baseline document: %v", err)
	}

	installOpisFailureTrigger(t, ctx, adminPool)
	updatedPayload := strings.Replace(payload, "Evidence before rollback", "Evidence must not persist", 1)
	updateResponse := httptest.NewRecorder()
	service.UpdatePortfolioDocument(updateResponse, portfolioDocumentMutationRequest(ctxA, fixture, http.MethodPut, updatedPayload, documentID))
	if updateResponse.Code != http.StatusInternalServerError {
		t.Fatalf("update with failed OPIS sync status=%d body=%s", updateResponse.Code, updateResponse.Body.String())
	}
	var title string
	if err := service.pool.QueryRow(ctxA, `select document_title from education_portfolio_documents where id = $1::uuid and institution_id = $2`, documentID, fixture.institutionA).Scan(&title); err != nil {
		t.Fatalf("load document after failed update: %v", err)
	}
	if title != "Evidence before rollback" {
		t.Fatalf("failed OPIS update changed document title to %q", title)
	}

	deleteResponse := httptest.NewRecorder()
	service.DeletePortfolioDocument(deleteResponse, portfolioDocumentMutationRequest(ctxA, fixture, http.MethodDelete, "", documentID))
	if deleteResponse.Code != http.StatusInternalServerError {
		t.Fatalf("delete with failed OPIS sync status=%d body=%s", deleteResponse.Code, deleteResponse.Body.String())
	}
	assertPortfolioDocumentCount(t, ctxA, service, fixture.portfolioID, fixture.institutionA, 1)
}

func portfolioDocumentMutationRequest(ctx context.Context, fixture governanceAuthorizationFixture, method, payload, documentID string) *http.Request {
	base := requestWithContext(ctx, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	request := httptest.NewRequest(method, "http://education.test", strings.NewReader(payload)).WithContext(base.Context())
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("recordID", fixture.portfolioID)
	if documentID != "" {
		routeContext.URLParams.Add("documentID", documentID)
	}
	return request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
}

func assertPortfolioDocumentCount(t *testing.T, ctx context.Context, service *Service, portfolioID, institutionID string, want int) {
	t.Helper()
	var got int
	if err := service.pool.QueryRow(ctx, `select count(*) from education_portfolio_documents where portfolio_id = $1::uuid and institution_id = $2`, portfolioID, institutionID).Scan(&got); err != nil {
		t.Fatalf("count portfolio documents: %v", err)
	}
	if got != want {
		t.Fatalf("portfolio document count=%d, want %d", got, want)
	}
}

func installOpisFailureTrigger(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
		create or replace function integration_fail_portfolio_opis_sync()
		returns trigger language plpgsql as $$
		begin
			raise exception 'forced portfolio opis synchronization failure';
		end;
		$$;
	`); err != nil {
		t.Fatalf("create OPIS failure function: %v", err)
	}
	if _, err := pool.Exec(ctx, `drop trigger if exists integration_fail_portfolio_opis_sync on education_portfolio_opis`); err != nil {
		t.Fatalf("drop previous OPIS failure trigger: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		create trigger integration_fail_portfolio_opis_sync
		before insert on education_portfolio_opis
		for each row execute function integration_fail_portfolio_opis_sync()
	`); err != nil {
		t.Fatalf("create OPIS failure trigger: %v", err)
	}
}

func dropOpisFailureTrigger(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(ctx, `drop trigger if exists integration_fail_portfolio_opis_sync on education_portfolio_opis`); err != nil {
		t.Fatalf("drop OPIS failure trigger: %v", err)
	}
}
