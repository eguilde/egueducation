package earchiva

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/eguilde/egueducation/internal/auth"
	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/google/uuid"
)

// UploadPortfolioOwnedDocument uses the archive pipeline with a trusted,
// consumer-provided transaction boundary, never browser-supplied authority.
func (s *DocumentService) UploadPortfolioOwnedDocument(w http.ResponseWriter, r *http.Request, scope ScopedUploadScope) {
	_, ownerErr := uuid.Parse(scope.OwnerUserID)
	_, portfolioErr := uuid.Parse(scope.PortfolioID)
	validIdentity := scope.InstitutionID != "" && scope.ActorSubject != ""
	validCallbacks := scope.Authorize != nil && scope.PersistAccess != nil
	if !validIdentity || !validCallbacks || ownerErr != nil || portfolioErr != nil ||
		scope.InstitutionID != auth.CurrentInstitutionIDFromRequest(r) || scope.ActorSubject != auth.CurrentSubjectFromRequest(r) {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "portfolio_upload_scope_required"})
		return
	}
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_upload_authorization_failed"})
		return
	}
	err = scope.Authorize(r.Context(), tx)
	rollbackErr := tx.Rollback(r.Context())
	if err != nil || rollbackErr != nil {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "portfolio_upload_forbidden"})
		return
	}
	s.uploadDocument(w, r, &scope)
}

func portfolioArchivePayload(input portfolioUploadInput, scope ScopedUploadScope) archiveUploadPayload {
	keyData, _ := json.Marshal([]string{scope.InstitutionID, scope.ActorSubject, scope.OwnerUserID, scope.PortfolioID, input.IdempotencyKey})
	keyHash := sha256.Sum256(keyData)
	payload := archiveUploadPayload{
		Title: input.Title, SourceKind: "upload", SourceSystem: "education_portfolio_own",
		FileName: input.File.Filename, FileSize: input.File.Size, MimeType: "application/pdf",
		IdempotencyKey: "portfolio-upload:" + hex.EncodeToString(keyHash[:]),
		Metadata: map[string]any{
			"portfolio_id": scope.PortfolioID, "portfolio_owner_user_id": scope.OwnerUserID,
			"upload_purpose": "education_portfolio_own",
		},
	}
	if input.DocumentDate != "" {
		payload.DocumentDate = &input.DocumentDate
	}
	return payload
}

func portfolioUploadFingerprint(payload archiveUploadPayload) string {
	data, _ := json.Marshal(struct {
		Title    string
		Date     *string
		FileName string
		SHA256   string
		Size     int64
	}{payload.Title, payload.DocumentDate, payload.FileName, payload.ChecksumSHA256, payload.FileSize})
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func portfolioUploadReplayMatches(payload archiveUploadPayload, detail ArchiveDocumentDetail) bool {
	for _, field := range []string{"portfolio_id", "portfolio_owner_user_id", "upload_purpose", "portfolio_upload_fingerprint"} {
		want, ok := payload.Metadata[field].(string)
		got, stored := detail.Metadata[field].(string)
		if !ok || !stored || want == "" || got != want {
			return false
		}
	}
	return true
}

// portfolioUploadInput contains only teacher-writable metadata. Actor, scope,
// archive location, grants and provenance must be derived by the command.
type portfolioUploadInput struct {
	Title          string
	DocumentDate   string
	IdempotencyKey string
	File           *multipart.FileHeader
}

func parsePortfolioUploadInput(form *multipart.Form, key string) (portfolioUploadInput, error) {
	invalid := errors.New("invalid portfolio upload input")
	input := portfolioUploadInput{IdempotencyKey: strings.TrimSpace(key)}
	if form == nil || input.IdempotencyKey == "" || len(input.IdempotencyKey) > 200 {
		return input, invalid
	}
	for field, values := range form.Value {
		if field != "title" && field != "document_date" {
			return input, invalid
		}
		if len(values) != 1 {
			return input, invalid
		}
	}
	if len(form.Value["title"]) != 1 || len(form.File) != 1 || len(form.File["file"]) != 1 {
		return input, invalid
	}
	input.Title = strings.TrimSpace(form.Value["title"][0])
	if input.Title == "" || !utf8.ValidString(input.Title) || utf8.RuneCountInString(input.Title) > 300 {
		return input, invalid
	}
	if dates := form.Value["document_date"]; len(dates) == 1 {
		input.DocumentDate = dates[0]
		if _, err := time.Parse(time.DateOnly, input.DocumentDate); err != nil {
			return input, invalid
		}
	}
	input.File = form.File["file"][0]
	if input.File == nil || input.File.Size <= 0 || input.File.Size > archiveUploadMaxBytes {
		return input, invalid
	}
	// File name and browser MIME are never proof of format: the existing PDF
	// staging/parser/scanner boundary must validate the actual bytes.
	return input, nil
}
