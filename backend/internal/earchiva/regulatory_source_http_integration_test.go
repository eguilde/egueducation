//go:build integration

package earchiva

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/eguilde/egueducation/internal/auth"
	appdb "github.com/eguilde/egueducation/internal/db"
	"github.com/eguilde/egueducation/internal/regulatorysource"
	"github.com/go-chi/chi/v5"
)

type regulatoryFetcherFake struct {
	bytes []byte
	calls *atomic.Int32
	after func()
}

func (f regulatoryFetcherFake) Fetch(_ context.Context, raw string) (regulatorysource.Evidence, error) {
	if f.calls != nil {
		f.calls.Add(1)
	}
	if f.after != nil {
		f.after()
	}
	h := sha256.Sum256(f.bytes)
	return regulatorysource.Evidence{Content: f.bytes, SHA256: hex.EncodeToString(h[:]), URL: raw, ContentType: "application/pdf", RetrievedAt: time.Now().UTC()}, nil
}

func TestRegulatorySourceHTTPPostgresIntegration(t *testing.T) {
	it := newArchiveIntegrationDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	admin := openArchiveIntegrationPool(t, ctx, it.databaseConfig)
	defer admin.Close()
	if err := appdb.Migrate(ctx, admin); err != nil {
		t.Fatal(err)
	}
	grantArchiveIntegrationAccess(t, ctx, admin)
	sessions := appdb.NewSessionPool(it.readerPool)
	adminSessions := appdb.NewSessionPool(admin)
	adminCtx, release := retentionScopedDB(t, ctx, admin, "reg-source-admin", true)
	defer release()
	creator := seedArchiveRetentionActor(t, adminCtx, adminSessions, "director")
	approver := seedArchiveRetentionActor(t, adminCtx, adminSessions, "director")
	var fetchCalls atomic.Int32
	evidenceBytes := []byte("%PDF-evidence")
	service := regulatorysource.NewService(sessions, regulatoryFetcherFake{bytes: evidenceBytes, calls: &fetchCalls})
	router := chi.NewRouter()
	router.Post("/sources", service.Register)
	router.Post("/sources/{sourceID}/verify", service.Verify)
	router.Post("/sources/{sourceID}/activate", service.Activate)
	router.Get("/sources/{sourceID}/evidence/{evidenceID}", service.DownloadEvidence)
	send := func(actor archiveRetentionHTTPActor, method, path, key string, payload any) *httptest.ResponseRecorder {
		c, rel := retentionScopedDB(t, ctx, it.readerPool, actor.Subject, false)
		defer rel()
		c = auth.WithSessionContextForIntegration(c, auth.SessionContext{TenantCode: "tenant-egueducation", InstitutionID: "inst-001", User: auth.SessionUser{ID: actor.ID, Sub: actor.Subject}})
		b, _ := json.Marshal(payload)
		r := httptest.NewRequest(method, path, bytes.NewReader(b)).WithContext(c)
		r.Header.Set("Idempotency-Key", key)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	reg := map[string]any{"citation": "law integration", "publisher_url": "https://publisher.test/law.pdf", "issuer": "test", "source_kind": "law", "applicable_from": "2026-01-01"}
	created := send(creator, "POST", "/sources", "register", reg)
	if created.Code != 201 {
		t.Fatalf("register %d %s", created.Code, created.Body.String())
	}
	var source regulatorysource.RegulatorySource
	if err := json.Unmarshal(created.Body.Bytes(), &source); err != nil || source.ID == "" {
		t.Fatal(err)
	}
	replay := send(creator, "POST", "/sources", "register", reg)
	if replay.Code != 200 {
		t.Fatalf("register replay %d", replay.Code)
	}
	verified := send(creator, "POST", "/sources/"+source.ID+"/verify", "verify", map[string]any{"expected_version": 1})
	if verified.Code != 200 {
		t.Fatalf("verify %d %s", verified.Code, verified.Body.String())
	}
	if err := json.Unmarshal(verified.Body.Bytes(), &source); err != nil || source.LatestEvidenceID == nil {
		t.Fatalf("verify dto %#v %v", source, err)
	}
	if fetchCalls.Load() != 1 {
		t.Fatalf("fetch calls=%d want 1", fetchCalls.Load())
	}
	if again := send(creator, "POST", "/sources/"+source.ID+"/verify", "verify", map[string]any{"expected_version": 1}); again.Code != 200 || fetchCalls.Load() != 1 {
		t.Fatalf("verify replay status=%d calls=%d", again.Code, fetchCalls.Load())
	}
	// A replay key is bound to the source target, not merely to a payload.
	other := reg
	other["citation"] = "other integration law"
	otherCreated := send(creator, "POST", "/sources", "other-register", other)
	var otherSource regulatorysource.RegulatorySource
	_ = json.Unmarshal(otherCreated.Body.Bytes(), &otherSource)
	if conflict := send(creator, "POST", "/sources/"+otherSource.ID+"/verify", "verify", map[string]any{"expected_version": 1}); conflict.Code != 409 {
		t.Fatalf("verify target conflict=%d", conflict.Code)
	}
	download := send(creator, "GET", "/sources/"+source.ID+"/evidence/"+*source.LatestEvidenceID, "", nil)
	if download.Code != 200 || !bytes.Equal(download.Body.Bytes(), evidenceBytes) {
		t.Fatalf("evidence download status=%d bytes=%q", download.Code, download.Body.Bytes())
	}
	self := send(creator, "POST", "/sources/"+source.ID+"/activate", "self", map[string]any{"expected_version": 2, "evidence_id": *source.LatestEvidenceID, "assessment": "applicable"})
	if self.Code != 409 {
		t.Fatalf("self activate %d %s", self.Code, self.Body.String())
	}
	active := send(approver, "POST", "/sources/"+source.ID+"/activate", "activate", map[string]any{"expected_version": 2, "evidence_id": *source.LatestEvidenceID, "assessment": "applicable"})
	if active.Code != 200 {
		t.Fatalf("activate %d %s", active.Code, active.Body.String())
	}
	if send(approver, "POST", "/sources/"+source.ID+"/activate", "activate", map[string]any{"expected_version": 2, "evidence_id": *source.LatestEvidenceID, "assessment": "applicable"}).Code != 200 {
		t.Fatal("activate replay")
	}

	// Revoke between reservation and finalization. The stale session remains,
	// so only the final live database authorization check can reject it.
	third := reg
	third["citation"] = "revoked-after-fetch"
	thirdCreated := send(approver, "POST", "/sources", "third-register", third)
	var thirdSource regulatorysource.RegulatorySource
	_ = json.Unmarshal(thirdCreated.Body.Bytes(), &thirdSource)
	service = regulatorysource.NewService(sessions, regulatoryFetcherFake{bytes: evidenceBytes, after: func() {
		_, _ = adminSessions.Exec(adminCtx, "update app_memberships set active=false where user_id=$1::uuid", approver.ID)
	}})
	router = chi.NewRouter()
	router.Post("/sources", service.Register)
	router.Post("/sources/{sourceID}/verify", service.Verify)
	revoked := send(approver, "POST", "/sources/"+thirdSource.ID+"/verify", "revoked-verify", map[string]any{"expected_version": 1})
	if revoked.Code != 403 {
		t.Fatalf("revoked after fetch status=%d body=%s", revoked.Code, revoked.Body.String())
	}
	var evidenceCount int
	if err := adminSessions.QueryRow(adminCtx, "select count(*) from school_regulatory_source_evidence where source_id=$1::uuid", thirdSource.ID).Scan(&evidenceCount); err != nil || evidenceCount != 0 {
		t.Fatalf("revoked evidence count=%d err=%v", evidenceCount, err)
	}
	// A source edit between reservation and finalization must reject the stale
	// evidence rather than attaching it to different cited metadata.
	fourth := reg
	fourth["citation"] = "edited-after-fetch"
	fourthCreated := send(creator, "POST", "/sources", "fourth-register", fourth)
	var fourthSource regulatorysource.RegulatorySource
	_ = json.Unmarshal(fourthCreated.Body.Bytes(), &fourthSource)
	service = regulatorysource.NewService(sessions, regulatoryFetcherFake{bytes: evidenceBytes, after: func() {
		_, _ = adminSessions.Exec(adminCtx, "update school_regulatory_sources set citation='changed during fetch',updated_by_subject=$1 where id=$2::uuid", creator.Subject, fourthSource.ID)
	}})
	router = chi.NewRouter()
	router.Post("/sources", service.Register)
	router.Post("/sources/{sourceID}/verify", service.Verify)
	changed := send(creator, "POST", "/sources/"+fourthSource.ID+"/verify", "changed-verify", map[string]any{"expected_version": 1})
	if changed.Code != 409 {
		t.Fatalf("source changed after fetch status=%d body=%s", changed.Code, changed.Body.String())
	}
	if err := adminSessions.QueryRow(adminCtx, "select count(*) from school_regulatory_source_evidence where source_id=$1::uuid", fourthSource.ID).Scan(&evidenceCount); err != nil || evidenceCount != 0 {
		t.Fatalf("changed source evidence count=%d err=%v", evidenceCount, err)
	}
}
