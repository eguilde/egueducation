package education

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/eguilde/egueducation/internal/httpx"
)

type EligibleGovernanceUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// EligiblePortfolioOwner is the canonical, immutable personnel/user pairing
// that an institution administrator may select when opening a teacher
// portfolio.  Returning the pair from one scoped query prevents a browser
// from composing identifiers belonging to different people.
type EligiblePortfolioOwner struct {
	UserID           string `json:"user_id"`
	PersonnelID      string `json:"personnel_id"`
	DisplayName      string `json:"display_name"`
	RoleTitle        string `json:"role_title"`
	EmploymentStatus string `json:"employment_status"`
}

// EligibleGovernanceUsers returns active users in the current tenant and institution.
func (s *Service) EligibleGovernanceUsers(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{"name": {}}, []string{"name"})
	if query.Sort == "" {
		query.Sort = "name"
	}
	where := `from app_users u join app_memberships m on m.user_id = u.id join app_tenants t on t.code = m.tenant_code
		where m.tenant_code = public.current_tenant_code() and t.institution_id = $1 and t.active and u.status = 'active' and m.active
		and m.start_date <= current_date and (m.end_date is null or m.end_date >= current_date)
		and nullif(trim(u.name), '') is not null`
	args := []any{s.institutionID(r)}
	if name := strings.TrimSpace(query.Filters["name"]); name != "" {
		args = append(args, "%"+strings.ToLower(name)+"%")
		where += " and lower(u.name) like $2"
	}
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(distinct u.id) "+where, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_eligible_users_failed"})
		return
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), `select u.id::text, u.name `+where+` group by u.id, u.name order by lower(u.name) `+query.Direction+`, u.id limit $`+strconv.Itoa(len(args)-1)+` offset $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_eligible_users_failed"})
		return
	}
	defer rows.Close()
	items := make([]EligibleGovernanceUser, 0)
	for rows.Next() {
		var item EligibleGovernanceUser
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_eligible_users_failed"})
			return
		}
		item.Name = strings.TrimSpace(item.Name)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_eligible_users_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

// EligiblePortfolioOwners returns only active personnel records with an active
// eligible application-user membership in this authenticated tenant and
// institution.  It is deliberately protected by portfolios.manage and does
// not expose raw user lookup outside this narrowly needed selection surface.
func (s *Service) EligiblePortfolioOwners(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"display_name": {}, "role_title": {}, "employment_status": {},
	}, []string{"display_name", "role_title", "employment_status"})
	if query.Sort == "" {
		query.Sort = "display_name"
	}
	where := ` from education_personnel person
		where person.institution_id = $1
		and person.status = 'active'
		and person.app_user_id is not null
		and public.education_membership_is_eligible(person.app_user_id, public.current_tenant_code(), $1, null)`
	args := []any{s.institutionID(r)}
	filters := map[string]string{
		"display_name":      "lower(person.full_name)",
		"role_title":        "lower(person.role_title)",
		"employment_status": "lower(person.status)",
	}
	for key, column := range filters {
		if value := strings.TrimSpace(query.Filters[key]); value != "" {
			args = append(args, "%"+strings.ToLower(value)+"%")
			where += " and " + column + " like $" + strconv.Itoa(len(args))
		}
	}
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*)"+where, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_eligible_portfolio_owners_failed"})
		return
	}
	sortColumn := map[string]string{
		"display_name": "person.full_name", "role_title": "person.role_title", "employment_status": "person.status",
	}[query.Sort]
	if sortColumn == "" {
		sortColumn = "person.full_name"
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), "select person.app_user_id::text, person.id::text, person.full_name, person.role_title, person.status"+where+" order by "+sortColumn+" "+strings.ToUpper(query.Direction)+", person.id limit $"+strconv.Itoa(len(args)-1)+" offset $"+strconv.Itoa(len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_eligible_portfolio_owners_failed"})
		return
	}
	defer rows.Close()
	items := make([]EligiblePortfolioOwner, 0, query.PageSize)
	for rows.Next() {
		var item EligiblePortfolioOwner
		if err := rows.Scan(&item.UserID, &item.PersonnelID, &item.DisplayName, &item.RoleTitle, &item.EmploymentStatus); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_eligible_portfolio_owners_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_eligible_portfolio_owners_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}
