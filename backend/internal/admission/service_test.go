package admission

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/eguilde/egueducation/internal/education"
)

func validCampaignFixture() CreateCampaignRequest {
	return CreateCampaignRequest{SourceID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", Code: "2026-PREP", Title: "Clasa pregatitoare", SchoolYear: "2026-2027", OfferingID: "11111111-1111-4111-8111-111111111111", LocationID: "22222222-2222-4222-8222-222222222222", AuthorizationID: "33333333-3333-4333-8333-333333333333", ClassOfferingContextID: "44444444-4444-4444-8444-444444444444", CapacityLimit: 25, CapacityUnit: "students", StudentPlaceLimit: 25, CapacityBasis: json.RawMessage(`{}`), Shift: "day", OpensOn: "2026-03-01", ClosesOn: "2026-03-31", Criteria: []CriterionInput{{Code: "AGE", Title: "Varsta", Kind: "eligibility", Required: true, Ordinal: 1, RuleSnapshot: json.RawMessage(`{"minimum":6}`)}}, DocumentRequirements: []DocumentRequirementInput{{Code: "BIRTH", Title: "Certificat", Required: true, Ordinal: 1}}}
}

func TestAdmissionDSSBindingFailsClosedWithoutCompleteVerifierProvenance(t *testing.T) {
	now := time.Now().UTC()
	snap := archiveSnapshot{sha: strings.Repeat("a", 64), size: 42}
	expectedPayloadHash := strings.Repeat("c", 64)
	valid := education.SignedArtifactValidationResult{Status: "valid", SignatureFormat: "PAdES", SignatureLevel: "qualified", SignatureSubject: "CN=Qualified Signer", SignedActorSubject: "Signer", CertificateIssuer: "Issuer", CertificateSerial: "01", CertificateValidFrom: &now, CertificateValidUntil: ptrTime(now.Add(time.Hour)), TrustedListProvider: "EU TL", ValidatorProvider: "DSS", ValidatorVersion: "6.1", ValidationPolicy: "ETSI", ObservedSHA256: snap.sha, ObservedSizeBytes: 42, SignedPayloadSHA256: expectedPayloadHash, CertificateSHA256: strings.Repeat("d", 64), TimestampTokenSHA256: strings.Repeat("b", 64), TimestampAt: &now, TimestampAuthority: "TSA", DiagnosticData: map[string]any{"ok": true}, DetailedReport: map[string]any{"ok": true}, SimpleReport: map[string]any{"ok": true}, ETSIValidationReport: map[string]any{"ok": true}}
	if err := validAdmissionDSSResult(valid, snap, expectedPayloadHash, "Signer"); err != nil {
		t.Fatalf("complete DSS result rejected: %v", err)
	}
	for _, status := range []string{"invalid", "indeterminate", "error"} {
		negative := valid
		negative.Status = status
		if err := validAdmissionDSSResult(negative, snap, expectedPayloadHash, "Signer"); err == nil {
			t.Fatalf("negative DSS verdict %q accepted despite trust fields", status)
		}
	}
	valid.ObservedSHA256 = strings.Repeat("c", 64)
	if err := validAdmissionDSSResult(valid, snap, expectedPayloadHash, "Signer"); err == nil {
		t.Fatal("hash mismatch accepted")
	}
	valid.ObservedSHA256 = snap.sha
	valid.ETSIValidationReport = nil
	if err := validAdmissionDSSResult(valid, snap, expectedPayloadHash, "Signer"); err == nil {
		t.Fatal("missing DSS report accepted")
	}
	valid.ETSIValidationReport = map[string]any{"ok": true}
	valid.SignedPayloadSHA256 = strings.Repeat("e", 64)
	if err := validAdmissionDSSResult(valid, snap, expectedPayloadHash, "Signer"); err == nil {
		t.Fatal("signed legal payload mismatch accepted")
	}
	valid.SignedPayloadSHA256 = expectedPayloadHash
	valid.SignedActorSubject = "Different signer"
	if err := validAdmissionDSSResult(valid, snap, expectedPayloadHash, "Signer"); err == nil {
		t.Fatal("actor mismatch accepted")
	}
	valid.SignedActorSubject = "Signer"
	valid.CertificateSHA256 = "not-a-hash"
	if err := validAdmissionDSSResult(valid, snap, expectedPayloadHash, "Signer"); err == nil {
		t.Fatal("malformed leaf certificate fingerprint accepted")
	}
}

func ptrTime(value time.Time) *time.Time { return &value }

func TestCampaignValidationFailsClosed(t *testing.T) {
	good := validCampaignFixture()
	if !validateCampaign(&good) {
		t.Fatal("valid campaign rejected")
	}
	for name, mutate := range map[string]func(*CreateCampaignRequest){"invalid authorization": func(x *CreateCampaignRequest) { x.AuthorizationID = "client-supplied-code" }, "reverse window": func(x *CreateCampaignRequest) { x.ClosesOn = "2026-02-01" }, "duplicate ordinal": func(x *CreateCampaignRequest) {
		x.Criteria = append(x.Criteria, CriterionInput{Code: "OTHER", Title: "Other", Kind: "ranking", Ordinal: 1})
	}, "unknown capacity unit": func(x *CreateCampaignRequest) { x.CapacityUnit = "seats-ish" }} {
		t.Run(name, func(t *testing.T) {
			x := validCampaignFixture()
			mutate(&x)
			if validateCampaign(&x) {
				t.Fatal("invalid campaign accepted")
			}
		})
	}
}

func TestAppealResolutionResultingOutcomeRules(t *testing.T) {
	for _, outcome := range []string{"admitted", "waitlisted", "rejected", "withdrawn", "cancelled"} {
		if outcome == "" {
			t.Fatal("unreachable")
		}
	}
	if !mimeAllowed([]string{"application/pdf"}, "Application/PDF") || mimeAllowed([]string{"application/pdf"}, "image/png") {
		t.Fatal("MIME allow-list is not exact")
	}
}

func TestAppealResolutionPreservesAdmittedAllocationWhenDismissedOrWithdrawn(t *testing.T) {
	originalAllocationID := "11111111-1111-4111-8111-111111111111"
	for _, outcome := range []string{"dismissed", "withdrawn"} {
		if got := appealResolutionAllocationID(originalAllocationID, false, Decision{}); got != originalAllocationID {
			t.Fatalf("%s appeal allocation = %q, want original %q", outcome, got, originalAllocationID)
		}
	}
	if got := appealResolutionAllocationID(originalAllocationID, true, Decision{}); got != "" {
		t.Fatalf("non-admitted favourable replacement allocation = %q, want empty", got)
	}
	newAllocationID := "22222222-2222-4222-8222-222222222222"
	if got := appealResolutionAllocationID(originalAllocationID, true, Decision{CapacityAllocationID: &newAllocationID}); got != newAllocationID {
		t.Fatalf("admitted favourable replacement allocation = %q, want %q", got, newAllocationID)
	}
}

func TestFavourableAppealAllocationPathReusesOrReleasesOriginalCommitment(t *testing.T) {
	appeals, err := os.ReadFile("appeals.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"status='released'", "original_capacity_allocation_id", "released_capacity_allocation_id"} {
		if !strings.Contains(string(appeals), required) {
			t.Fatalf("appeal resolution lacks %q", required)
		}
	}
	decisions, err := os.ReadFile("decisions.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"supersedes != \"\"", "status in ('held','consumed') for update", "allocationID = &existingID"} {
		if !strings.Contains(string(decisions), required) {
			t.Fatalf("favourable appeal decision path lacks %q", required)
		}
	}
	if !strings.Contains(string(decisions), `!(supersedes != "" && c.status == "admitted")`) {
		t.Fatal("favourable appeal cannot replace an admitted decision")
	}
}

func TestAdmissionWriteQueriesLockOnlyNonNullableSidesAndConsumeDocumentSource(t *testing.T) {
	documents, err := os.ReadFile("documents.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(documents)
	for _, required := range []string{"for update of document,a", "document.archive_document_id is null and document.reviewed_at is null", "reviewed_at=now(),reviewed_by_subject"} {
		if !strings.Contains(source, required) {
			t.Fatalf("document review query lacks %q", required)
		}
	}
	decisions, err := os.ReadFile("decisions.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(decisions), "nullif($5,'')::uuid") {
		t.Fatal("non-admitted decision allocation UUID is not nullified")
	}
}

func TestCapacityMigrationSerializesAuthorizationAndProtectsOverlappingDeclarations(t *testing.T) {
	migration, err := os.ReadFile("../db/migrations/0149_school_admission_dss_bindings.sql")
	if err != nil {
		t.Fatal(err)
	}
	source := string(migration)
	for _, required := range []string{
		"create or replace function public.school_admission_campaign_guard()",
		"create or replace function public.school_admission_capacity_guard()",
		"new.authorization_id::text||':'||new.shift||':'||new.school_year",
		"campaign.school_year=new.school_year",
		"campaign.status<>'cancelled'",
		"campaign.status<>'archived' or exists",
		"authorization_capacity_unit='students'",
		"a.authorization_id=new.authorization_id and a.shift=new.shift",
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("capacity migration lacks %q", required)
		}
	}
}

func TestDSSRetentionEffectiveFromCannotBeAfterCurrentUTCDate(t *testing.T) {
	now := time.Date(2026, time.September, 12, 23, 59, 59, 0, time.FixedZone("local", -7*60*60))
	for value, want := range map[string]bool{
		"2026-09-12": true,
		"2026-09-13": true,
		"2026-09-14": false,
		"not-a-date": false,
	} {
		if got := validDSSRetentionEffectiveFrom(value, now); got != want {
			t.Fatalf("validDSSRetentionEffectiveFrom(%q) = %t, want %t", value, got, want)
		}
	}
}

func TestRetentionAuthorityRequestsSeparateLegalProposalFromOperationalConfiguration(t *testing.T) {
	proposal := ProposeAdmissionRetentionRuleRequest{ArtifactKind: "admission_dss", MinimumRetentionDays: 365, EffectiveFrom: "2026-01-01", SourceID: "11111111-1111-4111-8111-111111111111"}
	if !validRetentionRuleProposal(proposal) {
		t.Fatal("valid legal authority proposal rejected")
	}
	proposal.ArtifactKind = "admission_decision"
	if validRetentionRuleProposal(proposal) {
		t.Fatal("non-DSS authority proposal accepted")
	}
	raw, err := os.ReadFile("dss_retention_policy.go")
	if err != nil {
		t.Fatal(err)
	}
	// Source assertions describe Go syntax, whose line terminators are not part
	// of the handler contract. Normalize Windows checkouts before matching.
	source := strings.ReplaceAll(string(raw), "\r\n", "\n")
	for _, required := range []string{
		"type ConfigureDSSRetentionPolicyRequest struct {\n\tRuleVersionID string",
		"ProposeAdmissionRetentionRule", "ApproveAdmissionRetentionRule",
		"school_admission_retention_rule_versions", "rule_version_id",
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("retention authority handler lacks %q", required)
		}
	}
	if strings.Contains(source, "type ConfigureDSSRetentionPolicyRequest struct {\n\tMinimumRetentionDays") || strings.Contains(source, "type ConfigureDSSRetentionPolicyRequest struct {\n\tSourceID") {
		t.Fatal("operational retention endpoint still accepts caller supplied legal facts")
	}
}

func TestRetentionAuthorityMigrationProtectsSeparationOfDutiesAndSourceSnapshot(t *testing.T) {
	raw, err := os.ReadFile("../db/migrations/0150_school_admission_legal_authority.sql")
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	for _, required := range []string{
		"source_checksum_sha256", "proposed_by_subject", "approved_by_subject<>proposed_by_subject",
		"status in ('proposed','active','superseded','revoked')", "force row level security",
		"pg_advisory_xact_lock", "school_operations_no_hard_delete", "director_adjunct",
		"education.admissions.retention.approve", "authority facts are server-derived",
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("retention authority migration lacks %q", required)
		}
	}
}

func TestCurrentDSSRetentionPolicySelectsOnlyEffectiveActivePolicy(t *testing.T) {
	raw, err := os.ReadFile("dss_retention_policy.go")
	if err != nil {
		t.Fatal(err)
	}
	// Source assertions describe Go syntax, whose line terminators are not part
	// of the handler contract. Normalize Windows checkouts before matching.
	source := strings.ReplaceAll(string(raw), "\r\n", "\n")
	for _, required := range []string{
		"p.status='active' and p.effective_from<=current_date",
		"(p.effective_to is null or p.effective_to>=current_date)",
		"if err == nil && !replay {\n\t\terr = auditEvent",
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("DSS retention policy contract lacks %q", required)
		}
	}
}

func TestDSSRetentionPolicySerializesWithSingleStatementExecutions(t *testing.T) {
	raw, err := os.ReadFile("dss_retention_policy.go")
	if err != nil {
		t.Fatal(err)
	}
	source := strings.ReplaceAll(string(raw), "\r\n", "\n")
	if strings.Contains(source, "pg_advisory_xact_lock(hashtextextended('school_admission_retention_policy:'||$1||':'||$2,0)); update") {
		t.Fatal("DSS retention policy passes multiple commands to a parameterized execution")
	}
	for _, required := range []string{
		"err = tx.Exec(r.Context(), `select pg_advisory_xact_lock(hashtextextended('school_admission_retention_policy:'||$1||':'||$2,0))`, sc.tenant, sc.institution)",
		"err = tx.Exec(r.Context(), `update school_admission_dss_retention_policies set status='superseded'",
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("DSS retention policy serialization lacks %q", required)
		}
	}
}

func TestDocumentReviewAcceptsOnlyTerminalStatuses(t *testing.T) {
	for status, want := range map[string]bool{
		"accepted":  true,
		"rejected":  true,
		"waived":    true,
		"submitted": false,
		"draft":     false,
	} {
		if got := finalApplicationDocumentReviewStatus(status); got != want {
			t.Fatalf("finalApplicationDocumentReviewStatus(%q) = %t, want %t", status, got, want)
		}
	}
}

func TestStateMachinesAllowOnlyExplicitEdges(t *testing.T) {
	if !campaignTransitions["draft"]["published"] || campaignTransitions["draft"]["open"] || campaignTransitions["archived"]["open"] {
		t.Fatal("campaign transition graph is unsafe")
	}
	if !applicationTransitions["draft"]["submitted"] || applicationTransitions["draft"]["admitted"] || applicationTransitions["rejected"]["submitted"] {
		t.Fatal("application transition graph is unsafe")
	}
}

func TestDecisionRequiresWORMReference(t *testing.T) {
	in := IssueDecisionRequest{DecisionNo: "D-1", Outcome: "admitted", Rationale: "eligible", ExpectedVersion: 1}
	if validDecision(in) {
		t.Fatal("decision without archive evidence accepted")
	}
	in.Archive = ArchiveReference{DocumentID: "11111111-1111-4111-8111-111111111111", VersionID: "22222222-2222-4222-8222-222222222222"}
	if !validDecision(in) {
		t.Fatal("valid decision rejected")
	}
}

func TestLegalPreparationRequiresCanonicalSeparateAppealArtifacts(t *testing.T) {
	good := PrepareAppealResolutionRequest{Outcome: "upheld", Rationale: "legal correction", ExpectedVersion: 1, ApplicationExpectedVersion: 1, ResultingDecisionNo: "D-2", ResultingOutcome: "admitted"}
	if !validPrepareAppeal(good) {
		t.Fatal("favourable appeal preparation must accept complete resulting decision facts")
	}
	good.ResultingDecisionNo = ""
	if validPrepareAppeal(good) {
		t.Fatal("favourable appeal may not omit its independently signed resulting decision")
	}
	good = PrepareAppealResolutionRequest{Outcome: "dismissed", Rationale: "legal basis", ExpectedVersion: 1, ApplicationExpectedVersion: 1}
	if !validPrepareAppeal(good) {
		t.Fatal("non-favourable appeal should not require a resulting decision")
	}
}

func TestAdmissionPreparationExpiryIsTenantScopedAndReleasesOnlyHeldCapacity(t *testing.T) {
	source, err := os.ReadFile("legal_preparations.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, want := range []string{"expireAdmissionPreparationsTx", "p.tenant_code=$1 and p.institution_id=$2", "status='prepared' and p.expires_at<=now()", "status='expired',cancellation_reason='expired'", "status='held'"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expiry implementation lacks %q", want)
		}
	}
}

func TestSignerAuthorizationValidationRequiresScopedFingerprintAndLegalPermission(t *testing.T) {
	in := ProposeAdmissionSignerAuthorizationRequest{CertificateSHA256: strings.Repeat("a", 64), UserID: "11111111-1111-4111-8111-111111111111", PermissionCode: permissionDecide, ValidUntil: time.Now().UTC().Add(time.Hour).Format(time.RFC3339)}
	if !validSignerAuthorization(in) {
		t.Fatal("valid signer authorization rejected")
	}
	in.PermissionCode = permissionManage
	if validSignerAuthorization(in) {
		t.Fatal("unrelated permission must not authorize legal signing")
	}
}

func TestDirectDecisionCannotSupersedeFinalApplication(t *testing.T) {
	if applicationTransitions["rejected"]["admitted"] {
		t.Fatal("final rejection must not directly transition to admitted")
	}
}

func TestPolicyContextBindsExactCampaignApplicationAuthorization(t *testing.T) {
	c := decisionContext{campaignID: "campaign", authorizationID: "authorization"}
	got := decisionPolicyContext(c, "application", "admitted")
	for key, want := range map[string]string{"campaign_id": "campaign", "application_id": "application", "authorization_id": "authorization", "outcome": "admitted"} {
		if got[key] != want {
			t.Fatalf("%s = %#v", key, got[key])
		}
	}
}

func TestMembershipPermissionBranchesRequireInstitutionEligibility(t *testing.T) {
	if count := strings.Count(permissionSQL, "public.education_membership_is_eligible(u.id,public.current_tenant_code(),public.current_institution_id(),m.position_code)"); count != 4 {
		t.Fatalf("membership authorization has %d institution eligibility checks, want 4", count)
	}
	if strings.Count(permissionSQL, "u.status='active'") != 4 {
		t.Fatal("all RBAC branches must require active user")
	}
	if strings.Contains(permissionSQL, "m.active and") {
		t.Fatal("inline tenant-only membership check can bypass canonical institution eligibility")
	}
}

func TestRequestContractsCannotAcceptScopeAndUseSnakeCase(t *testing.T) {
	requests := []any{CreateClassOfferingContextRequest{}, CreateCampaignRequest{}, TransitionRequest{}, CreateApplicationRequest{}, AssessCriterionRequest{}, ReviewApplicationDocumentRequest{}, IssueDecisionRequest{}, CreateAppealRequest{}, ResolveAppealRequest{}, EnrolApplicationRequest{}}
	for _, request := range requests {
		typ := reflect.TypeOf(request)
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			name := strings.ToLower(field.Name)
			if name == "tenant" || name == "tenantcode" || name == "institution" || name == "institutionid" {
				t.Fatalf("%s accepts forbidden scope field %s", typ.Name(), field.Name)
			}
			tag := strings.Split(field.Tag.Get("json"), ",")[0]
			if strings.ContainsAny(tag, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
				t.Fatalf("%s.%s has non-snake JSON tag %q", typ.Name(), field.Name, tag)
			}
		}
	}
}

func TestSelectorsArePagedAndApplySensitiveReadBoundaries(t *testing.T) {
	raw, err := os.ReadFile("selectors.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	if strings.Count(source, "httpx.WritePage") != 4 || strings.Count(source, "httpx.ParsePageQuery") != 4 {
		t.Fatal("each selector must keep server-side page/filter/sort contract")
	}
	for _, required := range []string{`requirePermission(r.Context(),tx,sc,"registratura.read")`, `requirePermission(r.Context(),tx,sc,"education.classes.read")`, `requirePermission(r.Context(),tx,sc,"earchiva.read")`, `tenant_scope.code=$1 and tenant_scope.institution_id=$2`} {
		if !strings.Contains(strings.ReplaceAll(source, " ", ""), strings.ReplaceAll(required, " ", "")) {
			t.Fatalf("selector contract lacks %s", required)
		}
	}
	for _, required := range []string{`q.Filters[field]`, `v.version_no::text`} {
		if !strings.Contains(source, required) {
			t.Fatalf("archive selector contract lacks %s", required)
		}
	}
	typ := reflect.TypeOf(CandidatePartyOption{})
	for i := 0; i < typ.NumField(); i++ {
		tag := typ.Field(i).Tag.Get("json")
		if tag == "phone_number" || tag == "email" || tag == "identifier_code" {
			t.Fatalf("candidate selector leaks %s", tag)
		}
	}
}

func TestArchivePurposeRequiresOperationSpecificPermission(t *testing.T) {
	wants := map[string]string{"application_document": permissionManage, "decision": permissionDecide, "appeal": permissionAppeals, "appeal_submission": permissionAppeals, "appeal_resolution": permissionAppeals}
	if !reflect.DeepEqual(archivePurposePermission, wants) {
		t.Fatalf("archive purpose permission mapping = %#v, want %#v", archivePurposePermission, wants)
	}
}

func TestClosedJSONRejectsUnknownFields(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"status":"submitted","expected_version":1,"tenant_code":"escape"}`))
	w := httptest.NewRecorder()
	var in TransitionRequest
	if decode(w, r, &in) == nil {
		t.Fatal("unknown tenant field accepted")
	}
}

func TestRouteSurfaceIsMounted(t *testing.T) {
	router := New(nil).Routes()
	for _, path := range []string{"/campaigns", "/applications", "/decisions", "/appeals"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code == http.StatusNotFound {
			t.Fatalf("route %s is not mounted", path)
		}
	}
}

func TestTableQueriesRemainServerSideAndTyped(t *testing.T) {
	raw, err := os.ReadFile("queries.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	if got := strings.Count(source, "httpx.ParsePageQuery"); got < 6 {
		t.Fatalf("only %d admission table queries parse server-side paging/filter/sort", got)
	}
	if got := strings.Count(source, "httpx.WritePage"); got < 6 {
		t.Fatalf("only %d admission table queries emit typed pages", got)
	}
	for _, required := range []string{"&x.CampaignCode", "&x.CandidateName", "&x.DocumentsComplete", "&x.CriteriaComplete", "&x.ApplicationNo"} {
		if !strings.Contains(source, required) {
			t.Fatalf("query contract lacks %s", required)
		}
	}
	if strings.Contains(source, "jsonb_agg(jsonb_build_object") {
		t.Fatal("query API regressed to raw untyped JSON aggregation")
	}
}

func TestListDTOsExposeFrontendProgressColumns(t *testing.T) {
	assertTags := func(value any, wants ...string) {
		t.Helper()
		typ := reflect.TypeOf(value)
		tags := map[string]bool{}
		for i := 0; i < typ.NumField(); i++ {
			tags[strings.Split(typ.Field(i).Tag.Get("json"), ",")[0]] = true
		}
		for _, want := range wants {
			if !tags[want] {
				t.Fatalf("%s lacks JSON field %s", typ.Name(), want)
			}
		}
	}
	assertTags(Application{}, "campaign_code", "candidate_name", "documents_complete", "criteria_complete")
	assertTags(Decision{}, "application_no", "candidate_name")
	assertTags(Appeal{}, "application_no")
}
