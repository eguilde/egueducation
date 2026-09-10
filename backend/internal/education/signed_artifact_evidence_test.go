package education

import (
	"net/http"
	"testing"
)

type deterministicSignedArtifactVerifier struct{}

func (deterministicSignedArtifactVerifier) Verify(_ *http.Request, _ SignedArtifactEvidence) SignedArtifactValidationResult {
	return SignedArtifactValidationResult{Status: "valid", TrustedListProvider: "test-trusted-list", Findings: map[string]any{"deterministic": true}}
}

func TestSignedArtifactVerifierFailsClosedUntilConfigured(t *testing.T) {
	ConfigureSignedArtifactVerifier(nil)
	if got := activeSignedArtifactVerifier().Verify(nil, SignedArtifactEvidence{}); got.Status != "error" {
		t.Fatalf("default verifier status=%q, want error", got.Status)
	}
	ConfigureSignedArtifactVerifier(deterministicSignedArtifactVerifier{})
	defer ConfigureSignedArtifactVerifier(nil)
	if got := activeSignedArtifactVerifier().Verify(nil, SignedArtifactEvidence{}); got.Status != "valid" || got.TrustedListProvider != "test-trusted-list" {
		t.Fatalf("configured verifier result=%+v", got)
	}
}
