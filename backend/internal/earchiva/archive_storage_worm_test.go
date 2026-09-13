package earchiva

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/eguilde/egueducation/internal/config"
)

type wormProtocolResponse struct {
	putVersion           string
	headVersion          string
	headMode             string
	headUntil            time.Time
	headSize             int64
	headMeta             map[string]string
	headETag             string
	headHold             *bool
	headStatus           int
	putStatus            int
	getBody              string
	getVersion           string
	getETag              string
	listedVersions       []string
	echoRequestedVersion bool
}

func TestPutImmutableObjectVerifiesExactWORMVersion(t *testing.T) {
	retention := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	response := wormProtocolResponse{putVersion: "version-1", headVersion: "version-1", headMode: "COMPLIANCE", headUntil: retention.Add(time.Minute), headSize: 3, headMeta: map[string]string{"policy-version": "v1"}, headHold: wormBool(true)}
	storage, observed := newWormProtocolStorage(t, response)
	object, err := storage.PutImmutableObject(context.Background(), ImmutableArchiveWrite{Key: "archive/document.pdf", ContentType: "application/pdf", Body: bytes.NewReader([]byte("pdf")), ContentLength: 3, RetentionUntil: retention, LegalHold: true, Metadata: map[string]string{"Policy-Version": "v1"}})
	if err != nil {
		t.Fatalf("immutable write: %v", err)
	}
	if !observed.putConditional || !observed.putCompliance || !observed.putLegalHold || observed.putMetadata != "v1" || observed.headVersion != "version-1" {
		t.Fatalf("immutable protocol observations = %#v", observed)
	}
	if object.VersionID != "version-1" || object.SizeBytes != 3 || object.ObjectLockMode != "COMPLIANCE" || !object.LegalHoldActive || object.Metadata["policy-version"] != "v1" || object.RetentionUntil.Before(retention) {
		t.Fatalf("unverified immutable result: %#v", object)
	}
}

func TestPutImmutableObjectRejectsObservedWORMMismatches(t *testing.T) {
	retention := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	for name, response := range map[string]wormProtocolResponse{
		"version":   {putVersion: "version-1", headVersion: "other", headMode: "COMPLIANCE", headUntil: retention, headSize: 3, headMeta: map[string]string{"policy-version": "v1"}},
		"retention": {putVersion: "version-1", headVersion: "version-1", headMode: "COMPLIANCE", headUntil: retention.Add(-time.Second), headSize: 3, headMeta: map[string]string{"policy-version": "v1"}},
		"mode":      {putVersion: "version-1", headVersion: "version-1", headMode: "GOVERNANCE", headUntil: retention, headSize: 3, headMeta: map[string]string{"policy-version": "v1"}},
		"size":      {putVersion: "version-1", headVersion: "version-1", headMode: "COMPLIANCE", headUntil: retention, headSize: 4, headMeta: map[string]string{"policy-version": "v1"}},
		"metadata":  {putVersion: "version-1", headVersion: "version-1", headMode: "COMPLIANCE", headUntil: retention, headSize: 3, headMeta: map[string]string{"policy-version": "other"}},
	} {
		t.Run(name, func(t *testing.T) {
			storage, _ := newWormProtocolStorage(t, response)
			object, err := storage.PutImmutableObject(context.Background(), ImmutableArchiveWrite{Key: "archive/document.pdf", Body: bytes.NewReader([]byte("pdf")), ContentLength: 3, RetentionUntil: retention, Metadata: map[string]string{"policy-version": "v1"}})
			var verification *ImmutableArchiveWriteVerificationError
			if !errors.As(err, &verification) || object.VersionID != "version-1" || verification.Object.VersionID != "version-1" {
				t.Fatalf("mismatch err=%v object=%#v, want recoverable verification error with real version", err, object)
			}
		})
	}
}

func TestPutImmutableObjectRetainsIdentityOnPostPutFailures(t *testing.T) {
	retention := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	for name, response := range map[string]wormProtocolResponse{
		"head http failure": {putVersion: "version-1", headStatus: http.StatusInternalServerError},
		"etag mismatch":     {putVersion: "version-1", headVersion: "version-1", headETag: "other-etag", headMode: "COMPLIANCE", headUntil: retention, headSize: 3, headHold: wormBool(true)},
		"missing hold":      {putVersion: "version-1", headVersion: "version-1", headMode: "COMPLIANCE", headUntil: retention, headSize: 3, headHold: wormBool(false)},
	} {
		t.Run(name, func(t *testing.T) {
			storage, _ := newWormProtocolStorage(t, response)
			object, err := storage.PutImmutableObject(context.Background(), ImmutableArchiveWrite{Key: "archive/document.pdf", Body: bytes.NewReader([]byte("pdf")), ContentLength: 3, RetentionUntil: retention, LegalHold: true})
			var verification *ImmutableArchiveWriteVerificationError
			if !errors.As(err, &verification) || object.Bucket != "archive-test" || object.Key != "archive/document.pdf" || object.VersionID != "version-1" {
				t.Fatalf("post-put failure err=%v object=%#v, want recoverable exact identity", err, object)
			}
		})
	}
}

func TestPutImmutableObjectMissingPutVersionRetainsKeyIdentity(t *testing.T) {
	storage, _ := newWormProtocolStorage(t, wormProtocolResponse{})
	object, err := storage.PutImmutableObject(context.Background(), ImmutableArchiveWrite{Key: "archive/document.pdf", Body: bytes.NewReader([]byte("pdf")), ContentLength: 3, RetentionUntil: time.Now().Add(time.Hour)})
	var verification *ImmutableArchiveWriteVerificationError
	if !errors.As(err, &verification) || object.Bucket != "archive-test" || object.Key != "archive/document.pdf" || object.VersionID != "" {
		t.Fatalf("missing version err=%v object=%#v, want typed recovery identity without adopted version", err, object)
	}
}

func TestPutImmutableObjectConditionalConflictDoesNotClaimVersion(t *testing.T) {
	retention := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	storage, observed := newWormProtocolStorage(t, wormProtocolResponse{putStatus: http.StatusPreconditionFailed})
	object, err := storage.PutImmutableObject(context.Background(), ImmutableArchiveWrite{Key: "archive/document.pdf", Body: bytes.NewReader([]byte("pdf")), ContentLength: 3, RetentionUntil: retention})
	var verification *ImmutableArchiveWriteVerificationError
	if err == nil || errors.As(err, &verification) || object.VersionID != "" || !observed.putConditional {
		t.Fatalf("conditional conflict err=%v object=%#v observed=%#v", err, object, observed)
	}
}

func TestPutVersionedObjectPersistsOnlyExactStorageVersion(t *testing.T) {
	storage, observed := newWormProtocolStorage(t, wormProtocolResponse{putVersion: "version-1", headVersion: "version-1", headSize: 3})
	object, err := storage.PutVersionedObject(context.Background(), "archive/document.pdf", "application/pdf", bytes.NewReader([]byte("pdf")), 3)
	if err != nil {
		t.Fatalf("versioned write: %v", err)
	}
	if object.VersionID != "version-1" || object.SizeBytes != 3 || observed.headVersion != "version-1" {
		t.Fatalf("versioned write did not verify exact identity: object=%#v observed=%#v", object, observed)
	}
}

func TestPutVersionedObjectFailsClosedWithoutStorageVersion(t *testing.T) {
	storage, _ := newWormProtocolStorage(t, wormProtocolResponse{headSize: 3})
	if _, err := storage.PutVersionedObject(context.Background(), "archive/document.pdf", "application/pdf", bytes.NewReader([]byte("pdf")), 3); err == nil {
		t.Fatal("versioned write must reject an object store without an exact version ID")
	}
}

func TestPutCustodyHeldObjectVerifiesExactHeldVersion(t *testing.T) {
	storage, observed := newWormProtocolStorage(t, wormProtocolResponse{putVersion: "version-1", headVersion: "version-1", headSize: 3, headHold: wormBool(true), headMeta: map[string]string{"portfolio-id": "p"}, getBody: "pdf", getVersion: "version-1", getETag: "etag-1"})
	object, err := storage.PutCustodyHeldObject(context.Background(), "archive/custody.pdf", "application/pdf", bytes.NewReader([]byte("pdf")), 3, "c35b21d6ca39aa7cc3b79a705d989f1a6e88b99ab43988d74048799e3db926a3", map[string]string{"portfolio-id": "p"})
	if err != nil || object.VersionID != "version-1" || !object.LegalHoldActive || observed.headVersion != "version-1" || observed.getVersion != "version-1" {
		t.Fatalf("custody held exact version=%#v err=%v observed=%#v", object, err, observed)
	}
}

func TestDiscoverCustodyHeldObjectVersion(t *testing.T) {
	intent := CustodyHeldArchiveRecovery{Key: "archive/custody.pdf", ExpectedSHA256: "c35b21d6ca39aa7cc3b79a705d989f1a6e88b99ab43988d74048799e3db926a3", ContentLength: 3, ContentType: "application/pdf", Metadata: map[string]string{"portfolio-id": "p"}}
	for name, response := range map[string]wormProtocolResponse{
		"zero":     {},
		"one":      {listedVersions: []string{"version-1"}, echoRequestedVersion: true, headSize: 3, headHold: wormBool(true), headMeta: map[string]string{"portfolio-id": "p"}, getBody: "pdf", getETag: "etag-1"},
		"multiple": {listedVersions: []string{"version-1", "version-2"}, echoRequestedVersion: true, headSize: 3, headHold: wormBool(true), headMeta: map[string]string{"portfolio-id": "p"}, getBody: "pdf", getETag: "etag-1"},
	} {
		t.Run(name, func(t *testing.T) {
			storage, _ := newWormProtocolStorage(t, response)
			object, err := storage.DiscoverCustodyHeldObjectVersion(context.Background(), intent)
			if name == "one" {
				if err != nil || object.VersionID != "version-1" {
					t.Fatalf("one candidate: object=%#v err=%v", object, err)
				}
				return
			}
			if err == nil || (name == "zero" && !errors.Is(err, errCustodyHeldObjectAbsent)) {
				t.Fatalf("%s candidates err=%v", name, err)
			}
		})
	}
}

type wormProtocolObserved struct {
	putConditional bool
	putCompliance  bool
	putLegalHold   bool
	putMetadata    string
	headVersion    string
	headCount      int
	getVersion     string
	getIfMatch     string
	putCount       int
	putUntil       time.Time
}

func newWormProtocolStorage(t *testing.T, response wormProtocolResponse) (*ArchiveStorage, *wormProtocolObserved) {
	t.Helper()
	observed := &wormProtocolObserved{}
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
		if r.Method == http.MethodGet && r.URL.Query().Has("versions") {
			w.Header().Set("Content-Type", "application/xml")
			var body strings.Builder
			body.WriteString(`<ListVersionsResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">`)
			for _, version := range response.listedVersions {
				fmt.Fprintf(&body, `<Version><Key>archive/custody.pdf</Key><VersionId>%s</VersionId><Size>3</Size><ETag>"etag-1"</ETag></Version>`, version)
			}
			body.WriteString(`</ListVersionsResult>`)
			_, _ = w.Write([]byte(body.String()))
			return
		}
		if r.Method == http.MethodPut {
			observed.putCount++
			observed.putUntil, _ = time.Parse(time.RFC3339Nano, r.Header.Get("X-Amz-Object-Lock-Retain-Until-Date"))
			observed.putConditional = r.Header.Get("If-None-Match") == "*"
			observed.putCompliance = r.Header.Get("X-Amz-Object-Lock-Mode") == "COMPLIANCE"
			observed.putLegalHold = r.Header.Get("X-Amz-Object-Lock-Legal-Hold") == "ON"
			observed.putMetadata = r.Header.Get("X-Amz-Meta-Policy-Version")
			if response.putStatus != 0 {
				http.Error(w, "precondition failed", response.putStatus)
				return
			}
			w.Header().Set("X-Amz-Version-Id", response.putVersion)
			w.Header().Set("ETag", `"etag-1"`)
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == http.MethodHead {
			observed.headCount++
			observed.headVersion = r.URL.Query().Get("versionId")
			if response.headStatus != 0 {
				http.Error(w, "head failed", response.headStatus)
				return
			}
			headVersion := response.headVersion
			if response.echoRequestedVersion {
				headVersion = r.URL.Query().Get("versionId")
			}
			w.Header().Set("X-Amz-Version-Id", headVersion)
			headETag := response.headETag
			if headETag == "" {
				headETag = "etag-1"
			}
			w.Header().Set("ETag", `"`+headETag+`"`)
			w.Header().Set("Content-Length", strconv.FormatInt(response.headSize, 10))
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("X-Amz-Object-Lock-Mode", response.headMode)
			w.Header().Set("X-Amz-Object-Lock-Retain-Until-Date", response.headUntil.UTC().Format(time.RFC3339Nano))
			if response.headHold == nil || *response.headHold {
				w.Header().Set("X-Amz-Object-Lock-Legal-Hold", "ON")
			}
			for key, value := range response.headMeta {
				w.Header().Set("X-Amz-Meta-"+key, value)
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == http.MethodGet && r.URL.Query().Get("versionId") != "" {
			observed.getVersion = r.URL.Query().Get("versionId")
			observed.getIfMatch = r.Header.Get("If-Match")
			getVersion := response.getVersion
			if response.echoRequestedVersion {
				getVersion = r.URL.Query().Get("versionId")
			}
			w.Header().Set("X-Amz-Version-Id", getVersion)
			w.Header().Set("ETag", `"`+response.getETag+`"`)
			w.Header().Set("Content-Length", strconv.Itoa(len(response.getBody)))
			w.Header().Set("Content-Type", "application/pdf")
			_, _ = w.Write([]byte(response.getBody))
			return
		}
		http.Error(w, "unexpected S3 protocol request "+r.Method+" "+r.URL.String(), http.StatusMethodNotAllowed)
	}))
	t.Cleanup(server.Close)
	storage, err := NewArchiveStorage(context.Background(), config.Config{ArchiveStorageEndpoint: server.URL, ArchiveStorageRegion: "us-east-1", ArchiveStorageBucket: "archive-test", ArchiveStorageAccessKey: "test", ArchiveStorageSecretKey: "test", ArchiveStorageUsePathStyle: true, ArchiveStorageRequireObjectLock: true})
	if err != nil {
		t.Fatalf("new WORM storage: %v", err)
	}
	return storage, observed
}

func wormBool(value bool) *bool { return &value }

func TestPutImmutableObjectNeverTruncatesPolicyDeadline(t *testing.T) {
	deadline := time.Now().UTC().Add(time.Hour).Truncate(time.Second).Add(123456 * time.Microsecond)
	storedDeadline := deadline.Truncate(time.Millisecond).Add(time.Millisecond)
	storage, observed := newWormProtocolStorage(t, wormProtocolResponse{
		putVersion: "version-1", headVersion: "version-1", headMode: "COMPLIANCE",
		headUntil: storedDeadline, headSize: 3,
	})
	_, err := storage.PutImmutableObject(context.Background(), ImmutableArchiveWrite{
		Key: "source.pdf", Body: strings.NewReader("pdf"), ContentLength: 3, RetentionUntil: deadline,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !observed.putUntil.Equal(storedDeadline) || observed.putUntil.Before(deadline) {
		t.Fatalf("storage deadline %s shortened or excessively extended policy %s", observed.putUntil, deadline)
	}
}

func TestPutImmutableObjectRejectsDuplicateNormalizedMetadata(t *testing.T) {
	storage, _ := newWormProtocolStorage(t, wormProtocolResponse{})
	_, err := storage.PutImmutableObject(context.Background(), ImmutableArchiveWrite{Key: "x", Body: strings.NewReader("x"), ContentLength: 1, RetentionUntil: time.Now().Add(time.Hour), Metadata: map[string]string{"Policy": "one", "policy": "two"}})
	if err == nil || !strings.Contains(err.Error(), "duplicate key") {
		t.Fatalf("duplicate normalized metadata err=%v", err)
	}
}
