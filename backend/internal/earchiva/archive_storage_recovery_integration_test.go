//go:build integration

package earchiva

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/eguilde/egueducation/internal/config"
	"github.com/google/uuid"
)

// Run only against disposable MinIO. COMPLIANCE objects deliberately cannot
// be cleaned up by deleting versions; destroy the test volume after the run.
func TestImmutableArchiveRecoveryMinIO(t *testing.T) {
	endpoint := os.Getenv("TEST_WORM_MINIO_ENDPOINT")
	if endpoint == "" {
		t.Skip("TEST_WORM_MINIO_ENDPOINT must point to disposable MinIO")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	storage, err := NewArchiveStorage(ctx, config.Config{
		ArchiveStorageEndpoint: endpoint, ArchiveStorageRegion: "us-east-1",
		ArchiveStorageBucket:       "worm-recovery-" + uuid.NewString(),
		ArchiveStorageAccessKey:    os.Getenv("TEST_WORM_MINIO_ACCESS_KEY"),
		ArchiveStorageSecretKey:    os.Getenv("TEST_WORM_MINIO_SECRET_KEY"),
		ArchiveStorageUsePathStyle: true, ArchiveStorageCreateBucket: true,
		ArchiveStorageRequireObjectLock: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	content := "disposable WORM integration evidence"
	digest := sha256.Sum256([]byte(content))
	deadline := time.Now().UTC().Add(time.Hour).Truncate(time.Microsecond)
	metadata := map[string]string{"intent-id": uuid.NewString(), "tenant-code": "test-tenant"}
	write := ImmutableArchiveWrite{
		Key: "test-tenant/intent/source.pdf", ContentType: "application/pdf",
		Body: strings.NewReader(content), ContentLength: int64(len(content)),
		RetentionUntil: deadline, LegalHold: true, Metadata: metadata,
	}
	stored, err := storage.PutImmutableObject(ctx, write)
	if err != nil {
		t.Fatalf("real MinIO COMPLIANCE PUT: %v; required=%s observed=%s", err, deadline.Format(time.RFC3339Nano), stored.RetentionUntil.Format(time.RFC3339Nano))
	}
	intent := ImmutableArchiveRecovery{
		Key: write.Key, ExpectedSHA256: hex.EncodeToString(digest[:]),
		ContentLength: write.ContentLength, RetentionUntil: deadline,
		LegalHold: true, Metadata: metadata,
	}
	for _, version := range []string{stored.VersionID, ""} {
		intent.VersionID = version
		recovered, err := storage.ReconcileImmutableObject(ctx, intent)
		if err != nil {
			t.Fatalf("recover MinIO version %q: %v", version, err)
		}
		if recovered.VersionID != stored.VersionID || recovered.ETag != stored.ETag {
			t.Fatalf("recovery changed identity: stored=%#v recovered=%#v", stored, recovered)
		}
	}
	write.Body = strings.NewReader(content)
	if _, err := storage.PutImmutableObject(ctx, write); err == nil {
		t.Fatal("conditional replay overwrote existing immutable key")
	}
	intent.ExpectedSHA256 = strings.Repeat("0", 64)
	if _, err := storage.ReconcileImmutableObject(ctx, intent); err == nil {
		t.Fatal("recovery accepted wrong digest despite matching metadata")
	}
	intent.ExpectedSHA256 = hex.EncodeToString(digest[:])
	intent.Metadata = map[string]string{"intent-id": "another-intent"}
	if _, err := storage.ReconcileImmutableObject(ctx, intent); err == nil {
		t.Fatal("recovery adopted another intent's object")
	}
}
