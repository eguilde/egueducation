package education

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeOwnPortfolioRequest(t *testing.T) {
	valid := OwnPortfolioRequest{
		SchoolYear:    " 2026-2027 ",
		LastUpdatedOn: "2026-09-09",
		Notes:         "  evidence pending  ",
	}
	if err := normalizeOwnPortfolioRequest(&valid); err != nil {
		t.Fatalf("valid own portfolio content rejected: %v", err)
	}
	if valid.SchoolYear != "2026-2027" || valid.Notes != "evidence pending" {
		t.Fatalf("own portfolio content was not normalized: %#v", valid)
	}

	for _, testCase := range []OwnPortfolioRequest{
		{SchoolYear: "", LastUpdatedOn: "2026-09-09"},
		{SchoolYear: "2026-2027", LastUpdatedOn: "09-09-2026"},
	} {
		if err := normalizeOwnPortfolioRequest(&testCase); err == nil {
			t.Fatalf("invalid own portfolio content was accepted: %#v", testCase)
		}
	}
}

func TestPortfolioRecordRequestRejectsClientRetentionDeadline(t *testing.T) {
	request := httptest.NewRequest("POST", "/education/portfolios/records", strings.NewReader(`{"retention_until":"2030-01-01"}`))
	var target CreatePortfolioRecordRequest
	if err := decodePortfolioRecordRequest(request, &target); err == nil {
		t.Fatal("client-supplied retention_until must be rejected")
	}

	request = httptest.NewRequest("POST", "/education/portfolios/records", strings.NewReader(`{"owner_name":"Profesor"}`))
	if err := decodePortfolioRecordRequest(request, &target); err != nil {
		t.Fatalf("ordinary portfolio payload rejected: %v", err)
	}
}

func TestPortfolioCreateAndUpdateUseExplicitOwnerPairingContracts(t *testing.T) {
	const payload = `{"owner_user_id":"11111111-1111-4111-8111-111111111111","owner_personnel_id":"22222222-2222-4222-8222-222222222222","owner_name":"Profesor"}`

	createRequest := httptest.NewRequest("POST", "/education/portfolios/records", strings.NewReader(payload))
	var createTarget CreatePortfolioRecordRequest
	if err := decodePortfolioRecordRequest(createRequest, &createTarget); err != nil {
		t.Fatalf("create portfolio contract rejected owner pairing: %v", err)
	}
	if createTarget.OwnerUserID == "" || createTarget.OwnerPersonnelID == "" {
		t.Fatalf("create portfolio contract lost owner pairing: %#v", createTarget)
	}

	updateRequest := httptest.NewRequest("PATCH", "/education/portfolios/records/portfolio-id", strings.NewReader(payload))
	var updateTarget UpdatePortfolioRecordRequest
	if err := decodePortfolioRecordRequest(updateRequest, &updateTarget); err != nil {
		t.Fatalf("update portfolio contract rejected owner assertions: %v", err)
	}
	if updateTarget.OwnerUserID != createTarget.OwnerUserID || updateTarget.OwnerPersonnelID != createTarget.OwnerPersonnelID {
		t.Fatalf("create/update owner assertions diverged: create=%#v update=%#v", createTarget, updateTarget)
	}
}

func TestOwnPortfolioRequestRejectsLegacyClientControlledEvidence(t *testing.T) {
	request := httptest.NewRequest("POST", "/education/portfolios/me", strings.NewReader(`{"school_year":"2026-2027","last_updated_on":"2026-09-09","notes":"","authenticity_declared":true}`))
	var target OwnPortfolioRequest
	if err := decodeOwnPortfolioRequest(request, &target); err == nil {
		t.Fatal("legacy client-controlled declaration flags must be rejected")
	}
}

func TestOwnPortfolioPermissionNamesStayLeastPrivilege(t *testing.T) {
	if portfolioReadOwnPermission == "education.portfolios.read" || portfolioManageOwnPermission == "education.portfolios.manage" {
		t.Fatal("own-portfolio permissions must remain distinct from institution-wide permissions")
	}
}

func TestOwnPortfolioDocumentForcesArchiveBackedAdministrativeFields(t *testing.T) {
	archiveReference := "archive://4bc1fb76-6a14-4771-86bd-9a2691d41180"
	canonical, valid := normalizeOwnPortfolioArchiveReference(archiveReference)
	if !valid || canonical != archiveReference {
		t.Fatalf("valid archive reference was rejected or changed: %q %t", canonical, valid)
	}
	for _, reference := range []string{"", "document", "archive://document", "archive://4bc1fb76-6a14-4771-86bd-9a2691d41180/other"} {
		if _, valid := normalizeOwnPortfolioArchiveReference(reference); valid {
			t.Fatalf("non-canonical archive reference accepted: %q", reference)
		}
	}
	command := ownPortfolioDocumentCommand(OwnPortfolioDocumentRequest{
		SectionCode: "I", ComponentCode: "I.1", DocumentTitle: "Dovada",
		EvidenceType: "adeverinta", IssuedOn: "2026-09-01", AddedOn: "2026-09-02",
		ChronologicalIndex: 1, FileReference: archiveReference,
	}, canonical)
	if command.SourceScope != "portofoliu" || command.AuthenticityStatus != "declarat" {
		t.Fatalf("owner controlled administrative fields: %#v", command)
	}
	if command.FileReference != archiveReference {
		t.Fatalf("archive reference was not preserved: %#v", command)
	}
}
