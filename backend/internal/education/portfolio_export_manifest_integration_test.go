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
	if _, err = tx.Exec(ctx, `select set_config('app.is_super_admin','true',true), set_config('app.tenant_id',$1,true), set_config('app.institution_id',$2,true)`, fixture.tenantA, fixture.institutionA); err != nil {
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
	archiveDocumentID := seedGovernancePortfolioArchiveAttachments(t, ctx, pool, fixture.institutionA, fixture.memberUserID)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin export evidence fixture: %v", err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `select set_config('app.is_super_admin','true',true), set_config('app.tenant_id',$1,true), set_config('app.institution_id',$2,true)`, fixture.tenantA, fixture.institutionA); err != nil {
		t.Fatalf("bind export evidence fixture: %v", err)
	}
	const sourceHash = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if _, err = tx.Exec(ctx, `update archive_document_versions set source_bucket='earhive',source_object_key='integration/evidence.pdf',source_sha256=$2,hash_sha256=$2 where document_id=$1::uuid and version_no=1`, archiveDocumentID, sourceHash); err != nil {
		t.Fatalf("seed archive snapshot hash: %v", err)
	}
	var versionID string
	if err = tx.QueryRow(ctx, `select id::text from archive_document_versions where document_id=$1::uuid and version_no=1`, archiveDocumentID).Scan(&versionID); err != nil {
		t.Fatalf("load archive version fixture: %v", err)
	}
	if _, err = tx.Exec(ctx, `insert into education_portfolio_documents (portfolio_id,section_code,component_code,document_title,source_scope,evidence_type,issued_on,added_on,chronological_index,sensitive_data,authenticity_status,file_reference,institution_id,archive_document_id,archive_version_id,archive_version_no,archive_source_bucket,archive_source_object_key,archive_sha256) values ($1::uuid,'identificare_profesionala','structura_cadru','Dovadă export','portofoliu','document',current_date,current_date,1,false,'declarat','archive://' || $2::text,$3,$2::uuid,$4::uuid,1,'earhive','integration/evidence.pdf',$5)`, fixture.portfolioID, archiveDocumentID, fixture.institutionA, versionID, sourceHash); err != nil {
		t.Fatalf("seed snapshotted portfolio evidence: %v", err)
	}
	if _, err = tx.Exec(ctx, `update education_portfolios set status='submitted' where id=$1::uuid and institution_id=$2`, fixture.portfolioID, fixture.institutionA); err != nil {
		t.Fatalf("submit export fixture portfolio: %v", err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatalf("commit export evidence fixture: %v", err)
	}
}
