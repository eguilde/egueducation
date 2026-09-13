package education

import (
	"errors"
	"net/http"
	"strings"

	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

// OwnPortfolioAppliedProcedureResponse is the immutable procedure version that
// governed the authenticated teacher's portfolio, together with its frozen
// section rules. It deliberately reuses the institution procedure projections
// while excluding lifecycle evidence and any unbound procedure version.
type OwnPortfolioAppliedProcedureResponse struct {
	Procedure PortfolioProcedure              `json:"procedure"`
	Rules     []PortfolioProcedureSectionRule `json:"rules"`
}

// PortfolioOwnAppliedProcedure returns only the procedure version explicitly
// bound to the authenticated teacher's portfolio. It never falls back to the
// latest institution procedure: a later, unrelated publication must not alter
// the rules governing existing portfolio evidence.
func (s *Service) PortfolioOwnAppliedProcedure(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	actorID, allowed, err := s.requireOwnPortfolio(r, recordID, portfolioReadOwnPermission)
	if err != nil || !allowed {
		writePortfolioAccessFailure(w, err)
		return
	}

	portfolio, err := s.loadOwnPortfolio(r, recordID, actorID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_own_portfolio_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_own_portfolio_procedure_failed"})
		return
	}
	if portfolio.AppliedProcedureID == "" {
		writeEducationNotFound(w, "education_own_portfolio_procedure_not_applied")
		return
	}

	var procedure PortfolioProcedure
	err = scanPortfolioProcedure(s.pool.QueryRow(r.Context(), `
		select `+portfolioProcedureColumns+`
		from education_portfolio_procedure_versions
		where id = $1::uuid
			and institution_id = $2
			and tenant_code = public.current_tenant_code()
			and lifecycle_status in ('published', 'superseded')
	`, portfolio.AppliedProcedureID, s.institutionID(r)), &procedure)
	if errors.Is(err, pgx.ErrNoRows) {
		// Do not disclose whether an old binding was withdrawn or otherwise
		// unavailable; it is not a currently readable governing procedure.
		writeEducationNotFound(w, "education_own_portfolio_procedure_not_applied")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_own_portfolio_procedure_failed"})
		return
	}

	rules := []PortfolioProcedureSectionRule{}
	rows, err := s.pool.Query(r.Context(), `
		select id::text, procedure_id::text, section_code, label_ro, label_en,
			source_catalog_version, required, sort_order, active
		from education_portfolio_procedure_section_rules
		where procedure_id = $1::uuid
			and institution_id = $2
			and tenant_code = public.current_tenant_code()
		order by sort_order, section_code
	`, procedure.ID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_own_portfolio_procedure_rules_failed"})
		return
	}
	defer rows.Close()
	for rows.Next() {
		var rule PortfolioProcedureSectionRule
		if err := rows.Scan(
			&rule.ID,
			&rule.ProcedureID,
			&rule.SectionCode,
			&rule.LabelRO,
			&rule.LabelEN,
			&rule.SourceCatalogVersion,
			&rule.Required,
			&rule.SortOrder,
			&rule.Active,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_own_portfolio_procedure_rules_failed"})
			return
		}
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_own_portfolio_procedure_rules_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, OwnPortfolioAppliedProcedureResponse{
		Procedure: procedure,
		Rules:     rules,
	})
}
