package education

import (
	"regexp"
	"testing"
	"time"
)

func TestNewEducationCodePreservesPrefixAndYearAndIsCollisionResistant(t *testing.T) {
	now := time.Date(2031, time.July, 14, 10, 0, 0, 0, time.FixedZone("test", -5*60*60))
	pattern := regexp.MustCompile(`^PORT-CD-2031-[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	seen := make(map[string]struct{}, 512)
	for range 512 {
		code := newEducationCodeAt("PORT-CD", now)
		if !pattern.MatchString(code) {
			t.Fatalf("unexpected code form: %q", code)
		}
		if _, exists := seen[code]; exists {
			t.Fatalf("duplicate UUID-backed code: %q", code)
		}
		seen[code] = struct{}{}
	}
}
