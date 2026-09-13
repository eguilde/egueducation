package earchiva

import (
	"errors"
	"net/http/httptest"
	"testing"
)

func TestPortfolioCustodyRecoveryDispositions(t *testing.T) {
	for _, disposition := range []string{"teacher_access", "institution_archive_only"} {
		if !validRecoveryDisposition(disposition) {
			t.Fatalf("expected %q to be an allowed disposition", disposition)
		}
	}
	for _, disposition := range []string{"", "teacher", "archive_only"} {
		if validRecoveryDisposition(disposition) {
			t.Fatalf("expected %q to be rejected", disposition)
		}
	}
}

func TestRecoveryFailureCode(t *testing.T) {
	for _, test := range []struct {
		cause     error
		permanent bool
	}{
		{errors.New("recovery_storage_verification_failed: custody recovery provenance mismatch"), true},
		{errors.New("recovery_teacher_authority_absent"), true},
		{errors.New("recovery_storage_verification_failed: head exact custody version: connection refused"), false},
	} {
		_, permanent := recoveryFailureCode(test.cause)
		if permanent != test.permanent {
			t.Fatalf("recoveryFailureCode(%q) permanent=%v, want %v", test.cause, permanent, test.permanent)
		}
	}
}

func TestSameRecoveryDate(t *testing.T) {
	date := "2026-09-12"
	other := "2026-09-13"
	if !sameRecoveryDate(nil, nil) || !sameRecoveryDate(&date, &date) {
		t.Fatal("equal omitted and equal explicit dates must be idempotent")
	}
	if sameRecoveryDate(nil, &date) || sameRecoveryDate(&date, &other) {
		t.Fatal("omitted or different dates must conflict on replay")
	}
}

func TestPortfolioCustodyRecoveryStatus(t *testing.T) {
	for _, status := range []string{"stored", "queued", "leased", "committed", "blocked", "deadletter"} {
		if !recoveryStatus(status) {
			t.Fatalf("expected %q to be an allowed status", status)
		}
	}
	if recoveryStatus("raw_storage_error") {
		t.Fatal("raw storage error must not be an API status")
	}
}

func TestParsePortfolioCustodyRecoveryListQuery(t *testing.T) {
	validIntent := "550e8400-e29b-41d4-a716-446655440000"
	request := httptest.NewRequest("GET", "/?page=2&pageSize=50&status=queued&disposition=teacher_access&intent_id="+validIntent+"&portfolio_id="+validIntent+"&title=Quarterly%25Report&original_file_name=a_b.pdf&created_from=2026-01-01T00:00:00Z&created_to=2026-01-02T00:00:00Z&sort=title&direction=asc", nil)
	query, err := parsePortfolioCustodyRecoveryListQuery(request)
	if err != nil {
		t.Fatalf("parse valid query: %v", err)
	}
	if query.page != 2 || query.pageSize != 50 || query.sort != "title" || query.direction != "asc" || query.createdFrom == nil || query.createdTo == nil {
		t.Fatalf("unexpected parsed query: %#v", query)
	}

	for _, raw := range []string{
		"/?unexpected=value",
		"/?status=queued&status=stored",
		"/?page=0",
		"/?pageSize=101",
		"/?intent_id=not-a-uuid",
		"/?created_from=2026-01-01",
		"/?created_from=2026-01-02T00:00:00Z&created_to=2026-01-01T00:00:00Z",
		"/?sort=created_at;drop+table+records",
		"/?direction=sideways",
	} {
		if _, err := parsePortfolioCustodyRecoveryListQuery(httptest.NewRequest("GET", raw, nil)); err == nil {
			t.Fatalf("expected query %q to be rejected", raw)
		}
	}
}
