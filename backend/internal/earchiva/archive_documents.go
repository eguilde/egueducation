package earchiva

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	authruntime "github.com/eguilde/egueducation/internal/auth"
	appdb "github.com/eguilde/egueducation/internal/db"
	"github.com/eguilde/egueducation/internal/httpx"
)

type DocumentService struct {
	pool    *appdb.SessionPool
	storage *ArchiveStorage
}

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
	OriginalBucket    string         `json:"original_bucket"`
	OriginalObjectKey string         `json:"original_object_key"`
	ArtifactBucket    string         `json:"artifact_bucket"`
	ArtifactObjectKey string         `json:"artifact_object_key"`
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
	SourceBucket      string `json:"source_bucket"`
	SourceObjectKey   string `json:"source_object_key"`
	ArtifactBucket    string `json:"artifact_bucket"`
	ArtifactObjectKey string `json:"artifact_object_key"`
	SourceSHA256      string `json:"source_sha256"`
	SourceSizeBytes   int64  `json:"source_size_bytes"`
	PageCount         int    `json:"page_count"`
	TextStatus        string `json:"text_status"`
	CreatedBy         string `json:"created_by"`
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
	`, where, len(args)+2, len(args)+3)
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
	`, len(args)+2, where, len(args)+2, len(args)+3, len(selectArgs)-1, len(selectArgs))

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

	document, err := s.loadDocumentDetail(r.Context(), documentID)
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
		where v.document_id::text = $1
		group by v.id
		order by v.version_no desc
	`, documentID)
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

	if err := r.ParseMultipartForm(32 << 20); err != nil {
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

	documentID := uuid.NewString()
	versionID := uuid.NewString()
	payload.OriginalBucket = s.storage.Bucket()
	payload.ArtifactBucket = s.storage.Bucket()
	payload.OriginalObjectKey = s.storage.OriginalObjectKey(institutionID, documentID, payload.FileName)
	payload.ArtifactObjectKey = s.storage.ArtifactObjectKey(institutionID, documentID, 1)

	hash := sha256.New()
	tee := io.TeeReader(file, hash)
	if err := s.storage.PutObject(r.Context(), payload.OriginalObjectKey, payload.MimeType, tee, payload.FileSize); err != nil {
		httpx.JSON(w, http.StatusBadGateway, map[string]any{"code": "archive_upload_storage_failed"})
		return
	}
	payload.ChecksumSHA256 = hex.EncodeToString(hash.Sum(nil))

	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		_ = s.storage.DeleteObject(r.Context(), payload.OriginalObjectKey)
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_upload_failed"})
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck

	taxonomyNodeID, taxonomyCode, taxonomyLabel, err := s.ensureTaxonomyNodeTx(r.Context(), tx, institutionID, payload.TaxonomyCode, payload.TaxonomyLabel, payload.TaxonomyParent)
	if err != nil {
		_ = s.storage.DeleteObject(r.Context(), payload.OriginalObjectKey)
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "archive_taxonomy_failed", "message": err.Error()})
		return
	}

	metadataJSON, err := json.Marshal(payload.Metadata)
	if err != nil {
		_ = s.storage.DeleteObject(r.Context(), payload.OriginalObjectKey)
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "archive_metadata_failed"})
		return
	}

	var document ArchiveDocument
	if err := tx.QueryRow(r.Context(), `
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
	`, documentID, institutionID, payload.Title, payload.FileName, payload.MimeType, payload.SourceKind, payload.SourceSystem, payload.ExternalReference, taxonomyNodeID, payload.OriginalBucket, payload.OriginalObjectKey, payload.ArtifactBucket, payload.ArtifactObjectKey, payload.DocumentDate, metadataJSON).Scan(
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
		_ = s.storage.DeleteObject(r.Context(), payload.OriginalObjectKey)
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_upload_failed"})
		return
	}
	if err := unmarshalMetadata(metadataJSON, &document.Metadata); err != nil {
		_ = s.storage.DeleteObject(r.Context(), payload.OriginalObjectKey)
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_upload_failed"})
		return
	}
	document.TaxonomyCode = taxonomyCode
	document.TaxonomyLabel = taxonomyLabel

	var version ArchiveDocumentVersion
	if err := tx.QueryRow(r.Context(), `
		insert into archive_document_versions (
			id,
			institution_id,
			document_id,
			version_no,
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
			0,
			'pending',
			'',
			$10::jsonb,
			$11
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
	`, versionID, institutionID, documentID, payload.OriginalBucket, payload.OriginalObjectKey, payload.ArtifactBucket, payload.ArtifactObjectKey, payload.ChecksumSHA256, payload.FileSize, metadataJSON, authruntime.CurrentSubjectFromRequest(r)).Scan(
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
		_ = s.storage.DeleteObject(r.Context(), payload.OriginalObjectKey)
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_upload_failed"})
		return
	}

	if _, err := tx.Exec(r.Context(), `
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
		_ = s.storage.DeleteObject(r.Context(), payload.OriginalObjectKey)
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_upload_failed"})
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		_ = s.storage.DeleteObject(r.Context(), payload.OriginalObjectKey)
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_upload_failed"})
		return
	}

	detail, err := s.loadDocumentDetail(r.Context(), documentID)
	if err != nil {
		detail = ArchiveDocumentDetail{ArchiveDocument: document}
	}
	httpx.JSON(w, http.StatusCreated, detail)
}

func (s *DocumentService) loadDocumentDetail(ctx context.Context, documentID string) (ArchiveDocumentDetail, error) {
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
		where d.id::text = $1
	`, documentID).Scan(
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
	if payload.FileSize <= 0 {
		return fmt.Errorf("missing file content")
	}
	if payload.MimeType != "application/pdf" && payload.MimeType != "text/plain" && payload.MimeType != "application/octet-stream" {
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
	decoded := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, fmt.Errorf("metadata must be valid json")
	}
	return decoded, nil
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
