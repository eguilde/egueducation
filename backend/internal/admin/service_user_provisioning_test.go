package admin

import (
	"errors"
	"testing"
)

func boolPointer(value bool) *bool { return &value }

func TestResolveAdminUserAuthenticationRequiresVerifiedLoginIdentity(t *testing.T) {
	tests := []struct {
		name        string
		request     UpsertUserRequest
		storedEmail bool
		storedPhone bool
		wantChannel string
		wantErr     error
	}{
		{
			name: "phone-only verified user defaults to sms",
			request: UpsertUserRequest{
				Phone: "0712345678",
			},
			storedPhone: true,
			wantChannel: "sms",
		},
		{
			name: "verified email-only user defaults to email",
			request: UpsertUserRequest{
				Email: "operator@example.test",
			},
			storedEmail: true,
			wantChannel: "email",
		},
		{
			name: "unverified identifiers are rejected",
			request: UpsertUserRequest{
				Email: "operator@example.test",
				Phone: "0712345678",
			},
			wantErr: errVerifiedLoginIdentifierMissing,
		},
		{
			name: "email channel cannot use unverified email",
			request: UpsertUserRequest{
				Email:               "operator@example.test",
				PreferredOTPChannel: "email",
			},
			wantErr: errVerifiedLoginIdentifierMissing,
		},
		{
			name: "omitted update flags preserve verified phone",
			request: UpsertUserRequest{
				Phone: "0712345678",
			},
			storedPhone: true,
			wantChannel: "sms",
		},
		{
			name:    "explicit verification revocation is enforced",
			request: UpsertUserRequest{Phone: "0712345678"},
			wantErr: errVerifiedLoginIdentifierMissing,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			channel, err := resolveAdminUserAuthentication(test.request, test.storedEmail, test.storedPhone)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("error = %v, want %v", err, test.wantErr)
			}
			if channel != test.wantChannel {
				t.Fatalf("channel = %q, want %q", channel, test.wantChannel)
			}
		})
	}
}

func TestNormalizeAdminPhoneUsesCanonicalE164Form(t *testing.T) {
	if got := normalizeAdminPhone("0712 345 678"); got != "+40712345678" {
		t.Fatalf("normalized phone = %q, want +40712345678", got)
	}
	if got := normalizeAdminPhone("  "); got != "" {
		t.Fatalf("empty phone normalized to %q", got)
	}
}

func TestAdminIdentifierNormalizationRejectsMalformedValues(t *testing.T) {
	for _, value := range []string{"+0123456789", "+401234", "+40123456789012345", "not-a-phone"} {
		if normalized := normalizeAdminPhone(value); normalized != "" && e164PhonePattern.MatchString(normalized) {
			t.Fatalf("malformed phone %q normalized to valid-looking %q", value, normalized)
		}
	}
	for _, value := range []string{"display <operator@example.test>", "operator@localhost", "operator@example"} {
		if _, err := normalizeAdminEmail(value); err == nil {
			t.Fatalf("malformed email %q was accepted", value)
		}
	}
	if got, err := normalizeAdminEmail(" Operator@Example.Test "); err != nil || got != "operator@example.test" {
		t.Fatalf("normalized email = (%q, %v), want operator@example.test with no error", got, err)
	}
}

func TestResolveRequestedVerificationDoesNotTransferTrustToChangedIdentifier(t *testing.T) {
	if resolveRequestedVerification(true, true, nil) {
		t.Fatal("a changed identifier must not inherit verified status when verification is omitted")
	}
	if !resolveRequestedVerification(true, false, nil) {
		t.Fatal("an unchanged identifier must preserve verified status when verification is omitted")
	}
	if !resolveRequestedVerification(false, true, boolPointer(true)) {
		t.Fatal("an explicit verification result must be honored")
	}
}
