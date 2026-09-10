package education

import (
	"testing"
	"time"
)

func TestSchoolReportCatalogHasServerOwnedColumnsAndSafeDefaultSorts(t *testing.T) {
	if len(schoolReportSpecs) < 5 {
		t.Fatalf("school report catalog has %d reports; want at least 5", len(schoolReportSpecs))
	}
	for code, spec := range schoolReportSpecs {
		if code != spec.catalog.Code || spec.catalog.Label == "" || spec.baseSQL == "" {
			t.Errorf("incomplete report specification for %q", code)
		}
		seen := map[string]bool{}
		for _, column := range spec.catalog.Columns {
			if column.Key == "" || column.Label == "" || seen[column.Key] {
				t.Errorf("invalid or duplicate column %q in report %q", column.Key, code)
			}
			seen[column.Key] = true
		}
		if !seen[spec.defaultSort] {
			t.Errorf("default sort %q is not an allowlisted report column for %q", spec.defaultSort, code)
		}
		if got := spec.catalog.Formats; len(got) != 3 || got[0] != "json" || got[1] != "csv" || got[2] != "pdf" {
			t.Errorf("report %q formats = %v; want json/csv/pdf", code, got)
		}
	}
}

func TestReportRowValuesUsesStableExportRepresentations(t *testing.T) {
	columns := schoolReportColumns("date", "Data", "valid", "Valid", "missing", "Lipsă", "count", "Număr")
	row := map[string]any{
		"date":  time.Date(2026, time.September, 10, 22, 15, 0, 0, time.FixedZone("test", 3*60*60)),
		"valid": true,
		"count": int64(7),
	}
	got := reportRowValues(row, columns)
	want := []string{"2026-09-10", "Da", "", "7"}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("export value %d = %q; want %q (all: %v)", index, got[index], want[index], got)
		}
	}
}

func TestPortfolioValorificationPurposeCompatibilityCoversCatalog(t *testing.T) {
	for _, purpose := range portfolioValorificationScopes() {
		allowed := portfolioValorificationPurposeAllowed("evaluare_profesionala", purpose) ||
			portfolioValorificationPurposeAllowed("mobilitate", purpose) ||
			portfolioValorificationPurposeAllowed("gradatie_merit", purpose)
		if !allowed {
			t.Errorf("catalog purpose %q cannot be represented by a strict evidence package", purpose)
		}
	}
	if portfolioValorificationPurposeAllowed("mobilitate", "licentiere") {
		t.Fatal("mobility source must not be accepted for licensing purpose")
	}
}
