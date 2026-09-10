package db

import (
	"strings"
	"testing"
)

func TestEArchiveWORMAndSignatureReportsMigrationContract(t *testing.T) {
	body, err := migrationFiles.ReadFile("migrations/0126_earchiva_worm_and_signature_validation_reports.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(body)
	for _, required := range []string{
		"archive document versions are immutable",
		"archive document version bitstream provenance is immutable",
		"education_signed_artifact_evidence_archive_version_fk",
		"diagnostic_data jsonb",
		"detailed_report jsonb",
		"simple_report jsonb",
		"etsi_validation_report jsonb",
		"observed_sha256",
		"source_object_version_id",
		"retention_until",
		"legal_hold_active",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration is missing %q", required)
		}
	}
}
