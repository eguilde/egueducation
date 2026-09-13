//go:build integration

package education

// This test exercises the public command handlers against the database
// backstops.  It intentionally does not bypass the handler for the legal
// declarations, procedure selection, or lifecycle commands.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eguilde/egueducation/internal/db"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPortfolioLegalWorkflowRequiresInstitutionProcedureDeclarationsAndLifecycleControls(t *testing.T) {
	it := newGovernanceIntegrationDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	adminPool := openGovernanceIntegrationPool(t, ctx, it.databaseConfig)
	defer adminPool.Close()
	if err := db.Migrate(ctx, adminPool); err != nil {
		t.Fatalf("migrate disposable legal-workflow database: %v", err)
	}
	grantGovernanceIntegrationAccess(t, ctx, adminPool, it.roleName)
	fixture := seedGovernanceAuthorizationFixture(t, ctx, adminPool)
	grantPortfolioLegalWorkflowPermissions(t, ctx, adminPool, fixture)

	service := NewService(db.NewSessionPool(it.readerPool))
	ownerCtx, ownerRelease := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	defer ownerRelease()

	// Institution administrators cannot pair one user's login with a different
	// teacher's personnel identity. This is tested through the real handler
	// before any workflow/procedure lookup can mask the semantic failure.
	forgedOwner := httptest.NewRecorder()
	service.CreatePortfolioRecord(forgedOwner, legalWorkflowRequest(ownerCtx, fixture, http.MethodPost, fmt.Sprintf(`{
		"owner_user_id":%q,"owner_personnel_id":%q,"owner_name":"Governance Integration Member","owner_role":"Profesor",
		"school_year":"2027-2028","status":"draft","section_count":0,"last_updated_on":"2027-09-01",
		"transfer_status":"none","authenticity_declared":false,"consent_captured":false,"custodian":"","notes":""
	}`, fixture.memberUserID, fixture.foreignPersonnelID), nil))
	assertHandlerCode(t, forgedOwner, http.StatusUnprocessableEntity, "portfolio_owner_identity_mismatch")

	// A portfolio must not be created merely because an owner has the normal
	// RBAC permission: the institution must have published a valid procedure.
	missingProcedure := httptest.NewRecorder()
	service.PortfolioOwnCreate(missingProcedure, legalWorkflowRequest(ownerCtx, fixture, http.MethodPost, `{"school_year":"2027-2028","last_updated_on":"2027-09-01","notes":"workflow"}`, nil))
	assertHandlerCode(t, missingProcedure, http.StatusUnprocessableEntity, "education_portfolio_published_procedure_required")

	procedureID := seedPublishedPortfolioProcedure(t, ctx, adminPool, fixture)
	created := httptest.NewRecorder()
	service.PortfolioOwnCreate(created, legalWorkflowRequest(ownerCtx, fixture, http.MethodPost, `{"school_year":"2027-2028","last_updated_on":"2027-09-01","notes":"workflow"}`, nil))
	if created.Code != http.StatusCreated {
		t.Fatalf("create with published procedure: status=%d body=%s", created.Code, created.Body.String())
	}
	var portfolio PortfolioRecord
	if err := json.Unmarshal(created.Body.Bytes(), &portfolio); err != nil {
		t.Fatalf("decode created portfolio: %v", err)
	}
	if portfolio.AppliedProcedureID != procedureID {
		t.Fatalf("created portfolio procedure=%q, want institution procedure %q", portfolio.AppliedProcedureID, procedureID)
	}
	if portfolio.OwnerPersonnelID != fixture.memberPersonnelID {
		t.Fatalf("self-service portfolio personnel=%q, want canonical owner %q", portfolio.OwnerPersonnelID, fixture.memberPersonnelID)
	}

	// A browser flag cannot make a portfolio ready.  Both current, server-issued
	// declaration acknowledgements and procedure-required evidence are required.
	missingAcknowledgements := httptest.NewRecorder()
	service.PortfolioOwnSubmit(missingAcknowledgements, legalWorkflowRequest(ownerCtx, fixture, http.MethodPost, "", map[string]string{"recordID": portfolio.ID}))
	assertHandlerCode(t, missingAcknowledgements, http.StatusUnprocessableEntity, "education_portfolio_submit_incomplete")
	assertResponseContains(t, missingAcknowledgements, "missing_declarations")

	for _, declarationType := range []string{"gdpr_information", "authenticity"} {
		acknowledged := httptest.NewRecorder()
		service.AcknowledgePortfolioOwnDeclaration(acknowledged, legalWorkflowRequest(ownerCtx, fixture, http.MethodPost, `{"confirmed":true}`, map[string]string{"recordID": portfolio.ID, "declarationType": declarationType}))
		if acknowledged.Code != http.StatusOK {
			t.Fatalf("acknowledge %s: status=%d body=%s", declarationType, acknowledged.Code, acknowledged.Body.String())
		}
	}

	missingEvidence := httptest.NewRecorder()
	service.PortfolioOwnSubmit(missingEvidence, legalWorkflowRequest(ownerCtx, fixture, http.MethodPost, "", map[string]string{"recordID": portfolio.ID}))
	assertHandlerCode(t, missingEvidence, http.StatusUnprocessableEntity, "education_portfolio_submit_incomplete")
	assertResponseContains(t, missingEvidence, "missing_components")

	addRequiredProcedureEvidence(t, ctx, adminPool, service, ownerCtx, fixture, portfolio.ID)

	accepted := httptest.NewRecorder()
	service.PortfolioOwnSubmit(accepted, legalWorkflowRequest(ownerCtx, fixture, http.MethodPost, "", map[string]string{"recordID": portfolio.ID}))
	if accepted.Code != http.StatusOK {
		t.Fatalf("submit after current acknowledgements and procedure evidence: status=%d body=%s", accepted.Code, accepted.Body.String())
	}
	if err := json.Unmarshal(accepted.Body.Bytes(), &portfolio); err != nil {
		t.Fatalf("decode submitted portfolio: %v", err)
	}
	if portfolio.Status != "submitted" || !portfolio.AuthenticityDeclared || !portfolio.ConsentCaptured {
		t.Fatalf("submission must derive legal projections from immutable acknowledgements, got %#v", portfolio)
	}
	ownerRelease()

	// Ownership and host-derived tenant scope are both part of the command
	// boundary; a second teacher and a different tenant cannot submit this ID.
	foreignSubject := subjectForUser(t, ctx, adminPool, fixture.foreignMemberUserID)
	foreignCtx, foreignRelease := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, foreignSubject)
	defer foreignRelease()
	foreign := httptest.NewRecorder()
	foreignFixture := fixture
	foreignFixture.memberSubject = foreignSubject
	service.PortfolioOwnSubmit(foreign, legalWorkflowRequest(foreignCtx, foreignFixture, http.MethodPost, "", map[string]string{"recordID": portfolio.ID}))
	assertHandlerCode(t, foreign, http.StatusForbidden, "education_portfolio_access_denied")
	foreignRelease()

	crossTenantCtx, crossTenantRelease := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantB, fixture.institutionB, fixture.memberSubject)
	defer crossTenantRelease()
	crossTenant := httptest.NewRecorder()
	service.PortfolioOwnSubmit(crossTenant, legalWorkflowRequest(crossTenantCtx, governanceAuthorizationFixture{tenantA: fixture.tenantB, institutionA: fixture.institutionB, memberSubject: fixture.memberSubject}, http.MethodPost, "", map[string]string{"recordID": portfolio.ID}))
	assertHandlerCode(t, crossTenant, http.StatusForbidden, "education_portfolio_access_denied")
	crossTenantRelease()
	ownerCtx, finalOwnerRelease := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	defer finalOwnerRelease()

	// Retention is produced only by the institution lifecycle command, then a
	// legal hold is persisted and visible in the returned authoritative record.
	cessation := httptest.NewRecorder()
	service.RecordPortfolioActivityCessation(cessation, legalWorkflowRequest(ownerCtx, fixture, http.MethodPost, `{"activity_ceased_on":"2023-10-01","reason":"Încetarea activității didactice"}`, map[string]string{"recordID": portfolio.ID}))
	if cessation.Code != http.StatusAccepted {
		t.Fatalf("record activity cessation: status=%d body=%s", cessation.Code, cessation.Body.String())
	}
	var cessationResponse PortfolioLifecycleOperationResponse
	if err := json.Unmarshal(cessation.Body.Bytes(), &cessationResponse); err != nil {
		t.Fatalf("decode cessation: %v", err)
	}
	if cessationResponse.Portfolio.ActivityCeasedOn != "2023-10-01" || cessationResponse.Portfolio.RetentionUntil != "2026-10-01" || cessationResponse.Operation.Type != "cessation_retention" || cessationResponse.Operation.TotalVersions == 0 {
		t.Fatalf("retention and durable storage work must be calculated from cessation event, got %#v", cessationResponse)
	}

	hold := httptest.NewRecorder()
	service.SetPortfolioLegalHold(hold, legalWorkflowRequest(ownerCtx, fixture, http.MethodPost, `{"active":true,"reason":"Litigiu în curs"}`, map[string]string{"recordID": portfolio.ID}))
	if hold.Code != http.StatusAccepted {
		t.Fatalf("set legal hold: status=%d body=%s", hold.Code, hold.Body.String())
	}
	var holdResponse PortfolioLifecycleOperationResponse
	if err := json.Unmarshal(hold.Body.Bytes(), &holdResponse); err != nil {
		t.Fatalf("decode legal hold: %v", err)
	}
	if !holdResponse.Portfolio.LegalHoldActive || holdResponse.Portfolio.LegalHoldReason != "Litigiu în curs" || holdResponse.Operation.Type != "legal_hold_reconcile" {
		t.Fatalf("legal hold was not persisted with durable reconciliation: %#v", holdResponse)
	}

	var persisted PortfolioRecord
	if err := scanPortfolioRecord(service.pool.QueryRow(ownerCtx, `select `+portfolioRecordColumns+` from education_portfolios where id=$1::uuid and institution_id=$2`, portfolio.ID, fixture.institutionA), &persisted); err != nil {
		t.Fatalf("load canonical persisted lifecycle portfolio: %v", err)
	}
	detailRecorder := httptest.NewRecorder()
	service.PortfolioRecordDetail(detailRecorder, legalWorkflowRequest(ownerCtx, fixture, http.MethodGet, "", map[string]string{"recordID": portfolio.ID}))
	if detailRecorder.Code != http.StatusOK {
		t.Fatalf("generic portfolio detail: status=%d body=%s", detailRecorder.Code, detailRecorder.Body.String())
	}
	var genericDetail PortfolioRecord
	if err := json.Unmarshal(detailRecorder.Body.Bytes(), &genericDetail); err != nil {
		t.Fatalf("decode generic portfolio detail: %v", err)
	}
	listRecorder := httptest.NewRecorder()
	service.PortfolioRecords(listRecorder, legalWorkflowRequest(ownerCtx, fixture, http.MethodGet, "", nil))
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("generic portfolio list: status=%d body=%s", listRecorder.Code, listRecorder.Body.String())
	}
	var genericList struct {
		Items []PortfolioRecord `json:"items"`
	}
	if err := json.Unmarshal(listRecorder.Body.Bytes(), &genericList); err != nil {
		t.Fatalf("decode generic portfolio list: %v", err)
	}
	var genericListed PortfolioRecord
	for _, candidate := range genericList.Items {
		if candidate.ID == portfolio.ID {
			genericListed = candidate
			break
		}
	}
	assertLifecycleProjection := func(label string, got PortfolioRecord) {
		t.Helper()
		if got.ID != persisted.ID || got.OwnerUserID != persisted.OwnerUserID || got.OwnerPersonnelID != persisted.OwnerPersonnelID ||
			got.ActivityCeasedOn != persisted.ActivityCeasedOn || got.RetentionUntil != persisted.RetentionUntil ||
			got.RetentionPeriodDays != persisted.RetentionPeriodDays || got.LegalHoldActive != persisted.LegalHoldActive ||
			got.LegalHoldReason != persisted.LegalHoldReason || got.AppliedProcedureID != persisted.AppliedProcedureID {
			t.Fatalf("%s lifecycle projection diverged: got=%#v persisted=%#v", label, got, persisted)
		}
	}
	assertLifecycleProjection("generic detail", genericDetail)
	assertLifecycleProjection("generic list", genericListed)

	statusRecorder := httptest.NewRecorder()
	service.PortfolioLifecycleOperationStatus(statusRecorder, legalWorkflowRequest(ownerCtx, fixture, http.MethodGet, "", map[string]string{"recordID": portfolio.ID, "operationID": cessationResponse.Operation.ID}))
	if statusRecorder.Code != http.StatusOK || !bytes.Contains(statusRecorder.Body.Bytes(), []byte(cessationResponse.Operation.ID)) {
		t.Fatalf("read owner-visible lifecycle operation: status=%d body=%s", statusRecorder.Code, statusRecorder.Body.String())
	}

	if _, err := service.pool.Exec(ownerCtx, `select set_config('app.actor_subject','portfolio-storage-lifecycle-worker',false)`); err != nil {
		t.Fatalf("bind lifecycle retry fixture worker: %v", err)
	}
	if _, err := service.pool.Exec(ownerCtx, `update education_portfolio_storage_transitions set status='blocked',last_error='storage_reconciliation_failed',locked_at=null,locked_by='' where operation_id=$1::uuid and status='pending'`, cessationResponse.Operation.ID); err != nil {
		t.Fatalf("block lifecycle transition for retry fixture: %v", err)
	}
	if _, err := service.pool.Exec(ownerCtx, `select set_config('app.actor_subject',$1,false)`, fixture.memberSubject); err != nil {
		t.Fatalf("restore lifecycle retry actor: %v", err)
	}
	retry := httptest.NewRecorder()
	service.RetryPortfolioLifecycleOperation(retry, legalWorkflowRequest(ownerCtx, fixture, http.MethodPost, `{"reason":"Storage operator remediation completed"}`, map[string]string{"recordID": portfolio.ID, "operationID": cessationResponse.Operation.ID}))
	if retry.Code != http.StatusAccepted {
		t.Fatalf("retry blocked lifecycle operation: status=%d body=%s", retry.Code, retry.Body.String())
	}
	var retryResponse PortfolioLifecycleOperationResponse
	if err := json.Unmarshal(retry.Body.Bytes(), &retryResponse); err != nil || retryResponse.Operation.Status != "pending" {
		t.Fatalf("retry response=%#v err=%v", retryResponse, err)
	}
	var retryHistory int
	if err := service.pool.QueryRow(ownerCtx, `select count(*) from education_portfolio_storage_transition_history where operation_id=$1::uuid`, cessationResponse.Operation.ID).Scan(&retryHistory); err != nil || retryHistory == 0 {
		t.Fatalf("durable retry history count=%d err=%v", retryHistory, err)
	}
	duplicateRetry := httptest.NewRecorder()
	service.RetryPortfolioLifecycleOperation(duplicateRetry, legalWorkflowRequest(ownerCtx, fixture, http.MethodPost, `{"reason":"Duplicate retry"}`, map[string]string{"recordID": portfolio.ID, "operationID": cessationResponse.Operation.ID}))
	assertHandlerCode(t, duplicateRetry, http.StatusConflict, "portfolio_lifecycle_retry_not_available")
}

func grantPortfolioLegalWorkflowPermissions(t *testing.T, ctx context.Context, pool *pgxpool.Pool, fixture governanceAuthorizationFixture) {
	t.Helper()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin permission seed transaction: %v", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `select set_config('app.is_super_admin','true',true), set_config('app.tenant_id',$1,true), set_config('app.institution_id',$2,true)`, fixture.tenantA, fixture.institutionA); err != nil {
		t.Fatalf("bind permission seed session: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		insert into app_user_permissions (user_id, permission_code, tenant_code)
		values
			($1::uuid, 'education.portfolios.read_own', $2),
			($1::uuid, 'education.portfolios.manage_own', $2),
			($1::uuid, 'education.portfolios.school.manage', $2),
			($3::uuid, 'education.portfolios.read_own', $2),
			($3::uuid, 'education.portfolios.manage_own', $2)
		on conflict do nothing
	`, fixture.memberUserID, fixture.tenantA, fixture.foreignMemberUserID); err != nil {
		t.Fatalf("grant legal workflow permissions: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit legal workflow permissions: %v", err)
	}
}

func seedPublishedPortfolioProcedure(t *testing.T, ctx context.Context, pool *pgxpool.Pool, fixture governanceAuthorizationFixture) string {
	t.Helper()
	procedureID := uuid.NewString()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin procedure seed transaction: %v", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `select set_config('app.is_super_admin','true',true), set_config('app.tenant_id',$1,true), set_config('app.institution_id',$2,true)`, fixture.tenantA, fixture.institutionA); err != nil {
		t.Fatalf("bind procedure seed session: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		insert into education_portfolio_procedure_versions (
			id, institution_id, tenant_code, procedure_code, version_no, title, source_ref,
			lifecycle_status, effective_from, created_by_user_id
		) values ($1::uuid, $2, $3, 'integration-portfolio', 1, 'Procedură portofoliu integrare',
			'Ordinul nr. 3.858/2026', 'draft', current_date - 1, $4::uuid)
	`, procedureID, fixture.institutionA, fixture.tenantA, fixture.memberUserID); err != nil {
		t.Fatalf("seed published institution procedure: %v", err)
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
		t.Fatalf("seed published procedure rules: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		update education_portfolio_procedure_versions
		set lifecycle_status='approved', approved_at=now(), approved_by_user_id=$2::uuid,
			approval_evidence='{"approved":true}'::jsonb
		where id=$1::uuid
	`, procedureID, fixture.memberUserID); err != nil {
		t.Fatalf("approve procedure fixture: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		update education_portfolio_procedure_versions
		set lifecycle_status='published', published_at=now(), published_by_user_id=$2::uuid,
			publication_evidence='{"published":true}'::jsonb
		where id=$1::uuid
		`, procedureID, fixture.memberUserID); err != nil {
		t.Fatalf("publish procedure fixture: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit published procedure fixture: %v", err)
	}
	return procedureID
}

func addRequiredProcedureEvidence(t *testing.T, setupCtx context.Context, adminPool *pgxpool.Pool, service *Service, requestCtx context.Context, fixture governanceAuthorizationFixture, portfolioID string) {
	t.Helper()
	for index, sectionCode := range []string{"identificare_profesionala", "predare_invatare_evaluare", "activitati_complementare", "managementul_clasei", "evolutie_dezvoltare_profesionala"} {
		archiveDocumentID := seedGovernancePortfolioArchiveAttachments(t, setupCtx, adminPool, fixture.institutionA, fixture.memberUserID)
		payload := fmt.Sprintf(`{"section_code":%q,"component_code":"structura_cadru","document_title":%q,"description":"Dovadă pedagogică verificabilă","school_year":"2027-2028","subject_discipline":"disciplină de test","applicable_class":"clasa a V-a","competencies":["competență de test"],"evidence_type":"document","issued_on":"2027-09-01","added_on":"2027-09-02","chronological_index":%d,"sensitive_data":false,"file_reference":%q,"notes":"evidence"}`,
			sectionCode, "Evidence "+sectionCode, index+1, "archive://"+archiveDocumentID)
		created := httptest.NewRecorder()
		service.PortfolioOwnDocumentCreate(created, legalWorkflowRequest(requestCtx, fixture, http.MethodPost, payload, map[string]string{"recordID": portfolioID}))
		if created.Code != http.StatusCreated {
			t.Fatalf("add required evidence for %s: status=%d body=%s", sectionCode, created.Code, created.Body.String())
		}
	}
}

func legalWorkflowRequest(ctx context.Context, fixture governanceAuthorizationFixture, method, payload string, params map[string]string) *http.Request {
	base := requestWithContext(ctx, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	request := httptest.NewRequest(method, "http://education.test", bytes.NewBufferString(payload)).WithContext(base.Context())
	if len(params) == 0 {
		return request
	}
	routeContext := chi.NewRouteContext()
	for key, value := range params {
		routeContext.URLParams.Add(key, value)
	}
	return request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
}

func assertHandlerCode(t *testing.T, recorder *httptest.ResponseRecorder, want int, code string) {
	t.Helper()
	if recorder.Code != want {
		t.Fatalf("handler status=%d, want %d, body=%s", recorder.Code, want, recorder.Body.String())
	}
	assertResponseContains(t, recorder, code)
}

func assertResponseContains(t *testing.T, recorder *httptest.ResponseRecorder, want string) {
	t.Helper()
	if !bytes.Contains(recorder.Body.Bytes(), []byte(want)) {
		t.Fatalf("handler response lacks %q: %s", want, recorder.Body.String())
	}
}

func subjectForUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID string) string {
	t.Helper()
	var subject string
	if err := pool.QueryRow(ctx, `select sub from app_users where id=$1::uuid`, userID).Scan(&subject); err != nil {
		t.Fatalf("load user subject: %v", err)
	}
	return subject
}
