package earchiva

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"
)

func TestReconcileImmutableObject(t *testing.T) {
	deadline := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	digest := sha256.Sum256([]byte("pdf"))
	for _, knownVersion := range []string{"", "version-1"} {
		name := "known version"
		if knownVersion == "" {
			name = "lost PUT response discovers then pins version"
		}
		t.Run(name, func(t *testing.T) {
			storage, observed := newWormProtocolStorage(t, wormProtocolResponse{
				headVersion: "version-1", headETag: "etag-1", headSize: 3,
				headMode: "COMPLIANCE", headUntil: deadline, headHold: wormBool(true),
				headMeta: map[string]string{"intent-id": "intent-1"},
				getBody:  "pdf", getVersion: "version-1", getETag: "etag-1",
			})
			object, err := storage.ReconcileImmutableObject(context.Background(), ImmutableArchiveRecovery{
				Key: "intent/source.pdf", VersionID: knownVersion, ContentLength: 3,
				ExpectedSHA256: hex.EncodeToString(digest[:]), RetentionUntil: deadline,
				LegalHold: true, Metadata: map[string]string{"intent-id": "intent-1"},
			})
			if err != nil {
				t.Fatal(err)
			}
			wantHeads := 1
			if knownVersion == "" {
				wantHeads = 2
			}
			if observed.headCount != wantHeads || observed.putCount != 0 {
				t.Fatalf("recovery performed unexpected reads/writes: %#v", observed)
			}
			if observed.getVersion != "version-1" || observed.getIfMatch != `"etag-1"` {
				t.Fatalf("GET did not pin exact version and ETag: %#v", observed)
			}
			if object.VersionID != "version-1" || object.Metadata["intent-id"] != "intent-1" {
				t.Fatalf("incorrect recovery result: %#v", object)
			}
		})
	}
}

func TestReconcileImmutableObjectRejectsMismatches(t *testing.T) {
	deadline := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	digest := sha256.Sum256([]byte("pdf"))
	cases := []struct {
		name   string
		change func(*wormProtocolResponse)
	}{
		{name: "missing version", change: func(r *wormProtocolResponse) { r.headVersion = "" }},
		{name: "null version", change: func(r *wormProtocolResponse) { r.headVersion = "null" }},
		{name: "wrong intent", change: func(r *wormProtocolResponse) { r.headMeta["intent-id"] = "other" }},
		{name: "governance", change: func(r *wormProtocolResponse) { r.headMode = "GOVERNANCE" }},
		{name: "short retention", change: func(r *wormProtocolResponse) { r.headUntil = deadline.Add(-time.Second) }},
		{name: "missing hold", change: func(r *wormProtocolResponse) { r.headHold = wormBool(false) }},
		{name: "GET version mismatch", change: func(r *wormProtocolResponse) { r.getVersion = "other" }},
		{name: "GET ETag mismatch", change: func(r *wormProtocolResponse) { r.getETag = "other" }},
		{name: "same size changed bytes", change: func(r *wormProtocolResponse) { r.getBody = "bad" }},
		{name: "longer bytes", change: func(r *wormProtocolResponse) { r.getBody = "pdfextra" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			response := wormProtocolResponse{
				headVersion: "version-1", headETag: "etag-1", headSize: 3,
				headMode: "COMPLIANCE", headUntil: deadline, headHold: wormBool(true),
				headMeta: map[string]string{"intent-id": "intent-1"},
				getBody:  "pdf", getVersion: "version-1", getETag: "etag-1",
			}
			tc.change(&response)
			storage, observed := newWormProtocolStorage(t, response)
			_, err := storage.ReconcileImmutableObject(context.Background(), ImmutableArchiveRecovery{
				Key: "intent/source.pdf", VersionID: "version-1", ContentLength: 3,
				ExpectedSHA256: hex.EncodeToString(digest[:]), RetentionUntil: deadline,
				LegalHold: true, Metadata: map[string]string{"intent-id": "intent-1"},
			})
			var mismatch *ImmutableArchiveWriteVerificationError
			if !errors.As(err, &mismatch) || observed.putCount != 0 {
				t.Fatalf("must fail read-only recovery: err=%v observations=%#v", err, observed)
			}
		})
	}
}
