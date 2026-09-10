package education

import (
	"os"
	"strings"
	"testing"
)

func TestOperationalCockpitsUseDedicatedPermissionsAndInstitutionScope(t *testing.T) {
	body, err := os.ReadFile("school_role_cockpits.go")
	if err != nil {
		t.Fatalf("read cockpit handlers: %v", err)
	}
	text := string(body)
	for _, required := range []string{"education.cockpit.secretariat.read", "education.cockpit.hr.read", "education.cockpit.committee.read", "education.cockpit.inspector.read", "institution_id=$1", "currentSubjectHasPermission"} {
		if !strings.Contains(text, required) {
			t.Errorf("cockpit handlers missing %q", required)
		}
	}
}
