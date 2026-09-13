package db

import (
	"os"
	"strings"
	"testing"
)

func TestSchoolAdmissionDSSBindingMigrationIsAtomicAndFailClosed(t *testing.T) {
	raw, err := os.ReadFile("migrations/0149_school_admission_dss_bindings.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{
		"school_admission_signed_artifact_bindings", "unique(tenant_code,institution_id,artifact_kind,artifact_id)",
		"education_signed_artifact_evidence", "education_signed_artifact_validations", "signature_format='PAdES'",
		"validator_provider", "validator_version", "validation_policy", "observed_sha256", "observed_size_bytes",
		"diagnostic_data<>'{}'::jsonb", "detailed_report<>'{}'::jsonb", "simple_report<>'{}'::jsonb", "etsi_validation_report<>'{}'::jsonb",
		"timestamp_token_sha256~'^[0-9a-f]{64}$'", "school_admission_requires_dss_binding", "deferrable initially deferred",
		"new.archive_retention_until >= artifact_at + make_interval", "school_admission_dss_retention_policies", "evaluation.evaluated_by_subject=artifact_actor",
		"e.submitted_by_subject=actor", "v.validated_by_subject=actor", "force row level security",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("DSS migration lacks %q", want)
		}
	}
}

func TestSchoolAdmissionCapacityReplacementGuardsAreAuthorizationWide(t *testing.T) {
	raw, err := os.ReadFile("migrations/0149_school_admission_dss_bindings.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{
		"create or replace function public.school_admission_campaign_guard()",
		"create or replace function public.school_admission_capacity_guard()",
		"pg_advisory_xact_lock(hashtextextended('school_admission_capacity:'||new.tenant_code||':'||new.institution_id||':'||new.authorization_id::text||':'||new.shift||':'||new.school_year,0))",
		"campaign.school_year=new.school_year",
		"campaign.status<>'cancelled'",
		"campaign.status<>'archived' or exists",
		"new_claim:=case when new.capacity_unit='students' then new.student_place_limit else new.capacity_limit end",
		"school-year admission campaigns exceed authorization capacity",
		"a.authorization_id=new.authorization_id and a.shift=new.shift",
		"authorization_capacity_unit='students'",
		"admission capacity allocation exceeds authorization student capacity",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("authorization-wide capacity guard lacks %q", want)
		}
	}
}

func TestSchoolAdmissionSignedPayloadContractBindsExactPayloadAndActor(t *testing.T) {
	raw, err := os.ReadFile("migrations/0151_school_admission_signed_payload_contract.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{
		"expected_canonical_legal_payload_sha256",
		"expected_actor_subject",
		"signed_payload_sha256",
		"certificate_sha256",
		"signed_actor_subject",
		"e.expected_canonical_legal_payload_sha256=new.canonical_legal_payload_sha256",
		"v.signed_payload_sha256=new.canonical_legal_payload_sha256",
		"v.signed_actor_subject=new.expected_actor_subject",
		"new.expected_actor_subject=actor",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("signed payload migration lacks %q", want)
		}
	}
}
