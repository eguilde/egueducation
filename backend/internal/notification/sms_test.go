package notification

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	appdb "github.com/eguilde/egueducation/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestStoredSMSMessageRedactsEverySixDigitSequenceAndBoundsUTF8(t *testing.T) {
	raw := "OTP=482615; embedded=A173829B; " + strings.Repeat("ș", maxStoredSMSMessageBytes)
	stored := storedSMSMessage(raw)
	if strings.Contains(stored, "482615") || strings.Contains(stored, "173829") {
		t.Fatalf("stored SMS message contains a six-digit secret: %q", stored)
	}
	if !strings.Contains(stored, "[redacted-otp]") {
		t.Fatalf("stored SMS message does not contain the redaction marker: %q", stored)
	}
	if len(stored) > maxStoredSMSMessageBytes {
		t.Fatalf("stored SMS message length=%d, want <=%d", len(stored), maxStoredSMSMessageBytes)
	}
	if !utf8.ValidString(stored) {
		t.Fatal("stored SMS truncation produced invalid UTF-8")
	}
}

func TestSMSProviderResponsesNeverReflectBodiesOrOTP(t *testing.T) {
	const otp = "482615"
	tests := []struct {
		name         string
		status       int
		body         string
		transportErr error
		wantCode     string
	}{
		{
			name:     "structured provider error ignores message",
			status:   http.StatusBadRequest,
			body:     `{"error":101,"message":"reflected OTP 482615 and entire request"}`,
			wantCode: "provider_error_101",
		},
		{
			name:     "provider error code reflects OTP",
			status:   http.StatusBadRequest,
			body:     `{"error":"482615","message":"ignored"}`,
			wantCode: "provider_error_reflected_secret",
		},
		{
			name:     "unstructured provider body",
			status:   http.StatusBadGateway,
			body:     "upstream reflected OTP 482615 and the complete raw body",
			wantCode: "provider_http_502",
		},
		{
			name:         "transport error",
			transportErr: errors.New("transport reflected raw form containing 482615"),
			wantCode:     "provider_transport_error",
		},
		{
			name:     "invalid success body",
			status:   http.StatusOK,
			body:     "successful raw reflection 482615",
			wantCode: "provider_response_invalid",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := testSMSService(func(*http.Request) (*http.Response, error) {
				if test.transportErr != nil {
					return nil, test.transportErr
				}
				return smsHTTPResponse(test.status, test.body), nil
			})
			_, err := service.sendToProvider(context.Background(), "+40700000000", "Cod OTP: "+otp)
			if err == nil {
				t.Fatal("provider response unexpectedly succeeded")
			}
			if got := providerFailureCode(err); got != test.wantCode {
				t.Fatalf("provider failure code=%q, want %q (error=%v)", got, test.wantCode, err)
			}
			if strings.Contains(err.Error(), otp) || strings.Contains(err.Error(), "raw") || strings.Contains(err.Error(), "entire request") {
				t.Fatalf("safe error reflected provider/request content: %v", err)
			}
		})
	}
}

func TestSMSProviderAcceptsOnlyBoundedParsedID(t *testing.T) {
	t.Run("valid parsed ID", func(t *testing.T) {
		service := testSMSService(func(*http.Request) (*http.Response, error) {
			return smsHTTPResponse(http.StatusOK, `{"count":1,"list":[{"id":"provider-abc_789"}],"message":"OTP 482615"}`), nil
		})
		providerID, err := service.sendToProvider(context.Background(), "+40700000000", "Cod OTP: 482615")
		if err != nil {
			t.Fatalf("send to provider: %v", err)
		}
		if providerID != "provider-abc_789" {
			t.Fatalf("provider ID=%q, want parsed ID", providerID)
		}
	})

	for _, test := range []struct {
		name string
		id   string
		code string
	}{
		{name: "OTP reflection", id: "echo-482615", code: "provider_id_reflected_secret"},
		{name: "unbounded ID", id: strings.Repeat("a", maxProviderIDBytes+1), code: "provider_id_invalid"},
		{name: "unsafe ID", id: "provider id with spaces", code: "provider_id_invalid"},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := testSMSService(func(*http.Request) (*http.Response, error) {
				return smsHTTPResponse(http.StatusOK, fmt.Sprintf(`{"list":[{"id":%q}]}`, test.id)), nil
			})
			_, err := service.sendToProvider(context.Background(), "+40700000000", "Cod OTP: 482615")
			if err == nil {
				t.Fatal("unsafe provider ID unexpectedly succeeded")
			}
			if got := providerFailureCode(err); got != test.code {
				t.Fatalf("provider failure code=%q, want %q", got, test.code)
			}
		})
	}
}

func TestSMSQueuePostgresPersistsOnlyRedactedBoundedMetadata(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("EGUEDUCATION_TEST_DATABASE_URL"))
	if databaseURL == "" {
		databaseURL = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	}
	if databaseURL == "" {
		t.Skip("SMS queue PostgreSQL test requires EGUEDUCATION_TEST_DATABASE_URL or TEST_DATABASE_URL")
	}

	ctx := context.Background()
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse integration database URL: %v", err)
	}
	config.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("connect integration database: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := appdb.Migrate(ctx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	phone := fmt.Sprintf("+409%08d", time.Now().UnixNano()%100000000)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `delete from sms_queue where to_phone=$1`, phone)
	})
	service := NewSMSService(pool, "configured-for-test", "test")
	service.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return smsHTTPResponse(http.StatusBadRequest, `{"error":101,"message":"provider reflected OTP 482615 and raw request"}`), nil
	})}
	message := "Cod OTP: 482615. " + strings.Repeat("conținut lung ", maxStoredSMSMessageBytes)
	if _, err := service.Send(ctx, phone, message); err == nil {
		t.Fatal("provider rejection unexpectedly succeeded")
	}

	var storedMessage, storedError, status string
	if err := pool.QueryRow(ctx, `select message, error, status from sms_queue where to_phone=$1 order by id desc limit 1`, phone).Scan(&storedMessage, &storedError, &status); err != nil {
		t.Fatalf("read queued SMS metadata: %v", err)
	}
	if status != "failed" || storedError != "provider_error_101" {
		t.Fatalf("queue status/error=(%q,%q), want (failed,provider_error_101)", status, storedError)
	}
	if strings.Contains(storedMessage, "482615") || strings.Contains(storedError, "482615") || strings.Contains(storedError, "raw request") {
		t.Fatalf("SMS queue reflected secret/provider body: message=%q error=%q", storedMessage, storedError)
	}
	if len(storedMessage) > maxStoredSMSMessageBytes || len(storedError) > maxProviderErrorCodeBytes {
		t.Fatalf("SMS queue metadata is unbounded: message=%d error=%d", len(storedMessage), len(storedError))
	}
}

func testSMSService(transport roundTripFunc) *SMSService {
	service := NewSMSService(nil, "configured-for-test", "test")
	service.client = &http.Client{Transport: transport}
	return service
}

func smsHTTPResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}
