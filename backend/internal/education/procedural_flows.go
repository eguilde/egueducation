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

func (s *Service) GovernanceMinuteItems(w http.ResponseWriter, r *http.Request) {
	meetingID := strings.TrimSpace(chi.URLParam(r, "meetingID"))
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"topic_title":          {},
		"follow_up_status":     {},
		"responsible_party":    {},
		"requires_publication": {},
	}, []string{"agenda_order", "topic_title", "follow_up_status", "responsible_party"})
	if query.Sort == "" {
		query.Sort = "agenda_order"
	}

	whereClause, args := buildGovernanceMinuteFilters(query.Filters, meetingID, s.institutionID(r))

	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_meeting_minutes emm "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_meeting_minutes_failed"})
		return
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select
			emm.id::text,
			emm.meeting_id::text,
			emm.agenda_order,
			emm.topic_title,
			emm.discussion_summary,
			emm.decision_summary,
			emm.responsible_party,
			coalesce(to_char(emm.due_on, 'YYYY-MM-DD'), ''),
			emm.follow_up_status,
			emm.requires_publication,
			emm.institution_id,
			emm.notes
		from education_meeting_minutes emm
		%s
		order by %s %s, emm.agenda_order, emm.topic_title
		limit $%d offset $%d
	`, whereClause, governanceMinuteSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_meeting_minutes_failed"})
		return
	}
	defer rows.Close()

	items := make([]GovernanceMinuteItem, 0, query.PageSize)
	for rows.Next() {
		var item GovernanceMinuteItem
		if err := rows.Scan(
			&item.ID,
			&item.MeetingID,
			&item.AgendaOrder,
			&item.TopicTitle,
			&item.DiscussionSummary,
			&item.DecisionSummary,
			&item.ResponsibleParty,
			&item.DueOn,
			&item.FollowUpStatus,
			&item.RequiresPublication,
			&item.InstitutionID,
			&item.Notes,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_meeting_minutes_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_meeting_minutes_scan_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) GovernanceMinuteItemDetail(w http.ResponseWriter, r *http.Request) {
	meetingID := strings.TrimSpace(chi.URLParam(r, "meetingID"))
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var item GovernanceMinuteItem
	err := s.pool.QueryRow(r.Context(), `
		select
			id::text,
			meeting_id::text,
			agenda_order,
			topic_title,
			discussion_summary,
			decision_summary,
			responsible_party,
			coalesce(to_char(due_on, 'YYYY-MM-DD'), ''),
			follow_up_status,
			requires_publication,
			institution_id,
			notes
		from education_meeting_minutes
		where id = $1 and meeting_id = $2 and institution_id = $3
	`, recordID, meetingID, s.institutionID(r)).Scan(
		&item.ID,
		&item.MeetingID,
		&item.AgendaOrder,
		&item.TopicTitle,
		&item.DiscussionSummary,
		&item.DecisionSummary,
		&item.ResponsibleParty,
		&item.DueOn,
		&item.FollowUpStatus,
		&item.RequiresPublication,
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_meeting_minute_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_meeting_minute_detail_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreateGovernanceMinuteItem(w http.ResponseWriter, r *http.Request) {
	meetingID := strings.TrimSpace(chi.URLParam(r, "meetingID"))
	var req CreateGovernanceMinuteItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_minute_payload"})
		return
	}

	normalizeGovernanceMinuteRequest(&req)
	if req.AgendaOrder < 1 || req.TopicTitle == "" || req.DiscussionSummary == "" || req.DecisionSummary == "" || req.FollowUpStatus == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_meeting_minute_fields"})
		return
	}
	if !containsString([]string{"de_stabilit", "in_urmarire", "realizat", "amanat", "inchis"}, req.FollowUpStatus) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_minute_follow_up_status"})
		return
	}
	dueOn := any(nil)
	if req.DueOn != "" {
		if _, err := time.Parse("2006-01-02", req.DueOn); err != nil {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_minute_due_on"})
			return
		}
		dueOn = req.DueOn
	}

	var item GovernanceMinuteItem
	err := s.pool.QueryRow(r.Context(), `
		insert into education_meeting_minutes (
			meeting_id, agenda_order, topic_title, discussion_summary, decision_summary, responsible_party, due_on, follow_up_status, requires_publication, institution_id, notes
		)
		select em.id, $2, $3, $4, $5, $6, $7, $8, $9, em.institution_id, $10
		from education_meetings em
		where em.id = $1 and em.institution_id = $11
		returning id::text, meeting_id::text, agenda_order, topic_title, discussion_summary, decision_summary, responsible_party,
			coalesce(to_char(due_on, 'YYYY-MM-DD'), ''), follow_up_status, requires_publication, institution_id, notes
	`, meetingID, req.AgendaOrder, req.TopicTitle, req.DiscussionSummary, req.DecisionSummary, req.ResponsibleParty, dueOn, req.FollowUpStatus, req.RequiresPublication, req.Notes, s.institutionID(r)).Scan(
		&item.ID,
		&item.MeetingID,
		&item.AgendaOrder,
		&item.TopicTitle,
		&item.DiscussionSummary,
		&item.DecisionSummary,
		&item.ResponsibleParty,
		&item.DueOn,
		&item.FollowUpStatus,
		&item.RequiresPublication,
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_meeting_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "meeting_minute_create_failed"})
		return
	}
	s.logAudit(r, "education.governance.minute.create", "governance_minute_item", item.ID, "Meeting minute item created.", map[string]any{
		"meeting_id":       item.MeetingID,
		"agenda_order":     item.AgendaOrder,
		"topic_title":      item.TopicTitle,
		"follow_up_status": item.FollowUpStatus,
	})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdateGovernanceMinuteItem(w http.ResponseWriter, r *http.Request) {
	meetingID := strings.TrimSpace(chi.URLParam(r, "meetingID"))
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreateGovernanceMinuteItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_minute_payload"})
		return
	}

	normalizeGovernanceMinuteRequest(&req)
	if req.AgendaOrder < 1 || req.TopicTitle == "" || req.DiscussionSummary == "" || req.DecisionSummary == "" || req.FollowUpStatus == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_meeting_minute_fields"})
		return
	}
	if !containsString([]string{"de_stabilit", "in_urmarire", "realizat", "amanat", "inchis"}, req.FollowUpStatus) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_minute_follow_up_status"})
		return
	}
	dueOn := any(nil)
	if req.DueOn != "" {
		if _, err := time.Parse("2006-01-02", req.DueOn); err != nil {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_minute_due_on"})
			return
		}
		dueOn = req.DueOn
	}

	var item GovernanceMinuteItem
	err := s.pool.QueryRow(r.Context(), `
		update education_meeting_minutes
		set agenda_order = $1, topic_title = $2, discussion_summary = $3, decision_summary = $4, responsible_party = $5,
			due_on = $6, follow_up_status = $7, requires_publication = $8, notes = $9, updated_at = now()
		where id = $10 and meeting_id = $11 and institution_id = $12
		returning id::text, meeting_id::text, agenda_order, topic_title, discussion_summary, decision_summary, responsible_party,
			coalesce(to_char(due_on, 'YYYY-MM-DD'), ''), follow_up_status, requires_publication, institution_id, notes
	`, req.AgendaOrder, req.TopicTitle, req.DiscussionSummary, req.DecisionSummary, req.ResponsibleParty, dueOn, req.FollowUpStatus, req.RequiresPublication, req.Notes, recordID, meetingID, s.institutionID(r)).Scan(
		&item.ID,
		&item.MeetingID,
		&item.AgendaOrder,
		&item.TopicTitle,
		&item.DiscussionSummary,
		&item.DecisionSummary,
		&item.ResponsibleParty,
		&item.DueOn,
		&item.FollowUpStatus,
		&item.RequiresPublication,
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_meeting_minute_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "meeting_minute_update_failed"})
		return
	}
	s.logAudit(r, "education.governance.minute.update", "governance_minute_item", item.ID, "Meeting minute item updated.", map[string]any{
		"meeting_id":       item.MeetingID,
		"agenda_order":     item.AgendaOrder,
		"topic_title":      item.TopicTitle,
		"follow_up_status": item.FollowUpStatus,
	})
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteGovernanceMinuteItem(w http.ResponseWriter, r *http.Request) {
	meetingID := strings.TrimSpace(chi.URLParam(r, "meetingID"))
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	tag, err := s.pool.Exec(r.Context(), `delete from education_meeting_minutes where id = $1 and meeting_id = $2 and institution_id = $3`, recordID, meetingID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "meeting_minute_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_meeting_minute_not_found")
		return
	}
	s.logAudit(r, "education.governance.minute.delete", "governance_minute_item", recordID, "Meeting minute item deleted.", map[string]any{"meeting_id": meetingID})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) PortfolioOpisEntries(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"section_code":       {},
		"component_code":     {},
		"entry_title":        {},
		"source_scope":       {},
		"document_reference": {},
	}, []string{"section_code", "component_code", "entry_title", "source_scope", "document_reference"})
	if query.Sort == "" {
		query.Sort = "chronological_index"
	}

	whereClause, args := buildPortfolioOpisFilters(query.Filters, recordID, s.institutionID(r))
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_portfolio_opis epo "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_opis_failed"})
		return
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select id::text, portfolio_id::text, section_code, component_code, entry_title, source_scope, chronological_index,
			document_reference, included_in_transfer, to_char(checked_on, 'YYYY-MM-DD'), checked_by, institution_id, notes
		from education_portfolio_opis epo
		%s
		order by %s %s, chronological_index, section_code, component_code
		limit $%d offset $%d
	`, whereClause, portfolioOpisSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_opis_failed"})
		return
	}
	defer rows.Close()

	items := make([]PortfolioOpisEntry, 0, query.PageSize)
	for rows.Next() {
		var item PortfolioOpisEntry
		if err := rows.Scan(&item.ID, &item.PortfolioID, &item.SectionCode, &item.ComponentCode, &item.EntryTitle, &item.SourceScope, &item.ChronologicalIndex, &item.DocumentReference, &item.IncludedInTransfer, &item.CheckedOn, &item.CheckedBy, &item.InstitutionID, &item.Notes); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_opis_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_opis_scan_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) PortfolioOpisEntryDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var item PortfolioOpisEntry
	err := s.pool.QueryRow(r.Context(), `
		select id::text, portfolio_id::text, section_code, component_code, entry_title, source_scope, chronological_index,
			document_reference, included_in_transfer, to_char(checked_on, 'YYYY-MM-DD'), checked_by, institution_id, notes
		from education_portfolio_opis
		where id = $1 and portfolio_id = $2 and institution_id = $3
	`, itemID, recordID, s.institutionID(r)).Scan(&item.ID, &item.PortfolioID, &item.SectionCode, &item.ComponentCode, &item.EntryTitle, &item.SourceScope, &item.ChronologicalIndex, &item.DocumentReference, &item.IncludedInTransfer, &item.CheckedOn, &item.CheckedBy, &item.InstitutionID, &item.Notes)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_portfolio_opis_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_opis_detail_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreatePortfolioOpisEntry(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreatePortfolioOpisEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_opis_payload"})
		return
	}

	normalizePortfolioOpisRequest(&req)
	if req.SectionCode == "" || req.ComponentCode == "" || req.EntryTitle == "" || req.SourceScope == "" || req.DocumentReference == "" || req.CheckedOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_portfolio_opis_fields"})
		return
	}
	if req.ChronologicalIndex < 0 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_opis_order"})
		return
	}
	if !containsString([]string{"portofoliu", "dosar_personal"}, req.SourceScope) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_opis_source_scope"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.CheckedOn); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_opis_checked_on"})
		return
	}

	var item PortfolioOpisEntry
	err := s.pool.QueryRow(r.Context(), `
		insert into education_portfolio_opis (
			portfolio_id, section_code, component_code, entry_title, source_scope, chronological_index, document_reference,
			included_in_transfer, checked_on, checked_by, institution_id, notes
		)
		select ep.id, $2, $3, $4, $5, $6, $7, $8, $9, $10, ep.institution_id, $11
		from education_portfolios ep
		where ep.id = $1 and ep.institution_id = $12
		returning id::text, portfolio_id::text, section_code, component_code, entry_title, source_scope, chronological_index,
			document_reference, included_in_transfer, to_char(checked_on, 'YYYY-MM-DD'), checked_by, institution_id, notes
	`, recordID, req.SectionCode, req.ComponentCode, req.EntryTitle, req.SourceScope, req.ChronologicalIndex, req.DocumentReference, req.IncludedInTransfer, req.CheckedOn, req.CheckedBy, req.Notes, s.institutionID(r)).Scan(
		&item.ID, &item.PortfolioID, &item.SectionCode, &item.ComponentCode, &item.EntryTitle, &item.SourceScope, &item.ChronologicalIndex, &item.DocumentReference, &item.IncludedInTransfer, &item.CheckedOn, &item.CheckedBy, &item.InstitutionID, &item.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_portfolio_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_opis_create_failed"})
		return
	}
	s.logAudit(r, "education.portfolios.opis.create", "portfolio_opis_entry", item.ID, "Portfolio opis entry created.", map[string]any{
		"portfolio_id":        item.PortfolioID,
		"section_code":        item.SectionCode,
		"chronological_index": item.ChronologicalIndex,
		"document_reference":  item.DocumentReference,
	})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdatePortfolioOpisEntry(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var req CreatePortfolioOpisEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_opis_payload"})
		return
	}

	normalizePortfolioOpisRequest(&req)
	if req.SectionCode == "" || req.ComponentCode == "" || req.EntryTitle == "" || req.SourceScope == "" || req.DocumentReference == "" || req.CheckedOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_portfolio_opis_fields"})
		return
	}
	if req.ChronologicalIndex < 0 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_opis_order"})
		return
	}
	if !containsString([]string{"portofoliu", "dosar_personal"}, req.SourceScope) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_opis_source_scope"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.CheckedOn); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_opis_checked_on"})
		return
	}

	var item PortfolioOpisEntry
	err := s.pool.QueryRow(r.Context(), `
		update education_portfolio_opis
		set section_code = $1, component_code = $2, entry_title = $3, source_scope = $4, chronological_index = $5,
			document_reference = $6, included_in_transfer = $7, checked_on = $8, checked_by = $9, notes = $10, updated_at = now()
		where id = $11 and portfolio_id = $12 and institution_id = $13
		returning id::text, portfolio_id::text, section_code, component_code, entry_title, source_scope, chronological_index,
			document_reference, included_in_transfer, to_char(checked_on, 'YYYY-MM-DD'), checked_by, institution_id, notes
	`, req.SectionCode, req.ComponentCode, req.EntryTitle, req.SourceScope, req.ChronologicalIndex, req.DocumentReference, req.IncludedInTransfer, req.CheckedOn, req.CheckedBy, req.Notes, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID, &item.PortfolioID, &item.SectionCode, &item.ComponentCode, &item.EntryTitle, &item.SourceScope, &item.ChronologicalIndex, &item.DocumentReference, &item.IncludedInTransfer, &item.CheckedOn, &item.CheckedBy, &item.InstitutionID, &item.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_portfolio_opis_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_opis_update_failed"})
		return
	}
	s.logAudit(r, "education.portfolios.opis.update", "portfolio_opis_entry", item.ID, "Portfolio opis entry updated.", map[string]any{
		"portfolio_id":        item.PortfolioID,
		"section_code":        item.SectionCode,
		"chronological_index": item.ChronologicalIndex,
		"document_reference":  item.DocumentReference,
	})
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeletePortfolioOpisEntry(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	tag, err := s.pool.Exec(r.Context(), `delete from education_portfolio_opis where id = $1 and portfolio_id = $2 and institution_id = $3`, itemID, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_opis_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_portfolio_opis_not_found")
		return
	}
	s.logAudit(r, "education.portfolios.opis.delete", "portfolio_opis_entry", itemID, "Portfolio opis entry deleted.", map[string]any{"portfolio_id": recordID})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) PortfolioCustodyEvents(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"event_type":     {},
		"holder_name":    {},
		"holder_role":    {},
		"location_label": {},
		"access_mode":    {},
	}, []string{"event_type", "holder_name", "holder_role", "location_label", "access_mode"})
	if query.Sort == "" {
		query.Sort = "started_on"
	}

	whereClause, args := buildPortfolioCustodyFilters(query.Filters, recordID, s.institutionID(r))
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_portfolio_custody epc "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_custody_failed"})
		return
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select id::text, portfolio_id::text, event_type, holder_name, holder_role, location_label, access_reason,
			to_char(started_on, 'YYYY-MM-DD'), coalesce(to_char(ended_on, 'YYYY-MM-DD'), ''), access_mode, sensitive_data_access, institution_id, notes
		from education_portfolio_custody epc
		%s
		order by %s %s, started_on desc, holder_name
		limit $%d offset $%d
	`, whereClause, portfolioCustodySortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_custody_failed"})
		return
	}
	defer rows.Close()

	items := make([]PortfolioCustodyEvent, 0, query.PageSize)
	for rows.Next() {
		var item PortfolioCustodyEvent
		if err := rows.Scan(&item.ID, &item.PortfolioID, &item.EventType, &item.HolderName, &item.HolderRole, &item.LocationLabel, &item.AccessReason, &item.StartedOn, &item.EndedOn, &item.AccessMode, &item.SensitiveDataAccess, &item.InstitutionID, &item.Notes); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_custody_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_custody_scan_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) PortfolioCustodyEventDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var item PortfolioCustodyEvent
	err := s.pool.QueryRow(r.Context(), `
		select id::text, portfolio_id::text, event_type, holder_name, holder_role, location_label, access_reason,
			to_char(started_on, 'YYYY-MM-DD'), coalesce(to_char(ended_on, 'YYYY-MM-DD'), ''), access_mode, sensitive_data_access, institution_id, notes
		from education_portfolio_custody
		where id = $1 and portfolio_id = $2 and institution_id = $3
	`, itemID, recordID, s.institutionID(r)).Scan(&item.ID, &item.PortfolioID, &item.EventType, &item.HolderName, &item.HolderRole, &item.LocationLabel, &item.AccessReason, &item.StartedOn, &item.EndedOn, &item.AccessMode, &item.SensitiveDataAccess, &item.InstitutionID, &item.Notes)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_portfolio_custody_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_custody_detail_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreatePortfolioCustodyEvent(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreatePortfolioCustodyEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_custody_payload"})
		return
	}

	normalizePortfolioCustodyRequest(&req)
	if req.EventType == "" || req.HolderName == "" || req.HolderRole == "" || req.LocationLabel == "" || req.AccessReason == "" || req.StartedOn == "" || req.AccessMode == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_portfolio_custody_fields"})
		return
	}
	if !containsString([]string{"preluare", "consultare", "transfer", "arhivare", "restituire"}, req.EventType) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_custody_event_type"})
		return
	}
	if !containsString([]string{"fizic", "digital", "mixt"}, req.AccessMode) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_custody_access_mode"})
		return
	}
	startedOn, err := time.Parse("2006-01-02", req.StartedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_custody_started_on"})
		return
	}
	endedOn := any(nil)
	if req.EndedOn != "" {
		parsedEndedOn, err := time.Parse("2006-01-02", req.EndedOn)
		if err != nil {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_custody_ended_on"})
			return
		}
		if parsedEndedOn.Before(startedOn) {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_custody_interval"})
			return
		}
		endedOn = req.EndedOn
	}

	var item PortfolioCustodyEvent
	err = s.pool.QueryRow(r.Context(), `
		insert into education_portfolio_custody (
			portfolio_id, event_type, holder_name, holder_role, location_label, access_reason, started_on, ended_on,
			access_mode, sensitive_data_access, institution_id, notes
		)
		select ep.id, $2, $3, $4, $5, $6, $7, $8, $9, $10, ep.institution_id, $11
		from education_portfolios ep
		where ep.id = $1 and ep.institution_id = $12
		returning id::text, portfolio_id::text, event_type, holder_name, holder_role, location_label, access_reason,
			to_char(started_on, 'YYYY-MM-DD'), coalesce(to_char(ended_on, 'YYYY-MM-DD'), ''), access_mode, sensitive_data_access, institution_id, notes
	`, recordID, req.EventType, req.HolderName, req.HolderRole, req.LocationLabel, req.AccessReason, req.StartedOn, endedOn, req.AccessMode, req.SensitiveDataAccess, req.Notes, s.institutionID(r)).Scan(
		&item.ID, &item.PortfolioID, &item.EventType, &item.HolderName, &item.HolderRole, &item.LocationLabel, &item.AccessReason, &item.StartedOn, &item.EndedOn, &item.AccessMode, &item.SensitiveDataAccess, &item.InstitutionID, &item.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_portfolio_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_custody_create_failed"})
		return
	}
	s.logAudit(r, "education.portfolios.custody.create", "portfolio_custody_event", item.ID, "Portfolio custody event created.", map[string]any{
		"portfolio_id": item.PortfolioID,
		"event_type":   item.EventType,
		"holder_name":  item.HolderName,
		"access_mode":  item.AccessMode,
	})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdatePortfolioCustodyEvent(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var req CreatePortfolioCustodyEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_custody_payload"})
		return
	}

	normalizePortfolioCustodyRequest(&req)
	if req.EventType == "" || req.HolderName == "" || req.HolderRole == "" || req.LocationLabel == "" || req.AccessReason == "" || req.StartedOn == "" || req.AccessMode == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_portfolio_custody_fields"})
		return
	}
	if !containsString([]string{"preluare", "consultare", "transfer", "arhivare", "restituire"}, req.EventType) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_custody_event_type"})
		return
	}
	if !containsString([]string{"fizic", "digital", "mixt"}, req.AccessMode) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_custody_access_mode"})
		return
	}
	startedOn, err := time.Parse("2006-01-02", req.StartedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_custody_started_on"})
		return
	}
	endedOn := any(nil)
	if req.EndedOn != "" {
		parsedEndedOn, err := time.Parse("2006-01-02", req.EndedOn)
		if err != nil {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_custody_ended_on"})
			return
		}
		if parsedEndedOn.Before(startedOn) {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_custody_interval"})
			return
		}
		endedOn = req.EndedOn
	}

	var item PortfolioCustodyEvent
	err = s.pool.QueryRow(r.Context(), `
		update education_portfolio_custody
		set event_type = $1, holder_name = $2, holder_role = $3, location_label = $4, access_reason = $5,
			started_on = $6, ended_on = $7, access_mode = $8, sensitive_data_access = $9, notes = $10, updated_at = now()
		where id = $11 and portfolio_id = $12 and institution_id = $13
		returning id::text, portfolio_id::text, event_type, holder_name, holder_role, location_label, access_reason,
			to_char(started_on, 'YYYY-MM-DD'), coalesce(to_char(ended_on, 'YYYY-MM-DD'), ''), access_mode, sensitive_data_access, institution_id, notes
	`, req.EventType, req.HolderName, req.HolderRole, req.LocationLabel, req.AccessReason, req.StartedOn, endedOn, req.AccessMode, req.SensitiveDataAccess, req.Notes, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID, &item.PortfolioID, &item.EventType, &item.HolderName, &item.HolderRole, &item.LocationLabel, &item.AccessReason, &item.StartedOn, &item.EndedOn, &item.AccessMode, &item.SensitiveDataAccess, &item.InstitutionID, &item.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_portfolio_custody_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_custody_update_failed"})
		return
	}
	s.logAudit(r, "education.portfolios.custody.update", "portfolio_custody_event", item.ID, "Portfolio custody event updated.", map[string]any{
		"portfolio_id": item.PortfolioID,
		"event_type":   item.EventType,
		"holder_name":  item.HolderName,
		"access_mode":  item.AccessMode,
	})
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeletePortfolioCustodyEvent(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	tag, err := s.pool.Exec(r.Context(), `delete from education_portfolio_custody where id = $1 and portfolio_id = $2 and institution_id = $3`, itemID, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_custody_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_portfolio_custody_not_found")
		return
	}
	s.logAudit(r, "education.portfolios.custody.delete", "portfolio_custody_event", itemID, "Portfolio custody event deleted.", map[string]any{"portfolio_id": recordID})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) PublicationRecords(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"domain":               {},
		"entity_type":          {},
		"entity_label":         {},
		"publication_channel":  {},
		"publication_status":   {},
		"anonymization_status": {},
	}, []string{"publication_code", "domain", "entity_type", "entity_label", "publication_channel", "publication_status", "anonymization_status"})
	if query.Sort == "" {
		query.Sort = "published_on"
	}

	whereClause, args := buildPublicationFilters(query.Filters, s.institutionID(r))
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_publications epu "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_publications_failed"})
		return
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select id::text, publication_code, domain, entity_type, entity_label, publication_channel, publication_status,
			anonymization_status, mandatory, coalesce(to_char(published_on, 'YYYY-MM-DD'), ''), reviewed_by, institution_id, notes
		from education_publications epu
		%s
		order by %s %s, publication_code desc
		limit $%d offset $%d
	`, whereClause, publicationSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_publications_failed"})
		return
	}
	defer rows.Close()

	items := make([]PublicationRecord, 0, query.PageSize)
	for rows.Next() {
		var item PublicationRecord
		if err := rows.Scan(&item.ID, &item.PublicationCode, &item.Domain, &item.EntityType, &item.EntityLabel, &item.PublicationChannel, &item.PublicationStatus, &item.AnonymizationStatus, &item.Mandatory, &item.PublishedOn, &item.ReviewedBy, &item.InstitutionID, &item.Notes); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_publications_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_publications_scan_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) PublicationRecordDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var item PublicationRecord
	err := s.pool.QueryRow(r.Context(), `
		select id::text, publication_code, domain, entity_type, entity_label, publication_channel, publication_status,
			anonymization_status, mandatory, coalesce(to_char(published_on, 'YYYY-MM-DD'), ''), reviewed_by, institution_id, notes
		from education_publications
		where id = $1 and institution_id = $2
	`, recordID, s.institutionID(r)).Scan(&item.ID, &item.PublicationCode, &item.Domain, &item.EntityType, &item.EntityLabel, &item.PublicationChannel, &item.PublicationStatus, &item.AnonymizationStatus, &item.Mandatory, &item.PublishedOn, &item.ReviewedBy, &item.InstitutionID, &item.Notes)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_publication_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_publication_detail_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreatePublicationRecord(w http.ResponseWriter, r *http.Request) {
	var req CreatePublicationRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_publication_payload"})
		return
	}

	normalizePublicationRequest(&req)
	if req.Domain == "" || req.EntityType == "" || req.EntityLabel == "" || req.PublicationChannel == "" || req.PublicationStatus == "" || req.AnonymizationStatus == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_publication_fields"})
		return
	}
	if !containsString([]string{"guvernanta", "documente_manageriale", "portofolii", "regulamente", "conformitate"}, req.Domain) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_publication_domain"})
		return
	}
	if !containsString([]string{"hotarare", "proces_verbal", "procedura_portofoliu", "rof", "roi", "pdi_pas", "raport", "anunt"}, req.EntityType) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_publication_entity_type"})
		return
	}
	if !containsString([]string{"site_public", "avizier", "intranet", "registratura"}, req.PublicationChannel) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_publication_channel"})
		return
	}
	if !containsString([]string{"pregatit", "publicat", "retras"}, req.PublicationStatus) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_publication_status"})
		return
	}
	if !containsString([]string{"necesara", "finalizata", "nu_este_necesara"}, req.AnonymizationStatus) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_publication_anonymization_status"})
		return
	}
	publishedOn := any(nil)
	if req.PublishedOn != "" {
		if _, err := time.Parse("2006-01-02", req.PublishedOn); err != nil {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_publication_published_on"})
			return
		}
		publishedOn = req.PublishedOn
	}
	if req.PublicationStatus == "publicat" && publishedOn == nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_publication_published_on"})
		return
	}

	code := fmt.Sprintf("PUB-%d-%04d", time.Now().UTC().Year(), time.Now().Unix()%10000)
	var item PublicationRecord
	err := s.pool.QueryRow(r.Context(), `
		insert into education_publications (
			publication_code, domain, entity_type, entity_label, publication_channel, publication_status, anonymization_status,
			mandatory, published_on, reviewed_by, institution_id, notes
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		returning id::text, publication_code, domain, entity_type, entity_label, publication_channel, publication_status,
			anonymization_status, mandatory, coalesce(to_char(published_on, 'YYYY-MM-DD'), ''), reviewed_by, institution_id, notes
	`, code, req.Domain, req.EntityType, req.EntityLabel, req.PublicationChannel, req.PublicationStatus, req.AnonymizationStatus, req.Mandatory, publishedOn, req.ReviewedBy, s.institutionID(r), req.Notes).Scan(
		&item.ID, &item.PublicationCode, &item.Domain, &item.EntityType, &item.EntityLabel, &item.PublicationChannel, &item.PublicationStatus, &item.AnonymizationStatus, &item.Mandatory, &item.PublishedOn, &item.ReviewedBy, &item.InstitutionID, &item.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "publication_create_failed"})
		return
	}
	s.logAudit(r, "education.publication.create", "education_publication", item.ID, "Publication record created.", map[string]any{
		"publication_code":   item.PublicationCode,
		"domain":             item.Domain,
		"entity_type":        item.EntityType,
		"publication_status": item.PublicationStatus,
	})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdatePublicationRecord(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreatePublicationRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_publication_payload"})
		return
	}

	normalizePublicationRequest(&req)
	if req.Domain == "" || req.EntityType == "" || req.EntityLabel == "" || req.PublicationChannel == "" || req.PublicationStatus == "" || req.AnonymizationStatus == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_publication_fields"})
		return
	}
	if !containsString([]string{"guvernanta", "documente_manageriale", "portofolii", "regulamente", "conformitate"}, req.Domain) ||
		!containsString([]string{"hotarare", "proces_verbal", "procedura_portofoliu", "rof", "roi", "pdi_pas", "raport", "anunt"}, req.EntityType) ||
		!containsString([]string{"site_public", "avizier", "intranet", "registratura"}, req.PublicationChannel) ||
		!containsString([]string{"pregatit", "publicat", "retras"}, req.PublicationStatus) ||
		!containsString([]string{"necesara", "finalizata", "nu_este_necesara"}, req.AnonymizationStatus) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_publication_fields"})
		return
	}
	publishedOn := any(nil)
	if req.PublishedOn != "" {
		if _, err := time.Parse("2006-01-02", req.PublishedOn); err != nil {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_publication_published_on"})
			return
		}
		publishedOn = req.PublishedOn
	}
	if req.PublicationStatus == "publicat" && publishedOn == nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_publication_published_on"})
		return
	}

	var item PublicationRecord
	err := s.pool.QueryRow(r.Context(), `
		update education_publications
		set domain = $1, entity_type = $2, entity_label = $3, publication_channel = $4, publication_status = $5,
			anonymization_status = $6, mandatory = $7, published_on = $8, reviewed_by = $9, notes = $10, updated_at = now()
		where id = $11 and institution_id = $12
		returning id::text, publication_code, domain, entity_type, entity_label, publication_channel, publication_status,
			anonymization_status, mandatory, coalesce(to_char(published_on, 'YYYY-MM-DD'), ''), reviewed_by, institution_id, notes
	`, req.Domain, req.EntityType, req.EntityLabel, req.PublicationChannel, req.PublicationStatus, req.AnonymizationStatus, req.Mandatory, publishedOn, req.ReviewedBy, req.Notes, recordID, s.institutionID(r)).Scan(
		&item.ID, &item.PublicationCode, &item.Domain, &item.EntityType, &item.EntityLabel, &item.PublicationChannel, &item.PublicationStatus, &item.AnonymizationStatus, &item.Mandatory, &item.PublishedOn, &item.ReviewedBy, &item.InstitutionID, &item.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_publication_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "publication_update_failed"})
		return
	}
	s.logAudit(r, "education.publication.update", "education_publication", item.ID, "Publication record updated.", map[string]any{
		"publication_code":   item.PublicationCode,
		"domain":             item.Domain,
		"entity_type":        item.EntityType,
		"publication_status": item.PublicationStatus,
	})
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeletePublicationRecord(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	tag, err := s.pool.Exec(r.Context(), `delete from education_publications where id = $1 and institution_id = $2`, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "publication_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_publication_not_found")
		return
	}
	s.logAudit(r, "education.publication.delete", "education_publication", recordID, "Publication record deleted.", nil)
	w.WriteHeader(http.StatusNoContent)
}

func buildGovernanceMinuteFilters(filters map[string]string, meetingID string, institutionID string) (string, []any) {
	where := []string{"emm.meeting_id = $1", "emm.institution_id = $2"}
	args := []any{meetingID, institutionID}
	for key, column := range map[string]string{
		"topic_title":       "emm.topic_title",
		"follow_up_status":  "emm.follow_up_status",
		"responsible_party": "emm.responsible_party",
	} {
		if value := strings.TrimSpace(filters[key]); value != "" {
			args = append(args, "%"+strings.ToLower(value)+"%")
			where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
		}
	}
	if value := strings.TrimSpace(filters["requires_publication"]); value != "" {
		args = append(args, value == "true")
		where = append(where, fmt.Sprintf("emm.requires_publication = $%d", len(args)))
	}
	return "where " + strings.Join(where, " and "), args
}

func buildPortfolioOpisFilters(filters map[string]string, recordID string, institutionID string) (string, []any) {
	where := []string{"epo.portfolio_id = $1", "epo.institution_id = $2"}
	args := []any{recordID, institutionID}
	for key, column := range map[string]string{
		"section_code":       "epo.section_code",
		"component_code":     "epo.component_code",
		"entry_title":        "epo.entry_title",
		"source_scope":       "epo.source_scope",
		"document_reference": "epo.document_reference",
	} {
		if value := strings.TrimSpace(filters[key]); value != "" {
			args = append(args, "%"+strings.ToLower(value)+"%")
			where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
		}
	}
	return "where " + strings.Join(where, " and "), args
}

func buildPortfolioCustodyFilters(filters map[string]string, recordID string, institutionID string) (string, []any) {
	where := []string{"epc.portfolio_id = $1", "epc.institution_id = $2"}
	args := []any{recordID, institutionID}
	for key, column := range map[string]string{
		"event_type":     "epc.event_type",
		"holder_name":    "epc.holder_name",
		"holder_role":    "epc.holder_role",
		"location_label": "epc.location_label",
		"access_mode":    "epc.access_mode",
	} {
		if value := strings.TrimSpace(filters[key]); value != "" {
			args = append(args, "%"+strings.ToLower(value)+"%")
			where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
		}
	}
	return "where " + strings.Join(where, " and "), args
}

func buildPublicationFilters(filters map[string]string, institutionID string) (string, []any) {
	where := []string{"epu.institution_id = $1"}
	args := []any{institutionID}
	for key, column := range map[string]string{
		"domain":               "epu.domain",
		"entity_type":          "epu.entity_type",
		"entity_label":         "epu.entity_label",
		"publication_channel":  "epu.publication_channel",
		"publication_status":   "epu.publication_status",
		"anonymization_status": "epu.anonymization_status",
		"publication_code":     "epu.publication_code",
	} {
		if value := strings.TrimSpace(filters[key]); value != "" {
			args = append(args, "%"+strings.ToLower(value)+"%")
			where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
		}
	}
	return "where " + strings.Join(where, " and "), args
}

func governanceMinuteSortColumn(value string) string {
	switch value {
	case "agenda_order":
		return "emm.agenda_order"
	case "topic_title":
		return "emm.topic_title"
	case "responsible_party":
		return "emm.responsible_party"
	case "due_on":
		return "emm.due_on"
	case "follow_up_status":
		return "emm.follow_up_status"
	default:
		return "emm.agenda_order"
	}
}

func portfolioOpisSortColumn(value string) string {
	switch value {
	case "section_code":
		return "epo.section_code"
	case "component_code":
		return "epo.component_code"
	case "entry_title":
		return "epo.entry_title"
	case "source_scope":
		return "epo.source_scope"
	case "chronological_index":
		return "epo.chronological_index"
	case "checked_on":
		return "epo.checked_on"
	default:
		return "epo.chronological_index"
	}
}

func portfolioCustodySortColumn(value string) string {
	switch value {
	case "event_type":
		return "epc.event_type"
	case "holder_name":
		return "epc.holder_name"
	case "holder_role":
		return "epc.holder_role"
	case "location_label":
		return "epc.location_label"
	case "started_on":
		return "epc.started_on"
	case "ended_on":
		return "epc.ended_on"
	case "access_mode":
		return "epc.access_mode"
	default:
		return "epc.started_on"
	}
}

func publicationSortColumn(value string) string {
	switch value {
	case "publication_code":
		return "epu.publication_code"
	case "domain":
		return "epu.domain"
	case "entity_type":
		return "epu.entity_type"
	case "entity_label":
		return "epu.entity_label"
	case "publication_channel":
		return "epu.publication_channel"
	case "publication_status":
		return "epu.publication_status"
	case "anonymization_status":
		return "epu.anonymization_status"
	case "published_on":
		return "epu.published_on"
	default:
		return "epu.published_on"
	}
}

func normalizeGovernanceMinuteRequest(req *CreateGovernanceMinuteItemRequest) {
	req.TopicTitle = strings.TrimSpace(req.TopicTitle)
	req.DiscussionSummary = strings.TrimSpace(req.DiscussionSummary)
	req.DecisionSummary = strings.TrimSpace(req.DecisionSummary)
	req.ResponsibleParty = strings.TrimSpace(req.ResponsibleParty)
	req.DueOn = strings.TrimSpace(req.DueOn)
	req.FollowUpStatus = strings.TrimSpace(req.FollowUpStatus)
	req.Notes = strings.TrimSpace(req.Notes)
}

func normalizePortfolioOpisRequest(req *CreatePortfolioOpisEntryRequest) {
	req.SectionCode = strings.TrimSpace(req.SectionCode)
	req.ComponentCode = strings.TrimSpace(req.ComponentCode)
	req.EntryTitle = strings.TrimSpace(req.EntryTitle)
	req.SourceScope = strings.TrimSpace(req.SourceScope)
	req.DocumentReference = strings.TrimSpace(req.DocumentReference)
	req.CheckedOn = strings.TrimSpace(req.CheckedOn)
	req.CheckedBy = strings.TrimSpace(req.CheckedBy)
	req.Notes = strings.TrimSpace(req.Notes)
}

func normalizePortfolioCustodyRequest(req *CreatePortfolioCustodyEventRequest) {
	req.EventType = strings.TrimSpace(req.EventType)
	req.HolderName = strings.TrimSpace(req.HolderName)
	req.HolderRole = strings.TrimSpace(req.HolderRole)
	req.LocationLabel = strings.TrimSpace(req.LocationLabel)
	req.AccessReason = strings.TrimSpace(req.AccessReason)
	req.StartedOn = strings.TrimSpace(req.StartedOn)
	req.EndedOn = strings.TrimSpace(req.EndedOn)
	req.AccessMode = strings.TrimSpace(req.AccessMode)
	req.Notes = strings.TrimSpace(req.Notes)
}

func normalizePublicationRequest(req *CreatePublicationRecordRequest) {
	req.Domain = strings.TrimSpace(req.Domain)
	req.EntityType = strings.TrimSpace(req.EntityType)
	req.EntityLabel = strings.TrimSpace(req.EntityLabel)
	req.PublicationChannel = strings.TrimSpace(req.PublicationChannel)
	req.PublicationStatus = strings.TrimSpace(req.PublicationStatus)
	req.AnonymizationStatus = strings.TrimSpace(req.AnonymizationStatus)
	req.PublishedOn = strings.TrimSpace(req.PublishedOn)
	req.ReviewedBy = strings.TrimSpace(req.ReviewedBy)
	req.Notes = strings.TrimSpace(req.Notes)
}
