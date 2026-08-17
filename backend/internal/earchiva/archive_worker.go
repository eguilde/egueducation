package earchiva

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/ledongthuc/pdf"
	"go.uber.org/zap"

	appdb "github.com/eguilde/egueducation/internal/db"
)

var errNoArchiveJob = errors.New("no archive job available")

type IngestionWorker struct {
	pool           *appdb.SessionPool
	storage        *ArchiveStorage
	extract        ArchiveOCR
	classification *ClassificationReviewService
	logger         *zap.Logger
	pollInterval   time.Duration
	maxAttempts    int
	workerID       string
	mu             sync.Mutex
	started        bool
}

type archiveIngestionJob struct {
	ID            string
	TenantID      string
	InstitutionID string
	DocumentID    string
	VersionID     string
	JobType       string
	Attempts      int
	CreatedAt     string
}

type archiveIngestionContext struct {
	DocumentID               string
	InstitutionID            string
	Title                    string
	OriginalFileName         string
	MimeType                 string
	SourceKind               string
	SourceSystem             string
	ExternalReference        string
	TaxonomyNodeID           *string
	TaxonomyCode             *string
	TaxonomyLabel            *string
	Status                   string
	OriginalBucket           string
	OriginalObjectKey        string
	ArtifactBucket           string
	ArtifactObjectKey        string
	DocumentDate             *string
	Metadata                 map[string]any
	CurrentVersionNo         int
	VersionID                string
	VersionNo                int
	SourceBucket             string
	SourceObjectKey          string
	SourceSHA256             string
	SourceSizeBytes          int64
	ArtifactVersionBucket    string
	ArtifactVersionObjectKey string
	ReceivedAt               string
	CreatedAt                string
	UpdatedAt                string
}

type archiveArtifact struct {
	DocumentID  string                  `json:"document_id"`
	VersionID   string                  `json:"version_id"`
	VersionNo   int                     `json:"version_no"`
	ExtractedAt string                  `json:"extracted_at"`
	PageCount   int                     `json:"page_count"`
	TextLength  int                     `json:"text_length"`
	ChunkCount  int                     `json:"chunk_count"`
	EntityCount int                     `json:"entity_count"`
	Relations   []archiveRelationRecord `json:"relations"`
	Chunks      []archiveChunkRecord    `json:"chunks"`
	Entities    []archiveEntityRecord   `json:"entities"`
	Metadata    map[string]any          `json:"metadata,omitempty"`
	Text        string                  `json:"text"`
}

type archiveChunkRecord struct {
	ChunkNo   int    `json:"chunk_no"`
	PageNo    int    `json:"page_no"`
	StartRune int    `json:"start_rune"`
	EndRune   int    `json:"end_rune"`
	Content   string `json:"content"`
}

type archiveEntityRecord struct {
	EntityType      string  `json:"entity_type"`
	EntityValue     string  `json:"entity_value"`
	NormalizedValue string  `json:"normalized_value"`
	Confidence      float64 `json:"confidence"`
	ChunkNo         int     `json:"chunk_no"`
	PageNo          int     `json:"page_no"`
}

type archiveRelationRecord struct {
	RelationType  string         `json:"relation_type"`
	RelationValue string         `json:"relation_value"`
	Confidence    float64        `json:"confidence"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

func NewIngestionWorker(pool *appdb.SessionPool, storage *ArchiveStorage, extract ArchiveOCR, logger *zap.Logger, pollInterval time.Duration) *IngestionWorker {
	return NewIngestionWorkerWithMaxAttempts(pool, storage, extract, logger, pollInterval, 5)
}

func NewIngestionWorkerWithMaxAttempts(pool *appdb.SessionPool, storage *ArchiveStorage, extract ArchiveOCR, logger *zap.Logger, pollInterval time.Duration, maxAttempts int) *IngestionWorker {
	if pollInterval <= 0 {
		pollInterval = 5 * time.Second
	}
	if maxAttempts < 1 || maxAttempts > 20 {
		maxAttempts = 5
	}
	return &IngestionWorker{
		pool:           pool,
		storage:        storage,
		extract:        extract,
		classification: NewClassificationReviewService(pool),
		logger:         logger,
		pollInterval:   pollInterval,
		maxAttempts:    maxAttempts,
		workerID:       uuid.NewString(),
	}
}

func (w *IngestionWorker) Enabled() bool {
	return w != nil && w.storage != nil && w.storage.Enabled() && w.extract != nil && w.extract.Enabled()
}

func (w *IngestionWorker) Start(ctx context.Context) {
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

func (w *IngestionWorker) run(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		if err := w.drainQueue(ctx); err != nil && !errors.Is(err, errNoArchiveJob) {
			w.logError("archive ingestion worker failed", zap.Error(err))
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *IngestionWorker) drainQueue(ctx context.Context) error {
	for {
		job, err := w.claimJob(ctx)
		if err != nil {
			return err
		}
		if job == nil {
			return errNoArchiveJob
		}

		jobCtx, release, err := w.acquireTenantJobContext(ctx, job.TenantID, job.InstitutionID)
		if err != nil {
			return fmt.Errorf("bind archive job tenant context: %w", err)
		}
		err = w.processJob(jobCtx, job)
		release()
		if err != nil {
			w.logError("archive ingestion job failed", zap.String("job_id", job.ID), zap.String("document_id", job.DocumentID), zap.Error(err))
		}
	}
}

func (w *IngestionWorker) claimJob(ctx context.Context) (*archiveIngestionJob, error) {
	// The queue is the sole cross-tenant operation. Use a fresh, short-lived
	// privileged session only to atomically claim one job; it is cleared before
	// any document, object, OCR, or search operation begins.
	claimCtx, release, err := appdb.AcquireRequestConn(ctx, w.pool.Raw(), appdb.SessionConfig{
		ActorSubject: "archive-ingestion-worker", IsSuperAdmin: true,
	})
	if err != nil {
		return nil, fmt.Errorf("acquire archive job claim session: %w", err)
	}
	defer release()
	tx, err := w.pool.Begin(claimCtx)
	if err != nil {
		return nil, fmt.Errorf("begin archive job claim: %w", err)
	}
	defer tx.Rollback(claimCtx) //nolint:errcheck

	var job archiveIngestionJob
	err = tx.QueryRow(claimCtx, `
		with reclaimed_jobs as (
			update archive_ingestion_jobs
			set status = 'pending', locked_at = null, locked_by = '',
				available_at = now(), updated_at = now(),
				last_error = case when last_error = '' then 'worker lease expired; job reclaimed' else last_error end
			where status = 'running'
				and (locked_at is null or locked_at < now() - interval '90 minutes')
			returning id
		), active_tenants as (
			select institution_id, min(code) as tenant_id
			from app_tenants
			where active = true
			group by institution_id
			having count(*) = 1
		), next_job as (
			select j.id, t.tenant_id
			from archive_ingestion_jobs j
			join active_tenants t on t.institution_id = j.institution_id
			where j.status = 'pending'
				and j.available_at <= now()
			order by j.available_at asc, j.created_at asc
			for update of j skip locked
			limit 1
		)
		update archive_ingestion_jobs j
		set status = 'running',
			attempts = j.attempts + 1,
			locked_at = now(),
			locked_by = $1,
			updated_at = now()
		from next_job
		where j.id = next_job.id
		returning
			j.id::text,
			next_job.tenant_id,
			j.institution_id,
			j.document_id::text,
			j.version_id::text,
			j.job_type,
			j.attempts,
			to_char(j.created_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
	`, w.workerID).Scan(
		&job.ID,
		&job.TenantID,
		&job.InstitutionID,
		&job.DocumentID,
		&job.VersionID,
		&job.JobType,
		&job.Attempts,
		&job.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("claim archive job: %w", err)
	}

	if err := tx.Commit(claimCtx); err != nil {
		return nil, fmt.Errorf("commit archive job claim: %w", err)
	}
	return &job, nil
}

func (w *IngestionWorker) acquireTenantJobContext(ctx context.Context, tenantID, institutionID string) (context.Context, func(), error) {
	tenantID = strings.TrimSpace(tenantID)
	institutionID = strings.TrimSpace(institutionID)
	if tenantID == "" || institutionID == "" {
		return nil, nil, fmt.Errorf("archive job is missing active tenant or institution binding")
	}
	return appdb.AcquireRequestConn(ctx, w.pool.Raw(), appdb.SessionConfig{
		TenantID:      tenantID,
		InstitutionID: institutionID,
		ActorSubject:  "archive-ingestion-worker",
		IsSuperAdmin:  false,
	})
}

func (w *IngestionWorker) processJob(ctx context.Context, job *archiveIngestionJob) error {
	ctxInfo, err := w.loadIngestionContext(ctx, job.DocumentID, job.VersionID)
	if err != nil {
		return w.failJob(ctx, job, err)
	}
	if err := w.markJobProcessing(ctx, job, ctxInfo); err != nil {
		return w.failJob(ctx, job, err)
	}

	tmpPath, err := w.downloadToTempFile(ctx, ctxInfo.SourceObjectKey, ctxInfo.SourceSizeBytes)
	if err != nil {
		return w.failJob(ctx, job, err)
	}
	defer os.Remove(tmpPath)
	if err := verifyArchiveSourceFile(tmpPath, ctxInfo.SourceSHA256, ctxInfo.SourceSizeBytes); err != nil {
		return w.failJob(ctx, job, err)
	}

	text, pageCount, metadata, err := extractArchiveText(ctx, tmpPath, ctxInfo, w.extract)
	if err != nil {
		return w.failJob(ctx, job, err)
	}
	if strings.TrimSpace(text) == "" {
		return w.failJob(ctx, job, fmt.Errorf("archive OCR produced no searchable text"))
	}

	chunks := splitArchiveText(text, 1400)
	entities := extractArchiveEntities(chunks)
	relations := extractArchiveRelations(ctxInfo, chunks, entities)
	mergedMetadata := mergeArchiveMetadata(ctxInfo.Metadata, metadata)
	searchEmbedding := buildArchiveEmbedding(archiveDocumentEmbeddingInput(ctxInfo, text))
	artifact := archiveArtifact{
		DocumentID:  ctxInfo.DocumentID,
		VersionID:   ctxInfo.VersionID,
		VersionNo:   ctxInfo.VersionNo,
		ExtractedAt: time.Now().UTC().Format(time.RFC3339),
		PageCount:   pageCount,
		TextLength:  len([]rune(text)),
		ChunkCount:  len(chunks),
		EntityCount: len(entities),
		Relations:   relations,
		Chunks:      chunks,
		Entities:    entities,
		Metadata:    mergedMetadata,
		Text:        text,
	}
	artifactBytes, err := json.Marshal(artifact)
	if err != nil {
		return w.failJob(ctx, job, fmt.Errorf("marshal archive artifact: %w", err))
	}

	if err := w.storage.PutObject(ctx, ctxInfo.ArtifactVersionObjectKey, "application/json", bytes.NewReader(artifactBytes), int64(len(artifactBytes))); err != nil {
		return w.failJob(ctx, job, err)
	}

	if err := w.persistExtraction(ctx, job, ctxInfo, text, pageCount, chunks, entities, relations, mergedMetadata, searchEmbedding, artifactBytes); err != nil {
		return w.failJob(ctx, job, err)
	}

	return nil
}

func (w *IngestionWorker) failJob(ctx context.Context, job *archiveIngestionJob, cause error) error {
	if isRetryableArchiveError(cause) && job.Attempts < w.maxAttempts {
		delay := archiveRetryDelay(job.Attempts)
		_, err := w.pool.Exec(ctx, `
			update archive_ingestion_jobs
			set status = 'pending', last_error = $1, available_at = now() + $2::interval,
				locked_at = null, locked_by = '', started_at = null, finished_at = null, updated_at = now()
			where id::text = $3
		`, truncateWorkerError(cause.Error()), intervalLiteral(delay), job.ID)
		if err != nil {
			return fmt.Errorf("reschedule archive job: %w", err)
		}
		_, _ = w.pool.Exec(ctx, `update archive_documents set status = 'queued', updated_at = now() where id::text = $1`, job.DocumentID)
		_, _ = w.pool.Exec(ctx, `update archive_document_versions set text_status = 'pending', updated_at = now() where id::text = $1`, job.VersionID)
		return cause
	}
	_, err := w.pool.Exec(ctx, `
		update archive_ingestion_jobs
		set status = 'failed',
			last_error = $1,
			locked_at = null,
			locked_by = '',
			finished_at = now(),
			updated_at = now()
		where id::text = $2
	`, truncateWorkerError(cause.Error()), job.ID)
	if err != nil {
		return fmt.Errorf("mark archive job failed: %w", err)
	}
	_, _ = w.pool.Exec(ctx, `
		update archive_documents
		set status = 'failed', updated_at = now()
		where id::text = $1
	`, job.DocumentID)
	_, _ = w.pool.Exec(ctx, `
		update archive_document_versions
		set text_status = 'failed', updated_at = now()
		where id::text = $1
	`, job.VersionID)
	return cause
}

func isRetryableArchiveError(err error) bool {
	if err == nil {
		return false
	}
	var networkErr net.Error
	if errors.As(err, &networkErr) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "http 408") || strings.Contains(message, "http 429") ||
		strings.Contains(message, "http 500") || strings.Contains(message, "http 502") ||
		strings.Contains(message, "http 503") || strings.Contains(message, "http 504")
}

func archiveRetryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	delay := 15 * time.Second * time.Duration(1<<(attempt-1))
	if delay > 15*time.Minute {
		return 15 * time.Minute
	}
	return delay
}

func intervalLiteral(delay time.Duration) string {
	return fmt.Sprintf("%d seconds", int(delay.Round(time.Second).Seconds()))
}

func (w *IngestionWorker) persistExtraction(ctx context.Context, job *archiveIngestionJob, info archiveIngestionContext, text string, pageCount int, chunks []archiveChunkRecord, entities []archiveEntityRecord, relations []archiveRelationRecord, extractedMetadata map[string]any, searchEmbedding []float64, artifactBytes []byte) error {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin archive extraction persist: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `delete from archive_document_chunks where version_id::text = $1`, info.VersionID); err != nil {
		return fmt.Errorf("clear archive chunks: %w", err)
	}
	if _, err := tx.Exec(ctx, `delete from archive_document_entities where version_id::text = $1`, info.VersionID); err != nil {
		return fmt.Errorf("clear archive entities: %w", err)
	}
	if _, err := tx.Exec(ctx, `delete from archive_document_relations where source_document_id::text = $1 and relation_type in ('classified_as', 'references_document_number', 'contains_entities')`, info.DocumentID); err != nil {
		return fmt.Errorf("clear archive relations: %w", err)
	}

	for _, chunk := range chunks {
		if _, err := tx.Exec(ctx, `
			insert into archive_document_chunks (
				institution_id,
				version_id,
				chunk_no,
				page_no,
				content,
				char_start,
				char_end
			) values ($1, $2::uuid, $3, $4, $5, $6, $7)
		`, info.InstitutionID, info.VersionID, chunk.ChunkNo, chunk.PageNo, chunk.Content, chunk.StartRune, chunk.EndRune); err != nil {
			return fmt.Errorf("insert archive chunk: %w", err)
		}
	}

	for _, entity := range entities {
		if _, err := tx.Exec(ctx, `
			insert into archive_document_entities (
				institution_id,
				document_id,
				version_id,
				entity_type,
				entity_value,
				normalized_value,
				confidence,
				chunk_no,
				page_no
			) values ($1, $2::uuid, $3::uuid, $4, $5, $6, $7, $8, $9)
		`, info.InstitutionID, info.DocumentID, info.VersionID, entity.EntityType, entity.EntityValue, entity.NormalizedValue, entity.Confidence, entity.ChunkNo, entity.PageNo); err != nil {
			return fmt.Errorf("insert archive entity: %w", err)
		}
	}

	for _, relation := range relations {
		metadataJSON, err := json.Marshal(relation.Metadata)
		if err != nil {
			return fmt.Errorf("marshal archive relation metadata: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			insert into archive_document_relations (
				institution_id,
				source_document_id,
				relation_type,
				relation_value,
				confidence,
				metadata
			) values ($1, $2::uuid, $3, $4, $5, $6::jsonb)
		`, info.InstitutionID, info.DocumentID, relation.RelationType, relation.RelationValue, relation.Confidence, metadataJSON); err != nil {
			return fmt.Errorf("insert archive relation: %w", err)
		}
	}

	if w.classification == nil {
		return fmt.Errorf("archive classification service is unavailable")
	}
	if _, err := w.classification.UpsertSuggestionTx(ctx, tx, ClassificationInput{InstitutionID: info.InstitutionID, DocumentID: info.DocumentID, VersionID: info.VersionID, OCRText: text}); err != nil {
		return fmt.Errorf("persist archive classification suggestion: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		update archive_document_versions
		set page_count = $1,
			text_status = 'processed',
			extracted_text = $2,
			extracted_metadata = $3::jsonb,
			search_embedding = $4::double precision[],
			artifact_bucket = $5,
			artifact_object_key = $6,
			updated_at = now()
		where id::text = $7
	`, pageCount, text, stringOrJSON(extractedMetadata), searchEmbedding, info.ArtifactVersionBucket, info.ArtifactVersionObjectKey, info.VersionID); err != nil {
		return fmt.Errorf("update archive version: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		update archive_documents
		set status = 'ready',
			current_version_no = $1,
			artifact_bucket = $2,
			artifact_object_key = $3,
			updated_at = now()
		where id::text = $4
	`, info.VersionNo, info.ArtifactVersionBucket, info.ArtifactVersionObjectKey, info.DocumentID); err != nil {
		return fmt.Errorf("update archive document: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		update archive_ingestion_jobs
		set status = 'succeeded',
			last_error = '',
			locked_at = null,
			locked_by = '',
			finished_at = now(),
			updated_at = now()
		where id::text = $1
	`, job.ID); err != nil {
		return fmt.Errorf("update archive job: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		// A transport error can lose the COMMIT acknowledgement after PostgreSQL
		// has durably committed. Reconcile through a fresh pooled query before the
		// caller retries or downgrades a successfully processed document.
		if w.extractionCommitVisible(job, info) {
			return nil
		}
		return fmt.Errorf("commit archive extraction: %w", err)
	}
	return nil
}

func (w *IngestionWorker) extractionCommitVisible(job *archiveIngestionJob, info archiveIngestionContext) bool {
	checkCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	scopedCtx, release, err := appdb.AcquireRequestConn(checkCtx, w.pool.Raw(), appdb.SessionConfig{
		TenantID:      job.TenantID,
		InstitutionID: info.InstitutionID,
		ActorSubject:  "archive-worker-commit-reconcile",
		IsSuperAdmin:  false,
	})
	if err != nil {
		return false
	}
	defer release()
	var jobStatus, documentStatus, textStatus string
	err = w.pool.QueryRow(scopedCtx, `
		select j.status, d.status, v.text_status
		from archive_ingestion_jobs j
		join archive_documents d on d.id = j.document_id and d.institution_id = j.institution_id
		join archive_document_versions v on v.id = j.version_id and v.institution_id = j.institution_id
		where j.id::text = $1 and j.institution_id = $2
	`, job.ID, info.InstitutionID).Scan(&jobStatus, &documentStatus, &textStatus)
	return err == nil && jobStatus == "succeeded" && documentStatus == "ready" && textStatus == "processed"
}

func (w *IngestionWorker) markJobProcessing(ctx context.Context, job *archiveIngestionJob, info archiveIngestionContext) error {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin archive processing state: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, `update archive_documents set status = 'processing', updated_at = now() where id::text = $1 and institution_id = $2`, info.DocumentID, info.InstitutionID); err != nil {
		return fmt.Errorf("mark archive document processing: %w", err)
	}
	if _, err := tx.Exec(ctx, `update archive_document_versions set text_status = 'processing', updated_at = now() where id::text = $1 and institution_id = $2`, info.VersionID, info.InstitutionID); err != nil {
		return fmt.Errorf("mark archive version processing: %w", err)
	}
	if _, err := tx.Exec(ctx, `update archive_ingestion_jobs set started_at = now(), finished_at = null, updated_at = now() where id::text = $1 and institution_id = $2`, job.ID, info.InstitutionID); err != nil {
		return fmt.Errorf("mark archive job processing: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit archive processing state: %w", err)
	}
	return nil
}

func verifyArchiveSourceFile(filePath, expectedSHA256 string, expectedSize int64) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open archive source for fixity verification: %w", err)
	}
	defer file.Close() //nolint:errcheck
	hash := sha256.New()
	actualSize, err := io.Copy(hash, file)
	if err != nil {
		return fmt.Errorf("hash archive source: %w", err)
	}
	if expectedSize > 0 && actualSize != expectedSize {
		return fmt.Errorf("archive source fixity size mismatch")
	}
	actualSHA256 := hex.EncodeToString(hash.Sum(nil))
	if expected := strings.ToLower(strings.TrimSpace(expectedSHA256)); expected == "" || actualSHA256 != expected {
		return fmt.Errorf("archive source fixity checksum mismatch")
	}
	return nil
}

func (w *IngestionWorker) loadIngestionContext(ctx context.Context, documentID, versionID string) (archiveIngestionContext, error) {
	var info archiveIngestionContext
	var taxonomyID sql.NullString
	var taxonomyCode sql.NullString
	var taxonomyLabel sql.NullString
	var documentDate sql.NullString
	var metadataJSON []byte
	if err := w.pool.QueryRow(ctx, `
		select
			d.id::text,
			d.institution_id,
			d.title,
			d.original_file_name,
			d.mime_type,
			d.source_kind,
			d.source_system,
			d.external_reference,
			d.taxonomy_node_id::text,
			t.code,
			t.label,
			d.status,
			d.original_bucket,
			d.original_object_key,
			d.artifact_bucket,
			d.artifact_object_key,
			case when d.document_date is null then null else to_char(d.document_date, 'YYYY-MM-DD') end,
			d.metadata,
			d.current_version_no,
			v.id::text,
			v.version_no,
			v.source_bucket,
			v.source_object_key,
			v.artifact_bucket,
			v.artifact_object_key,
			v.source_sha256,
			v.source_size_bytes,
			to_char(d.received_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			to_char(d.created_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			to_char(d.updated_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		from archive_documents d
		join archive_document_versions v on v.document_id = d.id and v.id::text = $2
		left join archive_taxonomy_nodes t on t.id = d.taxonomy_node_id
		where d.id::text = $1
	`, documentID, versionID).Scan(
		&info.DocumentID,
		&info.InstitutionID,
		&info.Title,
		&info.OriginalFileName,
		&info.MimeType,
		&info.SourceKind,
		&info.SourceSystem,
		&info.ExternalReference,
		&taxonomyID,
		&taxonomyCode,
		&taxonomyLabel,
		&info.Status,
		&info.OriginalBucket,
		&info.OriginalObjectKey,
		&info.ArtifactBucket,
		&info.ArtifactObjectKey,
		&documentDate,
		&metadataJSON,
		&info.CurrentVersionNo,
		&info.VersionID,
		&info.VersionNo,
		&info.SourceBucket,
		&info.SourceObjectKey,
		&info.ArtifactVersionBucket,
		&info.ArtifactVersionObjectKey,
		&info.SourceSHA256,
		&info.SourceSizeBytes,
		&info.ReceivedAt,
		&info.CreatedAt,
		&info.UpdatedAt,
	); err != nil {
		return archiveIngestionContext{}, err
	}
	if taxonomyID.Valid {
		value := strings.TrimSpace(taxonomyID.String)
		info.TaxonomyNodeID = &value
	}
	if taxonomyCode.Valid {
		value := strings.TrimSpace(taxonomyCode.String)
		info.TaxonomyCode = &value
	}
	if taxonomyLabel.Valid {
		value := strings.TrimSpace(taxonomyLabel.String)
		info.TaxonomyLabel = &value
	}
	if documentDate.Valid {
		value := strings.TrimSpace(documentDate.String)
		info.DocumentDate = &value
	}
	if err := unmarshalMetadata(metadataJSON, &info.Metadata); err != nil {
		return archiveIngestionContext{}, err
	}
	return info, nil
}

func (w *IngestionWorker) downloadToTempFile(ctx context.Context, objectKey string, expectedSize int64) (string, error) {
	if expectedSize <= 0 || expectedSize > archiveUploadMaxBytes {
		return "", fmt.Errorf("archive source has an invalid expected size")
	}
	reader, err := w.storage.OpenObject(ctx, objectKey)
	if err != nil {
		return "", err
	}
	defer reader.Close() //nolint:errcheck

	tempFile, err := os.CreateTemp("", "egueducation-archive-*.bin")
	if err != nil {
		return "", fmt.Errorf("create archive temp file: %w", err)
	}
	defer tempFile.Close() //nolint:errcheck

	written, err := io.Copy(tempFile, io.LimitReader(reader, expectedSize+1))
	if err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("copy archive object to temp file: %w", err)
	}
	if written != expectedSize {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("archive source size changed before processing")
	}
	if err := tempFile.Close(); err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("close archive temp file: %w", err)
	}
	return tempFile.Name(), nil
}

func extractArchiveText(ctx context.Context, filePath string, info archiveIngestionContext, textractProcessor ArchiveOCR) (string, int, map[string]any, error) {
	mimeType := strings.TrimSpace(info.MimeType)
	name := strings.TrimSpace(info.OriginalFileName)
	if textractProcessor != nil && textractProcessor.Enabled() && archiveTextractSupported(mimeType, name) {
		text, pageCount, metadata, err := textractProcessor.AnalyzeDocument(ctx, info.InstitutionID, info.DocumentID, info.VersionNo, filePath, name, mimeType)
		if err != nil {
			return "", 0, metadata, err
		}
		if strings.TrimSpace(text) == "" {
			return "", 0, metadata, fmt.Errorf("archive OCR produced no searchable text")
		}
		return text, pageCount, metadata, nil
	}

	text, pageCount, err := extractLocalArchiveText(filePath, mimeType)
	if err == nil && strings.TrimSpace(text) == "" {
		err = fmt.Errorf("archive document has no searchable text and no OCR provider succeeded")
	}
	return text, pageCount, map[string]any{"text_extraction_source": "local"}, err
}

func extractLocalArchiveText(filePath, mimeType string) (string, int, error) {
	switch {
	case strings.Contains(strings.ToLower(mimeType), "pdf") || strings.EqualFold(filepath.Ext(filePath), ".pdf"):
		file, reader, err := pdf.Open(filePath)
		if err != nil {
			return "", 0, fmt.Errorf("open pdf: %w", err)
		}
		defer file.Close() //nolint:errcheck

		plain, err := reader.GetPlainText()
		if err != nil {
			return "", 0, fmt.Errorf("extract pdf text: %w", err)
		}
		text, err := io.ReadAll(plain)
		if err != nil {
			return "", 0, fmt.Errorf("read pdf text: %w", err)
		}
		return string(text), reader.NumPage(), nil
	case strings.Contains(strings.ToLower(mimeType), "text"):
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", 0, fmt.Errorf("read text file: %w", err)
		}
		return string(data), 1, nil
	default:
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", 0, fmt.Errorf("read archive file: %w", err)
		}
		return string(data), 1, nil
	}
}

func archiveTextractSupported(mimeType, fileName string) bool {
	if strings.Contains(strings.ToLower(mimeType), "pdf") || strings.Contains(strings.ToLower(mimeType), "image/") {
		return true
	}
	switch strings.ToLower(filepath.Ext(fileName)) {
	case ".pdf", ".png", ".jpg", ".jpeg", ".tif", ".tiff":
		return true
	default:
		return false
	}
}

func splitArchiveText(text string, chunkSize int) []archiveChunkRecord {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil
	}
	runes := []rune(trimmed)
	if chunkSize <= 0 {
		chunkSize = 1400
	}
	chunks := make([]archiveChunkRecord, 0, (len(runes)/chunkSize)+1)
	for start := 0; start < len(runes); start += chunkSize {
		end := start + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		content := strings.TrimSpace(string(runes[start:end]))
		if content == "" {
			continue
		}
		chunks = append(chunks, archiveChunkRecord{
			ChunkNo:   len(chunks) + 1,
			PageNo:    0,
			StartRune: start,
			EndRune:   end,
			Content:   content,
		})
	}
	return chunks
}

var (
	emailRe      = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)
	phoneRe      = regexp.MustCompile(`(?i)(?:\+?40\s*)?(?:0?7\d{2}|0?2\d{2}|0?3\d{2})[\s.\-]?\d{3}[\s.\-]?\d{3}`)
	dateRe       = regexp.MustCompile(`\b\d{1,2}[./-]\d{1,2}[./-]\d{2,4}\b`)
	documentNoRe = regexp.MustCompile(`\b(?:ARH|ARCH|EAH|REG)-\d{4}-\d{4}\b`)
)

func extractArchiveEntities(chunks []archiveChunkRecord) []archiveEntityRecord {
	entities := make([]archiveEntityRecord, 0)
	for _, chunk := range chunks {
		entities = append(entities, extractPatternEntities(chunk, "email", emailRe, 1.0)...)
		entities = append(entities, extractPatternEntities(chunk, "phone", phoneRe, 1.0)...)
		entities = append(entities, extractPatternEntities(chunk, "date", dateRe, 0.95)...)
		entities = append(entities, extractPatternEntities(chunk, "document_number", documentNoRe, 0.9)...)
	}
	return dedupeArchiveEntities(entities)
}

func extractPatternEntities(chunk archiveChunkRecord, entityType string, re *regexp.Regexp, confidence float64) []archiveEntityRecord {
	matches := re.FindAllStringIndex(chunk.Content, -1)
	if len(matches) == 0 {
		return nil
	}
	entities := make([]archiveEntityRecord, 0, len(matches))
	for _, match := range matches {
		value := strings.TrimSpace(chunk.Content[match[0]:match[1]])
		if value == "" {
			continue
		}
		entities = append(entities, archiveEntityRecord{
			EntityType:      entityType,
			EntityValue:     value,
			NormalizedValue: normalizeEntityValue(value),
			Confidence:      confidence,
			ChunkNo:         chunk.ChunkNo,
			PageNo:          chunk.PageNo,
		})
	}
	return entities
}

func dedupeArchiveEntities(entities []archiveEntityRecord) []archiveEntityRecord {
	if len(entities) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(entities))
	result := make([]archiveEntityRecord, 0, len(entities))
	for _, entity := range entities {
		key := entity.EntityType + "|" + entity.NormalizedValue
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, entity)
	}
	return result
}

func extractArchiveRelations(info archiveIngestionContext, chunks []archiveChunkRecord, entities []archiveEntityRecord) []archiveRelationRecord {
	relations := make([]archiveRelationRecord, 0)
	if info.TaxonomyCode != nil {
		relations = append(relations, archiveRelationRecord{
			RelationType:  "classified_as",
			RelationValue: *info.TaxonomyCode,
			Confidence:    1,
			Metadata: map[string]any{
				"taxonomy_label": info.TaxonomyLabel,
			},
		})
	}

	refRe := regexp.MustCompile(`\b(?:ARH|ARCH|EAH|REG)-\d{4}-\d{4}\b`)
	seen := make(map[string]struct{})
	for _, chunk := range chunks {
		for _, value := range refRe.FindAllString(chunk.Content, -1) {
			key := strings.ToUpper(strings.TrimSpace(value))
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			relations = append(relations, archiveRelationRecord{
				RelationType:  "references_document_number",
				RelationValue: key,
				Confidence:    0.8,
				Metadata: map[string]any{
					"chunk_no": chunk.ChunkNo,
				},
			})
		}
	}

	if len(entities) > 0 {
		relations = append(relations, archiveRelationRecord{
			RelationType:  "contains_entities",
			RelationValue: fmt.Sprintf("%d entities", len(entities)),
			Confidence:    1,
			Metadata: map[string]any{
				"entity_count": len(entities),
			},
		})
	}

	return relations
}

func normalizeEntityValue(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "")
	value = strings.ReplaceAll(value, "-", "")
	value = strings.ReplaceAll(value, ".", "")
	value = strings.ReplaceAll(value, "/", "")
	return value
}

func truncateWorkerError(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "archive ingestion failed"
	}
	if len(value) > 500 {
		return value[:500]
	}
	return value
}

func mergeArchiveMetadata(base map[string]any, extra map[string]any) map[string]any {
	if len(base) == 0 && len(extra) == 0 {
		return map[string]any{}
	}
	merged := make(map[string]any, len(base)+len(extra))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range extra {
		merged[key] = value
	}
	return merged
}

func stringOrJSON(value map[string]any) string {
	if len(value) == 0 {
		return `{}`
	}
	data, err := json.Marshal(value)
	if err != nil {
		return `{}`
	}
	return string(data)
}

func (w *IngestionWorker) logError(msg string, fields ...zap.Field) {
	if w == nil || w.logger == nil {
		return
	}
	w.logger.Error(msg, fields...)
}
