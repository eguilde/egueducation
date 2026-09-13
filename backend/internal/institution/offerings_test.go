package institution

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestSchoolCatalogValidationIsClosedAndEffectiveDated(t *testing.T) {
	for _, value := range []string{"sediu", "gimnazial.ro", "nivel-1", "unitate_2"} {
		if !schoolCatalogCode.MatchString(value) {
			t.Fatalf("expected valid catalog code %q", value)
		}
	}
	for _, value := range []string{"", " Gimnazial", "../other", "Școală", strings.Repeat("a", 65)} {
		if schoolCatalogCode.MatchString(value) {
			t.Fatalf("expected invalid catalog code %q", value)
		}
	}
	validTo := "2027-08-31"
	invalidTo := "2025-08-31"
	if !validEffectiveWindow("2026-09-01", &validTo) || validEffectiveWindow("2026-09-01", &invalidTo) || validEffectiveWindow("09/01/2026", nil) {
		t.Fatal("effective-date windows are not validated deterministically")
	}
	blank := " "
	invalidDate := "2026-02-30"
	if !validOptionalDate(nil) || !validOptionalDate(&blank) || validOptionalDate(&invalidDate) {
		t.Fatal("optional effective dates are not validated deterministically")
	}
}

func TestSchoolCatalogPersistenceErrorClassification(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "idempotency mismatch", err: errIdempotencyPayloadConflict, status: http.StatusConflict, code: "idempotency_key_payload_conflict"},
		{name: "unique conflict", err: &pgconn.PgError{Code: "23505"}, status: http.StatusConflict, code: "school_location_conflict"},
		{name: "invalid foreign key", err: &pgconn.PgError{Code: "23503"}, status: http.StatusUnprocessableEntity, code: "school_location_persist_failed"},
		{name: "application dependency conflict", err: errAuthorizationDependencyConflict, status: http.StatusConflict, code: "school_location_authorization_dependency_conflict"},
		{name: "database dependency conflict", err: &pgconn.PgError{Code: "23514", ConstraintName: "school_location_authorization_dependency_conflict"}, status: http.StatusConflict, code: "school_location_authorization_dependency_conflict"},
		{name: "internal failure", err: errors.New("database unavailable"), status: http.StatusInternalServerError, code: "school_location_persist_failed"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			writeCatalogPersistenceError(recorder, test.err, "school_location")
			if recorder.Code != test.status || !strings.Contains(recorder.Body.String(), test.code) {
				t.Fatalf("got status=%d body=%s", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestCatalogLifecycleUpdateRequiresEndDateWhenInactive(t *testing.T) {
	active, inactive := true, false
	end := "2027-08-31"
	blank := " "
	if !validCatalogLifecycleUpdate(1, "Catalog item", &active, nil) {
		t.Fatal("active catalog item with an open interval was rejected")
	}
	if validCatalogLifecycleUpdate(1, "Catalog item", &inactive, nil) || validCatalogLifecycleUpdate(1, "Catalog item", &inactive, &blank) {
		t.Fatal("inactive catalog item without an explicit end date was accepted")
	}
	if !validCatalogLifecycleUpdate(1, "Catalog item", &inactive, &end) {
		t.Fatal("inactive catalog item with an explicit valid end date was rejected")
	}
}

func TestSchoolCatalogDecoderRejectsUnknownAndTrailingJSON(t *testing.T) {
	for name, payload := range map[string]string{
		"unknown":  `{"code":"sediu","forged_tenant":"other"}`,
		"trailing": `{"code":"sediu"}{"code":"other"}`,
	} {
		t.Run(name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest("POST", "/api/institution/locations", strings.NewReader(payload))
			var input CreateSchoolLocationRequest
			if decodeClosedJSON(recorder, request, &input) {
				t.Fatalf("decoder accepted %s payload", name)
			}
		})
	}
}

func TestSchoolCatalogFingerprintIncludesBusinessPayload(t *testing.T) {
	base := CreateEducationOfferingRequest{Code: "gimnazial", EducationLevel: "gimnazial", Title: "Gimnaziu", EffectiveFrom: "2026-09-01", IdempotencyKey: "same"}
	changed := base
	changed.Title = "Alt titlu"
	if requestFingerprint(base) == requestFingerprint(changed) {
		t.Fatal("idempotency fingerprint ignored changed business payload")
	}
	if requestFingerprint(base) != requestFingerprint(base) {
		t.Fatal("idempotency fingerprint is not deterministic")
	}
}

func TestOfferingAuthorizationStatusIsClosed(t *testing.T) {
	for _, value := range []string{"provisional", "accredited", "suspended", "withdrawn", "expired"} {
		if !validAuthorizationStatus(value) {
			t.Fatalf("expected valid authorization status %q", value)
		}
	}
	for _, value := range []string{"", "active", "approved", "authorized", "deleted"} {
		if validAuthorizationStatus(value) {
			t.Fatalf("unexpected authorization status %q", value)
		}
	}
}

func TestSchoolCatalogUpdateContractsRejectServerOwnedIdentity(t *testing.T) {
	for name, test := range map[string]struct {
		target  any
		payload string
	}{
		"location": {target: &UpdateSchoolLocationRequest{}, payload: `{"expected_version":1,"name":"Sediu","address":"Balotești","active":true,"effective_to":null,"tenant_code":"forged"}`},
		"offering": {target: &UpdateEducationOfferingRequest{}, payload: `{"expected_version":1,"title":"Ofertă","active":true,"effective_to":null,"tenant_code":"forged"}`},
	} {
		t.Run(name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest("PATCH", "/api/institution/catalog/id", strings.NewReader(test.payload))
			if decodeClosedJSON(recorder, request, test.target) {
				t.Fatal("update contract accepted a forged tenant field")
			}
		})
	}
}
