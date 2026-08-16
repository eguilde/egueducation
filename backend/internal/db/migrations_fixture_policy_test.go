package db

import (
	"regexp"
	"strings"
	"testing"
)

var migrationEmailPattern = regexp.MustCompile(`(?i)\b[A-Z0-9._%+\-]+@([A-Z0-9.\-]+)\b`)
var migrationInternationalPhonePattern = regexp.MustCompile(`\+40[0-9]{9}`)
var migrationNationalMobilePattern = regexp.MustCompile(`(?m)(?:^|[^0-9])07[0-9]{8}(?:$|[^0-9])`)
var migrationSyntheticPhonePattern = regexp.MustCompile(`^\+401[0-9]{8}$`)

// TestMigrationFixturesAreSynthetic is a source-level CI gate. Historical
// migration files are immutable deployment records, so the only permissible
// fixture contact values are the documented example.test addresses and the
// deliberately non-routable +401xxxxxxxx range.
func TestMigrationFixturesAreSynthetic(t *testing.T) {
	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		t.Fatalf("read embedded migrations: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		contents, err := migrationFiles.ReadFile("migrations/" + entry.Name())
		if err != nil {
			t.Fatalf("read migration %s: %v", entry.Name(), err)
		}
		text := string(contents)
		for _, match := range migrationEmailPattern.FindAllStringSubmatch(text, -1) {
			if !strings.EqualFold(match[1], "example.test") {
				t.Fatalf("migration %s contains a non-fixture email domain", entry.Name())
			}
		}
		for _, phone := range migrationInternationalPhonePattern.FindAllString(text, -1) {
			if !migrationSyntheticPhonePattern.MatchString(phone) {
				t.Fatalf("migration %s contains a non-synthetic Romanian phone literal", entry.Name())
			}
		}
		if migrationNationalMobilePattern.MatchString(text) {
			t.Fatalf("migration %s contains a national-format Romanian mobile literal", entry.Name())
		}
	}
}
