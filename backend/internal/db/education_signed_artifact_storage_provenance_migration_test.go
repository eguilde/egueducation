package db

import (
	"strings"
	"testing"
)

func TestSignedArtifactStorageProvenanceMigrationIsServerOwned(t *testing.T) {
	body, err := migrationFiles.ReadFile("migrations/0117_education_signed_artifact_storage_provenance.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, required := range []string{
		"storage_document_id is not null", "storage_version_id is not null",
		"version.source_sha256", "version.source_bucket", "version.source_object_key",
		"document.status='ready'", "version.status='active'",
		"new.document_sha256 := lower(canonical_sha256)",
		"new.storage_bucket := canonical_bucket", "new.storage_object_key := canonical_object_key",
		"new.tenant_code <> public.current_tenant_code()", "institution_id=new.institution_id",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("0117 is missing %q", required)
		}
	}
	for _, forbidden := range []string{"http://", "https://", "private_key", "secret_key"} {
		if strings.Contains(strings.ToLower(text), forbidden) {
			t.Errorf("0117 must not introduce verifier egress or key material: %q", forbidden)
		}
	}
}
