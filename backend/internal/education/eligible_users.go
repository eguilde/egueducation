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

// EligibleGovernanceUsers returns active users in the current tenant and institution.
func (s *Service) EligibleGovernanceUsers(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{"name": {}}, []string{"name"})
	if query.Sort == "" {
		query.Sort = "name"
	}
	where := `from app_users u join app_memberships m on m.user_id = u.id join app_tenants t on t.code = m.tenant_code
		where m.tenant_code = public.current_tenant_code() and t.institution_id = $1 and t.active and m.active
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
	rows, err := s.pool.Query(r.Context(), `select distinct u.id::text, u.name `+where+` order by lower(u.name) `+query.Direction+`, u.id limit $`+strconv.Itoa(len(args)-1)+` offset $`+strconv.Itoa(len(args)), args...)
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
