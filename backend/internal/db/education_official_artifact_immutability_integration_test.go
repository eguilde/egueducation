package db

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestEducationOfficialArtifactImmutabilityIntegration proves that an
// ordinary NOBYPASSRLS application role cannot alter or delete official
// records, while draft work remains possible and foreign-tenant artifacts
// remain invisible. The enforcement being exercised is a database trigger,
// not a handler convention.
func TestEducationOfficialArtifactImmutabilityIntegration(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("EGUEDUCATION_TEST_DATABASE_URL"))
	if databaseURL == "" {
		databaseURL = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	}
	if databaseURL == "" {
		t.Skip("official artifact PostgreSQL integration test requires EGUEDUCATION_TEST_DATABASE_URL or TEST_DATABASE_URL")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	it := newTenantGrantRLSIntegrationDatabase(t, ctx, databaseURL)
	adminPool, err := pgxpool.NewWithConfig(ctx, it.databaseConfig.Copy())
	if err != nil {
		t.Fatalf("open official artifact integration database: %v", err)
	}
	t.Cleanup(adminPool.Close)
	if err := Migrate(ctx, adminPool); err != nil {
		t.Fatalf("apply official artifact migrations: %v", err)
	}
	fixture := seedOfficialArtifactFixture(t, ctx, adminPool)
	grantOfficialArtifactApplicationAccess(t, ctx, adminPool, it.roleName)

	applicationConfig := it.databaseConfig.Copy()
	applicationConfig.ConnConfig.User = it.roleName
	applicationConfig.ConnConfig.Password = it.rolePassword
	applicationConfig.MaxConns = 1
	applicationPool, err := pgxpool.NewWithConfig(ctx, applicationConfig)
	if err != nil {
		t.Fatalf("open NOBYPASSRLS official artifact application pool: %v", err)
	}
	t.Cleanup(applicationPool.Close)
	servicePool := NewSessionPool(applicationPool)
	requestContext, release, err := AcquireRequestConn(ctx, applicationPool, SessionConfig{
		TenantID: "tenant-egueducation", InstitutionID: "inst-001", InstitutionName: "tenant-egueducation",
		ActorSubject: "official-artifact-integration", IsSuperAdmin: false,
	})
	if err != nil {
		t.Fatalf("bind NOBYPASSRLS tenant session: %v", err)
	}
	defer release()

	assertOfficialArtifactWriteBlocked(t, requestContext, servicePool, "published meeting update", `update education_meetings set title='tampered' where id=$1`, fixture.publishedMeetingID)
	assertOfficialArtifactWriteBlocked(t, requestContext, servicePool, "published meeting delete", `delete from education_meetings where id=$1`, fixture.publishedMeetingID)
	assertOfficialArtifactWriteBlocked(t, requestContext, servicePool, "evidenced meeting update", `update education_meetings set title='tampered' where id=$1`, fixture.evidencedMeetingID)
	assertOfficialArtifactWriteBlocked(t, requestContext, servicePool, "meeting delete with finalized evidence", `delete from education_meetings where id=$1`, fixture.evidencedMeetingID)
	assertOfficialArtifactWriteBlocked(t, requestContext, servicePool, "approved decision update", `update education_decisions set title='tampered' where id=$1`, fixture.decisionID)
	assertOfficialArtifactWriteBlocked(t, requestContext, servicePool, "approved decision delete", `delete from education_decisions where id=$1`, fixture.decisionID)
	assertOfficialArtifactWriteBlocked(t, requestContext, servicePool, "published meeting document update", `update education_meeting_documents set title='tampered' where id=$1`, fixture.documentID)
	assertOfficialArtifactWriteBlocked(t, requestContext, servicePool, "finalized minute delete", `delete from education_meeting_minutes where id=$1`, fixture.minuteID)
	assertOfficialArtifactWriteBlocked(t, requestContext, servicePool, "published resolution update", `update education_meeting_resolutions set title='tampered' where id=$1`, fixture.resolutionID)
	assertOfficialArtifactWriteBlocked(t, requestContext, servicePool, "evidenced meeting participant update", `update education_meeting_participants set full_name='tampered' where id=$1`, fixture.participantID)
	assertOfficialArtifactWriteBlocked(t, requestContext, servicePool, "evidenced meeting participant delete", `delete from education_meeting_participants where id=$1`, fixture.participantID)
	assertOfficialArtifactWriteBlocked(t, requestContext, servicePool, "evidenced meeting participant insert", `insert into education_meeting_participants (meeting_id, full_name, role_name, member_type, attendance_status, institution_id) values ($1, 'Tampered', 'Membru', 'membru', 'prezent', 'inst-001')`, fixture.evidencedMeetingID)
	assertOfficialArtifactWriteBlocked(t, requestContext, servicePool, "evidenced meeting vote update", `update education_meeting_votes set notes='tampered' where id=$1`, fixture.voteID)
	assertOfficialArtifactWriteBlocked(t, requestContext, servicePool, "evidenced meeting vote delete", `delete from education_meeting_votes where id=$1`, fixture.voteID)
	assertOfficialArtifactWriteBlocked(t, requestContext, servicePool, "evidenced meeting vote insert", `insert into education_meeting_votes (meeting_id, subject_title, agenda_order, decision_type, outcome, institution_id) values ($1, 'Tampered', 9, 'hotarare', 'adoptat', 'inst-001')`, fixture.evidencedMeetingID)
	assertOfficialArtifactWriteBlocked(t, requestContext, servicePool, "published managerial document delete", `delete from education_managerial_documents where id=$1`, fixture.managerialDocumentID)
	assertOfficialArtifactWriteBlocked(t, requestContext, servicePool, "published publication update", `update education_publications set entity_label='tampered' where id=$1`, fixture.publicationID)

	if _, err := servicePool.Exec(requestContext, `update education_decisions set status='published', publication_status='published' where id=$1`, fixture.decisionID); err != nil {
		t.Fatalf("approved-to-published decision transition must remain available: %v", err)
	}
	var decisionActor string
	var decisionPublishedAt time.Time
	if err := servicePool.QueryRow(requestContext, `select published_by_subject, published_at from education_decisions where id=$1`, fixture.decisionID).Scan(&decisionActor, &decisionPublishedAt); err != nil {
		t.Fatalf("read decision publication provenance: %v", err)
	}
	if decisionActor != "official-artifact-integration" || decisionPublishedAt.IsZero() {
		t.Fatalf("decision publication provenance=%q/%v, want server actor and timestamp", decisionActor, decisionPublishedAt)
	}
	assertOfficialArtifactWriteBlocked(t, requestContext, servicePool, "published decision mutation", `update education_decisions set title='tampered after publish' where id=$1`, fixture.decisionID)

	if _, err := servicePool.Exec(requestContext, `update education_publications set publication_status='retras', withdrawal_reason='Hotărâre înlocuită legal' where id=$1`, fixture.publicationID); err != nil {
		t.Fatalf("publicat-to-retras publication transition must remain available: %v", err)
	}
	var withdrawnActor string
	var withdrawnAt time.Time
	if err := servicePool.QueryRow(requestContext, `select withdrawn_by_subject, withdrawn_at from education_publications where id=$1`, fixture.publicationID).Scan(&withdrawnActor, &withdrawnAt); err != nil {
		t.Fatalf("read publication withdrawal provenance: %v", err)
	}
	if withdrawnActor != "official-artifact-integration" || withdrawnAt.IsZero() {
		t.Fatalf("publication withdrawal provenance=%q/%v, want server actor and timestamp", withdrawnActor, withdrawnAt)
	}
	assertOfficialArtifactWriteBlocked(t, requestContext, servicePool, "withdrawn publication mutation", `update education_publications set notes='tampered after withdrawal' where id=$1`, fixture.publicationID)

	if _, err := servicePool.Exec(requestContext, `update education_managerial_documents set document_status='archived', archival_reason='Înlocuit de versiune aprobată' where id=$1`, fixture.managerialDocumentID); err != nil {
		t.Fatalf("published-to-archived managerial document transition must remain available: %v", err)
	}
	var archivedActor string
	var archivedAt time.Time
	if err := servicePool.QueryRow(requestContext, `select archived_by_subject, archived_at from education_managerial_documents where id=$1`, fixture.managerialDocumentID).Scan(&archivedActor, &archivedAt); err != nil {
		t.Fatalf("read managerial archive provenance: %v", err)
	}
	if archivedActor != "official-artifact-integration" || archivedAt.IsZero() {
		t.Fatalf("managerial archive provenance=%q/%v, want server actor and timestamp", archivedActor, archivedAt)
	}
	assertOfficialArtifactWriteBlocked(t, requestContext, servicePool, "archived managerial document mutation", `update education_managerial_documents set notes='tampered after archive' where id=$1`, fixture.managerialDocumentID)

	if _, err := servicePool.Exec(requestContext, `update education_meetings set title='draft update is permitted' where id=$1`, fixture.draftMeetingID); err != nil {
		t.Fatalf("draft meeting update must remain available: %v", err)
	}
	if _, err := servicePool.Exec(requestContext, `update education_meetings set status='published' where id=$1`, fixture.draftMeetingID); err != nil {
		t.Fatalf("legitimate draft-to-published lifecycle transition must remain available: %v", err)
	}
	assertOfficialArtifactWriteBlocked(t, requestContext, servicePool, "newly published meeting mutation", `update education_meetings set title='tampered after publication' where id=$1`, fixture.draftMeetingID)

	var visibleForeign int
	if err := servicePool.QueryRow(requestContext, `select count(*) from education_meetings where id=$1`, fixture.foreignMeetingID).Scan(&visibleForeign); err != nil {
		t.Fatalf("query foreign meeting under RLS: %v", err)
	}
	if visibleForeign != 0 {
		t.Fatalf("foreign tenant official artifact is visible under RLS: count=%d", visibleForeign)
	}
}

type officialArtifactFixture struct {
	publishedMeetingID, evidencedMeetingID, draftMeetingID, foreignMeetingID uuid.UUID
	decisionID, documentID, minuteID, resolutionID, participantID, voteID    uuid.UUID
	managerialDocumentID, publicationID                                      uuid.UUID
}

func seedOfficialArtifactFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool) officialArtifactFixture {
	t.Helper()
	fixture := officialArtifactFixture{
		publishedMeetingID: uuid.New(), evidencedMeetingID: uuid.New(), draftMeetingID: uuid.New(), foreignMeetingID: uuid.New(),
		decisionID: uuid.New(), documentID: uuid.New(), minuteID: uuid.New(), resolutionID: uuid.New(), participantID: uuid.New(), voteID: uuid.New(),
		managerialDocumentID: uuid.New(), publicationID: uuid.New(),
	}
	dossierID := uuid.New()
	if _, err := pool.Exec(ctx, `
		insert into education_meetings (id, school_year, organism, title, meeting_type, status, quorum_required, participants_count, meeting_date, institution_id)
		values
			($1, '2031-2032', 'ca', 'Published official meeting', 'ordinary', 'published', 1, 1, current_date, 'inst-001'),
			($2, '2031-2032', 'ca', 'Meeting with finalized evidence', 'ordinary', 'held', 1, 1, current_date, 'inst-001'),
			($3, '2031-2032', 'ca', 'Draft meeting', 'ordinary', 'draft', 1, 0, current_date, 'inst-001'),
			($4, '2031-2032', 'ca', 'Foreign tenant meeting', 'ordinary', 'published', 1, 1, current_date, 'inst-balotesti')
	`, fixture.publishedMeetingID, fixture.evidencedMeetingID, fixture.draftMeetingID, fixture.foreignMeetingID); err != nil {
		t.Fatalf("seed meetings: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		insert into education_decisions (id, decision_code, school_year, organism, title, status, publication_status, decision_date, institution_id)
		values ($1, $2, '2031-2032', 'ca', 'Approved official decision', 'approved', 'internal', current_date, 'inst-001')
	`, fixture.decisionID, "OFFICIAL-DEC-"+uuid.NewString()); err != nil {
		t.Fatalf("seed approved decision: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		insert into education_meeting_participants (id, meeting_id, full_name, role_name, member_type, attendance_status, institution_id)
		values ($1, $2, 'Final evidence participant', 'Membru CA', 'membru', 'prezent', 'inst-001')
	`, fixture.participantID, fixture.evidencedMeetingID); err != nil {
		t.Fatalf("seed meeting participant before sealing evidence: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		insert into education_meeting_votes (id, meeting_id, subject_title, agenda_order, decision_type, outcome, institution_id)
		values ($1, $2, 'Published resolution vote', 2, 'hotarare', 'adoptat', 'inst-001')
	`, fixture.voteID, fixture.evidencedMeetingID); err != nil {
		t.Fatalf("seed resolution vote: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		insert into education_meeting_documents (id, meeting_id, document_type, title, publication_status, issued_on, institution_id)
		values ($1, $2, 'hotarare', 'Published official meeting document', 'publicat', current_date, 'inst-001')
	`, fixture.documentID, fixture.evidencedMeetingID); err != nil {
		t.Fatalf("seed published meeting document: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		insert into education_meeting_minutes (id, meeting_id, agenda_order, topic_title, discussion_summary, decision_summary, follow_up_status, institution_id)
		values ($1, $2, 1, 'Finalized minute', 'discussion', 'decision', 'inchis', 'inst-001')
	`, fixture.minuteID, fixture.evidencedMeetingID); err != nil {
		t.Fatalf("seed finalized meeting minute: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		insert into education_meeting_resolutions (id, meeting_id, vote_id, resolution_code, title, resolution_type, publication_status, anonymization_state, issued_on, institution_id)
		values ($1, $2, $3, $4, 'Published resolution', 'hotarare', 'publicat', 'finalizata', current_date, 'inst-001')
	`, fixture.resolutionID, fixture.evidencedMeetingID, fixture.voteID, "OFFICIAL-RES-"+uuid.NewString()); err != nil {
		t.Fatalf("seed published resolution: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		insert into education_managerial_dossiers (id, dossier_code, school_year, dossier_type, title, status, due_on, institution_id)
		values ($1, $2, '2031-2032', 'annual_plan', 'Official dossier', 'published', current_date, 'inst-001')
	`, dossierID, "OFFICIAL-DOS-"+uuid.NewString()); err != nil {
		t.Fatalf("seed official dossier: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		insert into education_managerial_documents (id, dossier_id, document_code, document_category, title, document_status, version_label, registered_on, institution_id)
		values ($1, $2, $3, 'hotarare', 'Published managerial document', 'published', 'v1', current_date, 'inst-001')
	`, fixture.managerialDocumentID, dossierID, "OFFICIAL-MDOC-"+uuid.NewString()); err != nil {
		t.Fatalf("seed published managerial document: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		insert into education_publications (id, publication_code, domain, entity_type, entity_label, publication_channel, publication_status, anonymization_status, institution_id)
		values ($1, $2, 'guvernanta', 'hotarare', 'Published publication', 'site_public', 'publicat', 'finalizata', 'inst-001')
	`, fixture.publicationID, "OFFICIAL-PUB-"+uuid.NewString()); err != nil {
		t.Fatalf("seed published education publication: %v", err)
	}
	return fixture
}

func grantOfficialArtifactApplicationAccess(t *testing.T, ctx context.Context, pool *pgxpool.Pool, roleName string) {
	t.Helper()
	role := tenantGrantQuoteIdentifier(roleName)
	for _, statement := range []string{
		"grant usage on schema public to " + role,
		"grant select, update, delete on education_meetings to " + role,
		"grant select, update, delete on education_decisions to " + role,
		"grant select, insert, update, delete on education_meeting_participants to " + role,
		"grant select, update, delete on education_meeting_documents to " + role,
		"grant select, update, delete on education_meeting_minutes to " + role,
		"grant select, insert, update, delete on education_meeting_votes to " + role,
		"grant select, update, delete on education_meeting_resolutions to " + role,
		"grant select, update, delete on education_managerial_documents to " + role,
		"grant select, update, delete on education_publications to " + role,
		"grant select, insert on app_entity_versions to " + role,
	} {
		if _, err := pool.Exec(ctx, statement); err != nil {
			t.Fatalf("grant official artifact application access: %v", err)
		}
	}
}

func assertOfficialArtifactWriteBlocked(t *testing.T, ctx context.Context, pool *SessionPool, name, query string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(ctx, query, args...); err == nil {
		t.Fatalf("%s unexpectedly succeeded", name)
	} else {
		var databaseError *pgconn.PgError
		if !errors.As(err, &databaseError) || databaseError.Code != "P0001" {
			t.Fatalf("%s error=%v, want immutable-record PostgreSQL P0001", name, err)
		}
	}
}
