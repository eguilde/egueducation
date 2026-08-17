package earchiva

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/eguilde/egueducation/internal/config"
)

func TestAzureDocumentIntelligenceOCRSubmitsAndPolls(t *testing.T) {
	var submitCount, pollCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Ocp-Apim-Subscription-Key") != "unit-test-key" {
			t.Errorf("missing subscription key")
		}
		switch r.URL.Path {
		case "/documentintelligence/documentModels/prebuilt-layout:analyze":
			submitCount++
			if got := r.URL.Query().Get("api-version"); got != "2024-11-30" {
				t.Errorf("api-version = %q", got)
			}
			body, _ := io.ReadAll(r.Body)
			if string(body) != "scanned-pdf" {
				t.Errorf("unexpected submitted body")
			}
			w.Header().Set("Operation-Location", serverURL(r)+"/operations/123")
			w.WriteHeader(http.StatusAccepted)
		case "/operations/123":
			pollCount++
			_, _ = w.Write([]byte(`{"status":"succeeded","analyzeResult":{"content":"Document OCR text","pages":[{},{}]}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cfg := config.Config{AzureDocumentIntelligenceEndpoint: server.URL, AzureDocumentIntelligenceKey: "unit-test-key", AzureDocumentIntelligenceModel: "prebuilt-layout", AzureDocumentIntelligenceAPIVersion: "2024-11-30"}
	ocr, err := newAzureDocumentIntelligenceOCR(cfg, server.Client())
	if err != nil {
		t.Fatalf("newAzureDocumentIntelligenceOCR() error = %v", err)
	}
	ocr.pollEvery = time.Millisecond
	file, err := os.CreateTemp(t.TempDir(), "scan-*.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("scanned-pdf"); err != nil {
		t.Fatal(err)
	}
	_ = file.Close()
	text, pages, metadata, err := ocr.AnalyzeDocument(context.Background(), "tenant-a", "doc-a", 1, file.Name(), "private.pdf", "application/pdf")
	if err != nil {
		t.Fatalf("AnalyzeDocument() error = %v", err)
	}
	if text != "Document OCR text" || pages != 2 {
		t.Fatalf("result = %q, %d", text, pages)
	}
	if metadata["ocr_source"] != "azure-document-intelligence" || submitCount != 1 || pollCount != 1 {
		t.Fatalf("metadata/counts = %#v/%d/%d", metadata, submitCount, pollCount)
	}
}

// serverURL reconstructs the httptest origin from the incoming request without
// binding the test to a particular local port.
func serverURL(r *http.Request) string { return "http://" + r.Host }

func TestAzureDocumentIntelligenceOCRRejectsPartialConfig(t *testing.T) {
	_, err := newAzureDocumentIntelligenceOCR(config.Config{AzureDocumentIntelligenceEndpoint: "https://example.test"}, http.DefaultClient)
	if err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("expected incomplete configuration error, got %v", err)
	}
}

func TestAzureOperationLocationMustRemainAtConfiguredOrigin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Operation-Location", "https://untrusted.example/operation/1")
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()
	cfg := config.Config{AzureDocumentIntelligenceEndpoint: server.URL, AzureDocumentIntelligenceKey: "unit-test-key", AzureDocumentIntelligenceModel: "prebuilt-layout", AzureDocumentIntelligenceAPIVersion: "2024-11-30"}
	ocr, err := newAzureDocumentIntelligenceOCR(cfg, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	file := t.TempDir() + "/scan.pdf"
	if err := os.WriteFile(file, []byte("scan"), 0600); err != nil {
		t.Fatal(err)
	}
	_, _, _, err = ocr.AnalyzeDocument(context.Background(), "tenant", "document", 1, file, "scan.pdf", "application/pdf")
	if err == nil || !strings.Contains(err.Error(), "outside") {
		t.Fatalf("expected operation-location rejection, got %v", err)
	}
}
