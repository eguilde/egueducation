package earchiva

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/eguilde/egueducation/internal/audit"
	authruntime "github.com/eguilde/egueducation/internal/auth"
	appdb "github.com/eguilde/egueducation/internal/db"
	"github.com/eguilde/egueducation/internal/httpx"
)

const (
	archiveUploadMaxBytes = 100 << 20
	archiveUploadMaxPages = 2000
	archiveUploadOverhead = 1 << 20
)

// Bound disk, scanner, and object-storage pressure from concurrent archive
// imports. The worker remains asynchronous after the upload is accepted.
var archiveUploadSlots = make(chan struct{}, 2)

// Scanner permits the archive boundary to require an antivirus verdict before
// any archive material is persisted to object storage.
type Scanner interface {
	Scan(context.Context, io.Reader) (clean bool, err error)
}

type DocumentService struct {
	pool    *appdb.SessionPool
	storage *ArchiveStorage
	scanner Scanner
}

func (s *DocumentService) SetScanner(scanner Scanner) { s.scanner = scanner }

func NewDocumentService(pool *appdb.SessionPool, storage *ArchiveStorage) *DocumentService {
	return &DocumentService{pool: pool, storage: storage}
}

type ArchiveDocument struct {
	ID                string         `json:"id"`
	InstitutionID     string         `json:"institution_id"`
	Title             string         `json:"title"`
	OriginalFileName  string         `json:"original_file_name"`
	MimeType          string         `json:"mime_type"`
	SourceKind        string         `json:"source_kind"`
	SourceSystem      string         `json:"source_system"`
	ExternalReference string         `json:"external_reference"`
	TaxonomyNodeID    *string        `json:"taxonomy_node_id,omitempty"`
	TaxonomyCode      *string        `json:"taxonomy_code,omitempty"`
	TaxonomyLabel     *string        `json:"taxonomy_label,omitempty"`
	Status            string         `json:"status"`
	OriginalBucket    string         `json:"-"`
	OriginalObjectKey string         `json:"-"`
	ArtifactBucket    string         `json:"-"`
	ArtifactObjectKey string         `json:"-"`
	DocumentDate      *string        `json:"document_date,omitempty"`
	Metadata          map[string]any `json:"metadata,omitempty"`
	CurrentVersionNo  int            `json:"current_version_no"`
	ReceivedAt        string         `json:"received_at"`
	CreatedAt         string         `json:"created_at"`
	UpdatedAt         string         `json:"updated_at"`
}

type ArchiveDocumentVersion struct {
	ID                string `json:"id"`
	DocumentID        string `json:"document_id"`
	VersionNo         int    `json:"version_no"`
	SourceBucket      string `json:"-"`
	SourceObjectKey   string `json:"-"`
	ArtifactBucket    string `json:"-"`
	ArtifactObjectKey string `json:"-"`
	SourceSHA256      string `json:"source_sha256"`
	SourceSizeBytes   int64  `json:"source_size_bytes"`
	PageCount         int    `json:"page_count"`
	TextStatus        string `json:"text_status"`
	CreatedBy         string `json:"-"`
	CreatedAt         string `json:"created_at"`
}

type ArchiveDocumentDetail struct {
	ArchiveDocument
	LatestVersion *ArchiveDocumentVersion `json:"latest_version,omitempty"`
}

type ArchiveDocumentSearchResult struct {
	ArchiveDocument
	Score   float64 `json:"score"`
	Snippet string  `json:"snippet,omitempty"`
}

type ArchiveDocumentVersionSummary struct {
	ArchiveDocumentVersion
	ChunkCount int `json:"chunk_count"`
}

type ArchiveTaxonomyNode struct {
	ID          string  `json:"id"`
	ParentID    *string `json:"parent_id,omitempty"`
	Code        string  `json:"code"`
	Label       string  `json:"label"`
	Description string  `json:"description"`
	Path        string  `json:"path"`
	Active      bool    `json:"active"`
	SortOrder   int     `json:"sort_order"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type archiveUploadPayload struct {
	Title             string
	SourceKind        string
	SourceSystem      string
	ExternalReference string
	TaxonomyCode      string
	TaxonomyLabel     string
	TaxonomyParent    string
	DocumentDate      *string
	Metadata          map[string]any
	MimeType          string
	FileName          string
	FileSize          int64
	ChecksumSHA256    string
	OriginalObjectKey string
	ArtifactObjectKey string
	OriginalBucket    string
	ArtifactBucket    string
	IdempotencyKey    string
}

func (s *DocumentService) SearchDocuments(w http.ResponseWriter, r *http.Request) {
	institutionID := strings.TrimSpace(authruntime.CurrentInstitutionIDFromRequest(r))
	if institutionID == "" {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "missing_institution_context"})
		return
	}
	queryText := strings.TrimSpace(r.URL.Query().Get("q"))
	mode := normalizeArchiveSearchMode(r.URL.Query().Get("mode"))
	if mode == archiveVectorSearchModeVector {
		s.SearchDocumentsVector(w, r)
		return
	}
	if mode == archiveVectorSearchModeHybrid && queryText != "" {
		s.SearchDocumentsHybrid(w, r)
		return
	}

	filters := map[string]string{
		"status":             strings.TrimSpace(r.URL.Query().Get("status")),
		"source_kind":        strings.TrimSpace(r.URL.Query().Get("source_kind")),
		"mime_type":          strings.TrimSpace(r.URL.Query().Get("mime_type")),
		"taxonomy_code":      strings.TrimSpace(r.URL.Query().Get("taxonomy_code")),
		"external_reference": strings.TrimSpace(r.URL.Query().Get("external_reference")),
		"document_date_from": strings.TrimSpace(r.URL.Query().Get("document_date_from")),
		"document_date_to":   strings.TrimSpace(r.URL.Query().Get("document_date_to")),
		"received_at_from":   strings.TrimSpace(r.URL.Query().Get("received_at_from")),
		"received_at_to":     strings.TrimSpace(r.URL.Query().Get("received_at_to")),
		"taxonomy_label":     strings.TrimSpace(r.URL.Query().Get("taxonomy_label")),
	}

	page := parsePositiveInt(r.URL.Query().Get("page"), 1)
	pageSize := parsePositiveInt(r.URL.Query().Get("pageSize"), 25)
	if pageSize > 100 {
		pageSize = 100
	}
	direction := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("direction")))
	if direction != "desc" {
		direction = "asc"
	}
	sortField := strings.TrimSpace(r.URL.Query().Get("sort"))
	if sortField == "" {
		sortField = "updated_at"
	}
	allowedSorts := map[string]string{
		"title":              "d.title",
		"status":             "d.status",
		"received_at":        "d.received_at",
		"updated_at":         "d.updated_at",
		"current_version_no": "d.current_version_no",
		"external_reference": "d.external_reference",
		"document_date":      "d.document_date",
		"source_kind":        "d.source_kind",
		"mime_type":          "d.mime_type",
	}
	sortColumn, ok := allowedSorts[sortField]
	if !ok {
		sortColumn = "d.updated_at"
	}

	where, args := buildArchiveDocumentFilters(institutionID, filters)
	limitArg := pageSize
	offsetArg := (page - 1) * pageSize
	var total int

	if queryText == "" {
		countSQL := "select count(*) from archive_documents d left join archive_taxonomy_nodes t on t.id = d.taxonomy_node_id " + where
		if err := s.pool.QueryRow(r.Context(), countSQL, args...).Scan(&total); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_documents_search_failed"})
			return
		}

		selectArgs := append(append([]any{}, args...), limitArg, offsetArg)
		selectSQL := fmt.Sprintf(`
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
				to_char(d.received_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
				to_char(d.created_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
				to_char(d.updated_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
			from archive_documents d
			left join archive_taxonomy_nodes t on t.id = d.taxonomy_node_id
			%s
			order by %s %s, d.updated_at desc, d.title asc
			limit $%d offset $%d
		`, where, sortColumn, direction, len(selectArgs)-1, len(selectArgs))

		rows, err := s.pool.Query(r.Context(), selectSQL, selectArgs...)
		if err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_documents_search_failed"})
			return
		}
		defer rows.Close()

		items := make([]ArchiveDocumentSearchResult, 0, pageSize)
		for rows.Next() {
			item, err := scanArchiveDocumentSearchResult(rows, false)
			if err != nil {
				httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_documents_search_failed"})
				return
			}
			items = append(items, item)
		}
		if err := rows.Err(); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_documents_search_failed"})
			return
		}

		httpx.WritePage(w, http.StatusOK, items, total, page, pageSize)
		return
	}

	searchArgIndex := len(args) + 1
	countSearchClause := fmt.Sprintf(` and (d.search_tsv @@ websearch_to_tsquery('simple', $%d) or exists (
		select 1
		from archive_document_versions v
		join archive_document_chunks c on c.version_id = v.id
		where v.document_id = d.id
			and c.content_tsv @@ websearch_to_tsquery('simple', $%d)
	))`, searchArgIndex, searchArgIndex)
	countArgs := append(append([]any{}, args...), queryText)
	countSQL := "select count(*) from archive_documents d left join archive_taxonomy_nodes t on t.id = d.taxonomy_node_id " + where + countSearchClause
	if err := s.pool.QueryRow(r.Context(), countSQL, countArgs...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_documents_search_failed"})
		return
	}

	selectArgs := append(append([]any{}, args...), queryText, limitArg, offsetArg)
	selectSQL := fmt.Sprintf(`
		with search as (
			select websearch_to_tsquery('simple', $%d) as q
		), matching_chunks as (
			select
				v.document_id,
				max(ts_rank_cd(c.content_tsv, search.q)) as chunk_rank,
				(array_agg(ts_headline('simple', c.content, search.q, 'MaxWords=24, MinWords=10, ShortWord=3') order by ts_rank_cd(c.content_tsv, search.q) desc, c.chunk_no asc))[1] as snippet
			from archive_document_versions v
			join archive_document_chunks c on c.version_id = v.id
			cross join search
			where c.content_tsv @@ search.q
			group by v.document_id
		)
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
			to_char(d.received_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			to_char(d.created_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			to_char(d.updated_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			coalesce(ts_rank_cd(d.search_tsv, search.q), 0) + coalesce(m.chunk_rank, 0) as score,
			coalesce(m.snippet, '')
		from archive_documents d
		left join archive_taxonomy_nodes t on t.id = d.taxonomy_node_id
		left join matching_chunks m on m.document_id = d.id
		cross join search
		%s and (d.search_tsv @@ search.q or exists (
			select 1
			from archive_document_versions v
			join archive_document_chunks c on c.version_id = v.id
			where v.document_id = d.id
				and c.content_tsv @@ search.q
		))
		order by score desc, d.updated_at desc, d.title asc
		limit $%d offset $%d
	`, searchArgIndex, where, len(selectArgs)-1, len(selectArgs))

	rows, err := s.pool.Query(r.Context(), selectSQL, selectArgs...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_documents_search_failed"})
		return
	}
	defer rows.Close()

	items := make([]ArchiveDocumentSearchResult, 0, pageSize)
	for rows.Next() {
		item, err := scanArchiveDocumentSearchResult(rows, true)
		if err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_documents_search_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_documents_search_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, page, pageSize)
}

func (s *DocumentService) SearchDocumentsVector(w http.ResponseWriter, r *http.Request) {
	institutionID := strings.TrimSpace(authruntime.CurrentInstitutionIDFromRequest(r))
	if institutionID == "" {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "missing_institution_context"})
		return
	}

	queryText := strings.TrimSpace(r.URL.Query().Get("q"))
	if queryText == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_search_query"})
		return
	}
	queryEmbedding := buildArchiveEmbedding(queryText)

	filters := map[string]string{
		"status":             strings.TrimSpace(r.URL.Query().Get("status")),
		"source_kind":        strings.TrimSpace(r.URL.Query().Get("source_kind")),
		"mime_type":          strings.TrimSpace(r.URL.Query().Get("mime_type")),
		"taxonomy_code":      strings.TrimSpace(r.URL.Query().Get("taxonomy_code")),
		"external_reference": strings.TrimSpace(r.URL.Query().Get("external_reference")),
		"document_date_from": strings.TrimSpace(r.URL.Query().Get("document_date_from")),
		"document_date_to":   strings.TrimSpace(r.URL.Query().Get("document_date_to")),
		"received_at_from":   strings.TrimSpace(r.URL.Query().Get("received_at_from")),
		"received_at_to":     strings.TrimSpace(r.URL.Query().Get("received_at_to")),
		"taxonomy_label":     strings.TrimSpace(r.URL.Query().Get("taxonomy_label")),
	}
	page := parsePositiveInt(r.URL.Query().Get("page"), 1)
	pageSize := parsePositiveInt(r.URL.Query().Get("pageSize"), 25)
	if pageSize > 100 {
		pageSize = 100
	}

	where, args := buildArchiveDocumentFilters(institutionID, filters)
	offset := (page - 1) * pageSize
	baseArgs := append(append([]any{}, args...), queryEmbedding, archiveVectorSearchThreshold)

	countSQL := fmt.Sprintf(`
		select count(*)
		from archive_documents d
		left join archive_taxonomy_nodes t on t.id = d.taxonomy_node_id
		join archive_document_versions v on v.document_id = d.id and v.version_no = d.current_version_no
		%s and archive_vector_dot(v.search_embedding, $%d::double precision[]) >= $%d
	`, where, len(args)+1, len(args)+2)
	var total int
	if err := s.pool.QueryRow(r.Context(), countSQL, baseArgs...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_documents_search_failed"})
		return
	}

	selectArgs := append(append([]any{}, baseArgs...), pageSize, offset)
	selectSQL := fmt.Sprintf(`
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
			to_char(d.received_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			to_char(d.created_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			to_char(d.updated_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			archive_vector_dot(v.search_embedding, $%d::double precision[]) as score,
			coalesce(
				nullif(left(regexp_replace(coalesce(v.extracted_text, ''), '\\s+', ' ', 'g'), 280), ''),
				left(coalesce(d.title, ''), 280)
			) as snippet
		from archive_documents d
		left join archive_taxonomy_nodes t on t.id = d.taxonomy_node_id
		join archive_document_versions v on v.document_id = d.id and v.version_no = d.current_version_no
		%s and archive_vector_dot(v.search_embedding, $%d::double precision[]) >= $%d
		order by score desc, d.updated_at desc, d.title asc
		limit $%d offset $%d
	`, len(args)+1, where, len(args)+1, len(args)+2, len(selectArgs)-1, len(selectArgs))

	rows, err := s.pool.Query(r.Context(), selectSQL, selectArgs...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_documents_search_failed"})
		return
	}
	defer rows.Close()

	items := make([]ArchiveDocumentSearchResult, 0, pageSize)
	for rows.Next() {
		item, err := scanArchiveDocumentSearchResult(rows, true)
		if err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_documents_search_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_documents_search_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, page, pageSize)
}

func (s *DocumentService) SearchDocumentsHybrid(w http.ResponseWriter, r *http.Request) {
	institutionID := strings.TrimSpace(authruntime.CurrentInstitutionIDFromRequest(r))
	if institutionID == "" {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "missing_institution_context"})
		return
	}

	queryText := strings.TrimSpace(r.URL.Query().Get("q"))
	if queryText == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_search_query"})
		return
	}
	queryEmbedding := buildArchiveEmbedding(queryText)

	filters := map[string]string{
		"status":             strings.TrimSpace(r.URL.Query().Get("status")),
		"source_kind":        strings.TrimSpace(r.URL.Query().Get("source_kind")),
		"mime_type":          strings.TrimSpace(r.URL.Query().Get("mime_type")),
		"taxonomy_code":      strings.TrimSpace(r.URL.Query().Get("taxonomy_code")),
		"external_reference": strings.TrimSpace(r.URL.Query().Get("external_reference")),
		"document_date_from": strings.TrimSpace(r.URL.Query().Get("document_date_from")),
		"document_date_to":   strings.TrimSpace(r.URL.Query().Get("document_date_to")),
		"received_at_from":   strings.TrimSpace(r.URL.Query().Get("received_at_from")),
		"received_at_to":     strings.TrimSpace(r.URL.Query().Get("received_at_to")),
		"taxonomy_label":     strings.TrimSpace(r.URL.Query().Get("taxonomy_label")),
	}
	page := parsePositiveInt(r.URL.Query().Get("page"), 1)
	pageSize := parsePositiveInt(r.URL.Query().Get("pageSize"), 25)
	if pageSize > 100 {
		pageSize = 100
	}

	where, args := buildArchiveDocumentFilters(institutionID, filters)
	offset := (page - 1) * pageSize
	baseArgs := append(append([]any{}, args...), queryText, queryEmbedding, archiveVectorSearchThreshold)
	scoreExpr := fmt.Sprintf(
		`(
			least(1, coalesce(ts_rank_cd(d.search_tsv, search.q), 0) + coalesce(m.chunk_rank, 0)) * %.2f
			+ coalesce(archive_vector_dot(v.search_embedding, $%d::double precision[]), 0) * %.2f
		) as score`,
		archiveHybridFTSWeight,
		len(args)+2,
		archiveHybridVectorWeight,
	)

	countSQL := fmt.Sprintf(`
		with search as (
			select websearch_to_tsquery('simple', $%d) as q
		), fts_matches as (
			select
				v.document_id,
				max(ts_rank_cd(c.content_tsv, search.q)) as chunk_rank,
				(array_agg(ts_headline('simple', c.content, search.q, 'MaxWords=24, MinWords=10, ShortWord=3') order by ts_rank_cd(c.content_tsv, search.q) desc, c.chunk_no asc))[1] as snippet
			from archive_document_versions v
			join archive_document_chunks c on c.version_id = v.id
			cross join search
			where c.content_tsv @@ search.q
			group by v.document_id
		)
		select count(*)
		from archive_documents d
		left join archive_taxonomy_nodes t on t.id = d.taxonomy_node_id
		join archive_document_versions v on v.document_id = d.id and v.version_no = d.current_version_no
		left join fts_matches m on m.document_id = d.id
		cross join search
		%s and (
			d.search_tsv @@ search.q
			or m.document_id is not null
			or archive_vector_dot(v.search_embedding, $%d::double precision[]) >= $%d
		)
	`, len(args)+1, where, len(args)+2, len(args)+3)
	var total int
	if err := s.pool.QueryRow(r.Context(), countSQL, baseArgs...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_documents_search_failed"})
		return
	}

	selectArgs := append(append([]any{}, baseArgs...), pageSize, offset)
	selectSQL := fmt.Sprintf(`
		with search as (
			select websearch_to_tsquery('simple', $%d) as q
		), fts_matches as (
			select
				v.document_id,
				max(ts_rank_cd(c.content_tsv, search.q)) as chunk_rank,
				(array_agg(ts_headline('simple', c.content, search.q, 'MaxWords=24, MinWords=10, ShortWord=3') order by ts_rank_cd(c.content_tsv, search.q) desc, c.chunk_no asc))[1] as snippet
			from archive_document_versions v
			join archive_document_chunks c on c.version_id = v.id
			cross join search
			where c.content_tsv @@ search.q
			group by v.document_id
		)
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
			to_char(d.received_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			to_char(d.created_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			to_char(d.updated_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			%s,
			coalesce(
				m.snippet,
				nullif(left(regexp_replace(coalesce(v.extracted_text, ''), '\\s+', ' ', 'g'), 280), ''),
				left(coalesce(d.title, ''), 280)
			) as snippet
		from archive_documents d
		left join archive_taxonomy_nodes t on t.id = d.taxonomy_node_id
		join archive_document_versions v on v.document_id = d.id and v.version_no = d.current_version_no
		left join fts_matches m on m.document_id = d.id
		cross join search
		%s and (
			d.search_tsv @@ search.q
			or m.document_id is not null
			or archive_vector_dot(v.search_embedding, $%d::double precision[]) >= $%d
		)
		order by score desc, d.updated_at desc, d.title asc
		limit $%d offset $%d
	`, len(args)+1, scoreExpr, where, len(args)+2, len(args)+3, len(selectArgs)-1, len(selectArgs))

	rows, err := s.pool.Query(r.Context(), selectSQL, selectArgs...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_documents_search_failed"})
		return
	}
	defer rows.Close()

	items := make([]ArchiveDocumentSearchResult, 0, pageSize)
	for rows.Next() {
		item, err := scanArchiveDocumentSearchResult(rows, true)
		if err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_documents_search_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_documents_search_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, page, pageSize)
}

func (s *DocumentService) GetDocument(w http.ResponseWriter, r *http.Request) {
	documentID := strings.TrimSpace(chi.URLParam(r, "documentID"))
	if documentID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_document_id"})
		return
	}

	document, err := s.loadDocumentDetail(r.Context(), authruntime.CurrentInstitutionIDFromRequest(r), documentID)
	if err != nil {
		if err == pgx.ErrNoRows {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "archive_document_not_found"})
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_document_load_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, document)
}

func (s *DocumentService) DownloadOriginal(w http.ResponseWriter, r *http.Request) {
	if s.storage == nil || !s.storage.Enabled() {
		httpx.JSON(w, http.StatusServiceUnavailable, map[string]any{"code": "archive_storage_unavailable"})
		return
	}
	institutionID := strings.TrimSpace(authruntime.CurrentInstitutionIDFromRequest(r))
	documentID := strings.TrimSpace(chi.URLParam(r, "documentID"))
	if institutionID == "" {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "missing_institution_context"})
		return
	}
	if _, err := uuid.Parse(documentID); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_archive_document_id"})
		return
	}
	var objectKey string
	if err := s.pool.QueryRow(r.Context(), `
		select original_object_key
		from archive_documents
		where id = $1::uuid and institution_id = $2
	`, documentID, institutionID).Scan(&objectKey); errors.Is(err, pgx.ErrNoRows) {
		httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "archive_document_not_found"})
		return
	} else if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_document_download_failed"})
		return
	}
	actor := authruntime.CurrentSubjectFromRequest(r)
	if err := audit.Log(r.Context(), s.pool, audit.Event{ActorSubject: actor, Action: "earchiva.document.download.requested", TargetType: "archive_document", TargetID: documentID, Summary: "Archive original download authorized."}); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_document_download_audit_failed"})
		return
	}
	content, err := s.storage.OpenObject(r.Context(), objectKey)
	if err != nil {
		httpx.JSON(w, http.StatusBadGateway, map[string]any{"code": "archive_document_download_failed"})
		return
	}
	defer content.Close() //nolint:errcheck
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="archive-%s.pdf"`, documentID))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, copyErr := io.Copy(w, content)
	auditCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 2*time.Second)
	defer cancel()
	outcome := "completed"
	if copyErr != nil {
		outcome = "interrupted"
	}
	_ = audit.Log(auditCtx, s.pool, audit.Event{ActorSubject: actor, Action: "earchiva.document.download." + outcome, TargetType: "archive_document", TargetID: documentID, Summary: "Archive original download " + outcome + "."})
}

func (s *DocumentService) ListDocumentVersions(w http.ResponseWriter, r *http.Request) {
	documentID := strings.TrimSpace(chi.URLParam(r, "documentID"))
	if documentID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_document_id"})
		return
	}

	rows, err := s.pool.Query(r.Context(), `
		select
			v.id::text,
			v.document_id::text,
			v.version_no,
			v.source_bucket,
			v.source_object_key,
			v.artifact_bucket,
			v.artifact_object_key,
			v.source_sha256,
			v.source_size_bytes,
			v.page_count,
			v.text_status,
			v.created_by,
			to_char(v.created_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			count(c.id)::int
		from archive_document_versions v
		left join archive_document_chunks c on c.version_id = v.id
		where v.document_id::text = $1 and v.institution_id = $2
		group by v.id
		order by v.version_no desc
	`, documentID, authruntime.CurrentInstitutionIDFromRequest(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_document_versions_failed"})
		return
	}
	defer rows.Close()

	items := make([]ArchiveDocumentVersionSummary, 0)
	for rows.Next() {
		var item ArchiveDocumentVersionSummary
		if err := rows.Scan(
			&item.ID,
			&item.DocumentID,
			&item.VersionNo,
			&item.SourceBucket,
			&item.SourceObjectKey,
			&item.ArtifactBucket,
			&item.ArtifactObjectKey,
			&item.SourceSHA256,
			&item.SourceSizeBytes,
			&item.PageCount,
			&item.TextStatus,
			&item.CreatedBy,
			&item.CreatedAt,
			&item.ChunkCount,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_document_versions_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_document_versions_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, items)
}

func (s *DocumentService) ListTaxonomyNodes(w http.ResponseWriter, r *http.Request) {
	institutionID := strings.TrimSpace(authruntime.CurrentInstitutionIDFromRequest(r))
	if institutionID == "" {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "missing_institution_context"})
		return
	}

	parentCode := strings.TrimSpace(r.URL.Query().Get("parent_code"))
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	activeOnly := strings.TrimSpace(r.URL.Query().Get("active"))
	args := []any{institutionID}
	clauses := []string{"institution_id = $1"}
	if parentCode != "" {
		args = append(args, parentCode)
		clauses = append(clauses, fmt.Sprintf("coalesce((select p.code from archive_taxonomy_nodes p where p.id = archive_taxonomy_nodes.parent_id), '') = $%d", len(args)))
	}
	if search != "" {
		args = append(args, "%"+strings.ToLower(search)+"%")
		clauses = append(clauses, fmt.Sprintf("(lower(code) like $%d or lower(label) like $%d or lower(path) like $%d)", len(args), len(args), len(args)))
	}
	if activeOnly == "true" || activeOnly == "1" {
		clauses = append(clauses, "active = true")
	}

	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select
			id::text,
			parent_id::text,
			code,
			label,
			description,
			path,
			active,
			sort_order,
			to_char(created_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			to_char(updated_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		from archive_taxonomy_nodes
		where %s
		order by path asc, sort_order asc, label asc
	`, strings.Join(clauses, " and ")), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_taxonomy_failed"})
		return
	}
	defer rows.Close()

	items := []ArchiveTaxonomyNode{}
	for rows.Next() {
		var item ArchiveTaxonomyNode
		var parentID sql.NullString
		if err := rows.Scan(&item.ID, &parentID, &item.Code, &item.Label, &item.Description, &item.Path, &item.Active, &item.SortOrder, &item.CreatedAt, &item.UpdatedAt); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_taxonomy_failed"})
			return
		}
		if parentID.Valid {
			value := strings.TrimSpace(parentID.String)
			item.ParentID = &value
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_taxonomy_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, items)
}

func (s *DocumentService) UploadDocument(w http.ResponseWriter, r *http.Request) {
	if s.storage == nil || !s.storage.Enabled() {
		httpx.JSON(w, http.StatusServiceUnavailable, map[string]any{"code": "archive_storage_unavailable"})
		return
	}

	select {
	case archiveUploadSlots <- struct{}{}:
		defer func() { <-archiveUploadSlots }()
	default:
		httpx.JSON(w, http.StatusTooManyRequests, map[string]any{"code": "archive_upload_busy"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, archiveUploadMaxBytes+archiveUploadOverhead)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_archive_upload"})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_archive_file"})
		return
	}
	defer file.Close() //nolint:errcheck

	institutionID := strings.TrimSpace(authruntime.CurrentInstitutionIDFromRequest(r))
	if institutionID == "" {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "missing_institution_context"})
		return
	}

	payload, err := parseArchiveUploadPayload(r, header)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_archive_upload", "message": err.Error()})
		return
	}

	if err := validateArchiveUploadPayload(payload); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_archive_upload", "message": err.Error()})
		return
	}
	if existingID, found, err := s.findIdempotentArchiveDocument(r.Context(), institutionID, payload.IdempotencyKey); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_idempotency_lookup_failed"})
		return
	} else if found {
		detail, err := s.loadDocumentDetail(r.Context(), institutionID, existingID)
		if err != nil {
			httpx.JSON(w, http.StatusConflict, map[string]any{"code": "archive_idempotency_conflict"})
			return
		}
		httpx.JSON(w, http.StatusOK, detail)
		return
	}

	staged, err := stageAndValidateArchivePDF(r.Context(), file, s.scanner)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_archive_file", "message": err.Error()})
		return
	}
	defer os.Remove(staged.Path)
	payload.FileSize, payload.ChecksumSHA256 = staged.Size, staged.SHA256
	payload.MimeType = "application/pdf"
	payload.Metadata["source_file_name"] = payload.FileName
	payload.Metadata["source_page_count"] = staged.PageCount
	if payload.IdempotencyKey == "" {
		payload.IdempotencyKey = deriveArchiveIdempotencyKey(payload)
	}
	if existingID, found, lookupErr := s.findIdempotentArchiveDocument(r.Context(), institutionID, payload.IdempotencyKey); lookupErr != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_idempotency_lookup_failed"})
		return
	} else if found {
		detail, loadErr := s.loadDocumentDetail(r.Context(), institutionID, existingID)
		if loadErr != nil {
			httpx.JSON(w, http.StatusConflict, map[string]any{"code": "archive_idempotency_conflict"})
			return
		}
		httpx.JSON(w, http.StatusOK, detail)
		return
	}

	documentID := uuid.NewString()
	versionID := uuid.NewString()
	payload.OriginalBucket = s.storage.Bucket()
	payload.ArtifactBucket = s.storage.Bucket()
	payload.OriginalObjectKey = s.storage.OriginalObjectKey(institutionID, documentID, canonicalArchiveStorageFileName(documentID))
	payload.ArtifactObjectKey = s.storage.ArtifactObjectKey(institutionID, documentID, 1)

	stagedFile, err := os.Open(staged.Path)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_upload_staging_failed"})
		return
	}
	defer stagedFile.Close() //nolint:errcheck
	if err := s.storage.PutObject(r.Context(), payload.OriginalObjectKey, payload.MimeType, stagedFile, payload.FileSize); err != nil {
		httpx.JSON(w, http.StatusBadGateway, map[string]any{"code": "archive_upload_storage_failed"})
		return
	}

	// Once object storage accepted the bitstream, finish the database boundary
	// independently of a client disconnect. Otherwise a cancelled request can
	// leave an untracked object or a committed document without its audit event.
	persistCtx, cancelPersist := context.WithTimeout(context.WithoutCancel(r.Context()), 30*time.Second)
	defer cancelPersist()
	cleanupObject := func() {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 30*time.Second)
		defer cancel()
		_ = s.storage.DeleteObject(cleanupCtx, payload.OriginalObjectKey)
	}

	tx, err := s.pool.Begin(persistCtx)
	if err != nil {
		cleanupObject()
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_upload_failed"})
		return
	}
	defer tx.Rollback(persistCtx) //nolint:errcheck

	taxonomyNodeID, taxonomyCode, taxonomyLabel, err := s.ensureTaxonomyNodeTx(persistCtx, tx, institutionID, payload.TaxonomyCode, payload.TaxonomyLabel, payload.TaxonomyParent)
	if err != nil {
		cleanupObject()
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "archive_taxonomy_failed", "message": err.Error()})
		return
	}

	metadataJSON, err := json.Marshal(payload.Metadata)
	if err != nil {
		cleanupObject()
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "archive_metadata_failed"})
		return
	}

	var document ArchiveDocument
	if err := tx.QueryRow(persistCtx, `
		insert into archive_documents (
			id,
			institution_id,
			title,
			original_file_name,
			mime_type,
			source_kind,
			source_system,
			external_reference,
			taxonomy_node_id,
			status,
			original_bucket,
			original_object_key,
			artifact_bucket,
			artifact_object_key,
			document_date,
			metadata,
			idempotency_key,
			created_by,
			current_version_no,
			received_at
		) values (
			$1::uuid,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9::uuid,
			'queued',
			$10,
			$11,
			$12,
			$13,
			$14::date,
			$15::jsonb,
			$16,
			$17,
			1,
			now()
		)
		returning
			id::text,
			institution_id,
			title,
			original_file_name,
			mime_type,
			source_kind,
			source_system,
			external_reference,
			taxonomy_node_id::text,
			status,
			original_bucket,
			original_object_key,
			artifact_bucket,
			artifact_object_key,
			case when document_date is null then null else to_char(document_date, 'YYYY-MM-DD') end,
			metadata,
			current_version_no,
			to_char(received_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			to_char(created_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			to_char(updated_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
	`, documentID, institutionID, payload.Title, payload.FileName, payload.MimeType, payload.SourceKind, payload.SourceSystem, payload.ExternalReference, taxonomyNodeID, payload.OriginalBucket, payload.OriginalObjectKey, payload.ArtifactBucket, payload.ArtifactObjectKey, payload.DocumentDate, metadataJSON, payload.IdempotencyKey, authruntime.CurrentSubjectFromRequest(r)).Scan(
		&document.ID,
		&document.InstitutionID,
		&document.Title,
		&document.OriginalFileName,
		&document.MimeType,
		&document.SourceKind,
		&document.SourceSystem,
		&document.ExternalReference,
		&document.TaxonomyNodeID,
		&document.Status,
		&document.OriginalBucket,
		&document.OriginalObjectKey,
		&document.ArtifactBucket,
		&document.ArtifactObjectKey,
		&document.DocumentDate,
		&metadataJSON,
		&document.CurrentVersionNo,
		&document.ReceivedAt,
		&document.CreatedAt,
		&document.UpdatedAt,
	); err != nil {
		cleanupObject()
		if isArchiveIdempotencyConflict(err) {
			if existingID, found, lookupErr := s.findIdempotentArchiveDocument(persistCtx, institutionID, payload.IdempotencyKey); lookupErr == nil && found {
				detail, loadErr := s.loadDocumentDetail(persistCtx, institutionID, existingID)
				if loadErr == nil {
					httpx.JSON(w, http.StatusOK, detail)
					return
				}
			}
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_upload_failed"})
		return
	}
	if err := unmarshalMetadata(metadataJSON, &document.Metadata); err != nil {
		cleanupObject()
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_upload_failed"})
		return
	}
	document.TaxonomyCode = taxonomyCode
	document.TaxonomyLabel = taxonomyLabel

	var version ArchiveDocumentVersion
	if err := tx.QueryRow(persistCtx, `
		insert into archive_document_versions (
			id,
			institution_id,
			document_id,
			version_no,
			mime_type,
			title,
			bucket_name,
			object_key,
			hash_sha256,
			size_bytes,
			metadata,
			ocr_text,
			status,
			source_bucket,
			source_object_key,
			artifact_bucket,
			artifact_object_key,
			source_sha256,
			source_size_bytes,
			page_count,
			text_status,
			extracted_text,
			extracted_metadata,
			created_by
		) values (
			$1::uuid,
			$2,
			$3::uuid,
			1,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10::jsonb,
			'',
			'active',
			$6,
			$7,
			$11,
			$12,
			$8,
			$9,
			0,
			'pending',
			'',
			$10::jsonb,
			$13
		)
		returning
			id::text,
			document_id::text,
			version_no,
			source_bucket,
			source_object_key,
			artifact_bucket,
			artifact_object_key,
			source_sha256,
			source_size_bytes,
			page_count,
			text_status,
			created_by,
			to_char(created_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
	`, versionID, institutionID, documentID, payload.MimeType, payload.Title, payload.OriginalBucket, payload.OriginalObjectKey, payload.ChecksumSHA256, payload.FileSize, metadataJSON, payload.ArtifactBucket, payload.ArtifactObjectKey, authruntime.CurrentSubjectFromRequest(r)).Scan(
		&version.ID,
		&version.DocumentID,
		&version.VersionNo,
		&version.SourceBucket,
		&version.SourceObjectKey,
		&version.ArtifactBucket,
		&version.ArtifactObjectKey,
		&version.SourceSHA256,
		&version.SourceSizeBytes,
		&version.PageCount,
		&version.TextStatus,
		&version.CreatedBy,
		&version.CreatedAt,
	); err != nil {
		cleanupObject()
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_upload_failed"})
		return
	}

	if _, err := tx.Exec(persistCtx, `
		insert into archive_ingestion_jobs (
			id,
			institution_id,
			document_id,
			version_id,
			job_type,
			status,
			available_at,
			created_by
		) values ($1::uuid, $2, $3::uuid, $4::uuid, 'extract_text', 'pending', now(), $5)
		`, uuid.NewString(), institutionID, documentID, version.ID, authruntime.CurrentSubjectFromRequest(r)); err != nil {
		cleanupObject()
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_upload_failed"})
		return
	}

	if err := audit.Log(persistCtx, tx, audit.Event{ActorSubject: authruntime.CurrentSubjectFromRequest(r), Action: "earchiva.document.upload", TargetType: "archive_document", TargetID: documentID, Summary: "Archive document accepted for ingestion.", Details: map[string]any{"source_kind": payload.SourceKind, "page_count": staged.PageCount}}); err != nil {
		cleanupObject()
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_upload_audit_failed"})
		return
	}

	if err := tx.Commit(persistCtx); err != nil {
		// Only delete when PostgreSQL confirms the transaction was rolled back.
		// For transport/commit-ack failures the outcome is indeterminate; deleting
		// then could remove the source object of a committed archive record.
		if errors.Is(err, pgx.ErrTxCommitRollback) {
			cleanupObject()
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_upload_failed"})
		return
	}
	detail, err := s.loadDocumentDetail(persistCtx, institutionID, documentID)
	if err != nil {
		detail = ArchiveDocumentDetail{ArchiveDocument: document}
	}
	httpx.JSON(w, http.StatusCreated, detail)
}

func (s *DocumentService) loadDocumentDetail(ctx context.Context, institutionID, documentID string) (ArchiveDocumentDetail, error) {
	var detail ArchiveDocumentDetail
	var metadataJSON []byte
	var taxonomyID sql.NullString
	var taxonomyCode sql.NullString
	var taxonomyLabel sql.NullString
	var documentDate sql.NullString
	if err := s.pool.QueryRow(ctx, `
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
			to_char(d.received_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			to_char(d.created_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			to_char(d.updated_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		from archive_documents d
		left join archive_taxonomy_nodes t on t.id = d.taxonomy_node_id
		where d.id::text = $1 and d.institution_id = $2
	`, documentID, institutionID).Scan(
		&detail.ID,
		&detail.InstitutionID,
		&detail.Title,
		&detail.OriginalFileName,
		&detail.MimeType,
		&detail.SourceKind,
		&detail.SourceSystem,
		&detail.ExternalReference,
		&taxonomyID,
		&taxonomyCode,
		&taxonomyLabel,
		&detail.Status,
		&detail.OriginalBucket,
		&detail.OriginalObjectKey,
		&detail.ArtifactBucket,
		&detail.ArtifactObjectKey,
		&documentDate,
		&metadataJSON,
		&detail.CurrentVersionNo,
		&detail.ReceivedAt,
		&detail.CreatedAt,
		&detail.UpdatedAt,
	); err != nil {
		return ArchiveDocumentDetail{}, err
	}
	if taxonomyID.Valid {
		value := strings.TrimSpace(taxonomyID.String)
		detail.TaxonomyNodeID = &value
	}
	if taxonomyCode.Valid {
		value := strings.TrimSpace(taxonomyCode.String)
		detail.TaxonomyCode = &value
	}
	if taxonomyLabel.Valid {
		value := strings.TrimSpace(taxonomyLabel.String)
		detail.TaxonomyLabel = &value
	}
	if documentDate.Valid {
		value := strings.TrimSpace(documentDate.String)
		detail.DocumentDate = &value
	}
	if err := unmarshalMetadata(metadataJSON, &detail.Metadata); err != nil {
		return ArchiveDocumentDetail{}, err
	}

	version, err := s.loadLatestVersion(ctx, documentID)
	if err != nil {
		if err != pgx.ErrNoRows {
			return ArchiveDocumentDetail{}, err
		}
	} else {
		detail.LatestVersion = &version
	}

	return detail, nil
}

func (s *DocumentService) loadLatestVersion(ctx context.Context, documentID string) (ArchiveDocumentVersion, error) {
	var version ArchiveDocumentVersion
	if err := s.pool.QueryRow(ctx, `
		select
			id::text,
			document_id::text,
			version_no,
			source_bucket,
			source_object_key,
			artifact_bucket,
			artifact_object_key,
			source_sha256,
			source_size_bytes,
			page_count,
			text_status,
			created_by,
			to_char(created_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		from archive_document_versions
		where document_id::text = $1
		order by version_no desc
		limit 1
	`, documentID).Scan(
		&version.ID,
		&version.DocumentID,
		&version.VersionNo,
		&version.SourceBucket,
		&version.SourceObjectKey,
		&version.ArtifactBucket,
		&version.ArtifactObjectKey,
		&version.SourceSHA256,
		&version.SourceSizeBytes,
		&version.PageCount,
		&version.TextStatus,
		&version.CreatedBy,
		&version.CreatedAt,
	); err != nil {
		return ArchiveDocumentVersion{}, err
	}
	return version, nil
}

func (s *DocumentService) ensureTaxonomyNodeTx(ctx context.Context, tx pgx.Tx, institutionID, code, label, parentCode string) (string, *string, *string, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return "", nil, nil, nil
	}
	if label == "" {
		label = code
	}

	var parentID sql.NullString
	var parentPath sql.NullString
	if parentCode = strings.TrimSpace(parentCode); parentCode != "" {
		if err := tx.QueryRow(ctx, `select id::text, path from archive_taxonomy_nodes where institution_id = $1 and code = $2`, institutionID, parentCode).Scan(&parentID, &parentPath); err != nil {
			return "", nil, nil, err
		}
	}

	var nodeID string
	var nodeLabel string
	var nodeCode string
	var pathValue string
	if parentPath.Valid && strings.TrimSpace(parentPath.String) != "" {
		pathValue = strings.TrimSpace(parentPath.String) + "/" + code
	} else {
		pathValue = code
	}

	if err := tx.QueryRow(ctx, `
		insert into archive_taxonomy_nodes (
			institution_id,
			parent_id,
			code,
			label,
			path,
			active
		) values ($1, $2::uuid, $3, $4, $5, true)
		on conflict (institution_id, code) do update
		set parent_id = excluded.parent_id,
			label = excluded.label,
			path = excluded.path,
			active = true,
			updated_at = now()
		returning id::text, code, label, path
	`, institutionID, nullStringOrNil(parentID), code, label, pathValue).Scan(&nodeID, &nodeCode, &nodeLabel, &pathValue); err != nil {
		return "", nil, nil, err
	}

	codeValue := strings.TrimSpace(nodeCode)
	labelValue := strings.TrimSpace(nodeLabel)
	return nodeID, &codeValue, &labelValue, nil
}

func buildArchiveDocumentFilters(institutionID string, filters map[string]string) (string, []any) {
	clauses := []string{"d.institution_id = $1"}
	args := []any{strings.TrimSpace(institutionID)}
	appendClause := func(sqlFragment string, value any) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf(sqlFragment, len(args)))
	}

	if value := strings.TrimSpace(filters["status"]); value != "" {
		appendClause("d.status = $%d", value)
	}
	if value := strings.TrimSpace(filters["source_kind"]); value != "" {
		appendClause("d.source_kind = $%d", value)
	}
	if value := strings.TrimSpace(filters["mime_type"]); value != "" {
		appendClause("d.mime_type = $%d", value)
	}
	if value := strings.TrimSpace(filters["external_reference"]); value != "" {
		appendClause("d.external_reference = $%d", value)
	}
	if value := strings.TrimSpace(filters["taxonomy_code"]); value != "" {
		appendClause("t.code = $%d", value)
	}
	if value := strings.TrimSpace(filters["taxonomy_label"]); value != "" {
		appendClause("lower(t.label) like lower('%%' || $%d || '%%')", value)
	}
	if value := strings.TrimSpace(filters["document_date_from"]); value != "" {
		appendClause("d.document_date >= $%d::date", value)
	}
	if value := strings.TrimSpace(filters["document_date_to"]); value != "" {
		appendClause("d.document_date <= $%d::date", value)
	}
	if value := strings.TrimSpace(filters["received_at_from"]); value != "" {
		appendClause("d.received_at >= $%d::timestamptz", value)
	}
	if value := strings.TrimSpace(filters["received_at_to"]); value != "" {
		appendClause("d.received_at <= $%d::timestamptz", value)
	}

	return " where " + strings.Join(clauses, " and "), args
}

func parseArchiveUploadPayload(r *http.Request, header *multipart.FileHeader) (archiveUploadPayload, error) {
	metadata, err := parseArchiveMetadata(strings.TrimSpace(r.FormValue("metadata")))
	if err != nil {
		return archiveUploadPayload{}, err
	}

	payload := archiveUploadPayload{
		Title:             strings.TrimSpace(r.FormValue("title")),
		SourceKind:        normalizeArchiveSourceKind(r.FormValue("source_kind")),
		SourceSystem:      strings.TrimSpace(r.FormValue("source_system")),
		ExternalReference: strings.TrimSpace(r.FormValue("external_reference")),
		TaxonomyCode:      strings.TrimSpace(r.FormValue("taxonomy_code")),
		TaxonomyLabel:     strings.TrimSpace(r.FormValue("taxonomy_label")),
		TaxonomyParent:    strings.TrimSpace(r.FormValue("taxonomy_parent_code")),
		Metadata:          metadata,
		MimeType:          strings.TrimSpace(header.Header.Get("Content-Type")),
		FileName:          strings.TrimSpace(header.Filename),
		FileSize:          header.Size,
		IdempotencyKey:    strings.TrimSpace(firstNonEmpty(r.Header.Get("Idempotency-Key"), r.FormValue("idempotency_key"))),
	}
	if payload.SourceSystem == "" {
		payload.SourceSystem = "manual-upload"
	}
	if payload.Title == "" {
		payload.Title = strings.TrimSuffix(filepath.Base(payload.FileName), filepath.Ext(payload.FileName))
	}
	if payload.Title == "" {
		payload.Title = "Archive document"
	}
	if payload.MimeType == "" {
		switch strings.ToLower(filepath.Ext(payload.FileName)) {
		case ".pdf":
			payload.MimeType = "application/pdf"
		case ".txt":
			payload.MimeType = "text/plain"
		default:
			payload.MimeType = "application/octet-stream"
		}
	}
	if payload.DocumentDate, err = parseOptionalDate(strings.TrimSpace(r.FormValue("document_date"))); err != nil {
		return archiveUploadPayload{}, err
	}
	if payload.SourceKind == "" {
		payload.SourceKind = "legacy_pdf"
	}
	return payload, nil
}

func validateArchiveUploadPayload(payload archiveUploadPayload) error {
	if payload.FileName == "" {
		return fmt.Errorf("missing file name")
	}
	if len(payload.IdempotencyKey) > 200 {
		return fmt.Errorf("idempotency key is too long")
	}
	if payload.FileSize <= 0 {
		return fmt.Errorf("missing file content")
	}
	if payload.MimeType != "application/pdf" && payload.MimeType != "application/octet-stream" {
		return fmt.Errorf("unsupported mime type %s", payload.MimeType)
	}
	if payload.SourceKind == "" {
		return fmt.Errorf("missing source kind")
	}
	return nil
}

func parseArchiveMetadata(raw string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}, nil
	}
	if len(raw) > 64<<10 {
		return nil, fmt.Errorf("metadata exceeds the 64 KiB limit")
	}
	decoded := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, fmt.Errorf("metadata must be valid json")
	}
	return decoded, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func (s *DocumentService) findIdempotentArchiveDocument(ctx context.Context, institutionID, key string) (string, bool, error) {
	if strings.TrimSpace(key) == "" {
		return "", false, nil
	}
	var id string
	err := s.pool.QueryRow(ctx, `select id::text from archive_documents where institution_id = $1 and idempotency_key = $2`, institutionID, key).Scan(&id)
	if err == pgx.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return id, true, nil
}

func isArchiveIdempotencyConflict(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && strings.Contains(pgErr.ConstraintName, "archive_documents_institution_idempotency")
}

func parseOptionalDate(raw string) (*string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, fmt.Errorf("document_date must use YYYY-MM-DD")
	}
	value := parsed.Format("2006-01-02")
	return &value, nil
}

func normalizeArchiveSourceKind(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "legacy_pdf":
		return "legacy_pdf"
	case "upload", "manual_upload":
		return "upload"
	case "import", "bulk_import":
		return "import"
	default:
		return ""
	}
}

func parsePositiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func scanArchiveDocumentSearchResult(rows pgx.Rows, includeScore bool) (ArchiveDocumentSearchResult, error) {
	var item ArchiveDocumentSearchResult
	var metadataJSON []byte
	var taxonomyID sql.NullString
	var taxonomyCode sql.NullString
	var taxonomyLabel sql.NullString
	var documentDate sql.NullString
	if includeScore {
		if err := rows.Scan(
			&item.ID,
			&item.InstitutionID,
			&item.Title,
			&item.OriginalFileName,
			&item.MimeType,
			&item.SourceKind,
			&item.SourceSystem,
			&item.ExternalReference,
			&taxonomyID,
			&taxonomyCode,
			&taxonomyLabel,
			&item.Status,
			&item.OriginalBucket,
			&item.OriginalObjectKey,
			&item.ArtifactBucket,
			&item.ArtifactObjectKey,
			&documentDate,
			&metadataJSON,
			&item.CurrentVersionNo,
			&item.ReceivedAt,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.Score,
			&item.Snippet,
		); err != nil {
			return ArchiveDocumentSearchResult{}, err
		}
	} else {
		if err := rows.Scan(
			&item.ID,
			&item.InstitutionID,
			&item.Title,
			&item.OriginalFileName,
			&item.MimeType,
			&item.SourceKind,
			&item.SourceSystem,
			&item.ExternalReference,
			&taxonomyID,
			&taxonomyCode,
			&taxonomyLabel,
			&item.Status,
			&item.OriginalBucket,
			&item.OriginalObjectKey,
			&item.ArtifactBucket,
			&item.ArtifactObjectKey,
			&documentDate,
			&metadataJSON,
			&item.CurrentVersionNo,
			&item.ReceivedAt,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return ArchiveDocumentSearchResult{}, err
		}
	}
	if taxonomyID.Valid {
		value := strings.TrimSpace(taxonomyID.String)
		item.TaxonomyNodeID = &value
	}
	if taxonomyCode.Valid {
		value := strings.TrimSpace(taxonomyCode.String)
		item.TaxonomyCode = &value
	}
	if taxonomyLabel.Valid {
		value := strings.TrimSpace(taxonomyLabel.String)
		item.TaxonomyLabel = &value
	}
	if documentDate.Valid {
		value := strings.TrimSpace(documentDate.String)
		item.DocumentDate = &value
	}
	if err := unmarshalMetadata(metadataJSON, &item.Metadata); err != nil {
		return ArchiveDocumentSearchResult{}, err
	}
	return item, nil
}

func nullStringOrNil(value sql.NullString) any {
	if value.Valid {
		return strings.TrimSpace(value.String)
	}
	return nil
}

func unmarshalMetadata(raw []byte, target *map[string]any) error {
	if len(raw) == 0 {
		*target = map[string]any{}
		return nil
	}
	decoded := map[string]any{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return err
	}
	if len(decoded) == 0 {
		*target = map[string]any{}
		return nil
	}
	keys := make([]string, 0, len(decoded))
	for key := range decoded {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	ordered := make(map[string]any, len(decoded))
	for _, key := range keys {
		ordered[key] = decoded[key]
	}
	*target = ordered
	return nil
}
