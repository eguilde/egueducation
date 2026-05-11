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
	Identifier string `json:"identifier"`
}

type VerifySMSOTPRequest struct {
	Identifier string `json:"identifier"`
	Code       string `json:"code"`
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
