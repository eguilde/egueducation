package education

import (
	"net/http"
	"strings"

	"github.com/eguilde/egueducation/internal/httpx"
)

type EligibleGovernanceUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// EligibleGovernanceUsers returns only tenant-scoped active users suitable for
// immutable governance identity selectors. Display names are not used for auth.
func (s *Service) EligibleGovernanceUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `
		select distinct u.id::text, u.name
		from app_users u
		join app_memberships m on m.user_id = u.id
		where m.tenant_code = public.current_tenant_code()
		  and m.active = true
		  and nullif(trim(u.name), '') is not null
		order by u.name, u.id
	`)
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
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}
