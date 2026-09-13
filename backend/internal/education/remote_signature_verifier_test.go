package education

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func validRemoteVerifierEvidence(payloadHash, actor string) SignedArtifactEvidence {
	return SignedArtifactEvidence{
		ExpectedCanonicalLegalPayloadSHA256: payloadHash,
		ExpectedActorSubject:                actor,
		StorageObjectVersionID:              "minio-version-1",
		StorageRetentionUntil:               time.Now().UTC().Add(time.Hour).Format(time.RFC3339Nano),
	}
}

func TestRemoteSignedArtifactVerifierPreservesNegativeVerdictWithoutTrustClaims(t *testing.T) {
	for _, status := range []string{"invalid", "indeterminate", "error"} {
		for _, withClaims := range []bool{false, true} {
			t.Run(status+"/claims="+fmt.Sprint(withClaims), func(t *testing.T) {
				response := remoteSignedArtifactVerificationResponse{Status: status, Findings: map[string]any{"code": "signature_not_validated"}}
				if withClaims {
					response.SignedPayloadSHA256 = strings.Repeat("a", 64)
					response.CertificateSHA256 = strings.Repeat("b", 64)
					response.SignedActorSubject = "actor-a"
					response.SignatureLevel = "qualified"
				}
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					_ = json.NewEncoder(w).Encode(response)
				}))
				defer server.Close()
				verifier, err := NewRemoteSignedArtifactVerifier(server.URL, "test-token", time.Second)
				if err != nil {
					t.Fatal(err)
				}
				result := verifier.verify(context.Background(), "tenant-a", "institution-a", validRemoteVerifierEvidence(strings.Repeat("a", 64), "actor-a"))
				if result.Status != status || result.Findings["code"] != "signature_not_validated" {
					t.Fatalf("negative verdict lost: %+v", result)
				}
				if result.SignedPayloadSHA256 != "" || result.CertificateSHA256 != "" || result.SignedActorSubject != "" || result.SignatureLevel != "" || result.TimestampAt != nil {
					t.Fatalf("negative verdict retained trust claims: %+v", result)
				}
			})
		}
	}
}

func TestRemoteSignedArtifactVerifierUsesServerTenantAndTrustResponse(t *testing.T) {
	const token = "trust-service-secret"
	expectedPayloadHash := strings.Repeat("1", 64)
	certificateHash := strings.Repeat("2", 64)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			t.Errorf("authorization header was not sent")
		}
		var payload remoteSignedArtifactVerificationRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode trust request: %v", err)
		}
		if payload.TenantCode != "tenant-a" || payload.InstitutionID != "institution-a" {
			t.Errorf("trust scope = %q/%q", payload.TenantCode, payload.InstitutionID)
		}
		_ = json.NewEncoder(w).Encode(remoteSignedArtifactVerificationResponse{
			Status:               "valid",
			SignatureSubject:     "CN=Qualified System Test Signer",
			SignedActorSubject:   "actor-a",
			TrustedListProvider:  "EU Trusted List",
			SignedPayloadSHA256:  expectedPayloadHash,
			CertificateSHA256:    certificateHash,
			TimestampTokenSHA256: "a9" + "00" + "000000000000000000000000000000000000000000000000000000000000",
			TimestampAt:          "2026-09-10T10:30:00Z",
			TimestampAuthority:   "Qualified TSA",
			Findings:             map[string]any{"certificate_path": "trusted"},
		})
	}))
	defer server.Close()
	verifier, err := NewRemoteSignedArtifactVerifier(server.URL, token, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	evidence := validRemoteVerifierEvidence(expectedPayloadHash, "actor-a")
	evidence.ID, evidence.ArtifactType = "evidence-a", "decision"
	result := verifier.verify(context.Background(), "tenant-a", "institution-a", evidence)
	if result.Status != "valid" || result.TrustedListProvider != "EU Trusted List" || result.TimestampAt == nil {
		t.Fatalf("verification result = %+v", result)
	}
	if result.SignedPayloadSHA256 != expectedPayloadHash || result.CertificateSHA256 != certificateHash {
		t.Fatalf("independently extracted hashes were not retained: %+v", result)
	}
}

func TestRemoteSignedArtifactVerifierFailsClosedOnInvalidProviderResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"valid","trusted_list_provider":"fake","unexpected":true}`))
	}))
	defer server.Close()
	verifier, err := NewRemoteSignedArtifactVerifier(server.URL, "secret", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	result := verifier.Verify(httptest.NewRequest(http.MethodPost, "http://application.test", nil), validRemoteVerifierEvidence(strings.Repeat("a", 64), "actor-a"))
	if result.Status != "error" || result.Findings["code"] != "trust_verifier_response_invalid" {
		t.Fatalf("verification result = %+v", result)
	}
}

func TestRemoteSignedArtifactVerifierDoesNotFollowRedirectsOrForwardToken(t *testing.T) {
	var redirectedAuthorization string
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectedAuthorization = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()
	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer redirector.Close()
	verifier, err := NewRemoteSignedArtifactVerifier(redirector.URL, "do-not-forward", time.Second)
	if err != nil {
		t.Fatalf("create localhost verifier: %v", err)
	}
	result := verifier.Verify(httptest.NewRequest(http.MethodPost, "http://application.test", nil), validRemoteVerifierEvidence(strings.Repeat("a", 64), "actor-a"))
	if result.Status != "error" || result.Findings["code"] != "trust_verifier_rejected" {
		t.Fatalf("redirect result=%+v, want rejected fail-closed", result)
	}
	if redirectedAuthorization != "" {
		t.Fatalf("redirect target received verifier bearer token: %q", redirectedAuthorization)
	}
}

func TestRemoteSignedArtifactVerifierReadinessIsAuthenticatedAndCapabilityBound(t *testing.T) {
	const token = "readiness-secret"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/verify/capabilities" || r.Method != http.MethodGet {
			t.Errorf("readiness request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer "+token {
			t.Errorf("readiness request was not authenticated")
		}
		_ = json.NewEncoder(w).Encode(remoteSignedArtifactVerifierCapabilities{
			Status: "ready", ProtocolVersion: remoteVerifierProtocolVersion,
			Capabilities: []string{"signed-payload-sha256", "leaf-certificate-sha256", "expected-actor-subject", "exact-storage-object-version"},
		})
	}))
	defer server.Close()
	verifier, err := NewRemoteSignedArtifactVerifier(server.URL+"/verify", token, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifier.Ready(context.Background()); err != nil {
		t.Fatalf("compatible verifier reported not ready: %v", err)
	}
}

func TestRemoteSignedArtifactVerifierReadinessFailsClosed(t *testing.T) {
	for name, handler := range map[string]http.HandlerFunc{
		"missing capability": func(w http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(w).Encode(remoteSignedArtifactVerifierCapabilities{Status: "ready", ProtocolVersion: remoteVerifierProtocolVersion, Capabilities: []string{"signed-payload-sha256"}})
		},
		"oversized response": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(strings.Repeat("x", remoteVerifierMaxResponseBody+1)))
		},
		"provider rejected": func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusUnauthorized) },
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(handler)
			defer server.Close()
			verifier, err := NewRemoteSignedArtifactVerifier(server.URL, "secret", time.Second)
			if err != nil {
				t.Fatal(err)
			}
			if err := verifier.Ready(context.Background()); err == nil {
				t.Fatal("incompatible verifier readiness accepted")
			}
		})
	}
}

type verifierWithoutReadiness struct{}

func (verifierWithoutReadiness) Verify(*http.Request, SignedArtifactEvidence) SignedArtifactValidationResult {
	return SignedArtifactValidationResult{Status: "valid"}
}

func TestSignedArtifactVerifierReadyFailsClosedWithoutActiveCapabilityProbe(t *testing.T) {
	ConfigureSignedArtifactVerifier(nil)
	if err := SignedArtifactVerifierReady(context.Background()); err == nil {
		t.Fatal("unconfigured verifier reported ready")
	}
	ConfigureSignedArtifactVerifier(verifierWithoutReadiness{})
	defer ConfigureSignedArtifactVerifier(nil)
	if err := SignedArtifactVerifierReady(context.Background()); err == nil {
		t.Fatal("verifier without readiness capability reported ready")
	}
}

func TestRemoteSignedArtifactVerifierRejectsMissingExpectedContractBeforeNetwork(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests++ }))
	defer server.Close()
	verifier, err := NewRemoteSignedArtifactVerifier(server.URL, "secret", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []struct {
		evidence SignedArtifactEvidence
		code     string
	}{
		{SignedArtifactEvidence{ExpectedActorSubject: "actor-a"}, "trust_verifier_expected_contract_invalid"},
		{SignedArtifactEvidence{ExpectedCanonicalLegalPayloadSHA256: strings.Repeat("a", 64)}, "trust_verifier_expected_contract_invalid"},
		{SignedArtifactEvidence{ExpectedCanonicalLegalPayloadSHA256: "not-a-sha256", ExpectedActorSubject: "actor-a"}, "trust_verifier_expected_contract_invalid"},
		{SignedArtifactEvidence{ExpectedCanonicalLegalPayloadSHA256: strings.Repeat("a", 64), ExpectedActorSubject: "actor-a", StorageRetentionUntil: time.Now().UTC().Add(time.Hour).Format(time.RFC3339Nano)}, "trust_verifier_expected_contract_invalid"},
		{SignedArtifactEvidence{ExpectedCanonicalLegalPayloadSHA256: strings.Repeat("a", 64), ExpectedActorSubject: "actor-a", StorageObjectVersionID: "version-1", StorageRetentionUntil: time.Now().UTC().Add(-time.Hour).Format(time.RFC3339Nano)}, "trust_verifier_storage_retention_invalid"},
	} {
		result := verifier.verify(context.Background(), "tenant-a", "institution-a", scenario.evidence)
		if result.Status != "error" || result.Findings["code"] != scenario.code {
			t.Fatalf("invalid expected contract result = %+v", result)
		}
	}
	if requests != 0 {
		t.Fatalf("invalid expected contract reached verifier %d times", requests)
	}
}

func TestRemoteSignedArtifactVerifierRejectsExtractedContractMismatch(t *testing.T) {
	expectedPayloadHash := strings.Repeat("a", 64)
	validResponse := func() remoteSignedArtifactVerificationResponse {
		return remoteSignedArtifactVerificationResponse{
			Status: "valid", SignatureSubject: "CN=Qualified Signer", SignedActorSubject: "actor-a", SignedPayloadSHA256: expectedPayloadHash,
			CertificateSHA256: strings.Repeat("b", 64), Findings: map[string]any{},
		}
	}
	for name, mutate := range map[string]func(*remoteSignedArtifactVerificationResponse){
		"missing signed payload hash": func(x *remoteSignedArtifactVerificationResponse) { x.SignedPayloadSHA256 = "" },
		"mismatched signed payload":   func(x *remoteSignedArtifactVerificationResponse) { x.SignedPayloadSHA256 = strings.Repeat("c", 64) },
		"malformed certificate hash":  func(x *remoteSignedArtifactVerificationResponse) { x.CertificateSHA256 = "certificate" },
		"mismatched actor":            func(x *remoteSignedArtifactVerificationResponse) { x.SignedActorSubject = "actor-b" },
	} {
		t.Run(name, func(t *testing.T) {
			response := validResponse()
			mutate(&response)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _ = json.NewEncoder(w).Encode(response) }))
			defer server.Close()
			verifier, err := NewRemoteSignedArtifactVerifier(server.URL, "secret", time.Second)
			if err != nil {
				t.Fatal(err)
			}
			result := verifier.verify(context.Background(), "tenant-a", "institution-a", validRemoteVerifierEvidence(expectedPayloadHash, "actor-a"))
			if result.Status != "error" {
				t.Fatalf("tampered extracted contract accepted: %+v", result)
			}
		})
	}
}
