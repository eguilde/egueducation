package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

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
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
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
		`, to, message).Scan(&queueID); err != nil {
			return "", fmt.Errorf("insert sms_queue: %w", err)
		}
	}

	if s.apiToken == "" {
		s.markFailed(ctx, queueID, "SMSAPI_TOKEN not configured")
		return "", fmt.Errorf("SMSAPI_TOKEN not configured")
	}

	providerID, err := s.sendToProvider(ctx, to, message)
	if err != nil {
		s.markFailed(ctx, queueID, err.Error())
		return "", err
	}
	s.markSent(ctx, queueID, providerID)
	return providerID, nil
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
		return "", fmt.Errorf("smsapi request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("smsapi error (status %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result struct {
		Error   int    `json:"error"`
		Message string `json:"message"`
		List    []struct {
			ID string `json:"id"`
		} `json:"list"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return strings.TrimSpace(string(body)), nil
	}
	if result.Error != 0 {
		return "", fmt.Errorf("smsapi error %d: %s", result.Error, result.Message)
	}
	if len(result.List) > 0 && result.List[0].ID != "" {
		return result.List[0].ID, nil
	}
	return strings.TrimSpace(string(body)), nil
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

func (s *SMSService) markFailed(ctx context.Context, queueID int64, message string) {
	if s.pool == nil || queueID == 0 {
		return
	}
	_, _ = s.pool.Exec(ctx, `
		update sms_queue
		set status = 'failed', error = $1, updated_at = now()
		where id = $2
	`, message, queueID)
}

func (s *SMSService) markSent(ctx context.Context, queueID int64, providerID string) {
	if s.pool == nil || queueID == 0 {
		return
	}
	_, _ = s.pool.Exec(ctx, `
		update sms_queue
		set status = 'sent', provider_id = $1, sent_at = now(), updated_at = now()
		where id = $2
	`, providerID, queueID)
}
