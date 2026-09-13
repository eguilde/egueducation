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

func TestPortfolioExportManifestIsOwnerScopedAndUsesSnapshotProvenance(t *testing.T) {
	it := newGovernanceIntegrationDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	defer cancel()
	adminPool := openGovernanceIntegrationPool(t, ctx, it.databaseConfig)
	defer adminPool.Close()
	if err := db.Migrate(ctx, adminPool); err != nil {
		t.Fatalf("migrate export-manifest database: %v", err)
	}
	grantGovernanceIntegrationAccess(t, ctx, adminPool, it.roleName)
	fixture := seedGovernanceAuthorizationFixture(t, ctx, adminPool)
	seedPortfolioExportableEvidence(t, ctx, adminPool, fixture)
	grantPortfolioExportOwnPermissions(t, ctx, adminPool, fixture)

	service := NewService(db.NewSessionPool(it.readerPool))
	ownerCtx, ownerRelease := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	defer ownerRelease()
	ownerRequest := withChiParams(requestWithContext(ownerCtx, fixture.tenantA, fixture.institutionA, fixture.memberSubject), map[string]string{"recordID": fixture.portfolioID})
	ownerRequest.Method = http.MethodPost
	ownerResponse := httptest.NewRecorder()
	service.PortfolioExportManifestCreate(ownerResponse, ownerRequest)
	if ownerResponse.Code != http.StatusCreated {
		t.Fatalf("owner export manifest: status=%d body=%s", ownerResponse.Code, ownerResponse.Body.String())
	}
	var response PortfolioExportManifestResponse
	if err := json.Unmarshal(ownerResponse.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode owner export manifest: %v", err)
	}
	if response.ExportManifestID == "" || len(response.Manifest.Documents) != 1 || !isSHA256Hex(response.Manifest.ManifestSHA256) {
		t.Fatalf("export manifest lacks immutable evidence metadata: %#v", response)
	}
	document := response.Manifest.Documents[0]
	if document.ArchiveVersionNo != 1 || document.SourceBucket != "earhive" || document.SourceObjectKey != "integration/evidence.pdf" || !isSHA256Hex(document.SourceSHA256) {
		t.Fatalf("manifest must contain snapshot provenance, got %#v", document)
	}
	ownerRelease()

	foreignSubject := subjectForUser(t, ctx, adminPool, fixture.foreignMemberUserID)
	foreignCtx, foreignRelease := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, foreignSubject)
	defer foreignRelease()
	foreignRequest := withChiParams(requestWithContext(foreignCtx, fixture.tenantA, fixture.institutionA, foreignSubject), map[string]string{"recordID": fixture.portfolioID})
	foreignRequest.Method = http.MethodPost
	foreignResponse := httptest.NewRecorder()
	service.PortfolioExportManifestCreate(foreignResponse, foreignRequest)
	if foreignResponse.Code != http.StatusForbidden {
		t.Fatalf("non-owner read_own export must be denied: status=%d body=%s", foreignResponse.Code, foreignResponse.Body.String())
	}
}

func grantPortfolioExportOwnPermissions(t *testing.T, ctx context.Context, pool *pgxpool.Pool, fixture governanceAuthorizationFixture) {
	t.Helper()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin export permission fixture: %v", err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `select set_config('app.is_super_admin','true',true), set_config('app.tenant_id',$1,true), set_config('app.institution_id',$2,true), set_config('app.actor_subject',$3,true)`, fixture.tenantA, fixture.institutionA, fixture.memberSubject); err != nil {
		t.Fatalf("bind export permission fixture: %v", err)
	}
	if _, err = tx.Exec(ctx, `insert into app_user_permissions(user_id, permission_code, tenant_code) values ($1::uuid,'education.portfolios.read_own',$2),($3::uuid,'education.portfolios.read_own',$2) on conflict do nothing`, fixture.memberUserID, fixture.tenantA, fixture.foreignMemberUserID); err != nil {
		t.Fatalf("grant export own permissions: %v", err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatalf("commit export permissions: %v", err)
	}
}

func seedPortfolioExportableEvidence(t *testing.T, ctx context.Context, pool *pgxpool.Pool, fixture governanceAuthorizationFixture) {
	t.Helper()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin export evidence fixture: %v", err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `select set_config('app.is_super_admin','true',true), set_config('app.tenant_id',$1,true), set_config('app.institution_id',$2,true)`, fixture.tenantA, fixture.institutionA); err != nil {
		t.Fatalf("bind export evidence fixture: %v", err)
	}
	const sourceHash = "95f26de9529f6f676d951137f4c38b30e56393459475d23928eef3a1475b50c5"
	archiveDocumentID, versionID, intentID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	if _, err = tx.Exec(ctx, `
		insert into archive_documents (id,institution_id,title,original_file_name,mime_type,source_kind,status,current_version_no)
		values ($1::uuid,$2,'Export manifest evidence','integration-evidence.pdf','application/pdf','upload','ready',1)`, archiveDocumentID, fixture.institutionA); err != nil {
		t.Fatalf("seed archive evidence document: %v", err)
	}
	if _, err = tx.Exec(ctx, `insert into portfolio_custody_upload_intents(id,tenant_code,institution_id,portfolio_id,actor_subject,idempotency_key,expected_sha256,expected_size_bytes,expected_mime_type,expected_request_fingerprint,expected_metadata,bucket_name,object_key,reserved_document_id,reserved_version_id)
		values($1::uuid,$2,$3,$4::uuid,$5,$6,$7,23,'application/pdf',repeat('a',64),'{}'::jsonb,'earhive','integration/evidence.pdf',$8::uuid,$9::uuid)`, intentID, fixture.tenantA, fixture.institutionA, fixture.portfolioID, fixture.memberSubject, "export-fixture-"+intentID, sourceHash, archiveDocumentID, versionID); err != nil {
		t.Fatalf("seed export custody reservation: %v", err)
	}
	if _, err = tx.Exec(ctx, `update portfolio_custody_upload_intents set status='stored',stored_version_id='integration-version-1',stored_etag='integration-etag-1',stored_size_bytes=23 where id=$1::uuid`, intentID); err != nil {
		t.Fatalf("mark export custody object verified: %v", err)
	}
	if _, err = tx.Exec(ctx, `
		insert into archive_document_versions (id,document_id,institution_id,version_no,mime_type,title,bucket_name,object_key,hash_sha256,status,source_bucket,source_object_key,source_object_version_id,source_object_etag,source_sha256,source_size_bytes,retention_until,custody_hold_active,portfolio_custody_intent_id)
		values ($1::uuid,$2::uuid,$3,1,'application/pdf','Export manifest evidence','earhive','integration/evidence.pdf',$4,'active','earhive','integration/evidence.pdf','integration-version-1','integration-etag-1',$4,23,current_date + 365,true,$5::uuid)`, versionID, archiveDocumentID, fixture.institutionA, sourceHash, intentID); err != nil {
		t.Fatalf("seed immutable archive version fixture: %v", err)
	}
	if _, err = tx.Exec(ctx, `update portfolio_custody_upload_intents set status='committed',final_disposition='teacher_access',recovery_committed_at=now() where id=$1::uuid`, intentID); err != nil {
		t.Fatalf("commit export custody intent: %v", err)
	}
	if _, err = tx.Exec(ctx, `insert into education_portfolio_documents (portfolio_id,section_code,component_code,document_title,description,school_year,subject_discipline,applicable_class,competencies,source_scope,evidence_type,issued_on,added_on,chronological_index,sensitive_data,authenticity_status,file_reference,institution_id,archive_document_id,archive_version_id,archive_version_no,archive_source_bucket,archive_source_object_key,archive_sha256,last_change_reason) values ($1::uuid,'identificare_profesionala','structura_cadru','Dovadă export','Dovadă profesională exportabilă','2026-2027','disciplină de test','clasa a V-a',array['competență de test'],'portofoliu','document',current_date,current_date,1,false,'declarat','archive://' || $2::text,$3,$2::uuid,$4::uuid,1,'earhive','integration/evidence.pdf',$5,'Fixture integrare export')`, fixture.portfolioID, archiveDocumentID, fixture.institutionA, versionID, sourceHash); err != nil {
		t.Fatalf("seed snapshotted portfolio evidence: %v", err)
	}
	if _, err = tx.Exec(ctx, `update education_portfolios set status='submitted' where id=$1::uuid and institution_id=$2`, fixture.portfolioID, fixture.institutionA); err != nil {
		t.Fatalf("submit export fixture portfolio: %v", err)
	}
	if _, err = tx.Exec(ctx, `insert into education_portfolio_archive_attachment_grants (institution_id,archive_document_id,grantee_user_id) values ($1,$2::uuid,$3::uuid)`, fixture.institutionA, archiveDocumentID, fixture.memberUserID); err != nil {
		t.Fatalf("grant export evidence attachment: %v", err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatalf("commit export evidence fixture: %v", err)
	}
}
