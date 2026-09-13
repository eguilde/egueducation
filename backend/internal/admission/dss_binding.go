package admission

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/eguilde/egueducation/internal/education"
	"github.com/jackc/pgx/v5"
)

// persistDSSBinding records the verifier's immutable facts and the admission
// bridge in the same transaction as its decision/resolution. Archive identity
// is always loaded from eArhiva, never accepted as browser provenance.
func persistDSSBinding(ctx context.Context, tx pgx.Tx, sc scope, artifactKind, artifactID, policyEvaluationID string, archive ArchiveReference, snap archiveSnapshot, actor string, expectedPayloadSHA256 ...string) error {
	// Variadic only preserves source compatibility while decision/resolution
	// prepare-finalize call sites are introduced. Zero or multiple expectations
	// still fail closed before retention lookup, network access, or persistence.
	if len(expectedPayloadSHA256) != 1 {
		return errInvalidInput
	}
	expectedCanonicalLegalPayloadSHA256 := strings.ToLower(strings.TrimSpace(expectedPayloadSHA256[0]))
	actor = strings.TrimSpace(actor)
	if !sha256Text(expectedCanonicalLegalPayloadSHA256) || actor == "" || actor != strings.TrimSpace(sc.actor) {
		return errInvalidInput
	}
	retentionPolicyID, minimumRetentionDays, err := admissionRetentionPolicy(ctx, tx, sc)
	if err != nil {
		return err
	}
	if snap.retention.Before(time.Now().UTC().AddDate(0, 0, minimumRetentionDays)) {
		return errInvalidInput
	}
	evidence := education.SignedArtifactEvidence{ArtifactType: artifactKind, ArtifactID: artifactID, DocumentSHA256: snap.sha, DocumentSizeBytes: snap.size, ExpectedCanonicalLegalPayloadSHA256: expectedCanonicalLegalPayloadSHA256, ExpectedActorSubject: actor, StorageDocumentID: archive.DocumentID, StorageVersionID: archive.VersionID, StorageBucket: snap.bucket, StorageObjectKey: snap.objectKey, StorageObjectVersionID: snap.objectVersion, StorageRetentionUntil: snap.retention.UTC().Format(time.RFC3339Nano)}
	result := education.VerifyConfiguredSignedArtifact(ctx, sc.tenant, sc.institution, evidence)
	if err := validAdmissionDSSResult(result, snap, expectedCanonicalLegalPayloadSHA256, actor); err != nil {
		return err
	}
	if err := requireAuthorizedAdmissionSigner(ctx, tx, sc, artifactKind, actor, result.CertificateSHA256, result.TimestampAt); err != nil {
		return err
	}
	evidence.SignatureFormat, evidence.SignatureLevel = result.SignatureFormat, result.SignatureLevel
	evidence.SignatureSubject, evidence.CertificateIssuer, evidence.CertificateSerial = result.SignatureSubject, result.CertificateIssuer, result.CertificateSerial
	evidence.CertificateValidFrom, evidence.CertificateValidUntil = result.CertificateValidFrom.Format(time.RFC3339), result.CertificateValidUntil.Format(time.RFC3339)
	var evidenceID string
	err = tx.QueryRow(ctx, `insert into education_signed_artifact_evidence(tenant_code,institution_id,artifact_type,artifact_id,document_sha256,expected_canonical_legal_payload_sha256,expected_actor_subject,signature_format,signature_level,signature_subject,certificate_issuer,certificate_serial,certificate_valid_from,certificate_valid_until,storage_document_id,storage_version_id,storage_bucket,storage_object_key) values($1,$2,$3,$4::uuid,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15::uuid,$16::uuid,$17,$18) returning id::text`, sc.tenant, sc.institution, artifactKind, artifactID, snap.sha, expectedCanonicalLegalPayloadSHA256, actor, result.SignatureFormat, result.SignatureLevel, result.SignatureSubject, result.CertificateIssuer, result.CertificateSerial, result.CertificateValidFrom, result.CertificateValidUntil, archive.DocumentID, archive.VersionID, snap.bucket, snap.objectKey).Scan(&evidenceID)
	if err != nil {
		return err
	}
	diagnostic, err := json.Marshal(result.DiagnosticData)
	if err != nil {
		return err
	}
	detailed, err := json.Marshal(result.DetailedReport)
	if err != nil {
		return err
	}
	simple, err := json.Marshal(result.SimpleReport)
	if err != nil {
		return err
	}
	etsi, err := json.Marshal(result.ETSIValidationReport)
	if err != nil {
		return err
	}
	var validationID string
	err = tx.QueryRow(ctx, `insert into education_signed_artifact_validations(evidence_id,tenant_code,institution_id,validation_status,trusted_list_provider,validator_provider,validator_version,validation_policy,observed_sha256,observed_size_bytes,signed_payload_sha256,certificate_sha256,signed_actor_subject,diagnostic_data,detailed_report,simple_report,etsi_validation_report,timestamp_token_sha256,timestamp_at,timestamp_authority,findings) values($1::uuid,$2,$3,'valid',$4,$5,$6,$7,$8,$9,$10,$11,$12,$13::jsonb,$14::jsonb,$15::jsonb,$16::jsonb,$17,$18,$19,'{}'::jsonb) returning id::text`, evidenceID, sc.tenant, sc.institution, result.TrustedListProvider, result.ValidatorProvider, result.ValidatorVersion, result.ValidationPolicy, result.ObservedSHA256, result.ObservedSizeBytes, result.SignedPayloadSHA256, result.CertificateSHA256, result.SignedActorSubject, string(diagnostic), string(detailed), string(simple), string(etsi), result.TimestampTokenSHA256, result.TimestampAt, result.TimestampAuthority).Scan(&validationID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `insert into school_admission_signed_artifact_bindings(tenant_code,institution_id,artifact_kind,artifact_id,evidence_id,validation_id,policy_evaluation_v2_id,retention_policy_id,canonical_legal_payload_sha256,expected_actor_subject,archive_document_id,archive_version_id,archive_version_no,archive_source_bucket,archive_source_object_key,archive_sha256,archive_size_bytes,bound_by_subject) values($1,$2,$3,$4::uuid,$5::uuid,$6::uuid,$7::uuid,$8::uuid,$9,$10,$11::uuid,$12::uuid,$13,$14,$15,$16,$17,$18)`, sc.tenant, sc.institution, artifactKind, artifactID, evidenceID, validationID, policyEvaluationID, retentionPolicyID, expectedCanonicalLegalPayloadSHA256, actor, archive.DocumentID, archive.VersionID, snap.versionNo, snap.bucket, snap.objectKey, snap.sha, snap.size, actor)
	return err
}

// requireAuthorizedAdmissionSigner joins the fingerprint returned by DSS to the
// tenant-scoped user/subject authorization.  It checks both trusted signing
// time and finalization time, so a revoked/expired credential cannot be used
// merely because it was once valid.
func requireAuthorizedAdmissionSigner(ctx context.Context, tx pgx.Tx, sc scope, artifactKind, actor, certificateSHA256 string, signedAt *time.Time) error {
	permission := permissionDecide
	if artifactKind == "admission_appeal_resolution" {
		permission = permissionAppeals
	}
	if !sha256Text(certificateSHA256) || signedAt == nil {
		return errInvalidInput
	}
	var allowed bool
	err := tx.QueryRow(ctx, `select exists(
  select 1 from school_admission_signer_authorizations a
  join app_users u on u.id=a.user_id
  where a.tenant_code=$1 and a.institution_id=$2
    and a.certificate_sha256=$3 and a.actor_subject=$4 and a.permission_code=$5
    and a.status='active' and a.valid_from<=$6::timestamptz and a.valid_until>$6::timestamptz
    and a.valid_from<=now() and a.valid_until>now() and u.status='active'
)`, sc.tenant, sc.institution, strings.ToLower(certificateSHA256), actor, permission, *signedAt).Scan(&allowed)
	if err != nil {
		return err
	}
	if !allowed {
		return errForbidden
	}
	return nil
}

func admissionRetentionPolicy(ctx context.Context, tx pgx.Tx, sc scope) (string, int, error) {
	var id string
	var days int
	err := tx.QueryRow(ctx, `select p.id::text,p.minimum_retention_days from school_admission_dss_retention_policies p join school_admission_retention_rule_versions rule on rule.tenant_code=p.tenant_code and rule.institution_id=p.institution_id and rule.id=p.rule_version_id and rule.status='active' and rule.artifact_kind='admission_dss' where p.tenant_code=$1 and p.institution_id=$2 and p.status='active' and p.effective_from<=current_date and (p.effective_to is null or p.effective_to>=current_date) order by p.effective_from desc,p.id desc limit 1`, sc.tenant, sc.institution).Scan(&id, &days)
	return id, days, err
}

func validAdmissionDSSResult(result education.SignedArtifactValidationResult, snap archiveSnapshot, expectedCanonicalLegalPayloadSHA256, expectedActorSubject string) error {
	if result.Status != "valid" || result.SignatureFormat != "PAdES" || (result.SignatureLevel != "advanced" && result.SignatureLevel != "qualified") || result.CertificateValidFrom == nil || result.CertificateValidUntil == nil || !result.CertificateValidUntil.After(*result.CertificateValidFrom) || !strings.EqualFold(result.ObservedSHA256, snap.sha) || result.ObservedSizeBytes != snap.size || result.ObservedSizeBytes <= 0 || !sha256Text(result.SignedPayloadSHA256) || !strings.EqualFold(result.SignedPayloadSHA256, expectedCanonicalLegalPayloadSHA256) || !sha256Text(result.CertificateSHA256) || strings.TrimSpace(expectedActorSubject) == "" || result.SignedActorSubject != strings.TrimSpace(expectedActorSubject) || strings.TrimSpace(result.SignatureSubject) == "" || strings.TrimSpace(result.TrustedListProvider) == "" || strings.TrimSpace(result.ValidatorProvider) == "" || strings.TrimSpace(result.ValidatorVersion) == "" || strings.TrimSpace(result.ValidationPolicy) == "" || result.TimestampAt == nil || strings.TrimSpace(result.TimestampAuthority) == "" || !sha256Text(result.TimestampTokenSHA256) || emptyReport(result.DiagnosticData) || emptyReport(result.DetailedReport) || emptyReport(result.SimpleReport) || emptyReport(result.ETSIValidationReport) || strings.TrimSpace(result.CertificateIssuer) == "" || strings.TrimSpace(result.CertificateSerial) == "" {
		return errInvalidInput
	}
	return nil
}

func sha256Text(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, c := range strings.ToLower(value) {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
func emptyReport(value map[string]any) bool { return len(value) == 0 }
