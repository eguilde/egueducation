//go:build integration

package admission

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/eguilde/egueducation/internal/db"
	"github.com/google/uuid"
)

// This executes the 0155 trigger rather than merely checking migration text.
func TestAdmissionWORMIngestionIntentPostgres(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("requires TEST_DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	pool := newAdmissionExpiryDatabase(t, ctx, dsn)
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	sessions := db.NewSessionPool(pool)
	requestCtx, release, err := db.AcquireRequestConn(ctx, pool, db.SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: "expiry-regression-actor", IsSuperAdmin: true})
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	applicationID, _, evaluationID := seedAdmissionExpiryGraph(t, requestCtx, sessions)
	seedExpiryRetentionAuthority(t, ctx, pool, sessions)
	authority := loadPreparationRetentionAuthority(t, requestCtx, sessions)
	preparationID := insertRetentionPreparation(t, requestCtx, sessions, applicationID, evaluationID, authority, "")
	var deadline time.Time
	if err = sessions.QueryRow(requestCtx, `select required_retention_until from school_admission_legal_preparations where id=$1::uuid`, preparationID).Scan(&deadline); err != nil {
		t.Fatal(err)
	}
	intentID, documentID, versionID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	_, err = sessions.Exec(requestCtx, `insert into archive_ingestion_intents(id,tenant_code,institution_id,actor_subject,idempotency_key,request_fingerprint,purpose,preparation_id,artifact_slot,expected_sha256,expected_size_bytes,reserved_document_id,reserved_version_id,bucket_name,object_key,retention_until) values($1::uuid,'tenant-egueducation','inst-001','expiry-regression-actor','intent-key',repeat('a',64),'admission_legal_preparation',$2::uuid,'primary',repeat('b',64),10,$3::uuid,$4::uuid,'test','admission/test.pdf',$5)`, intentID, preparationID, documentID, versionID, deadline)
	if err != nil {
		t.Fatalf("valid intent: %v", err)
	}
	// Exercise the persisted adoption graph and the finalizer's exact binding,
	// rather than only the reservation trigger.
	tx, err := sessions.Begin(requestCtx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(requestCtx, `update archive_ingestion_intents set status='stored',stored_version_id='s3-version',stored_etag='etag',stored_size_bytes=10,stored_retention_until=$2 where id=$1::uuid`, intentID, deadline); err != nil {
		t.Fatal(err)
	}
	artifactKey := "archive/artifact/" + documentID + ".pdf"
	if _, err = tx.Exec(requestCtx, `insert into archive_documents(id,institution_id,title,original_file_name,mime_type,source_kind,source_system,external_reference,status,original_bucket,original_object_key,artifact_bucket,artifact_object_key,metadata,idempotency_key,created_by,current_version_no,received_at) values($1::uuid,'inst-001','legal artifact','legal.pdf','application/pdf','upload','admission',$2,'queued','test','admission/test.pdf','test',$3,'{}','intent-test',$4,1,now())`, documentID, preparationID, artifactKey, "expiry-regression-actor"); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(requestCtx, `insert into archive_document_versions(id,institution_id,document_id,version_no,mime_type,title,bucket_name,object_key,hash_sha256,size_bytes,metadata,ocr_text,status,source_bucket,source_object_key,artifact_bucket,artifact_object_key,source_sha256,source_size_bytes,page_count,text_status,extracted_text,extracted_metadata,created_by,source_object_version_id,source_object_etag,retention_until,legal_hold_active,ingestion_intent_id) values($1::uuid,'inst-001',$2::uuid,1,'application/pdf','legal artifact','test','admission/test.pdf',repeat('b',64),10,'{}','','active','test','admission/test.pdf','test',$3,repeat('b',64),10,1,'pending','','{}',$4,'s3-version','etag',$5,true,$6::uuid)`, versionID, documentID, artifactKey, "expiry-regression-actor", deadline, intentID); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(requestCtx, `insert into archive_ingestion_jobs(id,institution_id,document_id,version_id,job_type,status,available_at,created_by) values($1::uuid,'inst-001',$2::uuid,$3::uuid,'extract_text','pending',now(),$4)`, uuid.NewString(), documentID, versionID, "expiry-regression-actor"); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(requestCtx, `update archive_ingestion_intents set status='committed',committed_at=now() where id=$1::uuid`, intentID); err != nil {
		t.Fatal(err)
	}
	if err = requirePreparationArtifactIntent(requestCtx, tx, scope{tenant: "tenant-egueducation", institution: "inst-001"}, preparationID, "primary", ArchiveReference{DocumentID: documentID, VersionID: versionID}); err != nil {
		t.Fatalf("matching committed binding: %v", err)
	}
	if err = requirePreparationArtifactIntent(requestCtx, tx, scope{tenant: "tenant-egueducation", institution: "inst-001"}, preparationID, "resulting_decision", ArchiveReference{DocumentID: documentID, VersionID: versionID}); err != errInvalidState {
		t.Fatalf("wrong slot binding=%v", err)
	}
	if err = tx.Commit(requestCtx); err != nil {
		t.Fatal(err)
	}
	if _, err = sessions.Exec(requestCtx, `update archive_ingestion_intents set expected_sha256=repeat('c',64) where id=$1::uuid`, intentID); err == nil || !strings.Contains(err.Error(), "provenance is immutable") {
		t.Fatalf("immutable intent err=%v", err)
	}
	_, err = sessions.Exec(requestCtx, `insert into archive_ingestion_intents(id,tenant_code,institution_id,actor_subject,idempotency_key,request_fingerprint,purpose,preparation_id,artifact_slot,expected_sha256,expected_size_bytes,reserved_document_id,reserved_version_id,bucket_name,object_key,retention_until,status,stored_version_id,stored_etag,stored_size_bytes,stored_retention_until,committed_at) values($1::uuid,'tenant-egueducation','inst-001','expiry-regression-actor','bad-insert',repeat('d',64),'admission_legal_preparation',$2::uuid,'primary',repeat('b',64),10,$3::uuid,$4::uuid,'test','bad.pdf',$5,'committed','v','e',10,$5,now())`, uuid.NewString(), preparationID, uuid.NewString(), uuid.NewString(), deadline)
	if err == nil || !strings.Contains(err.Error(), "must reserve") {
		t.Fatalf("committed insert err=%v", err)
	}
}
