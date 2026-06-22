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

type RoleCatalogItem struct {
	Code        string `json:"code"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Permissions []string `json:"permissions,omitempty"`
	Positions   []string `json:"positions,omitempty"`
}

type RoleCatalogResponse struct {
	Roles []RoleCatalogItem `json:"roles"`
}

type RolePositionItem struct {
	PositionCode string `json:"position_code"`
	PositionName string `json:"position_name"`
	RoleCode     string `json:"role_code"`
	RoleLabel    string `json:"role_label"`
}

type RolePositionResponse struct {
	Items []RolePositionItem `json:"items"`
}

type ConsentScope struct {
	Code        string `json:"code"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
}

type OIDCClientRegistrationRequest struct {
	ClientID     string   `json:"client_id"`
	ClientName   string   `json:"client_name"`
	PublicClient bool     `json:"public_client"`
	RequirePKCE  bool     `json:"require_pkce"`
	Active       bool     `json:"active"`
	RedirectURIs []string `json:"redirect_uris"`
}

type OIDCClientRegistrationResponse struct {
	ClientID     string   `json:"client_id"`
	ClientName   string   `json:"client_name"`
	PublicClient bool     `json:"public_client"`
	RequirePKCE  bool     `json:"require_pkce"`
	Active       bool     `json:"active"`
	RedirectURIs []string `json:"redirect_uris"`
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
