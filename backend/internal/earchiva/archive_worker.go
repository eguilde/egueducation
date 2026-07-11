package earchiva

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	pool         *appdb.SessionPool
	storage      *ArchiveStorage
	extract      *ArchiveTextract
	logger       *zap.Logger
	pollInterval time.Duration
	workerID     string
	mu           sync.Mutex
	started      bool
}

type archiveIngestionJob struct {
	ID            string
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

func NewIngestionWorker(pool *appdb.SessionPool, storage *ArchiveStorage, extract *ArchiveTextract, logger *zap.Logger, pollInterval time.Duration) *IngestionWorker {
	if pollInterval <= 0 {
		pollInterval = 5 * time.Second
	}
	return &IngestionWorker{
		pool:         pool,
		storage:      storage,
		extract:      extract,
		logger:       logger,
		pollInterval: pollInterval,
		workerID:     uuid.NewString(),
	}
}

func (w *IngestionWorker) Enabled() bool {
	return w != nil && w.storage != nil && w.storage.Enabled()
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

		if err := w.processJob(ctx, job); err != nil {
			w.logError("archive ingestion job failed", zap.String("job_id", job.ID), zap.String("document_id", job.DocumentID), zap.Error(err))
		}
	}
}

func (w *IngestionWorker) claimJob(ctx context.Context) (*archiveIngestionJob, error) {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin archive job claim: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var job archiveIngestionJob
	err = tx.QueryRow(ctx, `
		with next_job as (
			select id
			from archive_ingestion_jobs
			where status = 'pending'
				and available_at <= now()
			order by available_at asc, created_at asc
			for update skip locked
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
			j.institution_id,
			j.document_id::text,
			j.version_id::text,
			j.job_type,
			j.attempts,
			to_char(j.created_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
	`, w.workerID).Scan(
		&job.ID,
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

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit archive job claim: %w", err)
	}
	return &job, nil
}

func (w *IngestionWorker) processJob(ctx context.Context, job *archiveIngestionJob) error {
	ctxInfo, err := w.loadIngestionContext(ctx, job.DocumentID, job.VersionID)
	if err != nil {
		return w.failJob(ctx, job, err)
	}

	tmpPath, err := w.downloadToTempFile(ctx, ctxInfo.SourceObjectKey)
	if err != nil {
		return w.failJob(ctx, job, err)
	}
	defer os.Remove(tmpPath)

	text, pageCount, metadata, err := extractArchiveText(ctx, tmpPath, ctxInfo, w.extract)
	if err != nil {
		return w.failJob(ctx, job, err)
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
	_, err := w.pool.Exec(ctx, `
		update archive_ingestion_jobs
		set status = 'failed',
			last_error = $1,
			locked_at = null,
			locked_by = '',
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
	return cause
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
				version_id,
				entity_type,
				entity_value,
				normalized_value,
				confidence,
				chunk_no,
				page_no
			) values ($1, $2::uuid, $3, $4, $5, $6, $7, $8)
		`, info.InstitutionID, info.VersionID, entity.EntityType, entity.EntityValue, entity.NormalizedValue, entity.Confidence, entity.ChunkNo, entity.PageNo); err != nil {
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
			updated_at = now()
		where id::text = $1
	`, job.ID); err != nil {
		return fmt.Errorf("update archive job: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit archive extraction: %w", err)
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

func (w *IngestionWorker) downloadToTempFile(ctx context.Context, objectKey string) (string, error) {
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

	if _, err := io.Copy(tempFile, reader); err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("copy archive object to temp file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("close archive temp file: %w", err)
	}
	return tempFile.Name(), nil
}

func extractArchiveText(ctx context.Context, filePath string, info archiveIngestionContext, textractProcessor *ArchiveTextract) (string, int, map[string]any, error) {
	mimeType := strings.TrimSpace(info.MimeType)
	name := strings.TrimSpace(info.OriginalFileName)
	if textractProcessor != nil && textractProcessor.Enabled() && archiveTextractSupported(mimeType, name) {
		text, pageCount, metadata, err := textractProcessor.AnalyzeDocument(ctx, info.InstitutionID, info.DocumentID, info.VersionNo, filePath, name, mimeType)
		if err == nil && strings.TrimSpace(text) != "" {
			return text, pageCount, metadata, nil
		}
		if err == nil {
			metadata = mergeArchiveMetadata(metadata, map[string]any{
				"textract_empty_text": true,
			})
		} else {
			metadata = mergeArchiveMetadata(metadata, map[string]any{
				"textract_error": err.Error(),
			})
		}
		fallbackText, fallbackPages, fallbackErr := extractLocalArchiveText(filePath, mimeType)
		if fallbackErr == nil && strings.TrimSpace(fallbackText) != "" {
			metadata = mergeArchiveMetadata(metadata, map[string]any{
				"text_extraction_source": "local-fallback",
			})
			return fallbackText, fallbackPages, metadata, nil
		}
		if err == nil {
			err = fallbackErr
		}
		return fallbackText, fallbackPages, metadata, err
	}

	text, pageCount, err := extractLocalArchiveText(filePath, mimeType)
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
