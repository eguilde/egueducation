package education

import (
	"strings"
	"testing"
)

func TestSchoolPrimaryListFiltersAreBackedBySQL(t *testing.T) {
	type filterBuilder func(map[string]string, string) (string, []any)
	tests := []struct {
		name    string
		build   filterBuilder
		columns map[string]string
	}{
		{"meetings", buildMeetingFilters, map[string]string{"title": "em.title", "school_year": "em.school_year", "organism": "em.organism", "meeting_type": "em.meeting_type", "status": "em.status", "meeting_date": "em.meeting_date", "chairperson": "em.chairperson", "secretary_name": "em.secretary_name"}},
		{"decisions", buildDecisionFilters, map[string]string{"decision_code": "ed.decision_code", "title": "ed.title", "school_year": "ed.school_year", "organism": "ed.organism", "status": "ed.status", "publication_status": "ed.publication_status", "decision_date": "ed.decision_date", "legal_basis": "ed.legal_basis", "signed_by": "ed.signed_by", "summary": "ed.summary"}},
		{"managerial", buildManagerialFilters, map[string]string{"dossier_code": "emd.dossier_code", "title": "emd.title", "school_year": "emd.school_year", "dossier_type": "emd.dossier_type", "status": "emd.status", "due_on": "emd.due_on", "publication_required": "emd.publication_required", "owner_name": "emd.owner_name", "summary": "emd.summary"}},
		{"regulations", buildRegulationFilters, map[string]string{"regulation_code": "er.regulation_code", "title": "er.title", "school_year": "er.school_year", "regulation_type": "er.regulation_type", "status": "er.status", "approval_status": "er.approval_status", "review_due_on": "er.review_due_on", "approved_on": "er.approved_on", "owner_name": "er.owner_name", "summary": "er.summary"}},
		{"personnel", buildPersonnelFilters, map[string]string{"employee_code": "ep.employee_code", "full_name": "ep.full_name", "role_title": "ep.role_title", "employment_type": "ep.employment_type", "status": "ep.status", "evaluation_status": "ep.evaluation_status", "mobility_stage": "ep.mobility_stage", "school_year": "ep.school_year", "assigned_unit": "ep.assigned_unit", "phone": "ep.phone", "email": "ep.email", "has_portfolio": "ep.has_portfolio", "notes": "ep.notes"}},
		{"evaluations", buildEvaluationFilters, map[string]string{"evaluation_code": "ee.evaluation_code", "employee_code": "ee.employee_code", "full_name": "ee.full_name", "role_title": "ee.role_title", "school_year": "ee.school_year", "status": "ee.status", "score": "ee.score", "qualification": "ee.qualification", "finalized_on": "ee.finalized_on", "evaluator_name": "ee.evaluator_name", "summary": "ee.summary"}},
		{"declarations", buildDeclarationFilters, map[string]string{"declaration_code": "ed.declaration_code", "employee_code": "ed.employee_code", "full_name": "ed.full_name", "declaration_type": "ed.declaration_type", "status": "ed.status", "school_year": "ed.school_year", "submitted_on": "ed.submitted_on", "valid_until": "ed.valid_until", "summary": "ed.summary"}},
		{"portfolios", buildPortfolioFilters, map[string]string{"portfolio_code": "epf.portfolio_code", "owner_name": "epf.owner_name", "owner_role": "epf.owner_role", "school_year": "epf.school_year", "status": "epf.status", "section_count": "epf.section_count", "last_updated_on": "epf.last_updated_on", "transfer_status": "epf.transfer_status", "authenticity_declared": "epf.authenticity_declared", "consent_captured": "epf.consent_captured", "retention_until": "epf.retention_until", "custodian": "epf.custodian", "notes": "epf.notes"}},
		{"mobility", buildMobilityFilters, map[string]string{"case_code": "emc.case_code", "employee_code": "emc.employee_code", "full_name": "emc.full_name", "school_year": "emc.school_year", "request_type": "emc.request_type", "stage": "emc.stage", "status": "emc.status", "source_school": "emc.source_school", "destination_school": "emc.destination_school", "submitted_on": "emc.submitted_on", "reviewed_by": "emc.reviewed_by", "notes": "emc.notes"}},
		{"merit", buildMeritFilters, map[string]string{"grant_code": "emg.grant_code", "full_name": "emg.full_name", "role_title": "emg.role_title", "school_year": "emg.school_year", "category": "emg.category", "status": "emg.status", "score": "emg.score", "committee_name": "emg.committee_name", "decision_date": "emg.decision_date", "funded": "emg.funded", "notes": "emg.notes"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filters := make(map[string]string, len(test.columns))
			for field := range test.columns {
				filters[field] = "true"
			}
			sql, args := test.build(filters, "school-1")
			if got, want := len(args), len(test.columns)+1; got != want {
				t.Fatalf("%d filters reached SQL arguments; want %d: %s", got-1, want-1, sql)
			}
			for field, column := range test.columns {
				if !strings.Contains(sql, column) {
					t.Errorf("filter %q is not backed by expected SQL column %q: %s", field, column, sql)
				}
			}
		})
	}
}

func TestSchoolPrimaryListSortsMapToRequestedColumns(t *testing.T) {
	tests := []struct {
		name   string
		mapper func(string) string
		fields map[string]string
	}{
		{"meetings", meetingSortColumn, map[string]string{"title": "em.title", "school_year": "em.school_year", "organism": "em.organism", "meeting_type": "em.meeting_type", "status": "em.status", "meeting_date": "em.meeting_date", "chairperson": "em.chairperson", "secretary_name": "em.secretary_name"}},
		{"decisions", decisionSortColumn, map[string]string{"decision_code": "ed.decision_code", "title": "ed.title", "school_year": "ed.school_year", "organism": "ed.organism", "status": "ed.status", "publication_status": "ed.publication_status", "decision_date": "ed.decision_date", "legal_basis": "ed.legal_basis", "signed_by": "ed.signed_by", "summary": "ed.summary"}},
		{"managerial", managerialSortColumn, map[string]string{"dossier_code": "emd.dossier_code", "title": "emd.title", "school_year": "emd.school_year", "dossier_type": "emd.dossier_type", "status": "emd.status", "due_on": "emd.due_on", "publication_required": "emd.publication_required", "owner_name": "emd.owner_name", "summary": "emd.summary"}},
		{"regulations", regulationSortColumn, map[string]string{"regulation_code": "er.regulation_code", "title": "er.title", "school_year": "er.school_year", "regulation_type": "er.regulation_type", "status": "er.status", "approval_status": "er.approval_status", "review_due_on": "er.review_due_on", "approved_on": "er.approved_on", "owner_name": "er.owner_name", "summary": "er.summary"}},
		{"personnel", personnelSortColumn, map[string]string{"employee_code": "ep.employee_code", "full_name": "ep.full_name", "role_title": "ep.role_title", "employment_type": "ep.employment_type", "status": "ep.status", "evaluation_status": "ep.evaluation_status", "mobility_stage": "ep.mobility_stage", "school_year": "ep.school_year", "assigned_unit": "ep.assigned_unit", "phone": "ep.phone", "email": "ep.email", "has_portfolio": "ep.has_portfolio", "notes": "ep.notes"}},
		{"evaluations", evaluationSortColumn, map[string]string{"evaluation_code": "ee.evaluation_code", "employee_code": "ee.employee_code", "full_name": "ee.full_name", "role_title": "ee.role_title", "school_year": "ee.school_year", "status": "ee.status", "score": "ee.score", "qualification": "ee.qualification", "finalized_on": "ee.finalized_on", "evaluator_name": "ee.evaluator_name", "summary": "ee.summary"}},
		{"declarations", declarationSortColumn, map[string]string{"declaration_code": "ed.declaration_code", "employee_code": "ed.employee_code", "full_name": "ed.full_name", "declaration_type": "ed.declaration_type", "status": "ed.status", "school_year": "ed.school_year", "submitted_on": "ed.submitted_on", "valid_until": "ed.valid_until", "summary": "ed.summary"}},
		{"portfolios", portfolioSortColumn, map[string]string{"portfolio_code": "epf.portfolio_code", "owner_name": "epf.owner_name", "owner_role": "epf.owner_role", "school_year": "epf.school_year", "status": "epf.status", "section_count": "epf.section_count", "last_updated_on": "epf.last_updated_on", "transfer_status": "epf.transfer_status", "authenticity_declared": "epf.authenticity_declared", "consent_captured": "epf.consent_captured", "retention_until": "epf.retention_until", "custodian": "epf.custodian", "notes": "epf.notes"}},
		{"mobility", mobilitySortColumn, map[string]string{"case_code": "emc.case_code", "employee_code": "emc.employee_code", "full_name": "emc.full_name", "school_year": "emc.school_year", "request_type": "emc.request_type", "stage": "emc.stage", "status": "emc.status", "source_school": "emc.source_school", "destination_school": "emc.destination_school", "submitted_on": "emc.submitted_on", "reviewed_by": "emc.reviewed_by", "notes": "emc.notes"}},
		{"merit", meritSortColumn, map[string]string{"grant_code": "emg.grant_code", "full_name": "emg.full_name", "role_title": "emg.role_title", "school_year": "emg.school_year", "category": "emg.category", "status": "emg.status", "score": "emg.score", "committee_name": "emg.committee_name", "decision_date": "emg.decision_date", "funded": "emg.funded", "notes": "emg.notes"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for field, want := range test.fields {
				if got := test.mapper(field); got != want {
					t.Errorf("sort %q maps to %q; want %q", field, got, want)
				}
			}
		})
	}
}
