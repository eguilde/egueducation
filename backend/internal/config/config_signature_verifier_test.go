package config

import (
	"strings"
	"testing"
)

func TestValidateSignatureVerifierRequiresCompleteHTTPSConfiguration(t *testing.T) {
	for _, test := range []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{name: "disabled outside production", cfg: Config{}},
		{name: "disabled in production", cfg: Config{Environment: "production"}, wantErr: "required in production"},
		{name: "missing token", cfg: Config{SignatureVerifierURL: "https://trust.example.test/validate"}, wantErr: "configured together"},
		{name: "missing URL", cfg: Config{SignatureVerifierToken: strings.Repeat("s", 32)}, wantErr: "configured together"},
		{name: "production HTTP", cfg: Config{Environment: "production", SignatureVerifierURL: "http://trust.example.test/validate", SignatureVerifierToken: strings.Repeat("s", 32)}, wantErr: "HTTPS"},
		{name: "test HTTP", cfg: Config{Environment: "test", SignatureVerifierURL: "http://127.0.0.1:8099/validate", SignatureVerifierToken: strings.Repeat("s", 32)}},
		{name: "production HTTPS", cfg: Config{Environment: "production", SignatureVerifierURL: "https://trust.example.test/validate", SignatureVerifierToken: strings.Repeat("s", 32)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := test.cfg.ValidateSignatureVerifier()
			if test.wantErr == "" && err != nil {
				t.Fatalf("ValidateSignatureVerifier() error = %v", err)
			}
			if test.wantErr != "" && (err == nil || !strings.Contains(err.Error(), test.wantErr)) {
				t.Fatalf("ValidateSignatureVerifier() error = %v; want substring %q", err, test.wantErr)
			}
		})
	}
}
