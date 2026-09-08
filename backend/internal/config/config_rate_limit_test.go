package config

import "testing"

func TestHTTPRateLimitPerMinuteIsDisabledByDefault(t *testing.T) {
	t.Setenv("HTTP_RATE_LIMIT_PER_MINUTE", "")
	if got := Load().HTTPRateLimitPerMinute; got != 0 {
		t.Fatalf("HTTPRateLimitPerMinute = %d, want disabled value 0", got)
	}
}

func TestHTTPRateLimitPerMinuteCanBeEnabledExplicitly(t *testing.T) {
	t.Setenv("HTTP_RATE_LIMIT_PER_MINUTE", "120")
	if got := Load().HTTPRateLimitPerMinute; got != 120 {
		t.Fatalf("HTTPRateLimitPerMinute = %d, want 120", got)
	}
}

func TestHTTPRateLimitPerMinuteRejectsUnsafeBounds(t *testing.T) {
	for _, value := range []string{"-1", "100001", "invalid"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("HTTP_RATE_LIMIT_PER_MINUTE", value)
			if got := Load().HTTPRateLimitPerMinute; got != 0 {
				t.Fatalf("HTTPRateLimitPerMinute = %d, want disabled fallback 0", got)
			}
		})
	}
}
