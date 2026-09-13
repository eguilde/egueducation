package institution

import (
	"os"
	"strings"
	"testing"
)

// PostgreSQL treats AUTHORIZATION as a reserved keyword. Keep the queries that
// serve the authenticated offering-authorization endpoints parseable by
// forbidding it as a relation alias (the failure otherwise surfaces as HTTP 500).
func TestOfferingAuthorizationQueriesDoNotUseReservedAuthorizationAlias(t *testing.T) {
	source, err := os.ReadFile("offerings.go")
	if err != nil {
		t.Fatalf("read offerings source: %v", err)
	}
	for _, forbidden := range []string{
		"school_offering_authorizations authorization",
		"select authorization.",
		"where authorization.",
		",authorization.",
		"=authorization.",
	} {
		if strings.Contains(string(source), forbidden) {
			t.Fatalf("offering authorization SQL retains reserved alias fragment %q", forbidden)
		}
	}
	if !strings.Contains(string(source), "school_offering_authorizations authz") {
		t.Fatal("offering authorization SQL no longer has its non-reserved authz alias")
	}
}
