package education

import (
	"strings"
	"testing"
)

func TestEvaluationDashboardQueryDefinesEvaluationAlias(t *testing.T) {
	if !strings.Contains(evaluationDashboardQuery, "from education_evaluations ee") {
		t.Fatal("evaluation dashboard query must define the ee alias used by the correlated result-issues subquery")
	}
}

func TestGovernanceMeetingAuditActionsAreResourceSpecific(t *testing.T) {
	tests := map[string]string{
		"create": governanceMeetingAuditCreate,
		"update": governanceMeetingAuditUpdate,
		"delete": governanceMeetingAuditDelete,
	}
	for operation, action := range tests {
		want := "education.governance.meeting." + operation
		if action != want {
			t.Errorf("meeting %s audit action = %q, want %q", operation, action, want)
		}
	}
}
