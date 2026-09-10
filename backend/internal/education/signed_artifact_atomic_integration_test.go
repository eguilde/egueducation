//go:build integration

package education

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eguilde/egueducation/internal/db"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestSignedArtifactMutationsAreAtomic proves that the append-only evidence
// ledger cannot contain an orphan evidence or validation record if its
// companion legal audit event (or the initial validation) cannot be written.
func TestSignedArtifactMutationsAreAtomic(t *testing.T) {
	it := newGovernanceIntegrationDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	admin := openGovernanceIntegrationPool(t, ctx, it.databaseConfig)
	defer admin.Close()
	if err := db.Migrate(ctx, admin); err != nil {
		t.Fatalf("migrate signed-artifact atomic database: %v", err)
	}
	grantGovernanceIntegrationAccess(t, ctx, admin, it.roleName)
	fixture := seedGovernanceAuthorizationFixture(t, ctx, admin)
	for _, permission := range []string{signedArtifactManagePermission, signedArtifactValidatePermission} {
		if _, err := admin.Exec(ctx, `insert into app_user_permissions(user_id,permission_code,tenant_code) values($1::uuid,$2,$3) on conflict do nothing`, fixture.memberUserID, permission, fixture.tenantA); err != nil {
			t.Fatalf("grant %s: %v", permission, err)
		}
	}
	artifactID, documentID, versionID := seedAtomicSignedArtifactSource(t, ctx, admin, fixture.institutionA)
	service := NewService(db.NewSessionPool(it.readerPool))
	bound, release := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	defer release()
	payload := signedArtifactSubmitPayload(artifactID, documentID, versionID)

	installSignedArtifactValidationFailureTrigger(t, ctx, admin)
	failedSubmit := httptest.NewRecorder()
	service.SubmitSignedArtifactEvidence(failedSubmit, signedArtifactMutationRequest(bound, fixture, http.MethodPost, payload, ""))
	if failedSubmit.Code != http.StatusInternalServerError {
		t.Fatalf("submit with failed initial validation status=%d body=%s", failedSubmit.Code, failedSubmit.Body.String())
	}
	assertSignedArtifactMutationCounts(t, bound, service, 0, 0, 0)
	dropSignedArtifactValidationFailureTrigger(t, ctx, admin)

	created := httptest.NewRecorder()
	service.SubmitSignedArtifactEvidence(created, signedArtifactMutationRequest(bound, fixture, http.MethodPost, payload, ""))
	if created.Code != http.StatusCreated {
		t.Fatalf("baseline signed evidence submit status=%d body=%s", created.Code, created.Body.String())
	}
	var evidenceID string
	if err := service.pool.QueryRow(bound, `select id::text from education_signed_artifact_evidence where artifact_id=$1::uuid`, artifactID).Scan(&evidenceID); err != nil {
		t.Fatalf("load submitted evidence: %v", err)
	}
	assertSignedArtifactMutationCounts(t, bound, service, 1, 1, 1)

	ConfigureSignedArtifactVerifier(deterministicSignedArtifactVerifier{})
	defer ConfigureSignedArtifactVerifier(nil)
	installSignedArtifactAuditFailureTrigger(t, ctx, admin)
	failedRevalidation := httptest.NewRecorder()
	service.RevalidateSignedArtifactEvidence(failedRevalidation, signedArtifactMutationRequest(bound, fixture, http.MethodPost, "", evidenceID))
	if failedRevalidation.Code != http.StatusInternalServerError {
		t.Fatalf("revalidate with failed audit status=%d body=%s", failedRevalidation.Code, failedRevalidation.Body.String())
	}
	// The pending validation and the original submit audit remain, but the
	// failed revalidation created neither a new validation nor a false audit.
	assertSignedArtifactMutationCounts(t, bound, service, 1, 1, 1)
}

func seedAtomicSignedArtifactSource(t *testing.T, ctx context.Context, admin *pgxpool.Pool, institutionID string) (artifactID, documentID, versionID string) {
	t.Helper()
	artifactID, documentID, versionID = uuid.NewString(), uuid.NewString(), uuid.NewString()
	const sourceSHA256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if _, err := admin.Exec(ctx, `
		insert into education_decisions(id,decision_code,school_year,organism,title,status,publication_status,decision_date,institution_id)
		values($1::uuid,$2,'2026-2027','ca','Atomic signature evidence','approved','internal',current_date,$3)
	`, artifactID, "DEC-ATOMIC-"+strings.ReplaceAll(artifactID[:8], "-", ""), institutionID); err != nil {
		t.Fatalf("seed signed-artifact decision: %v", err)
	}
	if _, err := admin.Exec(ctx, `
		insert into archive_documents(id,institution_id,title,original_file_name,mime_type,source_kind,status,current_version_no)
		values($1::uuid,$2,'Atomic signed source','atomic.pdf','application/pdf','upload','ready',1)
	`, documentID, institutionID); err != nil {
		t.Fatalf("seed signed-artifact document: %v", err)
	}
	if _, err := admin.Exec(ctx, `
		insert into archive_document_versions(id,document_id,institution_id,version_no,mime_type,title,bucket_name,object_key,hash_sha256,status,source_bucket,source_object_key,source_sha256)
		values($1::uuid,$2::uuid,$3,1,'application/pdf','Atomic signed source','earhive','atomic.pdf',$4,'active','earhive','atomic.pdf',$4)
	`, versionID, documentID, institutionID, sourceSHA256); err != nil {
		t.Fatalf("seed signed-artifact version: %v", err)
	}
	return artifactID, documentID, versionID
}

func signedArtifactSubmitPayload(artifactID, documentID, versionID string) string {
	return `{"artifact_type":"decision","artifact_id":"` + artifactID + `","signature_format":"PAdES","signature_level":"advanced","signature_subject":"Atomic test signer","certificate_issuer":"Atomic test issuer","certificate_serial":"atomic-01","certificate_valid_from":"2026-01-01T00:00:00Z","certificate_valid_until":"2027-01-01T00:00:00Z","storage_document_id":"` + documentID + `","storage_version_id":"` + versionID + `"}`
}

func signedArtifactMutationRequest(ctx context.Context, fixture governanceAuthorizationFixture, method, payload, evidenceID string) *http.Request {
	base := requestWithContext(ctx, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	request := httptest.NewRequest(method, "http://education.test", strings.NewReader(payload)).WithContext(base.Context())
	if evidenceID == "" {
		return request
	}
	route := chi.NewRouteContext()
	route.URLParams.Add("evidenceID", evidenceID)
	return request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, route))
}

func assertSignedArtifactMutationCounts(t *testing.T, ctx context.Context, service *Service, evidence, validations, audits int) {
	t.Helper()
	var gotEvidence, gotValidations, gotAudits int
	if err := service.pool.QueryRow(ctx, `select count(*) from education_signed_artifact_evidence`).Scan(&gotEvidence); err != nil {
		t.Fatalf("count signed evidence: %v", err)
	}
	if err := service.pool.QueryRow(ctx, `select count(*) from education_signed_artifact_validations`).Scan(&gotValidations); err != nil {
		t.Fatalf("count signed validations: %v", err)
	}
	if err := service.pool.QueryRow(ctx, `select count(*) from app_audit_log where action like 'education.signatures.%'`).Scan(&gotAudits); err != nil {
		t.Fatalf("count signed audit events: %v", err)
	}
	if gotEvidence != evidence || gotValidations != validations || gotAudits != audits {
		t.Fatalf("atomic signed mutation counts evidence=%d validations=%d audits=%d; want %d/%d/%d", gotEvidence, gotValidations, gotAudits, evidence, validations, audits)
	}
}

func installSignedArtifactValidationFailureTrigger(t *testing.T, ctx context.Context, admin *pgxpool.Pool) {
	t.Helper()
	if _, err := admin.Exec(ctx, `
		create or replace function integration_fail_signed_validation()
		returns trigger language plpgsql as $$ begin raise exception 'forced signed validation failure'; end; $$
	`); err != nil {
		t.Fatalf("install signed validation failure function: %v", err)
	}
	if _, err := admin.Exec(ctx, `drop trigger if exists integration_fail_signed_validation on education_signed_artifact_validations`); err != nil {
		t.Fatalf("drop prior signed validation failure trigger: %v", err)
	}
	if _, err := admin.Exec(ctx, `create trigger integration_fail_signed_validation before insert on education_signed_artifact_validations for each row execute function integration_fail_signed_validation()`); err != nil {
		t.Fatalf("install signed validation failure trigger: %v", err)
	}
}

func dropSignedArtifactValidationFailureTrigger(t *testing.T, ctx context.Context, admin *pgxpool.Pool) {
	t.Helper()
	if _, err := admin.Exec(ctx, `drop trigger if exists integration_fail_signed_validation on education_signed_artifact_validations`); err != nil {
		t.Fatalf("drop signed validation failure trigger: %v", err)
	}
}

func installSignedArtifactAuditFailureTrigger(t *testing.T, ctx context.Context, admin *pgxpool.Pool) {
	t.Helper()
	if _, err := admin.Exec(ctx, `
		create or replace function integration_fail_signed_audit()
		returns trigger language plpgsql as $$ begin raise exception 'forced signed audit failure'; end; $$
	`); err != nil {
		t.Fatalf("install signed audit failure function: %v", err)
	}
	if _, err := admin.Exec(ctx, `drop trigger if exists integration_fail_signed_audit on app_audit_log`); err != nil {
		t.Fatalf("drop prior signed audit failure trigger: %v", err)
	}
	if _, err := admin.Exec(ctx, `create trigger integration_fail_signed_audit before insert on app_audit_log for each row execute function integration_fail_signed_audit()`); err != nil {
		t.Fatalf("install signed audit failure trigger: %v", err)
	}
	t.Cleanup(func() { dropSignedArtifactAuditFailureTrigger(t, ctx, admin) })
}

func dropSignedArtifactAuditFailureTrigger(t *testing.T, ctx context.Context, admin *pgxpool.Pool) {
	t.Helper()
	if _, err := admin.Exec(ctx, `drop trigger if exists integration_fail_signed_audit on app_audit_log`); err != nil {
		t.Fatalf("drop signed audit failure trigger: %v", err)
	}
}
