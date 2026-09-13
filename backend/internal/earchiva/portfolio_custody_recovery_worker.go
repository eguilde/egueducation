package earchiva

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/eguilde/egueducation/internal/audit"
	appdb "github.com/eguilde/egueducation/internal/db"
)

// PortfolioCustodyRecoveryWorker reads and verifies a held version only. It
// has no storage mutation capability: adoption is a database projection after
// ReconcileCustodyHeldObject proves exact immutable provenance.
type PortfolioCustodyRecoveryWorker struct {
	pool        *appdb.SessionPool
	storage     *ArchiveStorage
	logger      *zap.Logger
	workerID    string
	poll        time.Duration
	maxAttempts int
	mu          sync.Mutex
	started     bool
}
type custodyRecoveryClaim struct {
	ID, Tenant, Institution string
	Attempts                int
}

func NewPortfolioCustodyRecoveryWorker(pool *appdb.SessionPool, storage *ArchiveStorage, logger *zap.Logger, poll time.Duration, maxAttempts int) *PortfolioCustodyRecoveryWorker {
	if poll <= 0 {
		poll = 5 * time.Second
	}
	if maxAttempts < 1 || maxAttempts > 20 {
		maxAttempts = 5
	}
	return &PortfolioCustodyRecoveryWorker{pool: pool, storage: storage, logger: logger, workerID: uuid.NewString(), poll: poll, maxAttempts: maxAttempts}
}
func (w *PortfolioCustodyRecoveryWorker) Enabled() bool {
	return w != nil && w.pool != nil && w.pool.Raw() != nil && w.storage != nil && w.storage.Enabled()
}
func (w *PortfolioCustodyRecoveryWorker) Start(ctx context.Context) {
	if !w.Enabled() {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.started {
		return
	}
	w.started = true
	go w.run(ctx)
}
func (w *PortfolioCustodyRecoveryWorker) run(ctx context.Context) {
	ticker := time.NewTicker(w.poll)
	defer ticker.Stop()
	for {
		_ = w.drain(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
func (w *PortfolioCustodyRecoveryWorker) drain(ctx context.Context) error {
	for {
		claim, err := w.claim(ctx)
		if err != nil {
			return err
		}
		if claim == nil {
			return nil
		}
		tenantCtx, release, err := appdb.AcquireRequestConn(ctx, w.pool.Raw(), appdb.SessionConfig{TenantID: claim.Tenant, InstitutionID: claim.Institution, ActorSubject: "portfolio-custody-recovery-worker"})
		if err == nil {
			err = w.process(tenantCtx, claim)
			release()
		}
		if err != nil {
			_ = w.fail(ctx, claim, err)
		}
	}
}
func (w *PortfolioCustodyRecoveryWorker) claim(ctx context.Context) (*custodyRecoveryClaim, error) {
	claimCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	claimCtx, release, err := appdb.AcquireRequestConn(claimCtx, w.pool.Raw(), appdb.SessionConfig{ActorSubject: "portfolio-custody-recovery-worker", IsSuperAdmin: true})
	if err != nil {
		return nil, err
	}
	defer release()
	tx, err := w.pool.Begin(claimCtx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(claimCtx) //nolint:errcheck
	if _, err = tx.Exec(claimCtx, `select set_config('app.portfolio_custody_recovery_worker_id',$1,true)`, w.workerID); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(claimCtx, `update portfolio_custody_recovery_operations set status='queued',lease_owner='',lease_expires_at=null,last_error_code='recovery_lease_expired' where status='leased' and lease_expires_at<now()`); err != nil {
		return nil, err
	}
	var c custodyRecoveryClaim
	err = tx.QueryRow(claimCtx, `with next as (select o.id from portfolio_custody_recovery_operations o join app_tenants t on t.code=o.tenant_code and t.institution_id=o.institution_id and t.active where o.status='queued' order by o.created_at,o.id for update skip locked limit 1) update portfolio_custody_recovery_operations o set status='leased',attempts=attempts+1,lease_owner=$1,lease_expires_at=now()+interval '5 minutes' from next where o.id=next.id returning o.id::text,o.tenant_code,o.institution_id,o.attempts`, w.workerID).Scan(&c.ID, &c.Tenant, &c.Institution, &c.Attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, tx.Commit(claimCtx)
	}
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec(claimCtx, `insert into portfolio_custody_recovery_attempts(operation_id,attempt_no,worker_id,outcome) values($1::uuid,$2,$3,'claimed')`, c.ID, c.Attempts, w.workerID); err != nil {
		return nil, err
	}
	return &c, tx.Commit(claimCtx)
}
func (w *PortfolioCustodyRecoveryWorker) process(ctx context.Context, c *custodyRecoveryClaim) error {
	var intentID, portfolioID, actor, disposition, title, fileName, checksum, mime, bucket, key, version, etag, operationFingerprint, intentFingerprint string
	var size int64
	var metadata []byte
	var date *time.Time
	const recoveryIntentQuery = `select i.id::text,i.portfolio_id::text,i.actor_subject,o.disposition,o.title,o.original_file_name,o.document_date,i.expected_sha256,i.expected_mime_type,i.bucket_name,i.object_key,i.stored_version_id,i.stored_etag,i.expected_size_bytes,i.expected_metadata,o.expected_fingerprint,i.expected_request_fingerprint from portfolio_custody_recovery_operations o join portfolio_custody_upload_intents i on i.id=o.intent_id where o.id=$1::uuid and o.status='leased' and o.lease_owner=$2 and i.status='stored' and i.final_disposition is null`
	load := func(row pgx.Row) error {
		return row.Scan(&intentID, &portfolioID, &actor, &disposition, &title, &fileName, &date, &checksum, &mime, &bucket, &key, &version, &etag, &size, &metadata, &operationFingerprint, &intentFingerprint)
	}
	err := load(w.pool.QueryRow(ctx, recoveryIntentQuery, c.ID, w.workerID))
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("recovery_provenance_missing")
	}
	if err != nil {
		return err
	}
	if operationFingerprint != intentFingerprint {
		return fmt.Errorf("recovery_operation_fingerprint_mismatch")
	}
	var meta map[string]string
	if err = json.Unmarshal(metadata, &meta); err != nil {
		return fmt.Errorf("recovery_metadata_invalid")
	}
	if bucket != w.storage.Bucket() {
		return fmt.Errorf("recovery_bucket_mismatch")
	}
	storageCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	object, err := w.storage.ReconcileCustodyHeldObject(storageCtx, CustodyHeldArchiveRecovery{Key: key, VersionID: version, ETag: etag, ExpectedSHA256: checksum, ContentLength: size, ContentType: mime, Metadata: meta})
	if err != nil {
		return fmt.Errorf("recovery_storage_verification_failed: %w", err)
	}
	// Storage verification is deliberately outside the transaction. Before any
	// projection is written, acquire fresh row locks and prove the lease and
	// stored intent are still current; a restarted worker cannot adopt stale work.
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err = tx.Exec(ctx, `select set_config('app.portfolio_custody_recovery_worker_id',$1,true),set_config('app.portfolio_custody_recovery_operation_id',$2,true)`, w.workerID, c.ID); err != nil {
		return err
	}
	if err = load(tx.QueryRow(ctx, recoveryIntentQuery+` for update of o,i`, c.ID, w.workerID)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("recovery_lease_or_provenance_changed")
		}
		return err
	}
	if operationFingerprint != intentFingerprint {
		return fmt.Errorf("recovery_operation_fingerprint_mismatch")
	}
	var owner string
	if disposition == "teacher_access" {
		if err = archiveAdmissionActorPermission(ctx, tx, actor, "education.portfolios.manage_own"); err != nil {
			return fmt.Errorf("recovery_teacher_authority_absent")
		}
		err = tx.QueryRow(ctx, `select owner_user_id::text from education_portfolios where id=$1::uuid and institution_id=$2 and (owner_user_id::text=$3 or exists(select 1 from app_users u where u.id=owner_user_id and lower(u.sub)=lower($3))) and status in ('draft','returned') and withdrawn_at is null and not legal_hold_active for update`, portfolioID, c.Institution, actor).Scan(&owner)
		if err != nil {
			return fmt.Errorf("recovery_teacher_authority_absent")
		}
	}
	documentID, versionID := "", ""
	if err = tx.QueryRow(ctx, `select reserved_document_id::text,reserved_version_id::text from portfolio_custody_upload_intents where id=$1::uuid`, intentID).Scan(&documentID, &versionID); err != nil {
		return err
	}
	metadataJSON, _ := json.Marshal(map[string]any{"portfolio_custody_intent_id": intentID, "final_disposition": disposition, "recovery_operation_id": c.ID})
	if _, err = tx.Exec(ctx, `insert into archive_documents(id,institution_id,title,original_file_name,mime_type,source_kind,source_system,external_reference,status,original_bucket,original_object_key,artifact_bucket,artifact_object_key,document_date,metadata,idempotency_key,created_by,current_version_no,received_at) values($1::uuid,$2,$3,$4,$5,'upload','portfolio_custody_recovery','', 'queued',$6,$7,$6,$7,$8,$9::jsonb,$10,$11,1,now())`, documentID, c.Institution, title, fileName, mime, bucket, key, date, metadataJSON, "portfolio-custody-recovery:"+c.ID, actor); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `insert into archive_document_versions(id,institution_id,document_id,version_no,mime_type,title,bucket_name,object_key,hash_sha256,size_bytes,metadata,ocr_text,status,source_bucket,source_object_key,artifact_bucket,artifact_object_key,source_sha256,source_size_bytes,page_count,text_status,extracted_text,extracted_metadata,created_by,source_object_version_id,source_object_etag,legal_hold_active,custody_hold_active,portfolio_custody_intent_id) values($1::uuid,$2,$3::uuid,1,$4,$5,$6,$7,$8,$9,$10::jsonb,'','active',$6,$7,$6,$7,$8,$9,0,'pending','',$10::jsonb,$11,$12,$13,true,true,$14::uuid)`, versionID, c.Institution, documentID, mime, title, bucket, key, checksum, size, metadataJSON, actor, version, etag, intentID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `insert into archive_ingestion_jobs(id,institution_id,document_id,version_id,job_type,status,available_at,created_by) values($1::uuid,$2,$3::uuid,$4::uuid,'extract_text','pending',now(),$5)`, uuid.NewString(), c.Institution, documentID, versionID, actor); err != nil {
		return err
	}
	if disposition == "teacher_access" {
		if _, err = tx.Exec(ctx, `insert into education_portfolio_archive_attachment_grants(institution_id,archive_document_id,grantee_user_id,granted_by_user_id) values($1,$2::uuid,$3::uuid,$3::uuid)`, c.Institution, documentID, owner); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `insert into portfolio_custody_recovery_attempts(operation_id,attempt_no,worker_id,outcome) values($1::uuid,$2,$3,'committed')`, c.ID, c.Attempts, w.workerID); err != nil {
		return err
	}
	intentTag, err := tx.Exec(ctx, `update portfolio_custody_upload_intents set status='committed',final_disposition=$1,recovery_committed_at=now() where id=$2::uuid and status='stored' and final_disposition is null`, disposition, intentID)
	if err != nil {
		return err
	}
	if intentTag.RowsAffected() != 1 {
		return fmt.Errorf("recovery intent commit affected %d rows", intentTag.RowsAffected())
	}
	operationTag, err := tx.Exec(ctx, `update portfolio_custody_recovery_operations set status='committed',committed_at=now(),lease_owner='',lease_expires_at=null,last_error_code='' where id=$1::uuid and status='leased' and lease_owner=$2`, c.ID, w.workerID)
	if err != nil {
		return err
	}
	if operationTag.RowsAffected() != 1 {
		return fmt.Errorf("recovery operation commit affected %d rows", operationTag.RowsAffected())
	}
	if err = audit.Log(ctx, tx, audit.Event{ActorSubject: "portfolio-custody-recovery-worker", Action: "earchiva.portfolio_custody.recovered", TargetType: "portfolio_custody_recovery_operation", TargetID: c.ID, Summary: "Exact held version adopted.", Details: map[string]any{"intent_id": intentID, "disposition": disposition, "document_id": documentID, "version_id": versionID, "storage_version_id": object.VersionID}}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (w *PortfolioCustodyRecoveryWorker) fail(ctx context.Context, c *custodyRecoveryClaim, cause error) error {
	tenantCtx, release, err := appdb.AcquireRequestConn(ctx, w.pool.Raw(), appdb.SessionConfig{TenantID: c.Tenant, InstitutionID: c.Institution, ActorSubject: "portfolio-custody-recovery-worker"})
	if err != nil {
		return err
	}
	defer release()
	code, permanent := recoveryFailureCode(cause)
	status := "queued"
	if permanent {
		status = "blocked"
	} else if c.Attempts >= w.maxAttempts {
		status = "deadletter"
	}
	tx, err := w.pool.Begin(tenantCtx)
	if err != nil {
		return err
	}
	defer tx.Rollback(tenantCtx) //nolint:errcheck
	if _, err = tx.Exec(tenantCtx, `select set_config('app.portfolio_custody_recovery_worker_id',$1,true),set_config('app.portfolio_custody_recovery_operation_id',$2,true)`, w.workerID, c.ID); err != nil {
		return err
	}
	outcome := "retry"
	if status == "blocked" {
		outcome = "blocked"
	}
	if status == "deadletter" {
		outcome = "deadletter"
	}
	_, err = tx.Exec(tenantCtx, `insert into portfolio_custody_recovery_attempts(operation_id,attempt_no,worker_id,outcome,error_code) values($1::uuid,$2,$3,$4,$5)`, c.ID, c.Attempts, w.workerID, outcome, code)
	if err != nil {
		return err
	}
	tag, err := tx.Exec(tenantCtx, `update portfolio_custody_recovery_operations set status=$1,lease_owner='',lease_expires_at=null,last_error_code=$2 where id=$3::uuid and status='leased' and lease_owner=$4`, status, code, c.ID, w.workerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("recovery failure transition affected %d rows", tag.RowsAffected())
	}
	if status == "deadletter" {
		_, err = tx.Exec(tenantCtx, `insert into portfolio_custody_recovery_deadletters(operation_id,tenant_code,institution_id,error_code) values($1::uuid,$2,$3,$4) on conflict(operation_id) do nothing`, c.ID, c.Tenant, c.Institution, code)
		if err != nil {
			return err
		}
	}
	return tx.Commit(tenantCtx)
}

func recoveryFailureCode(cause error) (string, bool) {
	message := strings.ToLower(cause.Error())
	code := strings.Split(message, ":")[0]
	// An unavailable S3 endpoint or timed-out read is retryable. A completed
	// verification that disproves the persisted provenance is not: retries
	// cannot make a different version, hash, hold, or metadata admissible.
	if strings.Contains(message, "provenance mismatch") ||
		strings.Contains(message, "content mismatch") ||
		strings.Contains(message, "identity or size mismatch") ||
		strings.Contains(message, "authority") ||
		strings.Contains(message, "metadata_invalid") ||
		strings.Contains(message, "bucket_mismatch") ||
		strings.Contains(message, "operation_fingerprint_mismatch") ||
		strings.Contains(message, "lease_or_provenance_changed") ||
		strings.Contains(message, "provenance_missing") {
		return code, true
	}
	return "recovery_failed", false
}
