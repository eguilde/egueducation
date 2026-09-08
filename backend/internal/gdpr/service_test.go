package gdpr

import (
	"reflect"
	"strings"
	"testing"
)

func TestRetentionFiltersRemainTenantScopedAndParameterised(t *testing.T) {
	where, args := buildRetentionFilters("tenant-a", map[string]string{"domain_code": "school", "status": "active", "policy_code": "ABC"})
	if !strings.HasPrefix(where, "where grp.institution_id = $1") || !strings.Contains(where, "lower(grp.policy_code) like $2") {
		t.Fatalf("unexpected retention filter: %s", where)
	}
	want := []any{"tenant-a", "%abc%", "school", "active"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestGDPRFilterBuildersUseTenantBoundaryAndTypedDateBooleanFilters(t *testing.T) {
	tests := []struct {
		name, where string
		args        []any
	}{
		{"subject", func() string {
			w, _ := buildSubjectRequestFilters("tenant", map[string]string{"submitted_on": "2026-01-02", "anonymization_required": "true"})
			return w
		}(), nil},
		{"export", func() string {
			w, _ := buildSubjectExportFilters("tenant", map[string]string{"generated_on": "2026-01-02"})
			return w
		}(), nil},
		{"publication", func() string {
			w, _ := buildPublicationReviewFilters("tenant", map[string]string{"reviewed_on": "2026-01-02"})
			return w
		}(), nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(tc.where, "institution_id = $1") || !strings.Contains(tc.where, "::date") {
				t.Fatalf("filter lost tenant/date protection: %s", tc.where)
			}
		})
	}
	_, args := buildSubjectRequestFilters("tenant", map[string]string{"anonymization_required": "anything-else"})
	if got, ok := args[len(args)-1].(bool); !ok || got {
		t.Fatalf("boolean filter must not coerce arbitrary input to true: %#v", args)
	}
}

func TestGDPRSortColumnsFailClosedToStableDefaults(t *testing.T) {
	if retentionSortColumn("unknown") != "grp.review_due_on" || subjectRequestSortColumn("unknown") != "gsr.due_on" || subjectExportSortColumn("unknown") != "gse.created_at" || publicationReviewSortColumn("unknown") != "gpr.created_at" {
		t.Fatal("unknown sort fields must use stable allow-listed defaults")
	}
}
