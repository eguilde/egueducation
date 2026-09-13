package education

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	authruntime "github.com/eguilde/egueducation/internal/auth"
)

// RemoteSignedArtifactVerifier delegates cryptographic container, certificate,
// timestamp and EU trusted-list validation to a deployment-owned eIDAS trust
// service. EguEducation sends immutable identifiers and storage provenance;
// the trust service never receives a browser-supplied tenant.
type RemoteSignedArtifactVerifier struct {
	endpoint           string
	capabilityEndpoint string
	token              string
	client             *http.Client
}

const (
	remoteVerifierProtocolVersion = "egueducation-signed-artifact-verifier.v1"
	remoteVerifierMaxResponseBody = 64 << 10
)

type remoteSignedArtifactVerifierCapabilities struct {
	Status          string   `json:"status"`
	ProtocolVersion string   `json:"protocol_version"`
	Capabilities    []string `json:"capabilities"`
}

type remoteSignedArtifactVerificationRequest struct {
	TenantCode    string                 `json:"tenant_code"`
	InstitutionID string                 `json:"institution_id"`
	Evidence      SignedArtifactEvidence `json:"evidence"`
}

type remoteSignedArtifactVerificationResponse struct {
	Status                string         `json:"status"`
	SignatureFormat       string         `json:"signature_format"`
	SignatureLevel        string         `json:"signature_level"`
	SignatureSubject      string         `json:"signature_subject"`
	CertificateIssuer     string         `json:"certificate_issuer"`
	CertificateSerial     string         `json:"certificate_serial"`
	CertificateValidFrom  string         `json:"certificate_valid_from"`
	CertificateValidUntil string         `json:"certificate_valid_until"`
	TrustedListProvider   string         `json:"trusted_list_provider"`
	ValidatorProvider     string         `json:"validator_provider"`
	ValidatorVersion      string         `json:"validator_version"`
	ValidationPolicy      string         `json:"validation_policy"`
	ObservedSHA256        string         `json:"observed_sha256"`
	ObservedSizeBytes     int64          `json:"observed_size_bytes"`
	SignedPayloadSHA256   string         `json:"signed_payload_sha256"`
	CertificateSHA256     string         `json:"certificate_sha256"`
	SignedActorSubject    string         `json:"signed_actor_subject"`
	DiagnosticData        map[string]any `json:"diagnostic_data"`
	DetailedReport        map[string]any `json:"detailed_report"`
	SimpleReport          map[string]any `json:"simple_report"`
	ETSIValidationReport  map[string]any `json:"etsi_validation_report"`
	TimestampTokenSHA256  string         `json:"timestamp_token_sha256"`
	TimestampAt           string         `json:"timestamp_at"`
	TimestampAuthority    string         `json:"timestamp_authority"`
	Findings              map[string]any `json:"findings"`
}

func NewRemoteSignedArtifactVerifier(endpoint, token string, timeout time.Duration) (*RemoteSignedArtifactVerifier, error) {
	parsed, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil || parsed.Host == "" {
		return nil, fmt.Errorf("signature verifier endpoint is invalid")
	}
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && (parsed.Hostname() == "127.0.0.1" || parsed.Hostname() == "localhost")) {
		return nil, fmt.Errorf("signature verifier endpoint must use HTTPS")
	}
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("signature verifier token is required")
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	capabilityURL := *parsed
	capabilityURL.RawQuery = ""
	capabilityURL.Fragment = ""
	capabilityURL.Path = strings.TrimRight(capabilityURL.Path, "/") + "/capabilities"
	return &RemoteSignedArtifactVerifier{
		endpoint:           parsed.String(),
		capabilityEndpoint: capabilityURL.String(),
		token:              token,
		client: &http.Client{
			Timeout: timeout,
			// The bearer token is scoped to the configured trust endpoint. A
			// redirect could silently forward it to another host or downgrade a
			// path, so verification treats every redirect as a rejected response.
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
		},
	}, nil
}

// Ready authenticates to a bounded capability endpoint rather than treating a
// reachable TCP socket (or the deterministic test emulator's /healthz) as proof
// that the deployment can enforce the signed legal-payload contract.
func (v *RemoteSignedArtifactVerifier) Ready(ctx context.Context) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, v.capabilityEndpoint, nil)
	if err != nil {
		return fmt.Errorf("build signature verifier readiness request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+v.token)
	request.Header.Set("Accept", "application/json")
	response, err := v.client.Do(request)
	if err != nil {
		return fmt.Errorf("signature verifier readiness unavailable: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("signature verifier readiness rejected with status %d", response.StatusCode)
	}
	limited := io.LimitReader(response.Body, remoteVerifierMaxResponseBody+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return fmt.Errorf("read signature verifier readiness response: %w", err)
	}
	if len(body) > remoteVerifierMaxResponseBody {
		return fmt.Errorf("signature verifier readiness response exceeds limit")
	}
	var decoded remoteSignedArtifactVerifierCapabilities
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return fmt.Errorf("signature verifier readiness response is invalid")
	}
	if decoded.Status != "ready" || decoded.ProtocolVersion != remoteVerifierProtocolVersion || !containsString(decoded.Capabilities, "signed-payload-sha256") || !containsString(decoded.Capabilities, "leaf-certificate-sha256") || !containsString(decoded.Capabilities, "expected-actor-subject") || !containsString(decoded.Capabilities, "exact-storage-object-version") {
		return fmt.Errorf("signature verifier readiness capabilities are incompatible")
	}
	return nil
}

func (v *RemoteSignedArtifactVerifier) Verify(r *http.Request, evidence SignedArtifactEvidence) SignedArtifactValidationResult {
	return v.verify(r.Context(), authruntime.CurrentTenantCodeFromRequest(r), authruntime.CurrentInstitutionIDFromRequest(r), evidence)
}

func (v *RemoteSignedArtifactVerifier) VerifyForScope(ctx context.Context, tenantCode, institutionID string, evidence SignedArtifactEvidence) SignedArtifactValidationResult {
	return v.verify(ctx, tenantCode, institutionID, evidence)
}

func (v *RemoteSignedArtifactVerifier) verify(ctx context.Context, tenantCode, institutionID string, evidence SignedArtifactEvidence) SignedArtifactValidationResult {
	expectedPayloadHash := strings.ToLower(strings.TrimSpace(evidence.ExpectedCanonicalLegalPayloadSHA256))
	expectedActor := strings.TrimSpace(evidence.ExpectedActorSubject)
	if !isLowerSHA256(expectedPayloadHash) || expectedActor == "" || strings.TrimSpace(evidence.StorageObjectVersionID) == "" {
		return signatureVerificationError("trust_verifier_expected_contract_invalid", 0)
	}
	retentionUntil, retentionErr := time.Parse(time.RFC3339Nano, strings.TrimSpace(evidence.StorageRetentionUntil))
	if retentionErr != nil || !retentionUntil.After(time.Now().UTC()) {
		return signatureVerificationError("trust_verifier_storage_retention_invalid", 0)
	}
	evidence.ExpectedCanonicalLegalPayloadSHA256 = expectedPayloadHash
	evidence.ExpectedActorSubject = expectedActor
	payload, err := json.Marshal(remoteSignedArtifactVerificationRequest{
		TenantCode:    tenantCode,
		InstitutionID: institutionID,
		Evidence:      evidence,
	})
	if err != nil {
		return signatureVerificationError("trust_verifier_request_failed", 0)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, v.endpoint, bytes.NewReader(payload))
	if err != nil {
		return signatureVerificationError("trust_verifier_request_failed", 0)
	}
	request.Header.Set("Authorization", "Bearer "+v.token)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	response, err := v.client.Do(request)
	if err != nil {
		return signatureVerificationError("trust_verifier_unavailable", 0)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return signatureVerificationError("trust_verifier_rejected", response.StatusCode)
	}
	var decoded remoteSignedArtifactVerificationResponse
	decoder := json.NewDecoder(io.LimitReader(response.Body, remoteVerifierMaxResponseBody+1))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return signatureVerificationError("trust_verifier_response_invalid", 0)
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return signatureVerificationError("trust_verifier_response_invalid", 0)
	}
	decoded.Status = strings.ToLower(strings.TrimSpace(decoded.Status))
	if !containsString([]string{"valid", "invalid", "indeterminate", "error"}, decoded.Status) {
		return signatureVerificationError("trust_verifier_status_invalid", 0)
	}
	if decoded.Findings == nil {
		decoded.Findings = map[string]any{}
	}
	// A negative verdict may precede object or certificate extraction. Keep its
	// diagnostic status without requiring fabricated proof fields, and never
	// propagate trust assertions from a response that did not validate.
	if decoded.Status != "valid" {
		return SignedArtifactValidationResult{Status: decoded.Status, Findings: decoded.Findings}
	}
	var timestampAt *time.Time
	if strings.TrimSpace(decoded.TimestampAt) != "" {
		parsedTimestamp, err := time.Parse(time.RFC3339, decoded.TimestampAt)
		if err != nil {
			return signatureVerificationError("trust_verifier_timestamp_invalid", 0)
		}
		timestampAt = &parsedTimestamp
	}
	var validFrom, validUntil *time.Time
	if strings.TrimSpace(decoded.CertificateValidFrom) != "" {
		parsed, err := time.Parse(time.RFC3339, decoded.CertificateValidFrom)
		if err != nil {
			return signatureVerificationError("trust_verifier_certificate_window_invalid", 0)
		}
		validFrom = &parsed
	}
	if strings.TrimSpace(decoded.CertificateValidUntil) != "" {
		parsed, err := time.Parse(time.RFC3339, decoded.CertificateValidUntil)
		if err != nil {
			return signatureVerificationError("trust_verifier_certificate_window_invalid", 0)
		}
		validUntil = &parsed
	}
	timestampHash := strings.ToLower(strings.TrimSpace(decoded.TimestampTokenSHA256))
	if timestampHash != "" && !isLowerSHA256(timestampHash) {
		return signatureVerificationError("trust_verifier_timestamp_hash_invalid", 0)
	}
	observedHash := strings.ToLower(strings.TrimSpace(decoded.ObservedSHA256))
	if observedHash != "" && !isLowerSHA256(observedHash) {
		return signatureVerificationError("trust_verifier_observed_hash_invalid", 0)
	}
	signedPayloadHash := strings.ToLower(strings.TrimSpace(decoded.SignedPayloadSHA256))
	if !isLowerSHA256(signedPayloadHash) {
		return signatureVerificationError("trust_verifier_signed_payload_hash_invalid", 0)
	}
	if signedPayloadHash != expectedPayloadHash {
		return signatureVerificationError("trust_verifier_signed_payload_mismatch", 0)
	}
	certificateHash := strings.ToLower(strings.TrimSpace(decoded.CertificateSHA256))
	if !isLowerSHA256(certificateHash) {
		return signatureVerificationError("trust_verifier_certificate_hash_invalid", 0)
	}
	signedActorSubject := strings.TrimSpace(decoded.SignedActorSubject)
	if signedActorSubject == "" || signedActorSubject != expectedActor {
		return signatureVerificationError("trust_verifier_actor_mismatch", 0)
	}
	return SignedArtifactValidationResult{
		Status:                decoded.Status,
		SignatureFormat:       strings.TrimSpace(decoded.SignatureFormat),
		SignatureLevel:        strings.TrimSpace(decoded.SignatureLevel),
		SignatureSubject:      strings.TrimSpace(decoded.SignatureSubject),
		CertificateIssuer:     strings.TrimSpace(decoded.CertificateIssuer),
		CertificateSerial:     strings.TrimSpace(decoded.CertificateSerial),
		CertificateValidFrom:  validFrom,
		CertificateValidUntil: validUntil,
		TrustedListProvider:   strings.TrimSpace(decoded.TrustedListProvider),
		ValidatorProvider:     strings.TrimSpace(decoded.ValidatorProvider), ValidatorVersion: strings.TrimSpace(decoded.ValidatorVersion), ValidationPolicy: strings.TrimSpace(decoded.ValidationPolicy),
		ObservedSHA256: observedHash, ObservedSizeBytes: decoded.ObservedSizeBytes,
		SignedPayloadSHA256: signedPayloadHash, CertificateSHA256: certificateHash,
		SignedActorSubject: signedActorSubject,
		DiagnosticData:     decoded.DiagnosticData, DetailedReport: decoded.DetailedReport, SimpleReport: decoded.SimpleReport, ETSIValidationReport: decoded.ETSIValidationReport,
		TimestampTokenSHA256: timestampHash,
		TimestampAt:          timestampAt,
		TimestampAuthority:   strings.TrimSpace(decoded.TimestampAuthority),
		Findings:             decoded.Findings,
	}
}

func signatureVerificationError(code string, status int) SignedArtifactValidationResult {
	findings := map[string]any{"code": code}
	if status > 0 {
		findings["http_status"] = status
	}
	return SignedArtifactValidationResult{Status: "error", Findings: findings}
}

func isLowerSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, character := range value {
		if !((character >= '0' && character <= '9') || (character >= 'a' && character <= 'f')) {
			return false
		}
	}
	return true
}
