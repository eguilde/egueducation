package earchiva

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"

	aws "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/eguilde/egueducation/internal/config"
)

type ArchiveStorage struct {
	client  *s3.Client
	bucket  string
	region  string
	enabled bool
}

func NewArchiveStorage(ctx context.Context, cfg config.Config) (*ArchiveStorage, error) {
	endpoint := strings.TrimSpace(cfg.ArchiveStorageEndpoint)
	bucket := strings.TrimSpace(cfg.ArchiveStorageBucket)
	if endpoint == "" || bucket == "" {
		return &ArchiveStorage{bucket: bucket, enabled: false}, nil
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

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = cfg.ArchiveStorageUsePathStyle
	})

	storage := &ArchiveStorage{client: client, bucket: bucket, region: strings.TrimSpace(cfg.ArchiveStorageRegion), enabled: true}
	if err := storage.EnsureBucket(ctx); err != nil {
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
	}

	input := &s3.CreateBucketInput{Bucket: aws.String(s.bucket)}
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
