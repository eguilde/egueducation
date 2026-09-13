package education

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
)

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
			SourceObjectVersionID: "storage-version-1", SourceSizeBytes: 42, MimeType: "application/pdf", ZIPPath: "evidence/001-evidence.pdf",
			SourceSHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		}},
		GeneratedFiles: []PortfolioExportManifestGeneratedFile{{
			ZIPPath: "portfolio.json", SHA256: "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789", SizeBytes: 512,
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
	manifest.Documents[0].ArchiveVersionNo = 2
	for _, change := range []func(){
		func() { manifest.Documents[0].SourceObjectVersionID = "storage-version-2" },
		func() { manifest.Documents[0].SourceSizeBytes = 43 },
		func() { manifest.Documents[0].MimeType = "image/png" },
		func() { manifest.Documents[0].ZIPPath = "evidence/001-evidence.png" },
	} {
		change()
		if changed := portfolioExportManifestSHA256(manifest); changed == first {
			t.Fatal("v2 storage and ZIP provenance must be covered by the manifest hash")
		}
		manifest.Documents[0].SourceObjectVersionID, manifest.Documents[0].SourceSizeBytes, manifest.Documents[0].MimeType, manifest.Documents[0].ZIPPath = "storage-version-1", 42, "application/pdf", "evidence/001-evidence.pdf"
	}
	for _, change := range []func(){
		func() { manifest.GeneratedFiles[0].ZIPPath = "opis.json" },
		func() {
			manifest.GeneratedFiles[0].SHA256 = "bcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789a"
		},
		func() { manifest.GeneratedFiles[0].SizeBytes = 513 },
	} {
		change()
		if changed := portfolioExportManifestSHA256(manifest); changed == first {
			t.Fatal("generated file integrity must be covered by the manifest hash")
		}
		manifest.GeneratedFiles[0] = PortfolioExportManifestGeneratedFile{
			ZIPPath: "portfolio.json", SHA256: "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789", SizeBytes: 512,
		}
	}
}

func TestPortfolioExportManifestV1HashRegressionRemainsFrozen(t *testing.T) {
	legacy := `{"manifest_version":"egueducation.portfolio-export-manifest/v1","hash_algorithm":"SHA-256","manifest_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","tenant_code":"tenant-a","institution_id":"inst-a","portfolio":{"id":"p","portfolio_code":"P","school_year":"2026","status":"submitted"},"documents":[]}`
	digest := sha256.Sum256([]byte(legacy))
	if got, want := hex.EncodeToString(digest[:]), "d43eea1431c73c09dfb374ed11adff7077a86a561a55e471064c2dddc094bc29"; got != want {
		t.Fatalf("legacy v1 bytes changed: got %s want %s", got, want)
	}
	var decoded PortfolioExportManifest
	if err := json.Unmarshal([]byte(legacy), &decoded); err != nil {
		t.Fatalf("decode frozen v1 manifest: %v", err)
	}
	reserialized, err := json.Marshal(decoded)
	if err != nil {
		t.Fatalf("reserialize frozen v1 manifest: %v", err)
	}
	if string(reserialized) != legacy {
		t.Fatalf("additive generated-file fields changed v1 bytes:\n got %s\nwant %s", reserialized, legacy)
	}
	if portfolioExportManifestVersion == "egueducation.portfolio-export-manifest/v1" {
		t.Fatal("new manifests must not reinterpret the frozen v1 wire format")
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
