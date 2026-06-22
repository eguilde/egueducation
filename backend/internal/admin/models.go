package admin

type DashboardResponse struct {
	Stats         DashboardStats `json:"stats"`
	Modules       []ModuleStatus `json:"modules"`
	AdminSections []string       `json:"admin_sections"`
	Warnings      []string       `json:"warnings"`
}

type DashboardStats struct {
	Users           int `json:"users"`
	Memberships     int `json:"memberships"`
	Positions       int `json:"positions"`
	Permissions     int `json:"permissions"`
	Workflows       int `json:"workflows"`
	Archives        int `json:"archives"`
	ReadyDossiers   int `json:"ready_dossiers"`
	BlockedDossiers int `json:"blocked_dossiers"`
}

type ModuleStatus struct {
	Code   string `json:"code"`
	Active bool   `json:"active"`
}

type AdminUser struct {
	ID                  string `json:"id"`
	Sub                 string `json:"sub"`
	Name                string `json:"name"`
	Email               string `json:"email"`
	Phone               string `json:"phone"`
	Position            string `json:"position"`
	Locale              string `json:"locale"`
	Status              string `json:"status"`
	EmailVerified       bool   `json:"email_verified"`
	PhoneVerified       bool   `json:"phone_verified"`
	PreferredOTPChannel string `json:"preferred_otp_channel"`
	LastLoginAt         string `json:"last_login_at"`
}

type UpsertUserRequest struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Email               string `json:"email"`
	Phone               string `json:"phone"`
	Locale              string `json:"locale"`
	Status              string `json:"status"`
	EmailVerified       bool   `json:"email_verified"`
	PhoneVerified       bool   `json:"phone_verified"`
	PreferredOTPChannel string `json:"preferred_otp_channel"`
}

type Membership struct {
	ID               string `json:"id"`
	UserID           string `json:"user_id"`
	UserName         string `json:"user_name"`
	UserEmail        string `json:"user_email"`
	PositionCode     string `json:"position_code"`
	PositionName     string `json:"position_name"`
	OrgUnitCode      string `json:"org_unit_code"`
	OrganizationName string `json:"organization_name"`
	IsPrimary        bool   `json:"is_primary"`
	Active           bool   `json:"active"`
	StartDate        string `json:"start_date"`
	EndDate          string `json:"end_date"`
}

type UpsertMembershipRequest struct {
	ID               string `json:"id"`
	UserID           string `json:"user_id"`
	PositionCode     string `json:"position_code"`
	OrgUnitCode      string `json:"org_unit_code"`
	OrganizationName string `json:"organization_name"`
	IsPrimary        bool   `json:"is_primary"`
	Active           bool   `json:"active"`
	StartDate        string `json:"start_date"`
	EndDate          string `json:"end_date"`
}

type Position struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	ScopeModule string `json:"scope_module"`
	Active      bool   `json:"active"`
	SortOrder   int    `json:"sort_order"`
}

type UpsertPositionRequest struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	ScopeModule string `json:"scope_module"`
	Active      bool   `json:"active"`
	SortOrder   int    `json:"sort_order"`
}

type Permission struct {
	Code      string `json:"code"`
	Label     string `json:"label"`
	UserCount int    `json:"user_count"`
	RoleCount int    `json:"role_count"`
}

type PermissionAssignment struct {
	ID              string `json:"id"`
	PermissionCode  string `json:"permission_code"`
	PermissionLabel string `json:"permission_label"`
	PositionCode    string `json:"position_code"`
	PositionName    string `json:"position_name"`
	ScopeModule     string `json:"scope_module"`
}

type UpsertPermissionAssignmentRequest struct {
	PermissionCode string `json:"permission_code"`
	PositionCode   string `json:"position_code"`
	Assigned       bool   `json:"assigned"`
}

type CodeLabelOption struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

type CodeNameOption struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type RolePermissionAssignmentFilters struct {
	Roles       []CodeLabelOption `json:"roles"`
	Permissions []CodeLabelOption `json:"permissions"`
}

type RolePermissionAssignment struct {
	ID              string `json:"id"`
	RoleCode        string `json:"role_code"`
	RoleLabel       string `json:"role_label"`
	PermissionCode  string `json:"permission_code"`
	PermissionLabel string `json:"permission_label"`
}

type UpsertRolePermissionAssignmentRequest struct {
	RoleCode       string `json:"role_code"`
	PermissionCode string `json:"permission_code"`
	Assigned       bool   `json:"assigned"`
}

type PositionRoleAssignmentFilters struct {
	Positions []CodeNameOption   `json:"positions"`
	Roles     []CodeLabelOption `json:"roles"`
}

type PositionRoleAssignment struct {
	ID           string `json:"id"`
	PositionCode string `json:"position_code"`
	PositionName string `json:"position_name"`
	RoleCode     string `json:"role_code"`
	RoleLabel    string `json:"role_label"`
}

type UpsertPositionRoleAssignmentRequest struct {
	PositionCode string `json:"position_code"`
	RoleCode     string `json:"role_code"`
	Assigned     bool   `json:"assigned"`
}

type DossierRequirement struct {
	ID                   string `json:"id"`
	SourceModule         string `json:"source_module"`
	RelationType         string `json:"relation_type"`
	MinCount             int    `json:"min_count"`
	RequiredForReadiness bool   `json:"required_for_readiness"`
	RequiredForSubmit    bool   `json:"required_for_submit"`
	RequiredForApprove   bool   `json:"required_for_approve"`
}

type CreateDossierRequirementRequest struct {
	SourceModule         string `json:"source_module"`
	RelationType         string `json:"relation_type"`
	MinCount             int    `json:"min_count"`
	RequiredForReadiness bool   `json:"required_for_readiness"`
	RequiredForSubmit    bool   `json:"required_for_submit"`
	RequiredForApprove   bool   `json:"required_for_approve"`
}

type EducationTaxonomy struct {
	ID        string `json:"id"`
	Domain    string `json:"domain"`
	Code      string `json:"code"`
	LabelRO   string `json:"label_ro"`
	LabelEN   string `json:"label_en"`
	Active    bool   `json:"active"`
	SortOrder int    `json:"sort_order"`
}

type CreateEducationTaxonomyRequest struct {
	Domain    string `json:"domain"`
	Code      string `json:"code"`
	LabelRO   string `json:"label_ro"`
	LabelEN   string `json:"label_en"`
	Active    bool   `json:"active"`
	SortOrder int    `json:"sort_order"`
}

type WorkflowDefinition struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	InitialStep string `json:"initial_step"`
	SLAHours    int    `json:"sla_hours"`
	Active      bool   `json:"active"`
}

type CreateWorkflowDefinitionRequest struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	InitialStep string `json:"initial_step"`
	SLAHours    int    `json:"sla_hours"`
	Active      bool   `json:"active"`
}

type Nomenclature struct {
	ID        string `json:"id"`
	Domain    string `json:"domain"`
	Code      string `json:"code"`
	LabelRO   string `json:"label_ro"`
	LabelEN   string `json:"label_en"`
	Active    bool   `json:"active"`
	SortOrder int    `json:"sort_order"`
}

type CreateNomenclatureRequest struct {
	Domain    string `json:"domain"`
	Code      string `json:"code"`
	LabelRO   string `json:"label_ro"`
	LabelEN   string `json:"label_en"`
	Active    bool   `json:"active"`
	SortOrder int    `json:"sort_order"`
}

type AuthMethodSetting struct {
	Code          string `json:"code"`
	Enabled       bool   `json:"enabled"`
	PrimaryMethod bool   `json:"primary_method"`
	SortOrder     int    `json:"sort_order"`
}

type UpdateAuthMethodSettingRequest struct {
	Code          string `json:"code"`
	Enabled       bool   `json:"enabled"`
	PrimaryMethod bool   `json:"primary_method"`
	SortOrder     int    `json:"sort_order"`
}

type ModuleSetting struct {
	Code   string `json:"code"`
	Active bool   `json:"active"`
}

type UpdateModuleSettingRequest struct {
	Code   string `json:"code"`
	Active bool   `json:"active"`
}

type GdprSetting struct {
	Code      string `json:"code"`
	ValueType string `json:"value_type"`
	ValueText string `json:"value_text"`
	ValueBool bool   `json:"value_bool"`
	ValueInt  int    `json:"value_int"`
}

type UpdateGdprSettingRequest struct {
	Code      string `json:"code"`
	ValueType string `json:"value_type"`
	ValueText string `json:"value_text"`
	ValueBool bool   `json:"value_bool"`
	ValueInt  int    `json:"value_int"`
}

type OrgUnit struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	ParentCode string `json:"parent_code"`
	ParentName string `json:"parent_name"`
	Active     bool   `json:"active"`
	SortOrder  int    `json:"sort_order"`
}

type UpsertOrgUnitRequest struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	ParentCode string `json:"parent_code"`
	Active     bool   `json:"active"`
	SortOrder  int    `json:"sort_order"`
}

type OIDCClient struct {
	ClientID     string   `json:"client_id"`
	ClientName   string   `json:"client_name"`
	PublicClient bool     `json:"public_client"`
	RequirePKCE  bool     `json:"require_pkce"`
	Active       bool     `json:"active"`
	RedirectURIs []string `json:"redirect_uris"`
	CreatedAt    string   `json:"created_at"`
}

type UpsertOIDCClientRequest struct {
	ClientID     string   `json:"client_id"`
	ClientName   string   `json:"client_name"`
	PublicClient bool     `json:"public_client"`
	RequirePKCE  bool     `json:"require_pkce"`
	Active       bool     `json:"active"`
	RedirectURIs []string `json:"redirect_uris"`
}

type Role struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

type UpsertRoleRequest struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

type UserRoleAssignment struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`
	RoleCode  string `json:"role_code"`
	RoleLabel string `json:"role_label"`
}

type UpsertUserRoleAssignmentRequest struct {
	UserID   string `json:"user_id"`
	RoleCode string `json:"role_code"`
	Assigned bool   `json:"assigned"`
}

type AuditEvent struct {
	ID           string `json:"id"`
	ActorSubject string `json:"actor_subject"`
	Domain       string `json:"domain"`
	Action       string `json:"action"`
	TargetType   string `json:"target_type"`
	TargetID     string `json:"target_id"`
	Status       string `json:"status"`
	Summary      string `json:"summary"`
	CreatedAt    string `json:"created_at"`
}
