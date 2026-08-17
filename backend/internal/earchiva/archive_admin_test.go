package earchiva

import (
	"context"
	"testing"
)

func TestArchiveAdminPageBoundsInput(t *testing.T) {
	tests := []struct {
		raw  string
		want int
	}{
		{"", 25},
		{"0", 25},
		{"-3", 25},
		{"not-a-number", 25},
		{"26", 26},
		{"101", 100},
	}
	for _, test := range tests {
		if got := archiveAdminPage(test.raw, 25, 100); got != test.want {
			t.Fatalf("archiveAdminPage(%q) = %d, want %d", test.raw, got, test.want)
		}
	}
}

func TestArchiveAdminErrorSummaryDoesNotLeakRawError(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"Azure OCR rejected Binder1.pdf for ana.popescu@example.test", "ocr_processing_failed"},
		{"minio object scoalabalotesti/private/123.pdf cannot be read", "storage_operation_failed"},
		{"clamav detected content", "malware_scan_failed"},
		{"database processing error for secret file", "archive_processing_failed"},
	}
	for _, test := range tests {
		got := archiveAdminErrorSummary(test.raw)
		if got != test.want {
			t.Fatalf("summary = %q, want %q", got, test.want)
		}
		if got == test.raw || got == "" {
			t.Fatalf("raw error was exposed or omitted: %q", got)
		}
	}
}

func TestArchiveOCRProvider(t *testing.T) {
	if got := archiveOCRProvider(nil); got != "disabled" {
		t.Fatalf("nil provider = %q", got)
	}
	if got := archiveOCRProvider(testArchiveOCR{enabled: true}); got != "configured" {
		t.Fatalf("custom provider = %q", got)
	}
	if got := archiveOCRProvider(testArchiveOCR{}); got != "disabled" {
		t.Fatalf("disabled provider = %q", got)
	}
}

func TestArchiveJobStatusAllowlist(t *testing.T) {
	for _, status := range []string{"pending", "running", "succeeded", "failed"} {
		if !archiveJobStatus(status) {
			t.Fatalf("status %q was rejected", status)
		}
	}
	for _, status := range []string{"completed", "", "failed' or true"} {
		if archiveJobStatus(status) {
			t.Fatalf("invalid status %q was accepted", status)
		}
	}
}

type testArchiveOCR struct{ enabled bool }

func (o testArchiveOCR) Enabled() bool { return o.enabled }
func (testArchiveOCR) AnalyzeDocument(context.Context, string, string, int, string, string, string) (string, int, map[string]any, error) {
	return "", 0, nil, nil
}
