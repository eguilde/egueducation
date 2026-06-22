package education

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/eguilde/egueducation/internal/httpx"
)

func (s *Service) PersonnelPersonalFileDocuments(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"document_code":         {},
		"document_category":     {},
		"document_title":        {},
		"file_scope":            {},
		"confidentiality_level": {},
		"included_in_portfolio": {},
		"sensitive_data":        {},
	}, []string{"document_code", "document_category", "document_title", "file_scope", "confidentiality_level", "issued_on", "expires_on"})
	if query.Sort == "" {
		query.Sort = "issued_on"
	}

	whereClause, args := buildPersonnelDocumentFilters(query.Filters, recordID, s.institutionID(r))

	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_personnel_file_documents epfd "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_file_documents_failed"})
		return
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select
			epfd.id::text,
			epfd.personnel_id::text,
			epfd.document_code,
			epfd.document_category,
			epfd.document_title,
			epfd.file_scope,
			epfd.confidentiality_level,
			to_char(epfd.issued_on, 'YYYY-MM-DD'),
			coalesce(to_char(epfd.expires_on, 'YYYY-MM-DD'), ''),
			epfd.file_reference,
			epfd.sensitive_data,
			epfd.included_in_portfolio,
			epfd.institution_id,
			epfd.notes
		from education_personnel_file_documents epfd
		%s
		order by %s %s, epfd.document_code
		limit $%d offset $%d
	`, whereClause, personnelDocumentSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_file_documents_failed"})
		return
	}
	defer rows.Close()

	items := make([]PersonnelPersonalFileDocument, 0, query.PageSize)
	for rows.Next() {
		var item PersonnelPersonalFileDocument
		if err := rows.Scan(
			&item.ID,
			&item.PersonnelID,
			&item.DocumentCode,
			&item.DocumentCategory,
			&item.DocumentTitle,
			&item.FileScope,
			&item.ConfidentialityLevel,
			&item.IssuedOn,
			&item.ExpiresOn,
			&item.FileReference,
			&item.SensitiveData,
			&item.IncludedInPortfolio,
			&item.InstitutionID,
			&item.Notes,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_file_documents_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_file_documents_scan_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) PersonnelPersonalFileDocumentDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	documentID := strings.TrimSpace(chi.URLParam(r, "documentID"))
	var item PersonnelPersonalFileDocument
	err := s.pool.QueryRow(r.Context(), `
		select
			id::text,
			personnel_id::text,
			document_code,
			document_category,
			document_title,
			file_scope,
			confidentiality_level,
			to_char(issued_on, 'YYYY-MM-DD'),
			coalesce(to_char(expires_on, 'YYYY-MM-DD'), ''),
			file_reference,
			sensitive_data,
			included_in_portfolio,
			institution_id,
			notes
		from education_personnel_file_documents
		where id = $1 and personnel_id = $2 and institution_id = $3
	`, documentID, recordID, s.institutionID(r)).Scan(
		&item.ID,
		&item.PersonnelID,
		&item.DocumentCode,
		&item.DocumentCategory,
		&item.DocumentTitle,
		&item.FileScope,
		&item.ConfidentialityLevel,
		&item.IssuedOn,
		&item.ExpiresOn,
		&item.FileReference,
		&item.SensitiveData,
		&item.IncludedInPortfolio,
		&item.InstitutionID,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_personnel_file_document_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_file_document_detail_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreatePersonnelPersonalFileDocument(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreatePersonnelPersonalFileDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_file_document_payload"})
		return
	}
	normalizePersonnelDocumentRequest(&req)
	if req.DocumentCategory == "" || req.DocumentTitle == "" || req.FileScope == "" || req.ConfidentialityLevel == "" || req.IssuedOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_personnel_file_document_fields"})
		return
	}
	if !inStringSet(req.DocumentCategory, "identificare", "studii", "cariera", "evaluare", "declaratie", "medical", "disciplina", "management") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_file_document_category"})
		return
	}
	if !inStringSet(req.FileScope, "dosar_personal", "dosar_director", "dosar_director_adjunct") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_file_scope"})
		return
	}
	if !inStringSet(req.ConfidentialityLevel, "intern", "confidential", "strict_confidential") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_file_confidentiality"})
		return
	}
	issuedOn, err := parseRequiredEducationDate(req.IssuedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_file_issued_on"})
		return
	}
	expiresOn, err := parseOptionalEducationDate(req.ExpiresOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_file_expires_on"})
		return
	}
	if expiresOn != nil && expiresOn.Before(*issuedOn) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_file_interval"})
		return
	}

	documentCode := fmt.Sprintf("PFD-%d-%05d", time.Now().UTC().Year(), time.Now().UTC().UnixNano()%100000)
	var item PersonnelPersonalFileDocument
	err = s.pool.QueryRow(r.Context(), `
		insert into education_personnel_file_documents (
			personnel_id,
			document_code,
			document_category,
			document_title,
			file_scope,
			confidentiality_level,
			issued_on,
			expires_on,
			file_reference,
			sensitive_data,
			included_in_portfolio,
			institution_id,
			notes
		) values (
			$1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)
		returning
			id::text,
			personnel_id::text,
			document_code,
			document_category,
			document_title,
			file_scope,
			confidentiality_level,
			to_char(issued_on, 'YYYY-MM-DD'),
			coalesce(to_char(expires_on, 'YYYY-MM-DD'), ''),
			file_reference,
			sensitive_data,
			included_in_portfolio,
			institution_id,
			notes
	`, recordID, documentCode, req.DocumentCategory, req.DocumentTitle, req.FileScope, req.ConfidentialityLevel, issuedOn, expiresOn, req.FileReference, req.SensitiveData, req.IncludedInPortfolio, s.institutionID(r), req.Notes).Scan(
		&item.ID,
		&item.PersonnelID,
		&item.DocumentCode,
		&item.DocumentCategory,
		&item.DocumentTitle,
		&item.FileScope,
		&item.ConfidentialityLevel,
		&item.IssuedOn,
		&item.ExpiresOn,
		&item.FileReference,
		&item.SensitiveData,
		&item.IncludedInPortfolio,
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "personnel_file_document_create_failed"})
		return
	}

	s.logAudit(r, "education.personnel.file_document.create", "personnel_file_document", item.ID, "Personnel file document created.", map[string]any{
		"personnel_id":          recordID,
		"document_code":         item.DocumentCode,
		"document_category":     item.DocumentCategory,
		"file_scope":            item.FileScope,
		"confidentiality_level": item.ConfidentialityLevel,
	})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdatePersonnelPersonalFileDocument(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	documentID := strings.TrimSpace(chi.URLParam(r, "documentID"))
	var req CreatePersonnelPersonalFileDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_file_document_payload"})
		return
	}
	normalizePersonnelDocumentRequest(&req)
	if req.DocumentCategory == "" || req.DocumentTitle == "" || req.FileScope == "" || req.ConfidentialityLevel == "" || req.IssuedOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_personnel_file_document_fields"})
		return
	}
	if !inStringSet(req.DocumentCategory, "identificare", "studii", "cariera", "evaluare", "declaratie", "medical", "disciplina", "management") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_file_document_category"})
		return
	}
	if !inStringSet(req.FileScope, "dosar_personal", "dosar_director", "dosar_director_adjunct") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_file_scope"})
		return
	}
	if !inStringSet(req.ConfidentialityLevel, "intern", "confidential", "strict_confidential") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_file_confidentiality"})
		return
	}
	issuedOn, err := parseRequiredEducationDate(req.IssuedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_file_issued_on"})
		return
	}
	expiresOn, err := parseOptionalEducationDate(req.ExpiresOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_file_expires_on"})
		return
	}
	if expiresOn != nil && expiresOn.Before(*issuedOn) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_file_interval"})
		return
	}

	var item PersonnelPersonalFileDocument
	err = s.pool.QueryRow(r.Context(), `
		update education_personnel_file_documents
		set
			document_category = $1,
			document_title = $2,
			file_scope = $3,
			confidentiality_level = $4,
			issued_on = $5,
			expires_on = $6,
			file_reference = $7,
			sensitive_data = $8,
			included_in_portfolio = $9,
			notes = $10,
			updated_at = now()
		where id = $11 and personnel_id = $12 and institution_id = $13
		returning
			id::text,
			personnel_id::text,
			document_code,
			document_category,
			document_title,
			file_scope,
			confidentiality_level,
			to_char(issued_on, 'YYYY-MM-DD'),
			coalesce(to_char(expires_on, 'YYYY-MM-DD'), ''),
			file_reference,
			sensitive_data,
			included_in_portfolio,
			institution_id,
			notes
	`, req.DocumentCategory, req.DocumentTitle, req.FileScope, req.ConfidentialityLevel, issuedOn, expiresOn, req.FileReference, req.SensitiveData, req.IncludedInPortfolio, req.Notes, documentID, recordID, s.institutionID(r)).Scan(
		&item.ID,
		&item.PersonnelID,
		&item.DocumentCode,
		&item.DocumentCategory,
		&item.DocumentTitle,
		&item.FileScope,
		&item.ConfidentialityLevel,
		&item.IssuedOn,
		&item.ExpiresOn,
		&item.FileReference,
		&item.SensitiveData,
		&item.IncludedInPortfolio,
		&item.InstitutionID,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_personnel_file_document_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "personnel_file_document_update_failed"})
		return
	}

	s.logAudit(r, "education.personnel.file_document.update", "personnel_file_document", item.ID, "Personnel file document updated.", map[string]any{
		"personnel_id":          recordID,
		"document_code":         item.DocumentCode,
		"document_category":     item.DocumentCategory,
		"file_scope":            item.FileScope,
		"confidentiality_level": item.ConfidentialityLevel,
	})
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeletePersonnelPersonalFileDocument(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	documentID := strings.TrimSpace(chi.URLParam(r, "documentID"))
	tag, err := s.pool.Exec(r.Context(), `delete from education_personnel_file_documents where id = $1 and personnel_id = $2 and institution_id = $3`, documentID, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "personnel_file_document_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_personnel_file_document_not_found")
		return
	}

	s.logAudit(r, "education.personnel.file_document.delete", "personnel_file_document", documentID, "Personnel file document deleted.", map[string]any{"personnel_id": recordID})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) PersonnelPersonalAccessEvents(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"event_type":      {},
		"actor_name":      {},
		"actor_role":      {},
		"access_channel":  {},
		"sensitive_scope": {},
	}, []string{"event_type", "actor_name", "actor_role", "access_channel", "accessed_on", "closed_on"})
	if query.Sort == "" {
		query.Sort = "accessed_on"
	}

	whereClause, args := buildPersonnelAccessFilters(query.Filters, recordID, s.institutionID(r))

	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_personnel_access_events epae "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_access_events_failed"})
		return
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select
			epae.id::text,
			epae.personnel_id::text,
			epae.event_type,
			epae.actor_name,
			epae.actor_role,
			epae.purpose,
			epae.access_channel,
			to_char(epae.accessed_on, 'YYYY-MM-DD'),
			coalesce(to_char(epae.closed_on, 'YYYY-MM-DD'), ''),
			epae.sensitive_scope,
			epae.institution_id,
			epae.notes
		from education_personnel_access_events epae
		%s
		order by %s %s, epae.accessed_on desc
		limit $%d offset $%d
	`, whereClause, personnelAccessSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_access_events_failed"})
		return
	}
	defer rows.Close()

	items := make([]PersonnelPersonalAccessEvent, 0, query.PageSize)
	for rows.Next() {
		var item PersonnelPersonalAccessEvent
		if err := rows.Scan(
			&item.ID,
			&item.PersonnelID,
			&item.EventType,
			&item.ActorName,
			&item.ActorRole,
			&item.Purpose,
			&item.AccessChannel,
			&item.AccessedOn,
			&item.ClosedOn,
			&item.SensitiveScope,
			&item.InstitutionID,
			&item.Notes,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_access_events_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_access_events_scan_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) PersonnelPersonalAccessEventDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	eventID := strings.TrimSpace(chi.URLParam(r, "eventID"))
	var item PersonnelPersonalAccessEvent
	err := s.pool.QueryRow(r.Context(), `
		select
			id::text,
			personnel_id::text,
			event_type,
			actor_name,
			actor_role,
			purpose,
			access_channel,
			to_char(accessed_on, 'YYYY-MM-DD'),
			coalesce(to_char(closed_on, 'YYYY-MM-DD'), ''),
			sensitive_scope,
			institution_id,
			notes
		from education_personnel_access_events
		where id = $1 and personnel_id = $2 and institution_id = $3
	`, eventID, recordID, s.institutionID(r)).Scan(
		&item.ID,
		&item.PersonnelID,
		&item.EventType,
		&item.ActorName,
		&item.ActorRole,
		&item.Purpose,
		&item.AccessChannel,
		&item.AccessedOn,
		&item.ClosedOn,
		&item.SensitiveScope,
		&item.InstitutionID,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_personnel_access_event_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_access_event_detail_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreatePersonnelPersonalAccessEvent(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreatePersonnelPersonalAccessEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_access_event_payload"})
		return
	}
	normalizePersonnelAccessRequest(&req)
	if req.EventType == "" || req.ActorName == "" || req.ActorRole == "" || req.Purpose == "" || req.AccessChannel == "" || req.AccessedOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_personnel_access_event_fields"})
		return
	}
	if !inStringSet(req.EventType, "consultare", "predare", "actualizare", "arhivare", "export") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_access_event_type"})
		return
	}
	if !inStringSet(req.AccessChannel, "fizic", "digital", "mixt") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_access_channel"})
		return
	}
	accessedOn, err := parseRequiredEducationDate(req.AccessedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_access_accessed_on"})
		return
	}
	closedOn, err := parseOptionalEducationDate(req.ClosedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_access_closed_on"})
		return
	}
	if closedOn != nil && closedOn.Before(*accessedOn) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_access_interval"})
		return
	}

	var item PersonnelPersonalAccessEvent
	err = s.pool.QueryRow(r.Context(), `
		insert into education_personnel_access_events (
			personnel_id,
			event_type,
			actor_name,
			actor_role,
			purpose,
			access_channel,
			accessed_on,
			closed_on,
			sensitive_scope,
			institution_id,
			notes
		) values (
			$1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
		returning
			id::text,
			personnel_id::text,
			event_type,
			actor_name,
			actor_role,
			purpose,
			access_channel,
			to_char(accessed_on, 'YYYY-MM-DD'),
			coalesce(to_char(closed_on, 'YYYY-MM-DD'), ''),
			sensitive_scope,
			institution_id,
			notes
	`, recordID, req.EventType, req.ActorName, req.ActorRole, req.Purpose, req.AccessChannel, accessedOn, closedOn, req.SensitiveScope, s.institutionID(r), req.Notes).Scan(
		&item.ID,
		&item.PersonnelID,
		&item.EventType,
		&item.ActorName,
		&item.ActorRole,
		&item.Purpose,
		&item.AccessChannel,
		&item.AccessedOn,
		&item.ClosedOn,
		&item.SensitiveScope,
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "personnel_access_event_create_failed"})
		return
	}

	s.logAudit(r, "education.personnel.access_event.create", "personnel_access_event", item.ID, "Personnel access event created.", map[string]any{
		"personnel_id":   recordID,
		"event_type":     item.EventType,
		"actor_name":     item.ActorName,
		"access_channel": item.AccessChannel,
		"sensitive":      item.SensitiveScope,
	})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdatePersonnelPersonalAccessEvent(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	eventID := strings.TrimSpace(chi.URLParam(r, "eventID"))
	var req CreatePersonnelPersonalAccessEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_access_event_payload"})
		return
	}
	normalizePersonnelAccessRequest(&req)
	if req.EventType == "" || req.ActorName == "" || req.ActorRole == "" || req.Purpose == "" || req.AccessChannel == "" || req.AccessedOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_personnel_access_event_fields"})
		return
	}
	if !inStringSet(req.EventType, "consultare", "predare", "actualizare", "arhivare", "export") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_access_event_type"})
		return
	}
	if !inStringSet(req.AccessChannel, "fizic", "digital", "mixt") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_access_channel"})
		return
	}
	accessedOn, err := parseRequiredEducationDate(req.AccessedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_access_accessed_on"})
		return
	}
	closedOn, err := parseOptionalEducationDate(req.ClosedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_access_closed_on"})
		return
	}
	if closedOn != nil && closedOn.Before(*accessedOn) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_access_interval"})
		return
	}

	var item PersonnelPersonalAccessEvent
	err = s.pool.QueryRow(r.Context(), `
		update education_personnel_access_events
		set
			event_type = $1,
			actor_name = $2,
			actor_role = $3,
			purpose = $4,
			access_channel = $5,
			accessed_on = $6,
			closed_on = $7,
			sensitive_scope = $8,
			notes = $9,
			updated_at = now()
		where id = $10 and personnel_id = $11 and institution_id = $12
		returning
			id::text,
			personnel_id::text,
			event_type,
			actor_name,
			actor_role,
			purpose,
			access_channel,
			to_char(accessed_on, 'YYYY-MM-DD'),
			coalesce(to_char(closed_on, 'YYYY-MM-DD'), ''),
			sensitive_scope,
			institution_id,
			notes
	`, req.EventType, req.ActorName, req.ActorRole, req.Purpose, req.AccessChannel, accessedOn, closedOn, req.SensitiveScope, req.Notes, eventID, recordID, s.institutionID(r)).Scan(
		&item.ID,
		&item.PersonnelID,
		&item.EventType,
		&item.ActorName,
		&item.ActorRole,
		&item.Purpose,
		&item.AccessChannel,
		&item.AccessedOn,
		&item.ClosedOn,
		&item.SensitiveScope,
		&item.InstitutionID,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_personnel_access_event_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "personnel_access_event_update_failed"})
		return
	}

	s.logAudit(r, "education.personnel.access_event.update", "personnel_access_event", item.ID, "Personnel access event updated.", map[string]any{
		"personnel_id":   recordID,
		"event_type":     item.EventType,
		"actor_name":     item.ActorName,
		"access_channel": item.AccessChannel,
		"sensitive":      item.SensitiveScope,
	})
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeletePersonnelPersonalAccessEvent(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	eventID := strings.TrimSpace(chi.URLParam(r, "eventID"))
	tag, err := s.pool.Exec(r.Context(), `delete from education_personnel_access_events where id = $1 and personnel_id = $2 and institution_id = $3`, eventID, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "personnel_access_event_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_personnel_access_event_not_found")
		return
	}

	s.logAudit(r, "education.personnel.access_event.delete", "personnel_access_event", eventID, "Personnel access event deleted.", map[string]any{"personnel_id": recordID})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) EvaluationAppeals(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"appeal_code":  {},
		"submitted_by": {},
		"status":       {},
	}, []string{"appeal_code", "submitted_by", "status", "submitted_on", "hearing_on", "resolved_on"})
	if query.Sort == "" {
		query.Sort = "submitted_on"
	}

	whereClause, args := buildEvaluationAppealFilters(query.Filters, recordID, s.institutionID(r))

	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_evaluation_appeals eea "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluation_appeals_failed"})
		return
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select
			eea.id::text,
			eea.evaluation_id::text,
			eea.appeal_code,
			eea.submitted_by,
			to_char(eea.submitted_on, 'YYYY-MM-DD'),
			eea.status,
			eea.grounds,
			coalesce(to_char(eea.hearing_on, 'YYYY-MM-DD'), ''),
			coalesce(to_char(eea.resolved_on, 'YYYY-MM-DD'), ''),
			eea.decision_summary,
			eea.committee_note,
			eea.attached_to_personnel_file,
			eea.institution_id
		from education_evaluation_appeals eea
		%s
		order by %s %s, eea.submitted_on desc
		limit $%d offset $%d
	`, whereClause, evaluationAppealSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluation_appeals_failed"})
		return
	}
	defer rows.Close()

	items := make([]PersonnelEvaluationAppeal, 0, query.PageSize)
	for rows.Next() {
		var item PersonnelEvaluationAppeal
		if err := rows.Scan(
			&item.ID,
			&item.EvaluationID,
			&item.AppealCode,
			&item.SubmittedBy,
			&item.SubmittedOn,
			&item.Status,
			&item.Grounds,
			&item.HearingOn,
			&item.ResolvedOn,
			&item.DecisionSummary,
			&item.CommitteeNote,
			&item.AttachedToPersonnelFile,
			&item.InstitutionID,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluation_appeals_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluation_appeals_scan_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) EvaluationAppealDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	appealID := strings.TrimSpace(chi.URLParam(r, "appealID"))
	var item PersonnelEvaluationAppeal
	err := s.pool.QueryRow(r.Context(), `
		select
			id::text,
			evaluation_id::text,
			appeal_code,
			submitted_by,
			to_char(submitted_on, 'YYYY-MM-DD'),
			status,
			grounds,
			coalesce(to_char(hearing_on, 'YYYY-MM-DD'), ''),
			coalesce(to_char(resolved_on, 'YYYY-MM-DD'), ''),
			decision_summary,
			committee_note,
			attached_to_personnel_file,
			institution_id
		from education_evaluation_appeals
		where id = $1 and evaluation_id = $2 and institution_id = $3
	`, appealID, recordID, s.institutionID(r)).Scan(
		&item.ID,
		&item.EvaluationID,
		&item.AppealCode,
		&item.SubmittedBy,
		&item.SubmittedOn,
		&item.Status,
		&item.Grounds,
		&item.HearingOn,
		&item.ResolvedOn,
		&item.DecisionSummary,
		&item.CommitteeNote,
		&item.AttachedToPersonnelFile,
		&item.InstitutionID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_evaluation_appeal_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluation_appeal_detail_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreateEvaluationAppeal(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreatePersonnelEvaluationAppealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_appeal_payload"})
		return
	}
	normalizeEvaluationAppealRequest(&req)
	if req.SubmittedBy == "" || req.SubmittedOn == "" || req.Status == "" || req.Grounds == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_evaluation_appeal_fields"})
		return
	}
	if !inStringSet(req.Status, "submitted", "review", "accepted", "rejected", "resolved") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_appeal_status"})
		return
	}
	submittedOn, err := parseRequiredEducationDate(req.SubmittedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_appeal_submitted_on"})
		return
	}
	hearingOn, err := parseOptionalEducationDate(req.HearingOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_appeal_hearing_on"})
		return
	}
	resolvedOn, err := parseOptionalEducationDate(req.ResolvedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_appeal_resolved_on"})
		return
	}
	if hearingOn != nil && hearingOn.Before(*submittedOn) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_appeal_hearing_interval"})
		return
	}
	if resolvedOn != nil && resolvedOn.Before(*submittedOn) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_appeal_resolved_interval"})
		return
	}
	if inStringSet(req.Status, "accepted", "rejected", "resolved") && resolvedOn == nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_evaluation_appeal_resolved_on"})
		return
	}
	if inStringSet(req.Status, "accepted", "rejected") && req.DecisionSummary == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_evaluation_appeal_decision_summary"})
		return
	}

	appealCode := fmt.Sprintf("APEL-%d-%05d", time.Now().UTC().Year(), time.Now().UTC().UnixNano()%100000)
	var item PersonnelEvaluationAppeal
	err = s.pool.QueryRow(r.Context(), `
		insert into education_evaluation_appeals (
			evaluation_id,
			appeal_code,
			submitted_by,
			submitted_on,
			status,
			grounds,
			hearing_on,
			resolved_on,
			decision_summary,
			committee_note,
			attached_to_personnel_file,
			institution_id
		) values (
			$1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
		returning
			id::text,
			evaluation_id::text,
			appeal_code,
			submitted_by,
			to_char(submitted_on, 'YYYY-MM-DD'),
			status,
			grounds,
			coalesce(to_char(hearing_on, 'YYYY-MM-DD'), ''),
			coalesce(to_char(resolved_on, 'YYYY-MM-DD'), ''),
			decision_summary,
			committee_note,
			attached_to_personnel_file,
			institution_id
	`, recordID, appealCode, req.SubmittedBy, submittedOn, req.Status, req.Grounds, hearingOn, resolvedOn, req.DecisionSummary, req.CommitteeNote, req.AttachedToPersonnelFile, s.institutionID(r)).Scan(
		&item.ID,
		&item.EvaluationID,
		&item.AppealCode,
		&item.SubmittedBy,
		&item.SubmittedOn,
		&item.Status,
		&item.Grounds,
		&item.HearingOn,
		&item.ResolvedOn,
		&item.DecisionSummary,
		&item.CommitteeNote,
		&item.AttachedToPersonnelFile,
		&item.InstitutionID,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_appeal_create_failed"})
		return
	}

	s.logAudit(r, "education.evaluations.appeal.create", "evaluation_appeal", item.ID, "Evaluation appeal created.", map[string]any{
		"evaluation_id":              recordID,
		"appeal_code":                item.AppealCode,
		"submitted_by":               item.SubmittedBy,
		"status":                     item.Status,
		"attached_to_personnel_file": item.AttachedToPersonnelFile,
	})
	if err := s.syncEvaluationAppealEffects(r.Context(), recordID, s.institutionID(r)); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_appeal_sync_failed"})
		return
	}
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdateEvaluationAppeal(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	appealID := strings.TrimSpace(chi.URLParam(r, "appealID"))
	var req CreatePersonnelEvaluationAppealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_appeal_payload"})
		return
	}
	normalizeEvaluationAppealRequest(&req)
	if req.SubmittedBy == "" || req.SubmittedOn == "" || req.Status == "" || req.Grounds == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_evaluation_appeal_fields"})
		return
	}
	if !inStringSet(req.Status, "submitted", "review", "accepted", "rejected", "resolved") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_appeal_status"})
		return
	}
	submittedOn, err := parseRequiredEducationDate(req.SubmittedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_appeal_submitted_on"})
		return
	}
	hearingOn, err := parseOptionalEducationDate(req.HearingOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_appeal_hearing_on"})
		return
	}
	resolvedOn, err := parseOptionalEducationDate(req.ResolvedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_appeal_resolved_on"})
		return
	}
	if hearingOn != nil && hearingOn.Before(*submittedOn) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_appeal_hearing_interval"})
		return
	}
	if resolvedOn != nil && resolvedOn.Before(*submittedOn) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_appeal_resolved_interval"})
		return
	}
	if inStringSet(req.Status, "accepted", "rejected", "resolved") && resolvedOn == nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_evaluation_appeal_resolved_on"})
		return
	}
	if inStringSet(req.Status, "accepted", "rejected") && req.DecisionSummary == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_evaluation_appeal_decision_summary"})
		return
	}

	var item PersonnelEvaluationAppeal
	err = s.pool.QueryRow(r.Context(), `
		update education_evaluation_appeals
		set
			submitted_by = $1,
			submitted_on = $2,
			status = $3,
			grounds = $4,
			hearing_on = $5,
			resolved_on = $6,
			decision_summary = $7,
			committee_note = $8,
			attached_to_personnel_file = $9,
			updated_at = now()
		where id = $10 and evaluation_id = $11 and institution_id = $12
		returning
			id::text,
			evaluation_id::text,
			appeal_code,
			submitted_by,
			to_char(submitted_on, 'YYYY-MM-DD'),
			status,
			grounds,
			coalesce(to_char(hearing_on, 'YYYY-MM-DD'), ''),
			coalesce(to_char(resolved_on, 'YYYY-MM-DD'), ''),
			decision_summary,
			committee_note,
			attached_to_personnel_file,
			institution_id
	`, req.SubmittedBy, submittedOn, req.Status, req.Grounds, hearingOn, resolvedOn, req.DecisionSummary, req.CommitteeNote, req.AttachedToPersonnelFile, appealID, recordID, s.institutionID(r)).Scan(
		&item.ID,
		&item.EvaluationID,
		&item.AppealCode,
		&item.SubmittedBy,
		&item.SubmittedOn,
		&item.Status,
		&item.Grounds,
		&item.HearingOn,
		&item.ResolvedOn,
		&item.DecisionSummary,
		&item.CommitteeNote,
		&item.AttachedToPersonnelFile,
		&item.InstitutionID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_evaluation_appeal_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_appeal_update_failed"})
		return
	}

	s.logAudit(r, "education.evaluations.appeal.update", "evaluation_appeal", item.ID, "Evaluation appeal updated.", map[string]any{
		"evaluation_id":              recordID,
		"appeal_code":                item.AppealCode,
		"submitted_by":               item.SubmittedBy,
		"status":                     item.Status,
		"attached_to_personnel_file": item.AttachedToPersonnelFile,
	})
	if err := s.syncEvaluationAppealEffects(r.Context(), recordID, s.institutionID(r)); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_appeal_sync_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteEvaluationAppeal(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	appealID := strings.TrimSpace(chi.URLParam(r, "appealID"))
	tag, err := s.pool.Exec(r.Context(), `delete from education_evaluation_appeals where id = $1 and evaluation_id = $2 and institution_id = $3`, appealID, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_appeal_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_evaluation_appeal_not_found")
		return
	}

	s.logAudit(r, "education.evaluations.appeal.delete", "evaluation_appeal", appealID, "Evaluation appeal deleted.", map[string]any{"evaluation_id": recordID})
	if err := s.syncEvaluationAppealEffects(r.Context(), recordID, s.institutionID(r)); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_appeal_sync_failed"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func buildPersonnelDocumentFilters(filters map[string]string, recordID string, institutionID string) (string, []any) {
	where := []string{"epfd.personnel_id = $1", "epfd.institution_id = $2"}
	args := []any{recordID, institutionID}
	for key, column := range map[string]string{
		"document_code":         "epfd.document_code",
		"document_category":     "epfd.document_category",
		"document_title":        "epfd.document_title",
		"file_scope":            "epfd.file_scope",
		"confidentiality_level": "epfd.confidentiality_level",
	} {
		if value := strings.TrimSpace(filters[key]); value != "" {
			args = append(args, "%"+strings.ToLower(value)+"%")
			where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
		}
	}
	for _, booleanField := range []struct {
		key    string
		column string
	}{
		{key: "included_in_portfolio", column: "epfd.included_in_portfolio"},
		{key: "sensitive_data", column: "epfd.sensitive_data"},
	} {
		if value := strings.TrimSpace(filters[booleanField.key]); value != "" {
			args = append(args, value == "true")
			where = append(where, fmt.Sprintf("%s = $%d", booleanField.column, len(args)))
		}
	}
	return "where " + strings.Join(where, " and "), args
}

func buildPersonnelAccessFilters(filters map[string]string, recordID string, institutionID string) (string, []any) {
	where := []string{"epae.personnel_id = $1", "epae.institution_id = $2"}
	args := []any{recordID, institutionID}
	for key, column := range map[string]string{
		"event_type":     "epae.event_type",
		"actor_name":     "epae.actor_name",
		"actor_role":     "epae.actor_role",
		"access_channel": "epae.access_channel",
	} {
		if value := strings.TrimSpace(filters[key]); value != "" {
			args = append(args, "%"+strings.ToLower(value)+"%")
			where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
		}
	}
	if value := strings.TrimSpace(filters["sensitive_scope"]); value != "" {
		args = append(args, value == "true")
		where = append(where, fmt.Sprintf("epae.sensitive_scope = $%d", len(args)))
	}
	return "where " + strings.Join(where, " and "), args
}

func buildEvaluationAppealFilters(filters map[string]string, recordID string, institutionID string) (string, []any) {
	where := []string{"eea.evaluation_id = $1", "eea.institution_id = $2"}
	args := []any{recordID, institutionID}
	for key, column := range map[string]string{
		"appeal_code":  "eea.appeal_code",
		"submitted_by": "eea.submitted_by",
		"status":       "eea.status",
	} {
		if value := strings.TrimSpace(filters[key]); value != "" {
			args = append(args, "%"+strings.ToLower(value)+"%")
			where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
		}
	}
	return "where " + strings.Join(where, " and "), args
}

func personnelDocumentSortColumn(value string) string {
	switch value {
	case "document_code":
		return "epfd.document_code"
	case "document_category":
		return "epfd.document_category"
	case "document_title":
		return "epfd.document_title"
	case "file_scope":
		return "epfd.file_scope"
	case "confidentiality_level":
		return "epfd.confidentiality_level"
	case "issued_on":
		return "epfd.issued_on"
	case "expires_on":
		return "epfd.expires_on"
	default:
		return "epfd.issued_on"
	}
}

func personnelAccessSortColumn(value string) string {
	switch value {
	case "event_type":
		return "epae.event_type"
	case "actor_name":
		return "epae.actor_name"
	case "actor_role":
		return "epae.actor_role"
	case "access_channel":
		return "epae.access_channel"
	case "accessed_on":
		return "epae.accessed_on"
	case "closed_on":
		return "epae.closed_on"
	default:
		return "epae.accessed_on"
	}
}

func evaluationAppealSortColumn(value string) string {
	switch value {
	case "appeal_code":
		return "eea.appeal_code"
	case "submitted_by":
		return "eea.submitted_by"
	case "status":
		return "eea.status"
	case "submitted_on":
		return "eea.submitted_on"
	case "hearing_on":
		return "eea.hearing_on"
	case "resolved_on":
		return "eea.resolved_on"
	default:
		return "eea.submitted_on"
	}
}

func normalizePersonnelDocumentRequest(req *CreatePersonnelPersonalFileDocumentRequest) {
	req.DocumentCategory = strings.TrimSpace(req.DocumentCategory)
	req.DocumentTitle = strings.TrimSpace(req.DocumentTitle)
	req.FileScope = strings.TrimSpace(req.FileScope)
	req.ConfidentialityLevel = strings.TrimSpace(req.ConfidentialityLevel)
	req.IssuedOn = strings.TrimSpace(req.IssuedOn)
	req.ExpiresOn = strings.TrimSpace(req.ExpiresOn)
	req.FileReference = strings.TrimSpace(req.FileReference)
	req.Notes = strings.TrimSpace(req.Notes)
}

func normalizePersonnelAccessRequest(req *CreatePersonnelPersonalAccessEventRequest) {
	req.EventType = strings.TrimSpace(req.EventType)
	req.ActorName = strings.TrimSpace(req.ActorName)
	req.ActorRole = strings.TrimSpace(req.ActorRole)
	req.Purpose = strings.TrimSpace(req.Purpose)
	req.AccessChannel = strings.TrimSpace(req.AccessChannel)
	req.AccessedOn = strings.TrimSpace(req.AccessedOn)
	req.ClosedOn = strings.TrimSpace(req.ClosedOn)
	req.Notes = strings.TrimSpace(req.Notes)
}

func normalizeEvaluationAppealRequest(req *CreatePersonnelEvaluationAppealRequest) {
	req.SubmittedBy = strings.TrimSpace(req.SubmittedBy)
	req.SubmittedOn = strings.TrimSpace(req.SubmittedOn)
	req.Status = strings.TrimSpace(req.Status)
	req.Grounds = strings.TrimSpace(req.Grounds)
	req.HearingOn = strings.TrimSpace(req.HearingOn)
	req.ResolvedOn = strings.TrimSpace(req.ResolvedOn)
	req.DecisionSummary = strings.TrimSpace(req.DecisionSummary)
	req.CommitteeNote = strings.TrimSpace(req.CommitteeNote)
}
