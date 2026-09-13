package earchiva

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

var errCustodyHeldObjectAbsent = fmt.Errorf("custody-held object is absent")

// ImmutableArchiveRecovery comes only from a persisted, authorized ingestion
// intent. An unknown VersionID permits discovery, never adoption by key alone.
type ImmutableArchiveRecovery struct {
	Key            string
	VersionID      string
	ExpectedSHA256 string
	ContentLength  int64
	RetentionUntil time.Time
	LegalHold      bool
	Metadata       map[string]string
}

// CustodyHeldArchiveRecovery may adopt only a VersionId previously recorded
// by a durable portfolio ingestion intent. It never heads or reads a mutable
// current key.
type CustodyHeldArchiveRecovery struct {
	Key            string
	VersionID      string
	ExpectedSHA256 string
	ContentLength  int64
	ETag           string
	ContentType    string
	Metadata       map[string]string
}

// DiscoverCustodyHeldObjectVersion reconciles a PUT whose response was lost.
// Discovery is constrained to the exact intent key and accepts exactly one
// version that independently verifies against the persisted provenance.
func (s *ArchiveStorage) DiscoverCustodyHeldObjectVersion(ctx context.Context, intent CustodyHeldArchiveRecovery) (ImmutableArchiveObject, error) {
	if strings.TrimSpace(intent.VersionID) != "" {
		return ImmutableArchiveObject{}, fmt.Errorf("custody discovery requires an unknown version id")
	}
	if !s.Enabled() || !s.requireObjectLock || intent.Key == "" || intent.ContentLength <= 0 || intent.ContentLength > archiveUploadMaxBytes || len(intent.Metadata) == 0 {
		return ImmutableArchiveObject{}, fmt.Errorf("custody discovery requires bounded exact intent provenance")
	}
	versions, err := s.client.ListObjectVersions(ctx, &s3.ListObjectVersionsInput{Bucket: aws.String(s.bucket), Prefix: aws.String(intent.Key)})
	if err != nil {
		return ImmutableArchiveObject{}, fmt.Errorf("list exact custody object versions: %w", err)
	}
	if aws.ToBool(versions.IsTruncated) {
		return ImmutableArchiveObject{}, fmt.Errorf("custody version discovery is ambiguous")
	}
	var matched []ImmutableArchiveObject
	exactVersionCount := 0
	for _, version := range versions.Versions {
		if aws.ToString(version.Key) != intent.Key || aws.ToString(version.VersionId) == "" || aws.ToString(version.VersionId) == "null" {
			continue
		}
		exactVersionCount++
		if aws.ToInt64(version.Size) != intent.ContentLength {
			continue
		}
		candidate := intent
		candidate.VersionID = aws.ToString(version.VersionId)
		candidate.ETag = strings.Trim(aws.ToString(version.ETag), `"`)
		object, reconcileErr := s.ReconcileCustodyHeldObject(ctx, candidate)
		if reconcileErr == nil {
			matched = append(matched, object)
		}
	}
	if exactVersionCount == 0 {
		return ImmutableArchiveObject{}, errCustodyHeldObjectAbsent
	}
	if len(matched) != 1 {
		return ImmutableArchiveObject{}, fmt.Errorf("custody version discovery found %d verified candidates", len(matched))
	}
	return matched[0], nil
}

func (s *ArchiveStorage) ReconcileCustodyHeldObject(ctx context.Context, intent CustodyHeldArchiveRecovery) (ImmutableArchiveObject, error) {
	if !s.Enabled() || !s.requireObjectLock {
		return ImmutableArchiveObject{}, fmt.Errorf("custody recovery requires Object Lock storage")
	}
	if intent.Key == "" || strings.TrimSpace(intent.Key) != intent.Key || strings.TrimSpace(intent.VersionID) == "" || intent.VersionID == "null" || strings.TrimSpace(intent.ETag) == "" || intent.ContentLength <= 0 || intent.ContentLength > archiveUploadMaxBytes || strings.TrimSpace(intent.ContentType) == "" || len(intent.Metadata) == 0 {
		return ImmutableArchiveObject{}, fmt.Errorf("custody recovery requires exact verified intent provenance")
	}
	if _, err := hex.DecodeString(intent.ExpectedSHA256); err != nil || len(intent.ExpectedSHA256) != sha256.Size*2 {
		return ImmutableArchiveObject{}, fmt.Errorf("custody recovery requires a SHA256 digest")
	}
	object := ImmutableArchiveObject{Bucket: s.bucket, Key: intent.Key, VersionID: intent.VersionID}
	head, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(intent.Key), VersionId: aws.String(intent.VersionID)})
	if err != nil {
		return object, &ImmutableArchiveWriteVerificationError{Object: object, Err: fmt.Errorf("head exact custody version: %w", err)}
	}
	object.ETag = strings.Trim(aws.ToString(head.ETag), `"`)
	object.SizeBytes = aws.ToInt64(head.ContentLength)
	object.LegalHoldActive = head.ObjectLockLegalHoldStatus == s3types.ObjectLockLegalHoldStatusOn
	object.Metadata = make(map[string]string, len(head.Metadata))
	for key, value := range head.Metadata {
		object.Metadata[strings.ToLower(strings.TrimSpace(key))] = strings.TrimSpace(value)
	}
	if aws.ToString(head.VersionId) != intent.VersionID || object.ETag != strings.Trim(intent.ETag, `"`) || object.SizeBytes != intent.ContentLength || strings.TrimSpace(aws.ToString(head.ContentType)) != strings.TrimSpace(intent.ContentType) || !object.LegalHoldActive || !immutableMetadataMatches(intent.Metadata, object.Metadata) {
		return object, recoveryMismatch(object, "custody recovery provenance mismatch")
	}
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(intent.Key), VersionId: aws.String(intent.VersionID), IfMatch: aws.String(`"` + object.ETag + `"`)})
	if err != nil {
		return object, &ImmutableArchiveWriteVerificationError{Object: object, Err: fmt.Errorf("read exact custody version: %w", err)}
	}
	defer output.Body.Close()
	if aws.ToString(output.VersionId) != intent.VersionID || strings.Trim(aws.ToString(output.ETag), `"`) != object.ETag || aws.ToInt64(output.ContentLength) != intent.ContentLength || strings.TrimSpace(aws.ToString(output.ContentType)) != strings.TrimSpace(intent.ContentType) {
		return object, recoveryMismatch(object, "custody recovery GET identity or size mismatch")
	}
	digest := sha256.New()
	size, err := io.Copy(digest, io.LimitReader(output.Body, intent.ContentLength+1))
	if err != nil || size != intent.ContentLength || !strings.EqualFold(hex.EncodeToString(digest.Sum(nil)), intent.ExpectedSHA256) {
		return object, recoveryMismatch(object, "custody recovery content mismatch")
	}
	return object, nil
}

// ReconcileImmutableObject is read-only. Even after a lost PUT response, it
// pins the observed version and verifies both its WORM facts and actual bytes.
// A missing, mismatched or unreadable object is never repaired or deleted here.
func (s *ArchiveStorage) ReconcileImmutableObject(
	ctx context.Context,
	intent ImmutableArchiveRecovery,
) (ImmutableArchiveObject, error) {
	if !s.Enabled() || !s.requireObjectLock {
		return ImmutableArchiveObject{}, fmt.Errorf("archive recovery requires Object Lock storage")
	}
	if err := validateImmutableRecovery(intent); err != nil {
		return ImmutableArchiveObject{}, err
	}
	object := ImmutableArchiveObject{
		Bucket: s.bucket, Key: intent.Key, VersionID: intent.VersionID,
	}
	headInput := &s3.HeadObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(intent.Key)}
	if intent.VersionID != "" {
		headInput.VersionId = aws.String(intent.VersionID)
	}
	head, err := s.client.HeadObject(ctx, headInput)
	if err != nil {
		return object, &ImmutableArchiveWriteVerificationError{Object: object, Err: err}
	}
	observedVersion := aws.ToString(head.VersionId)
	if observedVersion == "" || observedVersion == "null" {
		return object, recoveryMismatch(object, "missing immutable version identity")
	}
	if intent.VersionID != "" && observedVersion != intent.VersionID {
		return object, recoveryMismatch(object, "version identity differs from intent")
	}
	object.VersionID = observedVersion
	// Discovery HEADs can race a new latest version. Re-read the pinned version
	// so all retained facts belong to the exact version subsequently hashed.
	if intent.VersionID == "" {
		headInput.VersionId = aws.String(observedVersion)
		head, err = s.client.HeadObject(ctx, headInput)
		if err != nil {
			return object, &ImmutableArchiveWriteVerificationError{Object: object, Err: err}
		}
	}
	object.ETag = strings.Trim(aws.ToString(head.ETag), `"`)
	object.SizeBytes = aws.ToInt64(head.ContentLength)
	object.RetentionUntil = aws.ToTime(head.ObjectLockRetainUntilDate).UTC()
	object.ObjectLockMode = head.ObjectLockMode
	object.LegalHoldActive = head.ObjectLockLegalHoldStatus == s3types.ObjectLockLegalHoldStatusOn
	object.Metadata = make(map[string]string, len(head.Metadata))
	for key, value := range head.Metadata {
		object.Metadata[strings.ToLower(key)] = value
	}
	identityMatches := aws.ToString(head.VersionId) == object.VersionID && object.ETag != ""
	retentionMatches := object.ObjectLockMode == s3types.ObjectLockModeCompliance &&
		!object.RetentionUntil.Before(intent.RetentionUntil)
	contentMatches := object.SizeBytes == intent.ContentLength &&
		immutableMetadataMatches(intent.Metadata, object.Metadata)
	holdMatches := !intent.LegalHold || object.LegalHoldActive
	if !identityMatches || !retentionMatches || !contentMatches || !holdMatches {
		return object, recoveryMismatch(object, "immutable recovery provenance mismatch")
	}
	if err := s.verifyRecoveredBytes(ctx, object, intent); err != nil {
		return object, &ImmutableArchiveWriteVerificationError{Object: object, Err: err}
	}
	return object, nil
}

func validateImmutableRecovery(intent ImmutableArchiveRecovery) error {
	if intent.Key == "" || strings.TrimSpace(intent.Key) != intent.Key {
		return fmt.Errorf("archive recovery requires an exact object key")
	}
	if intent.ContentLength <= 0 || intent.ContentLength > archiveUploadMaxBytes {
		return fmt.Errorf("archive recovery size is outside upload limits")
	}
	digest, err := hex.DecodeString(intent.ExpectedSHA256)
	if err != nil || len(digest) != sha256.Size {
		return fmt.Errorf("archive recovery requires a SHA256 digest")
	}
	if intent.RetentionUntil.IsZero() || len(intent.Metadata) == 0 {
		return fmt.Errorf("archive recovery requires retention and intent metadata")
	}
	seen := make(map[string]bool, len(intent.Metadata))
	for key, value := range intent.Metadata {
		normalized := strings.ToLower(key)
		invalidKey := key == "" || strings.TrimSpace(key) != key || seen[normalized]
		if invalidKey || strings.TrimSpace(value) == "" {
			return fmt.Errorf("archive recovery metadata is empty or ambiguous")
		}
		seen[normalized] = true
	}
	return nil
}

func (s *ArchiveStorage) verifyRecoveredBytes(
	ctx context.Context,
	object ImmutableArchiveObject,
	intent ImmutableArchiveRecovery,
) error {
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(object.Bucket), Key: aws.String(object.Key),
		VersionId: aws.String(object.VersionID), IfMatch: aws.String(`"` + object.ETag + `"`),
	})
	if err != nil {
		return fmt.Errorf("read exact immutable recovery version: %w", err)
	}
	defer output.Body.Close()
	identityMatches := aws.ToString(output.VersionId) == object.VersionID &&
		strings.Trim(aws.ToString(output.ETag), `"`) == object.ETag
	if !identityMatches || aws.ToInt64(output.ContentLength) != intent.ContentLength {
		return fmt.Errorf("immutable recovery GET identity or size mismatch")
	}
	digest := sha256.New()
	size, err := io.Copy(digest, io.LimitReader(output.Body, intent.ContentLength+1))
	if err != nil {
		return fmt.Errorf("hash immutable recovery bytes: %w", err)
	}
	if size != intent.ContentLength || !strings.EqualFold(hex.EncodeToString(digest.Sum(nil)), intent.ExpectedSHA256) {
		return fmt.Errorf("immutable recovery content does not match intent")
	}
	return nil
}

func recoveryMismatch(object ImmutableArchiveObject, reason string) error {
	return &ImmutableArchiveWriteVerificationError{Object: object, Err: fmt.Errorf("%s", reason)}
}
