package registratura

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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

func TestStageDocumentAttachmentFailsClosed(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/registratura/documents/document-id/attachments/stage", strings.NewReader(`{"file_name":"orphan.pdf"}`))

	(&Service{}).StageDocumentAttachment(recorder, request)

	if recorder.Code != http.StatusGone {
		t.Fatalf("stage status = %d, want %d", recorder.Code, http.StatusGone)
	}
	if !strings.Contains(recorder.Body.String(), `"code":"attachment_upload_required"`) {
		t.Fatalf("stage response = %s, want attachment_upload_required", recorder.Body.String())
	}
}
