package earchiva

import (
	"context"
	"crypto/md5" //nolint:gosec // S3 protocol assertion.
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eguilde/egueducation/internal/config"
)

func TestPortfolioCustodyLifecycleVerifiesRetentionBeforeHoldRelease(t *testing.T) {
	required := time.Now().UTC().Add(72 * time.Hour).Truncate(time.Second)
	storage, calls := newPortfolioLifecycleProtocolStorage(t, required, false)
	state, err := storage.ReconcilePortfolioCustodyLifecycle(context.Background(), PortfolioCustodyLifecycleRequest{
		Key: "archive/evidence.pdf", VersionID: "version-1", ETag: "etag-1", RetentionUntil: required, LegalHoldActive: false,
	})
	if err != nil {
		t.Fatalf("reconcile custody lifecycle: %v", err)
	}
	if state.RetentionUntil.Before(required) || state.LegalHoldActive {
		t.Fatalf("unverified lifecycle state: %#v", state)
	}
	joined := strings.Join(*calls, ",")
	if !strings.Contains(joined, "PUT retention,GET retention,GET legal-hold,PUT legal-hold,GET legal-hold") {
		t.Fatalf("retention was not verified before hold release: %s", joined)
	}
}

func TestPortfolioCustodyLifecycleNeverReleasesHoldWhenRetentionFails(t *testing.T) {
	required := time.Now().UTC().Add(72 * time.Hour).Truncate(time.Second)
	storage, calls := newPortfolioLifecycleProtocolStorage(t, required, true)
	_, err := storage.ReconcilePortfolioCustodyLifecycle(context.Background(), PortfolioCustodyLifecycleRequest{
		Key: "archive/evidence.pdf", VersionID: "version-1", ETag: "etag-1", RetentionUntil: required, LegalHoldActive: false,
	})
	if err == nil {
		t.Fatal("retention failure must fail closed")
	}
	for _, call := range *calls {
		if call == "PUT legal-hold" {
			t.Fatalf("hold release attempted after retention failure: %v", *calls)
		}
	}
}

func TestPortfolioCustodyLifecycleRejectsExpiredRetentionBeforeObjectMutation(t *testing.T) {
	expired := time.Now().UTC().Add(-time.Second).Truncate(time.Second)
	storage, calls := newPortfolioLifecycleProtocolStorage(t, expired, false)
	_, err := storage.ReconcilePortfolioCustodyLifecycle(context.Background(), PortfolioCustodyLifecycleRequest{
		Key: "archive/evidence.pdf", VersionID: "version-1", ETag: "etag-1", RetentionUntil: expired, LegalHoldActive: false,
	})
	if err == nil || !strings.Contains(err.Error(), "retention expired review required") {
		t.Fatalf("expired retention error=%v", err)
	}
	for _, call := range *calls {
		if call == "PUT retention" || call == "PUT legal-hold" {
			t.Fatalf("expired retention mutated object state: %v", *calls)
		}
	}
}

func TestPortfolioCustodyLifecycleRejectsMutableOrMismatchedIdentity(t *testing.T) {
	required := time.Now().UTC().Add(72 * time.Hour).Truncate(time.Second)
	storage, calls := newPortfolioLifecycleProtocolStorage(t, required, false)
	_, err := storage.ReconcilePortfolioCustodyLifecycle(context.Background(), PortfolioCustodyLifecycleRequest{
		Key: "archive/evidence.pdf", VersionID: "version-1", ETag: "different", RetentionUntil: required, LegalHoldActive: false,
	})
	if err == nil || !strings.Contains(err.Error(), "identity mismatch") {
		t.Fatalf("mismatched ETag error=%v", err)
	}
	if len(*calls) != 1 || (*calls)[0] != "HEAD exact" {
		t.Fatalf("identity mismatch performed mutation: %v", *calls)
	}
}

func newPortfolioLifecycleProtocolStorage(t *testing.T, required time.Time, failRetention bool) (*ArchiveStorage, *[]string) {
	t.Helper()
	calls := []string{}
	retentionStored := false
	holdActive := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead && r.URL.Path == "/archive-test" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == http.MethodGet && r.URL.Query().Has("versioning") {
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<VersioningConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Status>Enabled</Status></VersioningConfiguration>`))
			return
		}
		if r.Method == http.MethodGet && r.URL.Query().Has("object-lock") {
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<ObjectLockConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><ObjectLockEnabled>Enabled</ObjectLockEnabled></ObjectLockConfiguration>`))
			return
		}
		if r.Method == http.MethodHead && r.URL.Query().Get("versionId") == "version-1" {
			calls = append(calls, "HEAD exact")
			w.Header().Set("X-Amz-Version-Id", "version-1")
			w.Header().Set("ETag", `"etag-1"`)
			w.Header().Set("Content-Length", "3")
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Query().Has("retention") {
			calls = append(calls, r.Method+" retention")
			if r.Method == http.MethodPut {
				assertLifecycleContentMD5(t, r)
				if failRetention {
					http.Error(w, "retention unavailable", http.StatusInternalServerError)
					return
				}
				retentionStored = true
				w.WriteHeader(http.StatusOK)
				return
			}
			w.Header().Set("Content-Type", "application/xml")
			if retentionStored {
				_, _ = fmt.Fprintf(w, `<Retention xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Mode>COMPLIANCE</Mode><RetainUntilDate>%s</RetainUntilDate></Retention>`, required.Format(time.RFC3339))
			} else {
				_, _ = w.Write([]byte(`<Retention xmlns="http://s3.amazonaws.com/doc/2006-03-01/"/>`))
			}
			return
		}
		if r.URL.Query().Has("legal-hold") {
			calls = append(calls, r.Method+" legal-hold")
			if r.Method == http.MethodPut {
				assertLifecycleContentMD5(t, r)
				holdActive = false
				w.WriteHeader(http.StatusOK)
				return
			}
			status := "OFF"
			if holdActive {
				status = "ON"
			}
			w.Header().Set("Content-Type", "application/xml")
			_, _ = fmt.Fprintf(w, `<LegalHold xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Status>%s</Status></LegalHold>`, status)
			return
		}
		http.Error(w, "unexpected lifecycle protocol request "+r.Method+" "+r.URL.String(), http.StatusMethodNotAllowed)
	}))
	t.Cleanup(server.Close)
	storage, err := NewArchiveStorage(context.Background(), config.Config{
		ArchiveStorageEndpoint: server.URL, ArchiveStorageRegion: "us-east-1", ArchiveStorageBucket: "archive-test",
		ArchiveStorageAccessKey: "test", ArchiveStorageSecretKey: "test", ArchiveStorageUsePathStyle: true, ArchiveStorageRequireObjectLock: true,
	})
	if err != nil {
		t.Fatalf("create lifecycle protocol storage: %v", err)
	}
	return storage, &calls
}

func assertLifecycleContentMD5(t *testing.T, request *http.Request) {
	t.Helper()
	body, err := io.ReadAll(request.Body)
	if err != nil {
		t.Errorf("read lifecycle protocol body: %v", err)
		return
	}
	digest := md5.Sum(body) //nolint:gosec // S3 protocol assertion.
	want := base64.StdEncoding.EncodeToString(digest[:])
	if got := request.Header.Get("Content-Md5"); got != want {
		t.Errorf("Content-MD5=%q want=%q body=%s", got, want, body)
	}
}
