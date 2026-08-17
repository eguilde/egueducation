package earchiva

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/eguilde/egueducation/internal/audit"
	authruntime "github.com/eguilde/egueducation/internal/auth"
	appdb "github.com/eguilde/egueducation/internal/db"
	"github.com/eguilde/egueducation/internal/httpx"
)

// ArchiveAdminService exposes operational information for the current
// institution only. It deliberately does not perform storage or OCR probes:
// those can be slow, costly, and may disclose credentials through errors.
type ArchiveAdminService struct {
	pool        *appdb.SessionPool
	storage     *ArchiveStorage
	ocr         ArchiveOCR
	maxAttempts int
}

func NewArchiveAdminService(pool *appdb.SessionPool, storage *ArchiveStorage, ocr ArchiveOCR, maxAttempts int) *ArchiveAdminService {
	if maxAttempts < 1 || maxAttempts > 20 {
		maxAttempts = 5
	}
	return &ArchiveAdminService{pool: pool, storage: storage, ocr: ocr, maxAttempts: maxAttempts}
}

type ArchiveAdminHealth struct {
	StorageEnabled bool                   `json:"storage_enabled"`
	Storage        ArchiveComponentHealth `json:"storage"`
	OCR            ArchiveOCRHealth       `json:"ocr"`
	Queue          ArchiveJobCounts       `json:"queue"`
}

type ArchiveComponentHealth struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type ArchiveOCRHealth struct {
	Configured bool   `json:"configured"`
	Provider   string `json:"provider"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
}

type ArchiveJobCounts struct {
	Pending   int64 `json:"pending"`
	Running   int64 `json:"running"`
	Succeeded int64 `json:"succeeded"`
	Failed    int64 `json:"failed"`
}

type ArchiveAdminJob struct {
	ID           string `json:"id"`
	DocumentID   string `json:"document_id"`
	JobType      string `json:"job_type"`
	Stage        string `json:"stage"`
	Status       string `json:"status"`
	Attempts     int    `json:"attempts"`
	CreatedAt    string `json:"created_at"`
	AvailableAt  string `json:"available_at"`
	StartedAt    string `json:"started_at,omitempty"`
	FinishedAt   string `json:"finished_at,omitempty"`
	UpdatedAt    string `json:"updated_at"`
	HasError     bool   `json:"has_error"`
	ErrorSummary string `json:"error_summary,omitempty"`
	Error        string `json:"error,omitempty"`
}

type ArchiveAdminJobPage struct {
	Items    []ArchiveAdminJob `json:"items"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
	Total    int64             `json:"total"`
}

type ArchiveAdminStats struct {
	DocumentsByStatus map[string]int64 `json:"documents_by_status"`
	TotalDocuments    int64            `json:"total_documents"`
	TotalBytes        int64            `json:"total_bytes"`
	TotalPages        int64            `json:"total_pages"`
	JobsByStatus      map[string]int64 `json:"jobs_by_status"`
	TotalJobs         int64            `json:"total_jobs"`
	Queued            int64            `json:"queued"`
	Processing        int64            `json:"processing"`
	Completed         int64            `json:"completed"`
	Failed            int64            `json:"failed"`
}

func (s *ArchiveAdminService) Health(w http.ResponseWriter, r *http.Request) {
	institutionID, ok := archiveAdminInstitution(w, r)
	if !ok {
		return
	}
	queue, err := s.jobCounts(r, institutionID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_admin_health_failed"})
		return
	}
	storageEnabled := s.storage != nil && s.storage.Enabled()
	ocrEnabled := s.ocr != nil && s.ocr.Enabled()
	health := ArchiveAdminHealth{StorageEnabled: storageEnabled, Queue: queue}
	if storageEnabled {
		health.Storage = ArchiveComponentHealth{Status: "configured", Message: "live probe not performed"}
	} else {
		health.Storage = ArchiveComponentHealth{Status: "unavailable", Message: "not configured"}
	}
	health.OCR = ArchiveOCRHealth{Configured: ocrEnabled, Provider: archiveOCRProvider(s.ocr)}
	if ocrEnabled {
		health.OCR.Status, health.OCR.Message = "configured", "live probe not performed"
	} else {
		health.OCR.Status, health.OCR.Message = "unavailable", "not configured"
	}
	httpx.JSON(w, http.StatusOK, health)
}

func (s *ArchiveAdminService) ListJobs(w http.ResponseWriter, r *http.Request) {
	institutionID, ok := archiveAdminInstitution(w, r)
	if !ok {
		return
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status != "" && !archiveJobStatus(status) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_archive_job_status"})
		return
	}
	page := archiveAdminPage(r.URL.Query().Get("page"), 1, 100)
	pageSize := archiveAdminPage(r.URL.Query().Get("pageSize"), 25, 100)
	var total int64
	if err := s.pool.QueryRow(r.Context(), `
		select count(*) from archive_ingestion_jobs
		where institution_id = $1 and ($2 = '' or status = $2)
	`, institutionID, status).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_admin_jobs_failed"})
		return
	}
	rows, err := s.pool.Query(r.Context(), `
		select id::text, coalesce(document_id::text, ''), job_type, status, attempts,
			to_char(created_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			to_char(available_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			coalesce(to_char(started_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), ''),
			coalesce(to_char(finished_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), ''),
			to_char(updated_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			last_error
		from archive_ingestion_jobs
		where institution_id = $1 and ($2 = '' or status = $2)
		order by created_at desc, id desc
		limit $3 offset $4
	`, institutionID, status, pageSize, (page-1)*pageSize)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_admin_jobs_failed"})
		return
	}
	defer rows.Close()
	items := make([]ArchiveAdminJob, 0)
	for rows.Next() {
		var item ArchiveAdminJob
		var rawError string
		if err := rows.Scan(&item.ID, &item.DocumentID, &item.JobType, &item.Status, &item.Attempts, &item.CreatedAt, &item.AvailableAt, &item.StartedAt, &item.FinishedAt, &item.UpdatedAt, &rawError); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_admin_jobs_failed"})
			return
		}
		item.HasError = strings.TrimSpace(rawError) != ""
		item.ErrorSummary = archiveAdminErrorSummary(rawError)
		item.Error, item.Stage = item.ErrorSummary, item.JobType
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_admin_jobs_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, ArchiveAdminJobPage{Items: items, Page: page, PageSize: pageSize, Total: total})
}

func (s *ArchiveAdminService) RetryJob(w http.ResponseWriter, r *http.Request) {
	institutionID, ok := archiveAdminInstitution(w, r)
	if !ok {
		return
	}
	jobID := strings.TrimSpace(chi.URLParam(r, "jobID"))
	if _, err := uuid.Parse(jobID); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_archive_job_id"})
		return
	}
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_job_retry_failed"})
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck
	var retry ArchiveAdminJob
	var versionID string
	err = tx.QueryRow(r.Context(), `
		update archive_ingestion_jobs
		set status = 'pending', attempts = 0, available_at = now(), locked_at = null,
			locked_by = '', last_error = '', started_at = null, finished_at = null, updated_at = now()
		where id = $1::uuid and institution_id = $2 and status = 'failed'
		returning id::text, coalesce(document_id::text, ''), coalesce(version_id::text, ''), job_type, status, attempts,
			to_char(created_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			to_char(available_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			to_char(updated_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
	`, jobID, institutionID).Scan(&retry.ID, &retry.DocumentID, &versionID, &retry.JobType, &retry.Status, &retry.Attempts, &retry.CreatedAt, &retry.AvailableAt, &retry.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		// Treat both cross-tenant IDs and non-failed jobs as unavailable: callers
		// must not use this endpoint to enumerate another tenant's work.
		httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "archive_failed_job_not_found"})
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_job_retry_failed"})
		return
	}
	if _, err := tx.Exec(r.Context(), `update archive_documents set status = 'queued', updated_at = now() where id = $1::uuid and institution_id = $2`, retry.DocumentID, institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_job_retry_failed"})
		return
	}
	if _, err := tx.Exec(r.Context(), `update archive_document_versions set text_status = 'pending', updated_at = now() where id = $1::uuid and institution_id = $2`, versionID, institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_job_retry_failed"})
		return
	}
	if err := audit.Log(r.Context(), tx, audit.Event{ActorSubject: authruntime.CurrentSubjectFromRequest(r), Action: "earchiva.job.retry", TargetType: "archive_ingestion_job", TargetID: retry.ID, Summary: "Failed archive ingestion job requeued."}); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_job_retry_failed"})
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_job_retry_failed"})
		return
	}
	retry.Stage = retry.JobType
	httpx.JSON(w, http.StatusOK, retry)
}

func (s *ArchiveAdminService) Stats(w http.ResponseWriter, r *http.Request) {
	institutionID, ok := archiveAdminInstitution(w, r)
	if !ok {
		return
	}
	stats := ArchiveAdminStats{DocumentsByStatus: map[string]int64{}, JobsByStatus: map[string]int64{}}
	rows, err := s.pool.Query(r.Context(), `
		select d.status, count(*), coalesce(sum(v.source_size_bytes), 0), coalesce(sum(v.page_count), 0)
		from archive_documents d
		left join archive_document_versions v on v.document_id = d.id and v.version_no = d.current_version_no
		where d.institution_id = $1
		group by d.status
	`, institutionID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_admin_stats_failed"})
		return
	}
	for rows.Next() {
		var status string
		var count, bytes, pages int64
		if err := rows.Scan(&status, &count, &bytes, &pages); err != nil {
			rows.Close()
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_admin_stats_failed"})
			return
		}
		stats.DocumentsByStatus[status] = count
		stats.TotalDocuments += count
		stats.TotalBytes += bytes
		stats.TotalPages += pages
		switch status {
		case "queued":
			stats.Queued += count
		case "processing":
			stats.Processing += count
		case "ready", "archived":
			stats.Completed += count
		case "failed":
			stats.Failed += count
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_admin_stats_failed"})
		return
	}
	rows.Close()
	rows, err = s.pool.Query(r.Context(), `select status, count(*) from archive_ingestion_jobs where institution_id = $1 group by status`, institutionID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_admin_stats_failed"})
		return
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_admin_stats_failed"})
			return
		}
		stats.JobsByStatus[status] = count
		stats.TotalJobs += count
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_admin_stats_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, stats)
}

func (s *ArchiveAdminService) jobCounts(r *http.Request, institutionID string) (ArchiveJobCounts, error) {
	var c ArchiveJobCounts
	err := s.pool.QueryRow(r.Context(), `
		select count(*) filter (where status = 'pending'), count(*) filter (where status = 'running'),
			count(*) filter (where status = 'succeeded'), count(*) filter (where status = 'failed')
		from archive_ingestion_jobs where institution_id = $1
	`, institutionID).Scan(&c.Pending, &c.Running, &c.Succeeded, &c.Failed)
	return c, err
}

func archiveAdminInstitution(w http.ResponseWriter, r *http.Request) (string, bool) {
	institutionID := strings.TrimSpace(authruntime.CurrentInstitutionIDFromRequest(r))
	if institutionID == "" {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "missing_institution_context"})
		return "", false
	}
	return institutionID, true
}

func archiveOCRProvider(ocr ArchiveOCR) string {
	if ocr == nil || !ocr.Enabled() {
		return "disabled"
	}
	switch ocr.(type) {
	case *AzureDocumentIntelligenceOCR:
		return "azure-document-intelligence"
	case *ArchiveTextract:
		return "aws-textract"
	default:
		return "configured"
	}
}

func archiveJobStatus(status string) bool {
	return status == "pending" || status == "running" || status == "succeeded" || status == "failed"
}

func archiveAdminPage(raw string, fallback, maximum int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 1 {
		return fallback
	}
	if value > maximum {
		return maximum
	}
	return value
}

// archiveAdminErrorSummary intentionally returns a coarse category, never the
// persisted error. OCR errors commonly include file names or extracted data.
func archiveAdminErrorSummary(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return ""
	}
	switch {
	case strings.Contains(value, "azure"), strings.Contains(value, "textract"), strings.Contains(value, "ocr"):
		return "ocr_processing_failed"
	case strings.Contains(value, "storage"), strings.Contains(value, "object"), strings.Contains(value, "s3"), strings.Contains(value, "minio"):
		return "storage_operation_failed"
	case strings.Contains(value, "virus"), strings.Contains(value, "clam"):
		return "malware_scan_failed"
	default:
		return "archive_processing_failed"
	}
}
