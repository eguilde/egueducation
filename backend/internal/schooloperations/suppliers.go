package schooloperations

import (
	"net/http"

	"github.com/eguilde/egueducation/internal/httpx"
)

// ListSuppliers exposes the shared party registry as a tenant- and
// institution-scoped selector. School Operations never owns a duplicate
// supplier master.
func (s *Service) ListSuppliers(w http.ResponseWriter, r *http.Request) {
	tenant, institutionID, _, _, ok := contractScope(w, r, false)
	if !ok {
		return
	}
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"display_name": {}, "code": {}, "tax_id": {},
	}, []string{"display_name", "code", "tax_id"})
	where := " where p.tenant_code=$1 and p.institution_id=$2 and p.active and p.party_type in ('legal','institution')"
	args := []any{tenant, institutionID}
	for _, field := range []string{"display_name", "code", "tax_id"} {
		if value := query.Filters[field]; value != "" {
			args = append(args, value)
			where += " and p." + field + " ilike '%' || $" + itoa(len(args)) + " || '%'"
		}
	}
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from app_parties p"+where, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "school_supplier_list_failed"})
		return
	}
	sortColumns := map[string]string{"display_name": "p.display_name", "code": "p.code", "tax_id": "p.tax_id"}
	sortColumn := sortColumns[query.Sort]
	if sortColumn == "" {
		sortColumn = sortColumns["display_name"]
	}
	direction := "asc"
	if query.Direction == "desc" {
		direction = "desc"
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), "select p.id::text,p.code,p.display_name,p.tax_id from app_parties p"+where+" order by "+sortColumn+" "+direction+",p.id limit $"+itoa(len(args)-1)+" offset $"+itoa(len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "school_supplier_list_failed"})
		return
	}
	defer rows.Close()
	items := make([]SupplierOption, 0)
	for rows.Next() {
		var item SupplierOption
		if err = rows.Scan(&item.ID, &item.Code, &item.DisplayName, &item.TaxID); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "school_supplier_list_failed"})
			return
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "school_supplier_list_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}
