//go:build integration

package education

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eguilde/egueducation/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type portfolioOwnExportSecurityFixture struct {
	ctx       context.Context
	adminPool *pgxpool.Pool
	database  governanceIntegrationDatabase
	auth      governanceAuthorizationFixture
	reader    *fakePortfolioArchiveReader
	service   *Service
}

func newPortfolioOwnExportSecurityFixture(t *testing.T) portfolioOwnExportSecurityFixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	t.Cleanup(cancel)
	database := newGovernanceIntegrationDatabase(t)
	adminPool := openGovernanceIntegrationPool(t, ctx, database.databaseConfig)
	t.Cleanup(adminPool.Close)
	if err := db.Migrate(ctx, adminPool); err != nil {
		t.Fatalf("migrate portfolio export security database: %v", err)
	}
	grantGovernanceIntegrationAccess(t, ctx, adminPool, database.roleName)
	auth := seedGovernanceAuthorizationFixture(t, ctx, adminPool)
	grantPortfolioExportOwnPermissions(t, ctx, adminPool, auth)
	reader := &fakePortfolioArchiveReader{
		bucket: "earhive",
		objects: map[string][]byte{
			"integration/evidence.pdf\x00integration-version-1": []byte("original evidence bytes"),
		},
	}
	return portfolioOwnExportSecurityFixture{
		ctx:       ctx,
		adminPool: adminPool,
		database:  database,
		auth:      auth,
		reader:    reader,
		service:   NewService(db.NewSessionPool(database.readerPool), WithPortfolioArchiveReader(reader)),
	}
}

func (f portfolioOwnExportSecurityFixture) seedCessationDerivedEvidence(t *testing.T) {
	t.Helper()
	tx, err := f.adminPool.Begin(f.ctx)
	if err != nil {
		t.Fatalf("begin cessation-derived evidence fixture: %v", err)
	}
	defer tx.Rollback(f.ctx)
	if _, err = tx.Exec(f.ctx, `select set_config('app.is_super_admin','true',true), set_config('app.tenant_id',$1,true), set_config('app.institution_id',$2,true),set_config('app.actor_subject','portfolio-export-security-test',true)`, f.auth.tenantA, f.auth.institutionA); err != nil {
		t.Fatalf("bind cessation-derived evidence fixture: %v", err)
	}
	const sourceHash = "95f26de9529f6f676d951137f4c38b30e56393459475d23928eef3a1475b50c5"
	var archiveDocumentID string
	versionID := uuid.NewString()
	intentID := uuid.NewString()
	if err = tx.QueryRow(f.ctx, `
		insert into archive_documents (institution_id,title,original_file_name,mime_type,source_kind,status,current_version_no)
		values ($1,'Security export evidence','security-evidence.pdf','application/pdf','upload','ready',1)
		returning id::text
	`, f.auth.institutionA).Scan(&archiveDocumentID); err != nil {
		t.Fatalf("seed security archive document: %v", err)
	}
	if _, err = tx.Exec(f.ctx, `insert into portfolio_custody_upload_intents(
		id,tenant_code,institution_id,portfolio_id,actor_subject,idempotency_key,expected_sha256,expected_size_bytes,
		expected_mime_type,bucket_name,object_key,reserved_document_id,reserved_version_id)
		values($1::uuid,$2,$3,$4::uuid,'portfolio-export-security-test',$5,$6,23,'application/pdf','earhive','integration/evidence.pdf',$7::uuid,$8::uuid)`,
		intentID, f.auth.tenantA, f.auth.institutionA, f.auth.portfolioID, "portfolio-export-security-"+intentID, sourceHash, archiveDocumentID, versionID); err != nil {
		t.Fatalf("seed security custody intent: %v", err)
	}
	if _, err = tx.Exec(f.ctx, `update portfolio_custody_upload_intents set status='stored',stored_version_id='integration-version-1',stored_etag='integration-etag-1',stored_size_bytes=23 where id=$1::uuid`, intentID); err != nil {
		t.Fatalf("store security custody intent: %v", err)
	}
	if err = tx.QueryRow(f.ctx, `
		insert into archive_document_versions (
			id,document_id,institution_id,version_no,mime_type,title,bucket_name,object_key,hash_sha256,status,
			source_bucket,source_object_key,source_object_version_id,source_object_etag,source_sha256,source_size_bytes,custody_hold_active,portfolio_custody_intent_id
		) values ($1::uuid,$2::uuid,$3,1,'application/pdf','Security export evidence','earhive','integration/evidence.pdf',$4,'active',
			'earhive','integration/evidence.pdf','integration-version-1','integration-etag-1',$4,23,true,$5::uuid)
		returning id::text
	`, versionID, archiveDocumentID, f.auth.institutionA, sourceHash, intentID).Scan(&versionID); err != nil {
		t.Fatalf("seed cessation-retained archive version: %v", err)
	}
	if _, err = tx.Exec(f.ctx, `
		insert into education_portfolio_documents (
			portfolio_id,section_code,component_code,document_title,description,school_year,subject_discipline,applicable_class,
			competencies,source_scope,evidence_type,issued_on,added_on,chronological_index,sensitive_data,authenticity_status,
			file_reference,institution_id,archive_document_id,archive_version_id,archive_version_no,archive_source_bucket,
			archive_source_object_key,archive_sha256,last_change_reason
		) values ($1::uuid,'identificare_profesionala','structura_cadru','Dovadă export securitate','Dovadă profesională exportabilă',
			'2026-2027','disciplină de test','clasa a V-a',array['competență de test'],'portofoliu','document',current_date,current_date,
			1,false,'declarat','archive://' || $2::text,$3,$2::uuid,$4::uuid,1,'earhive','integration/evidence.pdf',$5,'Fixture securitate export')
	`, f.auth.portfolioID, archiveDocumentID, f.auth.institutionA, versionID, sourceHash); err != nil {
		t.Fatalf("seed portfolio evidence snapshot: %v", err)
	}
	if _, err = tx.Exec(f.ctx, `update education_portfolios set status='submitted' where id=$1::uuid and institution_id=$2`, f.auth.portfolioID, f.auth.institutionA); err != nil {
		t.Fatalf("submit cessation-retained portfolio fixture: %v", err)
	}
	if _, err = tx.Exec(f.ctx, `insert into education_portfolio_archive_attachment_grants (institution_id,archive_document_id,grantee_user_id) values ($1,$2::uuid,$3::uuid)`, f.auth.institutionA, archiveDocumentID, f.auth.memberUserID); err != nil {
		t.Fatalf("grant security export evidence attachment: %v", err)
	}
	if _, err = tx.Exec(f.ctx, `update education_portfolios set activity_ceased_on=date '2026-09-01',activity_cessation_reason='security integration fixture cessation' where id=$1::uuid and institution_id=$2`, f.auth.portfolioID, f.auth.institutionA); err != nil {
		t.Fatalf("derive portfolio retention from cessation event: %v", err)
	}
	if err = tx.Commit(f.ctx); err != nil {
		t.Fatalf("commit cessation-derived evidence fixture: %v", err)
	}
}

func (f portfolioOwnExportSecurityFixture) request(t *testing.T, tenant, institution, subject, recordID string, body io.Reader) (*http.Request, func()) {
	t.Helper()
	requestContext, release := governanceTenantContext(t, f.ctx, f.database.readerPool, tenant, institution, subject)
	base := requestWithContext(requestContext, tenant, institution, subject)
	request := httptest.NewRequest(http.MethodPost, "http://education.test/education/portfolios/me/"+recordID+"/export", body).WithContext(base.Context())
	return withChiParams(request, map[string]string{"recordID": recordID}), release
}

func (f portfolioOwnExportSecurityFixture) assertNoReleasePersistence(t *testing.T) {
	t.Helper()
	var manifests, audits int
	if err := f.adminPool.QueryRow(f.ctx, `select count(*) from education_portfolio_export_manifests where portfolio_id=$1::uuid`, f.auth.portfolioID).Scan(&manifests); err != nil {
		t.Fatalf("count portfolio export manifests: %v", err)
	}
	if err := f.adminPool.QueryRow(f.ctx, `select count(*) from app_audit_log where action='education.portfolios.own_export'`).Scan(&audits); err != nil {
		t.Fatalf("count portfolio own-export audits: %v", err)
	}
	if manifests != 0 || audits != 0 {
		t.Fatalf("denied export persisted release evidence: manifests=%d audits=%d", manifests, audits)
	}
}

func TestPortfolioOwnExportRejectsNonEmptyBodyWithoutPersistence(t *testing.T) {
	fixture := newPortfolioOwnExportSecurityFixture(t)
	request, release := fixture.request(t, fixture.auth.tenantA, fixture.auth.institutionA, fixture.auth.memberSubject, fixture.auth.portfolioID, strings.NewReader(`{}`))
	defer release()
	response := httptest.NewRecorder()

	fixture.service.PortfolioOwnExport(response, request)

	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "education_portfolio_export_body_not_allowed") {
		t.Fatalf("nonempty export body must fail closed: status=%d body=%s", response.Code, response.Body.String())
	}
	if fixture.reader.calls != 0 {
		t.Fatalf("rejected request reached archive storage: calls=%d", fixture.reader.calls)
	}
	fixture.assertNoReleasePersistence(t)
}

func TestPortfolioOwnExportDeniesForeignOwnerAndCrossTenant(t *testing.T) {
	fixture := newPortfolioOwnExportSecurityFixture(t)
	foreignSubject := subjectForUser(t, fixture.ctx, fixture.adminPool, fixture.auth.foreignMemberUserID)
	tests := []struct {
		name        string
		tenant      string
		institution string
		subject     string
		wantStatus  int
	}{
		{name: "foreign owner in same tenant", tenant: fixture.auth.tenantA, institution: fixture.auth.institutionA, subject: foreignSubject, wantStatus: http.StatusNotFound},
		{name: "owner through foreign tenant context", tenant: fixture.auth.tenantB, institution: fixture.auth.institutionB, subject: fixture.auth.memberSubject, wantStatus: http.StatusForbidden},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request, release := fixture.request(t, test.tenant, test.institution, test.subject, fixture.auth.portfolioID, nil)
			defer release()
			response := httptest.NewRecorder()

			fixture.service.PortfolioOwnExport(response, request)

			if response.Code != test.wantStatus || response.Header().Get("Content-Type") == "application/zip" {
				t.Fatalf("foreign export access was not denied: status=%d want=%d headers=%v body=%s", response.Code, test.wantStatus, response.Header(), response.Body.String())
			}
		})
	}
	if fixture.reader.calls != 0 {
		t.Fatalf("foreign access reached archive storage: calls=%d", fixture.reader.calls)
	}
	fixture.assertNoReleasePersistence(t)
}

func TestPortfolioOwnExportRejectsRevokedGrantWithoutPersistence(t *testing.T) {
	fixture := newPortfolioOwnExportSecurityFixture(t)
	fixture.seedCessationDerivedEvidence(t)
	tag, err := fixture.adminPool.Exec(fixture.ctx, `
		delete from education_portfolio_archive_attachment_grants attachment_grant
		where attachment_grant.institution_id=$1 and attachment_grant.grantee_user_id=$2::uuid
			and attachment_grant.archive_document_id in (
				select document.archive_document_id from education_portfolio_documents document
				where document.portfolio_id=$3::uuid and document.institution_id=$1
			)
	`, fixture.auth.institutionA, fixture.auth.memberUserID, fixture.auth.portfolioID)
	if err != nil {
		t.Fatalf("revoke portfolio evidence grant: %v", err)
	}
	if tag.RowsAffected() != 1 {
		t.Fatalf("revoked grant rows=%d want=1", tag.RowsAffected())
	}
	request, release := fixture.request(t, fixture.auth.tenantA, fixture.auth.institutionA, fixture.auth.memberSubject, fixture.auth.portfolioID, nil)
	defer release()
	response := httptest.NewRecorder()

	fixture.service.PortfolioOwnExport(response, request)

	if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), "education_portfolio_export_provenance_incomplete") || response.Header().Get("Content-Type") == "application/zip" {
		t.Fatalf("revoked evidence grant must fail before ZIP response: status=%d headers=%v body=%s", response.Code, response.Header(), response.Body.String())
	}
	if fixture.reader.calls != 0 {
		t.Fatalf("revoked grant reached archive storage: calls=%d", fixture.reader.calls)
	}
	fixture.assertNoReleasePersistence(t)
}

func TestPortfolioOwnExportReadsLegalHoldWithoutMutation(t *testing.T) {
	fixture := newPortfolioOwnExportSecurityFixture(t)
	fixture.seedCessationDerivedEvidence(t)
	const holdTimestamp = "2026-09-12T10:11:12.123456Z"
	tx, err := fixture.adminPool.Begin(fixture.ctx)
	if err != nil {
		t.Fatalf("begin legal-hold fixture: %v", err)
	}
	defer tx.Rollback(fixture.ctx)
	if _, err = tx.Exec(fixture.ctx, `select set_config('app.is_super_admin','true',true), set_config('app.tenant_id',$1,true), set_config('app.institution_id',$2,true), set_config('app.actor_subject','portfolio-export-security-test',true)`, fixture.auth.tenantA, fixture.auth.institutionA); err != nil {
		t.Fatalf("bind legal-hold fixture session: %v", err)
	}
	if _, err = tx.Exec(fixture.ctx, `update education_portfolios set legal_hold_active=true,legal_hold_reason='judicial review',legal_hold_set_at=$1::timestamptz,legal_hold_set_by_subject='legal-officer' where id=$2::uuid and institution_id=$3`, holdTimestamp, fixture.auth.portfolioID, fixture.auth.institutionA); err != nil {
		t.Fatalf("set portfolio legal hold: %v", err)
	}
	if err = tx.Commit(fixture.ctx); err != nil {
		t.Fatalf("commit legal-hold fixture: %v", err)
	}

	type holdState struct {
		PortfolioActive bool
		Reason          string
		SetAt           string
		SetBy           string
		ArchiveActive   bool
	}
	loadHoldState := func() holdState {
		t.Helper()
		var state holdState
		if err := fixture.adminPool.QueryRow(fixture.ctx, `
			select portfolio.legal_hold_active,portfolio.legal_hold_reason,
				to_char(portfolio.legal_hold_set_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
				portfolio.legal_hold_set_by_subject,version.legal_hold_active
			from education_portfolios portfolio
			join education_portfolio_documents document on document.portfolio_id=portfolio.id and document.institution_id=portfolio.institution_id
			join archive_document_versions version on version.id=document.archive_version_id and version.institution_id=document.institution_id
			where portfolio.id=$1::uuid and portfolio.institution_id=$2
		`, fixture.auth.portfolioID, fixture.auth.institutionA).Scan(&state.PortfolioActive, &state.Reason, &state.SetAt, &state.SetBy, &state.ArchiveActive); err != nil {
			t.Fatalf("load legal-hold state: %v", err)
		}
		return state
	}
	before := loadHoldState()
	request, release := fixture.request(t, fixture.auth.tenantA, fixture.auth.institutionA, fixture.auth.memberSubject, fixture.auth.portfolioID, nil)
	defer release()
	response := httptest.NewRecorder()

	fixture.service.PortfolioOwnExport(response, request)

	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/zip" {
		t.Fatalf("legal-held portfolio must remain readable: status=%d headers=%v body=%s", response.Code, response.Header(), response.Body.String())
	}
	after := loadHoldState()
	if !before.PortfolioActive || before.ArchiveActive || before.Reason != "judicial review" || before.SetBy != "legal-officer" || before.SetAt != "2026-09-12T10:11:12.123456Z" {
		t.Fatalf("legal-hold fixture was not established: %#v", before)
	}
	if after != before {
		t.Fatalf("read-only export mutated legal hold: before=%#v after=%#v", before, after)
	}
	if fixture.reader.calls != 1 {
		t.Fatalf("legal-held export exact-object reads=%d want=1", fixture.reader.calls)
	}
}
