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

func (s *Service) PersonnelAssignments(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"assignment_code":    {},
		"assignment_type":    {},
		"assignment_title":   {},
		"status":             {},
		"decision_reference": {},
	}, []string{"assignment_code", "assignment_type", "assignment_title", "status", "assigned_on", "ended_on", "weekly_hours"})
	if query.Sort == "" {
		query.Sort = "assigned_on"
	}

	whereClause, args := buildPersonnelAssignmentFilters(query.Filters, recordID, s.institutionID(r))
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_personnel_assignments epa "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_assignments_failed"})
		return
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select
			epa.id::text, epa.personnel_id::text, epa.assignment_code, epa.assignment_type, epa.assignment_title,
			epa.status, to_char(epa.assigned_on, 'YYYY-MM-DD'), coalesce(to_char(epa.ended_on, 'YYYY-MM-DD'), ''),
			epa.weekly_hours, epa.decision_reference, epa.institution_id, epa.notes
		from education_personnel_assignments epa
		%s
		order by %s %s, epa.assignment_code desc
		limit $%d offset $%d
	`, whereClause, personnelAssignmentSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_assignments_failed"})
		return
	}
	defer rows.Close()

	items := make([]PersonnelAssignment, 0, query.PageSize)
	for rows.Next() {
		var item PersonnelAssignment
		if err := rows.Scan(
			&item.ID, &item.PersonnelID, &item.AssignmentCode, &item.AssignmentType, &item.AssignmentTitle,
			&item.Status, &item.AssignedOn, &item.EndedOn, &item.WeeklyHours, &item.DecisionReference, &item.InstitutionID, &item.Notes,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_assignments_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_assignments_scan_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) PersonnelAssignmentDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var item PersonnelAssignment
	err := s.pool.QueryRow(r.Context(), `
		select
			id::text, personnel_id::text, assignment_code, assignment_type, assignment_title,
			status, to_char(assigned_on, 'YYYY-MM-DD'), coalesce(to_char(ended_on, 'YYYY-MM-DD'), ''),
			weekly_hours, decision_reference, institution_id, notes
		from education_personnel_assignments
		where id = $1 and personnel_id = $2 and institution_id = $3
	`, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID, &item.PersonnelID, &item.AssignmentCode, &item.AssignmentType, &item.AssignmentTitle,
		&item.Status, &item.AssignedOn, &item.EndedOn, &item.WeeklyHours, &item.DecisionReference, &item.InstitutionID, &item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_personnel_assignment_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_assignment_detail_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreatePersonnelAssignment(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreatePersonnelAssignmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_assignment_payload"})
		return
	}
	normalizePersonnelAssignmentRequest(&req)
	if req.AssignmentType == "" || req.AssignmentTitle == "" || req.Status == "" || req.AssignedOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_personnel_assignment_fields"})
		return
	}
	if !inStringSet(req.AssignmentType, "diriginte", "coordonator_proiect", "responsabil_comisie", "mentor", "membru_comisie", "administrator_structura") ||
		!inStringSet(req.Status, "propus", "activ", "suspendat", "incetat") ||
		req.WeeklyHours < 0 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_assignment_fields"})
		return
	}
	assignedOn, err := parseRequiredEducationDate(req.AssignedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_assignment_assigned_on"})
		return
	}
	endedOn, err := parseOptionalEducationDate(req.EndedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_assignment_ended_on"})
		return
	}
	if endedOn != nil && endedOn.Before(*assignedOn) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_assignment_interval"})
		return
	}

	code := fmt.Sprintf("ATR-%d-%05d", time.Now().UTC().Year(), time.Now().UTC().UnixNano()%100000)
	var item PersonnelAssignment
	err = s.pool.QueryRow(r.Context(), `
		insert into education_personnel_assignments (
			personnel_id, assignment_code, assignment_type, assignment_title, status,
			assigned_on, ended_on, weekly_hours, decision_reference, institution_id, notes
		) values ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		returning
			id::text, personnel_id::text, assignment_code, assignment_type, assignment_title,
			status, to_char(assigned_on, 'YYYY-MM-DD'), coalesce(to_char(ended_on, 'YYYY-MM-DD'), ''),
			weekly_hours, decision_reference, institution_id, notes
	`, recordID, code, req.AssignmentType, req.AssignmentTitle, req.Status, assignedOn, endedOn, req.WeeklyHours, req.DecisionReference, s.institutionID(r), req.Notes).Scan(
		&item.ID, &item.PersonnelID, &item.AssignmentCode, &item.AssignmentType, &item.AssignmentTitle,
		&item.Status, &item.AssignedOn, &item.EndedOn, &item.WeeklyHours, &item.DecisionReference, &item.InstitutionID, &item.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "personnel_assignment_create_failed"})
		return
	}

	s.logAudit(r, "education.personnel.assignment.create", "personnel_assignment", item.ID, "Personnel assignment created.", map[string]any{
		"personnel_id":      recordID,
		"assignment_code":   item.AssignmentCode,
		"assignment_type":   item.AssignmentType,
		"assignment_status": item.Status,
	})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdatePersonnelAssignment(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var req CreatePersonnelAssignmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_assignment_payload"})
		return
	}
	normalizePersonnelAssignmentRequest(&req)
	if req.AssignmentType == "" || req.AssignmentTitle == "" || req.Status == "" || req.AssignedOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_personnel_assignment_fields"})
		return
	}
	if !inStringSet(req.AssignmentType, "diriginte", "coordonator_proiect", "responsabil_comisie", "mentor", "membru_comisie", "administrator_structura") ||
		!inStringSet(req.Status, "propus", "activ", "suspendat", "incetat") ||
		req.WeeklyHours < 0 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_assignment_fields"})
		return
	}
	assignedOn, err := parseRequiredEducationDate(req.AssignedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_assignment_assigned_on"})
		return
	}
	endedOn, err := parseOptionalEducationDate(req.EndedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_assignment_ended_on"})
		return
	}
	if endedOn != nil && endedOn.Before(*assignedOn) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_assignment_interval"})
		return
	}

	var item PersonnelAssignment
	err = s.pool.QueryRow(r.Context(), `
		update education_personnel_assignments
		set
			assignment_type = $1,
			assignment_title = $2,
			status = $3,
			assigned_on = $4,
			ended_on = $5,
			weekly_hours = $6,
			decision_reference = $7,
			notes = $8,
			updated_at = now()
		where id = $9 and personnel_id = $10 and institution_id = $11
		returning
			id::text, personnel_id::text, assignment_code, assignment_type, assignment_title,
			status, to_char(assigned_on, 'YYYY-MM-DD'), coalesce(to_char(ended_on, 'YYYY-MM-DD'), ''),
			weekly_hours, decision_reference, institution_id, notes
	`, req.AssignmentType, req.AssignmentTitle, req.Status, assignedOn, endedOn, req.WeeklyHours, req.DecisionReference, req.Notes, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID, &item.PersonnelID, &item.AssignmentCode, &item.AssignmentType, &item.AssignmentTitle,
		&item.Status, &item.AssignedOn, &item.EndedOn, &item.WeeklyHours, &item.DecisionReference, &item.InstitutionID, &item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_personnel_assignment_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "personnel_assignment_update_failed"})
		return
	}

	s.logAudit(r, "education.personnel.assignment.update", "personnel_assignment", item.ID, "Personnel assignment updated.", map[string]any{
		"personnel_id":      recordID,
		"assignment_code":   item.AssignmentCode,
		"assignment_type":   item.AssignmentType,
		"assignment_status": item.Status,
	})
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeletePersonnelAssignment(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	tag, err := s.pool.Exec(r.Context(), `delete from education_personnel_assignments where id = $1 and personnel_id = $2 and institution_id = $3`, itemID, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "personnel_assignment_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_personnel_assignment_not_found")
		return
	}

	s.logAudit(r, "education.personnel.assignment.delete", "personnel_assignment", itemID, "Personnel assignment deleted.", map[string]any{"personnel_id": recordID})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) PersonnelDisciplinaryCases(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"case_code":      {},
		"case_type":      {},
		"status":         {},
		"committee_name": {},
		"sanction":       {},
		"legal_basis":    {},
	}, []string{"case_code", "case_type", "status", "reported_on", "hearing_on", "resolved_on", "committee_name"})
	if query.Sort == "" {
		query.Sort = "reported_on"
	}

	whereClause, args := buildPersonnelDisciplinaryFilters(query.Filters, recordID, s.institutionID(r))
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_personnel_disciplinary_cases epdc "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_disciplinary_cases_failed"})
		return
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select
			epdc.id::text, epdc.personnel_id::text, epdc.case_code, epdc.case_type, epdc.status,
			to_char(epdc.reported_on, 'YYYY-MM-DD'),
			coalesce(to_char(epdc.hearing_on, 'YYYY-MM-DD'), ''),
			coalesce(to_char(epdc.resolved_on, 'YYYY-MM-DD'), ''),
			epdc.committee_name, epdc.sanction, epdc.legal_basis, epdc.institution_id, epdc.notes
		from education_personnel_disciplinary_cases epdc
		%s
		order by %s %s, epdc.case_code desc
		limit $%d offset $%d
	`, whereClause, personnelDisciplinarySortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_disciplinary_cases_failed"})
		return
	}
	defer rows.Close()

	items := make([]PersonnelDisciplinaryCase, 0, query.PageSize)
	for rows.Next() {
		var item PersonnelDisciplinaryCase
		if err := rows.Scan(
			&item.ID, &item.PersonnelID, &item.CaseCode, &item.CaseType, &item.Status,
			&item.ReportedOn, &item.HearingOn, &item.ResolvedOn, &item.CommitteeName,
			&item.Sanction, &item.LegalBasis, &item.InstitutionID, &item.Notes,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_disciplinary_cases_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_disciplinary_cases_scan_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) PersonnelDisciplinaryCaseDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var item PersonnelDisciplinaryCase
	err := s.pool.QueryRow(r.Context(), `
		select
			id::text, personnel_id::text, case_code, case_type, status,
			to_char(reported_on, 'YYYY-MM-DD'),
			coalesce(to_char(hearing_on, 'YYYY-MM-DD'), ''),
			coalesce(to_char(resolved_on, 'YYYY-MM-DD'), ''),
			committee_name, sanction, legal_basis, institution_id, notes
		from education_personnel_disciplinary_cases
		where id = $1 and personnel_id = $2 and institution_id = $3
	`, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID, &item.PersonnelID, &item.CaseCode, &item.CaseType, &item.Status,
		&item.ReportedOn, &item.HearingOn, &item.ResolvedOn, &item.CommitteeName,
		&item.Sanction, &item.LegalBasis, &item.InstitutionID, &item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_personnel_disciplinary_case_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_disciplinary_case_detail_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreatePersonnelDisciplinaryCase(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreatePersonnelDisciplinaryCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_disciplinary_case_payload"})
		return
	}
	normalizePersonnelDisciplinaryRequest(&req)
	if req.CaseType == "" || req.Status == "" || req.ReportedOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_personnel_disciplinary_case_fields"})
		return
	}
	if !inStringSet(req.CaseType, "sesizare", "cercetare", "sanctiune", "contestatie") ||
		!inStringSet(req.Status, "deschis", "in_cercetare", "solutionat", "contestat", "inchis") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_disciplinary_case_fields"})
		return
	}
	reportedOn, err := parseRequiredEducationDate(req.ReportedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_disciplinary_reported_on"})
		return
	}
	hearingOn, err := parseOptionalEducationDate(req.HearingOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_disciplinary_hearing_on"})
		return
	}
	resolvedOn, err := parseOptionalEducationDate(req.ResolvedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_disciplinary_resolved_on"})
		return
	}
	if hearingOn != nil && hearingOn.Before(*reportedOn) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_disciplinary_hearing_interval"})
		return
	}
	if resolvedOn != nil && resolvedOn.Before(*reportedOn) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_disciplinary_resolved_interval"})
		return
	}

	code := fmt.Sprintf("DISC-%d-%05d", time.Now().UTC().Year(), time.Now().UTC().UnixNano()%100000)
	var item PersonnelDisciplinaryCase
	err = s.pool.QueryRow(r.Context(), `
		insert into education_personnel_disciplinary_cases (
			personnel_id, case_code, case_type, status, reported_on, hearing_on,
			resolved_on, committee_name, sanction, legal_basis, institution_id, notes
		) values ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		returning
			id::text, personnel_id::text, case_code, case_type, status,
			to_char(reported_on, 'YYYY-MM-DD'),
			coalesce(to_char(hearing_on, 'YYYY-MM-DD'), ''),
			coalesce(to_char(resolved_on, 'YYYY-MM-DD'), ''),
			committee_name, sanction, legal_basis, institution_id, notes
	`, recordID, code, req.CaseType, req.Status, reportedOn, hearingOn, resolvedOn, req.CommitteeName, req.Sanction, req.LegalBasis, s.institutionID(r), req.Notes).Scan(
		&item.ID, &item.PersonnelID, &item.CaseCode, &item.CaseType, &item.Status,
		&item.ReportedOn, &item.HearingOn, &item.ResolvedOn, &item.CommitteeName,
		&item.Sanction, &item.LegalBasis, &item.InstitutionID, &item.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "personnel_disciplinary_case_create_failed"})
		return
	}

	s.logAudit(r, "education.personnel.disciplinary.create", "personnel_disciplinary_case", item.ID, "Personnel disciplinary case created.", map[string]any{
		"personnel_id": recordID,
		"case_code":    item.CaseCode,
		"case_type":    item.CaseType,
		"status":       item.Status,
	})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdatePersonnelDisciplinaryCase(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var req CreatePersonnelDisciplinaryCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_disciplinary_case_payload"})
		return
	}
	normalizePersonnelDisciplinaryRequest(&req)
	if req.CaseType == "" || req.Status == "" || req.ReportedOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_personnel_disciplinary_case_fields"})
		return
	}
	if !inStringSet(req.CaseType, "sesizare", "cercetare", "sanctiune", "contestatie") ||
		!inStringSet(req.Status, "deschis", "in_cercetare", "solutionat", "contestat", "inchis") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_disciplinary_case_fields"})
		return
	}
	reportedOn, err := parseRequiredEducationDate(req.ReportedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_disciplinary_reported_on"})
		return
	}
	hearingOn, err := parseOptionalEducationDate(req.HearingOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_disciplinary_hearing_on"})
		return
	}
	resolvedOn, err := parseOptionalEducationDate(req.ResolvedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_disciplinary_resolved_on"})
		return
	}
	if hearingOn != nil && hearingOn.Before(*reportedOn) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_disciplinary_hearing_interval"})
		return
	}
	if resolvedOn != nil && resolvedOn.Before(*reportedOn) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_disciplinary_resolved_interval"})
		return
	}

	var item PersonnelDisciplinaryCase
	err = s.pool.QueryRow(r.Context(), `
		update education_personnel_disciplinary_cases
		set
			case_type = $1,
			status = $2,
			reported_on = $3,
			hearing_on = $4,
			resolved_on = $5,
			committee_name = $6,
			sanction = $7,
			legal_basis = $8,
			notes = $9,
			updated_at = now()
		where id = $10 and personnel_id = $11 and institution_id = $12
		returning
			id::text, personnel_id::text, case_code, case_type, status,
			to_char(reported_on, 'YYYY-MM-DD'),
			coalesce(to_char(hearing_on, 'YYYY-MM-DD'), ''),
			coalesce(to_char(resolved_on, 'YYYY-MM-DD'), ''),
			committee_name, sanction, legal_basis, institution_id, notes
	`, req.CaseType, req.Status, reportedOn, hearingOn, resolvedOn, req.CommitteeName, req.Sanction, req.LegalBasis, req.Notes, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID, &item.PersonnelID, &item.CaseCode, &item.CaseType, &item.Status,
		&item.ReportedOn, &item.HearingOn, &item.ResolvedOn, &item.CommitteeName,
		&item.Sanction, &item.LegalBasis, &item.InstitutionID, &item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_personnel_disciplinary_case_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "personnel_disciplinary_case_update_failed"})
		return
	}

	s.logAudit(r, "education.personnel.disciplinary.update", "personnel_disciplinary_case", item.ID, "Personnel disciplinary case updated.", map[string]any{
		"personnel_id": recordID,
		"case_code":    item.CaseCode,
		"case_type":    item.CaseType,
		"status":       item.Status,
	})
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeletePersonnelDisciplinaryCase(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	tag, err := s.pool.Exec(r.Context(), `delete from education_personnel_disciplinary_cases where id = $1 and personnel_id = $2 and institution_id = $3`, itemID, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "personnel_disciplinary_case_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_personnel_disciplinary_case_not_found")
		return
	}

	s.logAudit(r, "education.personnel.disciplinary.delete", "personnel_disciplinary_case", itemID, "Personnel disciplinary case deleted.", map[string]any{"personnel_id": recordID})
	w.WriteHeader(http.StatusNoContent)
}

func normalizePersonnelAssignmentRequest(req *CreatePersonnelAssignmentRequest) {
	req.AssignmentType = strings.TrimSpace(req.AssignmentType)
	req.AssignmentTitle = strings.TrimSpace(req.AssignmentTitle)
	req.Status = strings.TrimSpace(req.Status)
	req.AssignedOn = strings.TrimSpace(req.AssignedOn)
	req.EndedOn = strings.TrimSpace(req.EndedOn)
	req.DecisionReference = strings.TrimSpace(req.DecisionReference)
	req.Notes = strings.TrimSpace(req.Notes)
}

func normalizePersonnelDisciplinaryRequest(req *CreatePersonnelDisciplinaryCaseRequest) {
	req.CaseType = strings.TrimSpace(req.CaseType)
	req.Status = strings.TrimSpace(req.Status)
	req.ReportedOn = strings.TrimSpace(req.ReportedOn)
	req.HearingOn = strings.TrimSpace(req.HearingOn)
	req.ResolvedOn = strings.TrimSpace(req.ResolvedOn)
	req.CommitteeName = strings.TrimSpace(req.CommitteeName)
	req.Sanction = strings.TrimSpace(req.Sanction)
	req.LegalBasis = strings.TrimSpace(req.LegalBasis)
	req.Notes = strings.TrimSpace(req.Notes)
}

func buildPersonnelAssignmentFilters(filters map[string]string, recordID string, institutionID string) (string, []any) {
	where := []string{"epa.personnel_id = $1", "epa.institution_id = $2"}
	args := []any{recordID, institutionID}
	addContains := func(column string, value string) {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(value))+"%")
		where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
	}
	if value := filters["assignment_code"]; value != "" {
		addContains("epa.assignment_code", value)
	}
	if value := filters["assignment_type"]; value != "" {
		addContains("epa.assignment_type", value)
	}
	if value := filters["assignment_title"]; value != "" {
		addContains("epa.assignment_title", value)
	}
	if value := filters["status"]; value != "" {
		addContains("epa.status", value)
	}
	if value := filters["decision_reference"]; value != "" {
		addContains("epa.decision_reference", value)
	}
	return "where " + strings.Join(where, " and "), args
}

func buildPersonnelDisciplinaryFilters(filters map[string]string, recordID string, institutionID string) (string, []any) {
	where := []string{"epdc.personnel_id = $1", "epdc.institution_id = $2"}
	args := []any{recordID, institutionID}
	addContains := func(column string, value string) {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(value))+"%")
		where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
	}
	if value := filters["case_code"]; value != "" {
		addContains("epdc.case_code", value)
	}
	if value := filters["case_type"]; value != "" {
		addContains("epdc.case_type", value)
	}
	if value := filters["status"]; value != "" {
		addContains("epdc.status", value)
	}
	if value := filters["committee_name"]; value != "" {
		addContains("epdc.committee_name", value)
	}
	if value := filters["sanction"]; value != "" {
		addContains("epdc.sanction", value)
	}
	if value := filters["legal_basis"]; value != "" {
		addContains("epdc.legal_basis", value)
	}
	return "where " + strings.Join(where, " and "), args
}

func personnelAssignmentSortColumn(value string) string {
	switch value {
	case "assignment_code":
		return "epa.assignment_code"
	case "assignment_type":
		return "epa.assignment_type"
	case "assignment_title":
		return "epa.assignment_title"
	case "status":
		return "epa.status"
	case "ended_on":
		return "epa.ended_on"
	case "weekly_hours":
		return "epa.weekly_hours"
	default:
		return "epa.assigned_on"
	}
}

func personnelDisciplinarySortColumn(value string) string {
	switch value {
	case "case_code":
		return "epdc.case_code"
	case "case_type":
		return "epdc.case_type"
	case "status":
		return "epdc.status"
	case "hearing_on":
		return "epdc.hearing_on"
	case "resolved_on":
		return "epdc.resolved_on"
	case "committee_name":
		return "epdc.committee_name"
	default:
		return "epdc.reported_on"
	}
}
