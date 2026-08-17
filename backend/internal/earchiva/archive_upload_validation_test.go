package earchiva

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"
)

type archiveTestScanner struct {
	clean bool
	calls int
}

func (s *archiveTestScanner) Scan(_ context.Context, r io.Reader) (bool, error) {
	s.calls++
	_, _ = io.Copy(io.Discard, r)
	return s.clean, nil
}

func TestStageArchiveUploadRejectsNonPDFBeforeScanner(t *testing.T) {
	s := &archiveTestScanner{clean: true}
	_, err := stageAndValidateArchivePDF(context.Background(), bytes.NewBufferString("not a pdf"), s)
	if err == nil {
		t.Fatal("expected invalid PDF rejection")
	}
	if s.calls != 0 {
		t.Fatal("scanner must not receive an invalid file")
	}
}

func TestCanonicalArchiveStorageFileNameAvoidsSourceName(t *testing.T) {
	got := canonicalArchiveStorageFileName("ABC-123")
	if got != "document-abc-123.pdf" {
		t.Fatalf("canonical name = %q", got)
	}
}

func TestArchiveDTODoesNotExposeStorageCoordinates(t *testing.T) {
	data, err := json.Marshal(ArchiveDocument{OriginalBucket: "archive", OriginalObjectKey: "tenant/secret.pdf", ArtifactBucket: "archive", ArtifactObjectKey: "artifact.json"})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte("secret.pdf")) || bytes.Contains(data, []byte("artifact.json")) {
		t.Fatalf("storage coordinates leaked: %s", data)
	}
}
