package education

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestSignedArtifactSelectorQueryIsTypedAndPaged(t *testing.T) {
	query, err := parseSignedArtifactSelectorQuery(url.Values{"artifactType": {" Decision "}, "q": {" Hotărâre "}, "page": {"2"}, "pageSize": {"250"}})
	if err != nil {
		t.Fatal(err)
	}
	if query.ArtifactType != "decision" || query.Search != "Hotărâre" || query.Page != 2 || query.PageSize != 100 {
		t.Fatalf("unexpected normalized selector query: %#v", query)
	}
	for _, artifactType := range signedArtifactTypes {
		if _, err := parseSignedArtifactSelectorQuery(url.Values{"artifactType": {artifactType}}); err != nil {
			t.Errorf("valid artifact type %q rejected: %v", artifactType, err)
		}
	}
	if _, err := parseSignedArtifactSelectorQuery(url.Values{"artifactType": {"unknown"}}); err == nil {
		t.Fatal("unknown artifact type must be rejected")
	}
}

func TestSignedArtifactSelectorsRecheckManagePermission(t *testing.T) {
	for _, handler := range []func(http.ResponseWriter, *http.Request){new(Service).EligibleSignedArtifacts, new(Service).EligibleSignatureArchiveVersions} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/education/signatures/eligible-artifacts?artifactType=decision", nil)
		handler(recorder, request)
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("direct unauthenticated selector call = %d, want 403", recorder.Code)
		}
	}
}

func TestSubmitSignedArtifactEvidenceRejectsBrowserProvenance(t *testing.T) {
	valid := `{"artifact_type":"decision","artifact_id":"11111111-1111-1111-1111-111111111111","signature_format":"PAdES","signature_level":"advanced","signature_subject":"Signer","certificate_issuer":"Issuer","certificate_serial":"01","certificate_valid_from":"2026-01-01T00:00:00Z","certificate_valid_until":"2027-01-01T00:00:00Z","storage_document_id":"22222222-2222-2222-2222-222222222222","storage_version_id":"33333333-3333-3333-3333-333333333333"}`
	if _, err := decodeSubmitSignedArtifactEvidenceRequest(strings.NewReader(valid)); err != nil {
		t.Fatalf("valid server-provenance request rejected: %v", err)
	}
	for _, injected := range []string{`"document_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",`, `"storage_bucket":"attacker",`, `"storage_object_key":"attacker.pdf",`} {
		payload := strings.Replace(valid, `"artifact_type":`, injected+`"artifact_type":`, 1)
		if _, err := decodeSubmitSignedArtifactEvidenceRequest(strings.NewReader(payload)); err == nil {
			t.Fatalf("browser-owned provenance field accepted: %s", injected)
		}
	}
	missingStorage := strings.Replace(valid, `,"storage_version_id":"33333333-3333-3333-3333-333333333333"`, "", 1)
	if _, err := decodeSubmitSignedArtifactEvidenceRequest(strings.NewReader(missingStorage)); err == nil {
		t.Fatal("storage version must be required")
	}
}

func TestSignedArtifactSelectorPlansAreInstitutionScoped(t *testing.T) {
	for _, artifactType := range signedArtifactTypes {
		plan, err := signedArtifactSelectorPlanFor(artifactType)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(plan.fromWhere, "institution_id=public.current_institution_id()") || !strings.Contains(plan.searchClause, "$1") {
			t.Errorf("selector %q is not institution-scoped and parameterized: %#v", artifactType, plan)
		}
	}
}
