package db

import (
	"strings"
	"testing"
)

func TestOperationalCockpitPermissionMigration(t *testing.T) {
	body, err := migrationFiles.ReadFile("migrations/0116_education_operational_cockpit_permissions.sql")
	if err != nil {
		t.Fatalf("read 0116: %v", err)
	}
	for _, permission := range []string{"education.cockpit.secretariat.read", "education.cockpit.hr.read", "education.cockpit.committee.read", "education.cockpit.inspector.read"} {
		if !strings.Contains(string(body), permission) {
			t.Errorf("0116 missing %s", permission)
		}
	}
}
