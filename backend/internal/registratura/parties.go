package registratura

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"unicode"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/eguilde/egueducation/internal/httpx"
)

type PartyPage struct {
	Items    []Party `json:"items"`
	Total    int     `json:"total"`
	Page     int     `json:"page"`
	PageSize int     `json:"pageSize"`
}

func (s *Service) ListParties(w http.ResponseWriter, r *http.Request) {
	institutionID := s.institutionID(r)
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"code":       {},
			"party_type": {},
			"query":      {},
			"active":     {},
		},
		[]string{"display_name", "party_type", "code", "updated_at"},
	)

	whereClause, args := buildPartyFilters(institutionID, query.Filters)

	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from app_parties p "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "parties_list_failed"})
		return
	}

	sortColumn := partySortColumn(query.Sort)
	offset := (query.Page - 1) * query.PageSize
	args = append(args, query.PageSize, offset)

	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select
			p.id::text,
			p.code,
			p.party_type,
			p.display_name,
			p.short_name,
			p.first_name,
			p.last_name,
			p.legal_name,
			p.identifier_code,
			p.tax_id,
			p.phone_number,
			p.email,
			p.address_line1,
			p.address_line2,
			p.locality,
			p.county,
			p.country,
			p.notes,
			p.is_default_organization,
			p.active,
			to_char(p.created_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			to_char(p.updated_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		from app_parties p
		%s
		order by %s %s, p.display_name asc
		limit $%d offset $%d
	`, whereClause, sortColumn, strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "parties_list_failed"})
		return
	}
	defer rows.Close()

	items := make([]Party, 0, query.PageSize)
	for rows.Next() {
		var item Party
		if err := rows.Scan(
			&item.ID,
			&item.Code,
			&item.PartyType,
			&item.DisplayName,
			&item.ShortName,
			&item.FirstName,
			&item.LastName,
			&item.LegalName,
			&item.IdentifierCode,
			&item.TaxID,
			&item.PhoneNumber,
			&item.Email,
			&item.AddressLine1,
			&item.AddressLine2,
			&item.Locality,
			&item.County,
			&item.Country,
			&item.Notes,
			&item.IsDefaultOrganization,
			&item.Active,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "parties_list_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "parties_list_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) LookupParties(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("query")))
	institutionID := s.institutionID(r)

	args := []any{institutionID}
	where := `where institution_id = $1 and active = true`
	if query != "" {
		args = append(args, "%"+query+"%")
		where += fmt.Sprintf(" and (lower(display_name) like $%d or lower(code) like $%d or lower(legal_name) like $%d or lower(first_name) like $%d or lower(last_name) like $%d)", len(args), len(args), len(args), len(args), len(args))
	}

	rows, err := s.pool.Query(r.Context(), `
		select
			id::text,
			code,
			party_type,
			display_name,
			short_name,
			first_name,
			last_name,
			legal_name,
			identifier_code,
			tax_id,
			phone_number,
			email,
			address_line1,
			address_line2,
			locality,
			county,
			country,
			notes,
			is_default_organization,
			active,
			to_char(created_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			to_char(updated_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		from app_parties
		`+where+`
		order by is_default_organization desc, display_name asc
	`, args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "parties_lookup_failed"})
		return
	}
	defer rows.Close()

	items, err := scanParties(rows)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "parties_lookup_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

func (s *Service) GetParty(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_party_id"})
		return
	}

	item, err := s.findPartyByID(r.Context(), id)
	if err != nil {
		if err == pgx.ErrNoRows {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "party_not_found"})
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "party_load_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreateParty(w http.ResponseWriter, r *http.Request) {
	var req CreatePartyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_party_payload"})
		return
	}
	if strings.TrimSpace(req.InstitutionID) == "" {
		req.InstitutionID = s.institutionID(r)
	}

	item, err := s.createParty(r.Context(), req)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "party_create_failed", "message": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdateParty(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_party_id"})
		return
	}

	var req UpdatePartyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_party_payload"})
		return
	}

	item, err := s.updateParty(r.Context(), id, req)
	if err != nil {
		if err == pgx.ErrNoRows {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "party_not_found"})
			return
		}
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "party_update_failed", "message": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteParty(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_party_id"})
		return
	}
	if err := s.deleteParty(r.Context(), id); err != nil {
		if err == pgx.ErrNoRows {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "party_not_found"})
			return
		}
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "party_delete_failed", "message": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) DefaultOrganizationParty(w http.ResponseWriter, r *http.Request) {
	item, err := s.findDefaultOrganizationParty(r.Context(), s.institutionID(r))
	if err != nil {
		if err == pgx.ErrNoRows {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "party_not_found"})
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "party_load_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) findPartyByID(ctx context.Context, id string) (*Party, error) {
	row := s.pool.QueryRow(ctx, `
		select
			id::text,
			code,
			party_type,
			display_name,
			short_name,
			first_name,
			last_name,
			legal_name,
			identifier_code,
			tax_id,
			phone_number,
			email,
			address_line1,
			address_line2,
			locality,
			county,
			country,
			notes,
			is_default_organization,
			active,
			to_char(created_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			to_char(updated_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		from app_parties
		where id::text = $1
	`, id)
	var item Party
	if err := row.Scan(
		&item.ID,
		&item.Code,
		&item.PartyType,
		&item.DisplayName,
		&item.ShortName,
		&item.FirstName,
		&item.LastName,
		&item.LegalName,
		&item.IdentifierCode,
		&item.TaxID,
		&item.PhoneNumber,
		&item.Email,
		&item.AddressLine1,
		&item.AddressLine2,
		&item.Locality,
		&item.County,
		&item.Country,
		&item.Notes,
		&item.IsDefaultOrganization,
		&item.Active,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Service) findDefaultOrganizationParty(ctx context.Context, institutionID string) (*Party, error) {
	row := s.pool.QueryRow(ctx, `
		select
			id::text,
			code,
			party_type,
			display_name,
			short_name,
			first_name,
			last_name,
			legal_name,
			identifier_code,
			tax_id,
			phone_number,
			email,
			address_line1,
			address_line2,
			locality,
			county,
			country,
			notes,
			is_default_organization,
			active,
			to_char(created_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			to_char(updated_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		from app_parties
		where institution_id = $1
			and is_default_organization = true
			and active = true
		order by updated_at desc, created_at desc
		limit 1
	`, institutionID)
	var item Party
	if err := row.Scan(
		&item.ID,
		&item.Code,
		&item.PartyType,
		&item.DisplayName,
		&item.ShortName,
		&item.FirstName,
		&item.LastName,
		&item.LegalName,
		&item.IdentifierCode,
		&item.TaxID,
		&item.PhoneNumber,
		&item.Email,
		&item.AddressLine1,
		&item.AddressLine2,
		&item.Locality,
		&item.County,
		&item.Country,
		&item.Notes,
		&item.IsDefaultOrganization,
		&item.Active,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Service) createParty(ctx context.Context, req CreatePartyRequest) (*Party, error) {
	institutionID := strings.TrimSpace(req.InstitutionID)
	if institutionID == "" {
		return nil, fmt.Errorf("missing institution")
	}

	req.PartyType = normalizePartyType(req.PartyType)
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	req.ShortName = strings.TrimSpace(req.ShortName)
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)
	req.LegalName = strings.TrimSpace(req.LegalName)
	req.IdentifierCode = strings.TrimSpace(req.IdentifierCode)
	req.TaxID = strings.TrimSpace(req.TaxID)
	req.PhoneNumber = strings.TrimSpace(req.PhoneNumber)
	req.Email = strings.TrimSpace(req.Email)
	req.AddressLine1 = strings.TrimSpace(req.AddressLine1)
	req.AddressLine2 = strings.TrimSpace(req.AddressLine2)
	req.Locality = strings.TrimSpace(req.Locality)
	req.County = strings.TrimSpace(req.County)
	req.Country = strings.TrimSpace(req.Country)
	req.Notes = strings.TrimSpace(req.Notes)
	if req.DisplayName == "" || req.PartyType == "" {
		return nil, fmt.Errorf("missing party fields")
	}
	if req.Code == "" {
		req.Code = uniquePartyCode(req.PartyType, req.DisplayName, req.IdentifierCode, req.LegalName)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if req.IsDefaultOrganization {
		if _, err := tx.Exec(ctx, `update app_parties set is_default_organization = false, updated_at = now() where institution_id = $1`, institutionID); err != nil {
			return nil, err
		}
	}

	row := tx.QueryRow(ctx, `
		insert into app_parties (
			tenant_code,
			institution_id,
			code,
			party_type,
			display_name,
			short_name,
			first_name,
			last_name,
			legal_name,
			identifier_code,
			tax_id,
			phone_number,
			email,
			address_line1,
			address_line2,
			locality,
			county,
			country,
			notes,
			is_default_organization,
			active,
			created_at,
			updated_at
		)
		values (
			coalesce(
				(
					select p.tenant_code
					from app_parties p
					where p.institution_id = $1
						and p.is_default_organization = true
					order by p.updated_at desc, p.created_at desc
					limit 1
				),
				(
					select p.tenant_code
					from app_parties p
					where p.institution_id = $1
					order by p.is_default_organization desc, p.updated_at desc, p.created_at desc
					limit 1
				),
				(
					select t.code
					from app_tenants t
					where t.institution_id = $1
					order by t.updated_at desc, t.created_at desc
					limit 1
				)
			),
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, now(), now()
		)
		returning
			id::text,
			code,
			party_type,
			display_name,
			short_name,
			first_name,
			last_name,
			legal_name,
			identifier_code,
			tax_id,
			phone_number,
			email,
			address_line1,
			address_line2,
			locality,
			county,
			country,
			notes,
			is_default_organization,
			active,
			to_char(created_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			to_char(updated_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
	`, institutionID, req.Code, req.PartyType, req.DisplayName, req.ShortName, req.FirstName, req.LastName, req.LegalName, req.IdentifierCode, req.TaxID, req.PhoneNumber, req.Email, req.AddressLine1, req.AddressLine2, req.Locality, req.County, req.Country, req.Notes, req.IsDefaultOrganization, req.Active)

	item, err := scanParty(row)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *Service) updateParty(ctx context.Context, id string, req UpdatePartyRequest) (*Party, error) {
	current, err := s.findPartyByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.PartyType != nil {
		current.PartyType = normalizePartyType(*req.PartyType)
	}
	if req.DisplayName != nil {
		current.DisplayName = strings.TrimSpace(*req.DisplayName)
	}
	if req.ShortName != nil {
		current.ShortName = strings.TrimSpace(*req.ShortName)
	}
	if req.FirstName != nil {
		current.FirstName = strings.TrimSpace(*req.FirstName)
	}
	if req.LastName != nil {
		current.LastName = strings.TrimSpace(*req.LastName)
	}
	if req.LegalName != nil {
		current.LegalName = strings.TrimSpace(*req.LegalName)
	}
	if req.IdentifierCode != nil {
		current.IdentifierCode = strings.TrimSpace(*req.IdentifierCode)
	}
	if req.TaxID != nil {
		current.TaxID = strings.TrimSpace(*req.TaxID)
	}
	if req.PhoneNumber != nil {
		current.PhoneNumber = strings.TrimSpace(*req.PhoneNumber)
	}
	if req.Email != nil {
		current.Email = strings.TrimSpace(*req.Email)
	}
	if req.AddressLine1 != nil {
		current.AddressLine1 = strings.TrimSpace(*req.AddressLine1)
	}
	if req.AddressLine2 != nil {
		current.AddressLine2 = strings.TrimSpace(*req.AddressLine2)
	}
	if req.Locality != nil {
		current.Locality = strings.TrimSpace(*req.Locality)
	}
	if req.County != nil {
		current.County = strings.TrimSpace(*req.County)
	}
	if req.Country != nil {
		current.Country = strings.TrimSpace(*req.Country)
	}
	if req.Notes != nil {
		current.Notes = strings.TrimSpace(*req.Notes)
	}
	if req.IsDefaultOrganization != nil {
		current.IsDefaultOrganization = *req.IsDefaultOrganization
	}
	if req.Active != nil {
		current.Active = *req.Active
	}
	if current.DisplayName == "" || current.PartyType == "" {
		return nil, fmt.Errorf("missing party fields")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var institutionID string
	if err := tx.QueryRow(ctx, `select institution_id from app_parties where id::text = $1`, id).Scan(&institutionID); err != nil {
		return nil, err
	}
	if current.IsDefaultOrganization {
		if _, err := tx.Exec(ctx, `update app_parties set is_default_organization = false, updated_at = now() where institution_id = $1 and id::text <> $2`, institutionID, id); err != nil {
			return nil, err
		}
	}

	row := tx.QueryRow(ctx, `
		update app_parties
		set code = $1,
			party_type = $2,
			display_name = $3,
			short_name = $4,
			first_name = $5,
			last_name = $6,
			legal_name = $7,
			identifier_code = $8,
			tax_id = $9,
			phone_number = $10,
			email = $11,
			address_line1 = $12,
			address_line2 = $13,
			locality = $14,
			county = $15,
			country = $16,
			notes = $17,
			is_default_organization = $18,
			active = $19,
			updated_at = now()
		where id::text = $20
		returning
			id::text,
			code,
			party_type,
			display_name,
			short_name,
			first_name,
			last_name,
			legal_name,
			identifier_code,
			tax_id,
			phone_number,
			email,
			address_line1,
			address_line2,
			locality,
			county,
			country,
			notes,
			is_default_organization,
			active,
			to_char(created_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			to_char(updated_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
	`, current.Code, current.PartyType, current.DisplayName, current.ShortName, current.FirstName, current.LastName, current.LegalName, current.IdentifierCode, current.TaxID, current.PhoneNumber, current.Email, current.AddressLine1, current.AddressLine2, current.Locality, current.County, current.Country, current.Notes, current.IsDefaultOrganization, current.Active, id)

	item, err := scanParty(row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *Service) deleteParty(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `delete from app_parties where id::text = $1`, id)
	return err
}

func scanParty(row interface {
	Scan(dest ...any) error
}) (*Party, error) {
	var item Party
	if err := row.Scan(
		&item.ID,
		&item.Code,
		&item.PartyType,
		&item.DisplayName,
		&item.ShortName,
		&item.FirstName,
		&item.LastName,
		&item.LegalName,
		&item.IdentifierCode,
		&item.TaxID,
		&item.PhoneNumber,
		&item.Email,
		&item.AddressLine1,
		&item.AddressLine2,
		&item.Locality,
		&item.County,
		&item.Country,
		&item.Notes,
		&item.IsDefaultOrganization,
		&item.Active,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &item, nil
}

func scanParties(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]Party, error) {
	items := make([]Party, 0)
	for rows.Next() {
		var item Party
		if err := rows.Scan(
			&item.ID,
			&item.Code,
			&item.PartyType,
			&item.DisplayName,
			&item.ShortName,
			&item.FirstName,
			&item.LastName,
			&item.LegalName,
			&item.IdentifierCode,
			&item.TaxID,
			&item.PhoneNumber,
			&item.Email,
			&item.AddressLine1,
			&item.AddressLine2,
			&item.Locality,
			&item.County,
			&item.Country,
			&item.Notes,
			&item.IsDefaultOrganization,
			&item.Active,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func buildPartyFilters(institutionID string, filters map[string]string) (string, []any) {
	where := []string{"p.institution_id = $1"}
	args := []any{strings.TrimSpace(institutionID)}

	if value := strings.TrimSpace(filters["party_type"]); value != "" {
		args = append(args, value)
		where = append(where, fmt.Sprintf("p.party_type = $%d", len(args)))
	}
	if value := strings.TrimSpace(filters["code"]); value != "" {
		args = append(args, "%"+strings.ToLower(value)+"%")
		where = append(where, fmt.Sprintf("lower(p.code) like $%d", len(args)))
	}
	if value := strings.TrimSpace(filters["query"]); value != "" {
		args = append(args, "%"+strings.ToLower(value)+"%")
		where = append(where, fmt.Sprintf("(lower(p.display_name) like $%d or lower(p.legal_name) like $%d or lower(p.first_name) like $%d or lower(p.last_name) like $%d or lower(p.identifier_code) like $%d or lower(p.tax_id) like $%d)", len(args), len(args), len(args), len(args), len(args), len(args)))
	}
	if value := strings.TrimSpace(filters["active"]); value != "" {
		args = append(args, value == "true" || value == "1")
		where = append(where, fmt.Sprintf("p.active = $%d", len(args)))
	}

	return " where " + strings.Join(where, " and "), args
}

func partySortColumn(sort string) string {
	switch strings.TrimSpace(sort) {
	case "code":
		return "p.code"
	case "party_type":
		return "p.party_type"
	case "updated_at":
		return "p.updated_at"
	case "created_at":
		return "p.created_at"
	default:
		return "p.display_name"
	}
}

func normalizePartyType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "physical", "persoana_fizica", "persoana fizica", "physical_person":
		return "physical"
	case "legal", "persoana_juridica", "persoana juridica", "legal_entity":
		return "legal"
	case "institution", "institutie", "institutie publica", "institutional":
		return "institution"
	default:
		return ""
	}
}

func uniquePartyCode(partyType, displayName, identifierCode, legalName string) string {
	base := strings.TrimSpace(displayName)
	if base == "" {
		base = strings.TrimSpace(legalName)
	}
	if base == "" {
		base = strings.TrimSpace(identifierCode)
	}
	if base == "" {
		base = partyType
	}
	slug := slugify(base)
	if slug == "" {
		slug = partyType
	}
	return fmt.Sprintf("%s-%s", partyType, slug)
}

var slugSpaceRe = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		case unicode.IsSpace(r) || r == '-' || r == '_' || r == '.':
			b.WriteRune('-')
		}
	}
	slug := strings.Trim(b.String(), "-")
	slug = slugSpaceRe.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	return slug
}
