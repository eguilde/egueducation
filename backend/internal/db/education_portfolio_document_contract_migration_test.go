package db

import (
	"strings"
	"testing"
)

func TestPortfolioDocumentContractMigrationRequiresMetadataAndArchiveSnapshot(t *testing.T) {
	contents, err := migrationFiles.ReadFile("migrations/0131_education_portfolio_document_contract.sql")
	if err != nil {
		t.Fatalf("read portfolio document contract migration: %v", err)
	}
	text := strings.ToLower(string(contents))
	for _, required := range []string{
		"description", "school_year", "subject_discipline", "applicable_class", "competencies",
		"education_portfolio_documents_pedagogical_metadata_check",
		"education_portfolio_documents_archive_snapshot_check",
		"archive_document_id is not null", "archive_version_id is not null", "archive_sha256",
		"enforce_education_portfolio_document_archive_contract",
		"document.status = 'ready'", "version.status = 'active'",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("portfolio document contract migration missing %q", required)
		}
	}
	if strings.Contains(text, "disable row level security") {
		t.Fatal("portfolio document contract migration must not weaken RLS")
	}
}
