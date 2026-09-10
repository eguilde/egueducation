package auth

import (
	"os"
	"strings"
	"testing"
)

func TestTenantKeyedSessionContextQueries(t *testing.T) {
	for _, file := range []string{"service.go", "oidc_provider.go"} {
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		contents := strings.ToLower(string(body))
		for _, required := range []string{
			"join app_session_context sc",
			"sc.tenant_code",
			"sc.institution_id",
		} {
			if !strings.Contains(contents, required) {
				t.Errorf("%s missing tenant-keyed session-context query clause %q", file, required)
			}
		}
	}
}
