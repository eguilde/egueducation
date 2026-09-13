package education

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPortfolioProcedureRequestValidationRejectsInvalidEffectivePeriod(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := &CreatePortfolioProcedureRequest{
		ProcedureCode: "  portfolio-2026  ",
		Title:         "  Procedură  ",
		EffectiveFrom: "2026-10-01",
		EffectiveTo:   "2026-09-30",
	}
	if validatePortfolioProcedureRequest(recorder, req) {
		t.Fatal("expected invalid effective period to be rejected")
	}
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), "invalid_portfolio_procedure_effective_period") {
		t.Fatalf("unexpected validation response: %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestPortfolioProcedureRequestValidationNormalizesAndDefaultsSource(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := &CreatePortfolioProcedureRequest{ProcedureCode: "  portfolio-2026  ", Title: "  Procedură  "}
	if !validatePortfolioProcedureRequest(recorder, req) {
		t.Fatalf("expected valid procedure request, got %s", recorder.Body.String())
	}
	if req.ProcedureCode != "portfolio-2026" || req.Title != "Procedură" || req.SourceRef == "" {
		t.Fatalf("request was not normalized/defaulted: %#v", req)
	}
}

func TestPortfolioProcedureExpectedUpdatedAtRequiresRFC3339(t *testing.T) {
	if validExpectedUpdatedAt("not-a-timestamp") {
		t.Fatal("invalid timestamp accepted")
	}
	if !validExpectedUpdatedAt("2026-09-09T12:30:45.123456Z") {
		t.Fatal("valid RFC3339Nano timestamp rejected")
	}
}

func TestPortfolioProcedureTimestampsAreSerializedAsActualUTC(t *testing.T) {
	for _, column := range []string{"approved_at", "published_at", "created_at", "updated_at", "superseded_at", "withdrawn_at"} {
		projection := "to_char(" + column + " at time zone 'UTC'"
		if !strings.Contains(portfolioProcedureColumns, projection) {
			t.Fatalf("portfolio procedure timestamp %s is labelled Z without an explicit UTC conversion", column)
		}
	}
}
