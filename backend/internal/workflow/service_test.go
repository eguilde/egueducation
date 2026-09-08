package workflow

import (
	"reflect"
	"strings"
	"testing"
)

func TestWorkflowTransitionsAreExplicitAndRejectUnknownPaths(t *testing.T) {
	tests := []struct{ status, action, next, step string }{
		{"new", "start", "in_progress", "Verificare și completare"},
		{"in_progress", "submit", "waiting_approval", "Aprobare finală"},
		{"waiting_approval", "approve", "approved", "Finalizat"},
		{"approved", "archive", "archived", "Arhivat"},
	}
	for _, tc := range tests {
		next, step, ok := resolveTransition(tc.status, tc.action)
		if !ok || next != tc.next || step != tc.step {
			t.Fatalf("%s/%s = %q/%q/%v", tc.status, tc.action, next, step, ok)
		}
	}
	if _, _, ok := resolveTransition("approved", "approve"); ok {
		t.Fatal("invalid state transition must be rejected")
	}
}

func TestWorkflowActionsAndFiltersFailClosed(t *testing.T) {
	if got := availableActions("unknown"); len(got) != 0 {
		t.Fatalf("unknown status actions = %#v", got)
	}
	if !actionRequiresReady("submit") || !actionRequiresReady("approve") || actionRequiresReady("archive") {
		t.Fatal("readiness policy mismatch")
	}
	where, args := buildTaskFilters("tenant-a", map[string]string{"title": "Invoice", "status": "new", "due_on": "2026-01-02"})
	if !strings.HasPrefix(where, "where wi.institution_id = $1") || !strings.Contains(where, "wi.due_at = $4::date") {
		t.Fatalf("unsafe/malformed filters: %s", where)
	}
	want := []any{"tenant-a", "%invoice%", "new", "2026-01-02"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
	if sortColumn("bad") != "wi.started_at" {
		t.Fatal("unknown sort must fall back to an allow-listed column")
	}
}

func TestWorkflowSmallValidationHelpers(t *testing.T) {
	value := "  due  "
	if trimOrEmpty(&value) != "due" || trimOrEmpty(nil) != "" {
		t.Fatal("trimOrEmpty contract broken")
	}
	if !contains([]string{"low", "high"}, "high") || contains([]string{"low"}, "urgent") {
		t.Fatal("contains contract broken")
	}
}
