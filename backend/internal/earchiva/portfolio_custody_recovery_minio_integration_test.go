//go:build integration

package earchiva

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/eguilde/egueducation/internal/config"
	appdb "github.com/eguilde/egueducation/internal/db"
	"github.com/google/uuid"
)

// TestPortfolioCustodyRecoveryMinIOExactVersionIntegration proves that the
// recovery boundary only reads the stored Object-Lock version.  It deliberately
// creates a uniquely named bucket; COMPLIANCE objects are retained by MinIO and
// therefore cannot be removed by this test before their policy deadline.
func TestPortfolioCustodyRecoveryMinIOExactVersionIntegration(t *testing.T) {
	endpoint := strings.TrimSpace(os.Getenv("TEST_WORM_MINIO_ENDPOINT"))
	if endpoint == "" {
		t.Skip("requires explicit disposable TEST_WORM_MINIO_ENDPOINT")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	storage, err := NewArchiveStorage(ctx, config.Config{
		ArchiveStorageEndpoint: endpoint, ArchiveStorageRegion: "us-east-1",
		ArchiveStorageBucket:    "portfolio-custody-recovery-" + uuid.NewString(),
		ArchiveStorageAccessKey: os.Getenv("TEST_WORM_MINIO_ACCESS_KEY"), ArchiveStorageSecretKey: os.Getenv("TEST_WORM_MINIO_SECRET_KEY"),
		ArchiveStorageUsePathStyle: true, ArchiveStorageCreateBucket: true, ArchiveStorageRequireObjectLock: true,
	})
	if err != nil {
		t.Fatalf("create isolated Object Lock bucket: %v", err)
	}
	content := []byte("portfolio custody recovery immutable evidence")
	digest := sha256.Sum256(content)
	metadata := map[string]string{"portfolio-custody-intent-id": uuid.NewString(), "tenant-code": "tenant-egueducation", "purpose": "portfolio_custody"}
	write := ImmutableArchiveWrite{
		Key: "recovery/" + uuid.NewString() + "/original.pdf", ContentType: "application/pdf", Body: strings.NewReader(string(content)), ContentLength: int64(len(content)),
		RetentionUntil: time.Now().UTC().Add(time.Hour), LegalHold: true, Metadata: metadata,
	}
	stored, err := storage.PutImmutableObject(ctx, write)
	if err != nil {
		t.Fatalf("write custody-held COMPLIANCE version: %v", err)
	}
	intent := CustodyHeldArchiveRecovery{Key: write.Key, VersionID: stored.VersionID, ETag: stored.ETag, ExpectedSHA256: hex.EncodeToString(digest[:]), ContentLength: int64(len(content)), ContentType: write.ContentType, Metadata: metadata}
	observed, err := storage.ReconcileCustodyHeldObject(ctx, intent)
	if err != nil {
		t.Fatalf("reconcile exact held version: %v", err)
	}
	if observed.VersionID != stored.VersionID || observed.ETag != stored.ETag || !observed.LegalHoldActive {
		t.Fatalf("exact recovery identity/hold changed: stored=%#v observed=%#v", stored, observed)
	}
	// All checks below must fail without mutating retention or legal hold.
	for name, mutate := range map[string]func(*CustodyHeldArchiveRecovery){
		"version":  func(v *CustodyHeldArchiveRecovery) { v.VersionID = "wrong-version" },
		"etag":     func(v *CustodyHeldArchiveRecovery) { v.ETag = "wrong-etag" },
		"size":     func(v *CustodyHeldArchiveRecovery) { v.ContentLength++ },
		"mime":     func(v *CustodyHeldArchiveRecovery) { v.ContentType = "text/plain" },
		"metadata": func(v *CustodyHeldArchiveRecovery) { v.Metadata = map[string]string{"purpose": "other"} },
		"sha256":   func(v *CustodyHeldArchiveRecovery) { v.ExpectedSHA256 = strings.Repeat("0", 64) },
	} {
		t.Run("rejects_"+name, func(t *testing.T) {
			bad := intent
			mutate(&bad)
			if _, err := storage.ReconcileCustodyHeldObject(ctx, bad); err == nil {
				t.Fatalf("recovery accepted mismatched %s", name)
			}
		})
	}
	confirmed, err := storage.ReconcileCustodyHeldObject(ctx, intent)
	if err != nil || confirmed.VersionID != stored.VersionID || confirmed.ETag != stored.ETag || !confirmed.LegalHoldActive {
		t.Fatalf("negative recovery attempts changed immutable object: object=%#v err=%v", confirmed, err)
	}
}

func TestPortfolioCustodyRecoveryWorkerDispositionsPostgresMinIOIntegration(t *testing.T) {
	endpoint := strings.TrimSpace(os.Getenv("TEST_WORM_MINIO_ENDPOINT"))
	if endpoint == "" {
		t.Skip("requires explicit disposable TEST_WORM_MINIO_ENDPOINT")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	it := newArchiveIntegrationDatabase(t)
	admin := openArchiveIntegrationPool(t, ctx, it.databaseConfig)
	defer admin.Close()
	if err := appdb.Migrate(ctx, admin); err != nil {
		t.Fatal(err)
	}
	grantArchiveIntegrationAccess(t, ctx, admin)
	seedRecoveryOperationActor(t, ctx, admin)
	storage, err := NewArchiveStorage(ctx, config.Config{ArchiveStorageEndpoint: endpoint, ArchiveStorageRegion: "us-east-1", ArchiveStorageBucket: "portfolio-custody-worker-" + uuid.NewString(), ArchiveStorageAccessKey: os.Getenv("TEST_WORM_MINIO_ACCESS_KEY"), ArchiveStorageSecretKey: os.Getenv("TEST_WORM_MINIO_SECRET_KEY"), ArchiveStorageUsePathStyle: true, ArchiveStorageCreateBucket: true, ArchiveStorageRequireObjectLock: true})
	if err != nil {
		t.Fatalf("create worker Object Lock bucket: %v", err)
	}
	sessions := appdb.NewSessionPool(it.readerPool)
	worker := NewPortfolioCustodyRecoveryWorker(sessions, storage, nil, time.Millisecond, 3)
	for _, disposition := range []string{"teacher_access", "institution_archive_only"} {
		t.Run(disposition, func(t *testing.T) {
			portfolio := seedCustodyIntentPortfolio(t, ctx, admin, "inst-001")
			var actor string
			if err := admin.QueryRow(ctx, `select u.sub from education_portfolios p join app_users u on u.id=p.owner_user_id where p.id=$1::uuid`, portfolio).Scan(&actor); err != nil {
				t.Fatal(err)
			}
			tenantCtx, release, err := appdb.AcquireRequestConn(ctx, it.readerPool, appdb.SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: actor})
			if err != nil {
				t.Fatal(err)
			}
			defer release()
			content := []byte("worker recovery " + disposition)
			digest := sha256.Sum256(content)
			intentID, documentID, versionID := uuid.NewString(), uuid.NewString(), uuid.NewString()
			metadata := map[string]string{"portfolio-custody-intent-id": intentID, "purpose": "portfolio_custody"}
			stored, err := storage.PutImmutableObject(tenantCtx, ImmutableArchiveWrite{Key: "worker/" + intentID + "/original.pdf", ContentType: "application/pdf", Body: strings.NewReader(string(content)), ContentLength: int64(len(content)), RetentionUntil: time.Now().UTC().Add(time.Hour), LegalHold: true, Metadata: metadata})
			if err != nil {
				t.Fatalf("write held %s object: %v", disposition, err)
			}
			metadataJSON, _ := json.Marshal(metadata)
			documentDate := "2026-09-12"
			expectedFingerprint := portfolioUploadFingerprint(archiveUploadPayload{Title: "Recovered custody evidence", DocumentDate: &documentDate, FileName: "evidence.pdf", ChecksumSHA256: hex.EncodeToString(digest[:]), FileSize: int64(len(content)), MimeType: "application/pdf"})
			if _, err = sessions.Exec(tenantCtx, `insert into portfolio_custody_upload_intents(id,tenant_code,institution_id,portfolio_id,actor_subject,idempotency_key,expected_sha256,expected_size_bytes,expected_mime_type,expected_request_fingerprint,expected_metadata,bucket_name,object_key,reserved_document_id,reserved_version_id) values($1::uuid,'tenant-egueducation','inst-001',$2::uuid,$3,$4,$5,$6,'application/pdf',$7,$8::jsonb,$9,$10,$11::uuid,$12::uuid)`, intentID, portfolio, actor, "worker-"+intentID, hex.EncodeToString(digest[:]), len(content), expectedFingerprint, metadataJSON, storage.Bucket(), stored.Key, documentID, versionID); err != nil {
				t.Fatal(err)
			}
			if _, err = sessions.Exec(tenantCtx, `update portfolio_custody_upload_intents set status='stored',stored_version_id=$1,stored_etag=$2,stored_size_bytes=$3 where id=$4::uuid`, stored.VersionID, stored.ETag, len(content), intentID); err != nil {
				t.Fatal(err)
			}
			operationCtx, releaseOperation, err := appdb.AcquireRequestConn(ctx, it.readerPool, appdb.SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: "archive-integration-test"})
			if err != nil {
				t.Fatal(err)
			}
			defer releaseOperation()
			op := insertRecoveryOperation(t, operationCtx, sessions, custodyIntentFixture{intentID: intentID, expectedFingerprint: expectedFingerprint}, "tenant-egueducation", "inst-001", portfolio, disposition)
			claim, err := worker.claim(ctx)
			if err != nil || claim == nil {
				t.Fatalf("claim %s: %#v %v", disposition, claim, err)
			}
			bound, rel, err := appdb.AcquireRequestConn(ctx, it.readerPool, appdb.SessionConfig{TenantID: claim.Tenant, InstitutionID: claim.Institution, ActorSubject: "portfolio-custody-recovery-worker"})
			if err != nil {
				t.Fatal(err)
			}
			err = worker.process(bound, claim)
			rel()
			if err != nil {
				t.Fatalf("process %s: %v", disposition, err)
			}
			var status string
			if err := sessions.QueryRow(tenantCtx, `select status from portfolio_custody_recovery_operations where id=$1::uuid`, op).Scan(&status); err != nil || status != "committed" {
				t.Fatalf("operation=%q err=%v", status, err)
			}
			var grants int
			if err := sessions.QueryRow(tenantCtx, `select count(*) from education_portfolio_archive_attachment_grants where archive_document_id=$1::uuid`, documentID).Scan(&grants); err != nil {
				t.Fatal(err)
			}
			want := 0
			if disposition == "teacher_access" {
				want = 1
			}
			if grants != want {
				t.Fatalf("%s grants=%d want=%d", disposition, grants, want)
			}
			if _, err := storage.ReconcileCustodyHeldObject(tenantCtx, CustodyHeldArchiveRecovery{Key: stored.Key, VersionID: stored.VersionID, ETag: stored.ETag, ExpectedSHA256: hex.EncodeToString(digest[:]), ContentLength: int64(len(content)), ContentType: "application/pdf", Metadata: metadata}); err != nil {
				t.Fatalf("worker mutated held object: %v", err)
			}
		})
	}
}
