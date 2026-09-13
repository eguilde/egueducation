package education

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/eguilde/egueducation/internal/audit"
	authruntime "github.com/eguilde/egueducation/internal/auth"
	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

const portfolioExportMaxEvidenceBytes int64 = 100 << 20

var (
	errPortfolioExportEvidenceUnavailable = errors.New("portfolio export evidence unavailable")
	errPortfolioExportLimitExceeded       = errors.New("portfolio export size limit exceeded")
)

type portfolioExportEvidence struct {
	EvidenceRecordID string
	DocumentTitle    string
	MimeType         string
	ObjectKey        string
	ObjectVersionID  string
	SHA256           string
	SizeBytes        int64
}

// PortfolioOwnExport stages a complete interoperable portfolio ZIP before
// sending headers. A missing/revoked/changed evidence item therefore never
// produces a partial archive or a manifest-only download.
func (s *Service) PortfolioOwnExport(w http.ResponseWriter, r *http.Request) {
	if r.Body != nil {
		body, err := io.ReadAll(io.LimitReader(r.Body, 2))
		if err != nil || len(body) != 0 {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "education_portfolio_export_body_not_allowed"})
			return
		}
	}
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	actorID, allowed, err := s.ownPortfolioExportActorID(r)
	if err != nil || !allowed {
		writePortfolioAccessFailure(w, err)
		return
	}
	if s.portfolioArchiveReader == nil {
		httpx.JSON(w, http.StatusServiceUnavailable, map[string]any{"code": "education_portfolio_export_storage_unavailable"})
		return
	}
	portfolio, err := s.loadOwnPortfolio(r, recordID, actorID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_own_portfolio_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_export_failed"})
		return
	}

	manifest, err := s.buildPortfolioExportManifest(r, recordID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_own_portfolio_not_found")
		return
	}
	if errors.Is(err, errPortfolioExportNotReady) {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "education_portfolio_export_provenance_incomplete"})
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_export_failed"})
		return
	}
	if manifest.Portfolio.ID != portfolio.ID {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "education_portfolio_export_provenance_incomplete"})
		return
	}

	opis, err := s.portfolioExportOpis(r, portfolio.ID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_export_failed"})
		return
	}
	manifest, err = portfolioExportManifestWithGeneratedFiles(manifest, portfolio, opis)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_export_failed"})
		return
	}
	snapshotSHA256 := portfolioExportSnapshotSHA256(portfolio, opis)
	staged, err := s.stagePortfolioOwnExport(r, portfolio, actorID, manifest, opis)
	if errors.Is(err, errPortfolioExportEvidenceUnavailable) {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "education_portfolio_export_provenance_incomplete"})
		return
	}
	if errors.Is(err, errPortfolioExportLimitExceeded) {
		httpx.JSON(w, http.StatusRequestEntityTooLarge, map[string]any{"code": "education_portfolio_export_bundle_limit_exceeded"})
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusServiceUnavailable, map[string]any{"code": "education_portfolio_export_storage_unavailable"})
		return
	}
	defer os.Remove(staged.Name())
	defer staged.Close()

	if _, err := staged.Seek(0, io.SeekStart); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_export_failed"})
		return
	}
	info, err := staged.Stat()
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_export_failed"})
		return
	}
	_, err = s.finalizePortfolioOwnExport(r, recordID, actorID, manifest, snapshotSHA256)
	if errors.Is(err, errPortfolioExportEvidenceUnavailable) {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "education_portfolio_export_provenance_incomplete"})
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_export_manifest_failed"})
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="portfolio-%s.zip"`, portfolioExportHeaderName(portfolio.PortfolioCode)))
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, staged)
}

func (s *Service) finalizePortfolioOwnExport(r *http.Request, recordID, actorID string, staged PortfolioExportManifest, stagedSnapshotSHA256 string) (PortfolioExportManifestResponse, error) {
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		return PortfolioExportManifestResponse{}, err
	}
	defer tx.Rollback(r.Context())
	subject := strings.TrimSpace(authruntime.CurrentSubjectFromRequest(r))
	if subject == "" {
		return PortfolioExportManifestResponse{}, errPortfolioExportEvidenceUnavailable
	}
	allowed, err := educationSubjectHasPermission(r.Context(), tx, subject, portfolioReadOwnPermission)
	if err != nil || !allowed {
		allowed, err = educationSubjectHasPermission(r.Context(), tx, subject, "education.portfolios.export_own")
		if err != nil || !allowed {
			return PortfolioExportManifestResponse{}, errPortfolioExportEvidenceUnavailable
		}
	}
	var portfolioID string
	err = tx.QueryRow(r.Context(), `select portfolio.id::text from education_portfolios portfolio join education_personnel personnel on personnel.id=portfolio.owner_personnel_id and personnel.institution_id=portfolio.institution_id where portfolio.id=$1::uuid and portfolio.institution_id=$2 and portfolio.owner_user_id=$3::uuid and personnel.app_user_id=$3::uuid for share`, recordID, s.institutionID(r), actorID).Scan(&portfolioID)
	if err != nil {
		return PortfolioExportManifestResponse{}, errPortfolioExportEvidenceUnavailable
	}
	fresh, err := s.buildPortfolioExportManifestWithQuerier(r, recordID, tx)
	if err != nil || len(fresh.Documents) != len(staged.Documents) {
		return PortfolioExportManifestResponse{}, errPortfolioExportEvidenceUnavailable
	}
	for _, document := range staged.Documents {
		var evidenceID string
		if err := tx.QueryRow(r.Context(), `
			select document.id::text
			from education_portfolio_documents document
			join archive_documents archive_document on archive_document.id=document.archive_document_id and archive_document.institution_id=document.institution_id
			join archive_document_versions version on version.id=document.archive_version_id and version.document_id=archive_document.id and version.institution_id=archive_document.institution_id
			join education_portfolio_archive_attachment_grants attachment_grant on attachment_grant.archive_document_id=archive_document.id and attachment_grant.institution_id=archive_document.institution_id and attachment_grant.grantee_user_id=$3::uuid
			where document.portfolio_id=$1::uuid and document.institution_id=$2 and document.id=$4::uuid
				and document.status='active' and document.source_scope='portofoliu'
				and document.archive_document_id=$5::uuid and document.archive_version_id=$6::uuid and document.archive_version_no=$7
				and archive_document.status='ready' and version.status='active' and version.version_no=document.archive_version_no
				and document.archive_source_bucket=version.source_bucket and document.archive_source_object_key=version.source_object_key and lower(document.archive_sha256)=lower(version.source_sha256)
				and version.source_bucket=$8 and version.source_object_key=$9 and version.source_object_version_id=$10 and lower(version.source_sha256)=lower($11) and version.source_size_bytes=$12 and version.mime_type=$13
			for share of document, archive_document, version, attachment_grant
		`, recordID, s.institutionID(r), actorID, document.EvidenceRecordID, document.ArchiveDocumentID, document.ArchiveVersionID, document.ArchiveVersionNo, document.SourceBucket, document.SourceObjectKey, document.SourceObjectVersionID, document.SourceSHA256, document.SourceSizeBytes, document.MimeType).Scan(&evidenceID); err != nil {
			return PortfolioExportManifestResponse{}, errPortfolioExportEvidenceUnavailable
		}
	}
	var freshPortfolio PortfolioRecord
	if err := scanPortfolioRecord(tx.QueryRow(r.Context(), `select `+portfolioRecordColumns+` from education_portfolios where id=$1::uuid and institution_id=$2 and owner_user_id=$3::uuid for share`, recordID, s.institutionID(r), actorID), &freshPortfolio); err != nil {
		return PortfolioExportManifestResponse{}, errPortfolioExportEvidenceUnavailable
	}
	freshOpis, err := s.portfolioExportOpisWithQuerier(r, recordID, tx)
	if err != nil || portfolioExportSnapshotSHA256(freshPortfolio, freshOpis) != stagedSnapshotSHA256 {
		return PortfolioExportManifestResponse{}, errPortfolioExportEvidenceUnavailable
	}
	fresh, err = portfolioExportManifestWithGeneratedFiles(fresh, freshPortfolio, freshOpis)
	if err != nil || fresh.ManifestSHA256 != staged.ManifestSHA256 || len(fresh.GeneratedFiles) != len(staged.GeneratedFiles) {
		return PortfolioExportManifestResponse{}, errPortfolioExportEvidenceUnavailable
	}
	payload, err := json.Marshal(staged)
	if err != nil {
		return PortfolioExportManifestResponse{}, err
	}
	var result PortfolioExportManifestResponse
	err = tx.QueryRow(r.Context(), `insert into education_portfolio_export_manifests (portfolio_id,institution_id,tenant_code,manifest_version,manifest_sha256,manifest,generated_by_subject) values ($1::uuid,$2,$3,$4,$5,$6::jsonb,$7) on conflict (portfolio_id,manifest_sha256) do update set generated_by_subject=education_portfolio_export_manifests.generated_by_subject returning id::text,to_char(generated_at,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')`, recordID, s.institutionID(r), staged.TenantCode, staged.ManifestVersion, staged.ManifestSHA256, payload, strings.TrimSpace(authruntime.CurrentSubjectFromRequest(r))).Scan(&result.ExportManifestID, &result.GeneratedAt)
	if err != nil {
		return PortfolioExportManifestResponse{}, err
	}
	if err := audit.Log(r.Context(), tx, audit.Event{ActorSubject: subject, Action: "education.portfolios.own_export", TargetType: "portfolio_export_manifest", TargetID: result.ExportManifestID, Summary: "Teacher downloaded an authoritative portfolio bundle.", Details: map[string]any{"portfolio_id": recordID, "manifest_sha256": staged.ManifestSHA256, "document_count": len(staged.Documents)}}); err != nil {
		return PortfolioExportManifestResponse{}, err
	}
	if err := tx.Commit(r.Context()); err != nil {
		return PortfolioExportManifestResponse{}, err
	}
	result.Manifest = staged
	return result, nil
}

func (s *Service) ownPortfolioExportActorID(r *http.Request) (string, bool, error) {
	subject := strings.TrimSpace(authruntime.CurrentSubjectFromRequest(r))
	if subject == "" {
		return "", false, nil
	}
	for _, permission := range []string{portfolioReadOwnPermission, "education.portfolios.export_own"} {
		allowed, err := s.currentSubjectHasPermission(r, subject, permission)
		if err != nil {
			return "", false, err
		}
		if allowed {
			actorID, err := s.currentActorUserID(r, subject)
			return actorID, err == nil && actorID != "", err
		}
	}
	return "", false, nil
}

func (s *Service) stagePortfolioOwnExport(r *http.Request, portfolio PortfolioRecord, actorID string, manifest PortfolioExportManifest, opis []PortfolioOpisEntry) (*os.File, error) {
	if err := s.revalidatePortfolioExportEvidence(r, portfolio.ID, actorID, manifest); err != nil {
		return nil, errPortfolioExportEvidenceUnavailable
	}
	generated, err := portfolioExportGeneratedPayloads(portfolio, opis)
	if err != nil {
		return nil, err
	}
	if !portfolioExportGeneratedFilesMatch(manifest.GeneratedFiles, generated) || portfolioExportManifestSHA256(manifest) != manifest.ManifestSHA256 {
		return nil, errPortfolioExportEvidenceUnavailable
	}
	file, err := os.CreateTemp("", "egueducation-portfolio-export-*.zip")
	if err != nil {
		return nil, fmt.Errorf("create staged portfolio export: %w", err)
	}
	failed := true
	defer func() {
		if failed {
			_ = file.Close()
			_ = os.Remove(file.Name())
		}
	}()
	writer := zip.NewWriter(file)
	for _, item := range generated {
		if err := writePortfolioExportBytes(writer, item.ZIPPath, item.Payload); err != nil {
			return nil, err
		}
	}
	if err := writePortfolioExportJSON(writer, "manifest.json", manifest); err != nil {
		return nil, err
	}
	canonicalManifest := manifest
	canonicalManifest.ManifestSHA256 = ""
	canonicalPayload, err := json.Marshal(canonicalManifest)
	if err != nil {
		return nil, err
	}
	canonicalDigest := sha256.Sum256(canonicalPayload)
	if hex.EncodeToString(canonicalDigest[:]) != manifest.ManifestSHA256 {
		return nil, errPortfolioExportEvidenceUnavailable
	}
	if err := writePortfolioExportText(writer, "manifest.canonical.json", string(canonicalPayload)); err != nil {
		return nil, err
	}
	if err := writePortfolioExportText(writer, "manifest.sha256", manifest.ManifestSHA256+"  manifest.canonical.json\n"); err != nil {
		return nil, err
	}
	var total int64
	for _, document := range manifest.Documents {
		evidence, err := s.loadPortfolioExportEvidence(r, portfolio.ID, actorID, document)
		if err != nil {
			return nil, errPortfolioExportEvidenceUnavailable
		}
		if evidence.SizeBytes > portfolioExportMaxEvidenceBytes || evidence.SizeBytes > s.portfolioExportMaxBytes-total {
			return nil, errPortfolioExportLimitExceeded
		}
		total += evidence.SizeBytes
		entry, err := writer.Create(document.ZIPPath)
		if err != nil {
			return nil, err
		}
		body, err := s.portfolioArchiveReader.OpenObjectVersion(r.Context(), evidence.ObjectKey, evidence.ObjectVersionID)
		if err != nil {
			return nil, err
		}
		digest := sha256.New()
		written, copyErr := io.Copy(io.MultiWriter(entry, digest), io.LimitReader(body, evidence.SizeBytes+1))
		closeErr := body.Close()
		if copyErr != nil || closeErr != nil || written != evidence.SizeBytes || hex.EncodeToString(digest.Sum(nil)) != evidence.SHA256 {
			return nil, errPortfolioExportEvidenceUnavailable
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	if err := file.Sync(); err != nil {
		return nil, err
	}
	failed = false
	return file, nil
}

func (s *Service) revalidatePortfolioExportEvidence(r *http.Request, recordID, actorID string, manifest PortfolioExportManifest) error {
	portfolio, err := s.loadOwnPortfolio(r, recordID, actorID)
	if err != nil || portfolio.ID != recordID {
		return errPortfolioExportEvidenceUnavailable
	}
	for _, document := range manifest.Documents {
		if _, err := s.loadPortfolioExportEvidence(r, recordID, actorID, document); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) loadPortfolioExportEvidence(r *http.Request, recordID, actorID string, expected PortfolioExportManifestDocument) (portfolioExportEvidence, error) {
	var item portfolioExportEvidence
	err := s.pool.QueryRow(r.Context(), `
		select document.id::text, document.document_title, version.mime_type, version.source_object_key,
			version.source_object_version_id, lower(version.source_sha256), version.source_size_bytes
		from education_portfolios portfolio
		join education_portfolio_documents document on document.portfolio_id=portfolio.id and document.institution_id=portfolio.institution_id
		join archive_documents archive_document on archive_document.id=document.archive_document_id and archive_document.institution_id=document.institution_id
		join archive_document_versions version on version.id=document.archive_version_id and version.document_id=archive_document.id and version.institution_id=document.institution_id
		join education_portfolio_archive_attachment_grants attachment_grant on attachment_grant.archive_document_id=archive_document.id and attachment_grant.institution_id=archive_document.institution_id and attachment_grant.grantee_user_id=$3::uuid
		where portfolio.id=$1::uuid and portfolio.institution_id=$2 and portfolio.owner_user_id=$3::uuid
			and document.id=$4::uuid and document.status='active' and document.source_scope='portofoliu'
			and archive_document.status='ready' and version.status='active' and version.version_no=document.archive_version_no
			and document.archive_document_id=$5::uuid and document.archive_version_id=$6::uuid and document.archive_version_no=$7
			and document.archive_source_bucket=version.source_bucket and document.archive_source_object_key=version.source_object_key and lower(document.archive_sha256)=lower(version.source_sha256)
			and version.source_bucket=$8 and version.source_object_key=$9 and version.source_object_version_id=$10 and lower(version.source_sha256)=lower($11) and version.source_size_bytes=$12 and version.mime_type=$13
	`, recordID, s.institutionID(r), actorID, expected.EvidenceRecordID, expected.ArchiveDocumentID, expected.ArchiveVersionID, expected.ArchiveVersionNo, expected.SourceBucket, expected.SourceObjectKey, expected.SourceObjectVersionID, expected.SourceSHA256, expected.SourceSizeBytes, expected.MimeType).Scan(&item.EvidenceRecordID, &item.DocumentTitle, &item.MimeType, &item.ObjectKey, &item.ObjectVersionID, &item.SHA256, &item.SizeBytes)
	if err != nil {
		return portfolioExportEvidence{}, fmt.Errorf("%w: exact evidence query: %v", errPortfolioExportEvidenceUnavailable, err)
	}
	if item.SizeBytes < 1 || item.ObjectVersionID == "" || item.ObjectKey == "" || item.MimeType == "" || s.portfolioArchiveReader.Bucket() != expected.SourceBucket {
		return portfolioExportEvidence{}, errPortfolioExportEvidenceUnavailable
	}
	return item, nil
}

func (s *Service) portfolioExportOpis(r *http.Request, recordID string) ([]PortfolioOpisEntry, error) {
	return s.portfolioExportOpisWithQuerier(r, recordID, s.pool)
}

type portfolioOpisQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func (s *Service) portfolioExportOpisWithQuerier(r *http.Request, recordID string, queryer portfolioOpisQuerier) ([]PortfolioOpisEntry, error) {
	rows, err := queryer.Query(r.Context(), `select id::text, portfolio_id::text, section_code, component_code, entry_title, source_scope, chronological_index, document_reference, included_in_transfer, to_char(checked_on,'YYYY-MM-DD'), checked_by, institution_id, notes from education_portfolio_opis where portfolio_id=$1::uuid and institution_id=$2 order by chronological_index, checked_on, id`, recordID, s.institutionID(r))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []PortfolioOpisEntry{}
	for rows.Next() {
		var item PortfolioOpisEntry
		if err := rows.Scan(&item.ID, &item.PortfolioID, &item.SectionCode, &item.ComponentCode, &item.EntryTitle, &item.SourceScope, &item.ChronologicalIndex, &item.DocumentReference, &item.IncludedInTransfer, &item.CheckedOn, &item.CheckedBy, &item.InstitutionID, &item.Notes); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func portfolioExportSnapshotSHA256(portfolio PortfolioRecord, opis []PortfolioOpisEntry) string {
	payload, _ := json.Marshal(struct {
		Portfolio PortfolioRecord      `json:"portfolio"`
		Opis      []PortfolioOpisEntry `json:"opis"`
	}{Portfolio: portfolio, Opis: opis})
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:])
}

type portfolioExportGeneratedPayload struct {
	ZIPPath string
	Payload []byte
}

func portfolioExportManifestWithGeneratedFiles(manifest PortfolioExportManifest, portfolio PortfolioRecord, opis []PortfolioOpisEntry) (PortfolioExportManifest, error) {
	generated, err := portfolioExportGeneratedPayloads(portfolio, opis)
	if err != nil {
		return PortfolioExportManifest{}, err
	}
	manifest.GeneratedFiles = make([]PortfolioExportManifestGeneratedFile, 0, len(generated))
	for _, item := range generated {
		digest := sha256.Sum256(item.Payload)
		manifest.GeneratedFiles = append(manifest.GeneratedFiles, PortfolioExportManifestGeneratedFile{
			ZIPPath: item.ZIPPath, SHA256: hex.EncodeToString(digest[:]), SizeBytes: int64(len(item.Payload)),
		})
	}
	manifest.ManifestSHA256 = portfolioExportManifestSHA256(manifest)
	return manifest, nil
}

func portfolioExportGeneratedPayloads(portfolio PortfolioRecord, opis []PortfolioOpisEntry) ([]portfolioExportGeneratedPayload, error) {
	portfolioJSON, err := portfolioExportJSONBytes(portfolio)
	if err != nil {
		return nil, err
	}
	opisJSON, err := portfolioExportJSONBytes(opis)
	if err != nil {
		return nil, err
	}
	return []portfolioExportGeneratedPayload{
		{ZIPPath: "portfolio.json", Payload: portfolioJSON},
		{ZIPPath: "opis.json", Payload: opisJSON},
		{ZIPPath: "opis.txt", Payload: []byte(portfolioExportOpisText(opis))},
	}, nil
}

func portfolioExportGeneratedFilesMatch(expected []PortfolioExportManifestGeneratedFile, generated []portfolioExportGeneratedPayload) bool {
	if len(expected) != len(generated) {
		return false
	}
	for index, item := range generated {
		digest := sha256.Sum256(item.Payload)
		if expected[index].ZIPPath != item.ZIPPath ||
			expected[index].SizeBytes != int64(len(item.Payload)) ||
			expected[index].SHA256 != hex.EncodeToString(digest[:]) {
			return false
		}
	}
	return true
}

func portfolioExportJSONBytes(value any) ([]byte, error) {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(payload, '\n'), nil
}

func writePortfolioExportJSON(writer *zip.Writer, name string, value any) error {
	payload, err := portfolioExportJSONBytes(value)
	if err != nil {
		return err
	}
	return writePortfolioExportBytes(writer, name, payload)
}
func writePortfolioExportText(writer *zip.Writer, name, value string) error {
	return writePortfolioExportBytes(writer, name, []byte(value))
}
func writePortfolioExportBytes(writer *zip.Writer, name string, value []byte) error {
	entry, err := writer.Create(name)
	if err != nil {
		return err
	}
	_, err = entry.Write(value)
	return err
}
func portfolioExportOpisText(items []PortfolioOpisEntry) string {
	if len(items) == 0 {
		return "OPIS\n\nNu exista intrari OPIS persistate.\n"
	}
	var builder strings.Builder
	builder.WriteString("OPIS\n\n")
	for _, item := range items {
		fmt.Fprintf(&builder, "%03d | %s | %s | %s | %s\n", item.ChronologicalIndex, item.CheckedOn, item.SectionCode, item.ComponentCode, item.EntryTitle)
	}
	return builder.String()
}
func portfolioExportExtension(mime string) string {
	switch strings.ToLower(strings.TrimSpace(mime)) {
	case "application/pdf":
		return "pdf"
	case "image/jpeg":
		return "jpg"
	case "image/png":
		return "png"
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return "docx"
	default:
		return "bin"
	}
}

func portfolioExportEvidenceZipPath(index int, mime string) string {
	return fmt.Sprintf("evidence/%03d-evidence.%s", index, portfolioExportExtension(mime))
}

func portfolioExportHeaderName(value string) string {
	var builder strings.Builder
	for _, runeValue := range value {
		if (runeValue >= 'a' && runeValue <= 'z') || (runeValue >= 'A' && runeValue <= 'Z') || (runeValue >= '0' && runeValue <= '9') || runeValue == '_' || runeValue == '-' {
			builder.WriteRune(runeValue)
		}
		if builder.Len() == 64 {
			break
		}
	}
	if builder.Len() == 0 {
		return "portfolio"
	}
	return builder.String()
}
