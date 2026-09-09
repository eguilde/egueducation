package db

import (
	"strings"
	"testing"
)

func TestPortfolioExportManifestMigrationPreservesImmutableTenantScopedEvidence(t *testing.T) {
	contents, err := migrationFiles.ReadFile("migrations/0104_education_portfolio_export_manifests.sql")
	if err != nil {
		t.Fatalf("read export manifest migration: %v", err)
	}
	text := strings.ToLower(string(contents))
	for _, required := range []string{
		"education_portfolio_export_manifests",
		"manifest_version", "manifest_sha256", "manifest jsonb", "tenant_code", "institution_id",
		"unique (portfolio_id, manifest_sha256)", "cannot be deleted",
		"enable row level security", "force row level security", "tenant_code = public.current_tenant_code()",
		"record_entity_version()",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("export manifest migration missing %q", required)
		}
	}
	if strings.Contains(text, "disable row level security") {
		t.Fatal("export manifest migration must not weaken RLS")
	}
}
