package earchiva

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/eguilde/egueducation/internal/config"
)

const (
	azureDocumentIntelligencePollInterval = 2 * time.Second
	azureDocumentIntelligenceTimeout      = 15 * time.Minute
	azureDocumentIntelligenceMaxResult    = 64 << 20
)

// AzureDocumentIntelligenceOCR submits scan PDFs directly to the Azure REST
// API. It does not make the archive object public and never logs the key,
// operation URL, file name, or extracted content.
type AzureDocumentIntelligenceOCR struct {
	endpoint   string
	origin     *url.URL
	key        string
	model      string
	apiVersion string
	httpClient *http.Client
	pollEvery  time.Duration
	timeout    time.Duration
}

func NewAzureDocumentIntelligenceOCR(cfg config.Config) (*AzureDocumentIntelligenceOCR, error) {
	return newAzureDocumentIntelligenceOCR(cfg, http.DefaultClient)
}

func newAzureDocumentIntelligenceOCR(cfg config.Config, client *http.Client) (*AzureDocumentIntelligenceOCR, error) {
	if err := cfg.ValidateArchiveOCR(); err != nil {
		return nil, err
	}
	if !cfg.AzureDocumentIntelligenceEnabled() {
		return &AzureDocumentIntelligenceOCR{}, nil
	}
	endpoint := strings.TrimRight(strings.TrimSpace(cfg.AzureDocumentIntelligenceEndpoint), "/")
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return nil, fmt.Errorf("invalid Azure Document Intelligence endpoint")
	}
	if cfg.IsProduction() && !strings.EqualFold(parsed.Scheme, "https") {
		return nil, fmt.Errorf("Azure Document Intelligence endpoint must use HTTPS in production")
	}
	if client == nil {
		client = http.DefaultClient
	}
	clientCopy := *client
	clientCopy.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &AzureDocumentIntelligenceOCR{
		endpoint: endpoint, origin: parsed, key: strings.TrimSpace(cfg.AzureDocumentIntelligenceKey),
		model: strings.TrimSpace(cfg.AzureDocumentIntelligenceModel), apiVersion: strings.TrimSpace(cfg.AzureDocumentIntelligenceAPIVersion),
		httpClient: &clientCopy, pollEvery: azureDocumentIntelligencePollInterval, timeout: azureOCRTimeout(cfg.AzureDocumentIntelligenceTimeoutMinutes),
	}, nil
}

func azureOCRTimeout(minutes int) time.Duration {
	if minutes < 1 || minutes > 180 {
		return azureDocumentIntelligenceTimeout
	}
	return time.Duration(minutes) * time.Minute
}

func (a *AzureDocumentIntelligenceOCR) Enabled() bool {
	return a != nil && a.endpoint != "" && a.key != "" && a.model != "" && a.apiVersion != "" && a.httpClient != nil
}

func (a *AzureDocumentIntelligenceOCR) AnalyzeDocument(ctx context.Context, _ string, _ string, _ int, localPath, _ string, mimeType string) (string, int, map[string]any, error) {
	if !a.Enabled() {
		return "", 0, nil, fmt.Errorf("Azure Document Intelligence OCR is disabled")
	}
	if !archiveTextractSupported(mimeType, localPath) {
		return "", 0, nil, fmt.Errorf("Azure Document Intelligence does not support this archive file type")
	}

	file, err := os.Open(localPath)
	if err != nil {
		return "", 0, nil, fmt.Errorf("open Azure OCR source: %w", err)
	}
	defer file.Close() //nolint:errcheck

	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()
	operationURL, err := a.submit(ctx, file, mimeType)
	if err != nil {
		return "", 0, nil, err
	}
	result, err := a.poll(ctx, operationURL)
	if err != nil {
		return "", 0, nil, err
	}
	metadata := map[string]any{
		"ocr_source":      "azure-document-intelligence",
		"ocr_model":       a.model,
		"ocr_api_version": a.apiVersion,
		"ocr_page_count":  len(result.AnalyzeResult.Pages),
	}
	return strings.TrimSpace(result.AnalyzeResult.Content), len(result.AnalyzeResult.Pages), metadata, nil
}

func (a *AzureDocumentIntelligenceOCR) submit(ctx context.Context, body io.Reader, mimeType string) (string, error) {
	u := a.endpoint + "/documentintelligence/documentModels/" + url.PathEscape(a.model) + ":analyze"
	query := url.Values{}
	query.Set("api-version", a.apiVersion)
	u += "?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, body)
	if err != nil {
		return "", fmt.Errorf("create Azure OCR request: %w", err)
	}
	if strings.TrimSpace(mimeType) == "" {
		mimeType = "application/pdf"
	}
	req.Header.Set("Content-Type", mimeType)
	req.Header.Set("Ocp-Apim-Subscription-Key", a.key)
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("submit Azure OCR: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != http.StatusAccepted {
		return "", azureOCRHTTPError("submit", resp)
	}
	operationURL := strings.TrimSpace(resp.Header.Get("Operation-Location"))
	if operationURL == "" {
		return "", fmt.Errorf("Azure OCR response did not include an operation location")
	}
	parsed, err := url.Parse(operationURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("Azure OCR returned an invalid operation location")
	}
	if a.origin == nil || !strings.EqualFold(parsed.Scheme, a.origin.Scheme) || !strings.EqualFold(parsed.Host, a.origin.Host) {
		return "", fmt.Errorf("Azure OCR returned an operation location outside the configured endpoint")
	}
	return operationURL, nil
}

type azureAnalyzeResponse struct {
	Status string `json:"status"`
	Error  *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
	AnalyzeResult struct {
		Content string `json:"content"`
		// Only the number of pages is needed. Empty structs make the decoder
		// validate and count page objects without retaining Azure's verbose
		// polygon/word layout payload in memory.
		Pages []struct{} `json:"pages"`
	} `json:"analyzeResult"`
}

func (a *AzureDocumentIntelligenceOCR) poll(ctx context.Context, operationURL string) (azureAnalyzeResponse, error) {
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, operationURL, nil)
		if err != nil {
			return azureAnalyzeResponse{}, fmt.Errorf("create Azure OCR polling request: %w", err)
		}
		req.Header.Set("Ocp-Apim-Subscription-Key", a.key)
		resp, err := a.httpClient.Do(req)
		if err != nil {
			return azureAnalyzeResponse{}, fmt.Errorf("poll Azure OCR: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
			resp.Body.Close() //nolint:errcheck
			return azureAnalyzeResponse{}, azureOCRStatusError("poll", resp.StatusCode, body)
		}
		var result azureAnalyzeResponse
		decoder := json.NewDecoder(io.LimitReader(resp.Body, azureDocumentIntelligenceMaxResult))
		if err := decoder.Decode(&result); err != nil {
			resp.Body.Close() //nolint:errcheck
			return azureAnalyzeResponse{}, fmt.Errorf("decode Azure OCR response: %w", err)
		}
		resp.Body.Close() //nolint:errcheck
		switch strings.ToLower(strings.TrimSpace(result.Status)) {
		case "succeeded":
			return result, nil
		case "failed":
			if result.Error != nil && strings.TrimSpace(result.Error.Code) != "" {
				return azureAnalyzeResponse{}, fmt.Errorf("Azure OCR analysis failed (%s)", result.Error.Code)
			}
			return azureAnalyzeResponse{}, fmt.Errorf("Azure OCR analysis failed")
		case "notstarted", "running":
			select {
			case <-ctx.Done():
				return azureAnalyzeResponse{}, fmt.Errorf("Azure OCR analysis timed out: %w", ctx.Err())
			case <-time.After(a.pollEvery):
			}
		default:
			return azureAnalyzeResponse{}, fmt.Errorf("Azure OCR returned unexpected analysis status")
		}
	}
}

func azureOCRHTTPError(action string, resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	return azureOCRStatusError(action, resp.StatusCode, body)
}

func azureOCRStatusError(action string, status int, body []byte) error {
	// Azure error payloads can echo request details. Keep errors suitable for
	// persistence/logging without including response text or PII.
	_ = bytes.TrimSpace(body)
	return fmt.Errorf("Azure OCR %s returned HTTP %d", action, status)
}
