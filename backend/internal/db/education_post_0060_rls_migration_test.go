package db

import (
	"strings"
	"testing"
)

func TestPost0060EducationRLSMigrationContract(t *testing.T) {
	const migrationName = "migrations/0103_education_post_0060_rls.sql"
	contents, err := migrationFiles.ReadFile(migrationName)
	if err != nil {
		t.Fatalf("read %s: %v", migrationName, err)
	}
	text := strings.ToLower(string(contents))

	for _, table := range []string{
		"education_portfolio_valorifications",
		"education_committees",
		"education_committee_members",
	} {
		if !strings.Contains(text, "'"+table+"'") {
			t.Errorf("post-0060 RLS migration does not enumerate %s", table)
		}
	}
	for _, required := range []string{
		"enable row level security",
		"force row level security",
		"drop policy if exists tenant_isolation",
		"create policy tenant_isolation",
		"institution_id = public.current_institution_id()",
		"with check",
		"record_entity_version()",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("post-0060 RLS migration is missing %q", required)
		}
	}
	if strings.Contains(text, "disable row level security") {
		t.Fatal("post-0060 RLS migration must not weaken row-level security")
	}
}

func TestPost0060EducationTablesBelongToInstitutionRLSContract(t *testing.T) {
	byName := make(map[string]TableContract, len(SchemaContract()))
	for _, table := range SchemaContract() {
		byName[table.Name] = table
	}
	for _, name := range []string{
		"education_portfolio_valorifications",
		"education_committees",
		"education_committee_members",
	} {
		table, found := byName[name]
		if !found {
			t.Errorf("schema contract omits %s", name)
			continue
		}
		if table.Scope != SchemaScopeInstitution || !table.requiresRLS() {
			t.Errorf("%s must be institution scoped with forced RLS, got %#v", name, table)
		}
	}
}
