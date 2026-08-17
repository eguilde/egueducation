package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateRejectsMarkerOnlyPayload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fake.pdf")
	if err := os.WriteFile(path, []byte("%PDF-1.7\n%%EOF\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if pages, ok := validate(path, 2000); ok || pages != 0 {
		t.Fatalf("marker-only payload accepted: pages=%d ok=%v", pages, ok)
	}
}

func TestValidateAcceptsOnePagePDF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "one-page.pdf")
	if err := os.WriteFile(path, minimalPDF(), 0o600); err != nil {
		t.Fatal(err)
	}
	if pages, ok := validate(path, 2000); !ok || pages != 1 {
		t.Fatalf("valid PDF rejected: pages=%d ok=%v", pages, ok)
	}
	if _, ok := validate(path, 0); ok {
		t.Fatal("invalid page bound accepted")
	}
}

func minimalPDF() []byte {
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] /Contents 4 0 R >>",
		"<< /Length 0 >>\nstream\n\nendstream",
	}
	var output bytes.Buffer
	output.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for index, object := range objects {
		offsets[index+1] = output.Len()
		fmt.Fprintf(&output, "%d 0 obj\n%s\nendobj\n", index+1, object)
	}
	xref := output.Len()
	fmt.Fprintf(&output, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for index := 1; index <= len(objects); index++ {
		fmt.Fprintf(&output, "%010d 00000 n \n", offsets[index])
	}
	fmt.Fprintf(&output, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return output.Bytes()
}
