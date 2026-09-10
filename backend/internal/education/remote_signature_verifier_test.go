package education

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRemoteSignedArtifactVerifierUsesServerTenantAndTrustResponse(t *testing.T) {
	const token = "trust-service-secret"
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
			TrustedListProvider:  "EU Trusted List",
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
	result := verifier.verify(context.Background(), "tenant-a", "institution-a", SignedArtifactEvidence{ID: "evidence-a", ArtifactType: "decision"})
	if result.Status != "valid" || result.TrustedListProvider != "EU Trusted List" || result.TimestampAt == nil {
		t.Fatalf("verification result = %+v", result)
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
	result := verifier.Verify(httptest.NewRequest(http.MethodPost, "http://application.test", nil), SignedArtifactEvidence{})
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
	result := verifier.Verify(httptest.NewRequest(http.MethodPost, "http://application.test", nil), SignedArtifactEvidence{})
	if result.Status != "error" || result.Findings["code"] != "trust_verifier_rejected" {
		t.Fatalf("redirect result=%+v, want rejected fail-closed", result)
	}
	if redirectedAuthorization != "" {
		t.Fatalf("redirect target received verifier bearer token: %q", redirectedAuthorization)
	}
}
