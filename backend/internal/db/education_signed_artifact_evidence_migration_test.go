package db

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSignedArtifactEvidenceMigrationIsAppendOnlyAndTenantScoped(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("migrations", "0113_education_signed_artifact_evidence.sql"))
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"education_signed_artifact_evidence", "education_signed_artifact_validations",
		"signed artifact evidence is append-only", "signed artifact validation is append-only",
		"enable row level security", "force row level security", "education.signatures.read",
		"education.signatures.manage", "education.signatures.validate", "PAdES", "XAdES", "CAdES",
	} {
		if !strings.Contains(string(contents), required) {
			t.Errorf("migration missing %q", required)
		}
	}
}
