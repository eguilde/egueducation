package tenant

import "testing"

func TestResolveBrandingFailsClosedForUnknownPublicHost(t *testing.T) {
	ConfigureResolver(nil, ResolverOptions{Environment: "production", BaseDomain: "example.test"})
	if got := ResolveBranding("unknown.example.test", "Fallback", "inst-001"); got.TenantCode != "" || got.InstitutionID != "" {
		t.Fatal("unknown production host must not receive a fallback tenant")
	}
}

func TestResolveBrandingAllowsExplicitLocalDevelopmentFallback(t *testing.T) {
	ConfigureResolver(nil, ResolverOptions{Environment: "test", BaseDomain: "example.test"})
	got := ResolveBranding("127.0.0.1:8080", "Test Institution", "inst-001")
	if got.TenantCode != "tenant-egueducation" || got.Name != "Test Institution" {
		t.Fatalf("local development fallback = %#v", got)
	}
}

func TestNormalizeHost(t *testing.T) {
	if got := normalizeHost("HTTPS://Third.Example.Test:443/path/"); got != "third.example.test" {
		t.Fatalf("normalized hostname = %q", got)
	}
}
