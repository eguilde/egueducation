package earchiva

import (
	"mime/multipart"
	"strings"
	"testing"
)

func TestParsePortfolioUploadInput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		change func(*multipart.Form)
		key    string
		valid  bool
	}{
		{name: "minimal", key: "retry-1", valid: true},
		{name: "dated", key: "retry-1", valid: true, change: func(f *multipart.Form) { f.Value["document_date"] = []string{"2026-09-12"} }},
		{name: "missing key"},
		{name: "duplicate title", key: "retry-1", change: func(f *multipart.Form) { f.Value["title"] = []string{"a", "b"} }},
		{name: "invalid date", key: "retry-1", change: func(f *multipart.Form) { f.Value["document_date"] = []string{"2026-02-30"} }},
		{name: "oversized", key: "retry-1", change: func(f *multipart.Form) { f.File["file"][0].Size = archiveUploadMaxBytes + 1 }},
		{name: "empty title", key: "retry-1", change: func(f *multipart.Form) { f.Value["title"] = []string{"  "} }},
		{name: "long title", key: "retry-1", change: func(f *multipart.Form) { f.Value["title"] = []string{strings.Repeat("x", 301)} }},
	}
	for _, field := range []string{"tenant_code", "institution_id", "owner_user_id", "grantee_user_id", "metadata", "taxonomy_code", "idempotency_key"} {
		tests = append(tests, struct {
			name   string
			change func(*multipart.Form)
			key    string
			valid  bool
		}{
			name: "reject authority field " + field, key: "retry-1",
			change: func(f *multipart.Form) { f.Value[field] = []string{"forged"} },
		})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			form := &multipart.Form{Value: map[string][]string{"title": {"Planificare"}}, File: map[string][]*multipart.FileHeader{"file": {{Filename: "lesson.pdf", Size: 100}}}}
			if tt.change != nil {
				tt.change(form)
			}
			_, err := parsePortfolioUploadInput(form, tt.key)
			if (err == nil) != tt.valid {
				t.Fatalf("valid=%t error=%v", tt.valid, err)
			}
		})
	}
}
