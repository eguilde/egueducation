package education

import (
	"strings"
	"testing"
)

func TestSchoolListFiltersMapEveryNewlyExposedFieldToSQL(t *testing.T) {
	tests := []struct {
		name  string
		where string
		want  []string
	}{
		{
			name: "governance minute",
			where: func() string {
				where, _ := buildGovernanceMinuteFilters(map[string]string{"discussion_summary": "audit", "due_on": "2026-09-10", "notes": "sealed"}, "meeting", "institution")
				return where
			}(),
			want: []string{"emm.discussion_summary", "to_char(emm.due_on, 'YYYY-MM-DD')", "emm.notes"},
		},
		{
			name: "portfolio opis",
			where: func() string {
				where, _ := buildPortfolioOpisFilters(map[string]string{"chronological_index": "3", "checked_on": "2026-09-10", "notes": "verified"}, "portfolio", "institution")
				return where
			}(),
			want: []string{"epo.chronological_index::text", "to_char(epo.checked_on, 'YYYY-MM-DD')", "epo.notes"},
		},
		{
			name: "mobility document",
			where: func() string {
				where, _ := buildMobilityDocumentFilters(map[string]string{"mandatory": "true", "notes": "complete"}, "mobility", "institution")
				return where
			}(),
			want: []string{"emd.mandatory", "emd.notes"},
		},
		{
			name: "mobility final decision alias",
			where: func() string {
				where, _ := buildMobilityFinalDecisionFilters(map[string]string{"decision_stage": "final", "legal_basis": "ROFUIP", "notes": "published"}, "mobility", "institution")
				return where
			}(),
			want: []string{"emfd.decision_type", "emfd.legal_basis", "emfd.notes"},
		},
		{
			name: "personnel file document",
			where: func() string {
				where, _ := buildPersonnelDocumentFilters(map[string]string{"file_reference": "ARH-1", "notes": "verified"}, "personnel", "institution")
				return where
			}(),
			want: []string{"epfd.file_reference", "epfd.notes"},
		},
		{
			name: "portfolio valorification",
			where: func() string {
				where, _ := buildPortfolioValorificationFilters(map[string]string{"target_reference": "ORD-1", "started_on": "2026-09-10", "notes": "complete"}, "portfolio", "institution")
				return where
			}(),
			want: []string{"epv.target_reference", "to_char(epv.started_on, 'YYYY-MM-DD')", "epv.notes"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, expected := range tt.want {
				if !strings.Contains(tt.where, expected) {
					t.Fatalf("filter is accepted but does not produce SQL for %q: %s", expected, tt.where)
				}
			}
		})
	}
}

func TestSchoolListSortMappingsUseActualColumns(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"governance mandate", governanceMembershipSortColumn("mandate_to"), "egm.mandate_to"},
		{"governance resolution date", governanceResolutionSortColumn("issued_on"), "emr.issued_on"},
		{"portfolio transfer receipt", portfolioTransferSortColumn("received_on"), "ept.received_on"},
		{"portfolio review reviewer", portfolioReviewSortColumn("reviewer_name"), "epr.reviewer_name"},
		{"minute due date", governanceMinuteSortColumn("due_on"), "emm.due_on"},
		{"opis document reference", portfolioOpisSortColumn("document_reference"), "epo.document_reference"},
		{"custody close date", portfolioCustodySortColumn("ended_on"), "epc.ended_on"},
		{"valorification completion", portfolioValorificationSortColumn("completed_on"), "epv.completed_on"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("sort mapping = %q, want %q", tt.got, tt.want)
			}
		})
	}
}
