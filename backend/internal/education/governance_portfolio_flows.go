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

func (s *Service) GovernanceMemberships(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"school_year": {},
			"organism":    {},
			"full_name":   {},
			"role_name":   {},
			"status":      {},
		},
		[]string{"school_year", "organism", "full_name", "role_name", "status"},
	)
	if query.Sort == "" {
		query.Sort = "organism"
	}

	whereClause, args := buildGovernanceMembershipFilters(query.Filters, s.institutionID(r))

	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_governance_memberships egm "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_governance_memberships_failed"})
		return
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select
			egm.id::text,
			egm.school_year,
			egm.organism,
			egm.full_name,
			egm.role_name,
			to_char(egm.mandate_from, 'YYYY-MM-DD'),
			to_char(egm.mandate_to, 'YYYY-MM-DD'),
			egm.voting_right,
			egm.status,
			egm.institution_id,
			egm.notes
		from education_governance_memberships egm
		%s
		order by %s %s, egm.organism, egm.full_name
		limit $%d offset $%d
	`, whereClause, governanceMembershipSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_governance_memberships_failed"})
		return
	}
	defer rows.Close()

	items := make([]GovernanceMembership, 0, query.PageSize)
	for rows.Next() {
		var item GovernanceMembership
		if err := rows.Scan(
			&item.ID,
			&item.SchoolYear,
			&item.Organism,
			&item.FullName,
			&item.RoleName,
			&item.MandateFrom,
			&item.MandateTo,
			&item.VotingRight,
			&item.Status,
			&item.InstitutionID,
			&item.Notes,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_governance_memberships_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_governance_memberships_scan_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) GovernanceMembershipDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var item GovernanceMembership
	err := s.pool.QueryRow(r.Context(), `
		select
			egm.id::text,
			egm.school_year,
			egm.organism,
			egm.full_name,
			egm.role_name,
			to_char(egm.mandate_from, 'YYYY-MM-DD'),
			to_char(egm.mandate_to, 'YYYY-MM-DD'),
			egm.voting_right,
			egm.status,
			egm.institution_id,
			egm.notes
		from education_governance_memberships egm
		where egm.id = $1 and egm.institution_id = $2
	`, recordID, s.institutionID(r)).Scan(
		&item.ID,
		&item.SchoolYear,
		&item.Organism,
		&item.FullName,
		&item.RoleName,
		&item.MandateFrom,
		&item.MandateTo,
		&item.VotingRight,
		&item.Status,
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_governance_membership_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_governance_membership_detail_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreateGovernanceMembership(w http.ResponseWriter, r *http.Request) {
	var req CreateGovernanceMembershipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_governance_membership_payload"})
		return
	}

	normalizeGovernanceMembershipRequest(&req)
	if req.SchoolYear == "" || req.Organism == "" || req.FullName == "" || req.RoleName == "" || req.MandateFrom == "" || req.MandateTo == "" || req.Status == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_governance_membership_fields"})
		return
	}
	if !containsString([]string{"ca", "cp", "ceac", "cfdcd"}, req.Organism) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_governance_membership_organism"})
		return
	}
	if !containsString([]string{"activ", "suspendat", "expirat"}, req.Status) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_governance_membership_status"})
		return
	}
	mandateFrom, err := time.Parse("2006-01-02", req.MandateFrom)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_governance_membership_mandate_from"})
		return
	}
	mandateTo, err := time.Parse("2006-01-02", req.MandateTo)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_governance_membership_mandate_to"})
		return
	}
	if mandateTo.Before(mandateFrom) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_governance_membership_mandate_interval"})
		return
	}

	var item GovernanceMembership
	err = s.pool.QueryRow(r.Context(), `
		insert into education_governance_memberships (
			school_year,
			organism,
			full_name,
			role_name,
			mandate_from,
			mandate_to,
			voting_right,
			status,
			institution_id,
			notes
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		returning
			id::text,
			school_year,
			organism,
			full_name,
			role_name,
			to_char(mandate_from, 'YYYY-MM-DD'),
			to_char(mandate_to, 'YYYY-MM-DD'),
			voting_right,
			status,
			institution_id,
			notes
	`, req.SchoolYear, req.Organism, req.FullName, req.RoleName, req.MandateFrom, req.MandateTo, req.VotingRight, req.Status, s.institutionID(r), req.Notes).Scan(
		&item.ID,
		&item.SchoolYear,
		&item.Organism,
		&item.FullName,
		&item.RoleName,
		&item.MandateFrom,
		&item.MandateTo,
		&item.VotingRight,
		&item.Status,
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "governance_membership_create_failed"})
		return
	}
	s.logAudit(r, "education.governance.membership.create", "governance_membership", item.ID, "Governance membership created.", map[string]any{
		"school_year": item.SchoolYear,
		"organism":    item.Organism,
		"full_name":   item.FullName,
		"role_name":   item.RoleName,
		"status":      item.Status,
	})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdateGovernanceMembership(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreateGovernanceMembershipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_governance_membership_payload"})
		return
	}

	normalizeGovernanceMembershipRequest(&req)
	if req.SchoolYear == "" || req.Organism == "" || req.FullName == "" || req.RoleName == "" || req.MandateFrom == "" || req.MandateTo == "" || req.Status == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_governance_membership_fields"})
		return
	}
	if !containsString([]string{"ca", "cp", "ceac", "cfdcd"}, req.Organism) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_governance_membership_organism"})
		return
	}
	if !containsString([]string{"activ", "suspendat", "expirat"}, req.Status) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_governance_membership_status"})
		return
	}
	mandateFrom, err := time.Parse("2006-01-02", req.MandateFrom)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_governance_membership_mandate_from"})
		return
	}
	mandateTo, err := time.Parse("2006-01-02", req.MandateTo)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_governance_membership_mandate_to"})
		return
	}
	if mandateTo.Before(mandateFrom) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_governance_membership_mandate_interval"})
		return
	}

	var item GovernanceMembership
	err = s.pool.QueryRow(r.Context(), `
		update education_governance_memberships
		set school_year = $1, organism = $2, full_name = $3, role_name = $4, mandate_from = $5, mandate_to = $6,
			voting_right = $7, status = $8, notes = $9, updated_at = now()
		where id = $10 and institution_id = $11
		returning
			id::text,
			school_year,
			organism,
			full_name,
			role_name,
			to_char(mandate_from, 'YYYY-MM-DD'),
			to_char(mandate_to, 'YYYY-MM-DD'),
			voting_right,
			status,
			institution_id,
			notes
	`, req.SchoolYear, req.Organism, req.FullName, req.RoleName, req.MandateFrom, req.MandateTo, req.VotingRight, req.Status, req.Notes, recordID, s.institutionID(r)).Scan(
		&item.ID,
		&item.SchoolYear,
		&item.Organism,
		&item.FullName,
		&item.RoleName,
		&item.MandateFrom,
		&item.MandateTo,
		&item.VotingRight,
		&item.Status,
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_governance_membership_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "governance_membership_update_failed"})
		return
	}
	s.logAudit(r, "education.governance.membership.update", "governance_membership", item.ID, "Governance membership updated.", map[string]any{
		"school_year": item.SchoolYear,
		"organism":    item.Organism,
		"full_name":   item.FullName,
		"role_name":   item.RoleName,
		"status":      item.Status,
	})
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteGovernanceMembership(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	tag, err := s.pool.Exec(r.Context(), `delete from education_governance_memberships where id = $1 and institution_id = $2`, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "governance_membership_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_governance_membership_not_found")
		return
	}
	s.logAudit(r, "education.governance.membership.delete", "governance_membership", recordID, "Governance membership deleted.", nil)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) GovernanceResolutions(w http.ResponseWriter, r *http.Request) {
	meetingID := strings.TrimSpace(chi.URLParam(r, "meetingID"))
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"resolution_code":     {},
			"title":               {},
			"resolution_type":     {},
			"publication_status":  {},
			"anonymization_state": {},
		},
		[]string{"resolution_code", "title", "resolution_type", "publication_status", "anonymization_state"},
	)
	if query.Sort == "" {
		query.Sort = "issued_on"
	}
	whereClause, args := buildGovernanceResolutionFilters(query.Filters, meetingID, s.institutionID(r))
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_meeting_resolutions emr "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_meeting_resolutions_failed"})
		return
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select id::text, meeting_id::text, vote_id::text, resolution_code, title, resolution_type, publication_status,
			anonymization_state, to_char(issued_on, 'YYYY-MM-DD'), signed_by, institution_id, notes
		from education_meeting_resolutions emr
		%s
		order by %s %s, issued_on desc, resolution_code
		limit $%d offset $%d
	`, whereClause, governanceResolutionSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_meeting_resolutions_failed"})
		return
	}
	defer rows.Close()
	items := make([]GovernanceResolution, 0, query.PageSize)
	for rows.Next() {
		var item GovernanceResolution
		if err := rows.Scan(&item.ID, &item.MeetingID, &item.VoteID, &item.ResolutionCode, &item.Title, &item.ResolutionType, &item.PublicationStatus, &item.AnonymizationState, &item.IssuedOn, &item.SignedBy, &item.InstitutionID, &item.Notes); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_meeting_resolutions_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_meeting_resolutions_scan_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) GovernanceResolutionDetail(w http.ResponseWriter, r *http.Request) {
	meetingID := strings.TrimSpace(chi.URLParam(r, "meetingID"))
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var item GovernanceResolution
	err := s.pool.QueryRow(r.Context(), `
		select id::text, meeting_id::text, vote_id::text, resolution_code, title, resolution_type, publication_status,
			anonymization_state, to_char(issued_on, 'YYYY-MM-DD'), signed_by, institution_id, notes
		from education_meeting_resolutions
		where id = $1 and meeting_id = $2 and institution_id = $3
	`, recordID, meetingID, s.institutionID(r)).Scan(&item.ID, &item.MeetingID, &item.VoteID, &item.ResolutionCode, &item.Title, &item.ResolutionType, &item.PublicationStatus, &item.AnonymizationState, &item.IssuedOn, &item.SignedBy, &item.InstitutionID, &item.Notes)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_meeting_resolution_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_meeting_resolution_detail_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreateGovernanceResolution(w http.ResponseWriter, r *http.Request) {
	meetingID := strings.TrimSpace(chi.URLParam(r, "meetingID"))
	var req CreateGovernanceResolutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_governance_resolution_payload"})
		return
	}
	normalizeGovernanceResolutionRequest(&req)
	if req.Title == "" || req.ResolutionType == "" || req.PublicationStatus == "" || req.AnonymizationState == "" || req.IssuedOn == "" || req.VoteID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_governance_resolution_fields"})
		return
	}
	if !containsString([]string{"hotarare", "decizie", "aviz"}, req.ResolutionType) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_governance_resolution_type"})
		return
	}
	if !containsString([]string{"intern", "publicat", "pregatit_publicare"}, req.PublicationStatus) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_governance_resolution_publication_status"})
		return
	}
	if !containsString([]string{"necesara", "finalizata", "nu_este_necesara"}, req.AnonymizationState) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_governance_resolution_anonymization_state"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.IssuedOn); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_governance_resolution_issued_on"})
		return
	}
	code := fmt.Sprintf("HTR-%d-%04d", time.Now().UTC().Year(), time.Now().Unix()%10000)
	var item GovernanceResolution
	err := s.pool.QueryRow(r.Context(), `
		insert into education_meeting_resolutions (
			meeting_id, vote_id, resolution_code, title, resolution_type, publication_status, anonymization_state, issued_on, signed_by, institution_id, notes
		)
		select em.id, emv.id, $3, $4, $5, $6, $7, $8, $9, em.institution_id, $10
		from education_meetings em
		join education_meeting_votes emv on emv.id::text = $2 and emv.meeting_id = em.id
		where em.id = $1 and em.institution_id = $11
		returning id::text, meeting_id::text, vote_id::text, resolution_code, title, resolution_type, publication_status, anonymization_state, to_char(issued_on, 'YYYY-MM-DD'), signed_by, institution_id, notes
	`, meetingID, req.VoteID, code, req.Title, req.ResolutionType, req.PublicationStatus, req.AnonymizationState, req.IssuedOn, req.SignedBy, req.Notes, s.institutionID(r)).Scan(
		&item.ID, &item.MeetingID, &item.VoteID, &item.ResolutionCode, &item.Title, &item.ResolutionType, &item.PublicationStatus, &item.AnonymizationState, &item.IssuedOn, &item.SignedBy, &item.InstitutionID, &item.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_meeting_or_vote_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "governance_resolution_create_failed"})
		return
	}
	s.logAudit(r, "education.governance.resolution.create", "governance_resolution", item.ID, "Governance resolution created.", map[string]any{
		"meeting_id": item.MeetingID, "vote_id": item.VoteID, "resolution_code": item.ResolutionCode, "title": item.Title,
	})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdateGovernanceResolution(w http.ResponseWriter, r *http.Request) {
	meetingID := strings.TrimSpace(chi.URLParam(r, "meetingID"))
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreateGovernanceResolutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_governance_resolution_payload"})
		return
	}
	normalizeGovernanceResolutionRequest(&req)
	if req.Title == "" || req.ResolutionType == "" || req.PublicationStatus == "" || req.AnonymizationState == "" || req.IssuedOn == "" || req.VoteID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_governance_resolution_fields"})
		return
	}
	if !containsString([]string{"hotarare", "decizie", "aviz"}, req.ResolutionType) || !containsString([]string{"intern", "publicat", "pregatit_publicare"}, req.PublicationStatus) || !containsString([]string{"necesara", "finalizata", "nu_este_necesara"}, req.AnonymizationState) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_governance_resolution_fields"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.IssuedOn); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_governance_resolution_issued_on"})
		return
	}
	var item GovernanceResolution
	err := s.pool.QueryRow(r.Context(), `
		update education_meeting_resolutions emr
		set vote_id = emv.id, title = $1, resolution_type = $2, publication_status = $3, anonymization_state = $4, issued_on = $5, signed_by = $6, notes = $7, updated_at = now()
		from education_meeting_votes emv
		where emr.id = $8 and emr.meeting_id = $9 and emr.institution_id = $10 and emv.id::text = $11 and emv.meeting_id = emr.meeting_id
		returning emr.id::text, emr.meeting_id::text, emr.vote_id::text, emr.resolution_code, emr.title, emr.resolution_type, emr.publication_status, emr.anonymization_state, to_char(emr.issued_on, 'YYYY-MM-DD'), emr.signed_by, emr.institution_id, emr.notes
	`, req.Title, req.ResolutionType, req.PublicationStatus, req.AnonymizationState, req.IssuedOn, req.SignedBy, req.Notes, recordID, meetingID, s.institutionID(r), req.VoteID).Scan(
		&item.ID, &item.MeetingID, &item.VoteID, &item.ResolutionCode, &item.Title, &item.ResolutionType, &item.PublicationStatus, &item.AnonymizationState, &item.IssuedOn, &item.SignedBy, &item.InstitutionID, &item.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_meeting_resolution_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "governance_resolution_update_failed"})
		return
	}
	s.logAudit(r, "education.governance.resolution.update", "governance_resolution", item.ID, "Governance resolution updated.", map[string]any{
		"meeting_id": item.MeetingID, "vote_id": item.VoteID, "resolution_code": item.ResolutionCode, "title": item.Title,
	})
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteGovernanceResolution(w http.ResponseWriter, r *http.Request) {
	meetingID := strings.TrimSpace(chi.URLParam(r, "meetingID"))
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	tag, err := s.pool.Exec(r.Context(), `delete from education_meeting_resolutions where id = $1 and meeting_id = $2 and institution_id = $3`, recordID, meetingID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "governance_resolution_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_meeting_resolution_not_found")
		return
	}
	s.logAudit(r, "education.governance.resolution.delete", "governance_resolution", recordID, "Governance resolution deleted.", map[string]any{"meeting_id": meetingID})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) PortfolioTransfers(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"transfer_code":           {},
		"transfer_type":           {},
		"source_institution":      {},
		"destination_institution": {},
		"status":                  {},
	}, []string{"transfer_code", "transfer_type", "source_institution", "destination_institution", "status"})
	if query.Sort == "" {
		query.Sort = "handover_on"
	}
	whereClause, args := buildPortfolioTransferFilters(query.Filters, recordID, s.institutionID(r))
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_portfolio_transfers ept "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_transfers_failed"})
		return
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select id::text, portfolio_id::text, transfer_code, transfer_type, source_institution, destination_institution, status,
			to_char(handover_on, 'YYYY-MM-DD'), coalesce(to_char(received_on, 'YYYY-MM-DD'), ''), handover_by, received_by, institution_id, notes
		from education_portfolio_transfers ept
		%s
		order by %s %s, handover_on desc, transfer_code
		limit $%d offset $%d
	`, whereClause, portfolioTransferSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_transfers_failed"})
		return
	}
	defer rows.Close()
	items := make([]PortfolioTransferEvent, 0, query.PageSize)
	for rows.Next() {
		var item PortfolioTransferEvent
		if err := rows.Scan(&item.ID, &item.PortfolioID, &item.TransferCode, &item.TransferType, &item.SourceInstitution, &item.DestinationInstitution, &item.Status, &item.HandoverOn, &item.ReceivedOn, &item.HandoverBy, &item.ReceivedBy, &item.InstitutionID, &item.Notes); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_transfers_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_transfers_scan_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) PortfolioTransferDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var item PortfolioTransferEvent
	err := s.pool.QueryRow(r.Context(), `
		select id::text, portfolio_id::text, transfer_code, transfer_type, source_institution, destination_institution, status,
			to_char(handover_on, 'YYYY-MM-DD'), coalesce(to_char(received_on, 'YYYY-MM-DD'), ''), handover_by, received_by, institution_id, notes
		from education_portfolio_transfers
		where id = $1 and portfolio_id = $2 and institution_id = $3
	`, itemID, recordID, s.institutionID(r)).Scan(&item.ID, &item.PortfolioID, &item.TransferCode, &item.TransferType, &item.SourceInstitution, &item.DestinationInstitution, &item.Status, &item.HandoverOn, &item.ReceivedOn, &item.HandoverBy, &item.ReceivedBy, &item.InstitutionID, &item.Notes)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_portfolio_transfer_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_transfer_detail_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreatePortfolioTransfer(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreatePortfolioTransferEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_transfer_payload"})
		return
	}
	normalizePortfolioTransferRequest(&req)
	if req.TransferType == "" || req.SourceInstitution == "" || req.DestinationInstitution == "" || req.Status == "" || req.HandoverOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_portfolio_transfer_fields"})
		return
	}
	if !containsString([]string{"predare", "primire", "mutare", "detasare"}, req.TransferType) || !containsString([]string{"pregatit", "trimis", "receptionat", "inchis"}, req.Status) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_transfer_fields"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.HandoverOn); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_transfer_handover_on"})
		return
	}
	receivedOn := any(nil)
	if req.ReceivedOn != "" {
		if _, err := time.Parse("2006-01-02", req.ReceivedOn); err != nil {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_transfer_received_on"})
			return
		}
		receivedOn = req.ReceivedOn
	}
	if containsString([]string{"receptionat", "inchis"}, req.Status) && receivedOn == nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_portfolio_transfer_received_on"})
		return
	}
	code := fmt.Sprintf("TRF-%d-%04d", time.Now().UTC().Year(), time.Now().Unix()%10000)
	var item PortfolioTransferEvent
	err := s.pool.QueryRow(r.Context(), `
		insert into education_portfolio_transfers (
			portfolio_id, transfer_code, transfer_type, source_institution, destination_institution, status, handover_on, received_on, handover_by, received_by, institution_id, notes
		)
		select ep.id, $2, $3, $4, $5, $6, $7, $8, $9, $10, ep.institution_id, $11
		from education_portfolios ep
		where ep.id = $1 and ep.institution_id = $12
		returning id::text, portfolio_id::text, transfer_code, transfer_type, source_institution, destination_institution, status, to_char(handover_on, 'YYYY-MM-DD'),
			coalesce(to_char(received_on, 'YYYY-MM-DD'), ''), handover_by, received_by, institution_id, notes
	`, recordID, code, req.TransferType, req.SourceInstitution, req.DestinationInstitution, req.Status, req.HandoverOn, receivedOn, req.HandoverBy, req.ReceivedBy, req.Notes, s.institutionID(r)).Scan(
		&item.ID, &item.PortfolioID, &item.TransferCode, &item.TransferType, &item.SourceInstitution, &item.DestinationInstitution, &item.Status, &item.HandoverOn, &item.ReceivedOn, &item.HandoverBy, &item.ReceivedBy, &item.InstitutionID, &item.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_portfolio_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_transfer_create_failed"})
		return
	}
	s.logAudit(r, "education.portfolios.transfer.create", "portfolio_transfer", item.ID, "Portfolio transfer created.", map[string]any{
		"portfolio_id": item.PortfolioID, "transfer_code": item.TransferCode, "status": item.Status,
	})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdatePortfolioTransfer(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var req CreatePortfolioTransferEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_transfer_payload"})
		return
	}
	normalizePortfolioTransferRequest(&req)
	if req.TransferType == "" || req.SourceInstitution == "" || req.DestinationInstitution == "" || req.Status == "" || req.HandoverOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_portfolio_transfer_fields"})
		return
	}
	if !containsString([]string{"predare", "primire", "mutare", "detasare"}, req.TransferType) || !containsString([]string{"pregatit", "trimis", "receptionat", "inchis"}, req.Status) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_transfer_fields"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.HandoverOn); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_transfer_handover_on"})
		return
	}
	receivedOn := any(nil)
	if req.ReceivedOn != "" {
		if _, err := time.Parse("2006-01-02", req.ReceivedOn); err != nil {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_transfer_received_on"})
			return
		}
		receivedOn = req.ReceivedOn
	}
	if containsString([]string{"receptionat", "inchis"}, req.Status) && receivedOn == nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_portfolio_transfer_received_on"})
		return
	}
	var item PortfolioTransferEvent
	err := s.pool.QueryRow(r.Context(), `
		update education_portfolio_transfers
		set transfer_type = $1, source_institution = $2, destination_institution = $3, status = $4, handover_on = $5, received_on = $6, handover_by = $7, received_by = $8, notes = $9, updated_at = now()
		where id = $10 and portfolio_id = $11 and institution_id = $12
		returning id::text, portfolio_id::text, transfer_code, transfer_type, source_institution, destination_institution, status, to_char(handover_on, 'YYYY-MM-DD'),
			coalesce(to_char(received_on, 'YYYY-MM-DD'), ''), handover_by, received_by, institution_id, notes
	`, req.TransferType, req.SourceInstitution, req.DestinationInstitution, req.Status, req.HandoverOn, receivedOn, req.HandoverBy, req.ReceivedBy, req.Notes, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID, &item.PortfolioID, &item.TransferCode, &item.TransferType, &item.SourceInstitution, &item.DestinationInstitution, &item.Status, &item.HandoverOn, &item.ReceivedOn, &item.HandoverBy, &item.ReceivedBy, &item.InstitutionID, &item.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_portfolio_transfer_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_transfer_update_failed"})
		return
	}
	s.logAudit(r, "education.portfolios.transfer.update", "portfolio_transfer", item.ID, "Portfolio transfer updated.", map[string]any{
		"portfolio_id": item.PortfolioID, "transfer_code": item.TransferCode, "status": item.Status,
	})
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeletePortfolioTransfer(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	tag, err := s.pool.Exec(r.Context(), `delete from education_portfolio_transfers where id = $1 and portfolio_id = $2 and institution_id = $3`, itemID, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_transfer_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_portfolio_transfer_not_found")
		return
	}
	s.logAudit(r, "education.portfolios.transfer.delete", "portfolio_transfer", itemID, "Portfolio transfer deleted.", map[string]any{"portfolio_id": recordID})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) PortfolioReviews(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"review_code":   {},
		"review_stage":  {},
		"outcome":       {},
		"reviewer_name": {},
	}, []string{"review_code", "review_stage", "outcome", "reviewer_name"})
	if query.Sort == "" {
		query.Sort = "reviewed_on"
	}
	whereClause, args := buildPortfolioReviewFilters(query.Filters, recordID, s.institutionID(r))
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_portfolio_reviews epr "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_reviews_failed"})
		return
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select id::text, portfolio_id::text, review_code, review_stage, outcome, reviewer_name, to_char(reviewed_on, 'YYYY-MM-DD'),
			missing_documents, compliance_score, institution_id, notes
		from education_portfolio_reviews epr
		%s
		order by %s %s, reviewed_on desc, review_code
		limit $%d offset $%d
	`, whereClause, portfolioReviewSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_reviews_failed"})
		return
	}
	defer rows.Close()
	items := make([]PortfolioReviewEvent, 0, query.PageSize)
	for rows.Next() {
		var item PortfolioReviewEvent
		if err := rows.Scan(&item.ID, &item.PortfolioID, &item.ReviewCode, &item.ReviewStage, &item.Outcome, &item.ReviewerName, &item.ReviewedOn, &item.MissingDocuments, &item.ComplianceScore, &item.InstitutionID, &item.Notes); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_reviews_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_reviews_scan_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) PortfolioReviewDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var item PortfolioReviewEvent
	err := s.pool.QueryRow(r.Context(), `
		select id::text, portfolio_id::text, review_code, review_stage, outcome, reviewer_name, to_char(reviewed_on, 'YYYY-MM-DD'),
			missing_documents, compliance_score, institution_id, notes
		from education_portfolio_reviews
		where id = $1 and portfolio_id = $2 and institution_id = $3
	`, itemID, recordID, s.institutionID(r)).Scan(&item.ID, &item.PortfolioID, &item.ReviewCode, &item.ReviewStage, &item.Outcome, &item.ReviewerName, &item.ReviewedOn, &item.MissingDocuments, &item.ComplianceScore, &item.InstitutionID, &item.Notes)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_portfolio_review_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_review_detail_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreatePortfolioReview(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreatePortfolioReviewEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_review_payload"})
		return
	}
	normalizePortfolioReviewRequest(&req)
	if req.ReviewStage == "" || req.Outcome == "" || req.ReviewerName == "" || req.ReviewedOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_portfolio_review_fields"})
		return
	}
	if req.MissingDocuments < 0 || req.ComplianceScore < 0 || req.ComplianceScore > 100 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_review_scores"})
		return
	}
	if !containsString([]string{"depunere", "verificare_secretariat", "validare_manageriala", "reverificare"}, req.ReviewStage) || !containsString([]string{"acceptat", "completari", "respins"}, req.Outcome) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_review_fields"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.ReviewedOn); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_review_reviewed_on"})
		return
	}
	code := fmt.Sprintf("REV-%d-%04d", time.Now().UTC().Year(), time.Now().Unix()%10000)
	var item PortfolioReviewEvent
	err := s.pool.QueryRow(r.Context(), `
		insert into education_portfolio_reviews (
			portfolio_id, review_code, review_stage, outcome, reviewer_name, reviewed_on, missing_documents, compliance_score, institution_id, notes
		)
		select ep.id, $2, $3, $4, $5, $6, $7, $8, ep.institution_id, $9
		from education_portfolios ep
		where ep.id = $1 and ep.institution_id = $10
		returning id::text, portfolio_id::text, review_code, review_stage, outcome, reviewer_name, to_char(reviewed_on, 'YYYY-MM-DD'),
			missing_documents, compliance_score, institution_id, notes
	`, recordID, code, req.ReviewStage, req.Outcome, req.ReviewerName, req.ReviewedOn, req.MissingDocuments, req.ComplianceScore, req.Notes, s.institutionID(r)).Scan(
		&item.ID, &item.PortfolioID, &item.ReviewCode, &item.ReviewStage, &item.Outcome, &item.ReviewerName, &item.ReviewedOn, &item.MissingDocuments, &item.ComplianceScore, &item.InstitutionID, &item.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_portfolio_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_review_create_failed"})
		return
	}
	s.logAudit(r, "education.portfolios.review.create", "portfolio_review", item.ID, "Portfolio review created.", map[string]any{
		"portfolio_id": item.PortfolioID, "review_code": item.ReviewCode, "outcome": item.Outcome, "review_stage": item.ReviewStage,
	})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdatePortfolioReview(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var req CreatePortfolioReviewEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_review_payload"})
		return
	}
	normalizePortfolioReviewRequest(&req)
	if req.ReviewStage == "" || req.Outcome == "" || req.ReviewerName == "" || req.ReviewedOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_portfolio_review_fields"})
		return
	}
	if req.MissingDocuments < 0 || req.ComplianceScore < 0 || req.ComplianceScore > 100 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_review_scores"})
		return
	}
	if !containsString([]string{"depunere", "verificare_secretariat", "validare_manageriala", "reverificare"}, req.ReviewStage) || !containsString([]string{"acceptat", "completari", "respins"}, req.Outcome) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_review_fields"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.ReviewedOn); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_review_reviewed_on"})
		return
	}
	var item PortfolioReviewEvent
	err := s.pool.QueryRow(r.Context(), `
		update education_portfolio_reviews
		set review_stage = $1, outcome = $2, reviewer_name = $3, reviewed_on = $4, missing_documents = $5, compliance_score = $6, notes = $7, updated_at = now()
		where id = $8 and portfolio_id = $9 and institution_id = $10
		returning id::text, portfolio_id::text, review_code, review_stage, outcome, reviewer_name, to_char(reviewed_on, 'YYYY-MM-DD'),
			missing_documents, compliance_score, institution_id, notes
	`, req.ReviewStage, req.Outcome, req.ReviewerName, req.ReviewedOn, req.MissingDocuments, req.ComplianceScore, req.Notes, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID, &item.PortfolioID, &item.ReviewCode, &item.ReviewStage, &item.Outcome, &item.ReviewerName, &item.ReviewedOn, &item.MissingDocuments, &item.ComplianceScore, &item.InstitutionID, &item.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_portfolio_review_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_review_update_failed"})
		return
	}
	s.logAudit(r, "education.portfolios.review.update", "portfolio_review", item.ID, "Portfolio review updated.", map[string]any{
		"portfolio_id": item.PortfolioID, "review_code": item.ReviewCode, "outcome": item.Outcome, "review_stage": item.ReviewStage,
	})
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeletePortfolioReview(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	tag, err := s.pool.Exec(r.Context(), `delete from education_portfolio_reviews where id = $1 and portfolio_id = $2 and institution_id = $3`, itemID, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_review_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_portfolio_review_not_found")
		return
	}
	s.logAudit(r, "education.portfolios.review.delete", "portfolio_review", itemID, "Portfolio review deleted.", map[string]any{"portfolio_id": recordID})
	w.WriteHeader(http.StatusNoContent)
}

func buildGovernanceMembershipFilters(filters map[string]string, institutionID string) (string, []any) {
	where := []string{"egm.institution_id = $1"}
	args := []any{institutionID}
	for key, column := range map[string]string{
		"school_year": "egm.school_year",
		"organism":    "egm.organism",
		"full_name":   "egm.full_name",
		"role_name":   "egm.role_name",
		"status":      "egm.status",
	} {
		if value := strings.TrimSpace(filters[key]); value != "" {
			args = append(args, "%"+strings.ToLower(value)+"%")
			where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
		}
	}
	return "where " + strings.Join(where, " and "), args
}

func buildGovernanceResolutionFilters(filters map[string]string, meetingID string, institutionID string) (string, []any) {
	where := []string{"emr.meeting_id = $1", "emr.institution_id = $2"}
	args := []any{meetingID, institutionID}
	for key, column := range map[string]string{
		"resolution_code":     "emr.resolution_code",
		"title":               "emr.title",
		"resolution_type":     "emr.resolution_type",
		"publication_status":  "emr.publication_status",
		"anonymization_state": "emr.anonymization_state",
	} {
		if value := strings.TrimSpace(filters[key]); value != "" {
			args = append(args, "%"+strings.ToLower(value)+"%")
			where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
		}
	}
	return "where " + strings.Join(where, " and "), args
}

func buildPortfolioTransferFilters(filters map[string]string, recordID string, institutionID string) (string, []any) {
	where := []string{"ept.portfolio_id = $1", "ept.institution_id = $2"}
	args := []any{recordID, institutionID}
	for key, column := range map[string]string{
		"transfer_code":           "ept.transfer_code",
		"transfer_type":           "ept.transfer_type",
		"source_institution":      "ept.source_institution",
		"destination_institution": "ept.destination_institution",
		"status":                  "ept.status",
	} {
		if value := strings.TrimSpace(filters[key]); value != "" {
			args = append(args, "%"+strings.ToLower(value)+"%")
			where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
		}
	}
	return "where " + strings.Join(where, " and "), args
}

func buildPortfolioReviewFilters(filters map[string]string, recordID string, institutionID string) (string, []any) {
	where := []string{"epr.portfolio_id = $1", "epr.institution_id = $2"}
	args := []any{recordID, institutionID}
	for key, column := range map[string]string{
		"review_code":   "epr.review_code",
		"review_stage":  "epr.review_stage",
		"outcome":       "epr.outcome",
		"reviewer_name": "epr.reviewer_name",
	} {
		if value := strings.TrimSpace(filters[key]); value != "" {
			args = append(args, "%"+strings.ToLower(value)+"%")
			where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
		}
	}
	return "where " + strings.Join(where, " and "), args
}

func governanceMembershipSortColumn(value string) string {
	switch value {
	case "school_year":
		return "egm.school_year"
	case "organism":
		return "egm.organism"
	case "full_name":
		return "egm.full_name"
	case "role_name":
		return "egm.role_name"
	case "mandate_from":
		return "egm.mandate_from"
	case "mandate_to":
		return "egm.mandate_to"
	case "status":
		return "egm.status"
	default:
		return "egm.organism"
	}
}

func governanceResolutionSortColumn(value string) string {
	switch value {
	case "resolution_code":
		return "emr.resolution_code"
	case "title":
		return "emr.title"
	case "resolution_type":
		return "emr.resolution_type"
	case "publication_status":
		return "emr.publication_status"
	case "anonymization_state":
		return "emr.anonymization_state"
	case "issued_on":
		return "emr.issued_on"
	default:
		return "emr.issued_on"
	}
}

func portfolioTransferSortColumn(value string) string {
	switch value {
	case "transfer_code":
		return "ept.transfer_code"
	case "transfer_type":
		return "ept.transfer_type"
	case "source_institution":
		return "ept.source_institution"
	case "destination_institution":
		return "ept.destination_institution"
	case "status":
		return "ept.status"
	case "handover_on":
		return "ept.handover_on"
	case "received_on":
		return "ept.received_on"
	default:
		return "ept.handover_on"
	}
}

func portfolioReviewSortColumn(value string) string {
	switch value {
	case "review_code":
		return "epr.review_code"
	case "review_stage":
		return "epr.review_stage"
	case "outcome":
		return "epr.outcome"
	case "reviewer_name":
		return "epr.reviewer_name"
	case "reviewed_on":
		return "epr.reviewed_on"
	case "missing_documents":
		return "epr.missing_documents"
	case "compliance_score":
		return "epr.compliance_score"
	default:
		return "epr.reviewed_on"
	}
}

func normalizeGovernanceMembershipRequest(req *CreateGovernanceMembershipRequest) {
	req.SchoolYear = strings.TrimSpace(req.SchoolYear)
	req.Organism = strings.TrimSpace(req.Organism)
	req.FullName = strings.TrimSpace(req.FullName)
	req.RoleName = strings.TrimSpace(req.RoleName)
	req.MandateFrom = strings.TrimSpace(req.MandateFrom)
	req.MandateTo = strings.TrimSpace(req.MandateTo)
	req.Status = strings.TrimSpace(req.Status)
	req.Notes = strings.TrimSpace(req.Notes)
}

func normalizeGovernanceResolutionRequest(req *CreateGovernanceResolutionRequest) {
	req.VoteID = strings.TrimSpace(req.VoteID)
	req.Title = strings.TrimSpace(req.Title)
	req.ResolutionType = strings.TrimSpace(req.ResolutionType)
	req.PublicationStatus = strings.TrimSpace(req.PublicationStatus)
	req.AnonymizationState = strings.TrimSpace(req.AnonymizationState)
	req.IssuedOn = strings.TrimSpace(req.IssuedOn)
	req.SignedBy = strings.TrimSpace(req.SignedBy)
	req.Notes = strings.TrimSpace(req.Notes)
}

func normalizePortfolioTransferRequest(req *CreatePortfolioTransferEventRequest) {
	req.TransferType = strings.TrimSpace(req.TransferType)
	req.SourceInstitution = strings.TrimSpace(req.SourceInstitution)
	req.DestinationInstitution = strings.TrimSpace(req.DestinationInstitution)
	req.Status = strings.TrimSpace(req.Status)
	req.HandoverOn = strings.TrimSpace(req.HandoverOn)
	req.ReceivedOn = strings.TrimSpace(req.ReceivedOn)
	req.HandoverBy = strings.TrimSpace(req.HandoverBy)
	req.ReceivedBy = strings.TrimSpace(req.ReceivedBy)
	req.Notes = strings.TrimSpace(req.Notes)
}

func normalizePortfolioReviewRequest(req *CreatePortfolioReviewEventRequest) {
	req.ReviewStage = strings.TrimSpace(req.ReviewStage)
	req.Outcome = strings.TrimSpace(req.Outcome)
	req.ReviewerName = strings.TrimSpace(req.ReviewerName)
	req.ReviewedOn = strings.TrimSpace(req.ReviewedOn)
	req.Notes = strings.TrimSpace(req.Notes)
}
