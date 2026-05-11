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
)

type SMSService struct {
	apiToken   string
	senderName string
	client     *http.Client
}

func NewSMSService(apiToken, senderName string) *SMSService {
	return &SMSService{
		apiToken:   apiToken,
		senderName: senderName,
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

	if s.apiToken == "" {
		return "", fmt.Errorf("SMSAPI_TOKEN not configured")
	}

	form := url.Values{}
	form.Set("to", to)
	form.Set("message", message)
	form.Set("format", "json")
	if s.senderName != "" {
		form.Set("from", s.senderName)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://api.smsapi.ro/sms.do",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return "", err
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
		return "", fmt.Errorf("smsapi error (status %d): %s", resp.StatusCode, body)
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
