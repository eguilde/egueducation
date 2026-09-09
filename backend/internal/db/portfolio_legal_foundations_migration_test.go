package db

import (
	"strings"
	"testing"
)

func TestPortfolioLegalFoundationsMigrationPreservesStatutoryStructureAndEvidence(t *testing.T) {
	const migrationName = "migrations/0100_education_portfolio_legal_foundations.sql"
	contents, err := migrationFiles.ReadFile(migrationName)
	if err != nil {
		t.Fatalf("read %s: %v", migrationName, err)
	}
	text := string(contents)

	for _, required := range []string{
		"create table if not exists education_portfolio_section_catalog_versions",
		"ome-3858-2026-annexa-1-v1",
		"catalog_version = 'legacy-unversioned'",
		"active = false",
		"identificare_profesionala",
		"predare_invatare_evaluare",
		"activitati_complementare",
		"managementul_clasei",
		"evolutie_dezvoltare_profesionala",
		"create table if not exists education_portfolio_procedure_versions",
		"create table if not exists education_portfolio_procedure_section_rules",
		"unique (institution_id, procedure_code, version_no)",
		"lifecycle_status in ('draft', 'approved', 'published', 'superseded', 'withdrawn')",
		"create table if not exists education_portfolio_declaration_acknowledgements",
		"declaration_type in ('gdpr_information', 'authenticity')",
		"declaration_text text not null",
		"attestation_evidence jsonb not null",
		"portfolio declaration acknowledgements are immutable",
		"tenant_code = public.current_tenant_code()",
		"force row level security",
		"record_entity_version()",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("portfolio legal foundations migration is missing %q", required)
		}
	}

	if strings.Contains(text, "disable row level security") {
		t.Error("portfolio legal foundations migration must not disable row level security")
	}
}
