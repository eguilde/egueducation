package schooloperations

import (
	"testing"
)

func TestContractTransitionVocabularyIsClosed(t *testing.T) {
	for _, status := range []string{"verified", "approved", "signed", "active", "suspended", "expired", "terminated", "archived"} {
		if !validTransition(status) {
			t.Fatalf("%q must be a server-recognized lifecycle transition", status)
		}
	}
	for _, status := range []string{"draft", "deleted", "active; drop table school_contracts"} {
		if validTransition(status) {
			t.Fatalf("%q must not be client-transitionable", status)
		}
	}
}

func TestContractDatesAndCurrencyAreValidated(t *testing.T) {
	if !validContractDates("2026-09-01", "2027-08-31") || !validContractDates("2026-09-01", "") {
		t.Fatal("valid contract date range rejected")
	}
	for _, dates := range [][2]string{{"", "2027-08-31"}, {"invalid", ""}, {"2027-09-01", "2027-08-31"}} {
		if validContractDates(dates[0], dates[1]) {
			t.Fatalf("invalid contract dates accepted: %#v", dates)
		}
	}
	if !validCurrency("RON") || validCurrency("ron") || validCurrency("EURO") {
		t.Fatal("currency validation is not closed")
	}
}

func TestNormalizeCreateContractProducesStableIdempotencyInput(t *testing.T) {
	input := CreateContractRequest{SupplierPartyID: " supplier ", ContractNumber: " C-1 ", Title: " Contract ", StartsOn: " 2026-09-01 ", Currency: " eur ", IdempotencyKey: " key "}
	normalizeCreateContract(&input)
	if input.SupplierPartyID != "supplier" || input.ContractNumber != "C-1" || input.Title != "Contract" || input.Category != "general" || input.Currency != "EUR" || input.IdempotencyKey != "key" {
		t.Fatalf("unexpected normalization: %#v", input)
	}
}

func TestDefaultStringDoesNotAcceptEmptyClientValues(t *testing.T) {
	if got := defaultString("  ", "RON"); got != "RON" {
		t.Fatalf("got %q", got)
	}
	if got := defaultString(" EUR ", "RON"); got != "EUR" {
		t.Fatalf("got %q", got)
	}
}
