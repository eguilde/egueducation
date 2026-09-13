package db

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestEducationSignedArtifactEvidenceNOBYPASSRLS proves the evidence ledger is
// enforced by PostgreSQL, not by a caller convention. It is optional locally.
func TestEducationSignedArtifactEvidenceNOBYPASSRLS(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("EGUEDUCATION_TEST_DATABASE_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	}
	if dsn == "" {
		t.Skip("signed artifact evidence PostgreSQL integration requires EGUEDUCATION_TEST_DATABASE_URL or TEST_DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	it := newTenantGrantRLSIntegrationDatabase(t, ctx, dsn)
	admin, err := pgxpool.NewWithConfig(ctx, it.databaseConfig.Copy())
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	if err = Migrate(ctx, admin); err != nil {
		t.Fatal(err)
	}
	fixture := seedOfficialArtifactFixture(t, ctx, admin)
	archiveDocumentID, archiveVersionID := uuid.New(), uuid.New()
	foreignDocumentID, foreignVersionID := uuid.New(), uuid.New()
	inactiveDocumentID, inactiveDocumentVersionID := uuid.New(), uuid.New()
	inactiveVersionDocumentID, inactiveVersionID := uuid.New(), uuid.New()
	blockedDecisionID := uuid.New()
	canonicalSHA := "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	if _, err := admin.Exec(ctx, `
		insert into archive_documents(id,institution_id,title,original_file_name,mime_type,source_kind,status,current_version_no)
		values ($1,'inst-001','Signed evidence source','signed.pdf','application/pdf','upload','ready',1),
		       ($2,'inst-balotesti','Foreign signed source','foreign.pdf','application/pdf','upload','ready',1),
		       ($3,'inst-001','Inactive signed source','inactive-document.pdf','application/pdf','upload','archived',1),
		       ($4,'inst-001','Inactive signed version source','inactive-version.pdf','application/pdf','upload','ready',1)
	`, archiveDocumentID, foreignDocumentID, inactiveDocumentID, inactiveVersionDocumentID); err != nil {
		t.Fatalf("seed signed evidence archive documents: %v", err)
	}
	if _, err := admin.Exec(ctx, `
		insert into archive_document_versions(id,document_id,institution_id,version_no,mime_type,title,bucket_name,object_key,hash_sha256,status,source_bucket,source_object_key,source_sha256)
		values ($3,$1,'inst-001',1,'application/pdf','Signed evidence source','legacy','legacy.pdf',$5,'active','earhive','signed/evidence.pdf',$5),
		       ($4,$2,'inst-balotesti',1,'application/pdf','Foreign signed source','legacy','foreign.pdf',$5,'active','earhive','foreign/evidence.pdf',$5),
		       ($7,$6,'inst-001',1,'application/pdf','Inactive signed source','legacy','inactive-document.pdf',$5,'active','earhive','signed/inactive-document.pdf',$5),
		       ($9,$8,'inst-001',1,'application/pdf','Inactive signed version source','legacy','inactive-version.pdf',$5,'archived','earhive','signed/inactive-version.pdf',$5)
	`, archiveDocumentID, foreignDocumentID, archiveVersionID, foreignVersionID, canonicalSHA, inactiveDocumentID, inactiveDocumentVersionID, inactiveVersionDocumentID, inactiveVersionID); err != nil {
		t.Fatalf("seed signed evidence archive provenance: %v", err)
	}
	if _, err := admin.Exec(ctx, `
		insert into education_decisions(id,decision_code,school_year,organism,title,status,publication_status,decision_date,institution_id)
		values($1,$2,'2031-2032','ca','Blocked signed evidence decision','blocked','internal',current_date,'inst-001')
	`, blockedDecisionID, "SIGNED-EVIDENCE-BLOCKED-"+uuid.NewString()); err != nil {
		t.Fatalf("seed inactive signed evidence decision: %v", err)
	}
	grantOfficialArtifactApplicationAccess(t, ctx, admin, it.roleName)
	role := tenantGrantQuoteIdentifier(it.roleName)
	for _, statement := range []string{"grant select, insert, update, delete on education_signed_artifact_evidence to " + role, "grant select, insert, update, delete on education_signed_artifact_validations to " + role} {
		if _, err := admin.Exec(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	cfg := it.databaseConfig.Copy()
	cfg.ConnConfig.User = it.roleName
	cfg.ConnConfig.Password = it.rolePassword
	cfg.MaxConns = 1
	app, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	pool := NewSessionPool(app)
	request, release, err := AcquireRequestConn(ctx, app, SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: "signed-evidence-test"})
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	var id string
	err = pool.QueryRow(request, `insert into education_signed_artifact_evidence(tenant_code,institution_id,artifact_type,artifact_id,document_sha256,signature_format,signature_level,signature_subject,certificate_issuer,certificate_serial,certificate_valid_from,certificate_valid_until,storage_document_id,storage_version_id,storage_bucket,storage_object_key) values(public.current_tenant_code(),public.current_institution_id(),'decision',$1,$2,'PAdES','advanced','Test signer','Test issuer','01',now()-interval '1 day',now()+interval '1 day',$3,$4,'browser-bucket','browser-key') returning id::text`, fixture.decisionID, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", archiveDocumentID, archiveVersionID).Scan(&id)
	if err != nil {
		t.Fatalf("append evidence: %v", err)
	}
	var storedSHA, storedBucket, storedObjectKey string
	if err := pool.QueryRow(request, `select document_sha256,storage_bucket,storage_object_key from education_signed_artifact_evidence where id=$1::uuid`, id).Scan(&storedSHA, &storedBucket, &storedObjectKey); err != nil {
		t.Fatalf("read derived provenance: %v", err)
	}
	if storedSHA != canonicalSHA || storedBucket != "earhive" || storedObjectKey != "signed/evidence.pdf" {
		t.Fatalf("browser provenance was not replaced: sha=%q bucket=%q key=%q", storedSHA, storedBucket, storedObjectKey)
	}
	assertEvidenceRejected := func(name, artifactType string, artifactID, documentID, versionID uuid.UUID) {
		t.Helper()
		assertOfficialArtifactWriteBlocked(t, request, pool, name, `insert into education_signed_artifact_evidence(tenant_code,institution_id,artifact_type,artifact_id,document_sha256,signature_format,signature_level,signature_subject,certificate_issuer,certificate_serial,certificate_valid_from,certificate_valid_until,storage_document_id,storage_version_id) values(public.current_tenant_code(),public.current_institution_id(),$1,$2,$3,'PAdES','advanced','Test signer','Test issuer','inactive',now()-interval '1 day',now()+interval '1 day',$4,$5)`, artifactType, artifactID, canonicalSHA, documentID, versionID)
	}
	assertEvidenceRejected("inactive archive document provenance", "decision", fixture.decisionID, inactiveDocumentID, inactiveDocumentVersionID)
	assertEvidenceRejected("inactive archive version provenance", "decision", fixture.decisionID, inactiveVersionDocumentID, inactiveVersionID)
	assertEvidenceRejected("blocked decision evidence", "decision", blockedDecisionID, archiveDocumentID, archiveVersionID)
	if _, err := pool.Exec(request, `update education_publications set publication_status='retras',withdrawal_reason='Withdrawn before evidence' where id=$1`, fixture.publicationID); err != nil {
		t.Fatalf("withdraw signed evidence publication fixture: %v", err)
	}
	assertEvidenceRejected("withdrawn publication evidence", "publication", fixture.publicationID, archiveDocumentID, archiveVersionID)
	if _, err := pool.Exec(request, `update education_managerial_documents set document_status='archived',archival_reason='Archived before evidence' where id=$1`, fixture.managerialDocumentID); err != nil {
		t.Fatalf("archive signed evidence managerial fixture: %v", err)
	}
	assertEvidenceRejected("archived managerial document evidence", "managerial_document", fixture.managerialDocumentID, archiveDocumentID, archiveVersionID)
	assertOfficialArtifactWriteBlocked(t, request, pool, "missing archive provenance", `insert into education_signed_artifact_evidence(tenant_code,institution_id,artifact_type,artifact_id,document_sha256,signature_format,signature_level,signature_subject,certificate_issuer,certificate_serial,certificate_valid_from,certificate_valid_until) values(public.current_tenant_code(),public.current_institution_id(),'decision',$1,$2,'PAdES','advanced','Test signer','Test issuer','missing',now()-interval '1 day',now()+interval '1 day')`, fixture.decisionID, canonicalSHA)
	assertOfficialArtifactWriteBlocked(t, request, pool, "foreign archive provenance", `insert into education_signed_artifact_evidence(tenant_code,institution_id,artifact_type,artifact_id,document_sha256,signature_format,signature_level,signature_subject,certificate_issuer,certificate_serial,certificate_valid_from,certificate_valid_until,storage_document_id,storage_version_id) values(public.current_tenant_code(),public.current_institution_id(),'decision',$1,$2,'PAdES','advanced','Test signer','Test issuer','02',now()-interval '1 day',now()+interval '1 day',$3,$4)`, fixture.decisionID, canonicalSHA, foreignDocumentID, foreignVersionID)
	if _, err = pool.Exec(request, `insert into education_signed_artifact_validations(evidence_id,tenant_code,institution_id,validation_status,findings) values($1::uuid,public.current_tenant_code(),public.current_institution_id(),'pending','{}')`, id); err != nil {
		t.Fatalf("append pending validation: %v", err)
	}
	assertOfficialArtifactWriteBlocked(t, request, pool, "evidence update", `update education_signed_artifact_evidence set signature_subject='tampered' where id=$1::uuid`, id)
	assertOfficialArtifactWriteBlocked(t, request, pool, "evidence delete", `delete from education_signed_artifact_evidence where id=$1::uuid`, id)
	assertOfficialArtifactWriteBlocked(t, request, pool, "validation update", `update education_signed_artifact_validations set validation_status='valid' where evidence_id=$1::uuid`, id)
}
