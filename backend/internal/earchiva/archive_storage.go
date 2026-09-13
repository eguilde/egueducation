package earchiva

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"
	"time"

	aws "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/eguilde/egueducation/internal/config"
)

type ArchiveStorage struct {
	client            *s3.Client
	bucket            string
	region            string
	enabled           bool
	createBucket      bool
	requireObjectLock bool
}

type ImmutableArchiveObject struct {
	Bucket          string
	Key             string
	VersionID       string
	ETag            string
	SizeBytes       int64
	RetentionUntil  time.Time
	ObjectLockMode  s3types.ObjectLockMode
	LegalHoldActive bool
	Metadata        map[string]string
}

// VersionedArchiveObject identifies the exact stored original. A key alone is
// mutable and must never be used as evidence provenance.
type VersionedArchiveObject struct {
	Bucket    string
	Key       string
	VersionID string
	ETag      string
	SizeBytes int64
}

// ImmutableArchiveWrite is server-owned storage intent. Callers derive its
// deadline and metadata from an approved retention policy; browser input must
// never be passed through as this structure.
type ImmutableArchiveWrite struct {
	Key            string
	ContentType    string
	Body           io.Reader
	ContentLength  int64
	RetentionUntil time.Time
	LegalHold      bool
	Metadata       map[string]string
}

// ImmutableArchiveWriteVerificationError means S3 accepted a version but the
// post-write HEAD check could not prove the required WORM facts. The returned
// object identity is safe to persist only as a recovery intent; it is not a
// verified archive version and must be reconciled before being made active.
type ImmutableArchiveWriteVerificationError struct {
	Object ImmutableArchiveObject
	Err    error
}

func (e *ImmutableArchiveWriteVerificationError) Error() string {
	return fmt.Sprintf("immutable archive object %s version %s requires reconciliation: %v", e.Object.Key, e.Object.VersionID, e.Err)
}

func (e *ImmutableArchiveWriteVerificationError) Unwrap() error { return e.Err }

func NewArchiveStorage(ctx context.Context, cfg config.Config) (*ArchiveStorage, error) {
	endpoint := strings.TrimSpace(cfg.ArchiveStorageEndpoint)
	bucket := strings.TrimSpace(cfg.ArchiveStorageBucket)
	if endpoint == "" || bucket == "" {
		return &ArchiveStorage{bucket: bucket, enabled: false}, nil
	}
	parsedEndpoint, err := url.Parse(endpoint)
	if err != nil || parsedEndpoint.Scheme == "" || parsedEndpoint.Host == "" {
		return nil, fmt.Errorf("invalid archive storage endpoint")
	}
	if cfg.IsProduction() && !strings.EqualFold(parsedEndpoint.Scheme, "https") {
		return nil, fmt.Errorf("archive storage endpoint must use HTTPS in production")
	}

	loadOptions := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(strings.TrimSpace(cfg.ArchiveStorageRegion)),
	}
	if accessKey := strings.TrimSpace(cfg.ArchiveStorageAccessKey); accessKey != "" {
		loadOptions = append(loadOptions, awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, cfg.ArchiveStorageSecretKey, "")))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return nil, fmt.Errorf("load archive storage config: %w", err)
	}
	// ArchiveStorage uses an explicitly configured S3-compatible endpoint
	// (currently MinIO). Newer AWS SDK releases calculate optional CRC32
	// checksums for every PutObject by default. That mode cannot relay a
	// non-seekable GetObject response over plain HTTP because the checksum
	// middleware would need either TLS trailers or a second read. Required-only
	// keeps checksums for operations whose protocol mandates them while allowing
	// bounded, content-length-delimited streaming copies without buffering an
	// entire archive document in memory.
	awsCfg.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = cfg.ArchiveStorageUsePathStyle
	})

	storage := &ArchiveStorage{
		client:            client,
		bucket:            bucket,
		region:            strings.TrimSpace(cfg.ArchiveStorageRegion),
		enabled:           true,
		createBucket:      cfg.ArchiveStorageCreateBucket,
		requireObjectLock: cfg.ArchiveStorageRequireObjectLock,
	}
	if err := storage.EnsureBucket(ctx); err != nil {
		return nil, err
	}
	if err := storage.VerifyWORMPolicy(ctx); err != nil {
		return nil, err
	}
	return storage, nil
}

func (s *ArchiveStorage) Enabled() bool {
	return s != nil && s.enabled && s.client != nil && strings.TrimSpace(s.bucket) != ""
}

func (s *ArchiveStorage) Bucket() string {
	if s == nil {
		return ""
	}
	return s.bucket
}

func (s *ArchiveStorage) EnsureBucket(ctx context.Context) error {
	if !s.Enabled() {
		return nil
	}

	if _, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(s.bucket)}); err == nil {
		return nil
	} else if !s.createBucket {
		return fmt.Errorf("archive bucket is unavailable")
	}

	input := &s3.CreateBucketInput{Bucket: aws.String(s.bucket), ObjectLockEnabledForBucket: aws.Bool(s.requireObjectLock)}
	if region := strings.TrimSpace(s.region); region != "" && region != "us-east-1" {
		input.CreateBucketConfiguration = &s3types.CreateBucketConfiguration{
			LocationConstraint: s3types.BucketLocationConstraint(region),
		}
	}

	if _, err := s.client.CreateBucket(ctx, input); err != nil {
		message := err.Error()
		if strings.Contains(message, "BucketAlreadyOwnedByYou") || strings.Contains(message, "BucketAlreadyExists") {
			return nil
		}
		return fmt.Errorf("ensure archive bucket %s: %w", s.bucket, err)
	}
	return nil
}

// VerifyWORMPolicy proves the bucket prerequisites instead of treating an
// immutable-looking key as WORM. Retention itself is deliberately supplied per
// object by PutImmutableObject from the applicable archive-series policy. The
// check is read-only and does not silently reconfigure an existing bucket.
func (s *ArchiveStorage) VerifyWORMPolicy(ctx context.Context) error {
	if !s.Enabled() || !s.requireObjectLock {
		return nil
	}
	versioning, err := s.client.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{Bucket: aws.String(s.bucket)})
	if err != nil {
		return fmt.Errorf("verify archive bucket versioning: %w", err)
	}
	if versioning.Status != s3types.BucketVersioningStatusEnabled {
		return fmt.Errorf("archive bucket must have versioning enabled for WORM storage")
	}
	lock, err := s.client.GetObjectLockConfiguration(ctx, &s3.GetObjectLockConfigurationInput{Bucket: aws.String(s.bucket)})
	if err != nil {
		return fmt.Errorf("verify archive bucket Object Lock: %w", err)
	}
	if lock.ObjectLockConfiguration == nil || lock.ObjectLockConfiguration.ObjectLockEnabled != s3types.ObjectLockEnabledEnabled {
		return fmt.Errorf("archive bucket must have Object Lock enabled")
	}
	return nil
}

func (s *ArchiveStorage) PutObject(ctx context.Context, key, contentType string, body io.Reader, contentLength int64) error {
	if !s.Enabled() {
		return fmt.Errorf("archive storage is disabled")
	}

	input := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   body,
	}
	if strings.TrimSpace(contentType) != "" {
		input.ContentType = aws.String(contentType)
	}
	if contentLength >= 0 {
		input.ContentLength = aws.Int64(contentLength)
	}

	if _, err := s.client.PutObject(ctx, input); err != nil {
		return fmt.Errorf("put archive object %s: %w", key, err)
	}
	return nil
}

// PutVersionedObject requires bucket versioning and verifies the returned
// exact version before returning it for authoritative persistence.
func (s *ArchiveStorage) PutVersionedObject(ctx context.Context, key, contentType string, body io.Reader, contentLength int64) (VersionedArchiveObject, error) {
	if !s.Enabled() {
		return VersionedArchiveObject{}, fmt.Errorf("archive storage is disabled")
	}
	versioning, err := s.client.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{Bucket: aws.String(s.bucket)})
	if err != nil {
		return VersionedArchiveObject{}, fmt.Errorf("verify archive bucket versioning: %w", err)
	}
	if versioning.Status != s3types.BucketVersioningStatusEnabled {
		return VersionedArchiveObject{}, fmt.Errorf("archive bucket versioning must be enabled for original evidence")
	}
	input := &s3.PutObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key), Body: body}
	if strings.TrimSpace(contentType) != "" {
		input.ContentType = aws.String(contentType)
	}
	if contentLength >= 0 {
		input.ContentLength = aws.Int64(contentLength)
	}
	output, err := s.client.PutObject(ctx, input)
	if err != nil {
		return VersionedArchiveObject{}, fmt.Errorf("put versioned archive object %s: %w", key, err)
	}
	object := VersionedArchiveObject{Bucket: s.bucket, Key: key, VersionID: strings.TrimSpace(aws.ToString(output.VersionId)), ETag: strings.Trim(aws.ToString(output.ETag), `"`)}
	if object.VersionID == "" || object.VersionID == "null" {
		return VersionedArchiveObject{}, fmt.Errorf("archive object has no storage version id")
	}
	head, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key), VersionId: aws.String(object.VersionID)})
	if err != nil {
		return VersionedArchiveObject{}, fmt.Errorf("head exact versioned archive object: %w", err)
	}
	object.SizeBytes = aws.ToInt64(head.ContentLength)
	object.ETag = strings.Trim(aws.ToString(head.ETag), `"`)
	if strings.TrimSpace(aws.ToString(head.VersionId)) != object.VersionID || (contentLength >= 0 && object.SizeBytes != contentLength) {
		return VersionedArchiveObject{}, fmt.Errorf("versioned archive object verification mismatch")
	}
	return object, nil
}

// PutCustodyHeldObject is restricted to active-record custody: it requires
// Object Lock and a verified ON legal hold, but deliberately records no
// cessation-derived retention deadline.
func (s *ArchiveStorage) PutCustodyHeldObject(ctx context.Context, key, contentType string, body io.Reader, contentLength int64, expectedSHA256 string, metadata map[string]string) (ImmutableArchiveObject, error) {
	if !s.Enabled() || !s.requireObjectLock || contentLength < 0 || strings.TrimSpace(key) == "" || body == nil {
		return ImmutableArchiveObject{}, fmt.Errorf("custody-held archive write requires Object Lock, key, body, and length")
	}
	normalizedMetadata := make(map[string]string, len(metadata))
	for metadataKey, value := range metadata {
		metadataKey, value = strings.ToLower(strings.TrimSpace(metadataKey)), strings.TrimSpace(value)
		if metadataKey == "" || value == "" {
			return ImmutableArchiveObject{}, fmt.Errorf("custody-held archive metadata must have non-empty keys and values")
		}
		if _, exists := normalizedMetadata[metadataKey]; exists {
			return ImmutableArchiveObject{}, fmt.Errorf("custody-held archive metadata has duplicate key %q", metadataKey)
		}
		normalizedMetadata[metadataKey] = value
	}
	input := &s3.PutObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key), Body: body, ContentLength: aws.Int64(contentLength), IfNoneMatch: aws.String("*"), ObjectLockLegalHoldStatus: s3types.ObjectLockLegalHoldStatusOn, Metadata: normalizedMetadata}
	if contentType = strings.TrimSpace(contentType); contentType != "" {
		input.ContentType = aws.String(contentType)
	}
	output, err := s.client.PutObject(ctx, input)
	if err != nil {
		return ImmutableArchiveObject{}, fmt.Errorf("put custody-held archive object: %w", err)
	}
	object := ImmutableArchiveObject{Bucket: s.bucket, Key: key, VersionID: strings.TrimSpace(aws.ToString(output.VersionId)), ETag: strings.Trim(aws.ToString(output.ETag), `"`)}
	if object.VersionID == "" || object.VersionID == "null" {
		return object, &ImmutableArchiveWriteVerificationError{Object: object, Err: fmt.Errorf("custody-held archive object has no storage version id")}
	}
	verified, err := s.ReconcileCustodyHeldObject(ctx, CustodyHeldArchiveRecovery{Key: key, VersionID: object.VersionID, ExpectedSHA256: expectedSHA256, ContentLength: contentLength, ETag: object.ETag, ContentType: contentType, Metadata: normalizedMetadata})
	if err != nil {
		return object, err
	}
	return verified, nil
}

// PutImmutableObject stores one new exact S3 version under COMPLIANCE
// retention. It refuses overwrite/retry writes with If-None-Match: *, then
// reads the exact returned VersionId back and verifies storage-observed WORM
// facts. It deliberately does not delete on any post-PUT error: a COMPLIANCE
// version may exist and must be reconciled through a durable caller intent.
func (s *ArchiveStorage) PutImmutableObject(ctx context.Context, write ImmutableArchiveWrite) (ImmutableArchiveObject, error) {
	if !s.Enabled() {
		return ImmutableArchiveObject{}, fmt.Errorf("archive storage is disabled")
	}
	if !s.requireObjectLock {
		return ImmutableArchiveObject{}, fmt.Errorf("archive immutable write requires verified Object Lock storage")
	}
	key := strings.TrimSpace(write.Key)
	if key == "" || write.Body == nil || write.ContentLength < 0 {
		return ImmutableArchiveObject{}, fmt.Errorf("archive immutable write requires key, body, and non-negative content length")
	}
	retainUntil := write.RetentionUntil.UTC()
	if !retainUntil.After(time.Now().UTC()) {
		return ImmutableArchiveObject{}, fmt.Errorf("archive immutable write requires a future retention deadline")
	}
	// Smithy's S3 timestamp serializer emits milliseconds. Round upward at
	// that boundary so a PostgreSQL microsecond deadline is never shortened.
	// Keep the policy deadline unchanged and validate the observed retention.
	storageDeadline := retainUntil.Truncate(time.Millisecond)
	if storageDeadline.Before(retainUntil) {
		storageDeadline = storageDeadline.Add(time.Millisecond)
	}
	metadata := make(map[string]string, len(write.Metadata))
	for key, value := range write.Metadata {
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if key == "" || value == "" {
			return ImmutableArchiveObject{}, fmt.Errorf("archive immutable write metadata must have non-empty keys and values")
		}
		normalizedKey := strings.ToLower(key)
		if _, exists := metadata[normalizedKey]; exists {
			return ImmutableArchiveObject{}, fmt.Errorf("archive immutable write metadata has duplicate key %q", key)
		}
		metadata[normalizedKey] = value
	}
	input := &s3.PutObjectInput{
		Bucket:                    aws.String(s.bucket),
		Key:                       aws.String(key),
		Body:                      write.Body,
		ObjectLockMode:            s3types.ObjectLockModeCompliance,
		ObjectLockRetainUntilDate: aws.Time(storageDeadline),
		IfNoneMatch:               aws.String("*"),
		Metadata:                  metadata,
	}
	if write.LegalHold {
		input.ObjectLockLegalHoldStatus = s3types.ObjectLockLegalHoldStatusOn
	}
	if contentType := strings.TrimSpace(write.ContentType); contentType != "" {
		input.ContentType = aws.String(contentType)
	}
	input.ContentLength = aws.Int64(write.ContentLength)
	output, err := s.client.PutObject(ctx, input)
	if err != nil {
		return ImmutableArchiveObject{}, fmt.Errorf("put immutable archive object %s: %w", key, err)
	}
	versionID := strings.TrimSpace(aws.ToString(output.VersionId))
	object := ImmutableArchiveObject{Bucket: s.bucket, Key: key, VersionID: versionID, ETag: strings.Trim(aws.ToString(output.ETag), `"`)}
	if versionID == "" || versionID == "null" {
		return object, &ImmutableArchiveWriteVerificationError{Object: object, Err: fmt.Errorf("immutable archive object has no storage version id")}
	}
	putETag := object.ETag
	head, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key), VersionId: aws.String(versionID)})
	if err != nil {
		return object, &ImmutableArchiveWriteVerificationError{Object: object, Err: fmt.Errorf("head exact immutable object version: %w", err)}
	}
	object.SizeBytes = aws.ToInt64(head.ContentLength)
	object.ETag = strings.Trim(aws.ToString(head.ETag), `"`)
	object.RetentionUntil = aws.ToTime(head.ObjectLockRetainUntilDate).UTC()
	object.ObjectLockMode = head.ObjectLockMode
	object.LegalHoldActive = head.ObjectLockLegalHoldStatus == s3types.ObjectLockLegalHoldStatusOn
	object.Metadata = make(map[string]string, len(head.Metadata))
	for key, value := range head.Metadata {
		object.Metadata[strings.ToLower(strings.TrimSpace(key))] = strings.TrimSpace(value)
	}
	if strings.TrimSpace(aws.ToString(head.VersionId)) != versionID || (putETag != "" && object.ETag != putETag) || object.SizeBytes != write.ContentLength || object.ObjectLockMode != s3types.ObjectLockModeCompliance || object.RetentionUntil.Before(retainUntil) || (write.LegalHold && !object.LegalHoldActive) || !immutableMetadataMatches(metadata, object.Metadata) {
		return object, &ImmutableArchiveWriteVerificationError{Object: object, Err: fmt.Errorf("immutable object verification mismatch")}
	}
	return object, nil
}

func immutableMetadataMatches(expected, observed map[string]string) bool {
	for key, value := range expected {
		if observed[strings.ToLower(strings.TrimSpace(key))] != value {
			return false
		}
	}
	return true
}

func (s *ArchiveStorage) OpenObjectVersion(ctx context.Context, key, versionID string) (io.ReadCloser, error) {
	if !s.Enabled() {
		return nil, fmt.Errorf("archive storage is disabled")
	}
	if strings.TrimSpace(versionID) == "" {
		return nil, fmt.Errorf("archive object version id is required")
	}
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key), VersionId: aws.String(versionID)})
	if err != nil {
		return nil, fmt.Errorf("get archive object %s version %s: %w", key, versionID, err)
	}
	return output.Body, nil
}

func (s *ArchiveStorage) CopyObject(ctx context.Context, sourceKey, destinationKey, contentType string) error {
	if !s.Enabled() {
		return fmt.Errorf("archive storage is disabled")
	}

	input := &s3.CopyObjectInput{
		Bucket:     aws.String(s.bucket),
		Key:        aws.String(destinationKey),
		CopySource: aws.String(url.PathEscape(path.Join(s.bucket, sourceKey))),
	}
	if strings.TrimSpace(contentType) != "" {
		input.ContentType = aws.String(contentType)
		input.MetadataDirective = s3types.MetadataDirectiveReplace
	}
	if _, err := s.client.CopyObject(ctx, input); err != nil {
		return fmt.Errorf("copy archive object %s to %s: %w", sourceKey, destinationKey, err)
	}
	return nil
}

func (s *ArchiveStorage) OpenObject(ctx context.Context, key string) (io.ReadCloser, error) {
	if !s.Enabled() {
		return nil, fmt.Errorf("archive storage is disabled")
	}

	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	if err != nil {
		return nil, fmt.Errorf("get archive object %s: %w", key, err)
	}
	return output.Body, nil
}

func (s *ArchiveStorage) DeleteObject(ctx context.Context, key string) error {
	if !s.Enabled() {
		return nil
	}
	if _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)}); err != nil {
		return fmt.Errorf("delete archive object %s: %w", key, err)
	}
	return nil
}

func (s *ArchiveStorage) OriginalObjectKey(institutionID, documentID, fileName string) string {
	return path.Join("archive", sanitizeKeyPart(institutionID), documentID, "original", sanitizeKeyPart(fileName))
}

func (s *ArchiveStorage) ArtifactObjectKey(institutionID, documentID string, versionNo int) string {
	return path.Join("archive", sanitizeKeyPart(institutionID), documentID, "versions", fmt.Sprintf("%d", versionNo), "artifact.json")
}

func sanitizeKeyPart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	value = strings.ToLower(value)
	value = strings.ReplaceAll(value, "\\", "-")
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.ReplaceAll(value, "..", "-")
	return value
}
