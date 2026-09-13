package db

import (
	"strings"
	"testing"
)

func TestSignedArtifactProvenanceReconciliationPreservesFinalContract(t *testing.T) {
	body, err := migrationFiles.ReadFile("migrations/0153_education_signed_artifact_provenance_reconciliation.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, required := range []string{
		"new.tenant_code <> public.current_tenant_code()",
		"new.institution_id <> public.current_institution_id()",
		"new.submitted_by_subject := actor_subject",
		"when 'decision'",
		"status <> 'blocked'",
		"when 'publication'",
		"publication_status <> 'retras'",
		"when 'managerial_document'",
		"document_status <> 'archived'",
		"when 'meeting_document'",
		"when 'meeting_minute'",
		"when 'meeting_resolution'",
		"when 'admission_decision'",
		"when 'admission_appeal_resolution'",
		"new.storage_document_id is null or new.storage_version_id is null",
		"version.source_sha256",
		"version.source_bucket",
		"version.source_object_key",
		"document.status = 'ready'",
		"version.status = 'active'",
		"document.institution_id = new.institution_id",
		"version.institution_id = new.institution_id",
		"new.document_sha256 := canonical_sha256",
		"new.storage_bucket := canonical_bucket",
		"new.storage_object_key := canonical_object_key",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("0153 is missing %q", required)
		}
	}
	for _, forbidden := range []string{"http://", "https://", "private_key", "secret_key"} {
		if strings.Contains(strings.ToLower(text), forbidden) {
			t.Errorf("0153 must not introduce verifier egress or key material: %q", forbidden)
		}
	}
}
