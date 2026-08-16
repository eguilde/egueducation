package auth

import (
	"strings"
	"testing"

	"github.com/eguilde/egueducation/internal/config"
)

func TestOTPLoginUIMessageDoesNotExposeCodeWhenSMSIsConfigured(t *testing.T) {
	const code = "123456"
	message, ok := otpLoginUIMessage(&config.Config{
		Environment:    "development",
		FrontendOrigin: "https://school.example.test",
	}, code, true)
	if !ok {
		t.Fatal("configured SMS delivery should allow the login flow to continue")
	}
	if strings.Contains(message, code) {
		t.Fatal("configured SMS delivery must not expose the OTP in the browser")
	}
}

func TestOTPLoginUIMessageAllowsCodeOnlyForLoopbackDevelopment(t *testing.T) {
	const code = "123456"
	tests := []struct {
		name    string
		cfg     *config.Config
		allowed bool
		exposes bool
	}{
		{
			name: "localhost development",
			cfg: &config.Config{
				Environment:    "development",
				FrontendOrigin: "http://localhost:4200",
			},
			allowed: true,
			exposes: true,
		},
		{
			name: "loopback test",
			cfg: &config.Config{
				Environment:    "test",
				FrontendOrigin: "http://127.0.0.1:4200",
			},
			allowed: true,
			exposes: true,
		},
		{
			name: "public development host fails closed",
			cfg: &config.Config{
				Environment:    "development",
				FrontendOrigin: "https://school.example.test",
			},
		},
		{
			name: "production loopback fails closed",
			cfg: &config.Config{
				Environment:    "production",
				FrontendOrigin: "http://localhost:4200",
			},
		},
		{name: "missing configuration fails closed"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			message, ok := otpLoginUIMessage(test.cfg, code, false)
			if ok != test.allowed {
				t.Fatalf("allowed = %v, want %v", ok, test.allowed)
			}
			if got := strings.Contains(message, code); got != test.exposes {
				t.Fatalf("message exposes code = %v, want %v", got, test.exposes)
			}
		})
	}
}
