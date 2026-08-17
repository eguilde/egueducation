package earchiva

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestVerifyArchiveSourceFile(t *testing.T) {
	content := []byte("%PDF-1.7\nfixture")
	path := t.TempDir() + "/fixture.pdf"
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(content)
	if err := verifyArchiveSourceFile(path, hex.EncodeToString(digest[:]), int64(len(content))); err != nil {
		t.Fatalf("valid source rejected: %v", err)
	}
	if err := verifyArchiveSourceFile(path, strings.Repeat("0", 64), int64(len(content))); err == nil {
		t.Fatal("checksum mismatch must fail")
	}
	if err := verifyArchiveSourceFile(path, hex.EncodeToString(digest[:]), int64(len(content)+1)); err == nil {
		t.Fatal("size mismatch must fail")
	}
}

func TestArchiveRetryClassification(t *testing.T) {
	if !isRetryableArchiveError(fmt.Errorf("Azure OCR poll returned HTTP 429")) {
		t.Fatal("429 must retry")
	}
	if !isRetryableArchiveError(timeoutError{}) {
		t.Fatal("network timeout must retry")
	}
	if isRetryableArchiveError(errors.New("Azure OCR analysis failed (InvalidRequest)")) {
		t.Fatal("permanent request error must not retry")
	}
	if got := archiveRetryDelay(1); got != 15*time.Second {
		t.Fatalf("first delay = %s", got)
	}
}

type timeoutError struct{}

func (timeoutError) Error() string   { return "timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }
