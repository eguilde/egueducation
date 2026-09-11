//go:build integration

package education

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eguilde/egueducation/internal/db"
	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestPortfolioDocumentVersionHistoryIsTenantScopedAndIncludesImmutableArchiveProvenance(t *testing.T) {
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
	archiveDocumentID := seedGovernancePortfolioArchiveAttachments(t, ctx, adminPool, fixture.institutionA, fixture.memberUserID)
	service := NewService(db.NewSessionPool(it.readerPool))
	ctxA, releaseA := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	defer releaseA()

	payload := fmt.Sprintf(`{"section_code":"identificare_profesionala","component_code":"structura_cadru","document_title":"Planificare inițială","description":"Planificare verificabilă","school_year":"2026-2027","subject_discipline":"Matematică","applicable_class":"clasa a V-a","competencies":["rezolvare de probleme"],"evidence_type":"planificare","issued_on":"2026-09-01","added_on":"2026-09-02","chronological_index":1,"sensitive_data":false,"file_reference":"archive://%s","notes":"versiunea inițială"}`, archiveDocumentID)
	created := httptest.NewRecorder()
	service.PortfolioOwnDocumentCreate(created, portfolioDocumentMutationRequest(ctxA, fixture, http.MethodPost, payload, ""))
	if created.Code != http.StatusCreated {
		t.Fatalf("create portfolio document status=%d body=%s", created.Code, created.Body.String())
	}
	var document PortfolioDocument
	if err := json.Unmarshal(created.Body.Bytes(), &document); err != nil {
		t.Fatalf("decode created portfolio document: %v", err)
	}
	if document.ArchiveDocumentID != archiveDocumentID || document.ArchiveVersionNo != 1 || len(document.ArchiveSHA256) != 64 {
		t.Fatalf("archive snapshot not exposed by contract: %+v", document)
	}
	if document.ArchiveSHA256 != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
		t.Fatalf("uppercase legacy archive hash was not normalized: %q", document.ArchiveSHA256)
	}

	updatedPayload := fmt.Sprintf(`{"section_code":"identificare_profesionala","component_code":"structura_cadru","document_title":"Planificare revizuită","description":"Planificare verificabilă revizuită","school_year":"2026-2027","subject_discipline":"Matematică","applicable_class":"clasa a V-a","competencies":["rezolvare de probleme","argumentare"],"evidence_type":"planificare","issued_on":"2026-09-01","added_on":"2026-09-03","chronological_index":1,"sensitive_data":false,"file_reference":"archive://%s","notes":"revizuire"}`, archiveDocumentID)
	updated := httptest.NewRecorder()
	service.PortfolioOwnDocumentUpdate(updated, portfolioDocumentMutationRequest(ctxA, fixture, http.MethodPatch, updatedPayload, document.ID))
	if updated.Code != http.StatusOK {
		t.Fatalf("update portfolio document status=%d body=%s", updated.Code, updated.Body.String())
	}

	historyRequest := portfolioDocumentMutationRequest(ctxA, fixture, http.MethodGet, "", document.ID)
	historyRequest.URL.RawQuery = "sort=version_no&direction=desc&filter.change_type=update"
	history := httptest.NewRecorder()
	service.PortfolioOwnDocumentVersions(history, historyRequest)
	if history.Code != http.StatusOK {
		t.Fatalf("read own portfolio document history status=%d body=%s", history.Code, history.Body.String())
	}
	var page httpx.PageResponse[PortfolioDocumentVersion]
	if err := json.Unmarshal(history.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode portfolio document history: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("filtered update history total=%d items=%d", page.Total, len(page.Items))
	}
	if page.Items[0].ChangedBy != fixture.memberSubject || page.Items[0].Reason == "" || page.Items[0].Snapshot["document_title"] != "Planificare revizuită" {
		t.Fatalf("incomplete version history projection: %+v", page.Items[0])
	}
	for _, forbidden := range []string{"archive_source_bucket", "archive_source_object_key", "withdrawn_by_subject"} {
		if _, leaked := page.Items[0].Snapshot[forbidden]; leaked {
			t.Fatalf("version history leaked internal field %q: %+v", forbidden, page.Items[0].Snapshot)
		}
	}

	unavailableArchiveID := uuid.NewString()
	unavailablePayload := fmt.Sprintf(`{"section_code":"identificare_profesionala","component_code":"structura_cadru","document_title":"Planificare nereușită","description":"Dovadă cu arhivă indisponibilă","school_year":"2026-2027","subject_discipline":"Matematică","applicable_class":"clasa a V-a","competencies":["rezolvare de probleme"],"source_scope":"portofoliu","evidence_type":"planificare","issued_on":"2026-09-01","added_on":"2026-09-04","chronological_index":1,"sensitive_data":false,"authenticity_status":"declarat","file_reference":"archive://%s","notes":"invalid archive"}`, unavailableArchiveID)
	unavailable := httptest.NewRecorder()
	service.UpdatePortfolioDocument(unavailable, portfolioDocumentMutationRequest(ctxA, fixture, http.MethodPatch, unavailablePayload, document.ID))
	if unavailable.Code != http.StatusUnprocessableEntity {
		t.Fatalf("unavailable archive update status=%d body=%s", unavailable.Code, unavailable.Body.String())
	}

	releaseA()
	ctxB, releaseB := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantB, fixture.institutionB, "foreign-tenant-reader")
	defer releaseB()
	foreignBase := requestWithContext(ctxB, fixture.tenantB, fixture.institutionB, "foreign-tenant-reader")
	foreignRequest := httptest.NewRequest(http.MethodGet, "http://education.test", nil).WithContext(foreignBase.Context())
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("recordID", fixture.portfolioID)
	routeContext.URLParams.Add("documentID", document.ID)
	foreignRequest = foreignRequest.WithContext(context.WithValue(foreignRequest.Context(), chi.RouteCtxKey, routeContext))
	foreignHistory := httptest.NewRecorder()
	service.PortfolioDocumentVersions(foreignHistory, foreignRequest)
	if foreignHistory.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant history status=%d body=%s", foreignHistory.Code, foreignHistory.Body.String())
	}
}
