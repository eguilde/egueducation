package institution

import (
	"errors"
	"testing"
	"time"
)

func TestValidateOperationPolicyV2Request(t *testing.T) {
	validID := "5b70cc5b-f1db-49b6-86dd-003348c61359"
	valid := OperationPolicyV2Request{
		OperationCode: "school.admission.decision.issue", EffectiveOn: time.Now(),
		DecisionKind: "operation", OfferingID: validID, LocationID: validID,
		ActorSubject: "director@example.test",
	}
	if err := validateOperationPolicyV2Request(valid); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}
	for _, test := range []struct {
		name string
		edit func(*OperationPolicyV2Request)
	}{
		{"missing operation", func(v *OperationPolicyV2Request) { v.OperationCode = "" }},
		{"missing actor", func(v *OperationPolicyV2Request) { v.ActorSubject = "" }},
		{"missing effective date", func(v *OperationPolicyV2Request) { v.EffectiveOn = time.Time{} }},
		{"unsupported decision kind", func(v *OperationPolicyV2Request) { v.DecisionKind = "migration" }},
		{"half offering scope", func(v *OperationPolicyV2Request) { v.LocationID = "" }},
		{"invalid id", func(v *OperationPolicyV2Request) { v.OfferingID, v.LocationID = "not-a-uuid", validID }},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := valid
			test.edit(&input)
			if err := validateOperationPolicyV2Request(input); !errors.Is(err, ErrPolicyV2ContextInvalid) {
				t.Fatalf("expected context error, got %v", err)
			}
		})
	}
}

func TestCanonicalJSONChecksumIsStable(t *testing.T) {
	left, checksumLeft, err := canonicalJSONChecksum(map[string]any{"z": 1, "a": []string{"x", "y"}})
	if err != nil {
		t.Fatal(err)
	}
	right, checksumRight, err := canonicalJSONChecksum(map[string]any{"a": []string{"x", "y"}, "z": 1})
	if err != nil {
		t.Fatal(err)
	}
	if string(left) != string(right) || checksumLeft != checksumRight || len(checksumLeft) != 64 {
		t.Fatalf("canonical evidence is not stable: %s/%s %s/%s", left, right, checksumLeft, checksumRight)
	}
}
