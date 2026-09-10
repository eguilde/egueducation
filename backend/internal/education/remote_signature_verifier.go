package education

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	endpoint string
	token    string
	client   *http.Client
}

type remoteSignedArtifactVerificationRequest struct {
	TenantCode    string                 `json:"tenant_code"`
	InstitutionID string                 `json:"institution_id"`
	Evidence      SignedArtifactEvidence `json:"evidence"`
}

type remoteSignedArtifactVerificationResponse struct {
	Status               string         `json:"status"`
	TrustedListProvider  string         `json:"trusted_list_provider"`
	TimestampTokenSHA256 string         `json:"timestamp_token_sha256"`
	TimestampAt          string         `json:"timestamp_at"`
	TimestampAuthority   string         `json:"timestamp_authority"`
	Findings             map[string]any `json:"findings"`
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
	return &RemoteSignedArtifactVerifier{
		endpoint: parsed.String(),
		token:    token,
		client: &http.Client{
			Timeout: timeout,
			// The bearer token is scoped to the configured trust endpoint. A
			// redirect could silently forward it to another host or downgrade a
			// path, so verification treats every redirect as a rejected response.
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
		},
	}, nil
}

func (v *RemoteSignedArtifactVerifier) Verify(r *http.Request, evidence SignedArtifactEvidence) SignedArtifactValidationResult {
	return v.verify(r.Context(), authruntime.CurrentTenantCodeFromRequest(r), authruntime.CurrentInstitutionIDFromRequest(r), evidence)
}

func (v *RemoteSignedArtifactVerifier) verify(ctx context.Context, tenantCode, institutionID string, evidence SignedArtifactEvidence) SignedArtifactValidationResult {
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
	decoder := json.NewDecoder(response.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return signatureVerificationError("trust_verifier_response_invalid", 0)
	}
	decoded.Status = strings.ToLower(strings.TrimSpace(decoded.Status))
	if !containsString([]string{"valid", "invalid", "indeterminate", "error"}, decoded.Status) {
		return signatureVerificationError("trust_verifier_status_invalid", 0)
	}
	if decoded.Findings == nil {
		decoded.Findings = map[string]any{}
	}
	var timestampAt *time.Time
	if strings.TrimSpace(decoded.TimestampAt) != "" {
		parsedTimestamp, err := time.Parse(time.RFC3339, decoded.TimestampAt)
		if err != nil {
			return signatureVerificationError("trust_verifier_timestamp_invalid", 0)
		}
		timestampAt = &parsedTimestamp
	}
	timestampHash := strings.ToLower(strings.TrimSpace(decoded.TimestampTokenSHA256))
	if timestampHash != "" && !isLowerSHA256(timestampHash) {
		return signatureVerificationError("trust_verifier_timestamp_hash_invalid", 0)
	}
	return SignedArtifactValidationResult{
		Status:               decoded.Status,
		TrustedListProvider:  strings.TrimSpace(decoded.TrustedListProvider),
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
