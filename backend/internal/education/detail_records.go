package education

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/eguilde/egueducation/internal/httpx"
)

func (s *Service) GovernanceMeetingParticipants(w http.ResponseWriter, r *http.Request) {
	meetingID := strings.TrimSpace(chi.URLParam(r, "meetingID"))
	if meetingID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "meeting_id_required"})
		return
	}

	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"full_name":         {},
			"role_name":         {},
			"member_type":       {},
			"attendance_status": {},
			"signature_present": {},
			"voting_right":      {},
		},
		[]string{"full_name", "role_name", "member_type", "attendance_status", "signature_present", "voting_right"},
	)
	if query.Sort == "" {
		query.Sort = "full_name"
	}

	whereClause, args := buildMeetingParticipantFilters(query.Filters, meetingID, s.institutionID(r))

	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_meeting_participants emp "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_meeting_participants_failed"})
		return
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select
			emp.id::text,
			emp.meeting_id::text,
			emp.full_name,
			emp.role_name,
			emp.member_type,
			emp.attendance_status,
			emp.voting_right,
			emp.signature_present,
			emp.institution_id,
			emp.notes
		from education_meeting_participants emp
		%s
		order by %s %s, emp.full_name, emp.role_name
		limit $%d offset $%d
	`, whereClause, meetingParticipantSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_meeting_participants_failed"})
		return
	}
	defer rows.Close()

	items := make([]GovernanceMeetingParticipant, 0, query.PageSize)
	for rows.Next() {
		var item GovernanceMeetingParticipant
		if err := rows.Scan(
			&item.ID,
			&item.MeetingID,
			&item.FullName,
			&item.RoleName,
			&item.MemberType,
			&item.AttendanceStatus,
			&item.VotingRight,
			&item.SignaturePresent,
			&item.InstitutionID,
			&item.Notes,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_meeting_participants_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_meeting_participants_scan_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) GovernanceMeetingParticipantDetail(w http.ResponseWriter, r *http.Request) {
	meetingID := strings.TrimSpace(chi.URLParam(r, "meetingID"))
	participantID := strings.TrimSpace(chi.URLParam(r, "participantID"))
	var item GovernanceMeetingParticipant
	err := s.pool.QueryRow(r.Context(), `
		select
			emp.id::text,
			emp.meeting_id::text,
			emp.full_name,
			emp.role_name,
			emp.member_type,
			emp.attendance_status,
			emp.voting_right,
			emp.signature_present,
			emp.institution_id,
			emp.notes
		from education_meeting_participants emp
		where emp.id = $1
			and emp.meeting_id = $2
			and emp.institution_id = $3
	`, participantID, meetingID, s.institutionID(r)).Scan(
		&item.ID,
		&item.MeetingID,
		&item.FullName,
		&item.RoleName,
		&item.MemberType,
		&item.AttendanceStatus,
		&item.VotingRight,
		&item.SignaturePresent,
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_meeting_participant_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_meeting_participant_detail_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreateGovernanceMeetingParticipant(w http.ResponseWriter, r *http.Request) {
	meetingID := strings.TrimSpace(chi.URLParam(r, "meetingID"))
	var req CreateGovernanceMeetingParticipantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_participant_payload"})
		return
	}

	req.FullName = strings.TrimSpace(req.FullName)
	req.RoleName = strings.TrimSpace(req.RoleName)
	req.MemberType = strings.TrimSpace(req.MemberType)
	req.AttendanceStatus = strings.TrimSpace(req.AttendanceStatus)
	req.Notes = strings.TrimSpace(req.Notes)

	if req.FullName == "" || req.RoleName == "" || req.MemberType == "" || req.AttendanceStatus == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_meeting_participant_fields"})
		return
	}
	if !containsString([]string{"presedinte", "secretar", "membru", "invitat", "observator"}, req.MemberType) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_participant_type"})
		return
	}
	if !containsString([]string{"invitat", "prezent", "absent_motivat", "absent_nemotivat"}, req.AttendanceStatus) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_participant_attendance"})
		return
	}

	var item GovernanceMeetingParticipant
	err := s.pool.QueryRow(r.Context(), `
		insert into education_meeting_participants (
			meeting_id,
			full_name,
			role_name,
			member_type,
			attendance_status,
			voting_right,
			signature_present,
			institution_id,
			notes
		)
		select
			em.id,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			em.institution_id,
			$8
		from education_meetings em
		where em.id = $1 and em.institution_id = $9
		returning
			id::text,
			meeting_id::text,
			full_name,
			role_name,
			member_type,
			attendance_status,
			voting_right,
			signature_present,
			institution_id,
			notes
	`, meetingID, req.FullName, req.RoleName, req.MemberType, req.AttendanceStatus, req.VotingRight, req.SignaturePresent, req.Notes, s.institutionID(r)).Scan(
		&item.ID,
		&item.MeetingID,
		&item.FullName,
		&item.RoleName,
		&item.MemberType,
		&item.AttendanceStatus,
		&item.VotingRight,
		&item.SignaturePresent,
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_meeting_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "meeting_participant_create_failed"})
		return
	}

	s.logAudit(r, "education.governance.participant.create", "governance_meeting_participant", item.ID, "Meeting participant created.", map[string]any{
		"meeting_id":        item.MeetingID,
		"full_name":         item.FullName,
		"role_name":         item.RoleName,
		"member_type":       item.MemberType,
		"attendance_status": item.AttendanceStatus,
		"signature_present": item.SignaturePresent,
		"voting_right":      item.VotingRight,
	})

	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdateGovernanceMeetingParticipant(w http.ResponseWriter, r *http.Request) {
	meetingID := strings.TrimSpace(chi.URLParam(r, "meetingID"))
	participantID := strings.TrimSpace(chi.URLParam(r, "participantID"))
	var req CreateGovernanceMeetingParticipantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_participant_payload"})
		return
	}

	req.FullName = strings.TrimSpace(req.FullName)
	req.RoleName = strings.TrimSpace(req.RoleName)
	req.MemberType = strings.TrimSpace(req.MemberType)
	req.AttendanceStatus = strings.TrimSpace(req.AttendanceStatus)
	req.Notes = strings.TrimSpace(req.Notes)

	if req.FullName == "" || req.RoleName == "" || req.MemberType == "" || req.AttendanceStatus == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_meeting_participant_fields"})
		return
	}
	if !containsString([]string{"presedinte", "secretar", "membru", "invitat", "observator"}, req.MemberType) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_participant_type"})
		return
	}
	if !containsString([]string{"invitat", "prezent", "absent_motivat", "absent_nemotivat"}, req.AttendanceStatus) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_participant_attendance"})
		return
	}

	var item GovernanceMeetingParticipant
	err := s.pool.QueryRow(r.Context(), `
		update education_meeting_participants
		set
			full_name = $1,
			role_name = $2,
			member_type = $3,
			attendance_status = $4,
			voting_right = $5,
			signature_present = $6,
			notes = $7,
			updated_at = now()
		where id = $8
			and meeting_id = $9
			and institution_id = $10
		returning
			id::text,
			meeting_id::text,
			full_name,
			role_name,
			member_type,
			attendance_status,
			voting_right,
			signature_present,
			institution_id,
			notes
	`, req.FullName, req.RoleName, req.MemberType, req.AttendanceStatus, req.VotingRight, req.SignaturePresent, req.Notes, participantID, meetingID, s.institutionID(r)).Scan(
		&item.ID,
		&item.MeetingID,
		&item.FullName,
		&item.RoleName,
		&item.MemberType,
		&item.AttendanceStatus,
		&item.VotingRight,
		&item.SignaturePresent,
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_meeting_participant_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "meeting_participant_update_failed"})
		return
	}

	s.logAudit(r, "education.governance.participant.update", "governance_meeting_participant", item.ID, "Meeting participant updated.", map[string]any{
		"meeting_id":        item.MeetingID,
		"full_name":         item.FullName,
		"role_name":         item.RoleName,
		"member_type":       item.MemberType,
		"attendance_status": item.AttendanceStatus,
		"signature_present": item.SignaturePresent,
		"voting_right":      item.VotingRight,
	})

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteGovernanceMeetingParticipant(w http.ResponseWriter, r *http.Request) {
	meetingID := strings.TrimSpace(chi.URLParam(r, "meetingID"))
	participantID := strings.TrimSpace(chi.URLParam(r, "participantID"))
	tag, err := s.pool.Exec(r.Context(), `
		delete from education_meeting_participants
		where id = $1 and meeting_id = $2 and institution_id = $3
	`, participantID, meetingID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "meeting_participant_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_meeting_participant_not_found")
		return
	}

	s.logAudit(r, "education.governance.participant.delete", "governance_meeting_participant", participantID, "Meeting participant deleted.", map[string]any{
		"meeting_id": meetingID,
	})

	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) GovernanceMeetingDocuments(w http.ResponseWriter, r *http.Request) {
	meetingID := strings.TrimSpace(chi.URLParam(r, "meetingID"))
	if meetingID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "meeting_id_required"})
		return
	}

	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"document_type":      {},
			"title":              {},
			"document_number":    {},
			"registry_number":    {},
			"publication_status": {},
			"issued_on":          {},
			"custody_owner":      {},
		},
		[]string{"document_type", "title", "document_number", "registry_number", "publication_status", "issued_on", "custody_owner"},
	)
	if query.Sort == "" {
		query.Sort = "issued_on"
	}

	whereClause, args := buildMeetingDocumentFilters(query.Filters, meetingID, s.institutionID(r))

	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_meeting_documents emd "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_meeting_documents_failed"})
		return
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select
			emd.id::text,
			emd.meeting_id::text,
			emd.document_type,
			emd.title,
			emd.document_number,
			emd.registry_number,
			emd.publication_status,
			emd.custody_owner,
			emd.signed_by,
			to_char(emd.issued_on, 'YYYY-MM-DD'),
			emd.institution_id,
			emd.summary
		from education_meeting_documents emd
		%s
		order by %s %s, emd.issued_on desc, emd.title
		limit $%d offset $%d
	`, whereClause, meetingDocumentSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_meeting_documents_failed"})
		return
	}
	defer rows.Close()

	items := make([]GovernanceMeetingDocument, 0, query.PageSize)
	for rows.Next() {
		var item GovernanceMeetingDocument
		if err := rows.Scan(
			&item.ID,
			&item.MeetingID,
			&item.DocumentType,
			&item.Title,
			&item.DocumentNumber,
			&item.RegistryNumber,
			&item.PublicationStatus,
			&item.CustodyOwner,
			&item.SignedBy,
			&item.IssuedOn,
			&item.InstitutionID,
			&item.Summary,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_meeting_documents_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_meeting_documents_scan_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) GovernanceMeetingDocumentDetail(w http.ResponseWriter, r *http.Request) {
	meetingID := strings.TrimSpace(chi.URLParam(r, "meetingID"))
	documentID := strings.TrimSpace(chi.URLParam(r, "documentID"))
	var item GovernanceMeetingDocument
	err := s.pool.QueryRow(r.Context(), `
		select
			emd.id::text,
			emd.meeting_id::text,
			emd.document_type,
			emd.title,
			emd.document_number,
			emd.registry_number,
			emd.publication_status,
			emd.custody_owner,
			emd.signed_by,
			to_char(emd.issued_on, 'YYYY-MM-DD'),
			emd.institution_id,
			emd.summary
		from education_meeting_documents emd
		where emd.id = $1
			and emd.meeting_id = $2
			and emd.institution_id = $3
	`, documentID, meetingID, s.institutionID(r)).Scan(
		&item.ID,
		&item.MeetingID,
		&item.DocumentType,
		&item.Title,
		&item.DocumentNumber,
		&item.RegistryNumber,
		&item.PublicationStatus,
		&item.CustodyOwner,
		&item.SignedBy,
		&item.IssuedOn,
		&item.InstitutionID,
		&item.Summary,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_meeting_document_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_meeting_document_detail_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreateGovernanceMeetingDocument(w http.ResponseWriter, r *http.Request) {
	meetingID := strings.TrimSpace(chi.URLParam(r, "meetingID"))
	var req CreateGovernanceMeetingDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_document_payload"})
		return
	}

	req.DocumentType = strings.TrimSpace(req.DocumentType)
	req.Title = strings.TrimSpace(req.Title)
	req.DocumentNumber = strings.TrimSpace(req.DocumentNumber)
	req.RegistryNumber = strings.TrimSpace(req.RegistryNumber)
	req.PublicationStatus = strings.TrimSpace(req.PublicationStatus)
	req.CustodyOwner = strings.TrimSpace(req.CustodyOwner)
	req.SignedBy = strings.TrimSpace(req.SignedBy)
	req.IssuedOn = strings.TrimSpace(req.IssuedOn)
	req.Summary = strings.TrimSpace(req.Summary)

	if req.DocumentType == "" || req.Title == "" || req.PublicationStatus == "" || req.IssuedOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_meeting_document_fields"})
		return
	}
	if !containsString([]string{"convocator", "convocator_ca", "convocator_cp", "ordine_de_zi", "prezenta", "proces_verbal", "proces_verbal_ca", "proces_verbal_cp", "registru_ca", "registru_cp", "numire_secretar_cp", "anexa", "hotarare", "material_sedinta", "delegare"}, req.DocumentType) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_document_type"})
		return
	}
	if !containsString([]string{"intern", "anonimizare_necesara", "publicat"}, req.PublicationStatus) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_document_publication_status"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.IssuedOn); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_document_date"})
		return
	}

	var item GovernanceMeetingDocument
	err := s.pool.QueryRow(r.Context(), `
		insert into education_meeting_documents (
			meeting_id,
			document_type,
			title,
			document_number,
			registry_number,
			publication_status,
			custody_owner,
			signed_by,
			issued_on,
			institution_id,
			summary
		)
		select
			em.id,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			em.institution_id,
			$10
		from education_meetings em
		where em.id = $1 and em.institution_id = $11
		returning
			id::text,
			meeting_id::text,
			document_type,
			title,
			document_number,
			registry_number,
			publication_status,
			custody_owner,
			signed_by,
			to_char(issued_on, 'YYYY-MM-DD'),
			institution_id,
			summary
	`, meetingID, req.DocumentType, req.Title, req.DocumentNumber, req.RegistryNumber, req.PublicationStatus, req.CustodyOwner, req.SignedBy, req.IssuedOn, req.Summary, s.institutionID(r)).Scan(
		&item.ID,
		&item.MeetingID,
		&item.DocumentType,
		&item.Title,
		&item.DocumentNumber,
		&item.RegistryNumber,
		&item.PublicationStatus,
		&item.CustodyOwner,
		&item.SignedBy,
		&item.IssuedOn,
		&item.InstitutionID,
		&item.Summary,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_meeting_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "meeting_document_create_failed"})
		return
	}

	s.logAudit(r, "education.governance.document.create", "governance_meeting_document", item.ID, "Meeting document created.", map[string]any{
		"meeting_id":         item.MeetingID,
		"document_type":      item.DocumentType,
		"title":              item.Title,
		"document_number":    item.DocumentNumber,
		"publication_status": item.PublicationStatus,
		"issued_on":          item.IssuedOn,
	})

	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdateGovernanceMeetingDocument(w http.ResponseWriter, r *http.Request) {
	meetingID := strings.TrimSpace(chi.URLParam(r, "meetingID"))
	documentID := strings.TrimSpace(chi.URLParam(r, "documentID"))
	var req CreateGovernanceMeetingDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_document_payload"})
		return
	}

	req.DocumentType = strings.TrimSpace(req.DocumentType)
	req.Title = strings.TrimSpace(req.Title)
	req.DocumentNumber = strings.TrimSpace(req.DocumentNumber)
	req.RegistryNumber = strings.TrimSpace(req.RegistryNumber)
	req.PublicationStatus = strings.TrimSpace(req.PublicationStatus)
	req.CustodyOwner = strings.TrimSpace(req.CustodyOwner)
	req.SignedBy = strings.TrimSpace(req.SignedBy)
	req.IssuedOn = strings.TrimSpace(req.IssuedOn)
	req.Summary = strings.TrimSpace(req.Summary)

	if req.DocumentType == "" || req.Title == "" || req.PublicationStatus == "" || req.IssuedOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_meeting_document_fields"})
		return
	}
	if !containsString([]string{"convocator", "convocator_ca", "convocator_cp", "ordine_de_zi", "prezenta", "proces_verbal", "proces_verbal_ca", "proces_verbal_cp", "registru_ca", "registru_cp", "numire_secretar_cp", "anexa", "hotarare", "material_sedinta", "delegare"}, req.DocumentType) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_document_type"})
		return
	}
	if !containsString([]string{"intern", "anonimizare_necesara", "publicat"}, req.PublicationStatus) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_document_publication_status"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.IssuedOn); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_document_date"})
		return
	}

	var item GovernanceMeetingDocument
	err := s.pool.QueryRow(r.Context(), `
		update education_meeting_documents
		set
			document_type = $1,
			title = $2,
			document_number = $3,
			registry_number = $4,
			publication_status = $5,
			custody_owner = $6,
			signed_by = $7,
			issued_on = $8,
			summary = $9,
			updated_at = now()
		where id = $10
			and meeting_id = $11
			and institution_id = $12
		returning
			id::text,
			meeting_id::text,
			document_type,
			title,
			document_number,
			registry_number,
			publication_status,
			custody_owner,
			signed_by,
			to_char(issued_on, 'YYYY-MM-DD'),
			institution_id,
			summary
	`, req.DocumentType, req.Title, req.DocumentNumber, req.RegistryNumber, req.PublicationStatus, req.CustodyOwner, req.SignedBy, req.IssuedOn, req.Summary, documentID, meetingID, s.institutionID(r)).Scan(
		&item.ID,
		&item.MeetingID,
		&item.DocumentType,
		&item.Title,
		&item.DocumentNumber,
		&item.RegistryNumber,
		&item.PublicationStatus,
		&item.CustodyOwner,
		&item.SignedBy,
		&item.IssuedOn,
		&item.InstitutionID,
		&item.Summary,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_meeting_document_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "meeting_document_update_failed"})
		return
	}

	s.logAudit(r, "education.governance.document.update", "governance_meeting_document", item.ID, "Meeting document updated.", map[string]any{
		"meeting_id":         item.MeetingID,
		"document_type":      item.DocumentType,
		"title":              item.Title,
		"document_number":    item.DocumentNumber,
		"publication_status": item.PublicationStatus,
		"issued_on":          item.IssuedOn,
	})

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteGovernanceMeetingDocument(w http.ResponseWriter, r *http.Request) {
	meetingID := strings.TrimSpace(chi.URLParam(r, "meetingID"))
	documentID := strings.TrimSpace(chi.URLParam(r, "documentID"))
	tag, err := s.pool.Exec(r.Context(), `
		delete from education_meeting_documents
		where id = $1 and meeting_id = $2 and institution_id = $3
	`, documentID, meetingID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "meeting_document_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_meeting_document_not_found")
		return
	}

	s.logAudit(r, "education.governance.document.delete", "governance_meeting_document", documentID, "Meeting document deleted.", map[string]any{
		"meeting_id": meetingID,
	})

	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) GovernanceMeetingVotes(w http.ResponseWriter, r *http.Request) {
	meetingID := strings.TrimSpace(chi.URLParam(r, "meetingID"))
	if meetingID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "meeting_id_required"})
		return
	}

	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"subject_title":      {},
			"agenda_order":       {},
			"decision_type":      {},
			"outcome":            {},
			"requires_follow_up": {},
		},
		[]string{"subject_title", "agenda_order", "decision_type", "outcome", "requires_follow_up"},
	)
	if query.Sort == "" {
		query.Sort = "agenda_order"
	}

	whereClause, args := buildMeetingVoteFilters(query.Filters, meetingID, s.institutionID(r))

	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_meeting_votes emv "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_meeting_votes_failed"})
		return
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select
			emv.id::text,
			emv.meeting_id::text,
			emv.subject_title,
			emv.agenda_order,
			emv.decision_type,
			emv.votes_for,
			emv.votes_against,
			emv.abstentions,
			emv.outcome,
			emv.requires_follow_up,
			emv.legal_basis,
			emv.institution_id,
			emv.notes
		from education_meeting_votes emv
		%s
		order by %s %s, emv.agenda_order, emv.subject_title
		limit $%d offset $%d
	`, whereClause, meetingVoteSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_meeting_votes_failed"})
		return
	}
	defer rows.Close()

	items := make([]GovernanceMeetingVote, 0, query.PageSize)
	for rows.Next() {
		var item GovernanceMeetingVote
		if err := rows.Scan(
			&item.ID,
			&item.MeetingID,
			&item.SubjectTitle,
			&item.AgendaOrder,
			&item.DecisionType,
			&item.VotesFor,
			&item.VotesAgainst,
			&item.Abstentions,
			&item.Outcome,
			&item.RequiresFollowUp,
			&item.LegalBasis,
			&item.InstitutionID,
			&item.Notes,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_meeting_votes_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_meeting_votes_scan_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) GovernanceMeetingVoteDetail(w http.ResponseWriter, r *http.Request) {
	meetingID := strings.TrimSpace(chi.URLParam(r, "meetingID"))
	voteID := strings.TrimSpace(chi.URLParam(r, "voteID"))
	var item GovernanceMeetingVote
	err := s.pool.QueryRow(r.Context(), `
		select
			emv.id::text,
			emv.meeting_id::text,
			emv.subject_title,
			emv.agenda_order,
			emv.decision_type,
			emv.votes_for,
			emv.votes_against,
			emv.abstentions,
			emv.outcome,
			emv.requires_follow_up,
			emv.legal_basis,
			emv.institution_id,
			emv.notes
		from education_meeting_votes emv
		where emv.id = $1
			and emv.meeting_id = $2
			and emv.institution_id = $3
	`, voteID, meetingID, s.institutionID(r)).Scan(
		&item.ID,
		&item.MeetingID,
		&item.SubjectTitle,
		&item.AgendaOrder,
		&item.DecisionType,
		&item.VotesFor,
		&item.VotesAgainst,
		&item.Abstentions,
		&item.Outcome,
		&item.RequiresFollowUp,
		&item.LegalBasis,
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_meeting_vote_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_meeting_vote_detail_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreateGovernanceMeetingVote(w http.ResponseWriter, r *http.Request) {
	meetingID := strings.TrimSpace(chi.URLParam(r, "meetingID"))
	if meetingID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_id"})
		return
	}
	var req CreateGovernanceMeetingVoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_vote_payload"})
		return
	}

	normalizeMeetingVoteRequest(&req)
	if req.SubjectTitle == "" || req.DecisionType == "" || req.Outcome == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_meeting_vote_fields"})
		return
	}
	if req.AgendaOrder < 1 || req.VotesFor < 0 || req.VotesAgainst < 0 || req.Abstentions < 0 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_vote_counts"})
		return
	}
	if !containsString([]string{"hotarare", "aviz", "informare", "delegare", "aprobare"}, req.DecisionType) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_vote_decision_type"})
		return
	}
	if !containsString([]string{"adoptat", "respins", "amanat"}, req.Outcome) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_vote_outcome"})
		return
	}
	if !s.ensureGovernanceMeetingVoteAccess(w, r, meetingID) {
		return
	}

	var item GovernanceMeetingVote
	err := s.pool.QueryRow(r.Context(), `
		insert into education_meeting_votes (
			meeting_id,
			subject_title,
			agenda_order,
			decision_type,
			votes_for,
			votes_against,
			abstentions,
			outcome,
			requires_follow_up,
			legal_basis,
			institution_id,
			notes
		)
		select
			em.id,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10,
			em.institution_id,
			$11
		from education_meetings em
		where em.id = $1 and em.institution_id = $12
		returning
			id::text,
			meeting_id::text,
			subject_title,
			agenda_order,
			decision_type,
			votes_for,
			votes_against,
			abstentions,
			outcome,
			requires_follow_up,
			legal_basis,
			institution_id,
			notes
	`, meetingID, req.SubjectTitle, req.AgendaOrder, req.DecisionType, req.VotesFor, req.VotesAgainst, req.Abstentions, req.Outcome, req.RequiresFollowUp, req.LegalBasis, req.Notes, s.institutionID(r)).Scan(
		&item.ID,
		&item.MeetingID,
		&item.SubjectTitle,
		&item.AgendaOrder,
		&item.DecisionType,
		&item.VotesFor,
		&item.VotesAgainst,
		&item.Abstentions,
		&item.Outcome,
		&item.RequiresFollowUp,
		&item.LegalBasis,
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_meeting_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "meeting_vote_create_failed"})
		return
	}

	s.logAudit(r, "education.governance.vote.create", "governance_meeting_vote", item.ID, "Meeting vote created.", map[string]any{
		"meeting_id":    item.MeetingID,
		"subject_title": item.SubjectTitle,
		"agenda_order":  item.AgendaOrder,
		"decision_type": item.DecisionType,
		"outcome":       item.Outcome,
		"votes_for":     item.VotesFor,
		"votes_against": item.VotesAgainst,
		"abstentions":   item.Abstentions,
	})

	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdateGovernanceMeetingVote(w http.ResponseWriter, r *http.Request) {
	meetingID := strings.TrimSpace(chi.URLParam(r, "meetingID"))
	voteID := strings.TrimSpace(chi.URLParam(r, "voteID"))
	if meetingID == "" || voteID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_vote_id"})
		return
	}
	var req CreateGovernanceMeetingVoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_vote_payload"})
		return
	}

	normalizeMeetingVoteRequest(&req)
	if req.SubjectTitle == "" || req.DecisionType == "" || req.Outcome == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_meeting_vote_fields"})
		return
	}
	if req.AgendaOrder < 1 || req.VotesFor < 0 || req.VotesAgainst < 0 || req.Abstentions < 0 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_vote_counts"})
		return
	}
	if !containsString([]string{"hotarare", "aviz", "informare", "delegare", "aprobare"}, req.DecisionType) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_vote_decision_type"})
		return
	}
	if !containsString([]string{"adoptat", "respins", "amanat"}, req.Outcome) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_vote_outcome"})
		return
	}
	if !s.ensureGovernanceMeetingVoteAccess(w, r, meetingID) {
		return
	}

	var item GovernanceMeetingVote
	err := s.pool.QueryRow(r.Context(), `
		update education_meeting_votes
		set
			subject_title = $1,
			agenda_order = $2,
			decision_type = $3,
			votes_for = $4,
			votes_against = $5,
			abstentions = $6,
			outcome = $7,
			requires_follow_up = $8,
			legal_basis = $9,
			notes = $10,
			updated_at = now()
		where id = $11
			and meeting_id = $12
			and institution_id = $13
		returning
			id::text,
			meeting_id::text,
			subject_title,
			agenda_order,
			decision_type,
			votes_for,
			votes_against,
			abstentions,
			outcome,
			requires_follow_up,
			legal_basis,
			institution_id,
			notes
	`, req.SubjectTitle, req.AgendaOrder, req.DecisionType, req.VotesFor, req.VotesAgainst, req.Abstentions, req.Outcome, req.RequiresFollowUp, req.LegalBasis, req.Notes, voteID, meetingID, s.institutionID(r)).Scan(
		&item.ID,
		&item.MeetingID,
		&item.SubjectTitle,
		&item.AgendaOrder,
		&item.DecisionType,
		&item.VotesFor,
		&item.VotesAgainst,
		&item.Abstentions,
		&item.Outcome,
		&item.RequiresFollowUp,
		&item.LegalBasis,
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_meeting_vote_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "meeting_vote_update_failed"})
		return
	}

	s.logAudit(r, "education.governance.vote.update", "governance_meeting_vote", item.ID, "Meeting vote updated.", map[string]any{
		"meeting_id":    item.MeetingID,
		"subject_title": item.SubjectTitle,
		"agenda_order":  item.AgendaOrder,
		"decision_type": item.DecisionType,
		"outcome":       item.Outcome,
		"votes_for":     item.VotesFor,
		"votes_against": item.VotesAgainst,
		"abstentions":   item.Abstentions,
	})

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteGovernanceMeetingVote(w http.ResponseWriter, r *http.Request) {
	meetingID := strings.TrimSpace(chi.URLParam(r, "meetingID"))
	voteID := strings.TrimSpace(chi.URLParam(r, "voteID"))
	if meetingID == "" || voteID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_vote_id"})
		return
	}
	if !s.ensureGovernanceMeetingVoteAccess(w, r, meetingID) {
		return
	}
	tag, err := s.pool.Exec(r.Context(), `
		delete from education_meeting_votes
		where id = $1 and meeting_id = $2 and institution_id = $3
	`, voteID, meetingID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "meeting_vote_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_meeting_vote_not_found")
		return
	}

	s.logAudit(r, "education.governance.vote.delete", "governance_meeting_vote", voteID, "Meeting vote deleted.", map[string]any{
		"meeting_id": meetingID,
	})

	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) PortfolioDocuments(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	if recordID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "portfolio_id_required"})
		return
	}

	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"section_code":        {},
			"component_code":      {},
			"document_title":      {},
			"description":         {},
			"school_year":         {},
			"subject_discipline":  {},
			"applicable_class":    {},
			"competencies":        {},
			"source_scope":        {},
			"evidence_type":       {},
			"issued_on":           {},
			"chronological_index": {},
			"sensitive_data":      {},
			"authenticity_status": {},
			"archive_version_no":  {},
			"archive_sha256":      {},
		},
		[]string{"section_code", "component_code", "document_title", "description", "school_year", "subject_discipline", "applicable_class", "source_scope", "evidence_type", "issued_on", "chronological_index", "sensitive_data", "authenticity_status", "archive_version_no"},
	)
	if query.Sort == "" {
		query.Sort = "issued_on"
	}

	whereClause, args := buildPortfolioDocumentFilters(query.Filters, recordID, s.institutionID(r))

	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_portfolio_documents epd "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_documents_failed"})
		return
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select
			epd.id::text,
			epd.portfolio_id::text,
			epd.section_code,
			epd.component_code,
			epd.document_title,
			epd.description,
			epd.school_year,
			epd.subject_discipline,
			epd.applicable_class,
			epd.competencies,
			epd.source_scope,
			epd.evidence_type,
			to_char(epd.issued_on, 'YYYY-MM-DD'),
			to_char(epd.added_on, 'YYYY-MM-DD'),
			epd.chronological_index,
			epd.sensitive_data,
			epd.authenticity_status,
			epd.file_reference,
			coalesce(epd.archive_document_id::text, ''),
			coalesce(epd.archive_version_id::text, ''),
			coalesce(epd.archive_version_no, 0),
			epd.archive_sha256,
			epd.institution_id,
			epd.notes
		from education_portfolio_documents epd
		%s
		order by %s %s, epd.id asc
		limit $%d offset $%d
	`, whereClause, portfolioDocumentSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_documents_failed"})
		return
	}
	defer rows.Close()

	items := make([]PortfolioDocument, 0, query.PageSize)
	for rows.Next() {
		var item PortfolioDocument
		if err := rows.Scan(
			&item.ID,
			&item.PortfolioID,
			&item.SectionCode,
			&item.ComponentCode,
			&item.DocumentTitle,
			&item.Description,
			&item.SchoolYear,
			&item.SubjectDiscipline,
			&item.ApplicableClass,
			&item.Competencies,
			&item.SourceScope,
			&item.EvidenceType,
			&item.IssuedOn,
			&item.AddedOn,
			&item.ChronologicalIndex,
			&item.SensitiveData,
			&item.AuthenticityStatus,
			&item.FileReference,
			&item.ArchiveDocumentID,
			&item.ArchiveVersionID,
			&item.ArchiveVersionNo,
			&item.ArchiveSHA256,
			&item.InstitutionID,
			&item.Notes,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_documents_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_documents_scan_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) PortfolioDocumentDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	documentID := strings.TrimSpace(chi.URLParam(r, "documentID"))
	var item PortfolioDocument
	err := s.pool.QueryRow(r.Context(), `
		select
			epd.id::text,
			epd.portfolio_id::text,
			epd.section_code,
			epd.component_code,
			epd.document_title,
			epd.description,
			epd.school_year,
			epd.subject_discipline,
			epd.applicable_class,
			epd.competencies,
			epd.source_scope,
			epd.evidence_type,
			to_char(epd.issued_on, 'YYYY-MM-DD'),
			to_char(epd.added_on, 'YYYY-MM-DD'),
			epd.chronological_index,
			epd.sensitive_data,
			epd.authenticity_status,
			epd.file_reference,
			coalesce(epd.archive_document_id::text, ''),
			coalesce(epd.archive_version_id::text, ''),
			coalesce(epd.archive_version_no, 0),
			epd.archive_sha256,
			epd.institution_id,
			epd.notes
		from education_portfolio_documents epd
		where epd.id = $1
			and epd.portfolio_id = $2
			and epd.institution_id = $3
			and epd.status = 'active'
	`, documentID, recordID, s.institutionID(r)).Scan(
		&item.ID,
		&item.PortfolioID,
		&item.SectionCode,
		&item.ComponentCode,
		&item.DocumentTitle,
		&item.Description,
		&item.SchoolYear,
		&item.SubjectDiscipline,
		&item.ApplicableClass,
		&item.Competencies,
		&item.SourceScope,
		&item.EvidenceType,
		&item.IssuedOn,
		&item.AddedOn,
		&item.ChronologicalIndex,
		&item.SensitiveData,
		&item.AuthenticityStatus,
		&item.FileReference,
		&item.ArchiveDocumentID,
		&item.ArchiveVersionID,
		&item.ArchiveVersionNo,
		&item.ArchiveSHA256,
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_portfolio_document_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_document_detail_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) PortfolioDocumentVersions(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	documentID := strings.TrimSpace(chi.URLParam(r, "documentID"))
	if _, err := uuid.Parse(recordID); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_id"})
		return
	}
	if _, err := uuid.Parse(documentID); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_document_id"})
		return
	}

	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{"version_no": {}, "change_type": {}, "changed_by": {}, "changed_at": {}, "reason": {}},
		[]string{"version_no", "change_type", "changed_by", "changed_at", "reason"},
	)
	if query.Sort == "" {
		query.Sort = "version_no"
		query.Direction = "desc"
	}

	where := []string{
		"epd.id = $1::uuid",
		"epd.portfolio_id = $2::uuid",
		"epd.institution_id = $3",
		"ev.entity_table = 'education_portfolio_documents'",
		"ev.entity_id = epd.id",
		"ev.institution_id = epd.institution_id",
	}
	args := []any{documentID, recordID, s.institutionID(r)}
	addContains := func(expression, value string) {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(value))+"%")
		where = append(where, fmt.Sprintf("lower(%s) like $%d", expression, len(args)))
	}
	for key, value := range query.Filters {
		switch key {
		case "version_no":
			args = append(args, "%"+strings.TrimSpace(value)+"%")
			where = append(where, fmt.Sprintf("ev.version_no::text like $%d", len(args)))
		case "change_type":
			addContains("ev.change_type", value)
		case "changed_by":
			addContains("ev.changed_by", value)
		case "changed_at":
			addContains("to_char(ev.changed_at, 'YYYY-MM-DD HH24:MI:SS')", value)
		case "reason":
			addContains("coalesce(nullif(ev.snapshot->>'last_change_reason', ''), ev.change_type)", value)
		}
	}
	whereClause := "where " + strings.Join(where, " and ")

	var total int
	if err := s.pool.QueryRow(r.Context(), `
		select count(*)
		from app_entity_versions ev
		join education_portfolio_documents epd on epd.id = ev.entity_id
		`+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_document_versions_failed"})
		return
	}
	if total == 0 {
		var exists bool
		if err := s.pool.QueryRow(r.Context(), `select exists(select 1 from education_portfolio_documents where id = $1::uuid and portfolio_id = $2::uuid and institution_id = $3)`, documentID, recordID, s.institutionID(r)).Scan(&exists); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_document_versions_failed"})
			return
		}
		if !exists {
			writeEducationNotFound(w, "education_portfolio_document_not_found")
			return
		}
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	sortColumn := map[string]string{
		"version_no": "ev.version_no", "change_type": "ev.change_type", "changed_by": "ev.changed_by",
		"changed_at": "ev.changed_at", "reason": "coalesce(nullif(ev.snapshot->>'last_change_reason', ''), ev.change_type)",
	}[query.Sort]
	if sortColumn == "" {
		sortColumn = "ev.version_no"
	}
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select ev.version_no, ev.change_type, ev.changed_by,
			to_char(ev.changed_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			coalesce(nullif(ev.snapshot->>'last_change_reason', ''), ev.change_type),
			jsonb_build_object(
				'id', ev.snapshot->'id',
				'portfolio_id', ev.snapshot->'portfolio_id',
				'section_code', ev.snapshot->'section_code',
				'component_code', ev.snapshot->'component_code',
				'document_title', ev.snapshot->'document_title',
				'description', ev.snapshot->'description',
				'school_year', ev.snapshot->'school_year',
				'subject_discipline', ev.snapshot->'subject_discipline',
				'applicable_class', ev.snapshot->'applicable_class',
				'competencies', ev.snapshot->'competencies',
				'source_scope', ev.snapshot->'source_scope',
				'evidence_type', ev.snapshot->'evidence_type',
				'issued_on', ev.snapshot->'issued_on',
				'added_on', ev.snapshot->'added_on',
				'chronological_index', ev.snapshot->'chronological_index',
				'sensitive_data', ev.snapshot->'sensitive_data',
				'authenticity_status', ev.snapshot->'authenticity_status',
				'file_reference', ev.snapshot->'file_reference',
				'archive_document_id', ev.snapshot->'archive_document_id',
				'archive_version_id', ev.snapshot->'archive_version_id',
				'archive_version_no', ev.snapshot->'archive_version_no',
				'archive_sha256', ev.snapshot->'archive_sha256',
				'status', ev.snapshot->'status',
				'notes', ev.snapshot->'notes',
				'withdrawn_at', ev.snapshot->'withdrawn_at',
				'withdrawal_reason', ev.snapshot->'withdrawal_reason',
				'created_at', ev.snapshot->'created_at',
				'updated_at', ev.snapshot->'updated_at'
			)
		from app_entity_versions ev
		join education_portfolio_documents epd on epd.id = ev.entity_id
		%s
		order by %s %s, ev.version_no desc
		limit $%d offset $%d
	`, whereClause, sortColumn, strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_document_versions_failed"})
		return
	}
	defer rows.Close()
	items := make([]PortfolioDocumentVersion, 0, query.PageSize)
	for rows.Next() {
		var item PortfolioDocumentVersion
		if err := rows.Scan(&item.VersionNo, &item.ChangeType, &item.ChangedBy, &item.ChangedAt, &item.Reason, &item.Snapshot); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_document_versions_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_document_versions_scan_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) CreatePortfolioDocument(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreatePortfolioDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_document_payload"})
		return
	}

	normalizePortfolioDocumentRequest(&req)
	if req.AddedOn == "" {
		req.AddedOn = time.Now().Format("2006-01-02")
	}

	archiveReference, archiveDocumentID, archiveReferenceValid := portfolioArchiveReference(req.FileReference)
	if req.SectionCode == "" || req.ComponentCode == "" || req.DocumentTitle == "" || req.Description == "" || req.SchoolYear == "" || req.SubjectDiscipline == "" || req.ApplicableClass == "" || len(req.Competencies) == 0 || req.SourceScope == "" || req.EvidenceType == "" || req.IssuedOn == "" || req.AddedOn == "" || req.AuthenticityStatus == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_portfolio_document_fields"})
		return
	}
	if !archiveReferenceValid {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "portfolio_document_archive_reference_required"})
		return
	}
	req.FileReference = archiveReference
	if !containsString([]string{"portofoliu", "dosar_personal"}, req.SourceScope) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_document_source_scope"})
		return
	}
	if !containsString([]string{"declarat", "verificat", "respins"}, req.AuthenticityStatus) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_document_authenticity"})
		return
	}
	if req.ChronologicalIndex < 0 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_document_index"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.IssuedOn); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_document_issued_on"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.AddedOn); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_document_added_on"})
		return
	}

	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_document_create_failed"})
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	var item PortfolioDocument
	err = tx.QueryRow(r.Context(), `
		insert into education_portfolio_documents (
			portfolio_id,
			section_code,
			component_code,
			document_title,
			description,
			school_year,
			subject_discipline,
			applicable_class,
			competencies,
			source_scope,
			evidence_type,
			issued_on,
			added_on,
			chronological_index,
			sensitive_data,
			authenticity_status,
			file_reference,
			archive_document_id,
			archive_version_id,
			archive_version_no,
			archive_source_bucket,
			archive_source_object_key,
			archive_sha256,
			last_change_reason,
			institution_id,
			notes
		)
		select
			ep.id,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10,
			$11,
			$12,
			$13,
			$14,
			$15,
			$16,
			$17,
			document.id,
			version.id,
			version.version_no,
			version.source_bucket,
			version.source_object_key,
			lower(version.source_sha256),
			'Document adăugat în portofoliu',
			ep.institution_id,
			$19
		from education_portfolios ep
		join archive_documents document
			on document.id = $18::uuid and document.institution_id = ep.institution_id and document.status = 'ready'
		join archive_document_versions version
			on version.document_id = document.id and version.version_no = document.current_version_no
			and version.institution_id = document.institution_id and version.status = 'active'
			and btrim(version.source_bucket) <> '' and btrim(version.source_object_key) <> ''
			and lower(version.source_sha256) ~ '^[0-9a-f]{64}$'
		where ep.id = $1 and ep.institution_id = $20
		returning
			id::text,
			portfolio_id::text,
			section_code,
			component_code,
			document_title,
			description,
			school_year,
			subject_discipline,
			applicable_class,
			competencies,
			source_scope,
			evidence_type,
			to_char(issued_on, 'YYYY-MM-DD'),
			to_char(added_on, 'YYYY-MM-DD'),
			chronological_index,
			sensitive_data,
			authenticity_status,
			file_reference,
			archive_document_id::text,
			archive_version_id::text,
			archive_version_no,
			archive_sha256,
			institution_id,
			notes
	`, recordID, req.SectionCode, req.ComponentCode, req.DocumentTitle, req.Description, req.SchoolYear, req.SubjectDiscipline, req.ApplicableClass, req.Competencies, req.SourceScope, req.EvidenceType, req.IssuedOn, req.AddedOn, req.ChronologicalIndex, req.SensitiveData, req.AuthenticityStatus, req.FileReference, archiveDocumentID, req.Notes, s.institutionID(r)).Scan(
		&item.ID,
		&item.PortfolioID,
		&item.SectionCode,
		&item.ComponentCode,
		&item.DocumentTitle,
		&item.Description,
		&item.SchoolYear,
		&item.SubjectDiscipline,
		&item.ApplicableClass,
		&item.Competencies,
		&item.SourceScope,
		&item.EvidenceType,
		&item.IssuedOn,
		&item.AddedOn,
		&item.ChronologicalIndex,
		&item.SensitiveData,
		&item.AuthenticityStatus,
		&item.FileReference,
		&item.ArchiveDocumentID,
		&item.ArchiveVersionID,
		&item.ArchiveVersionNo,
		&item.ArchiveSHA256,
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "portfolio_document_archive_version_unavailable"})
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_document_create_failed"})
		return
	}

	if _, _, err := s.syncPortfolioOpisTx(r.Context(), tx, r, recordID, s.institutionID(r)); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_document_create_opis_sync_failed"})
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_document_create_failed"})
		return
	}

	// Auditing is deliberately post-commit: an audit event must never claim a
	// document exists when the document/OPIS transaction was rolled back.
	s.logAudit(r, "education.portfolios.document.create", "portfolio_document", item.ID, "Portfolio document created.", map[string]any{
		"portfolio_id":        item.PortfolioID,
		"section_code":        item.SectionCode,
		"component_code":      item.ComponentCode,
		"document_title":      item.DocumentTitle,
		"source_scope":        item.SourceScope,
		"authenticity_status": item.AuthenticityStatus,
		"chronological_index": item.ChronologicalIndex,
	})

	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdatePortfolioDocument(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	documentID := strings.TrimSpace(chi.URLParam(r, "documentID"))
	var req CreatePortfolioDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_document_payload"})
		return
	}

	normalizePortfolioDocumentRequest(&req)

	archiveReference, archiveDocumentID, archiveReferenceValid := portfolioArchiveReference(req.FileReference)
	if req.SectionCode == "" || req.ComponentCode == "" || req.DocumentTitle == "" || req.Description == "" || req.SchoolYear == "" || req.SubjectDiscipline == "" || req.ApplicableClass == "" || len(req.Competencies) == 0 || req.SourceScope == "" || req.EvidenceType == "" || req.IssuedOn == "" || req.AddedOn == "" || req.AuthenticityStatus == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_portfolio_document_fields"})
		return
	}
	if !archiveReferenceValid {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "portfolio_document_archive_reference_required"})
		return
	}
	req.FileReference = archiveReference
	if !containsString([]string{"portofoliu", "dosar_personal"}, req.SourceScope) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_document_source_scope"})
		return
	}
	if !containsString([]string{"declarat", "verificat", "respins"}, req.AuthenticityStatus) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_document_authenticity"})
		return
	}
	if req.ChronologicalIndex < 0 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_document_index"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.IssuedOn); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_document_issued_on"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.AddedOn); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_document_added_on"})
		return
	}

	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_document_update_failed"})
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	var item PortfolioDocument
	err = tx.QueryRow(r.Context(), `
		update education_portfolio_documents
		set
			section_code = $1,
			component_code = $2,
			document_title = $3,
			description = $4,
			school_year = $5,
			subject_discipline = $6,
			applicable_class = $7,
			competencies = $8,
			source_scope = $9,
			evidence_type = $10,
			issued_on = $11,
			added_on = $12,
			chronological_index = $13,
			sensitive_data = $14,
			authenticity_status = $15,
			file_reference = $16,
			archive_document_id = document.id,
			archive_version_id = version.id,
			archive_version_no = version.version_no,
			archive_source_bucket = version.source_bucket,
			archive_source_object_key = version.source_object_key,
			archive_sha256 = lower(version.source_sha256),
			last_change_reason = 'Document și metadate actualizate',
			notes = $18,
			updated_at = now()
		from archive_documents document
		join archive_document_versions version
			on version.document_id = document.id and version.version_no = document.current_version_no
			and version.institution_id = document.institution_id and version.status = 'active'
			and btrim(version.source_bucket) <> '' and btrim(version.source_object_key) <> ''
			and lower(version.source_sha256) ~ '^[0-9a-f]{64}$'
		where education_portfolio_documents.id = $19
			and education_portfolio_documents.portfolio_id = $20
			and education_portfolio_documents.institution_id = $21
			and document.id = $17::uuid
			and document.institution_id = education_portfolio_documents.institution_id
			and document.status = 'ready'
			and education_portfolio_documents.status = 'active'
		returning
			education_portfolio_documents.id::text,
			education_portfolio_documents.portfolio_id::text,
			education_portfolio_documents.section_code,
			education_portfolio_documents.component_code,
			education_portfolio_documents.document_title,
			education_portfolio_documents.description,
			education_portfolio_documents.school_year,
			education_portfolio_documents.subject_discipline,
			education_portfolio_documents.applicable_class,
			education_portfolio_documents.competencies,
			education_portfolio_documents.source_scope,
			education_portfolio_documents.evidence_type,
			to_char(education_portfolio_documents.issued_on, 'YYYY-MM-DD'),
			to_char(education_portfolio_documents.added_on, 'YYYY-MM-DD'),
			education_portfolio_documents.chronological_index,
			education_portfolio_documents.sensitive_data,
			education_portfolio_documents.authenticity_status,
			education_portfolio_documents.file_reference,
			education_portfolio_documents.archive_document_id::text,
			education_portfolio_documents.archive_version_id::text,
			education_portfolio_documents.archive_version_no,
			education_portfolio_documents.archive_sha256,
			education_portfolio_documents.institution_id,
			education_portfolio_documents.notes
	`, req.SectionCode, req.ComponentCode, req.DocumentTitle, req.Description, req.SchoolYear, req.SubjectDiscipline, req.ApplicableClass, req.Competencies, req.SourceScope, req.EvidenceType, req.IssuedOn, req.AddedOn, req.ChronologicalIndex, req.SensitiveData, req.AuthenticityStatus, req.FileReference, archiveDocumentID, req.Notes, documentID, recordID, s.institutionID(r)).Scan(
		&item.ID,
		&item.PortfolioID,
		&item.SectionCode,
		&item.ComponentCode,
		&item.DocumentTitle,
		&item.Description,
		&item.SchoolYear,
		&item.SubjectDiscipline,
		&item.ApplicableClass,
		&item.Competencies,
		&item.SourceScope,
		&item.EvidenceType,
		&item.IssuedOn,
		&item.AddedOn,
		&item.ChronologicalIndex,
		&item.SensitiveData,
		&item.AuthenticityStatus,
		&item.FileReference,
		&item.ArchiveDocumentID,
		&item.ArchiveVersionID,
		&item.ArchiveVersionNo,
		&item.ArchiveSHA256,
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			var exists bool
			if lookupErr := tx.QueryRow(r.Context(), `
				select exists(
					select 1 from education_portfolio_documents
					where id = $1::uuid and portfolio_id = $2::uuid and institution_id = $3 and status = 'active'
				)
			`, documentID, recordID, s.institutionID(r)).Scan(&exists); lookupErr != nil {
				httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_document_update_failed"})
				return
			}
			if exists {
				httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "portfolio_document_archive_version_unavailable"})
				return
			}
			writeEducationNotFound(w, "education_portfolio_document_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_document_update_failed"})
		return
	}

	if _, _, err := s.syncPortfolioOpisTx(r.Context(), tx, r, recordID, s.institutionID(r)); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_document_update_opis_sync_failed"})
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_document_update_failed"})
		return
	}

	s.logAudit(r, "education.portfolios.document.update", "portfolio_document", item.ID, "Portfolio document updated.", map[string]any{
		"portfolio_id":        item.PortfolioID,
		"section_code":        item.SectionCode,
		"component_code":      item.ComponentCode,
		"document_title":      item.DocumentTitle,
		"source_scope":        item.SourceScope,
		"authenticity_status": item.AuthenticityStatus,
		"chronological_index": item.ChronologicalIndex,
	})

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeletePortfolioDocument(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	documentID := strings.TrimSpace(chi.URLParam(r, "documentID"))
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_document_delete_failed"})
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	tag, err := tx.Exec(r.Context(), `
		update education_portfolio_documents document
		set status = 'withdrawn', last_change_reason = 'Document retras din portofoliu', updated_at = now()
		from education_portfolios portfolio
		where document.id = $1
			and document.portfolio_id = $2
			and document.institution_id = $3
			and document.status = 'active'
			and portfolio.id = document.portfolio_id
			and portfolio.institution_id = document.institution_id
			and portfolio.status in ('draft', 'returned')
			and portfolio.withdrawn_at is null
			and not portfolio.legal_hold_active
	`, documentID, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_document_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		if err := tx.QueryRow(r.Context(), `
			select exists(
				select 1 from education_portfolio_documents
				where id = $1 and portfolio_id = $2 and institution_id = $3
			)
		`, documentID, recordID, s.institutionID(r)).Scan(&exists); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_document_delete_failed"})
			return
		}
		if !exists {
			writeEducationNotFound(w, "education_portfolio_document_not_found")
			return
		}
		httpx.JSON(w, http.StatusConflict, map[string]any{"code": "education_portfolio_document_not_editable"})
		return
	}

	if _, _, err := s.syncPortfolioOpisTx(r.Context(), tx, r, recordID, s.institutionID(r)); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_document_delete_opis_sync_failed"})
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_document_delete_failed"})
		return
	}

	s.logAudit(r, "education.portfolios.document.withdraw", "portfolio_document", documentID, "Portfolio document withdrawn; evidence retained for audit.", map[string]any{
		"portfolio_id": recordID,
	})

	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) PortfolioChecklistItems(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	if recordID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "portfolio_id_required"})
		return
	}

	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"requirement_code":  {},
			"requirement_label": {},
			"section_code":      {},
			"status":            {},
			"document_count":    {},
		},
		[]string{"requirement_code", "section_code", "source_scope", "status", "mandatory", "checked_by"},
	)
	if query.Sort == "" {
		query.Sort = "requirement_code"
	}

	whereClause, args := buildPortfolioChecklistFilters(query.Filters, recordID, s.institutionID(r))

	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_portfolio_checklist epc "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_checklist_failed"})
		return
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select
			epc.id::text,
			epc.portfolio_id::text,
			epc.requirement_code,
			epc.requirement_label,
			epc.section_code,
			epc.source_scope,
			epc.mandatory,
			epc.status,
			epc.document_count,
			to_char(epc.last_checked_on, 'YYYY-MM-DD'),
			epc.checked_by,
			epc.institution_id,
			epc.notes
		from education_portfolio_checklist epc
		%s
		order by %s %s, epc.id asc
		limit $%d offset $%d
	`, whereClause, portfolioChecklistSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_checklist_failed"})
		return
	}
	defer rows.Close()

	items := make([]PortfolioChecklistItem, 0, query.PageSize)
	for rows.Next() {
		var item PortfolioChecklistItem
		if err := rows.Scan(
			&item.ID,
			&item.PortfolioID,
			&item.RequirementCode,
			&item.RequirementLabel,
			&item.SectionCode,
			&item.SourceScope,
			&item.Mandatory,
			&item.Status,
			&item.DocumentCount,
			&item.LastCheckedOn,
			&item.CheckedBy,
			&item.InstitutionID,
			&item.Notes,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_checklist_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_checklist_scan_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) PortfolioChecklistItemDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var item PortfolioChecklistItem
	err := s.pool.QueryRow(r.Context(), `
		select
			epc.id::text,
			epc.portfolio_id::text,
			epc.requirement_code,
			epc.requirement_label,
			epc.section_code,
			epc.source_scope,
			epc.mandatory,
			epc.status,
			epc.document_count,
			to_char(epc.last_checked_on, 'YYYY-MM-DD'),
			epc.checked_by,
			epc.institution_id,
			epc.notes
		from education_portfolio_checklist epc
		where epc.id = $1
			and epc.portfolio_id = $2
			and epc.institution_id = $3
	`, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID,
		&item.PortfolioID,
		&item.RequirementCode,
		&item.RequirementLabel,
		&item.SectionCode,
		&item.SourceScope,
		&item.Mandatory,
		&item.Status,
		&item.DocumentCount,
		&item.LastCheckedOn,
		&item.CheckedBy,
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_portfolio_checklist_item_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_checklist_item_detail_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreatePortfolioChecklistItem(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreatePortfolioChecklistItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_checklist_payload"})
		return
	}

	normalizePortfolioChecklistRequest(&req)
	if req.RequirementCode == "" || req.RequirementLabel == "" || req.SectionCode == "" || req.SourceScope == "" || req.Status == "" || req.LastCheckedOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_portfolio_checklist_fields"})
		return
	}
	if req.DocumentCount < 0 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_checklist_document_count"})
		return
	}
	if !containsString([]string{"portofoliu", "dosar_personal"}, req.SourceScope) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_checklist_source_scope"})
		return
	}
	if !containsString([]string{"complet", "partial", "lipsa", "in_verificare"}, req.Status) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_checklist_status"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.LastCheckedOn); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_checklist_last_checked_on"})
		return
	}

	var item PortfolioChecklistItem
	err := s.pool.QueryRow(r.Context(), `
		insert into education_portfolio_checklist (
			portfolio_id,
			requirement_code,
			requirement_label,
			section_code,
			source_scope,
			mandatory,
			status,
			document_count,
			last_checked_on,
			checked_by,
			institution_id,
			notes
		)
		select
			ep.id,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10,
			ep.institution_id,
			$11
		from education_portfolios ep
		where ep.id = $1 and ep.institution_id = $12
		returning
			id::text,
			portfolio_id::text,
			requirement_code,
			requirement_label,
			section_code,
			source_scope,
			mandatory,
			status,
			document_count,
			to_char(last_checked_on, 'YYYY-MM-DD'),
			checked_by,
			institution_id,
			notes
	`, recordID, req.RequirementCode, req.RequirementLabel, req.SectionCode, req.SourceScope, req.Mandatory, req.Status, req.DocumentCount, req.LastCheckedOn, req.CheckedBy, req.Notes, s.institutionID(r)).Scan(
		&item.ID,
		&item.PortfolioID,
		&item.RequirementCode,
		&item.RequirementLabel,
		&item.SectionCode,
		&item.SourceScope,
		&item.Mandatory,
		&item.Status,
		&item.DocumentCount,
		&item.LastCheckedOn,
		&item.CheckedBy,
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_portfolio_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_checklist_create_failed"})
		return
	}

	s.logAudit(r, "education.portfolios.checklist.create", "portfolio_checklist_item", item.ID, "Portfolio checklist item created.", map[string]any{
		"portfolio_id":     item.PortfolioID,
		"requirement_code": item.RequirementCode,
		"section_code":     item.SectionCode,
		"source_scope":     item.SourceScope,
		"status":           item.Status,
		"document_count":   item.DocumentCount,
	})

	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdatePortfolioChecklistItem(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var req CreatePortfolioChecklistItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_checklist_payload"})
		return
	}

	normalizePortfolioChecklistRequest(&req)
	if req.RequirementCode == "" || req.RequirementLabel == "" || req.SectionCode == "" || req.SourceScope == "" || req.Status == "" || req.LastCheckedOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_portfolio_checklist_fields"})
		return
	}
	if req.DocumentCount < 0 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_checklist_document_count"})
		return
	}
	if !containsString([]string{"portofoliu", "dosar_personal"}, req.SourceScope) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_checklist_source_scope"})
		return
	}
	if !containsString([]string{"complet", "partial", "lipsa", "in_verificare"}, req.Status) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_checklist_status"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.LastCheckedOn); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_checklist_last_checked_on"})
		return
	}

	var item PortfolioChecklistItem
	err := s.pool.QueryRow(r.Context(), `
		update education_portfolio_checklist
		set
			requirement_code = $1,
			requirement_label = $2,
			section_code = $3,
			source_scope = $4,
			mandatory = $5,
			status = $6,
			document_count = $7,
			last_checked_on = $8,
			checked_by = $9,
			notes = $10,
			updated_at = now()
		where id = $11
			and portfolio_id = $12
			and institution_id = $13
		returning
			id::text,
			portfolio_id::text,
			requirement_code,
			requirement_label,
			section_code,
			source_scope,
			mandatory,
			status,
			document_count,
			to_char(last_checked_on, 'YYYY-MM-DD'),
			checked_by,
			institution_id,
			notes
	`, req.RequirementCode, req.RequirementLabel, req.SectionCode, req.SourceScope, req.Mandatory, req.Status, req.DocumentCount, req.LastCheckedOn, req.CheckedBy, req.Notes, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID,
		&item.PortfolioID,
		&item.RequirementCode,
		&item.RequirementLabel,
		&item.SectionCode,
		&item.SourceScope,
		&item.Mandatory,
		&item.Status,
		&item.DocumentCount,
		&item.LastCheckedOn,
		&item.CheckedBy,
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_portfolio_checklist_item_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_checklist_update_failed"})
		return
	}

	s.logAudit(r, "education.portfolios.checklist.update", "portfolio_checklist_item", item.ID, "Portfolio checklist item updated.", map[string]any{
		"portfolio_id":     item.PortfolioID,
		"requirement_code": item.RequirementCode,
		"section_code":     item.SectionCode,
		"source_scope":     item.SourceScope,
		"status":           item.Status,
		"document_count":   item.DocumentCount,
	})

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeletePortfolioChecklistItem(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	tag, err := s.pool.Exec(r.Context(), `
		delete from education_portfolio_checklist
		where id = $1 and portfolio_id = $2 and institution_id = $3
	`, itemID, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_checklist_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_portfolio_checklist_item_not_found")
		return
	}

	s.logAudit(r, "education.portfolios.checklist.delete", "portfolio_checklist_item", itemID, "Portfolio checklist item deleted.", map[string]any{
		"portfolio_id": recordID,
	})

	w.WriteHeader(http.StatusNoContent)
}

func buildMeetingParticipantFilters(filters map[string]string, meetingID string, institutionID string) (string, []any) {
	where := []string{"emp.meeting_id = $1", "emp.institution_id = $2"}
	args := []any{meetingID, institutionID}

	addContains := func(column string, value string) {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(value))+"%")
		where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
	}
	addBoolean := func(column string, value string) {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "da" {
			normalized = "true"
		}
		if normalized == "nu" {
			normalized = "false"
		}
		if normalized != "true" && normalized != "false" {
			return
		}
		args = append(args, normalized == "true")
		where = append(where, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	if value := filters["full_name"]; value != "" {
		addContains("emp.full_name", value)
	}
	if value := filters["role_name"]; value != "" {
		addContains("emp.role_name", value)
	}
	if value := filters["member_type"]; value != "" {
		addContains("emp.member_type", value)
	}
	if value := filters["attendance_status"]; value != "" {
		addContains("emp.attendance_status", value)
	}
	if value := filters["signature_present"]; value != "" {
		addBoolean("emp.signature_present", value)
	}
	if value := filters["voting_right"]; value != "" {
		addBoolean("emp.voting_right", value)
	}

	return "where " + strings.Join(where, " and "), args
}

func buildMeetingDocumentFilters(filters map[string]string, meetingID string, institutionID string) (string, []any) {
	where := []string{"emd.meeting_id = $1", "emd.institution_id = $2"}
	args := []any{meetingID, institutionID}

	addContains := func(column string, value string) {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(value))+"%")
		where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
	}

	if value := filters["document_type"]; value != "" {
		addContains("emd.document_type", value)
	}
	if value := filters["title"]; value != "" {
		addContains("emd.title", value)
	}
	if value := filters["document_number"]; value != "" {
		addContains("emd.document_number", value)
	}
	if value := filters["registry_number"]; value != "" {
		addContains("emd.registry_number", value)
	}
	if value := filters["publication_status"]; value != "" {
		addContains("emd.publication_status", value)
	}
	if value := filters["issued_on"]; value != "" {
		addContains("to_char(emd.issued_on, 'YYYY-MM-DD')", value)
	}
	if value := filters["custody_owner"]; value != "" {
		addContains("emd.custody_owner", value)
	}

	return "where " + strings.Join(where, " and "), args
}

func buildPortfolioDocumentFilters(filters map[string]string, recordID string, institutionID string) (string, []any) {
	where := []string{"epd.portfolio_id = $1", "epd.institution_id = $2", "epd.status = 'active'"}
	args := []any{recordID, institutionID}

	addContains := func(column string, value string) {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(value))+"%")
		where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
	}
	addBoolean := func(column string, value string) {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "da" {
			normalized = "true"
		}
		if normalized == "nu" {
			normalized = "false"
		}
		if normalized != "true" && normalized != "false" {
			return
		}
		args = append(args, normalized == "true")
		where = append(where, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	if value := filters["section_code"]; value != "" {
		addContains("epd.section_code", value)
	}
	if value := filters["component_code"]; value != "" {
		addContains("epd.component_code", value)
	}
	if value := filters["document_title"]; value != "" {
		addContains("epd.document_title", value)
	}
	if value := filters["description"]; value != "" {
		addContains("epd.description", value)
	}
	if value := filters["school_year"]; value != "" {
		addContains("epd.school_year", value)
	}
	if value := filters["subject_discipline"]; value != "" {
		addContains("epd.subject_discipline", value)
	}
	if value := filters["applicable_class"]; value != "" {
		addContains("epd.applicable_class", value)
	}
	if value := filters["competencies"]; value != "" {
		addContains("array_to_string(epd.competencies, ' ')", value)
	}
	if value := filters["source_scope"]; value != "" {
		addContains("epd.source_scope", value)
	}
	if value := filters["evidence_type"]; value != "" {
		addContains("epd.evidence_type", value)
	}
	if value := filters["issued_on"]; value != "" {
		addContains("to_char(epd.issued_on, 'YYYY-MM-DD')", value)
	}
	if value := filters["chronological_index"]; value != "" {
		args = append(args, "%"+strings.TrimSpace(value)+"%")
		where = append(where, fmt.Sprintf("epd.chronological_index::text like $%d", len(args)))
	}
	if value := filters["sensitive_data"]; value != "" {
		addBoolean("epd.sensitive_data", value)
	}
	if value := filters["authenticity_status"]; value != "" {
		addContains("epd.authenticity_status", value)
	}
	if value := filters["archive_version_no"]; value != "" {
		args = append(args, "%"+strings.TrimSpace(value)+"%")
		where = append(where, fmt.Sprintf("epd.archive_version_no::text like $%d", len(args)))
	}
	if value := filters["archive_sha256"]; value != "" {
		addContains("epd.archive_sha256", value)
	}

	return "where " + strings.Join(where, " and "), args
}

func buildMeetingVoteFilters(filters map[string]string, meetingID string, institutionID string) (string, []any) {
	where := []string{"emv.meeting_id = $1", "emv.institution_id = $2"}
	args := []any{meetingID, institutionID}

	addContains := func(column string, value string) {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(value))+"%")
		where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
	}
	addNumber := func(column string, value string) {
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			return
		}
		args = append(args, "%"+normalized+"%")
		where = append(where, fmt.Sprintf("%s::text like $%d", column, len(args)))
	}
	addBoolean := func(column string, value string) {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "da" {
			normalized = "true"
		}
		if normalized == "nu" {
			normalized = "false"
		}
		if normalized != "true" && normalized != "false" {
			return
		}
		args = append(args, normalized == "true")
		where = append(where, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	if value := filters["subject_title"]; value != "" {
		addContains("emv.subject_title", value)
	}
	if value := filters["agenda_order"]; value != "" {
		addNumber("emv.agenda_order", value)
	}
	if value := filters["decision_type"]; value != "" {
		addContains("emv.decision_type", value)
	}
	if value := filters["outcome"]; value != "" {
		addContains("emv.outcome", value)
	}
	if value := filters["requires_follow_up"]; value != "" {
		addBoolean("emv.requires_follow_up", value)
	}

	return "where " + strings.Join(where, " and "), args
}

func buildPortfolioChecklistFilters(filters map[string]string, recordID string, institutionID string) (string, []any) {
	where := []string{"epc.portfolio_id = $1", "epc.institution_id = $2"}
	args := []any{recordID, institutionID}

	addContains := func(column string, value string) {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(value))+"%")
		where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
	}
	addBoolean := func(column string, value string) {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "da" {
			normalized = "true"
		}
		if normalized == "nu" {
			normalized = "false"
		}
		if normalized != "true" && normalized != "false" {
			return
		}
		args = append(args, normalized == "true")
		where = append(where, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	if value := filters["requirement_code"]; value != "" {
		addContains("epc.requirement_code", value)
	}
	if value := filters["section_code"]; value != "" {
		addContains("epc.section_code", value)
	}
	if value := filters["source_scope"]; value != "" {
		addContains("epc.source_scope", value)
	}
	if value := filters["status"]; value != "" {
		addContains("epc.status", value)
	}
	if value := filters["mandatory"]; value != "" {
		addBoolean("epc.mandatory", value)
	}
	if value := filters["checked_by"]; value != "" {
		addContains("epc.checked_by", value)
	}

	return "where " + strings.Join(where, " and "), args
}

func meetingParticipantSortColumn(value string) string {
	switch value {
	case "full_name":
		return "emp.full_name"
	case "role_name":
		return "emp.role_name"
	case "member_type":
		return "emp.member_type"
	case "attendance_status":
		return "emp.attendance_status"
	case "signature_present":
		return "emp.signature_present"
	case "voting_right":
		return "emp.voting_right"
	default:
		return "emp.full_name"
	}
}

func meetingDocumentSortColumn(value string) string {
	switch value {
	case "document_type":
		return "emd.document_type"
	case "title":
		return "emd.title"
	case "document_number":
		return "emd.document_number"
	case "registry_number":
		return "emd.registry_number"
	case "publication_status":
		return "emd.publication_status"
	case "issued_on":
		return "emd.issued_on"
	case "custody_owner":
		return "emd.custody_owner"
	default:
		return "emd.issued_on"
	}
}

func portfolioDocumentSortColumn(value string) string {
	switch value {
	case "section_code":
		return "epd.section_code"
	case "document_title":
		return "epd.document_title"
	case "description":
		return "epd.description"
	case "school_year":
		return "epd.school_year"
	case "subject_discipline":
		return "epd.subject_discipline"
	case "applicable_class":
		return "epd.applicable_class"
	case "source_scope":
		return "epd.source_scope"
	case "evidence_type":
		return "epd.evidence_type"
	case "issued_on":
		return "epd.issued_on"
	case "authenticity_status":
		return "epd.authenticity_status"
	case "chronological_index":
		return "epd.chronological_index"
	case "sensitive_data":
		return "epd.sensitive_data"
	case "archive_version_no":
		return "epd.archive_version_no"
	default:
		return "epd.issued_on"
	}
}

func meetingVoteSortColumn(value string) string {
	switch value {
	case "subject_title":
		return "emv.subject_title"
	case "agenda_order":
		return "emv.agenda_order"
	case "decision_type":
		return "emv.decision_type"
	case "outcome":
		return "emv.outcome"
	case "votes_for":
		return "emv.votes_for"
	case "votes_against":
		return "emv.votes_against"
	case "abstentions":
		return "emv.abstentions"
	default:
		return "emv.agenda_order"
	}
}

func portfolioChecklistSortColumn(value string) string {
	switch value {
	case "requirement_code":
		return "epc.requirement_code"
	case "requirement_label":
		return "epc.requirement_label"
	case "section_code":
		return "epc.section_code"
	case "status":
		return "epc.status"
	case "document_count":
		return "epc.document_count"
	default:
		return "epc.requirement_code"
	}
}

func containsString(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func normalizePortfolioDocumentRequest(req *CreatePortfolioDocumentRequest) {
	req.SectionCode = strings.TrimSpace(req.SectionCode)
	req.ComponentCode = strings.TrimSpace(req.ComponentCode)
	req.DocumentTitle = strings.TrimSpace(req.DocumentTitle)
	req.Description = strings.TrimSpace(req.Description)
	req.SchoolYear = strings.TrimSpace(req.SchoolYear)
	req.SubjectDiscipline = strings.TrimSpace(req.SubjectDiscipline)
	req.ApplicableClass = strings.TrimSpace(req.ApplicableClass)
	competencies := make([]string, 0, len(req.Competencies))
	seen := make(map[string]struct{}, len(req.Competencies))
	for _, competency := range req.Competencies {
		competency = strings.TrimSpace(competency)
		key := strings.ToLower(competency)
		if competency == "" {
			continue
		}
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		competencies = append(competencies, competency)
	}
	req.Competencies = competencies
	req.SourceScope = strings.TrimSpace(req.SourceScope)
	req.EvidenceType = strings.TrimSpace(req.EvidenceType)
	req.IssuedOn = strings.TrimSpace(req.IssuedOn)
	req.AddedOn = strings.TrimSpace(req.AddedOn)
	req.AuthenticityStatus = strings.TrimSpace(req.AuthenticityStatus)
	req.FileReference = strings.TrimSpace(req.FileReference)
	req.Notes = strings.TrimSpace(req.Notes)
}

func portfolioArchiveReference(value string) (string, string, bool) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(strings.ToLower(value), "archive://") {
		return "", "", false
	}
	documentID := strings.TrimSpace(value[len("archive://"):])
	parsed, err := uuid.Parse(documentID)
	if err != nil || parsed == uuid.Nil || parsed.String() != strings.ToLower(documentID) {
		return "", "", false
	}
	return "archive://" + parsed.String(), parsed.String(), true
}

func normalizeMeetingVoteRequest(req *CreateGovernanceMeetingVoteRequest) {
	req.SubjectTitle = strings.TrimSpace(req.SubjectTitle)
	req.DecisionType = strings.TrimSpace(req.DecisionType)
	req.Outcome = strings.TrimSpace(req.Outcome)
	req.LegalBasis = strings.TrimSpace(req.LegalBasis)
	req.Notes = strings.TrimSpace(req.Notes)
}

func normalizePortfolioChecklistRequest(req *CreatePortfolioChecklistItemRequest) {
	req.RequirementCode = strings.TrimSpace(req.RequirementCode)
	req.RequirementLabel = strings.TrimSpace(req.RequirementLabel)
	req.SectionCode = strings.TrimSpace(req.SectionCode)
	req.SourceScope = strings.TrimSpace(req.SourceScope)
	req.Status = strings.TrimSpace(req.Status)
	req.LastCheckedOn = strings.TrimSpace(req.LastCheckedOn)
	req.CheckedBy = strings.TrimSpace(req.CheckedBy)
	req.Notes = strings.TrimSpace(req.Notes)
}
