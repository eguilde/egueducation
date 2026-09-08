package earchiva

// This worker is the only bridge from a completed Registratura workflow to
// eArhiva.  The workflow transaction writes an outbox intent; delivery is
// deliberately asynchronous so an archive/OCR outage cannot lie about an
// approved document or cross a tenant boundary.

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

	appdb "github.com/eguilde/egueducation/internal/db"
)

var errNoRegistraturaArchiveOutboxEvent = errors.New("no registratura archive outbox event available")

type archiveObjectStore interface {
	Enabled() bool
	Bucket() string
	CopyObject(context.Context, string, string, string) error
	OriginalObjectKey(string, string, string) string
	ArtifactObjectKey(string, string, int) string
}

type RegistraturaArchiveOutboxWorker struct {
	pool         *appdb.SessionPool
	storage      archiveObjectStore
	logger       *zap.Logger
	pollInterval time.Duration
	maxAttempts  int
	workerID     string
	mu           sync.Mutex
	started      bool
}

type registraturaArchiveOutboxEvent struct {
	ID            string
	TenantCode    string
	InstitutionID string
	DocumentID    string
	Attempts      int
}

type registraturaArchiveSource struct {
	RegistryNumber string
	Subject        string
	DocumentType   string
	Direction      string
	Correspondent  string
	Summary        string
	DocumentDate   *string
	FileName       string
	MimeType       string
	StorageKey     string
	SizeBytes      int64
	ChecksumSHA256 string
}

type registraturaArchivePermanentError struct{ cause error }

func (e registraturaArchivePermanentError) Error() string { return e.cause.Error() }
func (e registraturaArchivePermanentError) Unwrap() error { return e.cause }

func permanentRegistraturaArchiveError(format string, args ...any) error {
	return registraturaArchivePermanentError{cause: fmt.Errorf(format, args...)}
}

func NewRegistraturaArchiveOutboxWorker(pool *appdb.SessionPool, storage *ArchiveStorage, logger *zap.Logger, pollInterval time.Duration, maxAttempts int) *RegistraturaArchiveOutboxWorker {
	if pollInterval <= 0 {
		pollInterval = 5 * time.Second
	}
	if maxAttempts < 1 || maxAttempts > 20 {
		maxAttempts = 5
	}
	return &RegistraturaArchiveOutboxWorker{
		pool: pool, storage: storage, logger: logger, pollInterval: pollInterval,
		maxAttempts: maxAttempts, workerID: uuid.NewString(),
	}
}

func (w *RegistraturaArchiveOutboxWorker) Enabled() bool {
	return w != nil && w.pool != nil && w.pool.Raw() != nil && w.storage != nil && w.storage.Enabled()
}

func (w *RegistraturaArchiveOutboxWorker) Start(ctx context.Context) {
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

func (w *RegistraturaArchiveOutboxWorker) run(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	for {
		if err := w.drainQueue(ctx); err != nil && !errors.Is(err, errNoRegistraturaArchiveOutboxEvent) {
			w.logError("registratura archive outbox worker failed", zap.Error(err))
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *RegistraturaArchiveOutboxWorker) drainQueue(ctx context.Context) error {
	for {
		event, err := w.claimEvent(ctx)
		if err != nil {
			return err
		}
		if event == nil {
			return errNoRegistraturaArchiveOutboxEvent
		}
		tenantCtx, release, err := appdb.AcquireRequestConn(ctx, w.pool.Raw(), appdb.SessionConfig{
			TenantID: event.TenantCode, InstitutionID: event.InstitutionID,
			ActorSubject: "registratura-archive-outbox-worker",
		})
		if err != nil {
			return fmt.Errorf("bind registratura archive outbox tenant context: %w", err)
		}
		err = w.processEvent(tenantCtx, event)
		release()
		if err != nil {
			w.logError("registratura archive outbox delivery failed", zap.String("outbox_id", event.ID), zap.String("document_id", event.DocumentID), zap.Error(err))
		}
	}
}

func (w *RegistraturaArchiveOutboxWorker) claimEvent(ctx context.Context) (*registraturaArchiveOutboxEvent, error) {
	claimCtx, release, err := appdb.AcquireRequestConn(ctx, w.pool.Raw(), appdb.SessionConfig{ActorSubject: "registratura-archive-outbox-worker", IsSuperAdmin: true})
	if err != nil {
		return nil, fmt.Errorf("acquire registratura archive outbox claim session: %w", err)
	}
	defer release()
	tx, err := w.pool.Begin(claimCtx)
	if err != nil {
		return nil, fmt.Errorf("begin registratura archive outbox claim: %w", err)
	}
	defer tx.Rollback(claimCtx) //nolint:errcheck

	if _, err := tx.Exec(claimCtx, `
		update registratura_archive_outbox
		set status = 'pending', locked_at = null, locked_by = '', available_at = now(),
			last_error = case when last_error = '' then 'worker lease expired; event reclaimed' else last_error end
		where status = 'processing' and (locked_at is null or locked_at < now() - interval '30 minutes')
	`); err != nil {
		return nil, fmt.Errorf("reclaim registratura archive outbox leases: %w", err)
	}

	var event registraturaArchiveOutboxEvent
	err = tx.QueryRow(claimCtx, `
		with next_event as (
			select o.id
			from registratura_archive_outbox o
			join app_tenants t on t.code = o.tenant_code and t.institution_id = o.institution_id and t.active
			where o.status = 'pending' and o.available_at <= now()
			order by o.available_at asc, o.created_at asc
			for update of o skip locked
			limit 1
		)
		update registratura_archive_outbox o
		set status = 'processing', attempts = o.attempts + 1, locked_at = now(), locked_by = $1
		from next_event
		where o.id = next_event.id
		returning o.id::text, o.tenant_code, o.institution_id, o.document_id::text, o.attempts
	`, w.workerID).Scan(&event.ID, &event.TenantCode, &event.InstitutionID, &event.DocumentID, &event.Attempts)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("claim registratura archive outbox event: %w", err)
	}
	if err := tx.Commit(claimCtx); err != nil {
		return nil, fmt.Errorf("commit registratura archive outbox claim: %w", err)
	}
	return &event, nil
}

func (w *RegistraturaArchiveOutboxWorker) processEvent(ctx context.Context, event *registraturaArchiveOutboxEvent) error {
	source, err := w.loadSource(ctx, event.DocumentID)
	if err != nil {
		return w.failEvent(ctx, event, err)
	}

	// A deterministic archive identifier makes object-copy retries idempotent
	// too: a crash after the object write cannot leave a new orphan on every
	// retry. UUIDv5 is scoped by tenant and the immutable Registratura ID.
	archiveDocumentID := uuidForRegistraturaArchive(event.InstitutionID, event.DocumentID)
	originalKey := w.storage.OriginalObjectKey(event.InstitutionID, archiveDocumentID, source.FileName)
	// The source was malware-scanned before Registratura accepted it. Both keys
	// live in the same tenant-scoped archive bucket, so use an authenticated
	// server-side copy. This avoids relaying large PDFs through the API process
	// and works for S3-compatible HTTP endpoints where a GetObject response is
	// not seekable for a second SigV4/checksum pass.
	if err := w.storage.CopyObject(ctx, source.StorageKey, originalKey, source.MimeType); err != nil {
		return w.failEvent(ctx, event, fmt.Errorf("copy registratura archive source: %w", err))
	}

	if err := w.persistArchiveRecord(ctx, event, source, archiveDocumentID, originalKey); err != nil {
		return w.failEvent(ctx, event, err)
	}
	if _, err := w.pool.Exec(ctx, `
		update registratura_archive_outbox
		set status = 'delivered', delivered_at = now(), locked_at = null, locked_by = '', last_error = ''
		where id::text = $1 and status = 'processing'
	`, event.ID); err != nil {
		return fmt.Errorf("mark registratura archive outbox delivered: %w", err)
	}
	return nil
}

func (w *RegistraturaArchiveOutboxWorker) loadSource(ctx context.Context, documentID string) (registraturaArchiveSource, error) {
	var source registraturaArchiveSource
	var documentDate *string
	err := w.pool.QueryRow(ctx, `
		select d.registry_number, d.subject, d.document_type, d.direction, d.correspondent, d.summary,
			case when d.entry_at is null then null else to_char(d.entry_at::date, 'YYYY-MM-DD') end,
			a.file_name, a.mime_type, a.storage_key, a.size_bytes, a.checksum_sha256
		from registratura_documents d
		join lateral (
			select file_name, mime_type, storage_key, size_bytes, checksum_sha256
			from registratura_document_attachments
			where document_id = d.id and storage_state = 'ready' and scan_status = 'clean'
				and mime_type = 'application/pdf' and size_bytes > 0 and checksum_sha256 <> ''
			order by uploaded_at asc, id asc
			limit 1
		) a on true
		where d.id::text = $1 and d.status = 'FINALIZAT'
	`, documentID).Scan(&source.RegistryNumber, &source.Subject, &source.DocumentType, &source.Direction, &source.Correspondent, &source.Summary,
		&documentDate, &source.FileName, &source.MimeType, &source.StorageKey, &source.SizeBytes, &source.ChecksumSHA256)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return source, permanentRegistraturaArchiveError("finalized document has no clean PDF attachment")
		}
		return source, fmt.Errorf("load finalized registratura document source: %w", err)
	}
	source.DocumentDate = documentDate
	return source, nil
}

func (w *RegistraturaArchiveOutboxWorker) persistArchiveRecord(ctx context.Context, event *registraturaArchiveOutboxEvent, source registraturaArchiveSource, archiveDocumentID, originalKey string) error {
	idempotencyKey := registraturaArchiveIdempotencyKey(event.DocumentID)
	metadata, err := json.Marshal(map[string]any{
		"registratura_document_id": event.DocumentID,
		"registry_number":          source.RegistryNumber,
		"document_type":            source.DocumentType,
		"direction":                source.Direction,
		"correspondent":            source.Correspondent,
		"summary":                  source.Summary,
		"outbox_event_id":          event.ID,
	})
	if err != nil {
		return fmt.Errorf("marshal registratura archive metadata: %w", err)
	}
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin archive record persist: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var documentID string
	err = tx.QueryRow(ctx, `select id::text from archive_documents where institution_id = $1 and idempotency_key = $2 for update`, event.InstitutionID, idempotencyKey).Scan(&documentID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("find idempotent archive record: %w", err)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		documentID = archiveDocumentID
		artifactKey := w.storage.ArtifactObjectKey(event.InstitutionID, documentID, 1)
		if _, err := tx.Exec(ctx, `
			insert into archive_documents (
				id, institution_id, title, original_file_name, mime_type, source_kind, source_system,
				external_reference, status, original_bucket, original_object_key, artifact_bucket,
				artifact_object_key, document_date, metadata, idempotency_key, created_by, current_version_no, received_at
			) values ($1::uuid, $2, $3, $4, $5, 'registratura_finalized', 'registratura', $6,
				'queued', $7, $8, $7, $9, $10::date, $11::jsonb, $12, 'registratura-archive-outbox-worker', 1, now())
		`, documentID, event.InstitutionID, source.Subject, source.FileName, source.MimeType, event.DocumentID,
			w.storage.Bucket(), originalKey, artifactKey, source.DocumentDate, metadata, idempotencyKey); err != nil {
			return fmt.Errorf("insert archive record: %w", err)
		}
		versionID := uuid.NewString()
		if _, err := tx.Exec(ctx, `
			insert into archive_document_versions (
				id, institution_id, document_id, version_no, mime_type, title, bucket_name, object_key,
				hash_sha256, size_bytes, metadata, ocr_text, status, source_bucket, source_object_key,
				artifact_bucket, artifact_object_key, source_sha256, source_size_bytes, page_count,
				text_status, extracted_text, extracted_metadata, created_by
			) values ($1::uuid, $2, $3::uuid, 1, $4, $5, $6, $7, $8, $9, $10::jsonb, '', 'active',
				$6, $7, $6, $11, $8, $9, 0, 'pending', '', $10::jsonb, 'registratura-archive-outbox-worker')
		`, versionID, event.InstitutionID, documentID, source.MimeType, source.Subject, w.storage.Bucket(), originalKey,
			source.ChecksumSHA256, source.SizeBytes, metadata, artifactKey); err != nil {
			return fmt.Errorf("insert archive record version: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			insert into archive_ingestion_jobs (id, institution_id, document_id, version_id, job_type, status, available_at, created_by)
			values ($1::uuid, $2, $3::uuid, $4::uuid, 'extract_text', 'pending', now(), 'registratura-archive-outbox-worker')
		`, uuid.NewString(), event.InstitutionID, documentID, versionID); err != nil {
			return fmt.Errorf("enqueue archive extraction: %w", err)
		}
	} else {
		var sourceSystem, reference string
		if err := tx.QueryRow(ctx, `select source_system, external_reference from archive_documents where id::text = $1`, documentID).Scan(&sourceSystem, &reference); err != nil {
			return fmt.Errorf("validate idempotent archive record: %w", err)
		}
		if sourceSystem != "registratura" || reference != event.DocumentID {
			return permanentRegistraturaArchiveError("archive idempotency key is owned by a different source record")
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit archive record persist: %w", err)
	}
	return nil
}

func (w *RegistraturaArchiveOutboxWorker) failEvent(ctx context.Context, event *registraturaArchiveOutboxEvent, cause error) error {
	var permanent registraturaArchivePermanentError
	if !errors.As(cause, &permanent) && event.Attempts < w.maxAttempts {
		delay := archiveRetryDelay(event.Attempts)
		if _, err := w.pool.Exec(ctx, `
			update registratura_archive_outbox
			set status = 'pending', available_at = now() + $1::interval, locked_at = null, locked_by = '', last_error = $2
			where id::text = $3 and status = 'processing'
		`, intervalLiteral(delay), truncateWorkerError(cause.Error()), event.ID); err != nil {
			return fmt.Errorf("reschedule registratura archive outbox event: %w", err)
		}
		return cause
	}
	if _, err := w.pool.Exec(ctx, `
		update registratura_archive_outbox
		set status = 'failed', dead_lettered_at = now(), locked_at = null, locked_by = '', last_error = $1
		where id::text = $2 and status = 'processing'
	`, truncateWorkerError(cause.Error()), event.ID); err != nil {
		return fmt.Errorf("dead-letter registratura archive outbox event: %w", err)
	}
	return cause
}

func registraturaArchiveIdempotencyKey(documentID string) string {
	return "registratura-finalized:" + strings.TrimSpace(documentID)
}

func uuidForRegistraturaArchive(institutionID, documentID string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(strings.TrimSpace(institutionID)+":"+strings.TrimSpace(documentID))).String()
}

func (w *RegistraturaArchiveOutboxWorker) logError(msg string, fields ...zap.Field) {
	if w == nil || w.logger == nil {
		return
	}
	w.logger.Error(msg, fields...)
}
