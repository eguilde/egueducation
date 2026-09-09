package db

import (
	"strings"
	"testing"
)

func TestPortfolioLifecycleSafetyMigrationRequiresCessationAndTombstones(t *testing.T) {
	const migrationName = "migrations/0101_education_portfolio_lifecycle_safety.sql"
	contents, err := migrationFiles.ReadFile(migrationName)
	if err != nil {
		t.Fatalf("read %s: %v", migrationName, err)
	}
	text := string(contents)

	for _, required := range []string{
		"activity_ceased_on date",
		"retention_period_days integer not null default 1095",
		"legal_hold_active boolean not null default false",
		"withdrawn_at timestamptz",
		"education portfolios are evidentiary records and cannot be hard-deleted",
		"new.retention_until := new.activity_ceased_on + new.retention_period_days",
		"portfolio withdrawal is blocked after submission, during retention, or under legal hold",
		"portfolio lifecycle records cannot be hard-deleted",
		"portfolio lifecycle record mutation is blocked by legal hold",
		"portfolio evidence mutation is blocked by withdrawal or legal hold",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("portfolio lifecycle migration is missing %q", required)
		}
	}
	if strings.Contains(text, "update education_portfolios\nset retention_until = null") {
		t.Error("lifecycle migration must preserve legacy values for audit instead of bulk-clearing data")
	}
}
