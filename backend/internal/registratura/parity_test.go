package registratura

import "testing"

func TestCanonicalWorkflowTransitions(t *testing.T) {
	cases := []struct{ status, action, want string }{
		{"INCOMING", "assign_department", "ALOCAT_COMPARTIMENT"},
		{"ALOCAT_COMPARTIMENT", "claim", "IN_LUCRU"},
		{"IN_LUCRU", "send_for_approval", "FLUX_APROBARE"},
		{"FLUX_APROBARE", "approve", "FINALIZAT"},
		{"FLUX_APROBARE", "reject", "IN_LUCRU"},
	}
	for _, tc := range cases {
		got, ok := workflowTransition(tc.status, tc.action)
		if !ok || got != tc.want {
			t.Fatalf("%s/%s = %q, %v", tc.status, tc.action, got, ok)
		}
	}
	if _, ok := workflowTransition("FINALIZAT", "claim"); ok {
		t.Fatal("finalized document must not be claimable")
	}
}

func TestNormalizeDocumentStatus(t *testing.T) {
	for in, want := range map[string]string{"": "INCOMING", "registered": "INCOMING", "in_workflow": "IN_LUCRU", "archived": "FINALIZAT", "anulat": "ANULAT"} {
		if got := normalizeDocumentStatus(in); got != want {
			t.Fatalf("%q = %q, want %q", in, got, want)
		}
	}
}
