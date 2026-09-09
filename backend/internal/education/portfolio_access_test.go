package education

import "testing"

func TestNormalizeOwnPortfolioRequest(t *testing.T) {
	valid := OwnPortfolioRequest{
		SchoolYear:    " 2026-2027 ",
		SectionCount:  0,
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
		{SchoolYear: "2026-2027", SectionCount: -1, LastUpdatedOn: "2026-09-09"},
		{SchoolYear: "2026-2027", LastUpdatedOn: "09-09-2026"},
	} {
		if err := normalizeOwnPortfolioRequest(&testCase); err == nil {
			t.Fatalf("invalid own portfolio content was accepted: %#v", testCase)
		}
	}
}

func TestOwnPortfolioRetentionIsServerDerived(t *testing.T) {
	if got := ownPortfolioRetentionUntil(); len(got) != len("2006-01-02") {
		t.Fatalf("server retention must be a canonical date, got %q", got)
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
