package db

import (
	"context"
	"fmt"
	"strings"
)

// SchoolPolicyCutoverPreflight is a read-only, scope-bound inventory of the
// evidence that must be reconciled before an institution can leave the legacy
// policy runtime. Counts describe gaps, not rows that may be repaired by an
// application request.
type SchoolPolicyCutoverPreflight struct {
	TenantCode    string
	InstitutionID string
	Phase         string

	LegacyProfiles      int64
	UnmappedProfiles    int64
	LegacyAssignments   int64
	UnmappedAssignments int64
	LegacyOverrides     int64
	UnmappedOverrides   int64
	LegacyEvaluations   int64
	UnmappedEvaluations int64

	MissingPackProvenance        int64
	InputsWithoutEffectiveDate   int64
	InputsWithMultipleDecisions  int64
	ConsumerProvenanceMismatches int64
	OpenBlockingIssues           int64
}

// ReadyForDual reports whether the current evidence is structurally complete.
// It intentionally does not authorize a transition: writer health, shadow
// comparison and an immutable reconciliation are separate mandatory gates.
func (p SchoolPolicyCutoverPreflight) ReadyForDual() bool {
	return p.Phase == "legacy" &&
		p.UnmappedProfiles == 0 &&
		p.UnmappedAssignments == 0 &&
		p.UnmappedOverrides == 0 &&
		p.UnmappedEvaluations == 0 &&
		p.MissingPackProvenance == 0 &&
		p.InputsWithoutEffectiveDate == 0 &&
		p.InputsWithMultipleDecisions == 0 &&
		p.ConsumerProvenanceMismatches == 0 &&
		p.OpenBlockingIssues == 0
}

// ReadSchoolPolicyCutoverPreflight derives tenant and institution exclusively
// from the pinned PostgreSQL request session. It never accepts scope from an
// HTTP payload or caller-supplied identifier.
func ReadSchoolPolicyCutoverPreflight(ctx context.Context, pool *SessionPool) (SchoolPolicyCutoverPreflight, error) {
	if pool == nil || pool.raw() == nil {
		return SchoolPolicyCutoverPreflight{}, fmt.Errorf("school policy cutover preflight requires a database session")
	}

	var report SchoolPolicyCutoverPreflight
	err := pool.QueryRow(ctx, schoolPolicyCutoverPreflightSQL).Scan(
		&report.TenantCode,
		&report.InstitutionID,
		&report.Phase,
		&report.LegacyProfiles,
		&report.UnmappedProfiles,
		&report.LegacyAssignments,
		&report.UnmappedAssignments,
		&report.LegacyOverrides,
		&report.UnmappedOverrides,
		&report.LegacyEvaluations,
		&report.UnmappedEvaluations,
		&report.MissingPackProvenance,
		&report.InputsWithoutEffectiveDate,
		&report.InputsWithMultipleDecisions,
		&report.ConsumerProvenanceMismatches,
		&report.OpenBlockingIssues,
	)
	if err != nil {
		return SchoolPolicyCutoverPreflight{}, fmt.Errorf("read school policy cutover preflight: %w", err)
	}
	if strings.TrimSpace(report.TenantCode) == "" || strings.TrimSpace(report.InstitutionID) == "" {
		return SchoolPolicyCutoverPreflight{}, fmt.Errorf("school policy cutover preflight requires a tenant-bound institution session")
	}
	return report, nil
}

const schoolPolicyCutoverPreflightSQL = `
with scope as (
	select public.current_tenant_code() as tenant_code,
		public.current_institution_id() as institution_id
), consumer_mismatches as (
	select count(*)::bigint as value from (
		select p.id
		from education_publications p
		left join school_policy_evaluation_cutover_identity m
			on (m.tenant_code, m.institution_id, m.legacy_evaluation_id)
			= (p.tenant_code, p.institution_id, p.policy_evaluation_id)
		join scope s on (s.tenant_code, s.institution_id) = (p.tenant_code, p.institution_id)
		where p.policy_evaluation_id is not null
			and (m.id is null or p.policy_evaluation_v2_id is distinct from m.v2_evaluation_id)
		union all
		select c.id from school_contracts c left join school_policy_evaluation_cutover_identity m on (m.tenant_code,m.institution_id,m.legacy_evaluation_id)=(c.tenant_code,c.institution_id,c.policy_evaluation_id) join scope s on (s.tenant_code,s.institution_id)=(c.tenant_code,c.institution_id) where c.policy_evaluation_id is not null and (m.id is null or c.policy_evaluation_v2_id is distinct from m.v2_evaluation_id)
		union all
		select c.id from school_contract_obligations c left join school_policy_evaluation_cutover_identity m on (m.tenant_code,m.institution_id,m.legacy_evaluation_id)=(c.tenant_code,c.institution_id,c.policy_evaluation_id) join scope s on (s.tenant_code,s.institution_id)=(c.tenant_code,c.institution_id) where c.policy_evaluation_id is not null and (m.id is null or c.policy_evaluation_v2_id is distinct from m.v2_evaluation_id)
		union all
		select c.id from school_utility_points c left join school_policy_evaluation_cutover_identity m on (m.tenant_code,m.institution_id,m.legacy_evaluation_id)=(c.tenant_code,c.institution_id,c.policy_evaluation_id) join scope s on (s.tenant_code,s.institution_id)=(c.tenant_code,c.institution_id) where c.policy_evaluation_id is not null and (m.id is null or c.policy_evaluation_v2_id is distinct from m.v2_evaluation_id)
		union all
		select c.id from school_utility_readings c left join school_policy_evaluation_cutover_identity m on (m.tenant_code,m.institution_id,m.legacy_evaluation_id)=(c.tenant_code,c.institution_id,c.policy_evaluation_id) join scope s on (s.tenant_code,s.institution_id)=(c.tenant_code,c.institution_id) where c.policy_evaluation_id is not null and (m.id is null or c.policy_evaluation_v2_id is distinct from m.v2_evaluation_id)
		union all
		select c.id from school_utility_invoices c left join school_policy_evaluation_cutover_identity m on (m.tenant_code,m.institution_id,m.legacy_evaluation_id)=(c.tenant_code,c.institution_id,c.policy_evaluation_id) join scope s on (s.tenant_code,s.institution_id)=(c.tenant_code,c.institution_id) where c.policy_evaluation_id is not null and (m.id is null or c.policy_evaluation_v2_id is distinct from m.v2_evaluation_id)
		union all
		select c.id from school_compliance_obligations c left join school_policy_evaluation_cutover_identity m on (m.tenant_code,m.institution_id,m.legacy_evaluation_id)=(c.tenant_code,c.institution_id,c.policy_evaluation_id) join scope s on (s.tenant_code,s.institution_id)=(c.tenant_code,c.institution_id) where c.policy_evaluation_id is not null and (m.id is null or c.policy_evaluation_v2_id is distinct from m.v2_evaluation_id)
		union all
		select c.id from school_compliance_inspections c left join school_policy_evaluation_cutover_identity m on (m.tenant_code,m.institution_id,m.legacy_evaluation_id)=(c.tenant_code,c.institution_id,c.policy_evaluation_id) join scope s on (s.tenant_code,s.institution_id)=(c.tenant_code,c.institution_id) where c.policy_evaluation_id is not null and (m.id is null or c.policy_evaluation_v2_id is distinct from m.v2_evaluation_id)
		union all
		select c.id from school_compliance_corrective_actions c left join school_policy_evaluation_cutover_identity m on (m.tenant_code,m.institution_id,m.legacy_evaluation_id)=(c.tenant_code,c.institution_id,c.policy_evaluation_id) join scope s on (s.tenant_code,s.institution_id)=(c.tenant_code,c.institution_id) where c.policy_evaluation_id is not null and (m.id is null or c.policy_evaluation_v2_id is distinct from m.v2_evaluation_id)
	) mismatches
)
select s.tenant_code,
	s.institution_id,
	coalesce((select phase from school_policy_cutover_state cs where (cs.tenant_code,cs.institution_id)=(s.tenant_code,s.institution_id)), 'missing'),
	(select count(*) from school_institution_profiles p where (p.tenant_code,p.institution_id)=(s.tenant_code,s.institution_id)),
	(select count(*) from school_institution_profiles p left join school_profile_cutover_identity i on (i.tenant_code,i.institution_id,i.legacy_profile_id,i.legacy_profile_version)=(p.tenant_code,p.institution_id,p.id,p.version) where (p.tenant_code,p.institution_id)=(s.tenant_code,s.institution_id) and i.id is null),
	(select count(*) from school_policy_assignments a where (a.tenant_code,a.institution_id)=(s.tenant_code,s.institution_id)),
	(select count(*) from school_policy_assignments a left join school_operation_policy_bindings_v2 b on (b.tenant_code,b.institution_id,b.legacy_assignment_id)=(a.tenant_code,a.institution_id,a.id) where (a.tenant_code,a.institution_id)=(s.tenant_code,s.institution_id) and b.id is null),
	(select count(*) from school_policy_overrides o where (o.tenant_code,o.institution_id)=(s.tenant_code,s.institution_id)),
	(select count(*) from school_policy_overrides o left join school_operation_policy_overrides_v2 v on (v.tenant_code,v.institution_id,v.legacy_override_id)=(o.tenant_code,o.institution_id,o.id) where (o.tenant_code,o.institution_id)=(s.tenant_code,s.institution_id) and v.id is null),
	(select count(*) from school_policy_evaluations e where (e.tenant_code,e.institution_id)=(s.tenant_code,s.institution_id)),
	(select count(*) from school_policy_evaluations e left join school_policy_evaluation_cutover_identity m on (m.tenant_code,m.institution_id,m.legacy_evaluation_id)=(e.tenant_code,e.institution_id,e.id) where (e.tenant_code,e.institution_id)=(s.tenant_code,s.institution_id) and m.id is null),
	(select count(*) from school_policy_evaluations e cross join lateral unnest(e.policy_pack_version_ids) p(pack_id) left join school_policy_evaluation_cutover_identity m on (m.tenant_code,m.institution_id,m.legacy_evaluation_id)=(e.tenant_code,e.institution_id,e.id) left join school_operation_policy_input_bindings ib on (ib.tenant_code,ib.institution_id,ib.input_id,ib.policy_pack_version_id)=(m.tenant_code,m.institution_id,m.input_id,p.pack_id) where (e.tenant_code,e.institution_id)=(s.tenant_code,s.institution_id) and ib.id is null),
	(select count(*) from school_operation_policy_inputs i where (i.tenant_code,i.institution_id)=(s.tenant_code,s.institution_id) and i.effective_on is null),
	(select count(*) from (select input_id from school_operation_policy_evaluations_v2 e where (e.tenant_code,e.institution_id)=(s.tenant_code,s.institution_id) group by input_id having count(*) > 1) duplicate_inputs),
	(select value from consumer_mismatches),
	(select count(*) from school_regulatory_migration_issues mi where (mi.tenant_code,mi.institution_id)=(s.tenant_code,s.institution_id) and mi.status='open' and mi.severity='blocking')
from scope s
`
