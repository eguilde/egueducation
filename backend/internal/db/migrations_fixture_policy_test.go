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

var approvedOperatorMigrationContacts = map[string]struct{}{
	"thomas@eguilde.cloud": {},
	"+40771364169":         {},
	"+40744652476":         {},
	"+40735091230":         {},
}

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
			if !strings.EqualFold(match[1], "example.test") && !isApprovedOperatorContact(entry.Name(), match[0]) {
				t.Fatalf("migration %s contains a non-fixture email domain", entry.Name())
			}
		}
		for _, phone := range migrationInternationalPhonePattern.FindAllString(text, -1) {
			if !migrationSyntheticPhonePattern.MatchString(phone) && !isApprovedOperatorContact(entry.Name(), phone) {
				t.Fatalf("migration %s contains a non-synthetic Romanian phone literal", entry.Name())
			}
		}
		if migrationNationalMobilePattern.MatchString(text) {
			t.Fatalf("migration %s contains a national-format Romanian mobile literal", entry.Name())
		}
	}
}

func isApprovedOperatorContact(migrationName, value string) bool {
	if migrationName != "0083_identity_contract_foundation.sql" {
		return false
	}
	_, ok := approvedOperatorMigrationContacts[strings.ToLower(value)]
	return ok
}
