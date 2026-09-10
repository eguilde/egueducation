package education

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/eguilde/egueducation/internal/audit"
	authruntime "github.com/eguilde/egueducation/internal/auth"
	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/go-chi/chi/v5"
)

const (
	signedArtifactReadPermission     = "education.signatures.read"
	signedArtifactManagePermission   = "education.signatures.manage"
	signedArtifactValidatePermission = "education.signatures.validate"
)

// SignedArtifactVerifier deliberately receives metadata, never key material.
// A deployment must install an actual trust-list verifier before it can report
// valid evidence; the default is fail-closed.
type SignedArtifactVerifier interface {
	Verify(r *http.Request, evidence SignedArtifactEvidence) SignedArtifactValidationResult
}

type SignedArtifactValidationResult struct {
	Status               string
	TrustedListProvider  string
	TimestampTokenSHA256 string
	TimestampAt          *time.Time
	TimestampAuthority   string
	Findings             map[string]any
}

type failClosedSignedArtifactVerifier struct{}

func (failClosedSignedArtifactVerifier) Verify(_ *http.Request, _ SignedArtifactEvidence) SignedArtifactValidationResult {
	return SignedArtifactValidationResult{Status: "error", Findings: map[string]any{"code": "trust_verifier_not_configured"}}
}

var signedArtifactVerifierRuntime = struct {
	sync.RWMutex
	verifier SignedArtifactVerifier
}{verifier: failClosedSignedArtifactVerifier{}}

// ConfigureSignedArtifactVerifier installs a deployment-owned verifier. Nil
// deliberately restores fail-closed behavior. It is intended for composition
// at server startup; no vendor protocol is encoded in this bounded context.
func ConfigureSignedArtifactVerifier(verifier SignedArtifactVerifier) {
	signedArtifactVerifierRuntime.Lock()
	defer signedArtifactVerifierRuntime.Unlock()
	if verifier == nil {
		verifier = failClosedSignedArtifactVerifier{}
	}
	signedArtifactVerifierRuntime.verifier = verifier
}

func activeSignedArtifactVerifier() SignedArtifactVerifier {
	signedArtifactVerifierRuntime.RLock()
	defer signedArtifactVerifierRuntime.RUnlock()
	return signedArtifactVerifierRuntime.verifier
}

type SignedArtifactEvidence struct {
	ID                    string                    `json:"id"`
	ArtifactType          string                    `json:"artifact_type"`
	ArtifactID            string                    `json:"artifact_id"`
	DocumentSHA256        string                    `json:"document_sha256"`
	SignatureFormat       string                    `json:"signature_format"`
	SignatureLevel        string                    `json:"signature_level"`
	SignatureSubject      string                    `json:"signature_subject"`
	CertificateIssuer     string                    `json:"certificate_issuer"`
	CertificateSerial     string                    `json:"certificate_serial"`
	CertificateValidFrom  string                    `json:"certificate_valid_from"`
	CertificateValidUntil string                    `json:"certificate_valid_until"`
	StorageDocumentID     string                    `json:"storage_document_id,omitempty"`
	StorageVersionID      string                    `json:"storage_version_id,omitempty"`
	StorageBucket         string                    `json:"storage_bucket,omitempty"`
	StorageObjectKey      string                    `json:"storage_object_key,omitempty"`
	SubmittedBySubject    string                    `json:"submitted_by_subject"`
	SubmittedAt           string                    `json:"submitted_at"`
	LatestValidation      *SignedArtifactValidation `json:"latest_validation,omitempty"`
}

type SignedArtifactValidation struct {
	ID                   string         `json:"id"`
	ValidationStatus     string         `json:"validation_status"`
	TrustedListProvider  string         `json:"trusted_list_provider"`
	ValidatedAt          string         `json:"validated_at"`
	TimestampTokenSHA256 string         `json:"timestamp_token_sha256,omitempty"`
	TimestampAt          string         `json:"timestamp_at,omitempty"`
	TimestampAuthority   string         `json:"timestamp_authority,omitempty"`
	Findings             map[string]any `json:"findings"`
	ValidatedBySubject   string         `json:"validated_by_subject"`
}

type SubmitSignedArtifactEvidenceRequest struct {
	ArtifactType          string `json:"artifact_type"`
	ArtifactID            string `json:"artifact_id"`
	SignatureFormat       string `json:"signature_format"`
	SignatureLevel        string `json:"signature_level"`
	SignatureSubject      string `json:"signature_subject"`
	CertificateIssuer     string `json:"certificate_issuer"`
	CertificateSerial     string `json:"certificate_serial"`
	CertificateValidFrom  string `json:"certificate_valid_from"`
	CertificateValidUntil string `json:"certificate_valid_until"`
	StorageDocumentID     string `json:"storage_document_id"`
	StorageVersionID      string `json:"storage_version_id"`
}

func decodeSubmitSignedArtifactEvidenceRequest(body io.Reader) (SubmitSignedArtifactEvidenceRequest, error) {
	var req SubmitSignedArtifactEvidenceRequest
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return SubmitSignedArtifactEvidenceRequest{}, err
	}
	if strings.TrimSpace(req.StorageDocumentID) == "" || strings.TrimSpace(req.StorageVersionID) == "" {
		return SubmitSignedArtifactEvidenceRequest{}, fmt.Errorf("archive document and version are required")
	}
	return req, nil
}

func (s *Service) requireSignedArtifactPermission(w http.ResponseWriter, r *http.Request, permission string) bool {
	if strings.TrimSpace(authruntime.CurrentSubjectFromRequest(r)) == "" {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "education_signature_permission_required", "permission": permission})
		return false
	}
	allowed, err := s.authorizeEducationPermission(r, EducationDelegationScope{PermissionCode: permission, ResourceType: "institution"})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_signature_authorization_failed"})
		return false
	}
	if !allowed {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "education_signature_permission_required", "permission": permission})
		return false
	}
	return true
}

func (s *Service) ListSignedArtifactEvidence(w http.ResponseWriter, r *http.Request) {
	if !s.requireSignedArtifactPermission(w, r, signedArtifactReadPermission) {
		return
	}
	q := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{"submitted_at": {}, "artifact_type": {}, "artifact_id": {}, "signature_format": {}, "signature_level": {}, "signature_subject": {}, "validation_status": {}}, []string{"submitted_at", "artifact_type", "artifact_id", "signature_format", "signature_level", "signature_subject", "validation_status"})
	where := []string{"e.tenant_code=public.current_tenant_code()", "e.institution_id=public.current_institution_id()"}
	args := []any{}
	for _, f := range []struct{ k, c string }{{"submitted_at", "e.submitted_at::text"}, {"artifact_type", "e.artifact_type"}, {"artifact_id", "e.artifact_id::text"}, {"signature_format", "e.signature_format"}, {"signature_level", "e.signature_level"}, {"signature_subject", "e.signature_subject"}, {"validation_status", "coalesce(v.validation_status, '')"}} {
		if value := strings.TrimSpace(q.Filters[f.k]); value != "" {
			args = append(args, "%"+strings.ToLower(value)+"%")
			where = append(where, fmt.Sprintf("lower(%s) like $%d", f.c, len(args)))
		}
	}
	join := " left join lateral (select * from education_signed_artifact_validations validation where validation.evidence_id=e.id order by validation.validated_at desc, validation.id desc limit 1) v on true"
	condition := " where " + strings.Join(where, " and ")
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_signed_artifact_evidence e"+join+condition, args...).Scan(&total); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_signed_evidence_list_failed"})
		return
	}
	sort := map[string]string{"submitted_at": "e.submitted_at", "artifact_type": "e.artifact_type", "artifact_id": "e.artifact_id", "signature_format": "e.signature_format", "signature_level": "e.signature_level", "signature_subject": "e.signature_subject", "validation_status": "coalesce(v.validation_status, '')"}[q.Sort]
	if sort == "" {
		sort = "e.submitted_at"
	}
	dir := "asc"
	if q.Direction == "desc" {
		dir = "desc"
	}
	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := s.pool.Query(r.Context(), signedArtifactEvidenceWithValidationSelect+join+condition+fmt.Sprintf(" order by %s %s, e.id desc limit $%d offset $%d", sort, dir, len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_signed_evidence_list_failed"})
		return
	}
	defer rows.Close()
	items := []SignedArtifactEvidence{}
	for rows.Next() {
		item, err := scanSignedArtifactEvidenceWithValidation(rows)
		if err != nil {
			httpx.JSON(w, 500, map[string]any{"code": "education_signed_evidence_list_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_signed_evidence_list_failed"})
		return
	}
	httpx.WritePage(w, 200, items, total, q.Page, q.PageSize)
}

func (s *Service) SignedArtifactEvidenceDetail(w http.ResponseWriter, r *http.Request) {
	if !s.requireSignedArtifactPermission(w, r, signedArtifactReadPermission) {
		return
	}
	id := strings.TrimSpace(chi.URLParam(r, "evidenceID"))
	row := s.pool.QueryRow(r.Context(), signedArtifactEvidenceWithValidationSelect+" left join lateral (select * from education_signed_artifact_validations validation where validation.evidence_id=e.id order by validation.validated_at desc, validation.id desc limit 1) v on true where e.id=$1::uuid and e.tenant_code=public.current_tenant_code() and e.institution_id=public.current_institution_id()", id)
	item, err := scanSignedArtifactEvidenceWithValidation(row)
	if err != nil {
		writeEducationNotFound(w, "education_signed_evidence_not_found")
		return
	}
	httpx.JSON(w, 200, item)
}

func (s *Service) SubmitSignedArtifactEvidence(w http.ResponseWriter, r *http.Request) {
	if !s.requireSignedArtifactPermission(w, r, signedArtifactManagePermission) {
		return
	}
	req, err := decodeSubmitSignedArtifactEvidenceRequest(r.Body)
	if err != nil {
		httpx.JSON(w, 400, map[string]any{"code": "invalid_education_signed_evidence_payload"})
		return
	}
	from, err := time.Parse(time.RFC3339, req.CertificateValidFrom)
	if err != nil {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_certificate_valid_from"})
		return
	}
	until, err := time.Parse(time.RFC3339, req.CertificateValidUntil)
	if err != nil {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_certificate_valid_until"})
		return
	}
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_signed_evidence_submit_failed"})
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	var item SignedArtifactEvidence
	err = tx.QueryRow(r.Context(), `insert into education_signed_artifact_evidence (tenant_code,institution_id,artifact_type,artifact_id,document_sha256,signature_format,signature_level,signature_subject,certificate_issuer,certificate_serial,certificate_valid_from,certificate_valid_until,storage_document_id,storage_version_id,storage_bucket,storage_object_key)
		select public.current_tenant_code(),public.current_institution_id(),$1,$2::uuid,lower(version.source_sha256),$3,$4,$5,$6,$7,$8,$9,document.id,version.id,version.source_bucket,version.source_object_key
		from archive_documents document join archive_document_versions version on version.document_id=document.id
		where document.id=$10::uuid and version.id=$11::uuid and document.institution_id=public.current_institution_id() and version.institution_id=public.current_institution_id()
			and document.status='ready' and version.status='active' and version.source_sha256 ~ '^[0-9a-fA-F]{64}$' and nullif(btrim(version.source_bucket),'') is not null and nullif(btrim(version.source_object_key),'') is not null
		returning id::text,artifact_type,artifact_id::text,document_sha256,signature_format,signature_level,signature_subject,certificate_issuer,certificate_serial,to_char(certificate_valid_from,'YYYY-MM-DD"T"HH24:MI:SS"Z"'),to_char(certificate_valid_until,'YYYY-MM-DD"T"HH24:MI:SS"Z"'),storage_document_id::text,storage_version_id::text,storage_bucket,storage_object_key,submitted_by_subject,to_char(submitted_at,'YYYY-MM-DD"T"HH24:MI:SS"Z"')`, req.ArtifactType, req.ArtifactID, req.SignatureFormat, req.SignatureLevel, req.SignatureSubject, req.CertificateIssuer, req.CertificateSerial, from, until, req.StorageDocumentID, req.StorageVersionID).Scan(&item.ID, &item.ArtifactType, &item.ArtifactID, &item.DocumentSHA256, &item.SignatureFormat, &item.SignatureLevel, &item.SignatureSubject, &item.CertificateIssuer, &item.CertificateSerial, &item.CertificateValidFrom, &item.CertificateValidUntil, &item.StorageDocumentID, &item.StorageVersionID, &item.StorageBucket, &item.StorageObjectKey, &item.SubmittedBySubject, &item.SubmittedAt)
	if err != nil {
		httpx.JSON(w, 422, map[string]any{"code": "education_signed_evidence_rejected"})
		return
	}
	if _, err := tx.Exec(r.Context(), `insert into education_signed_artifact_validations(evidence_id,tenant_code,institution_id,validation_status,findings) values($1::uuid,public.current_tenant_code(),public.current_institution_id(),'pending','{"code":"verification_pending"}'::jsonb)`, item.ID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_signed_evidence_submit_failed"})
		return
	}
	if err := logSignedArtifactAuditTx(r, tx, "education.signatures.submit", item.ID, "Signed artifact evidence submitted.", map[string]any{"artifact_type": item.ArtifactType, "artifact_id": item.ArtifactID}); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_signed_evidence_submit_failed"})
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_signed_evidence_submit_failed"})
		return
	}
	httpx.JSON(w, 201, item)
}

func (s *Service) RevalidateSignedArtifactEvidence(w http.ResponseWriter, r *http.Request) {
	if !s.requireSignedArtifactPermission(w, r, signedArtifactValidatePermission) {
		return
	}
	id := strings.TrimSpace(chi.URLParam(r, "evidenceID"))
	item, err := s.loadSignedArtifactEvidence(r, id)
	if err != nil {
		writeEducationNotFound(w, "education_signed_evidence_not_found")
		return
	}
	result := activeSignedArtifactVerifier().Verify(r, item)
	if result.Status == "valid" && (item.SignatureLevel == "qualified" && strings.TrimSpace(result.TrustedListProvider) == "") {
		result.Status = "error"
		result.Findings = map[string]any{"code": "qualified_signature_without_trust_validation"}
	}
	findings, err := json.Marshal(result.Findings)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_signature_revalidation_failed"})
		return
	}
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_signature_revalidation_failed"})
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	var validation SignedArtifactValidation
	err = tx.QueryRow(r.Context(), `insert into education_signed_artifact_validations(evidence_id,tenant_code,institution_id,validation_status,trusted_list_provider,timestamp_token_sha256,timestamp_at,timestamp_authority,findings) values($1::uuid,public.current_tenant_code(),public.current_institution_id(),$2,$3,lower($4),$5,$6,$7::jsonb) returning id::text,validation_status,trusted_list_provider,to_char(validated_at,'YYYY-MM-DD"T"HH24:MI:SS"Z"'),timestamp_token_sha256,coalesce(to_char(timestamp_at,'YYYY-MM-DD"T"HH24:MI:SS"Z"'),''),timestamp_authority,findings,validated_by_subject`, id, result.Status, result.TrustedListProvider, result.TimestampTokenSHA256, result.TimestampAt, result.TimestampAuthority, string(findings)).Scan(&validation.ID, &validation.ValidationStatus, &validation.TrustedListProvider, &validation.ValidatedAt, &validation.TimestampTokenSHA256, &validation.TimestampAt, &validation.TimestampAuthority, &validation.Findings, &validation.ValidatedBySubject)
	if err != nil {
		httpx.JSON(w, 422, map[string]any{"code": "education_signature_revalidation_rejected"})
		return
	}
	if err := logSignedArtifactAuditTx(r, tx, "education.signatures.revalidate", id, "Signed artifact evidence revalidated.", map[string]any{"status": validation.ValidationStatus}); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_signature_revalidation_failed"})
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_signature_revalidation_failed"})
		return
	}
	httpx.JSON(w, 201, validation)
}

// logSignedArtifactAuditTx keeps the legal evidence mutation and its audit
// assertion in one database transaction. A missing/failed audit is a failed
// legal operation, not a best-effort telemetry concern.
func logSignedArtifactAuditTx(r *http.Request, tx audit.DB, action, evidenceID, summary string, details map[string]any) error {
	return audit.Log(r.Context(), tx, audit.Event{
		ActorSubject: authruntime.CurrentSubjectFromRequest(r),
		Action:       action,
		TargetType:   "education_signed_artifact_evidence",
		TargetID:     evidenceID,
		Summary:      summary,
		Details:      details,
	})
}

const signedArtifactEvidenceSelect = `select e.id::text,e.artifact_type,e.artifact_id::text,e.document_sha256,e.signature_format,e.signature_level,e.signature_subject,e.certificate_issuer,e.certificate_serial,to_char(e.certificate_valid_from,'YYYY-MM-DD"T"HH24:MI:SS"Z"'),to_char(e.certificate_valid_until,'YYYY-MM-DD"T"HH24:MI:SS"Z"'),coalesce(e.storage_document_id::text,''),coalesce(e.storage_version_id::text,''),e.storage_bucket,e.storage_object_key,e.submitted_by_subject,to_char(e.submitted_at,'YYYY-MM-DD"T"HH24:MI:SS"Z"') from education_signed_artifact_evidence e`
const signedArtifactEvidenceWithValidationSelect = `select e.id::text,e.artifact_type,e.artifact_id::text,e.document_sha256,e.signature_format,e.signature_level,e.signature_subject,e.certificate_issuer,e.certificate_serial,to_char(e.certificate_valid_from,'YYYY-MM-DD"T"HH24:MI:SS"Z"'),to_char(e.certificate_valid_until,'YYYY-MM-DD"T"HH24:MI:SS"Z"'),coalesce(e.storage_document_id::text,''),coalesce(e.storage_version_id::text,''),e.storage_bucket,e.storage_object_key,e.submitted_by_subject,to_char(e.submitted_at,'YYYY-MM-DD"T"HH24:MI:SS"Z"'),coalesce(v.id::text,''),coalesce(v.validation_status,''),coalesce(v.trusted_list_provider,''),coalesce(to_char(v.validated_at,'YYYY-MM-DD"T"HH24:MI:SS"Z"'),''),coalesce(v.timestamp_token_sha256,''),coalesce(to_char(v.timestamp_at,'YYYY-MM-DD"T"HH24:MI:SS"Z"'),''),coalesce(v.timestamp_authority,''),coalesce(v.findings,'{}'::jsonb),coalesce(v.validated_by_subject,'') from education_signed_artifact_evidence e`

type signedArtifactRow interface{ Scan(...any) error }

func scanSignedArtifactEvidence(row signedArtifactRow) (SignedArtifactEvidence, error) {
	var x SignedArtifactEvidence
	err := row.Scan(&x.ID, &x.ArtifactType, &x.ArtifactID, &x.DocumentSHA256, &x.SignatureFormat, &x.SignatureLevel, &x.SignatureSubject, &x.CertificateIssuer, &x.CertificateSerial, &x.CertificateValidFrom, &x.CertificateValidUntil, &x.StorageDocumentID, &x.StorageVersionID, &x.StorageBucket, &x.StorageObjectKey, &x.SubmittedBySubject, &x.SubmittedAt)
	return x, err
}

func scanSignedArtifactEvidenceWithValidation(row signedArtifactRow) (SignedArtifactEvidence, error) {
	var evidence SignedArtifactEvidence
	var validation SignedArtifactValidation
	err := row.Scan(
		&evidence.ID, &evidence.ArtifactType, &evidence.ArtifactID, &evidence.DocumentSHA256,
		&evidence.SignatureFormat, &evidence.SignatureLevel, &evidence.SignatureSubject,
		&evidence.CertificateIssuer, &evidence.CertificateSerial, &evidence.CertificateValidFrom,
		&evidence.CertificateValidUntil, &evidence.StorageDocumentID, &evidence.StorageVersionID,
		&evidence.StorageBucket, &evidence.StorageObjectKey, &evidence.SubmittedBySubject,
		&evidence.SubmittedAt, &validation.ID, &validation.ValidationStatus,
		&validation.TrustedListProvider, &validation.ValidatedAt, &validation.TimestampTokenSHA256,
		&validation.TimestampAt, &validation.TimestampAuthority, &validation.Findings,
		&validation.ValidatedBySubject,
	)
	if err == nil && validation.ID != "" {
		evidence.LatestValidation = &validation
	}
	return evidence, err
}
func (s *Service) loadSignedArtifactEvidence(r *http.Request, id string) (SignedArtifactEvidence, error) {
	return scanSignedArtifactEvidence(s.pool.QueryRow(r.Context(), signedArtifactEvidenceSelect+" where e.id=$1::uuid and e.tenant_code=public.current_tenant_code() and e.institution_id=public.current_institution_id()", id))
}
