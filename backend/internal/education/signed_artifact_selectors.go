package education

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/eguilde/egueducation/internal/httpx"
)

var signedArtifactTypes = []string{"decision", "publication", "managerial_document", "meeting_document", "meeting_minute", "meeting_resolution"}

type EligibleSignedArtifact struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Type  string `json:"type"`
}

type EligibleSignatureArchiveVersion struct {
	DocumentID       string `json:"document_id"`
	VersionID        string `json:"version_id"`
	Title            string `json:"title"`
	VersionNo        int    `json:"version_no"`
	DocumentSHA256   string `json:"document_sha256"`
	StorageBucket    string `json:"storage_bucket"`
	StorageObjectKey string `json:"storage_object_key"`
}

type signedArtifactSelectorQuery struct {
	ArtifactType string
	Search       string
	Page         int
	PageSize     int
}

type signedArtifactSelectorPlan struct {
	fromWhere    string
	searchClause string
	orderBy      string
}

func parseSignedArtifactSelectorQuery(values url.Values) (signedArtifactSelectorQuery, error) {
	page := httpx.ParsePageQuery(values, map[string]struct{}{}, nil)
	query := signedArtifactSelectorQuery{
		ArtifactType: strings.ToLower(strings.TrimSpace(values.Get("artifactType"))),
		Search:       strings.TrimSpace(values.Get("q")),
		Page:         page.Page,
		PageSize:     page.PageSize,
	}
	if _, err := signedArtifactSelectorPlanFor(query.ArtifactType); err != nil {
		return signedArtifactSelectorQuery{}, err
	}
	return query, nil
}

func signedArtifactSelectorPlanFor(artifactType string) (signedArtifactSelectorPlan, error) {
	// Legacy School artifact tables predate tenant_code; institution_id is the
	// canonical tenant-owned key and is derived from the authenticated context.
	scope := "institution_id=public.current_institution_id()"
	switch artifactType {
	case "decision":
		return signedArtifactSelectorPlan{`from education_decisions where ` + scope + ` and status <> 'blocked'`, ` and (lower(decision_code) like $1 or lower(title) like $1)`, `lower(title), id`}, nil
	case "publication":
		return signedArtifactSelectorPlan{`from education_publications where ` + scope + ` and publication_status <> 'retras'`, ` and (lower(publication_code) like $1 or lower(entity_label) like $1)`, `lower(entity_label), id`}, nil
	case "managerial_document":
		return signedArtifactSelectorPlan{`from education_managerial_documents where ` + scope + ` and document_status <> 'archived'`, ` and (lower(document_code) like $1 or lower(title) like $1)`, `lower(title), id`}, nil
	case "meeting_document":
		return signedArtifactSelectorPlan{`from education_meeting_documents where ` + scope, ` and (lower(document_number) like $1 or lower(title) like $1)`, `lower(title), id`}, nil
	case "meeting_minute":
		return signedArtifactSelectorPlan{`from education_meeting_minutes where ` + scope, ` and lower(topic_title) like $1`, `lower(topic_title), id`}, nil
	case "meeting_resolution":
		return signedArtifactSelectorPlan{`from education_meeting_resolutions where ` + scope, ` and (lower(resolution_code) like $1 or lower(title) like $1)`, `lower(title), id`}, nil
	default:
		return signedArtifactSelectorPlan{}, errors.New("unsupported signed artifact type")
	}
}

func signedArtifactSelectorProjection(artifactType string) string {
	switch artifactType {
	case "decision":
		return `id::text, concat_ws(' · ', decision_code, title), 'decision'`
	case "publication":
		return `id::text, concat_ws(' · ', publication_code, entity_label), 'publication'`
	case "managerial_document":
		return `id::text, concat_ws(' · ', document_code, title), 'managerial_document'`
	case "meeting_document":
		return `id::text, concat_ws(' · ', nullif(document_number,''), title), 'meeting_document'`
	case "meeting_minute":
		return `id::text, topic_title, 'meeting_minute'`
	case "meeting_resolution":
		return `id::text, concat_ws(' · ', resolution_code, title), 'meeting_resolution'`
	default:
		return ""
	}
}

func (s *Service) EligibleSignedArtifacts(w http.ResponseWriter, r *http.Request) {
	if !s.requireSignedArtifactPermission(w, r, signedArtifactManagePermission) {
		return
	}
	query, err := parseSignedArtifactSelectorQuery(r.URL.Query())
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "education_signature_artifact_type_invalid"})
		return
	}
	plan, _ := signedArtifactSelectorPlanFor(query.ArtifactType)
	where := plan.fromWhere
	args := []any{}
	if query.Search != "" {
		args = append(args, "%"+strings.ToLower(query.Search)+"%")
		where += plan.searchClause
	}
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) "+where, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_signature_eligible_artifacts_failed"})
		return
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf("select %s %s order by %s limit $%d offset $%d", signedArtifactSelectorProjection(query.ArtifactType), where, plan.orderBy, len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_signature_eligible_artifacts_failed"})
		return
	}
	defer rows.Close()
	items := make([]EligibleSignedArtifact, 0, query.PageSize)
	for rows.Next() {
		var item EligibleSignedArtifact
		if err := rows.Scan(&item.ID, &item.Label, &item.Type); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_signature_eligible_artifacts_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_signature_eligible_artifacts_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) EligibleSignatureArchiveVersions(w http.ResponseWriter, r *http.Request) {
	if !s.requireSignedArtifactPermission(w, r, signedArtifactManagePermission) {
		return
	}
	page := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{}, nil)
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	where := `from archive_documents document
		join archive_document_versions version on version.document_id=document.id
		where document.institution_id=public.current_institution_id()
		and version.institution_id=public.current_institution_id()
		and document.status='ready' and version.status='active'
		and version.source_sha256 ~ '^[0-9a-fA-F]{64}$'
		and nullif(btrim(version.source_bucket),'') is not null
		and nullif(btrim(version.source_object_key),'') is not null`
	args := []any{}
	if search != "" {
		args = append(args, "%"+strings.ToLower(search)+"%")
		where += ` and (lower(document.title) like $1 or lower(version.title) like $1 or lower(document.original_file_name) like $1)`
	}
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) "+where, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_signature_eligible_archive_versions_failed"})
		return
	}
	args = append(args, page.PageSize, (page.Page-1)*page.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`select document.id::text,version.id::text,coalesce(nullif(version.title,''),document.title),version.version_no,lower(version.source_sha256),version.source_bucket,version.source_object_key %s order by lower(coalesce(nullif(version.title,''),document.title)),version.version_no desc,version.id limit $%d offset $%d`, where, len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_signature_eligible_archive_versions_failed"})
		return
	}
	defer rows.Close()
	items := make([]EligibleSignatureArchiveVersion, 0, page.PageSize)
	for rows.Next() {
		var item EligibleSignatureArchiveVersion
		if err := rows.Scan(&item.DocumentID, &item.VersionID, &item.Title, &item.VersionNo, &item.DocumentSHA256, &item.StorageBucket, &item.StorageObjectKey); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_signature_eligible_archive_versions_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_signature_eligible_archive_versions_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, page.Page, page.PageSize)
}
