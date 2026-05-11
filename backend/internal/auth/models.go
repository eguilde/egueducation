package auth

type SessionUser struct {
	ID          string   `json:"id"`
	Sub         string   `json:"sub"`
	Name        string   `json:"name"`
	Email       string   `json:"email"`
	PhoneNumber string   `json:"phone_number"`
	Locale      string   `json:"locale"`
	Roles       []string `json:"roles"`
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
