package db

import (
	"strings"
	"testing"
)

func TestEducationCommitteeParentIntegrityMigrationContract(t *testing.T) {
	const migrationName = "migrations/0123_education_committee_parent_integrity.sql"
	body, err := migrationFiles.ReadFile(migrationName)
	if err != nil {
		t.Fatalf("read %s: %v", migrationName, err)
	}
	contents := strings.ToLower(string(body))
	for _, required := range []string{
		"create unique index if not exists education_committees_institution_id_id_uq",
		"drop constraint if exists education_committee_members_committee_id_fkey",
		"foreign key (institution_id, committee_id)",
		"references public.education_committees (institution_id, id)",
		"on delete cascade",
		"unsafe data behind a not valid constraint",
	} {
		if !strings.Contains(contents, required) {
			t.Errorf("0123 missing committee parent-integrity requirement %q", required)
		}
	}
}
