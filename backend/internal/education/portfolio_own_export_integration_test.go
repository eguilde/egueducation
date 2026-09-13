//go:build integration

package education

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eguilde/egueducation/internal/db"
)

type fakePortfolioArchiveReader struct {
	bucket  string
	objects map[string][]byte
	calls   int
}

func (f *fakePortfolioArchiveReader) Bucket() string { return f.bucket }

func (f *fakePortfolioArchiveReader) OpenObjectVersion(_ context.Context, key, versionID string) (io.ReadCloser, error) {
	f.calls++
	return io.NopCloser(bytes.NewReader(f.objects[key+"\x00"+versionID])), nil
}

func TestPortfolioOwnExportStagesVerifiedOwnerBundle(t *testing.T) {
	it := newGovernanceIntegrationDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	defer cancel()
	adminPool := openGovernanceIntegrationPool(t, ctx, it.databaseConfig)
	defer adminPool.Close()
	if err := db.Migrate(ctx, adminPool); err != nil {
		t.Fatalf("migrate export database: %v", err)
	}
	grantGovernanceIntegrationAccess(t, ctx, adminPool, it.roleName)
	fixture := seedGovernanceAuthorizationFixture(t, ctx, adminPool)
	seedPortfolioExportableEvidence(t, ctx, adminPool, fixture)
	grantPortfolioExportOwnPermissions(t, ctx, adminPool, fixture)

	reader := &fakePortfolioArchiveReader{bucket: "earhive", objects: map[string][]byte{"integration/evidence.pdf\x00integration-version-1": []byte("original evidence bytes")}}
	service := NewService(db.NewSessionPool(it.readerPool), WithPortfolioArchiveReader(reader))
	ownerCtx, release := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	defer release()
	request := withChiParams(requestWithContext(ownerCtx, fixture.tenantA, fixture.institutionA, fixture.memberSubject), map[string]string{"recordID": fixture.portfolioID})
	request.Method = http.MethodPost
	response := httptest.NewRecorder()
	service.PortfolioOwnExport(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("owner ZIP status=%d body=%s", response.Code, response.Body.String())
	}
	if response.Header().Get("Content-Type") != "application/zip" || reader.calls != 1 {
		t.Fatalf("ZIP response or exact-object read missing: headers=%v calls=%d", response.Header(), reader.calls)
	}
	archive, err := zip.NewReader(bytes.NewReader(response.Body.Bytes()), int64(response.Body.Len()))
	if err != nil {
		t.Fatalf("read ZIP: %v", err)
	}
	members := map[string][]byte{}
	for _, item := range archive.File {
		body, err := item.Open()
		if err != nil {
			t.Fatalf("open ZIP member %s: %v", item.Name, err)
		}
		members[item.Name], err = io.ReadAll(body)
		_ = body.Close()
		if err != nil {
			t.Fatalf("read ZIP member %s: %v", item.Name, err)
		}
	}
	if got := string(members["evidence/001-evidence.pdf"]); got != "original evidence bytes" {
		t.Fatalf("ZIP must contain verified exact evidence bytes, got %q", got)
	}
	digest := sha256.Sum256(members["manifest.canonical.json"])
	if got, want := hex.EncodeToString(digest[:]), string(members["manifest.sha256"])[:64]; got != want {
		t.Fatalf("canonical manifest sidecar mismatch: got %s want %s", got, want)
	}
	for _, required := range []string{"portfolio.json", "opis.json", "opis.txt", "manifest.json", "manifest.canonical.json", "manifest.sha256"} {
		if _, ok := members[required]; !ok {
			t.Fatalf("missing required ZIP member %s", required)
		}
	}
	var manifest PortfolioExportManifest
	if err := json.Unmarshal(members["manifest.json"], &manifest); err != nil {
		t.Fatalf("decode ZIP manifest: %v", err)
	}
	wantPaths := []string{"portfolio.json", "opis.json", "opis.txt"}
	if len(manifest.GeneratedFiles) != len(wantPaths) {
		t.Fatalf("generated file manifest count=%d want=%d: %#v", len(manifest.GeneratedFiles), len(wantPaths), manifest.GeneratedFiles)
	}
	generated := make([]portfolioExportGeneratedPayload, 0, len(wantPaths))
	for index, path := range wantPaths {
		payload, ok := members[path]
		if !ok {
			t.Fatalf("missing generated ZIP payload %s", path)
		}
		digest := sha256.Sum256(payload)
		descriptor := manifest.GeneratedFiles[index]
		if descriptor.ZIPPath != path || descriptor.SizeBytes != int64(len(payload)) || descriptor.SHA256 != hex.EncodeToString(digest[:]) {
			t.Fatalf("generated descriptor does not bind ZIP bytes: descriptor=%#v path=%s size=%d sha256=%s", descriptor, path, len(payload), hex.EncodeToString(digest[:]))
		}
		generated = append(generated, portfolioExportGeneratedPayload{ZIPPath: path, Payload: payload})
	}
	if !portfolioExportGeneratedFilesMatch(manifest.GeneratedFiles, generated) {
		t.Fatal("manifest must verify the exact generated ZIP payloads")
	}
	tampered := append([]byte(nil), generated[1].Payload...)
	tampered[len(tampered)-2] ^= 1
	generated[1].Payload = tampered
	if portfolioExportGeneratedFilesMatch(manifest.GeneratedFiles, generated) {
		t.Fatal("manifest accepted tampered generated OPIS JSON bytes")
	}
}

func TestPortfolioOwnExportRejectsCorruptOriginalBeforeHeaders(t *testing.T) {
	it := newGovernanceIntegrationDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	defer cancel()
	adminPool := openGovernanceIntegrationPool(t, ctx, it.databaseConfig)
	defer adminPool.Close()
	if err := db.Migrate(ctx, adminPool); err != nil {
		t.Fatalf("migrate export database: %v", err)
	}
	grantGovernanceIntegrationAccess(t, ctx, adminPool, it.roleName)
	fixture := seedGovernanceAuthorizationFixture(t, ctx, adminPool)
	seedPortfolioExportableEvidence(t, ctx, adminPool, fixture)
	grantPortfolioExportOwnPermissions(t, ctx, adminPool, fixture)

	reader := &fakePortfolioArchiveReader{bucket: "earhive", objects: map[string][]byte{"integration/evidence.pdf\x00integration-version-1": []byte("corrupted original")}}
	service := NewService(db.NewSessionPool(it.readerPool), WithPortfolioArchiveReader(reader))
	ownerCtx, release := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	defer release()
	request := withChiParams(requestWithContext(ownerCtx, fixture.tenantA, fixture.institutionA, fixture.memberSubject), map[string]string{"recordID": fixture.portfolioID})
	request.Method = http.MethodPost
	response := httptest.NewRecorder()
	service.PortfolioOwnExport(response, request)
	if response.Code != http.StatusUnprocessableEntity || response.Header().Get("Content-Type") == "application/zip" || response.Body.Len() == 0 {
		t.Fatalf("corrupt immutable original must fail before ZIP headers: status=%d headers=%v body=%s", response.Code, response.Header(), response.Body.String())
	}
}
