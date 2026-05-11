package admin

import (
	"net/http"
	"slices"
	"sort"
	"strings"

	"github.com/eguilde/egueducation/internal/config"
	"github.com/eguilde/egueducation/internal/httpx"
)

type Service struct {
	cfg   config.Config
	users []AdminUser
}

func NewService(cfg config.Config) *Service {
	return &Service{
		cfg: cfg,
		users: []AdminUser{
			{ID: "usr-001", Name: "Ana Ionescu", Email: "ana.ionescu@egueducation.ro", Phone: "+40740100101", Position: "Super Admin", Locale: "ro", Status: "active", LastLoginAt: "2026-05-11T10:15:00Z"},
			{ID: "usr-002", Name: "Mihai Popa", Email: "mihai.popa@egueducation.ro", Phone: "+40740100102", Position: "Director", Locale: "ro", Status: "active", LastLoginAt: "2026-05-11T09:30:00Z"},
			{ID: "usr-003", Name: "Ioana Dumitrescu", Email: "ioana.dumitrescu@egueducation.ro", Phone: "+40740100103", Position: "Secretariat", Locale: "ro", Status: "active", LastLoginAt: "2026-05-10T15:48:00Z"},
			{ID: "usr-004", Name: "Daniel Georgescu", Email: "daniel.georgescu@egueducation.ro", Phone: "+40740100104", Position: "Registrator", Locale: "ro", Status: "pending", LastLoginAt: "2026-05-09T11:20:00Z"},
			{ID: "usr-005", Name: "Roxana Stan", Email: "roxana.stan@egueducation.ro", Phone: "+40740100105", Position: "GDPR Officer", Locale: "en", Status: "active", LastLoginAt: "2026-05-10T08:17:00Z"},
			{ID: "usr-006", Name: "Carmen Pavel", Email: "carmen.pavel@egueducation.ro", Phone: "+40740100106", Position: "HR", Locale: "ro", Status: "active", LastLoginAt: "2026-05-08T16:22:00Z"},
			{ID: "usr-007", Name: "Cristian Matei", Email: "cristian.matei@egueducation.ro", Phone: "+40740100107", Position: "Workflow Admin", Locale: "ro", Status: "suspended", LastLoginAt: "2026-05-06T09:45:00Z"},
			{ID: "usr-008", Name: "Andreea Nistor", Email: "andreea.nistor@egueducation.ro", Phone: "+40740100108", Position: "Arhivar", Locale: "ro", Status: "active", LastLoginAt: "2026-05-11T07:05:00Z"},
			{ID: "usr-009", Name: "Elena Marinescu", Email: "elena.marinescu@egueducation.ro", Phone: "+40740100109", Position: "Inspector", Locale: "ro", Status: "active", LastLoginAt: "2026-05-11T06:52:00Z"},
			{ID: "usr-010", Name: "Victor Tudor", Email: "victor.tudor@egueducation.ro", Phone: "+40740100110", Position: "Secretariat", Locale: "en", Status: "pending", LastLoginAt: "2026-05-05T14:14:00Z"},
			{ID: "usr-011", Name: "Larisa Ene", Email: "larisa.ene@egueducation.ro", Phone: "+40740100111", Position: "Director", Locale: "ro", Status: "active", LastLoginAt: "2026-05-10T10:00:00Z"},
			{ID: "usr-012", Name: "George Sandu", Email: "george.sandu@egueducation.ro", Phone: "+40740100112", Position: "Registrator", Locale: "ro", Status: "active", LastLoginAt: "2026-05-08T12:00:00Z"},
		},
	}
}

func (s *Service) Dashboard(w http.ResponseWriter, _ *http.Request) {
	httpx.JSON(w, http.StatusOK, DashboardResponse{
		Stats: DashboardStats{
			Users:       len(s.users),
			Memberships: 18,
			Positions:   8,
			Permissions: 42,
			Workflows:   9,
			Archives:    5,
		},
		Modules: []ModuleStatus{
			{Code: "registratura", Active: true},
			{Code: "workflow", Active: true},
			{Code: "earchiva", Active: true},
			{Code: "education", Active: true},
			{Code: "gdpr", Active: s.cfg.EnableGDPRFeatures},
		},
		AdminSections: []string{
			"users",
			"memberships",
			"org_units",
			"positions",
			"permissions",
			"workflow_definitions",
			"registry_nomenclatures",
			"archive_nomenclatures",
			"education_taxonomies",
			"gdpr_policies",
			"auth_methods",
		},
		Warnings: []string{
			"OIDC provider, workflow runtime, archive ingestion, and education domain services are scaffolded but not yet fully ported.",
		},
	})
}

func (s *Service) ListUsers(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"name":          {},
			"email":         {},
			"position":      {},
			"status":        {},
			"locale":        {},
			"last_login_at": {},
		},
		[]string{"name", "email", "position", "status", "locale"},
	)

	filtered := s.filteredUsers(query.Filters)
	s.sortUsers(filtered, query.Sort, query.Direction)

	start := (query.Page - 1) * query.PageSize
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + query.PageSize
	if end > len(filtered) {
		end = len(filtered)
	}

	httpx.WritePage(w, http.StatusOK, filtered[start:end], len(filtered), query.Page, query.PageSize)
}

func (s *Service) filteredUsers(filters map[string]string) []AdminUser {
	users := make([]AdminUser, 0, len(s.users))
	for _, user := range s.users {
		if !matchesFilter(user.Name, filters["name"]) {
			continue
		}
		if !matchesFilter(user.Email, filters["email"]) {
			continue
		}
		if !matchesFilter(user.Position, filters["position"]) {
			continue
		}
		if !matchesFilter(user.Status, filters["status"]) {
			continue
		}
		if !matchesFilter(user.Locale, filters["locale"]) {
			continue
		}
		users = append(users, user)
	}
	return users
}

func (s *Service) sortUsers(users []AdminUser, field, direction string) {
	if field == "" {
		return
	}

	sort.SliceStable(users, func(i, j int) bool {
		left := sortableValue(users[i], field)
		right := sortableValue(users[j], field)
		if direction == "desc" {
			return left > right
		}
		return left < right
	})
}

func sortableValue(user AdminUser, field string) string {
	switch field {
	case "name":
		return strings.ToLower(user.Name)
	case "email":
		return strings.ToLower(user.Email)
	case "position":
		return strings.ToLower(user.Position)
	case "status":
		return strings.ToLower(user.Status)
	case "locale":
		return strings.ToLower(user.Locale)
	case "last_login_at":
		return user.LastLoginAt
	default:
		return ""
	}
}

func matchesFilter(value, filter string) bool {
	if filter == "" {
		return true
	}
	return strings.Contains(strings.ToLower(value), strings.ToLower(filter))
}

func (s *Service) UserFilters(w http.ResponseWriter, _ *http.Request) {
	positions := []string{}
	statuses := []string{}
	locales := []string{}

	for _, user := range s.users {
		if !slices.Contains(positions, user.Position) {
			positions = append(positions, user.Position)
		}
		if !slices.Contains(statuses, user.Status) {
			statuses = append(statuses, user.Status)
		}
		if !slices.Contains(locales, user.Locale) {
			locales = append(locales, user.Locale)
		}
	}

	sort.Strings(positions)
	sort.Strings(statuses)
	sort.Strings(locales)

	httpx.JSON(w, http.StatusOK, map[string]any{
		"positions": positions,
		"statuses":  statuses,
		"locales":   locales,
	})
}
