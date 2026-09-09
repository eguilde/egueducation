package education

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestPortfolioDeclarationAcknowledgementRequestHasNoClientWordingFields(t *testing.T) {
	var request PortfolioDeclarationAcknowledgementRequest
	if err := json.Unmarshal([]byte(`{
		"confirmed": true,
		"declaration_version": "forged-v999",
		"declaration_text": "forged text",
		"attestation_evidence": {"forged": true}
	}`), &request); err != nil {
		t.Fatalf("decode acknowledgement command: %v", err)
	}
	if !request.Confirmed {
		t.Fatal("affirmative acknowledgement command must retain confirmation")
	}
	// The command DTO deliberately has a single field. JSON wording/evidence
	// extras are ignored and cannot reach the persistence insert.
	typ := reflect.TypeOf(request)
	if got := typ.NumField(); got != 1 {
		t.Fatalf("acknowledgement request exposes %d client-controlled fields, want 1", got)
	}
	if tag := typ.Field(0).Tag.Get("json"); tag != "confirmed" {
		t.Fatalf("acknowledgement field JSON tag = %q, want confirmed", tag)
	}
}
