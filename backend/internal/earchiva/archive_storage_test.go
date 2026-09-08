package earchiva

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eguilde/egueducation/internal/config"
)

func TestArchiveStorageCopiesObjectWithinHTTPMinIOEndpoint(t *testing.T) {
	t.Parallel()

	var copySource string
	var copiedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			w.WriteHeader(http.StatusOK)
		case http.MethodPut:
			copySource = r.Header.Get("X-Amz-Copy-Source")
			var err error
			copiedBody, err = io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read copy body: %v", err)
			}
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `<CopyObjectResult><ETag>&quot;test&quot;</ETag><LastModified>2026-09-08T00:00:00Z</LastModified></CopyObjectResult>`)
		default:
			http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	storage, err := NewArchiveStorage(context.Background(), config.Config{
		ArchiveStorageEndpoint:     server.URL,
		ArchiveStorageRegion:       "us-east-1",
		ArchiveStorageBucket:       "archive-test",
		ArchiveStorageAccessKey:    "test-access-key",
		ArchiveStorageSecretKey:    "test-secret-key",
		ArchiveStorageUsePathStyle: true,
	})
	if err != nil {
		t.Fatalf("create archive storage: %v", err)
	}

	if err := storage.CopyObject(context.Background(), "registratura/source document.pdf", "archive/document.pdf", "application/pdf"); err != nil {
		t.Fatalf("copy archive object: %v", err)
	}
	if copySource != "archive-test%2Fregistratura%2Fsource%20document.pdf" {
		t.Fatalf("copy source = %q", copySource)
	}
	if len(copiedBody) != 0 {
		t.Fatalf("server-side copy unexpectedly uploaded %d body bytes", len(copiedBody))
	}
}
