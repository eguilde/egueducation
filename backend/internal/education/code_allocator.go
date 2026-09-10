package education

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// newEducationCode creates an opaque, collision-resistant School record code.
// UUID v4 uses cryptographic randomness; the prefix and UTC year remain
// human-readable while uniqueness never depends on scheduler timing.
func newEducationCode(prefix string) string {
	return newEducationCodeAt(prefix, time.Now().UTC())
}

func newEducationCodeAt(prefix string, now time.Time) string {
	return fmt.Sprintf("%s-%d-%s", prefix, now.UTC().Year(), uuid.NewString())
}
