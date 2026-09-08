package config

import (
	"strings"
	"testing"
)

func TestValidateOTPStorageRequiresStrongProductionKey(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{name: "missing", cfg: Config{Environment: "production"}, want: "required"},
		{name: "short", cfg: Config{Environment: "production", OTPHMACKey: "too-short"}, want: "32 bytes"},
		{name: "strong", cfg: Config{Environment: "production", OTPHMACKey: strings.Repeat("k", 32)}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.cfg.ValidateOTPStorage()
			if test.want == "" {
				if err != nil {
					t.Fatalf("ValidateOTPStorage() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ValidateOTPStorage() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestTestEnvironmentUsesOnlyDeterministicSyntheticOTPKey(t *testing.T) {
	cfg := Config{Environment: "test"}
	if err := cfg.ValidateOTPStorage(); err != nil {
		t.Fatalf("test OTP key rejected: %v", err)
	}
	if cfg.OTPHMACKeyValue() != testOTPHMACKey {
		t.Fatal("test environment did not use the deterministic synthetic OTP key")
	}
}
