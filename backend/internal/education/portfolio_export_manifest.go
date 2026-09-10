package education

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	authruntime "github.com/eguilde/egueducation/internal/auth"
	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

const portfolioExportManifestVersion = "egueducation.portfolio-export-manifest/v1"

type PortfolioExportManifest struct {
	ManifestVersion string                            `json:"manifest_version"`
	HashAlgorithm   string                            `json:"hash_algorithm"`
	ManifestSHA256  string                            `json:"manifest_sha256"`
	TenantCode      string                            `json:"tenant_code"`
	InstitutionID   string                            `json:"institution_id"`
	Portfolio       PortfolioExportManifestPortfolio  `json:"portfolio"`
	Documents       []PortfolioExportManifestDocument `json:"documents"`
}

type PortfolioExportManifestPortfolio struct {
	ID                 string `json:"id"`
	PortfolioCode      string `json:"portfolio_code"`
	SchoolYear         string `json:"school_year"`
	Status             string `json:"status"`
	AppliedProcedureID string `json:"applied_procedure_id,omitempty"`
}

type PortfolioExportManifestDocument struct {
	EvidenceRecordID  string `json:"evidence_record_id"`
	SectionCode       string `json:"section_code"`
	ComponentCode     string `json:"component_code"`
	DocumentTitle     string `json:"document_title"`
	ChronologicalNo   int    `json:"chronological_no"`
	IssuedOn          string `json:"issued_on"`
	EvidenceType      string `json:"evidence_type"`
	ArchiveDocumentID string `json:"archive_document_id"`
	ArchiveVersionID  string `json:"archive_version_id"`
	ArchiveVersionNo  int    `json:"archive_version_no"`
	SourceBucket      string `json:"source_bucket"`
	SourceObjectKey   string `json:"source_object_key"`
	SourceSHA256      string `json:"source_sha256"`
}

type PortfolioExportManifestResponse struct {
	ExportManifestID string                  `json:"export_manifest_id"`
	GeneratedAt      string                  `json:"generated_at"`
	Manifest         PortfolioExportManifest `json:"manifest"`
}

// PortfolioExportManifestCreate produces an immutable evidence manifest from
// the active server-side tenant context.  It deliberately accepts no browser
// rows, storage locations, hashes, or document identifiers in its body.
func (s *Service) PortfolioExportManifestCreate(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	allowed, err := s.portfolioExportManifestAllowed(r, recordID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_export_authorization_failed"})
		return
	}
	if !allowed {
		writePortfolioAccessFailure(w, nil)
		return
	}
	result, err := s.createPortfolioExportManifestEvidence(r, recordID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_portfolio_not_found")
		return
	}
	if errors.Is(err, errPortfolioExportNotReady) {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "education_portfolio_export_provenance_incomplete"})
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_export_manifest_failed"})
		return
	}

	s.logAudit(r, "education.portfolios.export_manifest.create", "portfolio_export_manifest", result.ExportManifestID, "Authoritative portfolio evidence manifest generated.", map[string]any{"portfolio_id": recordID, "manifest_sha256": result.Manifest.ManifestSHA256, "document_count": len(result.Manifest.Documents)})
	httpx.JSON(w, http.StatusCreated, result)
}

func (s *Service) createPortfolioExportManifestEvidence(r *http.Request, recordID string) (PortfolioExportManifestResponse, error) {
	manifest, err := s.buildPortfolioExportManifest(r, recordID)
	if err != nil {
		return PortfolioExportManifestResponse{}, err
	}
	payload, err := json.Marshal(manifest)
	if err != nil {
		return PortfolioExportManifestResponse{}, err
	}
	var result PortfolioExportManifestResponse
	err = s.pool.QueryRow(r.Context(), `
		insert into education_portfolio_export_manifests (
			portfolio_id, institution_id, tenant_code, manifest_version,
			manifest_sha256, manifest, generated_by_subject
		) values ($1::uuid, $2, $3, $4, $5, $6::jsonb, $7)
		on conflict (portfolio_id, manifest_sha256) do update
			set generated_by_subject = education_portfolio_export_manifests.generated_by_subject
		returning id::text, to_char(generated_at, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
	`, recordID, s.institutionID(r), manifest.TenantCode, manifest.ManifestVersion, manifest.ManifestSHA256, payload, strings.TrimSpace(authruntime.CurrentSubjectFromRequest(r))).Scan(&result.ExportManifestID, &result.GeneratedAt)
	if err != nil {
		return PortfolioExportManifestResponse{}, err
	}
	result.Manifest = manifest
	return result, nil
}

// portfolioExportManifestAllowed preserves institution-reader behaviour while
// making a self-service grant safe: a read_own/export_own holder must also be
// the immutable portfolio owner.  This prevents a route-level read_own grant
// from becoming an IDOR against another teacher's UUID.
func (s *Service) portfolioExportManifestAllowed(r *http.Request, recordID string) (bool, error) {
	subject := strings.TrimSpace(authruntime.CurrentSubjectFromRequest(r))
	if subject == "" {
		return false, nil
	}
	for _, permission := range []string{"education.portfolios.school.read", "education.portfolios.read"} {
		allowed, err := s.currentSubjectHasPermission(r, subject, permission)
		if err != nil || allowed {
			return allowed, err
		}
	}
	selfService := false
	for _, permission := range []string{portfolioReadOwnPermission, "education.portfolios.export_own"} {
		allowed, err := s.currentSubjectHasPermission(r, subject, permission)
		if err != nil {
			return false, err
		}
		selfService = selfService || allowed
	}
	if !selfService {
		return false, nil
	}
	actorID, err := s.currentActorUserID(r, subject)
	if err != nil || actorID == "" {
		return false, err
	}
	var owned bool
	err = s.pool.QueryRow(r.Context(), `
		select exists(select 1 from education_portfolios
			where id=$1::uuid and institution_id=$2 and owner_user_id=$3::uuid and withdrawn_at is null)
	`, recordID, s.institutionID(r), actorID).Scan(&owned)
	return owned, err
}

var errPortfolioExportNotReady = errors.New("portfolio export provenance incomplete")

func (s *Service) buildPortfolioExportManifest(r *http.Request, recordID string) (PortfolioExportManifest, error) {
	var manifest PortfolioExportManifest
	manifest.ManifestVersion = portfolioExportManifestVersion
	manifest.HashAlgorithm = "SHA-256"
	manifest.InstitutionID = s.institutionID(r)
	if err := s.pool.QueryRow(r.Context(), `select public.current_tenant_code()`).Scan(&manifest.TenantCode); err != nil {
		return PortfolioExportManifest{}, err
	}
	if err := s.pool.QueryRow(r.Context(), `
		select id::text, portfolio_code, school_year, status, coalesce(applied_procedure_id::text, '')
		from education_portfolios
		where id=$1::uuid and institution_id=$2 and withdrawn_at is null
	`, recordID, manifest.InstitutionID).Scan(&manifest.Portfolio.ID, &manifest.Portfolio.PortfolioCode, &manifest.Portfolio.SchoolYear, &manifest.Portfolio.Status, &manifest.Portfolio.AppliedProcedureID); err != nil {
		return PortfolioExportManifest{}, err
	}
	if manifest.Portfolio.Status == "draft" || manifest.Portfolio.Status == "returned" {
		return PortfolioExportManifest{}, errPortfolioExportNotReady
	}
	rows, err := s.pool.Query(r.Context(), `
		select id::text, section_code, component_code, document_title, chronological_index,
			to_char(issued_on, 'YYYY-MM-DD'), evidence_type,
			coalesce(archive_document_id::text, ''), coalesce(archive_version_id::text, ''),
			archive_version_no, archive_source_bucket, archive_source_object_key, archive_sha256
		from education_portfolio_documents
		where portfolio_id=$1::uuid and institution_id=$2 and status='active' and source_scope='portofoliu'
		order by chronological_index, issued_on, id
	`, recordID, manifest.InstitutionID)
	if err != nil {
		return PortfolioExportManifest{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var document PortfolioExportManifestDocument
		if err := rows.Scan(&document.EvidenceRecordID, &document.SectionCode, &document.ComponentCode, &document.DocumentTitle, &document.ChronologicalNo, &document.IssuedOn, &document.EvidenceType, &document.ArchiveDocumentID, &document.ArchiveVersionID, &document.ArchiveVersionNo, &document.SourceBucket, &document.SourceObjectKey, &document.SourceSHA256); err != nil {
			return PortfolioExportManifest{}, err
		}
		document.SourceSHA256 = strings.ToLower(document.SourceSHA256)
		if document.ArchiveDocumentID == "" || document.ArchiveVersionID == "" || document.ArchiveVersionNo < 1 || document.SourceBucket == "" || document.SourceObjectKey == "" || !isSHA256Hex(document.SourceSHA256) {
			return PortfolioExportManifest{}, errPortfolioExportNotReady
		}
		manifest.Documents = append(manifest.Documents, document)
	}
	if err := rows.Err(); err != nil {
		return PortfolioExportManifest{}, err
	}
	if len(manifest.Documents) == 0 {
		return PortfolioExportManifest{}, errPortfolioExportNotReady
	}
	manifest.ManifestSHA256 = portfolioExportManifestSHA256(manifest)
	return manifest, nil
}

func portfolioExportManifestSHA256(manifest PortfolioExportManifest) string {
	manifest.ManifestSHA256 = ""
	payload, _ := json.Marshal(manifest)
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:])
}

func isSHA256Hex(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
