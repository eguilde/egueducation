package education

import (
	"net/http"
	"strings"

	authruntime "github.com/eguilde/egueducation/internal/auth"
	"github.com/eguilde/egueducation/internal/httpx"
)

func (s *Service) RequireInstitutionContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimSpace(authruntime.CurrentInstitutionIDFromRequest(r)) == "" {
			httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "education_institution_context_required"})
			return
		}

		next.ServeHTTP(w, r)
	})
}
