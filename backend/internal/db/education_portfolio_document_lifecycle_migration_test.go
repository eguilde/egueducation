package db

import (
	"slices"
	"strings"
	"testing"
)

func TestEducationPortfolioDocumentLifecycleMigrationContract(t *testing.T) {
	const migrationName = "migrations/0114_education_portfolio_document_lifecycle.sql"
	contents, err := migrationFiles.ReadFile(migrationName)
	if err != nil {
		t.Fatalf("read %s: %v", migrationName, err)
	}
	text := strings.ToLower(string(contents))

	for _, required := range []string{
		"status text not null default 'active'",
		"withdrawn_at timestamptz",
		"withdrawn_by_subject text not null default ''",
		"withdrawal_reason text not null default ''",
		"status in ('active', 'withdrawn')",
		"portfolio evidence hard-delete is forbidden",
		"withdrawn portfolio evidence is immutable",
		"portfolio evidence withdrawal cannot alter evidence content",
		"withdrawal requires authenticated actor provenance",
		"for update",
		"before insert or update or delete on education_portfolio_documents",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("portfolio document lifecycle migration is missing %q", required)
		}
	}

	for _, forbidden := range []string{
		"delete from education_portfolio_documents",
		"on delete cascade",
	} {
		if strings.Contains(text, forbidden) {
			t.Errorf("portfolio document lifecycle migration must not contain %q", forbidden)
		}
	}

	var contract TableContract
	for _, candidate := range SchemaContract() {
		if candidate.Name == "education_portfolio_documents" {
			contract = candidate
			break
		}
	}
	if contract.Name == "" {
		t.Fatal("schema contract omits education_portfolio_documents")
	}
	for _, column := range []string{"status", "withdrawn_at", "withdrawn_by_subject", "withdrawal_reason"} {
		if !slices.Contains(contract.RequiredColumns, column) {
			t.Errorf("schema contract education_portfolio_documents omits lifecycle column %s", column)
		}
	}
}
