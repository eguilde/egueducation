package db

import (
	"os"
	"strings"
	"testing"
)

func TestSchoolAdmissionsFoundationIsScopedFailClosedAndEvidentiary(t *testing.T) {
	b, err := os.ReadFile("migrations/0148_school_admissions.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	for _, needle := range []string{
		"school_admission_campaigns", "school_admission_criteria", "school_admission_document_requirements",
		"school_admission_applications", "school_admission_application_representatives", "school_admission_application_documents",
		"school_admission_criterion_assessments", "school_admission_decisions", "school_admission_decision_deliveries",
		"school_admission_appeals", "school_admission_appeal_submissions", "school_admission_appeal_resolutions",
		"school_admission_capacity_allocations", "school_admission_export_manifests", "school_admission_class_offering_contexts",
		"school_admission_idempotency", "school_admission_outbox", "application_id uuid not null", "allocated_capacity integer not null default 1 check(allocated_capacity=1)",
		"school_admission_capacity_allocations_one_active_application", "status in ('held','consumed')", "update of application_id,campaign_id,class_offering_context_id",
		"foreign key(tenant_code,institution_id", "force row level security", "school_admission_tenant_isolation",
		"school_admission_archive_snapshot_guard", "ready WORM archive version snapshot", "source_bucket=new.archive_source_bucket", "source_object_key=new.archive_source_object_key", "source_sha256)=new.archive_sha256", "source_object_version_id<>''", "archive_retention_until",
		"school_admission_capacity_guard", "school_admission_class_context_guard", "school_admission_authorization_campaign_dependency_guard", "school_admission_authorization_campaign_dependency_conflict", "admission campaign capacity exceeds authorization capacity", "admission capacity allocation exceeds campaign student place limit", "admission campaign student place limit cannot be reduced below active allocations", "student_place_limit,capacity_basis",
		"capacity_unit", "student_place_limit", "capacity_basis", "students_per_group", "shift", "school_admission_immutable_evidence", "school_operations_no_hard_delete",
		"school_admission_authorization_eligible", "a.status in ('provisional', 'accredited')",
		"education_student_enrolments_scope_id_unique", "education_students_party_scope_fk", "admission_application_id uuid", "app_parties(tenant_code,institution_id,id)",
		"policy_evaluation_v2_id uuid not null", "school_operation_policy_evaluations_v2", "admission.decision.issue", "admission.appeal.resolve", "'campaign_id',campaign.id::text,'application_id',app.id::text", "'appeal_id',appeal.id::text", "'outcome',new.outcome", "exact allowed operation context", "'indeterminate'", "admission decision fails closed while a required criterion is not met", "admission decision fails closed while a required document is not accepted or waived", "'rejected_late'", "appeal resolver must differ from decision issuer", "appeal resolution resulting decision must preserve exact outcome and allocation", "check(resulting_outcome is not null)", "student party must be an in-scope physical person", "admission candidate must be an active in-scope physical person", "appeal appellant must be candidate or registered legal representative", "application documents are immutable after final state or decision", "criterion assessments are immutable after final state or decision", "admission campaign requires an active regulatory source covering its window", "source_id uuid not null", "students_per_group') ~ '^[1-9][0-9]*$'", "class_row.school_year=new.school_year", "resulting_outcome", "occurred_by_subject",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("admission foundation lacks %q", needle)
		}
	}
	for _, forbidden := range []string{
		"a.status in ('provisional', 'authorized', 'accredited')",
		"a.status in ('authorized', 'accredited')",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("admission eligibility must fail closed for ambiguous authorized status: %q", forbidden)
		}
	}
}

func TestSchoolAdmissionsSchemaContractRequiresAdmissionTenantPolicy(t *testing.T) {
	contracts := make(map[string]TableContract, len(SchemaContract()))
	for _, contract := range SchemaContract() {
		contracts[contract.Name] = contract
	}
	for _, tableName := range []string{
		"school_admission_class_offering_contexts",
		"school_admission_campaigns",
		"school_admission_criteria",
		"school_admission_document_requirements",
		"school_admission_applications",
		"school_admission_capacity_allocations",
		"school_admission_application_representatives",
		"school_admission_application_documents",
		"school_admission_criterion_assessments",
		"school_admission_decisions",
		"school_admission_decision_deliveries",
		"school_admission_appeals",
		"school_admission_appeal_submissions",
		"school_admission_appeal_resolutions",
		"school_admission_export_manifests",
		"school_admission_idempotency",
		"school_admission_outbox",
	} {
		contract, found := contracts[tableName]
		if !found {
			t.Errorf("schema contract omits %s", tableName)
			continue
		}
		if contract.Scope != SchemaScopeInstitution || len(contract.RequiredPolicies) != 1 || contract.RequiredPolicies[0] != "school_admission_tenant_isolation" {
			t.Errorf("schema contract for %s must require school_admission_tenant_isolation, got %#v", tableName, contract)
		}
	}
}
