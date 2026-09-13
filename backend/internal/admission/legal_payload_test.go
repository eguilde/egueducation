package admission

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestLegalPreparationBase64PreservesExactCanonicalBytes(t *testing.T) {
	canonical := []byte(`{"schema_version":"v1","rationale":"<școală>"}`)
	prepared := preparationResponse(AdmissionLegalPreparation{CanonicalPayload: canonical, ResultingDecisionPayload: canonical})
	if prepared.CanonicalPayloadBase64 != base64.RawStdEncoding.EncodeToString(canonical) || prepared.ResultingDecisionPayloadBase64 != base64.RawStdEncoding.EncodeToString(canonical) {
		t.Fatal("canonical transport must be byte exact and independent of JSON re-serialization")
	}
}

func TestAdmissionDecisionLegalPayloadIsDeterministicAndBindsLegalFacts(t *testing.T) {
	ranking := 9.5
	deadline := " 2026-10-01 "
	in := AdmissionDecisionLegalPayloadInput{
		TenantCode: " tenant-a ", InstitutionID: " institution-a ", ArtifactID: "11111111-1111-4111-8111-111111111111",
		ApplicationID: "22222222-2222-4222-8222-222222222222", CampaignID: "33333333-3333-4333-8333-333333333333",
		OfferingID: "44444444-4444-4444-8444-444444444444", LocationID: "55555555-5555-4555-8555-555555555555",
		AuthorizationID: "66666666-6666-4666-8666-666666666666", ClassOfferingContextID: "77777777-7777-4777-8777-777777777777",
		DecisionNo: " DEC-1 ", Outcome: " admitted ", Rationale: " legal basis ", RankingValue: &ranking, AppealDeadline: &deadline,
		PolicyEvaluationV2ID: "88888888-8888-4888-8888-888888888888", ActorSubject: " actor-a ", DecidedAt: time.Date(2026, 9, 12, 10, 30, 0, 123, time.FixedZone("EEST", 3*60*60)),
		CapacityAllocationID: "99999999-9999-4999-8999-999999999999", SupersedesDecisionID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
	}
	payload, err := BuildAdmissionDecisionLegalPayload(in)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := payload.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	want := `{"schema_version":"egueducation.admission.decision.v1","artifact_kind":"admission_decision","scope":{"tenant_code":"tenant-a","institution_id":"institution-a"},"artifact_id":"11111111-1111-4111-8111-111111111111","application_id":"22222222-2222-4222-8222-222222222222","campaign_id":"33333333-3333-4333-8333-333333333333","offering_id":"44444444-4444-4444-8444-444444444444","location_id":"55555555-5555-4555-8555-555555555555","authorization_id":"66666666-6666-4666-8666-666666666666","class_offering_context_id":"77777777-7777-4777-8777-777777777777","decision_no":"DEC-1","outcome":"admitted","rationale":"legal basis","ranking_value":9.5,"appeal_deadline":"2026-10-01","policy_evaluation_v2_id":"88888888-8888-4888-8888-888888888888","actor_subject":"actor-a","decided_at":"2026-09-12T07:30:00.000000123Z","capacity_allocation_id":"99999999-9999-4999-8999-999999999999","supersedes_decision_id":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"}`
	if string(canonical) != want {
		t.Fatalf("canonical payload changed\n got: %s\nwant: %s", canonical, want)
	}
	first, err := AdmissionDecisionLegalPayloadSHA256(payload)
	if err != nil {
		t.Fatal(err)
	}
	second, _ := AdmissionDecisionLegalPayloadSHA256(payload)
	if first != second || len(first) != 64 {
		t.Fatalf("non-deterministic digest: %q / %q", first, second)
	}
	payload.Scope.TenantCode = "tenant-b"
	changed, _ := AdmissionDecisionLegalPayloadSHA256(payload)
	if changed == first {
		t.Fatal("tenant scope did not affect legal payload digest")
	}
}

func TestAdmissionAppealResolutionLegalPayloadBindsReplacementAndAllocationLinks(t *testing.T) {
	in := AdmissionAppealResolutionLegalPayloadInput{
		TenantCode: "tenant-a", InstitutionID: "institution-a", ArtifactID: "11111111-1111-4111-8111-111111111111",
		AppealID: "22222222-2222-4222-8222-222222222222", ApplicationID: "33333333-3333-4333-8333-333333333333", OriginalDecisionID: "44444444-4444-4444-8444-444444444444",
		Outcome: "upheld", Rationale: "new evidence", ResultingOutcome: "admitted", PolicyEvaluationV2ID: "55555555-5555-4555-8555-555555555555",
		ActorSubject: "actor-a", ResolvedAt: time.Date(2026, 9, 12, 8, 0, 0, 0, time.UTC), ResultingDecisionID: "66666666-6666-4666-8666-666666666666",
		SupersedesDecisionID: "44444444-4444-4444-8444-444444444444", CapacityAllocationID: "77777777-7777-4777-8777-777777777777",
		OriginalCapacityAllocationID: "88888888-8888-4888-8888-888888888888", ReleasedCapacityAllocationID: "88888888-8888-4888-8888-888888888888",
	}
	payload, err := BuildAdmissionAppealResolutionLegalPayload(in)
	if err != nil {
		t.Fatal(err)
	}
	base, err := AdmissionAppealResolutionLegalPayloadSHA256(payload)
	if err != nil || len(base) != 64 {
		t.Fatalf("appeal digest = %q, %v", base, err)
	}
	mutations := []func(*AdmissionAppealResolutionLegalPayload){
		func(x *AdmissionAppealResolutionLegalPayload) { x.ActorSubject = "actor-b" },
		func(x *AdmissionAppealResolutionLegalPayload) { x.ResultingDecisionID = nil },
		func(x *AdmissionAppealResolutionLegalPayload) { x.ReleasedCapacityAllocationID = nil },
		func(x *AdmissionAppealResolutionLegalPayload) {
			x.PolicyEvaluationV2ID = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
		},
	}
	for i, mutate := range mutations {
		changed := payload
		mutate(&changed)
		digest, _ := AdmissionAppealResolutionLegalPayloadSHA256(changed)
		if digest == base {
			t.Fatalf("legal fact mutation %d did not affect digest", i)
		}
	}
}

func TestAdmissionLegalPayloadBuildersRejectIncompleteOrMalformedServerFacts(t *testing.T) {
	decision := AdmissionDecisionLegalPayloadInput{TenantCode: "tenant", InstitutionID: "institution", ActorSubject: "actor", DecidedAt: time.Now().UTC()}
	if _, err := BuildAdmissionDecisionLegalPayload(decision); err == nil {
		t.Fatal("decision builder accepted missing identifiers")
	}
	appeal := AdmissionAppealResolutionLegalPayloadInput{TenantCode: "tenant", InstitutionID: "institution", ActorSubject: strings.Repeat(" ", 2), ResolvedAt: time.Now().UTC()}
	if _, err := BuildAdmissionAppealResolutionLegalPayload(appeal); err == nil {
		t.Fatal("appeal-resolution builder accepted blank actor")
	}
}
