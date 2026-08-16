package registratura

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/google/uuid"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	authruntime "github.com/eguilde/egueducation/internal/auth"
	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/go-chi/chi/v5"
)

const maxRegistraturaUpload = int64(100 << 20)

// Keep a small bound on concurrent disk-backed scans. This limits both clamd
// pressure and the amount of untrusted content retained on the local node.
var attachmentScanSlots = make(chan struct{}, 2)

func (s *Service) UploadDocumentAttachment(w http.ResponseWriter, r *http.Request) {
	if s.storage == nil || !s.storage.Enabled() {
		httpx.JSON(w, 503, map[string]any{"code": "attachment_storage_unavailable"})
		return
	}
	reader, err := r.MultipartReader()
	if err != nil {
		httpx.JSON(w, 400, map[string]any{"code": "invalid_attachment_upload"})
		return
	}
	documentID := chi.URLParam(r, "documentID")
	if s.scanner == nil {
		httpx.JSON(w, 503, map[string]any{"code": "attachment_scanner_unavailable"})
		return
	}
	select {
	case attachmentScanSlots <- struct{}{}:
		defer func() { <-attachmentScanSlots }()
	default:
		httpx.JSON(w, http.StatusTooManyRequests, map[string]any{"code": "attachment_scan_busy"})
		return
	}
	tmp, err := os.CreateTemp("", "egueducation-registratura-upload-*")
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "attachment_tempfile_failed"})
		return
	}
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
	}()
	if err := tmp.Chmod(0o600); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "attachment_tempfile_failed"})
		return
	}
	hash := sha256.New()
	var fileName, contentType, category string
	var size int64
	fileFound := false
	for {
		part, partErr := reader.NextPart()
		if partErr == io.EOF {
			break
		}
		if partErr != nil {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_attachment_upload"})
			return
		}
		if part.FormName() == "category" {
			value, readErr := io.ReadAll(io.LimitReader(part, 1025))
			_ = part.Close()
			if readErr != nil || len(value) > 1024 {
				httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_attachment_upload"})
				return
			}
			category = strings.TrimSpace(string(value))
			continue
		}
		if part.FormName() != "file" || fileFound {
			_, _ = io.Copy(io.Discard, io.LimitReader(part, 1025))
			_ = part.Close()
			continue
		}
		fileName = strings.TrimSpace(part.FileName())
		if fileName == "" || strings.ContainsAny(fileName, "\\/") || strings.Contains(fileName, "..") {
			_ = part.Close()
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_attachment_file"})
			return
		}
		contentType = strings.TrimSpace(part.Header.Get("Content-Type"))
		size, err = io.Copy(io.MultiWriter(tmp, hash), io.LimitReader(part, maxRegistraturaUpload+1))
		_ = part.Close()
		if err != nil || size <= 0 || size > maxRegistraturaUpload {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_attachment_file"})
			return
		}
		fileFound = true
	}
	if !fileFound {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_attachment_file"})
		return
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "attachment_tempfile_failed"})
		return
	}
	clean, err := s.scanner.Scan(r.Context(), tmp)
	if err != nil {
		httpx.JSON(w, 503, map[string]any{"code": "attachment_scanner_unavailable"})
		return
	}
	if !clean {
		httpx.JSON(w, 422, map[string]any{"code": "attachment_infected"})
		return
	}
	doc, err := s.loadDocument(r.Context(), documentID)
	if err != nil {
		httpx.JSON(w, 404, map[string]any{"code": "document_not_found"})
		return
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	key := attachmentObjectKey(strings.TrimSpace(authruntime.CurrentInstitutionIDFromRequest(r)), documentID, fileName)
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "attachment_tempfile_failed"})
		return
	}
	if err = s.storage.PutObject(r.Context(), key, contentType, tmp, size); err != nil {
		httpx.JSON(w, 502, map[string]any{"code": "attachment_storage_failed"})
		return
	}
	var item DocumentAttachment
	err = s.pool.QueryRow(r.Context(), `insert into registratura_document_attachments(document_id,title,file_name,mime_type,storage_key,size_bytes,category,status,uploaded_by,checksum_sha256,scan_status,storage_state) values($1::uuid,$2,$3,$4,$5,$6,$7,'ready',$8,$9,'clean','ready') returning id::text,document_id::text,title,file_name,mime_type,storage_key,size_bytes,category,status,uploaded_by,to_char(uploaded_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"')`, documentID, fileName, fileName, contentType, key, size, category, authruntime.CurrentSubjectFromRequest(r), hex.EncodeToString(hash.Sum(nil))).Scan(&item.ID, &item.DocumentID, &item.Title, &item.FileName, &item.MimeType, &item.StorageKey, &item.SizeBytes, &item.Category, &item.Status, &item.UploadedBy, &item.UploadedAt)
	if err != nil {
		_ = s.storage.DeleteObject(r.Context(), key)
		httpx.JSON(w, 500, map[string]any{"code": "attachment_persist_failed"})
		return
	}
	_ = doc
	httpx.JSON(w, 201, item)
}

func attachmentObjectKey(institutionID, documentID, fileName string) string {
	return path.Join("tenants", institutionID, "registratura", documentID, "attachments", uuid.NewString(), strings.ReplaceAll(fileName, " ", "-"))
}

func (s *Service) DownloadDocumentAttachment(w http.ResponseWriter, r *http.Request) {
	if s.storage == nil || !s.storage.Enabled() {
		httpx.JSON(w, 503, map[string]any{"code": "attachment_storage_unavailable"})
		return
	}
	if _, err := s.loadDocument(r.Context(), chi.URLParam(r, "documentID")); err != nil {
		httpx.JSON(w, 404, map[string]any{"code": "document_not_found"})
		return
	}
	var name, mime, key string
	err := s.pool.QueryRow(r.Context(), `select file_name,mime_type,storage_key from registratura_document_attachments where id::text=$1 and document_id::text=$2 and storage_state='ready' and scan_status='clean'`, chi.URLParam(r, "attachmentID"), chi.URLParam(r, "documentID")).Scan(&name, &mime, &key)
	if err != nil {
		httpx.JSON(w, 404, map[string]any{"code": "attachment_not_found"})
		return
	}
	body, err := s.storage.OpenObject(r.Context(), key)
	if err != nil {
		httpx.JSON(w, 502, map[string]any{"code": "attachment_download_failed"})
		return
	}
	defer body.Close()
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Content-Disposition", `attachment; filename="`+strings.ReplaceAll(name, `"`, "'")+`"`)
	_, _ = io.Copy(w, body)
}
