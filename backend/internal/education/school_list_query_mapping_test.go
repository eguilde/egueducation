package education

import (
	"net/url"
	"strings"
	"testing"

	"github.com/eguilde/egueducation/internal/httpx"
)

func TestSchoolDetailRecordListQueryMappings(t *testing.T) {
	where, args := buildMeetingParticipantFilters(map[string]string{
		"full_name": "Ana", "role_name": "Chair", "member_type": "staff", "attendance_status": "present", "signature_present": "true", "voting_right": "false",
	}, "meeting", "institution")
	requireFilterColumns(t, where, args, "emp.full_name", "emp.role_name", "emp.member_type", "emp.attendance_status", "emp.signature_present", "emp.voting_right")
	requireSortMappings(t, meetingParticipantSortColumn, map[string]string{"full_name": "emp.full_name", "role_name": "emp.role_name", "member_type": "emp.member_type", "attendance_status": "emp.attendance_status", "signature_present": "emp.signature_present", "voting_right": "emp.voting_right"})

	where, args = buildMeetingDocumentFilters(map[string]string{
		"document_type": "notice", "title": "Board", "document_number": "1", "registry_number": "2", "publication_status": "published", "issued_on": "2026-09-10", "custody_owner": "secretariat",
	}, "meeting", "institution")
	requireFilterColumns(t, where, args, "emd.document_type", "emd.title", "emd.document_number", "emd.registry_number", "emd.publication_status", "to_char(emd.issued_on", "emd.custody_owner")
	requireSortMappings(t, meetingDocumentSortColumn, map[string]string{"document_type": "emd.document_type", "title": "emd.title", "document_number": "emd.document_number", "registry_number": "emd.registry_number", "publication_status": "emd.publication_status", "issued_on": "emd.issued_on", "custody_owner": "emd.custody_owner"})

	where, args = buildMeetingVoteFilters(map[string]string{
		"subject_title": "Budget", "agenda_order": "2", "decision_type": "approval", "outcome": "approved", "requires_follow_up": "true",
	}, "meeting", "institution")
	requireFilterColumns(t, where, args, "emv.subject_title", "emv.agenda_order", "emv.decision_type", "emv.outcome", "emv.requires_follow_up")
	requireSortMappings(t, meetingVoteSortColumn, map[string]string{"subject_title": "emv.subject_title", "agenda_order": "emv.agenda_order", "decision_type": "emv.decision_type", "outcome": "emv.outcome", "votes_for": "emv.votes_for", "votes_against": "emv.votes_against", "abstentions": "emv.abstentions"})

	where, args = buildPortfolioDocumentFilters(map[string]string{
		"section_code": "P1", "component_code": "C1", "document_title": "Plan", "source_scope": "internal", "evidence_type": "file", "issued_on": "2026-09-10", "chronological_index": "3", "sensitive_data": "false", "authenticity_status": "verified",
	}, "portfolio", "institution")
	requireFilterColumns(t, where, args, "epd.section_code", "epd.component_code", "epd.document_title", "epd.source_scope", "epd.evidence_type", "to_char(epd.issued_on", "epd.chronological_index::text", "epd.sensitive_data", "epd.authenticity_status")
	requireSortMappings(t, portfolioDocumentSortColumn, map[string]string{"section_code": "epd.section_code", "document_title": "epd.document_title", "evidence_type": "epd.evidence_type", "issued_on": "epd.issued_on", "authenticity_status": "epd.authenticity_status"})

	where, args = buildPortfolioChecklistFilters(map[string]string{
		"requirement_code": "R1", "section_code": "P1", "source_scope": "internal", "status": "ready", "mandatory": "true", "checked_by": "Ana",
	}, "portfolio", "institution")
	requireFilterColumns(t, where, args, "epc.requirement_code", "epc.section_code", "epc.source_scope", "epc.status", "epc.mandatory", "epc.checked_by")
	requireSortMappings(t, portfolioChecklistSortColumn, map[string]string{"requirement_code": "epc.requirement_code", "requirement_label": "epc.requirement_label", "section_code": "epc.section_code", "status": "epc.status", "document_count": "epc.document_count"})
}

func TestPortfolioManagerialListSortContractsRejectUnknownAndHaveStableDefaults(t *testing.T) {
	tests := []struct {
		name       string
		allowed    map[string]struct{}
		mapper     func(string) string
		defaultSQL string
	}{
		{"documents", map[string]struct{}{"document_title": {}, "evidence_type": {}, "section_code": {}, "authenticity_status": {}, "issued_on": {}}, portfolioDocumentSortColumn, "epd.issued_on"},
		{"checklist", map[string]struct{}{"requirement_code": {}, "requirement_label": {}, "section_code": {}, "status": {}, "document_count": {}}, portfolioChecklistSortColumn, "epc.requirement_code"},
		{"opis", map[string]struct{}{"section_code": {}, "component_code": {}, "entry_title": {}, "chronological_index": {}, "document_reference": {}}, portfolioOpisSortColumn, "epo.chronological_index"},
		{"custody", map[string]struct{}{"event_type": {}, "holder_name": {}, "holder_role": {}, "started_on": {}, "ended_on": {}}, portfolioCustodySortColumn, "epc.started_on"},
		{"reviews", map[string]struct{}{"review_code": {}, "review_stage": {}, "outcome": {}, "reviewer_name": {}, "reviewed_on": {}}, portfolioReviewSortColumn, "epr.reviewed_on"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := httpx.ParsePageQuery(url.Values{"sort": {"untrusted_sql_column"}, "direction": {"DROP"}}, tt.allowed, nil)
			if query.Sort != "" || query.Direction != "asc" {
				t.Fatalf("untrusted sort must be rejected and direction normalized, got %#v", query)
			}
			if got := tt.mapper(query.Sort); got != tt.defaultSQL {
				t.Fatalf("default sort mapping = %q, want %q", got, tt.defaultSQL)
			}
		})
	}
}

func TestSchoolWorkflowListQueryMappings(t *testing.T) {
	where, args := buildCommitteeFilters(map[string]string{"school_year": "2026", "committee_type": "ethics", "title": "Ethics", "status": "active"}, "institution")
	requireFilterColumns(t, where, args, "ec.school_year", "ec.committee_type", "ec.title", "ec.status")
	requireSortMappings(t, committeeSortColumn, map[string]string{"committee_code": "ec.committee_code", "school_year": "ec.school_year", "committee_type": "ec.committee_type", "title": "ec.title", "status": "ec.status", "starts_on": "ec.starts_on"})

	where, args = buildCommitteeMemberFilters(map[string]string{"full_name": "Ana", "role_name": "Chair", "member_type": "staff", "status": "active"}, "committee", "institution")
	requireFilterColumns(t, where, args, "ecm.full_name", "ecm.role_name", "ecm.member_type", "ecm.status")
	requireSortMappings(t, committeeMemberSortColumn, map[string]string{"full_name": "ecm.full_name", "role_name": "ecm.role_name", "member_type": "ecm.member_type", "status": "ecm.status", "appointed_on": "ecm.appointed_on"})

	where, args = buildDecisionIssuanceFilters(map[string]string{"issuance_code": "DEC", "document_type": "decision", "recipient_name": "Ana", "recipient_role": "teacher", "delivery_channel": "email", "delivery_status": "sent"}, "decision", "institution")
	requireFilterColumns(t, where, args, "edi.issuance_code", "edi.document_type", "edi.recipient_name", "edi.recipient_role", "edi.delivery_channel", "edi.delivery_status")
	requireSortMappings(t, decisionIssuanceSortColumn, map[string]string{"issuance_code": "edi.issuance_code", "document_type": "edi.document_type", "recipient_name": "edi.recipient_name", "recipient_role": "edi.recipient_role", "delivery_channel": "edi.delivery_channel", "delivery_status": "edi.delivery_status", "signed_on": "edi.signed_on", "delivered_on": "edi.delivered_on"})

	where, args = buildDecisionPublicationStepFilters(map[string]string{"step_type": "publish", "status": "done", "responsible_name": "Ana", "publication_channel": "web", "publication_reference": "URL"}, "decision", "institution")
	requireFilterColumns(t, where, args, "edps.step_type", "edps.status", "edps.responsible_name", "edps.publication_channel", "edps.publication_reference")
	requireSortMappings(t, decisionPublicationStepSortColumn, map[string]string{"step_order": "edps.step_order", "step_type": "edps.step_type", "status": "edps.status", "responsible_name": "edps.responsible_name", "publication_channel": "edps.publication_channel", "due_on": "edps.due_on", "completed_on": "edps.completed_on"})
}

func TestSchoolManagementAndEvaluationListQueryMappings(t *testing.T) {
	where, args := buildManagerialDocumentFilters(map[string]string{"document_code": "DOC", "document_category": "plan", "title": "Plan", "document_status": "approved", "version_label": "v1", "owner_name": "Ana"}, "dossier", "institution")
	requireFilterColumns(t, where, args, "emd.document_code", "emd.document_category", "emd.title", "emd.document_status", "emd.version_label", "emd.owner_name")
	requireSortMappings(t, managerialDocumentSortColumn, map[string]string{"document_code": "emd.document_code", "document_category": "emd.document_category", "title": "emd.title", "document_status": "emd.document_status", "version_label": "emd.version_label", "owner_name": "emd.owner_name", "registered_on": "emd.registered_on", "approved_on": "emd.approved_on"})

	where, args = buildManagerialWorkflowFilters(map[string]string{"stage_type": "review", "status": "open", "assigned_to": "Ana", "decision_reference": "DEC"}, "dossier", "institution")
	requireFilterColumns(t, where, args, "emw.stage_type", "emw.status", "emw.assigned_to", "emw.decision_reference")
	requireSortMappings(t, managerialWorkflowSortColumn, map[string]string{"stage_order": "emw.stage_order", "stage_type": "emw.stage_type", "status": "emw.status", "assigned_to": "emw.assigned_to", "due_on": "emw.due_on", "completed_on": "emw.completed_on"})

	where, args = buildRegulationVersionFilters(map[string]string{"version_label": "v1", "version_status": "approved", "prepared_by": "Ana", "file_reference": "file"}, "regulation", "institution")
	requireFilterColumns(t, where, args, "erv.version_label", "erv.version_status", "erv.prepared_by", "erv.file_reference")
	requireSortMappings(t, regulationVersionSortColumn, map[string]string{"version_label": "erv.version_label", "version_status": "erv.version_status", "prepared_by": "erv.prepared_by", "approved_on": "erv.approved_on", "effective_from": "erv.effective_from", "published_on": "erv.published_on"})

	where, args = buildRegulationWorkflowFilters(map[string]string{"phase_type": "review", "status": "open", "audience": "staff", "decision_reference": "DEC"}, "regulation", "institution")
	requireFilterColumns(t, where, args, "erw.phase_type", "erw.status", "erw.audience", "erw.decision_reference")
	requireSortMappings(t, regulationWorkflowSortColumn, map[string]string{"phase_order": "erw.phase_order", "phase_type": "erw.phase_type", "status": "erw.status", "audience": "erw.audience", "started_on": "erw.started_on", "due_on": "erw.due_on", "completed_on": "erw.completed_on", "feedback_count": "erw.feedback_count"})

	where, args = buildEvaluationSelfReviewFilters(map[string]string{"review_code": "AUTO", "section_title": "Teaching", "narrative_type": "impact", "status": "submitted"}, "evaluation", "institution")
	requireFilterColumns(t, where, args, "eesr.review_code", "eesr.section_title", "eesr.narrative_type", "eesr.status")
	requireSortMappings(t, evaluationSelfReviewSortColumn, map[string]string{"review_code": "eesr.review_code", "section_title": "eesr.section_title", "narrative_type": "eesr.narrative_type", "status": "eesr.status", "completed_on": "eesr.completed_on", "assumed_score": "eesr.assumed_score"})

	where, args = buildEvaluationCriteriaFilters(map[string]string{"criterion_code": "CRIT", "criterion_category": "teaching", "criterion_label": "Lesson", "status": "validated"}, "evaluation", "institution")
	requireFilterColumns(t, where, args, "eec.criterion_code", "eec.criterion_category", "eec.criterion_label", "eec.status")
	requireSortMappings(t, evaluationCriteriaSortColumn, map[string]string{"criterion_code": "eec.criterion_code", "criterion_category": "eec.criterion_category", "criterion_label": "eec.criterion_label", "status": "eec.status", "max_score": "eec.max_score", "final_score": "eec.final_score"})

	where, args = buildEvaluationResultIssueFilters(map[string]string{"issue_code": "EVRES", "document_type": "decision", "recipient_name": "Ana", "recipient_role": "teacher", "delivery_channel": "email", "delivery_status": "sent", "issued_on": "2026-09-10", "delivered_on": "2026-09-10", "acknowledged_on": "2026-09-10", "attached_to_personnel_file": "true"}, "evaluation", "institution")
	requireFilterColumns(t, where, args, "eeri.issue_code", "eeri.document_type", "eeri.recipient_name", "eeri.recipient_role", "eeri.delivery_channel", "eeri.delivery_status", "eeri.issued_on", "eeri.delivered_on", "eeri.acknowledged_on", "eeri.attached_to_personnel_file")
	requireSortMappings(t, evaluationResultIssueSortColumn, map[string]string{"issue_code": "eeri.issue_code", "document_type": "eeri.document_type", "recipient_name": "eeri.recipient_name", "recipient_role": "eeri.recipient_role", "delivery_channel": "eeri.delivery_channel", "delivery_status": "eeri.delivery_status", "issued_on": "eeri.issued_on", "delivered_on": "eeri.delivered_on", "acknowledged_on": "eeri.acknowledged_on"})
}

func requireFilterColumns(t *testing.T, where string, args []any, columns ...string) {
	t.Helper()
	for _, column := range columns {
		if !strings.Contains(where, column) {
			t.Errorf("filter SQL does not contain %q: %s", column, where)
		}
	}
	baseArgs := 1
	if strings.Contains(where, "institution_id = $2") {
		baseArgs = 2
	}
	if got, want := len(args), len(columns)+baseArgs; got != want {
		t.Errorf("filter arguments = %d, want %d for %s", got, want, where)
	}
}

func requireSortMappings(t *testing.T, mapper func(string) string, cases map[string]string) {
	t.Helper()
	for value, want := range cases {
		if got := mapper(value); got != want {
			t.Errorf("sort mapping for %q = %q, want %q", value, got, want)
		}
	}
}
