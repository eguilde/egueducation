package registratura

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/eguilde/egueducation/internal/audit"
	authruntime "github.com/eguilde/egueducation/internal/auth"
	"github.com/eguilde/egueducation/internal/httpx"
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func (s *Service) logAudit(r *http.Request, action string, targetType string, targetID string, summary string, details map[string]any) {
	_ = audit.Log(r.Context(), s.pool, audit.Event{
		ActorSubject: authruntime.CurrentSubjectFromRequest(r),
		Action:       action,
		TargetType:   targetType,
		TargetID:     targetID,
		Summary:      summary,
		Details:      details,
	})
}

func (s *Service) Nomenclatures(w http.ResponseWriter, r *http.Request) {
	response := DocumentFiltersResponse{}

	load := func(domain string) ([]string, error) {
		rows, err := s.pool.Query(r.Context(), `
			select code
			from app_nomenclatures
			where domain = $1 and active = true
			order by sort_order, code
		`, domain)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		values := []string{}
		for rows.Next() {
			var value string
			if err := rows.Scan(&value); err != nil {
				return nil, err
			}
			values = append(values, value)
		}
		return values, rows.Err()
	}

	var err error
	if response.DocumentTypes, err = load("registratura_document_type"); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "registratura_nomenclatures_failed"})
		return
	}
	if response.Directions, err = load("registratura_direction"); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "registratura_nomenclatures_failed"})
		return
	}
	if response.Statuses, err = load("registratura_status"); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "registratura_nomenclatures_failed"})
		return
	}
	if response.Confidentialities, err = load("registratura_confidentiality"); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "registratura_nomenclatures_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) ListDocuments(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"registru_id":        {},
			"registry_number":    {},
			"subject":            {},
			"document_type":      {},
			"direction":          {},
			"status":             {},
			"correspondent":      {},
			"assigned_to":        {},
			"confidentiality":    {},
			"registered_at":      {},
			"registered_at_from": {},
			"registered_at_to":   {},
			"due_date":           {},
			"due_date_from":      {},
			"due_date_to":        {},
		},
		[]string{"registry_number", "subject", "document_type", "direction", "status", "correspondent", "assigned_to", "confidentiality", "registered_at"},
	)

	whereClause, args := buildDocumentFilters(query.Filters)

	var total int
	countSQL := "select count(*) from registratura_documents d " + whereClause
	if err := s.pool.QueryRow(r.Context(), countSQL, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "registratura_list_failed",
			"message": "Nu s-au putut incarca documentele.",
		})
		return
	}

	sortColumn := sortColumn(query.Sort)
	offset := (query.Page - 1) * query.PageSize
	args = append(args, query.PageSize, offset)

	querySQL := fmt.Sprintf(`
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
		order by %s %s, d.registry_number asc
		limit $%d offset $%d
	`, whereClause, sortColumn, strings.ToUpper(query.Direction), len(args)-1, len(args))

	rows, err := s.pool.Query(r.Context(), querySQL, args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "registratura_list_failed",
			"message": "Nu s-au putut incarca documentele.",
		})
		return
	}
	defer rows.Close()

	documents := make([]Document, 0, query.PageSize)
	for rows.Next() {
		var document Document
		var registruID sql.NullInt64
		if err := rows.Scan(
			&document.ID,
			&registruID,
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
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{
				"code":    "registratura_list_failed",
				"message": "Nu s-au putut incarca documentele.",
			})
			return
		}
		if registruID.Valid {
			value := registruID.Int64
			document.RegistruID = &value
		}
		documents = append(documents, document)
	}

	httpx.WritePage(w, http.StatusOK, documents, total, query.Page, query.PageSize)
}

func (s *Service) DocumentFilters(w http.ResponseWriter, r *http.Request) {
	s.Nomenclatures(w, r)
}

func (s *Service) CreateDocument(w http.ResponseWriter, r *http.Request) {
	var req CreateDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{
			"code":    "invalid_document_payload",
			"message": "Cererea nu este valida.",
		})
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

	if req.Subject == "" || req.DocumentType == "" || req.Direction == "" || req.Status == "" || req.Correspondent == "" || req.Confidentiality == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{
			"code":    "missing_document_fields",
			"message": "Campurile obligatorii lipsesc.",
		})
		return
	}

	ctx := r.Context()
	if !s.isNomenclatureAllowed(ctx, "registratura_direction", req.Direction) ||
		!s.isNomenclatureAllowed(ctx, "registratura_status", req.Status) ||
		!s.isNomenclatureAllowed(ctx, "registratura_confidentiality", req.Confidentiality) ||
		!s.isNomenclatureAllowed(ctx, "registratura_document_type", req.DocumentType) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{
			"code":    "invalid_document_fields",
			"message": "Valorile selectate nu sunt permise.",
		})
		return
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_create_failed"})
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	document, err := s.createDocumentTx(ctx, tx, req, authruntime.CurrentSubjectFromRequest(r))
	if err != nil {
		if err == pgx.ErrNoRows {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "registry_not_found"})
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_create_failed"})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_create_failed"})
		return
	}

	s.logAudit(r, "registratura.documents.create", "document", document.ID, "Registratura document created.", map[string]any{
		"registry_number": document.RegistryNumber,
		"document_type":   document.DocumentType,
		"direction":       document.Direction,
		"status":          document.Status,
		"confidentiality": document.Confidentiality,
		"assigned_to":     document.AssignedTo,
		"correspondent":   document.Correspondent,
		"institution_id":  document.InstitutionID,
	})

	httpx.JSON(w, http.StatusCreated, document)
}

func (s *Service) UpdateDocument(w http.ResponseWriter, r *http.Request) {
	documentID := strings.TrimSpace(chi.URLParam(r, "documentID"))
	if documentID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_document_id"})
		return
	}

	var req UpdateDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_document_payload"})
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
	req.ChangeNotes = strings.TrimSpace(req.ChangeNotes)

	if req.Subject == "" || req.DocumentType == "" || req.Direction == "" || req.Status == "" || req.Correspondent == "" || req.Confidentiality == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_document_fields"})
		return
	}

	if !s.isNomenclatureAllowed(r.Context(), "registratura_direction", req.Direction) ||
		!s.isNomenclatureAllowed(r.Context(), "registratura_status", req.Status) ||
		!s.isNomenclatureAllowed(r.Context(), "registratura_confidentiality", req.Confidentiality) ||
		!s.isNomenclatureAllowed(r.Context(), "registratura_document_type", req.DocumentType) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_document_fields"})
		return
	}

	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_update_failed"})
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck

	current, err := s.loadDocumentTx(r.Context(), tx, documentID)
	if err != nil {
		if err == pgx.ErrNoRows {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "document_not_found"})
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_update_failed"})
		return
	}

	registruID := current.RegistruID
	if req.RegistruID != nil {
		registruID = req.RegistruID
	}
	resolvedRegistruID, err := s.resolveRegistryID(r.Context(), tx, registruID)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "registry_not_found"})
		return
	}

	var dueDate any
	if req.DueDate != nil && strings.TrimSpace(*req.DueDate) != "" {
		dueDate = strings.TrimSpace(*req.DueDate)
	}

	if _, err := tx.Exec(r.Context(), `
		update registratura_documents
		set registru_id = $1,
			subject = $2,
			document_type = $3,
			direction = $4,
			status = $5,
			correspondent = $6,
			assigned_to = $7,
			confidentiality = $8,
			summary = $9,
			due_date = $10,
			updated_at = now()
		where id::text = $11
	`,
		resolvedRegistruID,
		req.Subject,
		req.DocumentType,
		req.Direction,
		req.Status,
		req.Correspondent,
		req.AssignedTo,
		req.Confidentiality,
		req.Summary,
		dueDate,
		documentID,
	); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_update_failed"})
		return
	}

	if _, err := tx.Exec(r.Context(), `
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
		) values (
			$1::uuid,
			(select coalesce(max(version_no), 0) + 1 from registratura_document_versions where document_id = $1::uuid),
			$2, $3, $4, $5, $6, $7, $8, $9, $10::date, $11, $12
		)
	`,
		documentID,
		req.Subject,
		req.DocumentType,
		req.Direction,
		req.Status,
		req.Correspondent,
		req.AssignedTo,
		req.Confidentiality,
		req.Summary,
		dueDate,
		func() string {
			if req.ChangeNotes != "" {
				return req.ChangeNotes
			}
			return "Actualizare document"
		}(),
		authruntime.CurrentSubjectFromRequest(r),
	); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_version_init_failed"})
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_update_failed"})
		return
	}

	updated, err := s.loadDocument(r.Context(), documentID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_update_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, updated)
}

func (s *Service) CancelDocument(w http.ResponseWriter, r *http.Request) {
	documentID := strings.TrimSpace(chi.URLParam(r, "documentID"))
	if documentID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_document_id"})
		return
	}

	var req CancelDocumentRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	req.Reason = strings.TrimSpace(req.Reason)
	if req.Reason == "" {
		req.Reason = "Anulare document"
	}

	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_cancel_failed"})
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck

	var exists bool
	if err := tx.QueryRow(r.Context(), `select exists(select 1 from registratura_documents where id::text = $1)`, documentID).Scan(&exists); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_cancel_failed"})
		return
	}
	if !exists {
		httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "document_not_found"})
		return
	}

	if _, err := tx.Exec(r.Context(), `
		update registratura_documents
		set status = 'archived',
			summary = case when summary = '' then $1 else summary || ' | ' || $1 end,
			updated_at = now()
		where id::text = $2
	`, req.Reason, documentID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_cancel_failed"})
		return
	}

	if _, err := tx.Exec(r.Context(), `
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
			change_notes,
			created_by
		)
		select
			d.id,
			(select coalesce(max(version_no), 0) + 1 from registratura_document_versions where document_id = d.id),
			d.subject,
			d.document_type,
			d.direction,
			'archived',
			d.correspondent,
			d.assigned_to,
			d.confidentiality,
			d.summary,
			$1,
			$2
		from registratura_documents d
		where d.id::text = $3
	`, req.Reason, authruntime.CurrentSubjectFromRequest(r), documentID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_version_init_failed"})
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_cancel_failed"})
		return
	}

	updated, err := s.loadDocument(r.Context(), documentID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_cancel_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, updated)
}

func (s *Service) GetDocument(w http.ResponseWriter, r *http.Request) {
	documentID := strings.TrimSpace(chi.URLParam(r, "documentID"))
	if documentID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_document_id"})
		return
	}

	document, err := s.loadDocument(r.Context(), documentID)
	if err != nil {
		if err == pgx.ErrNoRows {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "document_not_found"})
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_load_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, document)
}

func (s *Service) ListDocumentVersions(w http.ResponseWriter, r *http.Request) {
	documentID := strings.TrimSpace(chi.URLParam(r, "documentID"))
	if documentID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_document_id"})
		return
	}

	rows, err := s.pool.Query(r.Context(), `
		select
			id::text,
			document_id::text,
			version_no,
			subject,
			document_type,
			direction,
			status,
			correspondent,
			assigned_to,
			confidentiality,
			summary,
			case when due_date is null then null else to_char(due_date, 'YYYY-MM-DD') end as due_date,
			change_notes,
			created_by,
			to_char(created_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') as created_at
		from registratura_document_versions
		where document_id::text = $1
		order by version_no desc
	`, documentID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_versions_failed"})
		return
	}
	defer rows.Close()

	items := []DocumentVersion{}
	for rows.Next() {
		var item DocumentVersion
		if err := rows.Scan(
			&item.ID,
			&item.DocumentID,
			&item.VersionNo,
			&item.Subject,
			&item.DocumentType,
			&item.Direction,
			&item.Status,
			&item.Correspondent,
			&item.AssignedTo,
			&item.Confidentiality,
			&item.Summary,
			&item.DueDate,
			&item.ChangeNotes,
			&item.CreatedBy,
			&item.CreatedAt,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_versions_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_versions_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, items)
}

func (s *Service) CreateDocumentVersion(w http.ResponseWriter, r *http.Request) {
	documentID := strings.TrimSpace(chi.URLParam(r, "documentID"))
	if documentID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_document_id"})
		return
	}

	var req CreateDocumentVersionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_document_version_payload"})
		return
	}

	req.Subject = strings.TrimSpace(req.Subject)
	req.Status = strings.TrimSpace(req.Status)
	req.AssignedTo = strings.TrimSpace(req.AssignedTo)
	req.Confidentiality = strings.TrimSpace(req.Confidentiality)
	req.Summary = strings.TrimSpace(req.Summary)
	req.ChangeNotes = strings.TrimSpace(req.ChangeNotes)

	if req.Subject == "" || req.Status == "" || req.Confidentiality == "" || req.ChangeNotes == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_document_version_fields"})
		return
	}
	if !s.isNomenclatureAllowed(r.Context(), "registratura_status", req.Status) ||
		!s.isNomenclatureAllowed(r.Context(), "registratura_confidentiality", req.Confidentiality) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_document_version_fields"})
		return
	}

	ctx := r.Context()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_version_create_failed"})
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var current Document
	if err := tx.QueryRow(ctx, `
		select
			id::text,
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
			to_char(registered_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') as registered_at,
			case when due_date is null then null else to_char(due_date, 'YYYY-MM-DD') end as due_date
		from registratura_documents
		where id::text = $1
		for update
	`, documentID).Scan(
		&current.ID,
		&current.RegistryNumber,
		&current.Subject,
		&current.DocumentType,
		&current.Direction,
		&current.Status,
		&current.Correspondent,
		&current.AssignedTo,
		&current.InstitutionID,
		&current.Confidentiality,
		&current.Summary,
		&current.RegisteredAt,
		&current.DueDate,
	); err != nil {
		if err == pgx.ErrNoRows {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "document_not_found"})
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_version_create_failed"})
		return
	}

	var nextVersionNo int
	if err := tx.QueryRow(ctx, `
		select coalesce(max(version_no), 0) + 1
		from registratura_document_versions
		where document_id::text = $1
	`, documentID).Scan(&nextVersionNo); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_version_create_failed"})
		return
	}

	var dueDate any
	if req.DueDate != nil && strings.TrimSpace(*req.DueDate) != "" {
		dueDate = strings.TrimSpace(*req.DueDate)
	}

	if _, err := tx.Exec(ctx, `
		update registratura_documents
		set
			subject = $2,
			status = $3,
			assigned_to = $4,
			confidentiality = $5,
			summary = $6,
			due_date = $7::date,
			updated_at = now()
		where id::text = $1
	`,
		documentID,
		req.Subject,
		req.Status,
		req.AssignedTo,
		req.Confidentiality,
		req.Summary,
		dueDate,
	); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_version_create_failed"})
		return
	}

	var version DocumentVersion
	if err := tx.QueryRow(ctx, `
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
		) values ($1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::date, $12, $13)
		returning
			id::text,
			document_id::text,
			version_no,
			subject,
			document_type,
			direction,
			status,
			correspondent,
			assigned_to,
			confidentiality,
			summary,
			case when due_date is null then null else to_char(due_date, 'YYYY-MM-DD') end as due_date,
			change_notes,
			created_by,
			to_char(created_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') as created_at
	`,
		documentID,
		nextVersionNo,
		req.Subject,
		current.DocumentType,
		current.Direction,
		req.Status,
		current.Correspondent,
		req.AssignedTo,
		req.Confidentiality,
		req.Summary,
		dueDate,
		req.ChangeNotes,
		authruntime.CurrentSubjectFromRequest(r),
	).Scan(
		&version.ID,
		&version.DocumentID,
		&version.VersionNo,
		&version.Subject,
		&version.DocumentType,
		&version.Direction,
		&version.Status,
		&version.Correspondent,
		&version.AssignedTo,
		&version.Confidentiality,
		&version.Summary,
		&version.DueDate,
		&version.ChangeNotes,
		&version.CreatedBy,
		&version.CreatedAt,
	); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_version_create_failed"})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_version_create_failed"})
		return
	}

	s.logAudit(r, "registratura.documents.version.create", "document_version", version.ID, "Registratura document version created.", map[string]any{
		"document_id":     version.DocumentID,
		"version_no":      version.VersionNo,
		"status":          version.Status,
		"confidentiality": version.Confidentiality,
		"assigned_to":     version.AssignedTo,
		"change_notes":    version.ChangeNotes,
		"registry_number": current.RegistryNumber,
	})

	httpx.JSON(w, http.StatusCreated, version)
}

func (s *Service) ListDocumentAttachments(w http.ResponseWriter, r *http.Request) {
	documentID := strings.TrimSpace(chi.URLParam(r, "documentID"))
	if documentID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_document_id"})
		return
	}

	rows, err := s.pool.Query(r.Context(), `
		select
			id::text,
			document_id::text,
			title,
			file_name,
			mime_type,
			storage_key,
			size_bytes,
			category,
			status,
			uploaded_by,
			to_char(uploaded_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') as uploaded_at
		from registratura_document_attachments
		where document_id::text = $1
		order by uploaded_at desc, title asc
	`, documentID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_attachments_failed"})
		return
	}
	defer rows.Close()

	items := []DocumentAttachment{}
	for rows.Next() {
		var item DocumentAttachment
		if err := rows.Scan(
			&item.ID,
			&item.DocumentID,
			&item.Title,
			&item.FileName,
			&item.MimeType,
			&item.StorageKey,
			&item.SizeBytes,
			&item.Category,
			&item.Status,
			&item.UploadedBy,
			&item.UploadedAt,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_attachments_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_attachments_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, items)
}

func (s *Service) CreateDocumentAttachment(w http.ResponseWriter, r *http.Request) {
	documentID := strings.TrimSpace(chi.URLParam(r, "documentID"))
	if documentID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_document_id"})
		return
	}

	var req CreateDocumentAttachmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_document_attachment_payload"})
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	req.FileName = strings.TrimSpace(req.FileName)
	req.MimeType = strings.TrimSpace(req.MimeType)
	req.StorageKey = strings.TrimSpace(req.StorageKey)
	req.Category = strings.TrimSpace(req.Category)
	req.Status = strings.TrimSpace(req.Status)
	req.UploadedBy = strings.TrimSpace(req.UploadedBy)

	if req.Title == "" || req.FileName == "" || req.MimeType == "" || req.StorageKey == "" || req.Category == "" || req.Status == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_document_attachment_fields"})
		return
	}
	if req.SizeBytes < 0 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_document_attachment_size"})
		return
	}

	var item DocumentAttachment
	err := s.pool.QueryRow(r.Context(), `
		insert into registratura_document_attachments (
			document_id,
			title,
			file_name,
			mime_type,
			storage_key,
			size_bytes,
			category,
			status,
			uploaded_by
		) values ($1::uuid, $2, $3, $4, $5, $6, $7, $8, $9)
		returning
			id::text,
			document_id::text,
			title,
			file_name,
			mime_type,
			storage_key,
			size_bytes,
			category,
			status,
			uploaded_by,
			to_char(uploaded_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') as uploaded_at
	`,
		documentID,
		req.Title,
		req.FileName,
		req.MimeType,
		req.StorageKey,
		req.SizeBytes,
		req.Category,
		req.Status,
		req.UploadedBy,
	).Scan(
		&item.ID,
		&item.DocumentID,
		&item.Title,
		&item.FileName,
		&item.MimeType,
		&item.StorageKey,
		&item.SizeBytes,
		&item.Category,
		&item.Status,
		&item.UploadedBy,
		&item.UploadedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "document_not_found"})
			return
		}
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "document_attachment_create_failed"})
		return
	}

	s.logAudit(r, "registratura.documents.attachment.create", "document_attachment", item.ID, "Registratura document attachment created.", map[string]any{
		"document_id": documentID,
		"title":       item.Title,
		"file_name":   item.FileName,
		"category":    item.Category,
		"status":      item.Status,
		"size_bytes":  item.SizeBytes,
	})

	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) LookupDocuments(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("query"))
	args := []any{"inst-001"}
	whereClause := "where d.institution_id = $1"
	if search != "" {
		args = append(args, "%"+strings.ToLower(search)+"%")
		whereClause += fmt.Sprintf(" and (lower(d.registry_number) like $%d or lower(d.subject) like $%d or lower(d.correspondent) like $%d)", len(args), len(args), len(args))
	}

	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select
			d.id::text,
			d.registry_number,
			d.subject,
			d.document_type,
			d.status
		from registratura_documents d
		%s
		order by d.registered_at desc
		limit 20
	`, whereClause), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_lookup_failed"})
		return
	}
	defer rows.Close()

	items := []DocumentLookupItem{}
	for rows.Next() {
		var item DocumentLookupItem
		if err := rows.Scan(&item.ID, &item.RegistryNumber, &item.Subject, &item.DocumentType, &item.Status); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_lookup_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_lookup_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, items)
}

func (s *Service) ListDocumentLinks(w http.ResponseWriter, r *http.Request) {
	sourceModule := strings.TrimSpace(r.URL.Query().Get("source_module"))
	sourceRecordID := strings.TrimSpace(r.URL.Query().Get("source_record_id"))
	if sourceModule == "" || sourceRecordID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_document_link_source"})
		return
	}

	rows, err := s.pool.Query(r.Context(), `
		select
			ld.id::text,
			d.id::text,
			d.registry_number,
			d.subject,
			d.document_type,
			d.status,
			ld.relation_type,
			to_char(d.registered_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			d.confidentiality
		from registratura_document_links ld
		join registratura_documents d on d.id = ld.document_id
		where ld.source_module = $1
			and ld.source_record_id::text = $2
		order by d.registered_at desc
	`, sourceModule, sourceRecordID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_links_failed"})
		return
	}
	defer rows.Close()

	items := []LinkedDocument{}
	for rows.Next() {
		var item LinkedDocument
		if err := rows.Scan(
			&item.LinkID,
			&item.DocumentID,
			&item.RegistryNumber,
			&item.Subject,
			&item.DocumentType,
			&item.Status,
			&item.RelationType,
			&item.RegisteredAt,
			&item.Confidentiality,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_links_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_links_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, items)
}

func (s *Service) CreateDocumentLink(w http.ResponseWriter, r *http.Request) {
	var req CreateDocumentLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_document_link_payload"})
		return
	}

	req.DocumentID = strings.TrimSpace(req.DocumentID)
	req.SourceModule = strings.TrimSpace(req.SourceModule)
	req.SourceRecordID = strings.TrimSpace(req.SourceRecordID)
	req.RelationType = strings.TrimSpace(req.RelationType)

	if req.DocumentID == "" || req.SourceModule == "" || req.SourceRecordID == "" || req.RelationType == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_document_link_fields"})
		return
	}
	if !contains([]string{"primary", "supporting", "decision", "archive_basis", "gdpr_basis"}, req.RelationType) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_document_link_relation"})
		return
	}

	var item LinkedDocument
	err := s.pool.QueryRow(r.Context(), `
		with inserted as (
			insert into registratura_document_links (document_id, source_module, source_record_id, relation_type)
			values ($1::uuid, $2, $3::uuid, $4)
			on conflict (document_id, source_module, source_record_id, relation_type) do update
			set relation_type = excluded.relation_type
			returning id::text, document_id, relation_type
		)
		select
			inserted.id,
			d.id::text,
			d.registry_number,
			d.subject,
			d.document_type,
			d.status,
			inserted.relation_type,
			to_char(d.registered_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			d.confidentiality
		from inserted
		join registratura_documents d on d.id = inserted.document_id
	`,
		req.DocumentID,
		req.SourceModule,
		req.SourceRecordID,
		req.RelationType,
	).Scan(
		&item.LinkID,
		&item.DocumentID,
		&item.RegistryNumber,
		&item.Subject,
		&item.DocumentType,
		&item.Status,
		&item.RelationType,
		&item.RegisteredAt,
		&item.Confidentiality,
	)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "document_link_create_failed"})
		return
	}

	s.logAudit(r, "registratura.document_links.create", "document_link", item.LinkID, "Registratura document link created.", map[string]any{
		"document_id":      item.DocumentID,
		"registry_number":  item.RegistryNumber,
		"source_module":    req.SourceModule,
		"source_record_id": req.SourceRecordID,
		"relation_type":    item.RelationType,
	})

	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) DeleteDocumentLink(w http.ResponseWriter, r *http.Request) {
	linkID := strings.TrimSpace(chi.URLParam(r, "linkID"))
	if linkID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_document_link_id"})
		return
	}

	result, err := s.pool.Exec(r.Context(), `delete from registratura_document_links where id::text = $1`, linkID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_link_delete_failed"})
		return
	}
	if result.RowsAffected() == 0 {
		httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "document_link_not_found"})
		return
	}

	s.logAudit(r, "registratura.document_links.delete", "document_link", linkID, "Registratura document link deleted.", nil)

	w.WriteHeader(http.StatusNoContent)
}

func buildDocumentFilters(filters map[string]string) (string, []any) {
	clauses := []string{}
	args := []any{}

	addContains := func(column string, value string) {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
	}
	addEqual := func(column string, value string) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	if value := filters["registry_number"]; value != "" {
		addContains("d.registry_number", value)
	}
	if value := filters["registru_id"]; value != "" {
		addEqual("d.registru_id::text", value)
	}
	if value := filters["subject"]; value != "" {
		addContains("d.subject", value)
	}
	if value := filters["document_type"]; value != "" {
		addEqual("d.document_type", value)
	}
	if value := filters["direction"]; value != "" {
		addEqual("d.direction", value)
	}
	if value := filters["status"]; value != "" {
		addEqual("d.status", value)
	}
	if value := filters["correspondent"]; value != "" {
		addContains("d.correspondent", value)
	}
	if value := filters["assigned_to"]; value != "" {
		addContains("d.assigned_to", value)
	}
	if value := filters["confidentiality"]; value != "" {
		addEqual("d.confidentiality", value)
	}
	if value := filters["registered_on"]; value != "" {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("d.registered_at::date = $%d::date", len(args)))
	}
	if value := filters["registered_at_from"]; value != "" {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("d.registered_at::date >= $%d::date", len(args)))
	}
	if value := filters["registered_at_to"]; value != "" {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("d.registered_at::date <= $%d::date", len(args)))
	}
	if value := filters["due_date_from"]; value != "" {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("d.due_date >= $%d::date", len(args)))
	}
	if value := filters["due_date_to"]; value != "" {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("d.due_date <= $%d::date", len(args)))
	}

	if len(clauses) == 0 {
		return "", args
	}
	return "where " + strings.Join(clauses, " and "), args
}

func sortColumn(field string) string {
	switch field {
	case "registry_number":
		return "d.registry_number"
	case "subject":
		return "d.subject"
	case "document_type":
		return "d.document_type"
	case "direction":
		return "d.direction"
	case "status":
		return "d.status"
	case "correspondent":
		return "d.correspondent"
	case "assigned_to":
		return "d.assigned_to"
	case "confidentiality":
		return "d.confidentiality"
	case "due_date":
		return "d.due_date"
	default:
		return "d.registered_at"
	}
}

func (s *Service) loadDocument(ctx context.Context, documentID string) (Document, error) {
	var document Document
	var registruID sql.NullInt64
	err := s.pool.QueryRow(ctx, `
		select
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
			to_char(registered_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') as registered_at,
			case when due_date is null then null else to_char(due_date, 'YYYY-MM-DD') end as due_date
		from registratura_documents
		where id::text = $1
	`, documentID).Scan(
		&document.ID,
		&registruID,
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
	if err == nil && registruID.Valid {
		value := registruID.Int64
		document.RegistruID = &value
	}
	return document, err
}

func (s *Service) loadDocumentTx(ctx context.Context, tx pgx.Tx, documentID string) (Document, error) {
	var document Document
	var registruID sql.NullInt64
	err := tx.QueryRow(ctx, `
		select
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
			to_char(registered_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') as registered_at,
			case when due_date is null then null else to_char(due_date, 'YYYY-MM-DD') end as due_date
		from registratura_documents
		where id::text = $1
	`, documentID).Scan(
		&document.ID,
		&registruID,
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
	if err == nil && registruID.Valid {
		value := registruID.Int64
		document.RegistruID = &value
	}
	return document, err
}

func contains(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func (s *Service) isNomenclatureAllowed(ctx context.Context, domain string, code string) bool {
	var exists bool
	if err := s.pool.QueryRow(ctx, `
		select exists(
			select 1
			from app_nomenclatures
			where domain = $1 and code = $2 and active = true
		)
	`, domain, code).Scan(&exists); err != nil {
		return false
	}
	return exists
}
