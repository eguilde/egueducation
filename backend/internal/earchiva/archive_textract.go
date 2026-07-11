package earchiva

import (
	"context"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	aws "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/textract"
	textracttypes "github.com/aws/aws-sdk-go-v2/service/textract/types"

	"github.com/eguilde/egueducation/internal/config"
)

type ArchiveTextract struct {
	textract *textract.Client
	s3       *s3.Client
	bucket   string
	region   string
	enabled  bool
}

func NewArchiveTextract(ctx context.Context, cfg config.Config) (*ArchiveTextract, error) {
	bucket := strings.TrimSpace(cfg.ArchiveTextractBucket)
	if bucket == "" {
		return &ArchiveTextract{bucket: bucket, enabled: false}, nil
	}

	region := strings.TrimSpace(cfg.ArchiveTextractRegion)
	if region == "" {
		region = "us-east-1"
	}

	loadOptions := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(region),
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return nil, fmt.Errorf("load textract config: %w", err)
	}

	client := &ArchiveTextract{
		textract: textract.NewFromConfig(awsCfg),
		s3:       s3.NewFromConfig(awsCfg),
		bucket:   bucket,
		region:   region,
		enabled:  true,
	}
	if err := client.ensureBucket(ctx); err != nil {
		return nil, err
	}
	return client, nil
}

func (t *ArchiveTextract) Enabled() bool {
	return t != nil && t.enabled && t.textract != nil && t.s3 != nil && strings.TrimSpace(t.bucket) != ""
}

func (t *ArchiveTextract) ensureBucket(ctx context.Context) error {
	if !t.Enabled() {
		return nil
	}

	if _, err := t.s3.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(t.bucket)}); err == nil {
		return nil
	}

	input := &s3.CreateBucketInput{Bucket: aws.String(t.bucket)}
	if region := strings.TrimSpace(t.region); region != "" && region != "us-east-1" {
		input.CreateBucketConfiguration = &s3types.CreateBucketConfiguration{LocationConstraint: s3types.BucketLocationConstraint(region)}
	}
	if _, err := t.s3.CreateBucket(ctx, input); err != nil {
		message := err.Error()
		if strings.Contains(message, "BucketAlreadyOwnedByYou") || strings.Contains(message, "BucketAlreadyExists") {
			return nil
		}
		return fmt.Errorf("ensure textract bucket %s: %w", t.bucket, err)
	}
	return nil
}

func (t *ArchiveTextract) AnalyzeDocument(ctx context.Context, institutionID, documentID string, versionNo int, localPath, originalFileName, mimeType string) (string, int, map[string]any, error) {
	if !t.Enabled() {
		return "", 0, nil, fmt.Errorf("textract is disabled")
	}

	stageKey := t.stageKey(institutionID, documentID, versionNo, originalFileName)
	if err := t.putFile(ctx, stageKey, localPath, mimeType); err != nil {
		return "", 0, nil, err
	}
	defer func() {
		_ = t.deleteObject(context.Background(), stageKey)
	}()

	jobID, err := t.startAnalysis(ctx, stageKey)
	if err != nil {
		return "", 0, nil, err
	}

	lines, pageCount, blockCount, err := t.collectLines(ctx, jobID)
	if err != nil {
		return "", 0, nil, err
	}

	text := buildTextFromPages(lines)
	metadata := map[string]any{
		"textract_job_id":      jobID,
		"textract_page_count":  pageCount,
		"textract_block_count": blockCount,
		"textract_stage_key":   stageKey,
		"textract_source":      "aws-textract",
	}
	return text, pageCount, metadata, nil
}

func (t *ArchiveTextract) stageKey(institutionID, documentID string, versionNo int, originalFileName string) string {
	base := sanitizeKeyPart(originalFileName)
	if base == "unknown" {
		base = "document"
	}
	if !strings.HasSuffix(strings.ToLower(base), ".pdf") && !strings.HasSuffix(strings.ToLower(base), ".png") && !strings.HasSuffix(strings.ToLower(base), ".jpg") && !strings.HasSuffix(strings.ToLower(base), ".jpeg") && !strings.HasSuffix(strings.ToLower(base), ".tiff") {
		base += ".pdf"
	}
	return path.Join("archive-textract", sanitizeKeyPart(institutionID), documentID, fmt.Sprintf("v%d", versionNo), base)
}

func (t *ArchiveTextract) putFile(ctx context.Context, key, localPath, mimeType string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open textract upload file: %w", err)
	}
	defer file.Close() //nolint:errcheck

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat textract upload file: %w", err)
	}

	input := &s3.PutObjectInput{
		Bucket:        aws.String(t.bucket),
		Key:           aws.String(key),
		Body:          file,
		ContentLength: aws.Int64(info.Size()),
	}
	if strings.TrimSpace(mimeType) != "" {
		input.ContentType = aws.String(mimeType)
	}

	if _, err := t.s3.PutObject(ctx, input); err != nil {
		return fmt.Errorf("stage textract object %s: %w", key, err)
	}
	return nil
}

func (t *ArchiveTextract) deleteObject(ctx context.Context, key string) error {
	if !t.Enabled() {
		return nil
	}
	if _, err := t.s3.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(t.bucket), Key: aws.String(key)}); err != nil {
		return fmt.Errorf("delete textract object %s: %w", key, err)
	}
	return nil
}

func (t *ArchiveTextract) startAnalysis(ctx context.Context, key string) (string, error) {
	out, err := t.textract.StartDocumentAnalysis(ctx, &textract.StartDocumentAnalysisInput{
		DocumentLocation: &textracttypes.DocumentLocation{S3Object: &textracttypes.S3Object{Bucket: aws.String(t.bucket), Name: aws.String(key)}},
		FeatureTypes:     []textracttypes.FeatureType{textracttypes.FeatureTypeForms, textracttypes.FeatureTypeTables},
	})
	if err != nil {
		return "", fmt.Errorf("start textract analysis: %w", err)
	}
	jobID := strings.TrimSpace(aws.ToString(out.JobId))
	if jobID == "" {
		return "", fmt.Errorf("start textract analysis returned empty job id")
	}
	return jobID, nil
}

func (t *ArchiveTextract) collectLines(ctx context.Context, jobID string) (map[int][]string, int, int, error) {
	linesByPage := map[int][]string{}
	pageCount := 0
	blockCount := 0
	nextToken := ""
	deadline := time.Now().Add(20 * time.Minute)

	for {
		if time.Now().After(deadline) {
			return nil, 0, 0, fmt.Errorf("textract analysis timed out")
		}

		input := &textract.GetDocumentAnalysisInput{JobId: aws.String(jobID)}
		if nextToken != "" {
			input.NextToken = aws.String(nextToken)
		}

		out, err := t.textract.GetDocumentAnalysis(ctx, input)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("get textract analysis: %w", err)
		}

		status := out.JobStatus
		switch status {
		case textracttypes.JobStatusInProgress:
			time.Sleep(3 * time.Second)
			continue
		case textracttypes.JobStatusFailed:
			return nil, 0, 0, fmt.Errorf("textract analysis failed")
		case textracttypes.JobStatusSucceeded:
			// Continue below.
		default:
			return nil, 0, 0, fmt.Errorf("unexpected textract job status %s", status)
		}

		for _, block := range out.Blocks {
			blockCount++
			if block.Page != nil && int(*block.Page) > pageCount {
				pageCount = int(*block.Page)
			}
			if block.BlockType != textracttypes.BlockTypeLine || strings.TrimSpace(aws.ToString(block.Text)) == "" {
				continue
			}
			page := 0
			if block.Page != nil {
				page = int(*block.Page)
			}
			linesByPage[page] = append(linesByPage[page], strings.TrimSpace(aws.ToString(block.Text)))
		}

		nextToken = strings.TrimSpace(aws.ToString(out.NextToken))
		if nextToken == "" {
			return linesByPage, pageCount, blockCount, nil
		}
	}
}

func buildTextFromPages(linesByPage map[int][]string) string {
	if len(linesByPage) == 0 {
		return ""
	}
	pages := make([]int, 0, len(linesByPage))
	for page := range linesByPage {
		pages = append(pages, page)
	}
	sort.Ints(pages)

	parts := make([]string, 0, len(pages))
	for _, page := range pages {
		lines := linesByPage[page]
		if len(lines) == 0 {
			continue
		}
		parts = append(parts, strings.Join(lines, "\n"))
	}
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}
