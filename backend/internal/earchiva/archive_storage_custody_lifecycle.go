package earchiva

import (
	"context"
	"crypto/md5" //nolint:gosec // S3 Object Lock mandates Content-MD5 for these protocol bodies.
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	aws "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	smithytime "github.com/aws/smithy-go/time"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"

	"github.com/eguilde/egueducation/internal/audit"
	appdb "github.com/eguilde/egueducation/internal/db"
)

// ErrPortfolioRetentionReleaseBlocked marks evidence or policy failures that
// must be resolved by an administrator before an irreversible storage change.
// Transport and database failures deliberately do not wrap this sentinel and
// remain retryable until the durable receipt is committed.
var ErrPortfolioRetentionReleaseBlocked = errors.New("portfolio retention release blocked")

// PortfolioCustodyLifecycleRequest is server-owned exact-version state. The
// retention timestamp is the exclusive UTC instant after the inclusive
// portfolio retention-through date.
type PortfolioCustodyLifecycleRequest struct {
	Key             string
	VersionID       string
	ETag            string
	RetentionUntil  time.Time
	LegalHoldActive bool
}

type PortfolioCustodyLifecycleState struct {
	RetentionUntil  time.Time
	LegalHoldActive bool
}

// ReleaseExpiredPortfolioCustodyRequest is available only to the durable
// independently-approved disposition worker. It never changes retention.
type ReleaseExpiredPortfolioCustodyRequest struct {
	Key, VersionID, ETag string
	ExpectedSHA256       string
	ExpectedSize         int64
	ExpectedMIME         string
	RequiredRetention    time.Time
}

func (s *ArchiveStorage) ReleaseExpiredPortfolioCustody(ctx context.Context, request ReleaseExpiredPortfolioCustodyRequest) (PortfolioCustodyLifecycleState, error) {
	if !s.Enabled() || !s.requireObjectLock || request.RequiredRetention.UTC().After(time.Now().UTC()) {
		return PortfolioCustodyLifecycleState{}, fmt.Errorf("%w: release is not eligible", ErrPortfolioRetentionReleaseBlocked)
	}
	request.Key, request.VersionID = strings.TrimSpace(request.Key), strings.TrimSpace(request.VersionID)
	request.ETag = strings.Trim(strings.TrimSpace(request.ETag), `"`)
	request.ExpectedSHA256, request.ExpectedMIME = strings.ToLower(strings.TrimSpace(request.ExpectedSHA256)), strings.TrimSpace(request.ExpectedMIME)
	if request.Key == "" || request.VersionID == "" || request.VersionID == "null" || request.ETag == "" || request.ExpectedSize <= 0 || request.ExpectedSize > archiveUploadMaxBytes || request.ExpectedMIME == "" {
		return PortfolioCustodyLifecycleState{}, fmt.Errorf("%w: release requires exact identity", ErrPortfolioRetentionReleaseBlocked)
	}
	if decoded, err := hex.DecodeString(request.ExpectedSHA256); err != nil || len(decoded) != sha256.Size {
		return PortfolioCustodyLifecycleState{}, fmt.Errorf("%w: release requires exact SHA256", ErrPortfolioRetentionReleaseBlocked)
	}
	head, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(request.Key), VersionId: aws.String(request.VersionID)})
	if err != nil {
		return PortfolioCustodyLifecycleState{}, fmt.Errorf("read expired portfolio custody identity: %w", err)
	}
	if aws.ToString(head.VersionId) != request.VersionID || strings.Trim(aws.ToString(head.ETag), `"`) != request.ETag || aws.ToInt64(head.ContentLength) != request.ExpectedSize || strings.TrimSpace(aws.ToString(head.ContentType)) != request.ExpectedMIME {
		return PortfolioCustodyLifecycleState{}, fmt.Errorf("%w: exact storage identity mismatch", ErrPortfolioRetentionReleaseBlocked)
	}
	object, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(request.Key), VersionId: aws.String(request.VersionID), IfMatch: aws.String(`"` + request.ETag + `"`)})
	if err != nil {
		return PortfolioCustodyLifecycleState{}, fmt.Errorf("read exact expired portfolio custody version: %w", err)
	}
	digest := sha256.New()
	readSize, readErr := io.Copy(digest, io.LimitReader(object.Body, request.ExpectedSize+1))
	closeErr := object.Body.Close()
	if readErr != nil || closeErr != nil || readSize != request.ExpectedSize || !strings.EqualFold(hex.EncodeToString(digest.Sum(nil)), request.ExpectedSHA256) {
		return PortfolioCustodyLifecycleState{}, fmt.Errorf("%w: exact storage bytes mismatch", ErrPortfolioRetentionReleaseBlocked)
	}
	retention, err := s.client.GetObjectRetention(ctx, &s3.GetObjectRetentionInput{Bucket: aws.String(s.bucket), Key: aws.String(request.Key), VersionId: aws.String(request.VersionID)})
	if err != nil || retention.Retention == nil || retention.Retention.Mode != s3types.ObjectLockRetentionModeCompliance {
		if err != nil {
			return PortfolioCustodyLifecycleState{}, fmt.Errorf("read expired portfolio custody retention: %w", err)
		}
		return PortfolioCustodyLifecycleState{}, fmt.Errorf("%w: COMPLIANCE retention evidence missing", ErrPortfolioRetentionReleaseBlocked)
	}
	state := PortfolioCustodyLifecycleState{RetentionUntil: aws.ToTime(retention.Retention.RetainUntilDate).UTC()}
	if state.RetentionUntil.Before(request.RequiredRetention.UTC()) || state.RetentionUntil.After(time.Now().UTC()) {
		return state, fmt.Errorf("%w: retention is not releasable", ErrPortfolioRetentionReleaseBlocked)
	}
	hold, err := s.client.GetObjectLegalHold(ctx, &s3.GetObjectLegalHoldInput{Bucket: aws.String(s.bucket), Key: aws.String(request.Key), VersionId: aws.String(request.VersionID)})
	if err != nil {
		return state, fmt.Errorf("read expired portfolio custody hold: %w", err)
	}
	state.LegalHoldActive = hold.LegalHold != nil && hold.LegalHold.Status == s3types.ObjectLockLegalHoldStatusOn
	if !state.LegalHoldActive {
		return state, nil
	}
	off := &s3types.ObjectLockLegalHold{Status: s3types.ObjectLockLegalHoldStatusOff}
	if _, err = s.client.PutObjectLegalHold(ctx, &s3.PutObjectLegalHoldInput{Bucket: aws.String(s.bucket), Key: aws.String(request.Key), VersionId: aws.String(request.VersionID), LegalHold: off, ContentMD5: aws.String(objectLockLegalHoldContentMD5(off))}); err != nil {
		return state, fmt.Errorf("release expired portfolio custody hold: %w", err)
	}
	hold, err = s.client.GetObjectLegalHold(ctx, &s3.GetObjectLegalHoldInput{Bucket: aws.String(s.bucket), Key: aws.String(request.Key), VersionId: aws.String(request.VersionID)})
	if err != nil {
		return state, fmt.Errorf("verify expired portfolio custody hold release: %w", err)
	}
	state.LegalHoldActive = hold.LegalHold != nil && hold.LegalHold.Status == s3types.ObjectLockLegalHoldStatusOn
	if state.LegalHoldActive {
		return state, fmt.Errorf("expired portfolio custody hold release verification mismatch")
	}
	return state, nil
}

type portfolioCustodyLifecycleStorage interface {
	Enabled() bool
	Bucket() string
	ReconcilePortfolioCustodyLifecycle(context.Context, PortfolioCustodyLifecycleRequest) (PortfolioCustodyLifecycleState, error)
}

// ReconcilePortfolioCustodyLifecycle extends COMPLIANCE retention and changes
// the hold on one exact VersionId. It never uses the mutable current key.
func (s *ArchiveStorage) ReconcilePortfolioCustodyLifecycle(ctx context.Context, request PortfolioCustodyLifecycleRequest) (PortfolioCustodyLifecycleState, error) {
	if !s.Enabled() || !s.requireObjectLock {
		return PortfolioCustodyLifecycleState{}, fmt.Errorf("portfolio custody lifecycle requires Object Lock storage")
	}
	request.Key = strings.TrimSpace(request.Key)
	request.VersionID = strings.TrimSpace(request.VersionID)
	request.ETag = strings.Trim(strings.TrimSpace(request.ETag), `"`)
	if request.Key == "" || request.VersionID == "" || request.VersionID == "null" || request.ETag == "" {
		return PortfolioCustodyLifecycleState{}, fmt.Errorf("portfolio custody lifecycle requires exact key, VersionId, and ETag")
	}
	// A COMPLIANCE retention deadline is an exclusive instant. Once it has
	// passed, releasing the independent legal hold is a disposition decision,
	// not a reconciliation operation. This guard deliberately precedes every
	// mutating S3 call (and is repeated by the worker before it reaches here).
	if !request.LegalHoldActive && !request.RetentionUntil.IsZero() && !request.RetentionUntil.UTC().After(time.Now().UTC()) {
		return PortfolioCustodyLifecycleState{}, fmt.Errorf("portfolio retention expired review required")
	}
	head, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(request.Key), VersionId: aws.String(request.VersionID)})
	if err != nil {
		return PortfolioCustodyLifecycleState{}, fmt.Errorf("head exact portfolio custody version: %w", err)
	}
	if strings.TrimSpace(aws.ToString(head.VersionId)) != request.VersionID || strings.Trim(aws.ToString(head.ETag), `"`) != request.ETag {
		return PortfolioCustodyLifecycleState{}, fmt.Errorf("portfolio custody exact-version identity mismatch")
	}

	state := PortfolioCustodyLifecycleState{}
	if !request.RetentionUntil.IsZero() {
		required := request.RetentionUntil.UTC()
		retention, getErr := s.client.GetObjectRetention(ctx, &s3.GetObjectRetentionInput{Bucket: aws.String(s.bucket), Key: aws.String(request.Key), VersionId: aws.String(request.VersionID)})
		if getErr == nil && retention.Retention != nil {
			state.RetentionUntil = aws.ToTime(retention.Retention.RetainUntilDate).UTC()
		}
		if getErr != nil || retention == nil || retention.Retention == nil || retention.Retention.Mode != s3types.ObjectLockRetentionModeCompliance || state.RetentionUntil.Before(required) {
			retentionValue := &s3types.ObjectLockRetention{Mode: s3types.ObjectLockRetentionModeCompliance, RetainUntilDate: aws.Time(required)}
			_, err = s.client.PutObjectRetention(ctx, &s3.PutObjectRetentionInput{
				Bucket: aws.String(s.bucket), Key: aws.String(request.Key), VersionId: aws.String(request.VersionID),
				Retention: retentionValue, ContentMD5: aws.String(objectLockRetentionContentMD5(retentionValue)),
			})
			if err != nil {
				return state, fmt.Errorf("extend exact portfolio custody retention: %w", err)
			}
		}
		verified, err := s.client.GetObjectRetention(ctx, &s3.GetObjectRetentionInput{Bucket: aws.String(s.bucket), Key: aws.String(request.Key), VersionId: aws.String(request.VersionID)})
		if err != nil || verified.Retention == nil {
			return state, fmt.Errorf("verify exact portfolio custody retention: %w", err)
		}
		state.RetentionUntil = aws.ToTime(verified.Retention.RetainUntilDate).UTC()
		if verified.Retention.Mode != s3types.ObjectLockRetentionModeCompliance || state.RetentionUntil.Before(required) {
			return state, fmt.Errorf("portfolio custody retention verification mismatch")
		}
	}
	if !request.LegalHoldActive && (request.RetentionUntil.IsZero() || state.RetentionUntil.Before(request.RetentionUntil.UTC())) {
		return state, fmt.Errorf("portfolio custody hold cannot be released before retention verification")
	}

	hold, err := s.client.GetObjectLegalHold(ctx, &s3.GetObjectLegalHoldInput{Bucket: aws.String(s.bucket), Key: aws.String(request.Key), VersionId: aws.String(request.VersionID)})
	if err != nil {
		return state, fmt.Errorf("read exact portfolio custody hold: %w", err)
	}
	state.LegalHoldActive = hold.LegalHold != nil && hold.LegalHold.Status == s3types.ObjectLockLegalHoldStatusOn
	if state.LegalHoldActive != request.LegalHoldActive {
		status := s3types.ObjectLockLegalHoldStatusOff
		if request.LegalHoldActive {
			status = s3types.ObjectLockLegalHoldStatusOn
		}
		legalHoldValue := &s3types.ObjectLockLegalHold{Status: status}
		_, err = s.client.PutObjectLegalHold(ctx, &s3.PutObjectLegalHoldInput{Bucket: aws.String(s.bucket), Key: aws.String(request.Key), VersionId: aws.String(request.VersionID), LegalHold: legalHoldValue, ContentMD5: aws.String(objectLockLegalHoldContentMD5(legalHoldValue))})
		if err != nil {
			return state, fmt.Errorf("reconcile exact portfolio custody hold: %w", err)
		}
		hold, err = s.client.GetObjectLegalHold(ctx, &s3.GetObjectLegalHoldInput{Bucket: aws.String(s.bucket), Key: aws.String(request.Key), VersionId: aws.String(request.VersionID)})
		if err != nil {
			return state, fmt.Errorf("verify exact portfolio custody hold: %w", err)
		}
		state.LegalHoldActive = hold.LegalHold != nil && hold.LegalHold.Status == s3types.ObjectLockLegalHoldStatusOn
	}
	if state.LegalHoldActive != request.LegalHoldActive {
		return state, fmt.Errorf("portfolio custody hold verification mismatch")
	}
	return state, nil
}

func objectLockRetentionContentMD5(retention *s3types.ObjectLockRetention) string {
	body := `<Retention xmlns="http://s3.amazonaws.com/doc/2006-03-01/">`
	if retention != nil && retention.Mode != "" {
		body += `<Mode>` + string(retention.Mode) + `</Mode>`
	}
	if retention != nil && retention.RetainUntilDate != nil {
		body += `<RetainUntilDate>` + smithytime.FormatDateTime(*retention.RetainUntilDate) + `</RetainUntilDate>`
	}
	body += `</Retention>`
	digest := md5.Sum([]byte(body)) //nolint:gosec // Protocol checksum, not a security hash.
	return base64.StdEncoding.EncodeToString(digest[:])
}

func objectLockLegalHoldContentMD5(hold *s3types.ObjectLockLegalHold) string {
	body := `<LegalHold xmlns="http://s3.amazonaws.com/doc/2006-03-01/">`
	if hold != nil && hold.Status != "" {
		body += `<Status>` + string(hold.Status) + `</Status>`
	}
	body += `</LegalHold>`
	digest := md5.Sum([]byte(body)) //nolint:gosec // Protocol checksum, not a security hash.
	return base64.StdEncoding.EncodeToString(digest[:])
}

var errNoPortfolioStorageTransition = errors.New("no portfolio storage transition available")

type portfolioStoragePermanentError struct {
	code string
	err  error
}

func (e portfolioStoragePermanentError) Error() string { return e.err.Error() }
func (e portfolioStoragePermanentError) Unwrap() error { return e.err }

type portfolioStorageTransition struct {
	ID            string
	OperationID   string
	TenantCode    string
	InstitutionID string
	PortfolioID   string
	VersionID     string
	Attempts      int
}

type PortfolioStorageLifecycleWorker struct {
	pool         *appdb.SessionPool
	storage      portfolioCustodyLifecycleStorage
	logger       *zap.Logger
	pollInterval time.Duration
	maxAttempts  int
	workerID     string
	mu           sync.Mutex
	started      bool
}

func NewPortfolioStorageLifecycleWorker(pool *appdb.SessionPool, storage *ArchiveStorage, logger *zap.Logger, pollInterval time.Duration, maxAttempts int) *PortfolioStorageLifecycleWorker {
	return newPortfolioStorageLifecycleWorker(pool, storage, logger, pollInterval, maxAttempts)
}

func newPortfolioStorageLifecycleWorker(pool *appdb.SessionPool, storage portfolioCustodyLifecycleStorage, logger *zap.Logger, pollInterval time.Duration, maxAttempts int) *PortfolioStorageLifecycleWorker {
	if pollInterval <= 0 {
		pollInterval = 5 * time.Second
	}
	if maxAttempts < 1 || maxAttempts > 20 {
		maxAttempts = 5
	}
	return &PortfolioStorageLifecycleWorker{pool: pool, storage: storage, logger: logger, pollInterval: pollInterval, maxAttempts: maxAttempts, workerID: uuid.NewString()}
}

func (w *PortfolioStorageLifecycleWorker) Enabled() bool {
	return w != nil && w.pool != nil && w.pool.Raw() != nil && w.storage != nil && w.storage.Enabled()
}

func (w *PortfolioStorageLifecycleWorker) Start(ctx context.Context) {
	if !w.Enabled() {
		return
	}
	w.mu.Lock()
	if w.started {
		w.mu.Unlock()
		return
	}
	w.started = true
	w.mu.Unlock()
	go w.run(ctx)
}

func (w *PortfolioStorageLifecycleWorker) run(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	for {
		if err := w.drainQueue(ctx); err != nil && !errors.Is(err, errNoPortfolioStorageTransition) {
			w.logError("portfolio storage lifecycle worker failed", zap.Error(err))
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *PortfolioStorageLifecycleWorker) drainQueue(ctx context.Context) error {
	for {
		transition, err := w.claimTransition(ctx)
		if err != nil {
			return err
		}
		if transition == nil {
			return errNoPortfolioStorageTransition
		}
		tenantCtx, release, err := appdb.AcquireRequestConn(ctx, w.pool.Raw(), appdb.SessionConfig{TenantID: transition.TenantCode, InstitutionID: transition.InstitutionID, ActorSubject: "portfolio-storage-lifecycle-worker"})
		if err != nil {
			return fmt.Errorf("bind portfolio storage lifecycle tenant: %w", err)
		}
		err = w.processTransition(tenantCtx, transition)
		if err != nil {
			err = w.failTransition(tenantCtx, transition, err)
		}
		release()
		if err != nil {
			w.logError("portfolio storage transition failed", zap.String("transition_id", transition.ID), zap.String("operation_id", transition.OperationID), zap.Error(err))
		}
	}
}

func (w *PortfolioStorageLifecycleWorker) claimTransition(ctx context.Context) (*portfolioStorageTransition, error) {
	claimCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	claimCtx, release, err := appdb.AcquireRequestConn(claimCtx, w.pool.Raw(), appdb.SessionConfig{ActorSubject: "portfolio-storage-lifecycle-worker", IsSuperAdmin: true})
	if err != nil {
		return nil, err
	}
	defer release()
	tx, err := w.pool.Begin(claimCtx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(claimCtx) //nolint:errcheck
	if _, err := tx.Exec(claimCtx, `update education_portfolio_storage_transitions set status='pending',locked_at=null,locked_by='',available_at=now(),last_error='worker_lease_expired'
		where status='processing' and (locked_at is null or locked_at<now()-interval '5 minutes')`); err != nil {
		return nil, fmt.Errorf("reclaim portfolio storage transition: %w", err)
	}
	// Completed pre-expiry reconciliation deliberately retains the independent
	// custody hold. Once the exclusive deadline arrives, turn that exact
	// verified version into the durable human-review gate; never let time alone
	// make it deletable.
	if _, err := tx.Exec(claimCtx, `insert into education_portfolio_retention_expiry_events(transition_id,tenant_code,institution_id,required_retention_until,promoted_by_subject)
		select id,tenant_code,institution_id,greatest(required_retention_until,coalesce(observed_retention_until,required_retention_until)),'portfolio-storage-lifecycle-worker' from education_portfolio_storage_transitions
		where status='completed' and required_retention_until is not null and greatest(required_retention_until,coalesce(observed_retention_until,required_retention_until))<=now() and observed_hold_active=true on conflict(transition_id) do nothing`); err != nil {
		return nil, fmt.Errorf("record expired portfolio retention review: %w", err)
	}
	if _, err := tx.Exec(claimCtx, `update education_portfolio_storage_transitions set status='pending',available_at=now(),completed_at=null,last_error='portfolio_retention_expiry_review_due'
		where status='completed' and exists(select 1 from education_portfolio_retention_expiry_events e where e.transition_id=education_portfolio_storage_transitions.id and e.consumed_at is null)`); err != nil {
		return nil, fmt.Errorf("schedule expired portfolio retention review: %w", err)
	}
	var transition portfolioStorageTransition
	err = tx.QueryRow(claimCtx, `with next_transition as (
		select s.id from education_portfolio_storage_transitions s join app_tenants t on t.code=s.tenant_code and t.institution_id=s.institution_id and t.active
		where s.status='pending' and s.available_at<=now() order by s.available_at,s.created_at for update of s skip locked limit 1)
		update education_portfolio_storage_transitions s set status='processing',attempts=s.attempts+1,locked_at=now(),locked_by=$1
		from next_transition n where s.id=n.id
		returning s.id::text,s.operation_id::text,s.tenant_code,s.institution_id,s.portfolio_id::text,s.archive_version_id::text,s.attempts`, w.workerID).Scan(
		&transition.ID, &transition.OperationID, &transition.TenantCode, &transition.InstitutionID, &transition.PortfolioID, &transition.VersionID, &transition.Attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("claim portfolio storage transition: %w", err)
	}
	if _, err := tx.Exec(claimCtx, `update education_portfolio_lifecycle_operations set status='processing' where id=$1::uuid and status='pending'`, transition.OperationID); err != nil {
		return nil, err
	}
	if err := tx.Commit(claimCtx); err != nil {
		return nil, err
	}
	return &transition, nil
}

func (w *PortfolioStorageLifecycleWorker) processTransition(ctx context.Context, transition *portfolioStorageTransition) error {
	operationCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	tx, err := w.pool.Begin(operationCtx)
	if err != nil {
		return err
	}
	defer tx.Rollback(operationCtx) //nolint:errcheck

	var bucket, key, versionID, etag string
	err = tx.QueryRow(operationCtx, `select v.source_bucket,v.source_object_key,v.source_object_version_id,v.source_object_etag
		from education_portfolio_storage_transitions s
		join archive_document_versions v on v.institution_id=s.institution_id and v.id=s.archive_version_id and v.document_id=s.archive_document_id
		  and v.source_bucket=s.source_bucket and v.source_object_key=s.source_object_key
		  and v.source_object_version_id=s.source_object_version_id and v.source_object_etag=s.source_object_etag
		  and lower(v.source_sha256)=lower(s.source_sha256) and v.source_size_bytes=s.source_size_bytes
		where s.id=$1::uuid and s.operation_id=$2::uuid and s.status='processing'
		for update of v,s`, transition.ID, transition.OperationID).Scan(&bucket, &key, &versionID, &etag)
	if errors.Is(err, pgx.ErrNoRows) {
		return portfolioStoragePermanentError{code: "lifecycle_provenance_missing", err: err}
	}
	if err != nil {
		return fmt.Errorf("load portfolio storage aggregate: %w", err)
	}
	rows, err := tx.Query(operationCtx, `select p.activity_ceased_on,p.legal_hold_active,p.retention_until
		from education_portfolios p where p.institution_id=$1 and exists(
			select 1 from education_portfolio_documents pd where pd.institution_id=p.institution_id
			and pd.portfolio_id=p.id and pd.archive_version_id=$2::uuid)
		order by p.id for update`, transition.InstitutionID, transition.VersionID)
	if err != nil {
		return fmt.Errorf("lock portfolio storage references: %w", err)
	}
	defer rows.Close()
	var requiredRetention *time.Time
	var custodyRequired, legalHoldRequired, foundReference bool
	for rows.Next() {
		var ceasedOn, retentionThrough *time.Time
		var legalHold bool
		if err := rows.Scan(&ceasedOn, &legalHold, &retentionThrough); err != nil {
			return fmt.Errorf("scan portfolio storage reference: %w", err)
		}
		foundReference = true
		custodyRequired = custodyRequired || ceasedOn == nil
		legalHoldRequired = legalHoldRequired || legalHold
		if retentionThrough != nil {
			exclusive := time.Date(retentionThrough.Year(), retentionThrough.Month(), retentionThrough.Day()+1, 0, 0, 0, 0, time.UTC)
			if requiredRetention == nil || exclusive.After(*requiredRetention) {
				requiredRetention = &exclusive
			}
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate portfolio storage references: %w", err)
	}
	if !foundReference {
		return portfolioStoragePermanentError{code: "lifecycle_reference_missing", err: fmt.Errorf("portfolio storage reference missing")}
	}
	if bucket != w.storage.Bucket() || strings.TrimSpace(versionID) == "" || strings.TrimSpace(etag) == "" {
		return portfolioStoragePermanentError{code: "lifecycle_provenance_mismatch", err: fmt.Errorf("portfolio storage provenance mismatch")}
	}
	retentionDispositionHoldRequired := requiredRetention != nil
	desiredHold := custodyRequired || legalHoldRequired || retentionDispositionHoldRequired
	if !desiredHold && requiredRetention == nil {
		return portfolioStoragePermanentError{code: "lifecycle_retention_required", err: fmt.Errorf("hold release lacks retention")}
	}
	if !custodyRequired && !legalHoldRequired && requiredRetention != nil && !requiredRetention.UTC().After(time.Now().UTC()) {
		// Do not call storage. A passed COMPLIANCE deadline requires a durable
		// human disposition; a normal reconciliation must fail closed.
		return portfolioStoragePermanentError{
			code: "portfolio_retention_expired_review_required",
			err:  fmt.Errorf("portfolio retention expired review required"),
		}
	}
	storageRequest := PortfolioCustodyLifecycleRequest{Key: key, VersionID: versionID, ETag: etag, LegalHoldActive: desiredHold}
	if requiredRetention != nil {
		storageRequest.RetentionUntil = requiredRetention.UTC()
	}
	state, err := w.storage.ReconcilePortfolioCustodyLifecycle(operationCtx, storageRequest)
	if err != nil {
		return fmt.Errorf("reconcile portfolio custody storage: %w", err)
	}
	if state.LegalHoldActive != desiredHold || (requiredRetention != nil && state.RetentionUntil.Before(requiredRetention.UTC())) {
		return fmt.Errorf("portfolio custody storage verification mismatch")
	}
	if _, err := tx.Exec(operationCtx, `update education_portfolio_storage_transitions set storage_verified_at=now(),
		observed_retention_until=$1,observed_hold_active=$2,observed_custody_required=$3,observed_legal_hold_required=$4,observed_retention_disposition_hold_required=$5,
		protective_extension_provenance=case when $6::timestamptz is not null and $7::timestamptz>$6::timestamptz
			then jsonb_build_object('observed_retention_until',$7::timestamptz,'required_retention_until',$6::timestamptz,'source','s3_object_lock_compliance') else '{}'::jsonb end,last_error=''
		where id=$8::uuid and operation_id=$9::uuid and status='processing'`, nullTime(state.RetentionUntil), state.LegalHoldActive, custodyRequired, legalHoldRequired, retentionDispositionHoldRequired, nullTime(requiredRetentionValue(requiredRetention)), nullTime(state.RetentionUntil), transition.ID, transition.OperationID); err != nil {
		return fmt.Errorf("persist portfolio storage verification: %w", err)
	}
	if _, err := tx.Exec(operationCtx, `select set_config('app.portfolio_storage_lifecycle_operation_id',$1,true)`, transition.OperationID); err != nil {
		return err
	}
	if _, err := tx.Exec(operationCtx, `update archive_document_versions set
		retention_until=case when $1::timestamptz is null then retention_until else greatest(retention_until,$1::timestamptz) end,
		custody_hold_active=$2,legal_hold_active=$3,retention_disposition_hold_active=$4 where id=$5::uuid and institution_id=$6`, nullTime(state.RetentionUntil), custodyRequired, legalHoldRequired, retentionDispositionHoldRequired, transition.VersionID, transition.InstitutionID); err != nil {
		return fmt.Errorf("persist verified portfolio archive lifecycle: %w", err)
	}
	if _, err := tx.Exec(operationCtx, `update education_portfolio_storage_transitions set status='completed',completed_at=now(),locked_at=null,locked_by=''
		where id=$1::uuid and status='processing'`, transition.ID); err != nil {
		return err
	}
	if err := updatePortfolioLifecycleOperationStatus(operationCtx, tx, transition.OperationID); err != nil {
		return err
	}
	if err := audit.Log(operationCtx, tx, audit.Event{ActorSubject: "portfolio-storage-lifecycle-worker", Action: "education.portfolios.storage_lifecycle_verified", TargetType: "portfolio_storage_transition", TargetID: transition.ID, Summary: "Exact archive-version retention and hold state verified.", Details: map[string]any{"operation_id": transition.OperationID, "archive_version_id": transition.VersionID, "custody_required": custodyRequired, "legal_hold_required": legalHoldRequired}}); err != nil {
		return err
	}
	return tx.Commit(operationCtx)
}

func (w *PortfolioStorageLifecycleWorker) failTransition(ctx context.Context, transition *portfolioStorageTransition, cause error) error {
	status := "pending"
	code := "storage_reconciliation_failed"
	var permanent portfolioStoragePermanentError
	if errors.As(cause, &permanent) {
		status, code = "blocked", permanent.code
	} else if transition.Attempts >= w.maxAttempts {
		status = "dead_letter"
	}
	availableAt := time.Now().UTC().Add(time.Duration(1<<min(transition.Attempts, 8)) * time.Second)
	_, err := w.pool.Exec(ctx, `update education_portfolio_storage_transitions set status=$1,available_at=$2,locked_at=null,locked_by='',
		last_error=$3,dead_lettered_at=case when $1 in ('blocked','dead_letter') then now() else null end where id=$4::uuid and status='processing'`, status, availableAt, code, transition.ID)
	if err != nil {
		return fmt.Errorf("record portfolio storage transition failure: %w", err)
	}
	if err := updatePortfolioLifecycleOperationStatus(ctx, w.pool, transition.OperationID); err != nil {
		return err
	}
	return cause
}

type lifecycleStatusDB interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func updatePortfolioLifecycleOperationStatus(ctx context.Context, db lifecycleStatusDB, operationID string) error {
	_, err := db.Exec(ctx, `update education_portfolio_lifecycle_operations o set
		status=case
		 when exists(select 1 from education_portfolio_storage_transitions s where s.operation_id=o.id and s.status='dead_letter') then 'dead_letter'
		 when exists(select 1 from education_portfolio_storage_transitions s where s.operation_id=o.id and s.status='blocked') then 'blocked'
		 when not exists(select 1 from education_portfolio_storage_transitions s where s.operation_id=o.id and s.status<>'completed') then 'completed'
		 when exists(select 1 from education_portfolio_storage_transitions s where s.operation_id=o.id and s.status='processing') then 'processing'
		 else 'pending' end,
		completed_at=case when not exists(select 1 from education_portfolio_storage_transitions s where s.operation_id=o.id and s.status<>'completed') then coalesce(completed_at,now()) else null end,
		last_error=coalesce((select s.last_error from education_portfolio_storage_transitions s where s.operation_id=o.id and s.last_error<>'' order by s.created_at limit 1),'')
		where o.id=$1::uuid`, operationID)
	return err
}

func nullTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value.UTC()
}

func requiredRetentionValue(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return value.UTC()
}

func (w *PortfolioStorageLifecycleWorker) logError(message string, fields ...zap.Field) {
	if w != nil && w.logger != nil {
		w.logger.Error(message, fields...)
	}
}
