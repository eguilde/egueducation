package registratura

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	authruntime "github.com/eguilde/egueducation/internal/auth"
	"github.com/eguilde/egueducation/internal/httpx"
)

func (s *Service) ListRegistries(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `
		select
			id,
			nume,
			prefix_nr,
			nr_inceput,
			nr_curent,
			nr_urmator,
			case when data_resetare is null then null else to_char(data_resetare, 'YYYY-MM-DD') end,
			tip_registru,
			is_default,
			to_char(created_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			to_char(updated_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		from registre
		order by is_default desc, nume asc
	`)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "registries_list_failed"})
		return
	}
	defer rows.Close()

	items, err := scanRegistries(rows)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "registries_list_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

func (s *Service) GetRegistry(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(strings.TrimSpace(chi.URLParam(r, "id")), 10, 64)
	if err != nil || id <= 0 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_registry_id"})
		return
	}

	item, err := s.findRegistryByID(r.Context(), id)
	if err != nil {
		if err == pgx.ErrNoRows {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "registry_not_found"})
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "registry_load_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) GetDefaultRegistry(w http.ResponseWriter, r *http.Request) {
	item, err := s.findDefaultRegistry(r.Context())
	if err != nil {
		if err == pgx.ErrNoRows {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "registry_not_found"})
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "registry_load_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreateRegistru(w http.ResponseWriter, r *http.Request) {
	var req CreateRegistruRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_registry_payload"})
		return
	}
	item, err := s.createRegistry(r.Context(), req)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "registry_create_failed", "message": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdateRegistru(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(strings.TrimSpace(chi.URLParam(r, "id")), 10, 64)
	if err != nil || id <= 0 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_registry_id"})
		return
	}

	var req UpdateRegistruRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_registry_payload"})
		return
	}

	item, err := s.updateRegistry(r.Context(), id, req)
	if err != nil {
		if err == pgx.ErrNoRows {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "registry_not_found"})
			return
		}
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "registry_update_failed", "message": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteRegistru(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(strings.TrimSpace(chi.URLParam(r, "id")), 10, 64)
	if err != nil || id <= 0 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_registry_id"})
		return
	}
	if err := s.deleteRegistry(r.Context(), id); err != nil {
		if err == pgx.ErrNoRows {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "registry_not_found"})
			return
		}
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "registry_delete_failed", "message": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) SetDefaultRegistru(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(strings.TrimSpace(chi.URLParam(r, "id")), 10, 64)
	if err != nil || id <= 0 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_registry_id"})
		return
	}

	item, err := s.setDefaultRegistry(r.Context(), id)
	if err != nil {
		if err == pgx.ErrNoRows {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "registry_not_found"})
			return
		}
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "registry_update_failed", "message": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) BatchCreateDocuments(w http.ResponseWriter, r *http.Request) {
	var req BatchCreateDocumentsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_batch_payload"})
		return
	}

	req.Subject = strings.TrimSpace(req.Subject)
	req.DocumentType = strings.TrimSpace(req.DocumentType)
	req.Direction = strings.TrimSpace(req.Direction)
	req.Status = strings.TrimSpace(req.Status)
	req.Correspondent = strings.TrimSpace(req.Correspondent)
	req.AssignedTo = strings.TrimSpace(req.AssignedTo)
	req.Confidentiality = strings.TrimSpace(req.Confidentiality)
	req.Summary = strings.TrimSpace(req.Summary)
	if req.Count < 1 || req.Count > 100 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_batch_count"})
		return
	}
	if req.Subject == "" || req.DocumentType == "" || req.Direction == "" || req.Status == "" || req.Correspondent == "" || req.Confidentiality == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_document_fields"})
		return
	}

	items := make([]Document, 0, req.Count)
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "batch_create_failed"})
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck

	for i := 0; i < req.Count; i++ {
		createReq := CreateDocumentRequest{
			RegistruID:      &req.RegistruID,
			Subject:         req.Subject,
			DocumentType:    req.DocumentType,
			Direction:       req.Direction,
			Status:          req.Status,
			Correspondent:   req.Correspondent,
			AssignedTo:      req.AssignedTo,
			Confidentiality: req.Confidentiality,
			Summary:         req.Summary,
			DueDate:         req.DueDate,
		}
		if req.Count > 1 {
			createReq.Subject = fmt.Sprintf("%s #%d", req.Subject, i+1)
		}

		item, err := s.createDocumentTx(r.Context(), tx, createReq, authruntime.CurrentSubjectFromRequest(r), s.institutionID(r))
		if err != nil {
			if err == pgx.ErrNoRows {
				httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "registry_not_found"})
				return
			}
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "batch_create_failed", "message": err.Error()})
			return
		}
		items = append(items, item)
	}

	if err := tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "batch_create_failed"})
		return
	}

	httpx.JSON(w, http.StatusCreated, items)
}

func (s *Service) ExportPDF(w http.ResponseWriter, r *http.Request) {
	var req ExportDocumentsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_export_payload"})
		return
	}

	docs, registry, err := s.fetchDocumentsForExport(r.Context(), req)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "registratura_export_failed"})
		return
	}

	title := "Registratura"
	if registry != nil {
		title = fmt.Sprintf("Registratura - %s", registry.Nume)
	}
	pdf := buildSimplePDF(title, exportLines(docs))
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="registratura.pdf"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdf)
}

func (s *Service) createRegistry(ctx context.Context, req CreateRegistruRequest) (*Registru, error) {
	nume := strings.TrimSpace(req.Nume)
	prefixNr := strings.TrimSpace(req.PrefixNr)
	tipRegistru := strings.TrimSpace(req.TipRegistru)
	if nume == "" || prefixNr == "" || tipRegistru == "" {
		return nil, fmt.Errorf("missing registry fields")
	}
	if req.NrInceput < 1 {
		req.NrInceput = 1
	}
	if req.NrCurent == "" {
		req.NrCurent = fmt.Sprintf("%04d", req.NrInceput-1)
	}
	if req.NrUrmator == "" {
		req.NrUrmator = fmt.Sprintf("%04d", req.NrInceput)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if req.IsDefault {
		if _, err := tx.Exec(ctx, `update registre set is_default = false, updated_at = now()`); err != nil {
			return nil, err
		}
	}

	var id int64
	var created Registru
	var dataResetare sql.NullString
	err = tx.QueryRow(ctx, `
		insert into registre (nume, prefix_nr, nr_inceput, nr_curent, nr_urmator, data_resetare, tip_registru, is_default, created_at, updated_at)
		values ($1,$2,$3,$4,$5,$6,$7,$8,now(),now())
		returning id,
			nume,
			prefix_nr,
			nr_inceput,
			nr_curent,
			nr_urmator,
			case when data_resetare is null then null else to_char(data_resetare, 'YYYY-MM-DD') end,
			tip_registru,
			is_default,
			to_char(created_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			to_char(updated_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
	`, nume, prefixNr, req.NrInceput, req.NrCurent, req.NrUrmator, req.DataResetare, tipRegistru, req.IsDefault).Scan(
		&id,
		&created.Nume,
		&created.PrefixNr,
		&created.NrInceput,
		&created.NrCurent,
		&created.NrUrmator,
		&dataResetare,
		&created.TipRegistru,
		&created.IsDefault,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if dataResetare.Valid {
		created.DataResetare = &dataResetare.String
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	created.ID = id
	return &created, nil
}

func (s *Service) updateRegistry(ctx context.Context, id int64, req UpdateRegistruRequest) (*Registru, error) {
	current, err := s.findRegistryByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Nume != nil {
		current.Nume = strings.TrimSpace(*req.Nume)
	}
	if req.PrefixNr != nil {
		current.PrefixNr = strings.TrimSpace(*req.PrefixNr)
	}
	if req.NrInceput != nil && *req.NrInceput > 0 {
		current.NrInceput = *req.NrInceput
	}
	if req.NrCurent != nil {
		current.NrCurent = strings.TrimSpace(*req.NrCurent)
	}
	if req.NrUrmator != nil {
		current.NrUrmator = strings.TrimSpace(*req.NrUrmator)
	}
	if req.DataResetare != nil {
		current.DataResetare = req.DataResetare
	}
	if req.TipRegistru != nil {
		current.TipRegistru = strings.TrimSpace(*req.TipRegistru)
	}
	if req.IsDefault != nil {
		current.IsDefault = *req.IsDefault
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if current.IsDefault {
		if _, err := tx.Exec(ctx, `update registre set is_default = false, updated_at = now() where id <> $1`, id); err != nil {
			return nil, err
		}
	}

	if _, err := tx.Exec(ctx, `
		update registre
		set nume = $1,
			prefix_nr = $2,
			nr_inceput = $3,
			nr_curent = $4,
			nr_urmator = $5,
			data_resetare = $6,
			tip_registru = $7,
			is_default = $8,
			updated_at = now()
		where id = $9
	`, current.Nume, current.PrefixNr, current.NrInceput, current.NrCurent, current.NrUrmator, current.DataResetare, current.TipRegistru, current.IsDefault, id); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.findRegistryByID(ctx, id)
}

func (s *Service) deleteRegistry(ctx context.Context, id int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var wasDefault bool
	if err := tx.QueryRow(ctx, `delete from registre where id = $1 returning is_default`, id).Scan(&wasDefault); err != nil {
		return err
	}
	if wasDefault {
		_, _ = tx.Exec(ctx, `
			update registre
			set is_default = true, updated_at = now()
			where id = (
				select id
				from registre
				order by is_default desc, id asc
				limit 1
			)
		`)
	}
	return tx.Commit(ctx)
}

func (s *Service) setDefaultRegistry(ctx context.Context, id int64) (*Registru, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `update registre set is_default = false, updated_at = now()`); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `update registre set is_default = true, updated_at = now() where id = $1`, id); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.findRegistryByID(ctx, id)
}

func (s *Service) findRegistryByID(ctx context.Context, id int64) (*Registru, error) {
	row := s.pool.QueryRow(ctx, `
		select
			id,
			nume,
			prefix_nr,
			nr_inceput,
			nr_curent,
			nr_urmator,
			case when data_resetare is null then null else to_char(data_resetare, 'YYYY-MM-DD') end,
			tip_registru,
			is_default,
			to_char(created_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			to_char(updated_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		from registre
		where id = $1
	`, id)
	var (
		item         Registru
		dataResetare sql.NullString
	)
	if err := row.Scan(
		&item.ID,
		&item.Nume,
		&item.PrefixNr,
		&item.NrInceput,
		&item.NrCurent,
		&item.NrUrmator,
		&dataResetare,
		&item.TipRegistru,
		&item.IsDefault,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if dataResetare.Valid {
		item.DataResetare = &dataResetare.String
	}
	return &item, nil
}

func (s *Service) findDefaultRegistry(ctx context.Context) (*Registru, error) {
	row := s.pool.QueryRow(ctx, `
		select
			id,
			nume,
			prefix_nr,
			nr_inceput,
			nr_curent,
			nr_urmator,
			case when data_resetare is null then null else to_char(data_resetare, 'YYYY-MM-DD') end,
			tip_registru,
			is_default,
			to_char(created_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			to_char(updated_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		from registre
		where is_default = true
		order by id asc
		limit 1
	`)
	var (
		item         Registru
		dataResetare sql.NullString
	)
	if err := row.Scan(
		&item.ID,
		&item.Nume,
		&item.PrefixNr,
		&item.NrInceput,
		&item.NrCurent,
		&item.NrUrmator,
		&dataResetare,
		&item.TipRegistru,
		&item.IsDefault,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if dataResetare.Valid {
		item.DataResetare = &dataResetare.String
	}
	return &item, nil
}

func (s *Service) resolveRegistryID(ctx context.Context, tx pgx.Tx, registruID *int64) (int64, error) {
	if registruID != nil && *registruID > 0 {
		var id int64
		if err := tx.QueryRow(ctx, `select id from registre where id = $1`, *registruID).Scan(&id); err != nil {
			return 0, err
		}
		return id, nil
	}

	var id int64
	if err := tx.QueryRow(ctx, `select id from registre where is_default = true order by id asc limit 1`).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Service) createDocumentTx(ctx context.Context, tx pgx.Tx, req CreateDocumentRequest, createdBy string, institutionID string) (Document, error) {
	var document Document

	registruID, err := s.resolveRegistryID(ctx, tx, req.RegistruID)
	if err != nil {
		return document, err
	}

	registryNumber, err := nextRegistryNumber(ctx, tx, registruID)
	if err != nil {
		return document, err
	}

	var dueDate any
	if req.DueDate != nil && strings.TrimSpace(*req.DueDate) != "" {
		dueDate = strings.TrimSpace(*req.DueDate)
	}

	err = tx.QueryRow(ctx, `
		insert into registratura_documents (
			registru_id,
			registry_number,
			subject,
			document_type,
			direction,
			status,
			correspondent,
			assigned_to,
			institution_id,
			confidentiality,
			summary,
			due_date
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		returning
			id::text,
			registru_id,
			registry_number,
			subject,
			document_type,
			direction,
			status,
			correspondent,
			assigned_to,
			institution_id,
			confidentiality,
			summary,
			to_char(registered_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			case when due_date is null then null else to_char(due_date, 'YYYY-MM-DD') end
	`,
		registruID,
		registryNumber,
		req.Subject,
		req.DocumentType,
		req.Direction,
		req.Status,
		req.Correspondent,
		req.AssignedTo,
		institutionID,
		req.Confidentiality,
		req.Summary,
		dueDate,
	).Scan(
		&document.ID,
		&document.RegistruID,
		&document.RegistryNumber,
		&document.Subject,
		&document.DocumentType,
		&document.Direction,
		&document.Status,
		&document.Correspondent,
		&document.AssignedTo,
		&document.InstitutionID,
		&document.Confidentiality,
		&document.Summary,
		&document.RegisteredAt,
		&document.DueDate,
	)
	if err != nil {
		return document, err
	}

	if _, err := tx.Exec(ctx, `
		update registre
		set nr_curent = $1,
			nr_urmator = $2,
			updated_at = now()
		where id = $3
	`, registryNumber, nextRegistryDisplayNumber(registryNumber), registruID); err != nil {
		return document, err
	}

	if _, err := tx.Exec(ctx, `
		insert into registratura_document_versions (
			document_id,
			version_no,
			subject,
			document_type,
			direction,
			status,
			correspondent,
			assigned_to,
			confidentiality,
			summary,
			due_date,
			change_notes,
			created_by
		) values ($1::uuid, 1, $2, $3, $4, $5, $6, $7, $8, $9, $10::date, $11, $12)
	`,
		document.ID,
		document.Subject,
		document.DocumentType,
		document.Direction,
		document.Status,
		document.Correspondent,
		document.AssignedTo,
		document.Confidentiality,
		document.Summary,
		document.DueDate,
		"Inițializare document",
		createdBy,
	); err != nil {
		return document, err
	}

	return document, nil
}

func (s *Service) fetchDocumentsForExport(ctx context.Context, req ExportDocumentsRequest) ([]Document, *Registru, error) {
	filters := map[string]string{}
	if req.StartDate != nil && strings.TrimSpace(*req.StartDate) != "" {
		filters["registered_at_from"] = strings.TrimSpace(*req.StartDate)
	}
	if req.EndDate != nil && strings.TrimSpace(*req.EndDate) != "" {
		filters["registered_at_to"] = strings.TrimSpace(*req.EndDate)
	}
	whereClause, args := buildDocumentFilters(filters)
	if req.RegistruID != nil && *req.RegistruID > 0 {
		if whereClause == "" {
			whereClause = "where d.registru_id = $1"
			args = append(args, *req.RegistruID)
		} else {
			args = append(args, *req.RegistruID)
			whereClause += fmt.Sprintf(" and d.registru_id = $%d", len(args))
		}
	}

	var registry *Registru
	if req.RegistruID != nil && *req.RegistruID > 0 {
		found, err := s.findRegistryByID(ctx, *req.RegistruID)
		if err == nil {
			registry = found
		}
	}

	sql := fmt.Sprintf(`
		select
			d.id::text,
			d.registru_id,
			d.registry_number,
			d.subject,
			d.document_type,
			d.direction,
			d.status,
			d.correspondent,
			d.assigned_to,
			d.institution_id,
			d.confidentiality,
			d.summary,
			to_char(d.registered_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') as registered_at,
			case when d.due_date is null then null else to_char(d.due_date, 'YYYY-MM-DD') end as due_date
		from registratura_documents d
		%s
		order by d.registered_at desc, d.registry_number asc
	`, whereClause)

	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	items := []Document{}
	for rows.Next() {
		var item Document
		if err := rows.Scan(
			&item.ID,
			&item.RegistruID,
			&item.RegistryNumber,
			&item.Subject,
			&item.DocumentType,
			&item.Direction,
			&item.Status,
			&item.Correspondent,
			&item.AssignedTo,
			&item.InstitutionID,
			&item.Confidentiality,
			&item.Summary,
			&item.RegisteredAt,
			&item.DueDate,
		); err != nil {
			return nil, nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return items, registry, nil
}

func scanRegistries(rows pgx.Rows) ([]Registru, error) {
	items := make([]Registru, 0)
	for rows.Next() {
		var (
			item         Registru
			dataResetare sql.NullString
			createdAt    string
			updatedAt    string
		)
		if err := rows.Scan(
			&item.ID,
			&item.Nume,
			&item.PrefixNr,
			&item.NrInceput,
			&item.NrCurent,
			&item.NrUrmator,
			&dataResetare,
			&item.TipRegistru,
			&item.IsDefault,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, err
		}
		if dataResetare.Valid {
			item.DataResetare = &dataResetare.String
		}
		item.CreatedAt = createdAt
		item.UpdatedAt = updatedAt
		items = append(items, item)
	}
	return items, rows.Err()
}

func nextRegistryNumber(ctx context.Context, tx pgx.Tx, registruID int64) (string, error) {
	var prefix string
	if err := tx.QueryRow(ctx, `
		select prefix_nr
		from registre
		where id = $1
		for update
	`, registruID).Scan(&prefix); err != nil {
		return "", err
	}

	year := time.Now().UTC().Year()
	var count int
	if err := tx.QueryRow(ctx, `
		select count(*)
		from registratura_documents
		where registru_id = $1 and extract(year from registered_at) = $2
	`, registruID, year).Scan(&count); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%d-%04d", prefix, year, count+1), nil
}

func nextRegistryDisplayNumber(registryNumber string) string {
	parts := strings.Split(registryNumber, "-")
	if len(parts) < 3 {
		return registryNumber
	}
	last := parts[len(parts)-1]
	value, err := strconv.Atoi(last)
	if err != nil {
		return registryNumber
	}
	parts[len(parts)-1] = fmt.Sprintf("%04d", value+1)
	return strings.Join(parts, "-")
}

func exportLines(docs []Document) []string {
	lines := make([]string, 0, len(docs)+2)
	lines = append(lines, fmt.Sprintf("Total documente: %d", len(docs)))
	lines = append(lines, "")
	for _, doc := range docs {
		lines = append(lines, fmt.Sprintf("%s | %s | %s | %s | %s", doc.RegistryNumber, doc.Subject, doc.DocumentType, doc.Direction, doc.RegisteredAt))
	}
	return lines
}

func buildSimplePDF(title string, lines []string) []byte {
	var content bytes.Buffer
	writeLine := func(text string, x, y int) {
		fmt.Fprintf(&content, "BT /F1 10 Tf %d %d Td (%s) Tj ET\n", x, y, escapePDFText(text))
	}

	writeLine(title, 50, 780)
	y := 760
	for _, line := range lines {
		if y < 40 {
			break
		}
		if len(line) > 120 {
			line = line[:120]
		}
		writeLine(line, 50, y)
		y -= 14
	}

	stream := content.String()
	var pdf bytes.Buffer
	offsets := make([]int, 0, 6)
	writeObj := func(obj string) {
		offsets = append(offsets, pdf.Len())
		pdf.WriteString(obj)
	}

	pdf.WriteString("%PDF-1.4\n")
	writeObj("1 0 obj << /Type /Catalog /Pages 2 0 R >> endobj\n")
	writeObj("2 0 obj << /Type /Pages /Kids [3 0 R] /Count 1 >> endobj\n")
	writeObj("3 0 obj << /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >> endobj\n")
	writeObj("4 0 obj << /Type /Font /Subtype /Type1 /BaseFont /Helvetica >> endobj\n")
	writeObj(fmt.Sprintf("5 0 obj << /Length %d >> stream\n%s\nendstream endobj\n", len(stream), stream))

	xrefStart := pdf.Len()
	pdf.WriteString("xref\n0 6\n0000000000 65535 f \n")
	for _, offset := range offsets {
		pdf.WriteString(fmt.Sprintf("%010d 00000 n \n", offset))
	}
	pdf.WriteString("trailer << /Size 6 /Root 1 0 R >>\nstartxref\n")
	pdf.WriteString(fmt.Sprintf("%d\n%%EOF\n", xrefStart))
	return pdf.Bytes()
}

func escapePDFText(text string) string {
	text = strings.ReplaceAll(text, "\\", "\\\\")
	text = strings.ReplaceAll(text, "(", "\\(")
	text = strings.ReplaceAll(text, ")", "\\)")
	return text
}
