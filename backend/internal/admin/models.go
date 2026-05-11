package admin

type DashboardResponse struct {
	Stats         DashboardStats `json:"stats"`
	Modules       []ModuleStatus `json:"modules"`
	AdminSections []string       `json:"admin_sections"`
	Warnings      []string       `json:"warnings"`
}

type DashboardStats struct {
	Users       int `json:"users"`
	Memberships int `json:"memberships"`
	Positions   int `json:"positions"`
	Permissions int `json:"permissions"`
	Workflows   int `json:"workflows"`
	Archives    int `json:"archives"`
}

type ModuleStatus struct {
	Code   string `json:"code"`
	Active bool   `json:"active"`
}

type AdminUser struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Position    string `json:"position"`
	Locale      string `json:"locale"`
	Status      string `json:"status"`
	LastLoginAt string `json:"last_login_at"`
}
