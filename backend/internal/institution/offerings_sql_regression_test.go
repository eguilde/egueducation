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

func TestOfferingAuthorizationSourceCarriesAuthorizationEffectiveWindow(t *testing.T) {
	source, err := os.ReadFile("offerings.go")
	if err != nil {
		t.Fatalf("read offerings source: %v", err)
	}
	contents := string(source)
	for _, required := range []string{
		"published_on,consolidated_on,effective_from,effective_to,checksum_sha256",
		"nullif($8,'')::date,nullif($9,'')::date,$10::date,nullif($11,'')::date,$12,'active'",
		"input.EffectiveFrom, optionalString(input.EffectiveTo), strings.TrimSpace(input.Source.ChecksumSHA256)",
	} {
		if !strings.Contains(contents, required) {
			t.Fatalf("offering authorization source no longer retains its effective window: missing %q", required)
		}
	}
}
