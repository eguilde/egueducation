package earchiva

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	errPortfolioCustodyIntentReserved   = errors.New("portfolio custody upload recovery required")
	errPortfolioCustodyIntentConflict   = errors.New("portfolio custody idempotency conflict")
	errPortfolioCustodyIntentTransition = errors.New("portfolio custody intent transition already observed")
)

type portfolioCustodyIntent struct {
	ID, Status, SHA256, MimeType, RequestFingerprint   string
	Size                                               int64
	StoredVersionID, StoredETag, DocumentID, VersionID string
	Bucket, ObjectKey                                  string
	Metadata                                           map[string]string
}

func (s *DocumentService) findPortfolioCustodyIntent(ctx context.Context, scope *ScopedUploadScope, payload archiveUploadPayload) (portfolioCustodyIntent, bool, error) {
	if scope == nil {
		return portfolioCustodyIntent{}, false, nil
	}
	var i portfolioCustodyIntent
	var metadataJSON []byte
	err := s.pool.QueryRow(ctx, `select id::text,status,expected_sha256,expected_size_bytes,expected_mime_type,expected_request_fingerprint,expected_metadata,stored_version_id,stored_etag,reserved_document_id::text,reserved_version_id::text,bucket_name,object_key from portfolio_custody_upload_intents where tenant_code=public.current_tenant_code() and institution_id=$1 and portfolio_id=$2::uuid and actor_subject=$3 and idempotency_key=$4`, scope.InstitutionID, scope.PortfolioID, scope.ActorSubject, payload.IdempotencyKey).Scan(&i.ID, &i.Status, &i.SHA256, &i.Size, &i.MimeType, &i.RequestFingerprint, &metadataJSON, &i.StoredVersionID, &i.StoredETag, &i.DocumentID, &i.VersionID, &i.Bucket, &i.ObjectKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return portfolioCustodyIntent{}, false, nil
	}
	if err != nil {
		return i, false, err
	}
	if i.SHA256 != payload.ChecksumSHA256 || i.Size != payload.FileSize || i.RequestFingerprint != portfolioUploadFingerprint(payload) {
		return i, true, errPortfolioCustodyIntentConflict
	}
	if err := json.Unmarshal(metadataJSON, &i.Metadata); err != nil {
		return i, true, fmt.Errorf("decode custody intent metadata: %w", err)
	}
	if i.MimeType != payload.MimeType {
		return i, true, errPortfolioCustodyIntentConflict
	}
	return i, true, nil
}

func (s *DocumentService) reservePortfolioCustodyIntent(ctx context.Context, scope *ScopedUploadScope, payload archiveUploadPayload, documentID, versionID string) (string, error) {
	if scope == nil {
		return "", nil
	}
	id := uuid.NewString()
	metadata, err := json.Marshal(portfolioCustodyMetadata(id, scope, payload.TenantCode, documentID, versionID, payload.ChecksumSHA256))
	if err != nil {
		return "", fmt.Errorf("encode custody intent metadata: %w", err)
	}
	err = s.pool.QueryRow(ctx, `insert into portfolio_custody_upload_intents(id,tenant_code,institution_id,portfolio_id,actor_subject,idempotency_key,expected_sha256,expected_size_bytes,expected_mime_type,expected_request_fingerprint,expected_metadata,bucket_name,object_key,reserved_document_id,reserved_version_id) values($1::uuid,public.current_tenant_code(),$2,$3::uuid,$4,$5,$6,$7,$8,$9,$10::jsonb,$11,$12,$13::uuid,$14::uuid) returning id::text`, id, scope.InstitutionID, scope.PortfolioID, scope.ActorSubject, payload.IdempotencyKey, payload.ChecksumSHA256, payload.FileSize, payload.MimeType, portfolioUploadFingerprint(payload), metadata, payload.OriginalBucket, payload.OriginalObjectKey, documentID, versionID).Scan(&id)
	return id, err
}

func portfolioCustodyMetadata(intentID string, scope *ScopedUploadScope, tenantCode, documentID, versionID, sha256 string) map[string]string {
	return map[string]string{
		"archive-custody-intent-id": intentID,
		"archive-document-id":       documentID,
		"archive-version-id":        versionID,
		"institution-id":            scope.InstitutionID,
		"tenant-code":               tenantCode,
		"portfolio-id":              scope.PortfolioID,
		"source-sha256":             sha256,
	}
}

func (s *DocumentService) markPortfolioCustodyIntentStored(ctx context.Context, intentID string, object ImmutableArchiveObject) error {
	if intentID == "" {
		return nil
	}
	if object.VersionID == "" || object.ETag == "" || object.SizeBytes < 1 || !object.LegalHoldActive {
		return fmt.Errorf("unverified custody object")
	}
	tag, err := s.pool.Exec(ctx, `update portfolio_custody_upload_intents set status='stored',stored_version_id=$1,stored_etag=$2,stored_size_bytes=$3 where id=$4::uuid and status='reserved'`, object.VersionID, object.ETag, object.SizeBytes, intentID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("%w: affected %d rows", errPortfolioCustodyIntentTransition, tag.RowsAffected())
	}
	return nil
}

func commitPortfolioCustodyIntent(ctx context.Context, tx pgx.Tx, intentID string) error {
	if intentID == "" {
		return nil
	}
	tag, err := tx.Exec(ctx, `update portfolio_custody_upload_intents set status='committed',final_disposition='teacher_access',recovery_committed_at=now() where id=$1::uuid and status='stored' and final_disposition is null`, intentID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("commit custody intent affected %d rows", tag.RowsAffected())
	}
	return nil
}
