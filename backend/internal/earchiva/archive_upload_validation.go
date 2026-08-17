package earchiva

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type stagedArchiveUpload struct {
	Path, SHA256 string
	Size         int64
	PageCount    int
}

// Keep parser memory bounded relative to the API container. Upload staging and
// malware scanning may be concurrent, but structural parsing is serialized.
var archivePDFValidationSlots = make(chan struct{}, 1)

func stageAndValidateArchivePDF(ctx context.Context, source io.Reader, scanner Scanner) (stagedArchiveUpload, error) {
	if scanner == nil {
		return stagedArchiveUpload{}, fmt.Errorf("archive malware scanner is unavailable")
	}
	temp, err := os.CreateTemp("", "egueducation-archive-upload-*.pdf")
	if err != nil {
		return stagedArchiveUpload{}, fmt.Errorf("create secure archive staging file: %w", err)
	}
	path := temp.Name()
	cleanup := func(e error) (stagedArchiveUpload, error) {
		_ = temp.Close()
		_ = os.Remove(path)
		return stagedArchiveUpload{}, e
	}
	if err := temp.Chmod(0o600); err != nil {
		return cleanup(fmt.Errorf("secure archive staging file: %w", err))
	}
	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(temp, hash), io.LimitReader(source, archiveUploadMaxBytes+1))
	if err != nil {
		return cleanup(fmt.Errorf("stage archive upload: %w", err))
	}
	if written <= 0 || written > archiveUploadMaxBytes {
		return cleanup(fmt.Errorf("archive file exceeds the %d MiB limit", archiveUploadMaxBytes>>20))
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(path)
		return stagedArchiveUpload{}, fmt.Errorf("close archive staging file: %w", err)
	}
	probe, err := os.Open(path)
	if err != nil {
		_ = os.Remove(path)
		return stagedArchiveUpload{}, fmt.Errorf("open staged archive file: %w", err)
	}
	header := make([]byte, 5)
	_, headerErr := io.ReadFull(probe, header)
	_ = probe.Close()
	if headerErr != nil || !bytes.Equal(header, []byte("%PDF-")) {
		_ = os.Remove(path)
		return stagedArchiveUpload{}, fmt.Errorf("archive upload must be a PDF")
	}
	// Scan the bounded bitstream before any structural interpretation. The
	// validation below is deliberately linear and does not invoke an in-process
	// PDF parser on attacker-controlled object graphs.
	f, err := os.Open(path)
	if err != nil {
		_ = os.Remove(path)
		return stagedArchiveUpload{}, fmt.Errorf("open staged archive file for scanning: %w", err)
	}
	clean, scanErr := scanner.Scan(ctx, f)
	_ = f.Close()
	if scanErr != nil {
		_ = os.Remove(path)
		return stagedArchiveUpload{}, fmt.Errorf("archive malware scan failed: %w", scanErr)
	}
	if !clean {
		_ = os.Remove(path)
		return stagedArchiveUpload{}, fmt.Errorf("archive malware scan rejected the file")
	}

	// Encrypted PDFs cannot be safely OCRed. This is a bounded streaming scan;
	// structural parsing happens in a separate, time- and memory-bounded process.
	encrypted, err := archiveFileContains(path, []byte("/Encrypt"))
	if err != nil {
		_ = os.Remove(path)
		return stagedArchiveUpload{}, fmt.Errorf("inspect staged archive file: %w", err)
	}
	if encrypted {
		_ = os.Remove(path)
		return stagedArchiveUpload{}, fmt.Errorf("encrypted PDFs are not accepted")
	}
	pages, err := validateArchivePDFInSubprocess(ctx, path)
	if err != nil {
		_ = os.Remove(path)
		return stagedArchiveUpload{}, err
	}
	if pages < 1 || pages > archiveUploadMaxPages {
		_ = os.Remove(path)
		return stagedArchiveUpload{}, fmt.Errorf("PDF page count is outside the allowed range")
	}
	return stagedArchiveUpload{Path: path, SHA256: hex.EncodeToString(hash.Sum(nil)), Size: written, PageCount: pages}, nil
}

func validateArchivePDFInSubprocess(ctx context.Context, path string) (int, error) {
	select {
	case archivePDFValidationSlots <- struct{}{}:
		defer func() { <-archivePDFValidationSlots }()
	case <-ctx.Done():
		return 0, fmt.Errorf("archive PDF validation cancelled")
	}
	executable := strings.TrimSpace(os.Getenv("ARCHIVE_PDF_VALIDATOR_PATH"))
	if executable == "" {
		name := "egueducation-pdf-validator"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		self, err := os.Executable()
		if err != nil {
			return 0, fmt.Errorf("resolve archive PDF validator: %w", err)
		}
		executable = filepath.Join(filepath.Dir(self), name)
	}
	validationCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	command := exec.CommandContext(validationCtx, executable, path, strconv.Itoa(archiveUploadMaxPages)) //nolint:gosec -- fixed executable and server-created path
	// The helper needs no application configuration or credentials. Scrubbing
	// its environment prevents DB, MinIO and Azure secrets from being inherited.
	command.Env = []string{"GOMEMLIMIT=128MiB", "GOMAXPROCS=1"}
	output, err := command.Output()
	if validationCtx.Err() != nil {
		return 0, fmt.Errorf("archive PDF validation timed out")
	}
	if err != nil {
		return 0, fmt.Errorf("invalid or corrupt PDF")
	}
	pages, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return 0, fmt.Errorf("invalid archive PDF validator response")
	}
	return pages, nil
}

func archiveFileContains(path string, needle []byte) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close() //nolint:errcheck
	if len(needle) == 0 {
		return true, nil
	}
	buffer := make([]byte, 64<<10)
	carry := make([]byte, 0, len(needle)-1)
	for {
		n, readErr := file.Read(buffer)
		if n > 0 {
			window := make([]byte, 0, len(carry)+n)
			window = append(window, carry...)
			window = append(window, buffer[:n]...)
			if bytes.Contains(window, needle) {
				return true, nil
			}
			keep := len(needle) - 1
			if keep > len(window) {
				keep = len(window)
			}
			carry = append(carry[:0], window[len(window)-keep:]...)
		}
		if readErr == io.EOF {
			return false, nil
		}
		if readErr != nil {
			return false, readErr
		}
	}
}

func canonicalArchiveStorageFileName(documentID string) string {
	return "document-" + strings.ToLower(strings.TrimSpace(documentID)) + ".pdf"
}

func deriveArchiveIdempotencyKey(payload archiveUploadPayload) string {
	input := strings.Join([]string{payload.ChecksumSHA256, payload.Title, payload.SourceKind, payload.ExternalReference}, "\x00")
	sum := sha256.Sum256([]byte(input))
	return "derived-" + hex.EncodeToString(sum[:])
}
