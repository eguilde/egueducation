package db

import (
	"strings"
	"testing"
)

func TestEducationOfficialArtifactImmutabilityMigrationContract(t *testing.T) {
	const migrationName = "migrations/0108_education_official_artifact_immutability.sql"
	contents, err := migrationFiles.ReadFile(migrationName)
	if err != nil {
		t.Fatalf("read %s: %v", migrationName, err)
	}
	text := string(contents)

	for _, required := range []string{
		"create or replace function public.enforce_education_official_artifact_immutability()",
		"create or replace function public.education_meeting_is_officially_sealed(",
		"meeting.status = 'published'",
		"old.status in ('approved', 'published')",
		"old.publication_status = 'publicat'",
		"old.document_status = 'published'",
		"old.follow_up_status = 'inchis'",
		"published meeting or meeting with finalized evidence is immutable",
		"meeting participant or vote mutation is blocked after publication or finalized evidence",
		"document.institution_id = meeting.institution_id",
		"minute.institution_id = meeting.institution_id",
		"resolution.institution_id = meeting.institution_id",
		"published meeting or meeting with finalized evidence cannot be hard-deleted",
		"provenance-bearing approved-to-published transition",
		"provenance-bearing publicat-to-retras transition",
		"provenance-bearing published-to-archived transition",
		"published_by_subject text not null default ''",
		"withdrawn_by_subject text not null default ''",
		"archived_by_subject text not null default ''",
		"before update or delete on education_meetings",
		"before insert or update or delete on education_meeting_participants",
		"before insert or update or delete on education_meeting_votes",
		"before update or delete on education_decisions",
		"before update or delete on education_meeting_documents",
		"before update or delete on education_managerial_documents",
		"before update or delete on education_meeting_minutes",
		"before update or delete on education_meeting_resolutions",
		"before update or delete on education_publications",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("official artifact immutability migration is missing %q", required)
		}
	}

	if strings.Contains(text, "on delete cascade") {
		t.Fatal("immutability migration must not introduce cascade deletion for official artifacts")
	}
}
