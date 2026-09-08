package config

import "testing"

func TestHTTPRateLimitPerMinuteDefaultsToProductionSafeValue(t *testing.T) {
	t.Setenv("HTTP_RATE_LIMIT_PER_MINUTE", "")
	if got := Load().HTTPRateLimitPerMinute; got != 120 {
		t.Fatalf("HTTPRateLimitPerMinute = %d, want 120", got)
	}
}

func TestHTTPRateLimitPerMinuteAllowsExplicitIsolatedTestCeiling(t *testing.T) {
	t.Setenv("HTTP_RATE_LIMIT_PER_MINUTE", "10000")
	if got := Load().HTTPRateLimitPerMinute; got != 10000 {
		t.Fatalf("HTTPRateLimitPerMinute = %d, want 10000", got)
	}
}

func TestHTTPRateLimitPerMinuteRejectsUnsafeBounds(t *testing.T) {
	for _, value := range []string{"0", "100001", "invalid"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("HTTP_RATE_LIMIT_PER_MINUTE", value)
			if got := Load().HTTPRateLimitPerMinute; got != 120 {
				t.Fatalf("HTTPRateLimitPerMinute = %d, want safe fallback 120", got)
			}
		})
	}
}
