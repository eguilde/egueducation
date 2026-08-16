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
