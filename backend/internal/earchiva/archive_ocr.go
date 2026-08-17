package earchiva

import (
	"context"
	"fmt"

	"github.com/eguilde/egueducation/internal/config"
)

// ArchiveOCR is the provider-neutral contract used by the ingestion worker.
// Implementations receive a temporary local copy only after the original has
// been persisted in tenant-namespaced archive storage.
type ArchiveOCR interface {
	Enabled() bool
	AnalyzeDocument(ctx context.Context, institutionID, documentID string, versionNo int, localPath, originalFileName, mimeType string) (text string, pageCount int, metadata map[string]any, err error)
}

// NewArchiveOCR selects Azure Document Intelligence when it is explicitly and
// completely configured. AWS Textract remains a compatible optional fallback
// only when Azure has not been configured at all.
func NewArchiveOCR(ctx context.Context, cfg config.Config) (ArchiveOCR, error) {
	if err := cfg.ValidateArchiveOCR(); err != nil {
		return nil, err
	}
	if cfg.AzureDocumentIntelligenceEnabled() {
		return NewAzureDocumentIntelligenceOCR(cfg)
	}
	textract, err := NewArchiveTextract(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("initialize AWS Textract OCR: %w", err)
	}
	return textract, nil
}
