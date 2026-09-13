package earchiva

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/eguilde/egueducation/internal/audit"
	appdb "github.com/eguilde/egueducation/internal/db"
)

type PortfolioRetentionDispositionWorker struct {
	pool                *appdb.SessionPool
	storage             *ArchiveStorage
	logger              *zap.Logger
	workerID            string
	poll                time.Duration
	maxBackoffExponent  int
	afterStorageRelease func() error
	mu                  sync.Mutex
	started             bool
}
type retentionDispositionClaim struct {
	id, tenant, institution string
	attempts                int
}

var errRetentionDispositionPermanentlyBlocked = errors.New("retention disposition permanently blocked")
var errRetentionDispositionPreStorageBlocked = errors.New("retention disposition blocked before storage mutation")

func NewPortfolioRetentionDispositionWorker(pool *appdb.SessionPool, storage *ArchiveStorage, logger *zap.Logger, poll time.Duration, maxBackoffExponent int) *PortfolioRetentionDispositionWorker {
	if poll <= 0 {
		poll = 5 * time.Second
	}
	if maxBackoffExponent < 1 || maxBackoffExponent > 8 {
		maxBackoffExponent = 5
	}
	return &PortfolioRetentionDispositionWorker{pool: pool, storage: storage, logger: logger, workerID: uuid.NewString(), poll: poll, maxBackoffExponent: maxBackoffExponent}
}
func (w *PortfolioRetentionDispositionWorker) Enabled() bool {
	return w != nil && w.pool != nil && w.pool.Raw() != nil && w.storage != nil && w.storage.Enabled()
}
func (w *PortfolioRetentionDispositionWorker) Start(ctx context.Context) {
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
func (w *PortfolioRetentionDispositionWorker) run(ctx context.Context) {
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
func (w *PortfolioRetentionDispositionWorker) drain(ctx context.Context) error {
	for {
		c, err := w.claim(ctx)
		if err != nil || c == nil {
			return err
		}
		tenantCtx, release, err := appdb.AcquireRequestConn(ctx, w.pool.Raw(), appdb.SessionConfig{TenantID: c.tenant, InstitutionID: c.institution, ActorSubject: "portfolio-retention-disposition-worker"})
		if err == nil {
			err = w.process(tenantCtx, c)
			release()
		}
		if err != nil {
			if failErr := w.fail(ctx, c, err); failErr != nil {
				return failErr
			}
			// Retryable work is delayed before being requeued, so this drain can
			// continue with newer releases. It is never dead-lettered: the external
			// hold may already be OFF while the database receipt still needs commit.
			continue
		}
	}
}
func (w *PortfolioRetentionDispositionWorker) claim(ctx context.Context) (*retentionDispositionClaim, error) {
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cctx, release, err := appdb.AcquireRequestConn(cctx, w.pool.Raw(), appdb.SessionConfig{ActorSubject: "portfolio-retention-disposition-worker", IsSuperAdmin: true})
	if err != nil {
		return nil, err
	}
	defer release()
	tx, err := w.pool.Begin(cctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(cctx) //nolint:errcheck
	if _, err = tx.Exec(cctx, `update portfolio_retention_disposition_operations set status='queued',lease_owner='',lease_expires_at=null,available_at=now(),last_error_code='lease_expired' where status='leased' and lease_expires_at<now()`); err != nil {
		return nil, err
	}
	var c retentionDispositionClaim
	err = tx.QueryRow(cctx, `with next as (select o.id from portfolio_retention_disposition_operations o join app_tenants t on t.code=o.tenant_code and t.institution_id=o.institution_id and t.active where o.status='queued' and o.available_at<=now() order by o.available_at,o.created_at for update skip locked limit 1) update portfolio_retention_disposition_operations o set status='leased',attempts=attempts+1,lease_owner=$1,lease_expires_at=now()+interval '5 minutes' from next where o.id=next.id returning o.id::text,o.tenant_code,o.institution_id,o.attempts`, w.workerID).Scan(&c.id, &c.tenant, &c.institution, &c.attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, tx.Commit(cctx)
	}
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec(cctx, `insert into portfolio_retention_disposition_attempts(operation_id,attempt_no,worker_id,outcome) values($1::uuid,$2,$3,'claimed')`, c.id, c.attempts, w.workerID); err != nil {
		return nil, err
	}
	return &c, tx.Commit(cctx)
}
func (w *PortfolioRetentionDispositionWorker) process(ctx context.Context, c *retentionDispositionClaim) error {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	var requestID, transitionID, bucket, key, version, etag, sha256Hex, mimeType string
	var sizeBytes int64
	var required time.Time
	err = tx.QueryRow(ctx, `select r.id::text,t.id::text,r.source_bucket,r.source_object_key,r.source_object_version_id,r.source_object_etag,r.source_sha256,r.source_size_bytes,r.source_mime_type,r.required_retention_until
		from portfolio_retention_disposition_operations o
		join portfolio_retention_disposition_requests r on r.id=o.request_id
		join portfolio_retention_disposition_decisions d on d.request_id=r.id and d.decision='approved'
		join education_portfolio_storage_transitions t on t.id=r.transition_id
		join archive_document_versions v on v.id=r.archive_version_id and v.institution_id=r.institution_id
		where o.id=$1::uuid and o.status='leased' and o.lease_owner=$2 and r.status='approved'
		  and t.status='blocked' and t.last_error='portfolio_retention_expired_review_required'
		  and t.tenant_code=r.tenant_code and t.institution_id=r.institution_id and t.portfolio_id=r.portfolio_id
		  and t.archive_document_id=r.archive_document_id and t.archive_version_id=r.archive_version_id
		  and t.source_bucket=r.source_bucket and t.source_object_key=r.source_object_key
		  and t.source_object_version_id=r.source_object_version_id and t.source_object_etag=r.source_object_etag
		  and lower(t.source_sha256)=lower(r.source_sha256) and t.source_size_bytes=r.source_size_bytes
		  and t.required_retention_until=r.required_retention_until
		  and v.source_bucket=r.source_bucket and v.source_object_key=r.source_object_key
		  and v.source_object_version_id=r.source_object_version_id and v.source_object_etag=r.source_object_etag
		  and lower(v.source_sha256)=lower(r.source_sha256) and v.source_size_bytes=r.source_size_bytes and v.mime_type=r.source_mime_type
		for update of o,r,t,v`, c.id, w.workerID).Scan(&requestID, &transitionID, &bucket, &key, &version, &etag, &sha256Hex, &sizeBytes, &mimeType, &required)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: exact disposition provenance changed", errRetentionDispositionPermanentlyBlocked)
		}
		return fmt.Errorf("read disposition provenance: %w", err)
	}
	if bucket != w.storage.Bucket() {
		return fmt.Errorf("%w: storage bucket mismatch", errRetentionDispositionPermanentlyBlocked)
	}
	// This fence is shared by legal-hold and document-reference mutations (the
	// 0165 triggers take the same transaction advisory lock). Keep it until the
	// verified object transition and durable receipt commit together.
	if _, err = tx.Exec(ctx, `select pg_advisory_xact_lock(hashtextextended((select archive_version_id::text from portfolio_retention_disposition_requests where id=$1::uuid),0))`, requestID); err != nil {
		return fmt.Errorf("acquire disposition release fence: %w", err)
	}
	rows, err := tx.Query(ctx, `select pd.id::text from portfolio_retention_disposition_requests r join education_portfolio_documents pd on pd.archive_version_id=r.archive_version_id and pd.institution_id=r.institution_id join education_portfolios p on p.id=pd.portfolio_id and p.institution_id=pd.institution_id where r.id=$1::uuid order by pd.id for update of pd,p`, requestID)
	if err != nil {
		return fmt.Errorf("lock disposition references: %w", err)
	}
	for rows.Next() {
		var ignored string
		if err = rows.Scan(&ignored); err != nil {
			rows.Close()
			return fmt.Errorf("scan disposition reference lock: %w", err)
		}
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate disposition reference locks: %w", err)
	}
	rows.Close()
	var blocked bool
	err = tx.QueryRow(ctx, `select exists(select 1 from portfolio_retention_disposition_requests r join education_portfolio_documents pd on pd.archive_version_id=r.archive_version_id and pd.institution_id=r.institution_id join education_portfolios p on p.id=pd.portfolio_id and p.institution_id=pd.institution_id where r.id=$1::uuid and (p.legal_hold_active or p.activity_ceased_on is null or (p.retention_until is not null and (p.retention_until+1)::timestamp at time zone 'UTC'>r.required_retention_until)))`, requestID).Scan(&blocked)
	if err != nil {
		return err
	}
	if blocked {
		return fmt.Errorf("%w: %w: current legal or retention blocker", errRetentionDispositionPermanentlyBlocked, errRetentionDispositionPreStorageBlocked)
	}
	state, err := w.storage.ReleaseExpiredPortfolioCustody(ctx, ReleaseExpiredPortfolioCustodyRequest{
		Key: key, VersionID: version, ETag: etag, ExpectedSHA256: sha256Hex,
		ExpectedSize: sizeBytes, ExpectedMIME: mimeType, RequiredRetention: required,
	})
	if err != nil {
		return err
	}
	if state.LegalHoldActive {
		return fmt.Errorf("disposition_hold_verification_failed")
	}
	if w.afterStorageRelease != nil {
		if err = w.afterStorageRelease(); err != nil {
			return fmt.Errorf("after disposition storage release: %w", err)
		}
	}
	if _, err = tx.Exec(ctx, `select set_config('app.portfolio_retention_disposition_operation_id',$1,true)`, c.id); err != nil {
		return fmt.Errorf("bind retention disposition operation: %w", err)
	}
	if _, err = tx.Exec(ctx, `insert into portfolio_retention_disposition_receipts(request_id,transition_id,observed_retention_until,observed_hold_active,closed_by_subject,outcome) values($1::uuid,$2::uuid,$3,false,'portfolio-retention-disposition-worker','released')`, requestID, transitionID, state.RetentionUntil); err != nil {
		return err
	}
	if tag, updateErr := tx.Exec(ctx, `update education_portfolio_storage_transitions set status='completed',completed_at=now(),locked_at=null,locked_by='',last_error='' where id=$1::uuid and status='blocked' and last_error='portfolio_retention_expired_review_required'`, transitionID); updateErr != nil {
		return updateErr
	} else if tag.RowsAffected() != 1 {
		return fmt.Errorf("disposition exact transition projection missing")
	}
	if tag, updateErr := tx.Exec(ctx, `update education_portfolio_retention_expiry_events set consumed_at=now() where transition_id=$1::uuid and consumed_at is null`, transitionID); updateErr != nil {
		return updateErr
	} else if tag.RowsAffected() != 1 {
		return fmt.Errorf("disposition exact expiry event missing")
	}
	if tag, updateErr := tx.Exec(ctx, `update archive_document_versions set custody_hold_active=false,legal_hold_active=false,retention_disposition_hold_active=false,retention_until=greatest(retention_until,$1::timestamptz) where id=(select archive_version_id from portfolio_retention_disposition_requests where id=$2::uuid) and institution_id=public.current_institution_id() and source_bucket=$3 and source_object_key=$4 and source_object_version_id=$5 and source_object_etag=$6 and lower(source_sha256)=lower($7) and source_size_bytes=$8 and mime_type=$9`, state.RetentionUntil, requestID, bucket, key, version, etag, sha256Hex, sizeBytes, mimeType); updateErr != nil {
		return updateErr
	} else if tag.RowsAffected() != 1 {
		return fmt.Errorf("disposition exact archive projection missing")
	}
	if tag, updateErr := tx.Exec(ctx, `update portfolio_retention_disposition_requests set status='closed' where id=$1::uuid and status='approved'`, requestID); updateErr != nil {
		return updateErr
	} else if tag.RowsAffected() != 1 {
		return fmt.Errorf("disposition exact request projection missing")
	}
	if tag, updateErr := tx.Exec(ctx, `update portfolio_retention_disposition_operations set status='released',completed_at=now(),lease_owner='',lease_expires_at=null,last_error_code='' where id=$1::uuid and status='leased' and lease_owner=$2`, c.id, w.workerID); updateErr != nil {
		return updateErr
	} else if tag.RowsAffected() != 1 {
		return fmt.Errorf("disposition exact operation projection missing")
	}
	if _, err = tx.Exec(ctx, `insert into portfolio_retention_disposition_attempts(operation_id,attempt_no,worker_id,outcome) values($1::uuid,$2,$3,'released')`, c.id, c.attempts, w.workerID); err != nil {
		return err
	}
	if err = audit.Log(ctx, tx, audit.Event{ActorSubject: "portfolio-retention-disposition-worker", Action: "education.portfolios.retention_disposition_released", TargetType: "portfolio_retention_disposition_operation", TargetID: c.id, Summary: "Approved exact-version custody hold released after verification.", Details: map[string]any{"request_id": requestID, "transition_id": transitionID}}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (w *PortfolioRetentionDispositionWorker) fail(ctx context.Context, c *retentionDispositionClaim, cause error) error {
	tenantCtx, release, err := appdb.AcquireRequestConn(ctx, w.pool.Raw(), appdb.SessionConfig{TenantID: c.tenant, InstitutionID: c.institution, ActorSubject: "portfolio-retention-disposition-worker"})
	if err != nil {
		return err
	}
	defer release()
	permanent := errors.Is(cause, errRetentionDispositionPermanentlyBlocked) || errors.Is(cause, ErrPortfolioRetentionReleaseBlocked)
	code, status := "disposition_retry_required", "queued"
	availableAt := time.Now().UTC()
	restoreRequired := permanent && c.attempts > 1 && w.storage != nil && w.storage.Enabled()
	if permanent {
		code, status = "disposition_fresh_review_required", "blocked"
	} else {
		exponent := c.attempts - 1
		if exponent < 0 {
			exponent = 0
		}
		if exponent > w.maxBackoffExponent {
			exponent = w.maxBackoffExponent
		}
		availableAt = availableAt.Add(time.Second * time.Duration(1<<exponent))
	}
	tx, err := w.pool.Begin(tenantCtx)
	if err != nil {
		return err
	}
	defer tx.Rollback(tenantCtx) //nolint:errcheck
	if permanent {
		// A previous attempt may have completed the irreversible S3 hold-off
		// before its database receipt committed. If the fresh locked check now
		// finds a blocker, converge storage back to the protective state before
		// closing this disposition. Failure to prove the exact re-hold remains
		// retryable and never records a misleading blocked receipt.
		if restoreRequired {
			if err = w.restoreBlockedDispositionHold(tenantCtx, c); err != nil {
				return fmt.Errorf("restore blocked disposition hold: %w", err)
			}
		}
		if _, err = tx.Exec(tenantCtx, `select set_config('app.portfolio_retention_disposition_operation_id',$1,true)`, c.id); err != nil {
			return err
		}
		var requestID, transitionID string
		if err = tx.QueryRow(tenantCtx, `select r.id::text,r.transition_id::text from portfolio_retention_disposition_operations o join portfolio_retention_disposition_requests r on r.id=o.request_id where o.id=$1::uuid and o.status='leased' and o.lease_owner=$2 and r.status='approved' for update of o,r`, c.id, w.workerID).Scan(&requestID, &transitionID); err != nil {
			return err
		}
		if _, err = tx.Exec(tenantCtx, `insert into portfolio_retention_disposition_receipts(request_id,transition_id,closed_by_subject,outcome) values($1::uuid,$2::uuid,'portfolio-retention-disposition-worker','blocked')`, requestID, transitionID); err != nil {
			return err
		}
		if tag, updateErr := tx.Exec(tenantCtx, `update portfolio_retention_disposition_requests set status='blocked' where id=$1::uuid and status='approved'`, requestID); updateErr != nil {
			return updateErr
		} else if tag.RowsAffected() != 1 {
			return fmt.Errorf("block disposition request projection missing")
		}
	}
	if _, err = tx.Exec(tenantCtx, `update portfolio_retention_disposition_operations set status=$1,lease_owner='',lease_expires_at=null,available_at=$2,last_error_code=$3 where id=$4::uuid and status='leased'`, status, availableAt, code, c.id); err != nil {
		return err
	}
	outcome := "retry"
	if permanent {
		outcome = "blocked"
	}
	if _, err = tx.Exec(tenantCtx, `insert into portfolio_retention_disposition_attempts(operation_id,attempt_no,worker_id,outcome,error_code) values($1::uuid,$2,$3,$4,$5)`, c.id, c.attempts, w.workerID, outcome, code); err != nil {
		return err
	}
	if permanent {
		if err = audit.Log(tenantCtx, tx, audit.Event{ActorSubject: "portfolio-retention-disposition-worker", Action: "education.portfolios.retention_disposition_review_required", TargetType: "portfolio_retention_disposition_operation", TargetID: c.id, Summary: "Storage custody remains protected; a fresh independent disposition review is required.", Details: map[string]any{"error_code": code}}); err != nil {
			return err
		}
	}
	return tx.Commit(tenantCtx)
}

func (w *PortfolioRetentionDispositionWorker) restoreBlockedDispositionHold(ctx context.Context, c *retentionDispositionClaim) error {
	var bucket, key, versionID, etag string
	err := w.pool.QueryRow(ctx, `select r.source_bucket,r.source_object_key,r.source_object_version_id,r.source_object_etag
		from portfolio_retention_disposition_operations o
		join portfolio_retention_disposition_requests r on r.id=o.request_id
		join archive_document_versions v on v.id=r.archive_version_id and v.institution_id=r.institution_id
		where o.id=$1::uuid and o.status='leased' and o.lease_owner=$2
		and v.source_bucket=r.source_bucket and v.source_object_key=r.source_object_key
		and v.source_object_version_id=r.source_object_version_id and v.source_object_etag=r.source_object_etag`, c.id, w.workerID).Scan(&bucket, &key, &versionID, &etag)
	if err != nil {
		return fmt.Errorf("load exact blocked disposition identity: %w", err)
	}
	if bucket != w.storage.Bucket() {
		return fmt.Errorf("blocked disposition storage bucket mismatch")
	}
	state, err := w.storage.ReconcilePortfolioCustodyLifecycle(ctx, PortfolioCustodyLifecycleRequest{
		Key:             key,
		VersionID:       versionID,
		ETag:            etag,
		LegalHoldActive: true,
	})
	if err != nil {
		return err
	}
	if !state.LegalHoldActive {
		return fmt.Errorf("blocked disposition hold restoration verification mismatch")
	}
	return nil
}
