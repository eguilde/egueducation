package education

import "testing"

func TestPortfolioExportManifestHashIsDeterministicAndCoversProvenance(t *testing.T) {
	manifest := PortfolioExportManifest{
		ManifestVersion: portfolioExportManifestVersion,
		HashAlgorithm:   "SHA-256",
		TenantCode:      "tenant-balotesti",
		InstitutionID:   "inst-balotesti",
		Portfolio:       PortfolioExportManifestPortfolio{ID: "portfolio-1", PortfolioCode: "PORT-1", SchoolYear: "2026-2027", Status: "submitted"},
		Documents: []PortfolioExportManifestDocument{{
			EvidenceRecordID: "evidence-1", SectionCode: "identificare_profesionala", ComponentCode: "structura_cadru", DocumentTitle: "Act",
			ChronologicalNo: 1, IssuedOn: "2026-09-01", EvidenceType: "document", ArchiveDocumentID: "archive-document-1",
			ArchiveVersionID: "archive-version-1", ArchiveVersionNo: 2, SourceBucket: "earhive", SourceObjectKey: "tenant/a/document.pdf",
			SourceSHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		}},
	}
	first := portfolioExportManifestSHA256(manifest)
	second := portfolioExportManifestSHA256(manifest)
	if first != second || !isSHA256Hex(first) {
		t.Fatalf("manifest hash must be deterministic lower-case SHA-256: %q / %q", first, second)
	}
	manifest.Documents[0].SourceObjectKey = "tenant/a/other.pdf"
	if changed := portfolioExportManifestSHA256(manifest); changed == first {
		t.Fatal("storage provenance must be covered by the manifest hash")
	}
	manifest.Documents[0].SourceObjectKey = "tenant/a/document.pdf"
	manifest.Documents[0].ArchiveVersionNo = 3
	if changed := portfolioExportManifestSHA256(manifest); changed == first {
		t.Fatal("archive version provenance must be covered by the manifest hash")
	}
}

func TestPortfolioExportManifestValidatesSHA256Encoding(t *testing.T) {
	if !isSHA256Hex("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef") {
		t.Fatal("valid SHA-256 rejected")
	}
	for _, invalid := range []string{"", "not-a-hash", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcde"} {
		if isSHA256Hex(invalid) {
			t.Fatalf("invalid SHA-256 accepted: %q", invalid)
		}
	}
}
