package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	maxStoredSMSMessageBytes  = 1024
	maxProviderResponseBytes  = 64 * 1024
	maxProviderIDBytes        = 128
	maxProviderErrorCodeBytes = 64
)

var (
	otpCodePattern           = regexp.MustCompile(`[0-9]{6}`)
	providerIDPattern        = regexp.MustCompile(`^[A-Za-z0-9._:-]+$`)
	providerErrorCodePattern = regexp.MustCompile(`^[A-Za-z0-9._:-]+$`)
)

type smsProviderError struct {
	code string
}

func newSMSProviderError(code string) *smsProviderError {
	return &smsProviderError{code: sanitizeProviderErrorCode(code)}
}

func (e *smsProviderError) Error() string {
	return "sms provider failure: " + e.code
}

type SMSService struct {
	apiToken   string
	senderName string
	client     *http.Client
	pool       *pgxpool.Pool
}

func NewSMSService(pool *pgxpool.Pool, apiToken, senderName string) *SMSService {
	return &SMSService{
		apiToken:   apiToken,
		senderName: senderName,
		pool:       pool,
		client:     &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *SMSService) Configured() bool {
	return s.apiToken != ""
}

func (s *SMSService) Send(ctx context.Context, to, message string) (string, error) {
	to = NormalizePhone(to)

	var queueID int64
	if s.pool != nil {
		if err := s.pool.QueryRow(ctx, `
			insert into sms_queue (to_phone, message, status, created_at, updated_at)
			values ($1, $2, 'pending', now(), now())
			returning id
		`, to, storedSMSMessage(message)).Scan(&queueID); err != nil {
			return "", fmt.Errorf("insert sms_queue: %w", err)
		}
	}

	if s.apiToken == "" {
		err := newSMSProviderError("provider_not_configured")
		s.markFailed(ctx, queueID, err.code)
		return "", err
	}

	providerID, err := s.sendToProvider(ctx, to, message)
	if err != nil {
		s.markFailed(ctx, queueID, providerFailureCode(err))
		return "", err
	}
	s.markSent(ctx, queueID, providerID)
	return providerID, nil
}

// sms_queue is operational metadata, not a credential store. The provider
// request still receives the original message; persisted queue content never
// contains a six digit OTP and is bounded to prevent provider/user-controlled
// payloads from becoming an unbounded log sink.
func redactOTP(message string) string {
	return otpCodePattern.ReplaceAllString(message, "[redacted-otp]")
}

func storedSMSMessage(message string) string {
	return truncateUTF8(redactOTP(message), maxStoredSMSMessageBytes)
}

func (s *SMSService) sendToProvider(ctx context.Context, to, message string) (string, error) {
	form := url.Values{}
	form.Set("to", to)
	form.Set("message", message)
	form.Set("format", "json")

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://api.smsapi.ro/sms.do",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return "", fmt.Errorf("smsapi request create: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+s.apiToken)

	resp, err := s.client.Do(req)
	if err != nil {
		// Transport errors are intentionally collapsed to a stable code. A
		// custom transport or intermediary can reflect the request body in its
		// error text, including the OTP.
		return "", newSMSProviderError("provider_transport_error")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxProviderResponseBytes+1))
	if err != nil {
		return "", newSMSProviderError("provider_response_read_failed")
	}
	if len(body) > maxProviderResponseBytes {
		return "", newSMSProviderError("provider_response_too_large")
	}

	result, parseErr := parseProviderResponse(body, message)
	if resp.StatusCode != http.StatusOK {
		if parseErr == nil && result.errorCode != "" {
			return "", newSMSProviderError("provider_error_" + result.errorCode)
		}
		return "", newSMSProviderError(fmt.Sprintf("provider_http_%d", resp.StatusCode))
	}
	if parseErr != nil {
		return "", parseErr
	}
	if result.errorCode != "" && result.errorCode != "0" {
		return "", newSMSProviderError("provider_error_" + result.errorCode)
	}
	if result.providerID == "" {
		return "", newSMSProviderError("provider_id_missing")
	}
	return result.providerID, nil
}

type parsedProviderResponse struct {
	providerID string
	errorCode  string
}

func parseProviderResponse(body []byte, outboundMessage string) (parsedProviderResponse, error) {
	var payload struct {
		Error json.RawMessage `json:"error"`
		List  []struct {
			ID string `json:"id"`
		} `json:"list"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return parsedProviderResponse{}, newSMSProviderError("provider_response_invalid")
	}

	secrets := sixDigitSecrets(outboundMessage)
	errorCode := parseProviderErrorCode(payload.Error)
	for _, secret := range secrets {
		if secret != "" && strings.Contains(errorCode, secret) {
			errorCode = "reflected_secret"
			break
		}
	}
	result := parsedProviderResponse{errorCode: errorCode}
	if len(payload.List) == 0 {
		return result, nil
	}
	providerID, err := sanitizeProviderID(payload.List[0].ID, secrets)
	if err != nil {
		return parsedProviderResponse{}, err
	}
	result.providerID = providerID
	return result, nil
}

func parseProviderErrorCode(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	var code string
	switch typed := value.(type) {
	case string:
		code = strings.TrimSpace(typed)
	case float64:
		if typed != float64(int64(typed)) {
			return ""
		}
		code = fmt.Sprintf("%d", int64(typed))
	default:
		return ""
	}
	if len(code) == 0 || len(code) > maxProviderErrorCodeBytes || !providerErrorCodePattern.MatchString(code) {
		return ""
	}
	return code
}

func sanitizeProviderID(raw string, secrets []string) (string, error) {
	providerID := strings.TrimSpace(raw)
	if len(providerID) == 0 || len(providerID) > maxProviderIDBytes || !providerIDPattern.MatchString(providerID) {
		return "", newSMSProviderError("provider_id_invalid")
	}
	for _, secret := range secrets {
		if secret != "" && strings.Contains(providerID, secret) {
			return "", newSMSProviderError("provider_id_reflected_secret")
		}
	}
	return providerID, nil
}

func sixDigitSecrets(message string) []string {
	matches := otpCodePattern.FindAllString(message, -1)
	if len(matches) == 0 {
		return nil
	}
	unique := make(map[string]struct{}, len(matches))
	secrets := make([]string, 0, len(matches))
	for _, match := range matches {
		if _, found := unique[match]; found {
			continue
		}
		unique[match] = struct{}{}
		secrets = append(secrets, match)
	}
	return secrets
}

func providerFailureCode(err error) string {
	var providerErr *smsProviderError
	if errors.As(err, &providerErr) {
		return sanitizeProviderErrorCode(providerErr.code)
	}
	return "provider_failure"
}

func sanitizeProviderErrorCode(code string) string {
	code = strings.TrimSpace(code)
	if len(code) == 0 || len(code) > maxProviderErrorCodeBytes || !providerErrorCodePattern.MatchString(code) {
		return "provider_failure"
	}
	return code
}

func truncateUTF8(value string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(value) <= maxBytes {
		return value
	}
	value = value[:maxBytes]
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value
}

func NormalizePhone(phone string) string {
	var digits strings.Builder
	for _, char := range phone {
		if char >= '0' && char <= '9' {
			digits.WriteRune(char)
		}
	}
	value := digits.String()

	switch {
	case len(value) == 10 && strings.HasPrefix(value, "0"):
		return "+40" + value[1:]
	case len(value) == 11 && strings.HasPrefix(value, "40"):
		return "+" + value
	case strings.HasPrefix(phone, "+"):
		return phone
	default:
		return "+" + value
	}
}

func (s *SMSService) markFailed(ctx context.Context, queueID int64, errorCode string) {
	if s.pool == nil || queueID == 0 {
		return
	}
	_, _ = s.pool.Exec(ctx, `
		update sms_queue
		set status = 'failed', error = $1, updated_at = now()
		where id = $2
	`, sanitizeProviderErrorCode(errorCode), queueID)
}

func (s *SMSService) markSent(ctx context.Context, queueID int64, providerID string) {
	if s.pool == nil || queueID == 0 {
		return
	}
	providerID, err := sanitizeProviderID(providerID, nil)
	if err != nil {
		s.markFailed(ctx, queueID, providerFailureCode(err))
		return
	}
	_, _ = s.pool.Exec(ctx, `
		update sms_queue
		set status = 'sent', provider_id = $1, sent_at = now(), updated_at = now()
		where id = $2
	`, providerID, queueID)
}
