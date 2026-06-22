package auth

type SessionUser struct {
	ID                  string   `json:"id"`
	Sub                 string   `json:"sub"`
	Name                string   `json:"name"`
	Email               string   `json:"email"`
	EmailVerified       bool     `json:"email_verified"`
	PhoneNumber         string   `json:"phone_number"`
	PhoneNumberVerified bool     `json:"phone_number_verified"`
	PreferredOTPChannel string   `json:"preferred_otp_channel"`
	Locale              string   `json:"locale"`
	Roles               []string `json:"roles"`
}

type SessionModule struct {
	Code   string `json:"code"`
	Active bool   `json:"active"`
}

type SessionContext struct {
	User             SessionUser     `json:"user"`
	InstitutionID    string          `json:"institution_id"`
	InstitutionName  string          `json:"institution_name"`
	Permissions      []string        `json:"permissions"`
	Modules          []SessionModule `json:"modules"`
	Authentication   []string        `json:"authentication"`
	GDPRCapabilities []string        `json:"gdpr_capabilities"`
}

type RequestSMSOTPRequest struct {
	PhoneNumber string `json:"phone_number"`
	Identifier  string `json:"identifier,omitempty"`
}

type VerifySMSOTPRequest struct {
	PhoneNumber string `json:"phone_number"`
	Identifier  string `json:"identifier,omitempty"`
	Code        string `json:"code"`
}

type SMSOTPRequestResponse struct {
	Status      string `json:"status"`
	Channel     string `json:"channel"`
	PhoneNumber string `json:"phone_number"`
	MaskedPhone string `json:"masked_phone,omitempty"`
}

type SMSOTPVerifyResponse struct {
	Status  string         `json:"status"`
	Channel string         `json:"channel"`
	Session SessionContext `json:"session"`
}

type ConsentScope struct {
	Code     string `json:"code"`
	Label    string `json:"label"`
	Required bool   `json:"required"`
}

type ConsentRequestResponse struct {
	RequestID  string         `json:"request_id"`
	ClientID   string         `json:"client_id"`
	ClientName string         `json:"client_name"`
	Scopes     []ConsentScope `json:"scopes"`
	ExpiresAt  string         `json:"expires_at"`
}

type ConsentDecisionRequest struct {
	RequestID     string   `json:"request_id"`
	Decision      string   `json:"decision"`
	GrantedScopes []string `json:"granted_scopes"`
}

type UpdateProfileRequest struct {
	Name        string `json:"name"`
	PhoneNumber string `json:"phone_number"`
	Locale      string `json:"locale"`
}

type PasskeyCredentialSummary struct {
	ID           string `json:"id"`
	CredentialID string `json:"credential_id"`
	DeviceName   string `json:"device_name"`
	CreatedAt    string `json:"created_at"`
	LastUsedAt   string `json:"last_used_at,omitempty"`
}

type PasskeyRegistrationOptions struct {
	Challenge string `json:"challenge"`
	RP        struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	} `json:"rp"`
	User struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
	} `json:"user"`
	PubKeyCredParams []map[string]any `json:"pubKeyCredParams"`
	Timeout          int              `json:"timeout"`
	Attestation      string           `json:"attestation"`
}

type FinishPasskeyRegistrationRequest struct {
	CredentialID string         `json:"credential_id"`
	DeviceName   string         `json:"device_name"`
	Challenge    string         `json:"challenge"`
	Response     map[string]any `json:"response"`
}

type PasskeyAuthenticationOptions struct {
	Challenge string `json:"challenge"`
	RP        struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	} `json:"rp"`
	AllowCredentials []map[string]any `json:"allowCredentials,omitempty"`
	Timeout          int              `json:"timeout"`
	UserVerification string           `json:"userVerification"`
}

type BeginPasskeyAuthenticationResponse struct {
	Status  string                       `json:"status"`
	Options PasskeyAuthenticationOptions `json:"options"`
}

type FinishPasskeyAuthenticationRequest struct {
	Challenge    string         `json:"challenge"`
	CredentialID string         `json:"credential_id"`
	Response     map[string]any `json:"response"`
}
