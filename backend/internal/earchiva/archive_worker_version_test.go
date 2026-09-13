package earchiva

import (
	"context"
	"os"
	"testing"
)

func TestArchiveWorkerDownloadsPinnedSourceVersion(t *testing.T) {
	storage, observed := newWormProtocolStorage(t, wormProtocolResponse{
		getVersion: "original-version", getETag: "etag-1", getBody: "original bytes",
	})
	worker := &IngestionWorker{storage: storage}
	filename, err := worker.downloadToTempFile(
		context.Background(),
		"intent/source.pdf",
		"original-version",
		int64(len("original bytes")),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(filename) })
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if observed.getVersion != "original-version" || string(content) != "original bytes" {
		t.Fatalf("worker did not consume pinned original: version=%s content=%s", observed.getVersion, content)
	}
}
