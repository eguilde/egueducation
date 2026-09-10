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
	Bucket    string
	Key       string
	VersionID string
	ETag      string
}

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

// PutImmutableObject stores one exact S3 version under COMPLIANCE retention.
// Callers must derive retainUntil from the applicable archive-series policy;
// this boundary deliberately refuses missing or elapsed deadlines instead of
// inventing a bucket-wide retention period.
func (s *ArchiveStorage) PutImmutableObject(ctx context.Context, key, contentType string, body io.Reader, contentLength int64, retainUntil time.Time, legalHold bool) (ImmutableArchiveObject, error) {
	if !s.Enabled() {
		return ImmutableArchiveObject{}, fmt.Errorf("archive storage is disabled")
	}
	if !s.requireObjectLock {
		return ImmutableArchiveObject{}, fmt.Errorf("archive immutable write requires verified Object Lock storage")
	}
	retainUntil = retainUntil.UTC()
	if !retainUntil.After(time.Now().UTC()) {
		return ImmutableArchiveObject{}, fmt.Errorf("archive immutable write requires a future retention deadline")
	}
	input := &s3.PutObjectInput{
		Bucket:                    aws.String(s.bucket),
		Key:                       aws.String(key),
		Body:                      body,
		ObjectLockMode:            s3types.ObjectLockModeCompliance,
		ObjectLockRetainUntilDate: aws.Time(retainUntil),
	}
	if legalHold {
		input.ObjectLockLegalHoldStatus = s3types.ObjectLockLegalHoldStatusOn
	}
	if strings.TrimSpace(contentType) != "" {
		input.ContentType = aws.String(contentType)
	}
	if contentLength >= 0 {
		input.ContentLength = aws.Int64(contentLength)
	}
	output, err := s.client.PutObject(ctx, input)
	if err != nil {
		return ImmutableArchiveObject{}, fmt.Errorf("put immutable archive object %s: %w", key, err)
	}
	versionID := strings.TrimSpace(aws.ToString(output.VersionId))
	if versionID == "" {
		return ImmutableArchiveObject{}, fmt.Errorf("immutable archive object %s has no storage version id", key)
	}
	return ImmutableArchiveObject{Bucket: s.bucket, Key: key, VersionID: versionID, ETag: strings.Trim(aws.ToString(output.ETag), `"`)}, nil
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
