package nomenclature

import (
	"encoding/json"
	"testing"
)

func TestItemJSONContract(t *testing.T) {
	item := Item{ID: "id", Domain: "gdpr_domain", Code: "education", LabelRO: "Educație", LabelEN: "Education", Active: true, SortOrder: 4}
	bytes, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(bytes, &payload); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"id", "domain", "code", "label_ro", "label_en", "active", "sort_order"} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("missing public nomenclature key %q: %s", key, bytes)
		}
	}
}

func TestNewServiceAcceptsConfiguredPool(t *testing.T) {
	service := NewService(nil)
	if service == nil || service.pool != nil {
		t.Fatal("service constructor must preserve supplied dependency")
	}
}
