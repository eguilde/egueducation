//go:build integration

package earchiva

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/eguilde/egueducation/internal/config"
	appdb "github.com/eguilde/egueducation/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestPortfolioRetentionDispositionReleaseExactVersionPostgresMinIOIntegration
// verifies the storage half of the irreversible boundary against a disposable
// Object-Lock bucket.  A deliberately short retention is necessary here: a
// valid COMPLIANCE object cannot be made expired by a test without waiting for
// its actual policy deadline.
func TestPortfolioRetentionDispositionReleaseExactVersionPostgresMinIOIntegration(t *testing.T) {
	endpoint := strings.TrimSpace(os.Getenv("TEST_WORM_MINIO_ENDPOINT"))
	if endpoint == "" {
		t.Skip("requires explicit disposable TEST_WORM_MINIO_ENDPOINT")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	storage, err := NewArchiveStorage(ctx, config.Config{
		ArchiveStorageEndpoint: endpoint, ArchiveStorageRegion: "us-east-1",
		ArchiveStorageBucket:    "retention-disp-" + strings.ReplaceAll(uuid.NewString()[:20], "-", ""),
		ArchiveStorageAccessKey: os.Getenv("TEST_WORM_MINIO_ACCESS_KEY"), ArchiveStorageSecretKey: os.Getenv("TEST_WORM_MINIO_SECRET_KEY"),
		ArchiveStorageUsePathStyle: true, ArchiveStorageCreateBucket: true, ArchiveStorageRequireObjectLock: true,
	})
	if err != nil {
		t.Fatalf("create isolated Object Lock bucket: %v", err)
	}
	content := []byte("expired portfolio retention disposition evidence")
	digest := sha256.Sum256(content)
	retainUntil := time.Now().UTC().Add(2 * time.Second)
	stored, err := storage.PutImmutableObject(ctx, ImmutableArchiveWrite{
		Key: "retention/" + uuid.NewString() + "/original.pdf", ContentType: "application/pdf", Body: strings.NewReader(string(content)), ContentLength: int64(len(content)),
		RetentionUntil: retainUntil, LegalHold: true, Metadata: map[string]string{"purpose": "portfolio_custody"},
	})
	if err != nil {
		t.Fatalf("write held compliance version: %v", err)
	}
	request := ReleaseExpiredPortfolioCustodyRequest{Key: stored.Key, VersionID: stored.VersionID, ETag: stored.ETag, ExpectedSHA256: hex.EncodeToString(digest[:]), ExpectedSize: int64(len(content)), ExpectedMIME: "application/pdf", RequiredRetention: retainUntil}
	// Every identity mismatch fails while the legal hold remains ON.
	for _, tc := range []struct {
		name   string
		mutate func(*ReleaseExpiredPortfolioCustodyRequest)
	}{
		{"sha256", func(r *ReleaseExpiredPortfolioCustodyRequest) { r.ExpectedSHA256 = strings.Repeat("0", 64) }},
		{"size", func(r *ReleaseExpiredPortfolioCustodyRequest) { r.ExpectedSize++ }},
		{"mime", func(r *ReleaseExpiredPortfolioCustodyRequest) { r.ExpectedMIME = "text/plain" }},
		{"version", func(r *ReleaseExpiredPortfolioCustodyRequest) { r.VersionID = "wrong-version" }},
	} {
		t.Run("rejects_"+tc.name+"_without_mutation", func(t *testing.T) {
			bad := request
			tc.mutate(&bad)
			if _, err := storage.ReleaseExpiredPortfolioCustody(ctx, bad); err == nil {
				t.Fatalf("release accepted wrong %s", tc.name)
			}
			state, err := storage.ReconcileCustodyHeldObject(ctx, CustodyHeldArchiveRecovery{Key: stored.Key, VersionID: stored.VersionID, ETag: stored.ETag, ExpectedSHA256: request.ExpectedSHA256, ContentLength: request.ExpectedSize, ContentType: request.ExpectedMIME, Metadata: map[string]string{"purpose": "portfolio_custody"}})
			if err != nil || !state.LegalHoldActive {
				t.Fatalf("failed release mutated held object: state=%#v err=%v", state, err)
			}
		})
	}
	for time.Now().UTC().Before(retainUntil.Add(time.Second)) {
		time.Sleep(100 * time.Millisecond)
	}
	state, err := storage.ReleaseExpiredPortfolioCustody(ctx, request)
	if err != nil || state.LegalHoldActive {
		t.Fatalf("release exact expired version: state=%#v err=%v", state, err)
	}
	// A completed call is restart-safe: a second worker observes OFF and must
	// not attempt a second mutation.
	state, err = storage.ReleaseExpiredPortfolioCustody(ctx, request)
	if err != nil || state.LegalHoldActive {
		t.Fatalf("idempotent restart after hold-off: state=%#v err=%v", state, err)
	}
	// Model the recovery branch where an S3 hold-off succeeded but its database
	// receipt rolled back, then an intervening legal/reference blocker appears.
	// The worker must converge the exact version back to a verified ON state
	// before it can close a blocked disposition.
	state, err = storage.ReconcilePortfolioCustodyLifecycle(ctx, PortfolioCustodyLifecycleRequest{
		Key: stored.Key, VersionID: stored.VersionID, ETag: stored.ETag, LegalHoldActive: true,
	})
	if err != nil || !state.LegalHoldActive {
		t.Fatalf("post-rollback blocker did not restore exact legal hold: state=%#v err=%v", state, err)
	}
}

func TestPortfolioRetentionDispositionDatabaseAuthorizationAndReleaseGuardsIntegration(t *testing.T) {
	it := newArchiveIntegrationDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	admin := openArchiveIntegrationPool(t, ctx, it.databaseConfig)
	defer admin.Close()
	if err := appdb.Migrate(ctx, admin); err != nil {
		t.Fatalf("migrate integration database: %v", err)
	}
	grantArchiveIntegrationAccess(t, ctx, admin)
	fx := seedRetentionDispositionFixture(t, ctx, admin, it.readerPool)
	defer fx.release()
	sessions := appdb.NewSessionPool(it.readerPool)

	// A same-tenant principal without the owner permission cannot submit a
	// request even when it knows all immutable transition facts.
	noPermission := seedRetentionTestUser(t, ctx, admin, "profesor-no-retention", "profesor")
	noPermissionCtx, releaseNoPermission, err := appdb.AcquireRequestConn(ctx, it.readerPool, appdb.SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: noPermission})
	if err != nil {
		t.Fatal(err)
	}
	defer releaseNoPermission()
	if _, err = sessions.Exec(noPermissionCtx, retentionRequestInsertSQL, fx.transitionID, `{"statement":"test"}`, noPermission); err == nil || !strings.Contains(err.Error(), "lacks exact expired transition provenance") && !strings.Contains(err.Error(), "portfolio retention") {
		t.Fatalf("same-tenant non-owner submitted disposition: %v", err)
	}

	requestID := insertRetentionDispositionRequest(t, fx.ownerCtx, sessions, fx.transitionID, fx.ownerSubject)
	var requesterUserID string
	if err = admin.QueryRow(ctx, `select id::text from app_users where sub=$1`, fx.ownerSubject).Scan(&requesterUserID); err != nil {
		t.Fatal(err)
	}
	for _, permission := range []string{"education.portfolios.custody.manage", "earchiva.manage"} {
		if _, err = admin.Exec(ctx, `insert into app_user_permissions(user_id,tenant_code,permission_code) values($1::uuid,'tenant-egueducation',$2) on conflict do nothing`, requesterUserID, permission); err != nil {
			t.Fatal(err)
		}
	}
	requesterUUIDCtx, releaseRequesterUUID, err := appdb.AcquireRequestConn(ctx, it.readerPool, appdb.SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: requesterUserID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = sessions.Exec(requesterUUIDCtx, `insert into portfolio_retention_disposition_decisions(request_id,decision,reason,decided_by_subject) values($1::uuid,'approved','same account alternate identity',$2)`, requestID, requesterUserID); err == nil || !strings.Contains(err.Error(), "separate approver") {
		t.Fatalf("requester approved through alternate UUID identity: %v", err)
	}
	releaseRequesterUUID()
	// A separate teacher is not a dual custody/archive approver.
	if _, err = sessions.Exec(noPermissionCtx, `insert into portfolio_retention_disposition_decisions(request_id,decision,reason,decided_by_subject) values($1::uuid,'approved','test',$2)`, requestID, noPermission); err == nil || !strings.Contains(err.Error(), "approver lacks current permissions") {
		t.Fatalf("single/zero-permission approver was accepted: %v", err)
	}

	// No worker operation GUC means neither a receipt nor an archive custody
	// projection can release the held version.
	workerCtx, releaseWorker, err := appdb.AcquireRequestConn(ctx, it.readerPool, appdb.SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: "portfolio-retention-disposition-worker"})
	if err != nil {
		t.Fatal(err)
	}
	defer releaseWorker()
	if _, err = sessions.Exec(workerCtx, `update archive_document_versions set custody_hold_active=false,legal_hold_active=false,retention_disposition_hold_active=false where id=$1::uuid`, fx.versionID); err == nil || !strings.Contains(err.Error(), "disposition") {
		t.Fatalf("archive custody release without operation GUC was accepted: %v", err)
	}
	var custody, legal, retentionDisposition bool
	if err = sessions.QueryRow(fx.ownerCtx, `select custody_hold_active,legal_hold_active,retention_disposition_hold_active from archive_document_versions where id=$1::uuid`, fx.versionID).Scan(&custody, &legal, &retentionDisposition); err != nil {
		t.Fatal(err)
	}
	if !custody || legal || retentionDisposition {
		t.Fatalf("failed DB release changed flags custody=%t legal=%t disposition=%t", custody, legal, retentionDisposition)
	}

	// A legal/reference blocker consumes no expiry evidence, closes the old
	// approval as blocked, and leaves the transition eligible for a new review
	// once the outside blocker has been resolved.
	approver := seedRetentionApprover(t, ctx, admin)
	approverCtx, releaseApprover, err := appdb.AcquireRequestConn(ctx, it.readerPool, appdb.SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: approver})
	if err != nil {
		t.Fatal(err)
	}
	defer releaseApprover()
	if _, err = sessions.Exec(approverCtx, `insert into portfolio_retention_disposition_decisions(request_id,decision,reason,decided_by_subject) values($1::uuid,'approved','independent approval',$2)`, requestID, approver); err != nil {
		t.Fatalf("record independent decision: %v", err)
	}
	if _, err = sessions.Exec(approverCtx, `update portfolio_retention_disposition_requests set status='approved' where id=$1::uuid`, requestID); err != nil {
		t.Fatalf("approve disposition: %v", err)
	}
	if _, err = sessions.Exec(approverCtx, `insert into portfolio_retention_disposition_operations(request_id,tenant_code,institution_id) select id,tenant_code,institution_id from portfolio_retention_disposition_requests where id=$1::uuid`, requestID); err != nil {
		t.Fatalf("enqueue approved disposition: %v", err)
	}
	if _, err = sessions.Exec(fx.ownerCtx, `update education_portfolios set legal_hold_active=true,legal_hold_reason='test blocker',legal_hold_set_at=now(),legal_hold_set_by_subject=$2 where id=(select portfolio_id from portfolio_retention_disposition_requests where id=$1::uuid)`, requestID, fx.ownerSubject); err != nil {
		t.Fatalf("set temporary legal blocker: %v", err)
	}
	blockingWorker := NewPortfolioRetentionDispositionWorker(sessions, &ArchiveStorage{bucket: "archive-test"}, nil, time.Millisecond, 3)
	claim, err := blockingWorker.claim(ctx)
	if err != nil || claim == nil {
		t.Fatalf("claim approved blocked disposition: claim=%#v err=%v", claim, err)
	}
	blockedWorkerCtx, releaseBlockedWorker, err := appdb.AcquireRequestConn(ctx, it.readerPool, appdb.SessionConfig{TenantID: claim.tenant, InstitutionID: claim.institution, ActorSubject: "portfolio-retention-disposition-worker"})
	if err != nil {
		t.Fatal(err)
	}
	err = blockingWorker.process(blockedWorkerCtx, claim)
	releaseBlockedWorker()
	if !errors.Is(err, errRetentionDispositionPermanentlyBlocked) {
		t.Fatalf("legal blocker must fail closed before storage: %v", err)
	}
	if err = blockingWorker.fail(ctx, claim, err); err != nil {
		t.Fatalf("record blocked disposition: %v", err)
	}
	var status string
	var consumed *time.Time
	if err = sessions.QueryRow(fx.ownerCtx, `select status from portfolio_retention_disposition_requests where id=$1::uuid`, requestID).Scan(&status); err != nil || status != "blocked" {
		t.Fatalf("blocked request status=%q err=%v", status, err)
	}
	if err = sessions.QueryRow(fx.ownerCtx, `select consumed_at from education_portfolio_retention_expiry_events where transition_id=$1::uuid`, fx.transitionID).Scan(&consumed); err != nil || consumed != nil {
		t.Fatalf("blocked review consumed expiry event=%v err=%v", consumed, err)
	}
	if _, err = sessions.Exec(fx.ownerCtx, `update education_portfolios set legal_hold_active=false,legal_hold_reason='released test blocker',legal_hold_set_at=now(),legal_hold_set_by_subject=$2 where id=(select portfolio_id from portfolio_retention_disposition_requests where id=$1::uuid)`, requestID, fx.ownerSubject); err != nil {
		t.Fatalf("release temporary legal blocker: %v", err)
	}
	if freshID := insertRetentionDispositionRequest(t, fx.ownerCtx, sessions, fx.transitionID, fx.ownerSubject); freshID == requestID {
		t.Fatal("fresh disposition request reused immutable blocked request")
	}
}

func TestPortfolioRetentionDispositionWorkerRestoresHoldAfterRollbackAndInterveningBlockerIntegration(t *testing.T) {
	endpoint := strings.TrimSpace(os.Getenv("TEST_WORM_MINIO_ENDPOINT"))
	if endpoint == "" {
		t.Skip("requires explicit disposable TEST_WORM_MINIO_ENDPOINT")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	defer cancel()
	it := newArchiveIntegrationDatabase(t)
	admin := openArchiveIntegrationPool(t, ctx, it.databaseConfig)
	defer admin.Close()
	if err := appdb.Migrate(ctx, admin); err != nil {
		t.Fatal(err)
	}
	grantArchiveIntegrationAccess(t, ctx, admin)
	storage, err := NewArchiveStorage(ctx, config.Config{ArchiveStorageEndpoint: endpoint, ArchiveStorageRegion: "us-east-1", ArchiveStorageBucket: "retention-work-" + strings.ReplaceAll(uuid.NewString()[:20], "-", ""), ArchiveStorageAccessKey: os.Getenv("TEST_WORM_MINIO_ACCESS_KEY"), ArchiveStorageSecretKey: os.Getenv("TEST_WORM_MINIO_SECRET_KEY"), ArchiveStorageUsePathStyle: true, ArchiveStorageCreateBucket: true, ArchiveStorageRequireObjectLock: true})
	if err != nil {
		t.Fatalf("create worker test object-lock bucket: %v", err)
	}
	sessions := appdb.NewSessionPool(it.readerPool)
	portfolioID := seedCustodyIntentPortfolio(t, ctx, admin, "inst-001")
	var owner string
	if err = admin.QueryRow(ctx, `select u.sub from education_portfolios p join app_users u on u.id=p.owner_user_id where p.id=$1::uuid`, portfolioID).Scan(&owner); err != nil {
		t.Fatal(err)
	}
	ownerCtx, releaseOwner, err := appdb.AcquireRequestConn(ctx, it.readerPool, appdb.SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: owner})
	if err != nil {
		t.Fatal(err)
	}
	defer releaseOwner()
	fixtureCtx, releaseFixture := archiveTenantContext(t, ctx, it.readerPool, "tenant-egueducation", "inst-001")
	defer releaseFixture()
	intentID, documentID, versionID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	content := []byte("retention worker exact projection")
	digest := sha256.Sum256(content)
	retentionUntil := time.Now().UTC().Add(2 * time.Second)
	stored, err := storage.PutImmutableObject(ctx, ImmutableArchiveWrite{Key: "worker/" + intentID + "/original.pdf", ContentType: "application/pdf", Body: strings.NewReader(string(content)), ContentLength: int64(len(content)), RetentionUntil: retentionUntil, LegalHold: true, Metadata: map[string]string{"portfolio-custody-intent-id": intentID, "purpose": "portfolio_custody"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = sessions.Exec(fixtureCtx, `insert into portfolio_custody_upload_intents(id,tenant_code,institution_id,portfolio_id,actor_subject,idempotency_key,expected_sha256,expected_size_bytes,expected_mime_type,expected_request_fingerprint,expected_metadata,bucket_name,object_key,reserved_document_id,reserved_version_id) values($1::uuid,'tenant-egueducation','inst-001',$2::uuid,'archive-integration-test',$3,$4,$5,'application/pdf',$6,$7::jsonb,$8,$9,$10::uuid,$11::uuid)`, intentID, portfolioID, "retention-worker-"+intentID, hex.EncodeToString(digest[:]), len(content), strings.Repeat("a", 64), `{"portfolio-custody-intent-id":"`+intentID+`","purpose":"portfolio_custody"}`, storage.Bucket(), stored.Key, documentID, versionID); err != nil {
		t.Fatal(err)
	}
	if _, err = sessions.Exec(fixtureCtx, `update portfolio_custody_upload_intents set status='stored',stored_version_id=$1,stored_etag=$2,stored_size_bytes=$3 where id=$4::uuid`, stored.VersionID, stored.ETag, len(content), intentID); err != nil {
		t.Fatal(err)
	}
	seedCustodyArchiveDocument(t, fixtureCtx, sessions, "inst-001", documentID)
	versionSQL := strings.Replace(custodyVersionInsertSQL, "custody_hold_active,portfolio_custody_intent_id", "custody_hold_active,retention_disposition_hold_active,portfolio_custody_intent_id", 1)
	versionSQL = strings.Replace(versionSQL, "true,$10::uuid)", "true,true,$10::uuid)", 1)
	if _, err = sessions.Exec(fixtureCtx, versionSQL, versionID, "inst-001", documentID, storage.Bucket(), stored.Key, hex.EncodeToString(digest[:]), int64(len(content)), stored.VersionID, stored.ETag, intentID); err != nil {
		t.Fatal(err)
	}
	if _, err = sessions.Exec(fixtureCtx, `update archive_documents set status='ready',current_version_no=1 where id=$1::uuid`, documentID); err != nil {
		t.Fatal(err)
	}
	if _, err = sessions.Exec(ownerCtx, lifecycleAttachmentSQL, portfolioID, documentID, documentID, versionID, storage.Bucket(), stored.Key, hex.EncodeToString(digest[:])); err != nil {
		t.Fatalf("attach exact worker lifecycle version: %v", err)
	}
	if _, err = sessions.Exec(ownerCtx, `update education_portfolios set activity_ceased_on='2020-01-01',activity_cessation_reason='worker disposition integration' where id=$1::uuid`, portfolioID); err != nil {
		t.Fatal(err)
	}
	operationID := uuid.NewString()
	var ceasedOn, retentionThrough time.Time
	if err = sessions.QueryRow(ownerCtx, `select activity_ceased_on,retention_until from education_portfolios where id=$1::uuid`, portfolioID).Scan(&ceasedOn, &retentionThrough); err != nil {
		t.Fatal(err)
	}
	if _, err = sessions.Exec(ownerCtx, `insert into education_portfolio_lifecycle_operations(id,tenant_code,institution_id,portfolio_id,operation_type,activity_ceased_on,retention_through,reason,requested_by_subject) values($1::uuid,'tenant-egueducation','inst-001',$2::uuid,'cessation_retention',$3,$4,'worker disposition integration',$5)`, operationID, portfolioID, ceasedOn, retentionThrough, owner); err != nil {
		t.Fatal(err)
	}
	if _, err = sessions.Exec(ownerCtx, `insert into education_portfolio_storage_transitions(operation_id,tenant_code,institution_id,portfolio_id,archive_document_id,archive_version_id,source_bucket,source_object_key,source_object_version_id,source_object_etag,source_sha256,source_size_bytes,required_retention_until) values($1::uuid,'tenant-egueducation','inst-001',$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,$8,$9,$10,(select (retention_until+1)::timestamp at time zone 'UTC' from education_portfolios where id=$2::uuid))`, operationID, portfolioID, documentID, versionID, storage.Bucket(), stored.Key, stored.VersionID, stored.ETag, hex.EncodeToString(digest[:]), len(content)); err != nil {
		t.Fatal(err)
	}
	var transitionID string
	if err = sessions.QueryRow(ownerCtx, `select id::text from education_portfolio_storage_transitions where operation_id=$1::uuid`, operationID).Scan(&transitionID); err != nil {
		t.Fatal(err)
	}
	workerCtx, releaseWorker, err := appdb.AcquireRequestConn(ctx, it.readerPool, appdb.SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: "portfolio-storage-lifecycle-worker"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = sessions.Exec(workerCtx, `update education_portfolio_storage_transitions set status='blocked',last_error='portfolio_retention_expired_review_required' where id=$1::uuid`, transitionID); err != nil {
		t.Fatal(err)
	}
	if _, err = sessions.Exec(workerCtx, `insert into education_portfolio_retention_expiry_events(transition_id,tenant_code,institution_id,required_retention_until,promoted_by_subject) select id,tenant_code,institution_id,required_retention_until,'portfolio-storage-lifecycle-worker' from education_portfolio_storage_transitions where id=$1::uuid`, transitionID); err != nil {
		t.Fatal(err)
	}
	releaseWorker()
	requestID := insertRetentionDispositionRequest(t, ownerCtx, sessions, transitionID, owner)
	approver := seedRetentionApprover(t, ctx, admin)
	approverCtx, releaseApprover, err := appdb.AcquireRequestConn(ctx, it.readerPool, appdb.SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: approver})
	if err != nil {
		t.Fatal(err)
	}
	defer releaseApprover()
	if _, err = sessions.Exec(approverCtx, `insert into portfolio_retention_disposition_decisions(request_id,decision,reason,decided_by_subject) values($1::uuid,'approved','independent approval',$2)`, requestID, approver); err != nil {
		t.Fatal(err)
	}
	if _, err = sessions.Exec(approverCtx, `update portfolio_retention_disposition_requests set status='approved' where id=$1::uuid`, requestID); err != nil {
		t.Fatal(err)
	}
	if _, err = sessions.Exec(approverCtx, `insert into portfolio_retention_disposition_operations(request_id,tenant_code,institution_id) select id,tenant_code,institution_id from portfolio_retention_disposition_requests where id=$1::uuid`, requestID); err != nil {
		t.Fatal(err)
	}
	for time.Now().UTC().Before(retentionUntil.Add(time.Second)) {
		time.Sleep(100 * time.Millisecond)
	}
	dispositionWorker := NewPortfolioRetentionDispositionWorker(sessions, storage, nil, time.Millisecond, 3)
	forcedRollback := true
	dispositionWorker.afterStorageRelease = func() error {
		if forcedRollback {
			forcedRollback = false
			return errors.New("forced post-storage database rollback")
		}
		return nil
	}
	claim, err := dispositionWorker.claim(ctx)
	if err != nil || claim == nil {
		t.Fatalf("claim exact approved release: claim=%#v err=%v", claim, err)
	}
	claimedCtx, releaseClaimed, err := appdb.AcquireRequestConn(ctx, it.readerPool, appdb.SessionConfig{TenantID: claim.tenant, InstitutionID: claim.institution, ActorSubject: "portfolio-retention-disposition-worker"})
	if err != nil {
		t.Fatal(err)
	}
	err = dispositionWorker.process(claimedCtx, claim)
	releaseClaimed()
	if err == nil || !strings.Contains(err.Error(), "forced post-storage database rollback") {
		t.Fatalf("expected forced rollback after exact hold release, got %v", err)
	}
	if err = dispositionWorker.fail(ctx, claim, err); err != nil {
		t.Fatalf("queue mutation-uncertain retry: %v", err)
	}
	if _, err = sessions.Exec(ownerCtx, `update education_portfolios set legal_hold_active=true,legal_hold_reason='intervening blocker',legal_hold_set_at=now(),legal_hold_set_by_subject=$2 where id=$1::uuid`, portfolioID, owner); err != nil {
		t.Fatalf("set intervening legal blocker: %v", err)
	}
	retryAfter := time.Now().UTC().Add(1100 * time.Millisecond)
	for time.Now().UTC().Before(retryAfter) {
		time.Sleep(100 * time.Millisecond)
	}
	retry, err := dispositionWorker.claim(ctx)
	if err != nil || retry == nil || retry.id != claim.id || retry.attempts < 2 {
		t.Fatalf("claim mutation-uncertain retry: claim=%#v err=%v", retry, err)
	}
	retryCtx, releaseRetry, err := appdb.AcquireRequestConn(ctx, it.readerPool, appdb.SessionConfig{TenantID: retry.tenant, InstitutionID: retry.institution, ActorSubject: "portfolio-retention-disposition-worker"})
	if err != nil {
		t.Fatal(err)
	}
	err = dispositionWorker.process(retryCtx, retry)
	releaseRetry()
	if !errors.Is(err, errRetentionDispositionPermanentlyBlocked) {
		t.Fatalf("intervening blocker must stop retry before release: %v", err)
	}
	if err = dispositionWorker.fail(ctx, retry, err); err != nil {
		t.Fatalf("restore exact hold and close blocked review: %v", err)
	}
	var custody, legal, disposition bool
	var requestStatus, operationStatus string
	var receipts int
	if err = sessions.QueryRow(ownerCtx, `select custody_hold_active,legal_hold_active,retention_disposition_hold_active from archive_document_versions where id=$1::uuid`, versionID).Scan(&custody, &legal, &disposition); err != nil {
		t.Fatal(err)
	}
	if !custody || legal || !disposition {
		t.Fatalf("blocked projection flags custody=%t legal=%t disposition=%t", custody, legal, disposition)
	}
	if err = sessions.QueryRow(ownerCtx, `select status from portfolio_retention_disposition_requests where id=$1::uuid`, requestID).Scan(&requestStatus); err != nil || requestStatus != "blocked" {
		t.Fatalf("request status=%q err=%v", requestStatus, err)
	}
	if err = sessions.QueryRow(ownerCtx, `select status from portfolio_retention_disposition_operations where request_id=$1::uuid`, requestID).Scan(&operationStatus); err != nil || operationStatus != "blocked" {
		t.Fatalf("operation status=%q err=%v", operationStatus, err)
	}
	if err = sessions.QueryRow(ownerCtx, `select count(*) from portfolio_retention_disposition_receipts where request_id=$1::uuid and outcome='blocked'`, requestID).Scan(&receipts); err != nil || receipts != 1 {
		t.Fatalf("receipt count=%d err=%v", receipts, err)
	}
	hold, err := storage.client.GetObjectLegalHold(ctx, &s3.GetObjectLegalHoldInput{Bucket: aws.String(storage.Bucket()), Key: aws.String(stored.Key), VersionId: aws.String(stored.VersionID)})
	if err != nil || hold.LegalHold == nil || hold.LegalHold.Status != s3types.ObjectLockLegalHoldStatusOn {
		t.Fatalf("exact WORM hold was not restored ON: hold=%#v err=%v", hold.LegalHold, err)
	}
}

func TestPortfolioRetentionDispositionRestoreFailureCannotCloseBlockedReceiptIntegration(t *testing.T) {
	it := newArchiveIntegrationDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	defer cancel()
	admin := openArchiveIntegrationPool(t, ctx, it.databaseConfig)
	defer admin.Close()
	if err := appdb.Migrate(ctx, admin); err != nil {
		t.Fatal(err)
	}
	grantArchiveIntegrationAccess(t, ctx, admin)
	sessions := appdb.NewSessionPool(it.readerPool)
	fx := seedRetentionDispositionFixture(t, ctx, admin, it.readerPool)
	defer fx.release()
	requestID := insertRetentionDispositionRequest(t, fx.ownerCtx, sessions, fx.transitionID, fx.ownerSubject)
	approver := seedRetentionApprover(t, ctx, admin)
	approverCtx, releaseApprover, err := appdb.AcquireRequestConn(ctx, it.readerPool, appdb.SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: approver})
	if err != nil {
		t.Fatal(err)
	}
	defer releaseApprover()
	if _, err = sessions.Exec(approverCtx, `insert into portfolio_retention_disposition_decisions(request_id,decision,reason,decided_by_subject) values($1::uuid,'approved','restore failure proof',$2)`, requestID, approver); err != nil {
		t.Fatal(err)
	}
	if _, err = sessions.Exec(approverCtx, `update portfolio_retention_disposition_requests set status='approved' where id=$1::uuid`, requestID); err != nil {
		t.Fatal(err)
	}
	if _, err = sessions.Exec(approverCtx, `insert into portfolio_retention_disposition_operations(request_id,tenant_code,institution_id) select id,tenant_code,institution_id from portfolio_retention_disposition_requests where id=$1::uuid`, requestID); err != nil {
		t.Fatal(err)
	}
	worker := NewPortfolioRetentionDispositionWorker(sessions, &ArchiveStorage{bucket: "wrong-bucket", enabled: true, client: &s3.Client{}}, nil, time.Millisecond, 3)
	first, err := worker.claim(ctx)
	if err != nil || first == nil {
		t.Fatalf("claim first attempt: claim=%#v err=%v", first, err)
	}
	if err = worker.fail(ctx, first, errors.New("simulated transport interruption")); err != nil {
		t.Fatalf("queue uncertain attempt: %v", err)
	}
	retryAfter := time.Now().UTC().Add(1100 * time.Millisecond)
	for time.Now().UTC().Before(retryAfter) {
		time.Sleep(100 * time.Millisecond)
	}
	retry, err := worker.claim(ctx)
	if err != nil || retry == nil || retry.attempts < 2 {
		t.Fatalf("claim uncertain retry: claim=%#v err=%v", retry, err)
	}
	retryCtx, releaseRetry, err := appdb.AcquireRequestConn(ctx, it.readerPool, appdb.SessionConfig{TenantID: retry.tenant, InstitutionID: retry.institution, ActorSubject: "portfolio-retention-disposition-worker"})
	if err != nil {
		t.Fatal(err)
	}
	processErr := worker.process(retryCtx, retry)
	releaseRetry()
	if !errors.Is(processErr, errRetentionDispositionPermanentlyBlocked) {
		t.Fatalf("expected permanent exact-storage mismatch, got %v", processErr)
	}
	if err = worker.fail(ctx, retry, processErr); err == nil || !strings.Contains(err.Error(), "restore blocked disposition hold") {
		t.Fatalf("restoration failure must remain uncommitted, got %v", err)
	}
	var requestStatus, operationStatus string
	var receipts int
	if err = sessions.QueryRow(fx.ownerCtx, `select status from portfolio_retention_disposition_requests where id=$1::uuid`, requestID).Scan(&requestStatus); err != nil || requestStatus != "approved" {
		t.Fatalf("failed restoration changed request status=%q err=%v", requestStatus, err)
	}
	if err = sessions.QueryRow(fx.ownerCtx, `select status from portfolio_retention_disposition_operations where request_id=$1::uuid`, requestID).Scan(&operationStatus); err != nil || operationStatus != "leased" {
		t.Fatalf("failed restoration changed operation status=%q err=%v", operationStatus, err)
	}
	if err = sessions.QueryRow(fx.ownerCtx, `select count(*) from portfolio_retention_disposition_receipts where request_id=$1::uuid`, requestID).Scan(&receipts); err != nil || receipts != 0 {
		t.Fatalf("failed restoration persisted misleading receipts=%d err=%v", receipts, err)
	}
}

func TestPortfolioRetentionDispositionRetryBackoffDoesNotStarveQueueIntegration(t *testing.T) {
	it := newArchiveIntegrationDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	defer cancel()
	admin := openArchiveIntegrationPool(t, ctx, it.databaseConfig)
	defer admin.Close()
	if err := appdb.Migrate(ctx, admin); err != nil {
		t.Fatal(err)
	}
	grantArchiveIntegrationAccess(t, ctx, admin)
	sessions := appdb.NewSessionPool(it.readerPool)
	first := seedRetentionDispositionFixture(t, ctx, admin, it.readerPool)
	defer first.release()
	second := seedRetentionDispositionFixture(t, ctx, admin, it.readerPool)
	defer second.release()
	approver := seedRetentionApprover(t, ctx, admin)
	approverCtx, releaseApprover, err := appdb.AcquireRequestConn(ctx, it.readerPool, appdb.SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: approver})
	if err != nil {
		t.Fatal(err)
	}
	defer releaseApprover()
	enqueue := func(fx retentionDispositionFixture) string {
		requestID := insertRetentionDispositionRequest(t, fx.ownerCtx, sessions, fx.transitionID, fx.ownerSubject)
		if _, insertErr := sessions.Exec(approverCtx, `insert into portfolio_retention_disposition_decisions(request_id,decision,reason,decided_by_subject) values($1::uuid,'approved','queue fairness proof',$2)`, requestID, approver); insertErr != nil {
			t.Fatal(insertErr)
		}
		if _, updateErr := sessions.Exec(approverCtx, `update portfolio_retention_disposition_requests set status='approved' where id=$1::uuid`, requestID); updateErr != nil {
			t.Fatal(updateErr)
		}
		var operationID string
		if queryErr := sessions.QueryRow(approverCtx, `insert into portfolio_retention_disposition_operations(request_id,tenant_code,institution_id) select id,tenant_code,institution_id from portfolio_retention_disposition_requests where id=$1::uuid returning id::text`, requestID).Scan(&operationID); queryErr != nil {
			t.Fatal(queryErr)
		}
		return operationID
	}
	firstOperation := enqueue(first)
	time.Sleep(10 * time.Millisecond)
	secondOperation := enqueue(second)
	worker := NewPortfolioRetentionDispositionWorker(sessions, &ArchiveStorage{bucket: "archive-test"}, nil, time.Millisecond, 3)
	claim, err := worker.claim(ctx)
	if err != nil || claim == nil || claim.id != firstOperation {
		t.Fatalf("claim oldest operation: claim=%#v err=%v", claim, err)
	}
	if err = worker.fail(ctx, claim, errors.New("temporary storage transport outage")); err != nil {
		t.Fatal(err)
	}
	var delayed bool
	if err = sessions.QueryRow(first.ownerCtx, `select status='queued' and available_at>now() from portfolio_retention_disposition_operations where id=$1::uuid`, firstOperation).Scan(&delayed); err != nil || !delayed {
		t.Fatalf("retry was not delayed: delayed=%t err=%v", delayed, err)
	}
	claim, err = worker.claim(ctx)
	if err != nil || claim == nil || claim.id != secondOperation {
		t.Fatalf("delayed oldest operation starved newer work: claim=%#v err=%v", claim, err)
	}
}

type retentionDispositionFixture struct {
	ownerCtx                              context.Context
	release                               func()
	ownerSubject, transitionID, versionID string
}

func seedRetentionDispositionFixture(t *testing.T, ctx context.Context, admin *pgxpool.Pool, readerPool *pgxpool.Pool) retentionDispositionFixture {
	t.Helper()
	portfolioID := seedCustodyIntentPortfolio(t, ctx, admin, "inst-001")
	var owner string
	if err := admin.QueryRow(ctx, `select u.sub from education_portfolios p join app_users u on u.id=p.owner_user_id where p.id=$1::uuid`, portfolioID).Scan(&owner); err != nil {
		t.Fatalf("load fixture owner: %v", err)
	}
	ownerCtx, release, err := appdb.AcquireRequestConn(ctx, readerPool, appdb.SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: owner})
	if err != nil {
		t.Fatalf("bind owner context: %v", err)
	}
	sessions := appdb.NewSessionPool(readerPool)
	fixtureCtx, releaseFixture := archiveTenantContext(t, ctx, readerPool, "tenant-egueducation", "inst-001")
	defer releaseFixture()
	intent := seedCustodyUploadIntent(t, fixtureCtx, sessions, "tenant-egueducation", "inst-001", portfolioID)
	markCustodyIntentStored(t, fixtureCtx, sessions, intent)
	seedCustodyArchiveDocument(t, fixtureCtx, sessions, "inst-001", intent.documentID)
	if _, err := sessions.Exec(ownerCtx, custodyVersionInsertSQL, intent.versionID, "inst-001", intent.documentID, intent.bucket, intent.objectKey, strings.Repeat("a", 64), int64(3), intent.storageVersion, intent.etag, intent.intentID); err != nil {
		release()
		t.Fatalf("seed retention archive version: %v", err)
	}
	if _, err := sessions.Exec(ownerCtx, `update archive_documents set status='ready',current_version_no=1 where id=$1::uuid`, intent.documentID); err != nil {
		release()
		t.Fatalf("activate retention archive document: %v", err)
	}
	attachLifecycleVersion(t, ownerCtx, sessions, portfolioID, intent)
	if _, err := sessions.Exec(ownerCtx, `update education_portfolios set activity_ceased_on='2020-01-01',activity_cessation_reason='retention integration fixture' where id=$1::uuid`, portfolioID); err != nil {
		release()
		t.Fatalf("cease retention fixture portfolio: %v", err)
	}
	operationID := seedLifecycleTransition(t, ownerCtx, sessions, portfolioID, intent, "cessation_retention", nil)
	var transitionID string
	if err := sessions.QueryRow(ownerCtx, `select id::text from education_portfolio_storage_transitions where operation_id=$1::uuid`, operationID).Scan(&transitionID); err != nil {
		release()
		t.Fatalf("read retention fixture transition: %v", err)
	}
	workerCtx, workerRelease, err := appdb.AcquireRequestConn(ctx, readerPool, appdb.SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: "portfolio-storage-lifecycle-worker"})
	if err != nil {
		release()
		t.Fatalf("bind lifecycle worker context: %v", err)
	}
	defer workerRelease()
	if _, err := sessions.Exec(workerCtx, `update education_portfolio_storage_transitions set status='blocked',last_error='portfolio_retention_expired_review_required' where id=$1::uuid`, transitionID); err != nil {
		release()
		t.Fatalf("mark expired retention transition: %v", err)
	}
	if _, err := sessions.Exec(workerCtx, `insert into education_portfolio_retention_expiry_events(transition_id,tenant_code,institution_id,required_retention_until,promoted_by_subject) select id,tenant_code,institution_id,required_retention_until,'portfolio-storage-lifecycle-worker' from education_portfolio_storage_transitions where id=$1::uuid`, transitionID); err != nil {
		release()
		t.Fatalf("record expired retention event: %v", err)
	}
	return retentionDispositionFixture{ownerCtx: ownerCtx, release: release, ownerSubject: owner, transitionID: transitionID, versionID: intent.versionID}
}

const retentionRequestInsertSQL = `insert into portfolio_retention_disposition_requests(
	transition_id,tenant_code,institution_id,portfolio_id,archive_document_id,archive_version_id,
	source_bucket,source_object_key,source_object_version_id,source_object_etag,source_sha256,source_size_bytes,source_mime_type,required_retention_until,evidence,requested_by_subject)
	select t.id,t.tenant_code,t.institution_id,t.portfolio_id,t.archive_document_id,t.archive_version_id,
	t.source_bucket,t.source_object_key,t.source_object_version_id,t.source_object_etag,t.source_sha256,t.source_size_bytes,v.mime_type,t.required_retention_until,$2::jsonb,$3
	from education_portfolio_storage_transitions t join archive_document_versions v on v.id=t.archive_version_id and v.institution_id=t.institution_id
	where t.id=$1::uuid returning id::text`

func insertRetentionDispositionRequest(t *testing.T, ctx context.Context, sessions *appdb.SessionPool, transitionID, subject string) string {
	t.Helper()
	var id string
	if err := sessions.QueryRow(ctx, retentionRequestInsertSQL, transitionID, `{"statement":"independent review evidence"}`, subject).Scan(&id); err != nil {
		t.Fatalf("submit valid retention disposition request: %v", err)
	}
	return id
}

func seedRetentionTestUser(t *testing.T, ctx context.Context, admin *pgxpool.Pool, prefix, position string) string {
	t.Helper()
	id := uuid.NewString()
	subject := prefix + "-" + strings.ReplaceAll(id[:8], "-", "")
	if _, err := admin.Exec(ctx, `insert into app_users(id,sub,name,email,phone_number,locale,status) values($1::uuid,$2,$2,$2||'@example.test','','ro','active')`, id, subject); err != nil {
		t.Fatalf("seed retention test user: %v", err)
	}
	if _, err := admin.Exec(ctx, `insert into app_memberships(user_id,tenant_code,position_code,org_unit_code,organization_name,is_primary,active,start_date) values($1::uuid,'tenant-egueducation',$2,'unit-root','Retention Test School',true,true,current_date)`, id, position); err != nil {
		t.Fatalf("seed retention test membership: %v", err)
	}
	return subject
}

func seedRetentionApprover(t *testing.T, ctx context.Context, admin *pgxpool.Pool) string {
	t.Helper()
	subject := seedRetentionTestUser(t, ctx, admin, "retention-approver", "director")
	var userID string
	if err := admin.QueryRow(ctx, `select id::text from app_users where sub=$1`, subject).Scan(&userID); err != nil {
		t.Fatalf("read retention approver: %v", err)
	}
	for _, permission := range []string{"education.portfolios.custody.manage", "earchiva.manage"} {
		if _, err := admin.Exec(ctx, `insert into app_user_permissions(user_id,permission_code,tenant_code) values($1::uuid,$2,'tenant-egueducation') on conflict do nothing`, userID, permission); err != nil {
			t.Fatalf("grant retention approver %s: %v", permission, err)
		}
	}
	return subject
}
