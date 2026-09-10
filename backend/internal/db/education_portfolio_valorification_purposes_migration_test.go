package db

import (
	"os"
	"strings"
	"testing"
)

func TestEducationPortfolioValorificationPurposesMigrationContract(t *testing.T) {
	payload, err := os.ReadFile("migrations/0115_education_portfolio_valorification_purposes.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(string(payload))
	for _, required := range []string{
		"add column if not exists purpose",
		"'licentiere'", "'definitivat'", "'grad_ii'", "'grad_i'",
		"'dezvoltare_profesionala'", "'inspectie_scolara'",
		"'evaluare_externa_calitate'", "'distinctie_premiu'",
		"portfolio valorification purpose is immutable",
		"portfolio_id, purpose, source_evaluation_id",
	} {
		if !strings.Contains(sql, required) {
			t.Errorf("valorification purpose migration is missing %q", required)
		}
	}
}
