package education

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type CommitteeRecord struct {
	ID                string `json:"id"`
	CommitteeCode     string `json:"committee_code"`
	SchoolYear        string `json:"school_year"`
	CommitteeType     string `json:"committee_type"`
	Title             string `json:"title"`
	Status            string `json:"status"`
	DecisionReference string `json:"decision_reference"`
	StartsOn          string `json:"starts_on"`
	EndsOn            string `json:"ends_on"`
	EvaluationScope   bool   `json:"evaluation_scope"`
	InstitutionID     string `json:"institution_id"`
	Notes             string `json:"notes"`
}

type CreateCommitteeRecordRequest struct {
	SchoolYear        string `json:"school_year"`
	CommitteeType     string `json:"committee_type"`
	Title             string `json:"title"`
	Status            string `json:"status"`
	DecisionReference string `json:"decision_reference"`
	StartsOn          string `json:"starts_on"`
	EndsOn            string `json:"ends_on"`
	EvaluationScope   bool   `json:"evaluation_scope"`
	Notes             string `json:"notes"`
}

type CommitteeMember struct {
	ID            string `json:"id"`
	CommitteeID   string `json:"committee_id"`
	FullName      string `json:"full_name"`
	RoleName      string `json:"role_name"`
	MemberType    string `json:"member_type"`
	VotingRight   bool   `json:"voting_right"`
	Status        string `json:"status"`
	AppointedOn   string `json:"appointed_on"`
	ReleasedOn    string `json:"released_on"`
	InstitutionID string `json:"institution_id"`
	Notes         string `json:"notes"`
}

type CreateCommitteeMemberRequest struct {
	FullName    string `json:"full_name"`
	RoleName    string `json:"role_name"`
	MemberType  string `json:"member_type"`
	VotingRight bool   `json:"voting_right"`
	Status      string `json:"status"`
	AppointedOn string `json:"appointed_on"`
	ReleasedOn  string `json:"released_on"`
	Notes       string `json:"notes"`
}

type CommitteeCompletenessSummary struct {
	Committee  CommitteeRecord                    `json:"committee"`
	Membership CommitteeCompletenessMemberBlock   `json:"membership"`
	Readiness  CommitteeCompletenessReadiness     `json:"readiness"`
}

type CommitteeCompletenessMemberBlock struct {
	ActiveMembers      int      `json:"active_members"`
	VotingMembers      int      `json:"voting_members"`
	ChairpersonCovered bool     `json:"chairperson_covered"`
	SecretaryCovered   bool     `json:"secretary_covered"`
	MemberNames        []string `json:"member_names"`
}

type CommitteeCompletenessReadiness struct {
	ReadyForOperation bool     `json:"ready_for_operation"`
	Blockers          []string `json:"blockers"`
}

func (s *Service) Committees(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"school_year":    {},
		"committee_type": {},
		"title":          {},
		"status":         {},
	}, []string{"committee_code", "school_year", "committee_type", "title", "status", "starts_on"})
	if query.Sort == "" {
		query.Sort = "starts_on"
	}
	whereClause, args := buildCommitteeFilters(query.Filters, s.institutionID(r))
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_committees ec "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_committees_failed"})
		return
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select
			ec.id::text, ec.committee_code, ec.school_year, ec.committee_type, ec.title, ec.status, ec.decision_reference,
			to_char(ec.starts_on, 'YYYY-MM-DD'), coalesce(to_char(ec.ends_on, 'YYYY-MM-DD'), ''), ec.evaluation_scope, ec.institution_id, ec.notes
		from education_committees ec
		%s
		order by %s %s, ec.committee_code
		limit $%d offset $%d
	`, whereClause, committeeSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_committees_failed"})
		return
	}
	defer rows.Close()
	items := make([]CommitteeRecord, 0, query.PageSize)
	for rows.Next() {
		var item CommitteeRecord
		if err := rows.Scan(&item.ID, &item.CommitteeCode, &item.SchoolYear, &item.CommitteeType, &item.Title, &item.Status, &item.DecisionReference, &item.StartsOn, &item.EndsOn, &item.EvaluationScope, &item.InstitutionID, &item.Notes); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_committees_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_committees_scan_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) CommitteeDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var item CommitteeRecord
	err := s.pool.QueryRow(r.Context(), `
		select id::text, committee_code, school_year, committee_type, title, status, decision_reference,
			to_char(starts_on, 'YYYY-MM-DD'), coalesce(to_char(ends_on, 'YYYY-MM-DD'), ''), evaluation_scope, institution_id, notes
		from education_committees where id = $1 and institution_id = $2
	`, recordID, s.institutionID(r)).Scan(&item.ID, &item.CommitteeCode, &item.SchoolYear, &item.CommitteeType, &item.Title, &item.Status, &item.DecisionReference, &item.StartsOn, &item.EndsOn, &item.EvaluationScope, &item.InstitutionID, &item.Notes)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_committee_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_committee_detail_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreateCommittee(w http.ResponseWriter, r *http.Request) {
	var req CreateCommitteeRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_committee_payload"})
		return
	}
	normalizeCommitteeRequest(&req)
	if req.SchoolYear == "" || req.CommitteeType == "" || req.Title == "" || req.Status == "" || req.StartsOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_committee_fields"})
		return
	}
	if !inStringSet(req.CommitteeType, "permanenta", "temporara", "evaluare_personal_didactic", "curriculum", "mentorat", "securitate", "burse", "alta") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_committee_type"})
		return
	}
	if !inStringSet(req.Status, "draft", "active", "completed", "archived") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_committee_status"})
		return
	}
	startsOn, err := time.Parse("2006-01-02", req.StartsOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_committee_starts_on"})
		return
	}
	_ = startsOn
	endsOn, err := parseOptionalEducationDate(req.EndsOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_committee_ends_on"})
		return
	}
	code := fmt.Sprintf("COM-%d-%04d", time.Now().Year(), time.Now().Nanosecond()%10000)
	var item CommitteeRecord
	err = s.pool.QueryRow(r.Context(), `
		insert into education_committees (
			committee_code, school_year, committee_type, title, status, decision_reference, starts_on, ends_on, evaluation_scope, institution_id, notes
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		returning id::text, committee_code, school_year, committee_type, title, status, decision_reference,
			to_char(starts_on, 'YYYY-MM-DD'), coalesce(to_char(ends_on, 'YYYY-MM-DD'), ''), evaluation_scope, institution_id, notes
	`, code, req.SchoolYear, req.CommitteeType, req.Title, req.Status, req.DecisionReference, req.StartsOn, endsOn, req.EvaluationScope, s.institutionID(r), req.Notes).Scan(
		&item.ID, &item.CommitteeCode, &item.SchoolYear, &item.CommitteeType, &item.Title, &item.Status, &item.DecisionReference, &item.StartsOn, &item.EndsOn, &item.EvaluationScope, &item.InstitutionID, &item.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "committee_create_failed"})
		return
	}
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdateCommittee(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreateCommitteeRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_committee_payload"})
		return
	}
	normalizeCommitteeRequest(&req)
	if req.SchoolYear == "" || req.CommitteeType == "" || req.Title == "" || req.Status == "" || req.StartsOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_committee_fields"})
		return
	}
	if !inStringSet(req.CommitteeType, "permanenta", "temporara", "evaluare_personal_didactic", "curriculum", "mentorat", "securitate", "burse", "alta") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_committee_type"})
		return
	}
	if !inStringSet(req.Status, "draft", "active", "completed", "archived") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_committee_status"})
		return
	}
	endsOn, err := parseOptionalEducationDate(req.EndsOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_committee_ends_on"})
		return
	}
	var item CommitteeRecord
	err = s.pool.QueryRow(r.Context(), `
		update education_committees
		set school_year=$1, committee_type=$2, title=$3, status=$4, decision_reference=$5, starts_on=$6, ends_on=$7, evaluation_scope=$8, notes=$9, updated_at=now()
		where id = $10 and institution_id = $11
		returning id::text, committee_code, school_year, committee_type, title, status, decision_reference,
			to_char(starts_on, 'YYYY-MM-DD'), coalesce(to_char(ends_on, 'YYYY-MM-DD'), ''), evaluation_scope, institution_id, notes
	`, req.SchoolYear, req.CommitteeType, req.Title, req.Status, req.DecisionReference, req.StartsOn, endsOn, req.EvaluationScope, req.Notes, recordID, s.institutionID(r)).Scan(
		&item.ID, &item.CommitteeCode, &item.SchoolYear, &item.CommitteeType, &item.Title, &item.Status, &item.DecisionReference, &item.StartsOn, &item.EndsOn, &item.EvaluationScope, &item.InstitutionID, &item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_committee_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "committee_update_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteCommittee(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	tag, err := s.pool.Exec(r.Context(), `delete from education_committees where id = $1 and institution_id = $2`, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "committee_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_committee_not_found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) CommitteeMembers(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"full_name":   {},
		"role_name":   {},
		"member_type": {},
		"status":      {},
	}, []string{"full_name", "role_name", "member_type", "status", "appointed_on"})
	if query.Sort == "" {
		query.Sort = "appointed_on"
	}
	whereClause, args := buildCommitteeMemberFilters(query.Filters, recordID, s.institutionID(r))
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_committee_members ecm "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_committee_members_failed"})
		return
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select id::text, committee_id::text, full_name, role_name, member_type, voting_right, status,
			to_char(appointed_on, 'YYYY-MM-DD'), coalesce(to_char(released_on, 'YYYY-MM-DD'), ''), institution_id, notes
		from education_committee_members ecm
		%s
		order by %s %s, full_name
		limit $%d offset $%d
	`, whereClause, committeeMemberSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_committee_members_failed"})
		return
	}
	defer rows.Close()
	items := make([]CommitteeMember, 0, query.PageSize)
	for rows.Next() {
		var item CommitteeMember
		if err := rows.Scan(&item.ID, &item.CommitteeID, &item.FullName, &item.RoleName, &item.MemberType, &item.VotingRight, &item.Status, &item.AppointedOn, &item.ReleasedOn, &item.InstitutionID, &item.Notes); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_committee_members_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_committee_members_scan_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) CommitteeMemberDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var item CommitteeMember
	err := s.pool.QueryRow(r.Context(), `
		select id::text, committee_id::text, full_name, role_name, member_type, voting_right, status,
			to_char(appointed_on, 'YYYY-MM-DD'), coalesce(to_char(released_on, 'YYYY-MM-DD'), ''), institution_id, notes
		from education_committee_members
		where id = $1 and committee_id = $2 and institution_id = $3
	`, itemID, recordID, s.institutionID(r)).Scan(&item.ID, &item.CommitteeID, &item.FullName, &item.RoleName, &item.MemberType, &item.VotingRight, &item.Status, &item.AppointedOn, &item.ReleasedOn, &item.InstitutionID, &item.Notes)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_committee_member_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_committee_member_detail_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreateCommitteeMember(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreateCommitteeMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_committee_member_payload"})
		return
	}
	normalizeCommitteeMemberRequest(&req)
	if req.FullName == "" || req.RoleName == "" || req.MemberType == "" || req.Status == "" || req.AppointedOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_committee_member_fields"})
		return
	}
	if !inStringSet(req.MemberType, "presedinte", "secretar", "membru", "observator", "invitat") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_committee_member_type"})
		return
	}
	if !inStringSet(req.Status, "active", "inactive", "replaced") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_committee_member_status"})
		return
	}
	releasedOn, err := parseOptionalEducationDate(req.ReleasedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_committee_member_released_on"})
		return
	}
	var item CommitteeMember
	err = s.pool.QueryRow(r.Context(), `
		insert into education_committee_members (
			committee_id, full_name, role_name, member_type, voting_right, status, appointed_on, released_on, institution_id, notes
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		returning id::text, committee_id::text, full_name, role_name, member_type, voting_right, status,
			to_char(appointed_on, 'YYYY-MM-DD'), coalesce(to_char(released_on, 'YYYY-MM-DD'), ''), institution_id, notes
	`, recordID, req.FullName, req.RoleName, req.MemberType, req.VotingRight, req.Status, req.AppointedOn, releasedOn, s.institutionID(r), req.Notes).Scan(
		&item.ID, &item.CommitteeID, &item.FullName, &item.RoleName, &item.MemberType, &item.VotingRight, &item.Status, &item.AppointedOn, &item.ReleasedOn, &item.InstitutionID, &item.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "committee_member_create_failed"})
		return
	}
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdateCommitteeMember(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var req CreateCommitteeMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_committee_member_payload"})
		return
	}
	normalizeCommitteeMemberRequest(&req)
	if req.FullName == "" || req.RoleName == "" || req.MemberType == "" || req.Status == "" || req.AppointedOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_committee_member_fields"})
		return
	}
	if !inStringSet(req.MemberType, "presedinte", "secretar", "membru", "observator", "invitat") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_committee_member_type"})
		return
	}
	if !inStringSet(req.Status, "active", "inactive", "replaced") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_committee_member_status"})
		return
	}
	releasedOn, err := parseOptionalEducationDate(req.ReleasedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_committee_member_released_on"})
		return
	}
	var item CommitteeMember
	err = s.pool.QueryRow(r.Context(), `
		update education_committee_members
		set full_name=$1, role_name=$2, member_type=$3, voting_right=$4, status=$5, appointed_on=$6, released_on=$7, notes=$8, updated_at=now()
		where id = $9 and committee_id = $10 and institution_id = $11
		returning id::text, committee_id::text, full_name, role_name, member_type, voting_right, status,
			to_char(appointed_on, 'YYYY-MM-DD'), coalesce(to_char(released_on, 'YYYY-MM-DD'), ''), institution_id, notes
	`, req.FullName, req.RoleName, req.MemberType, req.VotingRight, req.Status, req.AppointedOn, releasedOn, req.Notes, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID, &item.CommitteeID, &item.FullName, &item.RoleName, &item.MemberType, &item.VotingRight, &item.Status, &item.AppointedOn, &item.ReleasedOn, &item.InstitutionID, &item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_committee_member_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "committee_member_update_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteCommitteeMember(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	tag, err := s.pool.Exec(r.Context(), `delete from education_committee_members where id = $1 and committee_id = $2 and institution_id = $3`, itemID, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "committee_member_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_committee_member_not_found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) CommitteeCompleteness(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var summary CommitteeCompletenessSummary
	err := s.pool.QueryRow(r.Context(), `
		select id::text, committee_code, school_year, committee_type, title, status, decision_reference,
			to_char(starts_on, 'YYYY-MM-DD'), coalesce(to_char(ends_on, 'YYYY-MM-DD'), ''), evaluation_scope, institution_id, notes
		from education_committees where id = $1 and institution_id = $2
	`, recordID, s.institutionID(r)).Scan(
		&summary.Committee.ID, &summary.Committee.CommitteeCode, &summary.Committee.SchoolYear, &summary.Committee.CommitteeType, &summary.Committee.Title, &summary.Committee.Status, &summary.Committee.DecisionReference, &summary.Committee.StartsOn, &summary.Committee.EndsOn, &summary.Committee.EvaluationScope, &summary.Committee.InstitutionID, &summary.Committee.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_committee_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_committee_completeness_failed"})
		return
	}
	if err := s.pool.QueryRow(r.Context(), `
		select
			count(*) filter (where status = 'active') as active_members,
			count(*) filter (where status = 'active' and voting_right) as voting_members,
			coalesce(bool_or(status = 'active' and member_type = 'presedinte'), false),
			coalesce(bool_or(status = 'active' and member_type = 'secretar'), false)
		from education_committee_members
		where committee_id = $1 and institution_id = $2
	`, recordID, s.institutionID(r)).Scan(&summary.Membership.ActiveMembers, &summary.Membership.VotingMembers, &summary.Membership.ChairpersonCovered, &summary.Membership.SecretaryCovered); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_committee_completeness_failed"})
		return
	}
	rows, err := s.pool.Query(r.Context(), `select full_name from education_committee_members where committee_id = $1 and institution_id = $2 and status = 'active' order by full_name`, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_committee_completeness_failed"})
		return
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_committee_completeness_failed"})
			return
		}
		summary.Membership.MemberNames = append(summary.Membership.MemberNames, name)
	}
	blockers := make([]string, 0)
	if summary.Membership.ActiveMembers == 0 {
		blockers = append(blockers, "no_active_members")
	}
	if !summary.Membership.ChairpersonCovered {
		blockers = append(blockers, "chairperson_missing")
	}
	if !summary.Membership.SecretaryCovered {
		blockers = append(blockers, "secretary_missing")
	}
	if strings.TrimSpace(summary.Committee.DecisionReference) == "" {
		blockers = append(blockers, "decision_reference_missing")
	}
	if summary.Committee.CommitteeType == "evaluare_personal_didactic" && !summary.Committee.EvaluationScope {
		blockers = append(blockers, "evaluation_scope_missing")
	}
	summary.Readiness = CommitteeCompletenessReadiness{
		ReadyForOperation: len(blockers) == 0,
		Blockers:          blockers,
	}
	httpx.JSON(w, http.StatusOK, summary)
}

func buildCommitteeFilters(filters map[string]string, institutionID string) (string, []any) {
	where := []string{"ec.institution_id = $1"}
	args := []any{institutionID}
	for key, column := range map[string]string{"school_year": "ec.school_year", "committee_type": "ec.committee_type", "title": "ec.title", "status": "ec.status"} {
		if value := strings.TrimSpace(filters[key]); value != "" {
			args = append(args, "%"+strings.ToLower(value)+"%")
			where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
		}
	}
	return "where " + strings.Join(where, " and "), args
}

func buildCommitteeMemberFilters(filters map[string]string, recordID, institutionID string) (string, []any) {
	where := []string{"ecm.committee_id = $1", "ecm.institution_id = $2"}
	args := []any{recordID, institutionID}
	for key, column := range map[string]string{"full_name": "ecm.full_name", "role_name": "ecm.role_name", "member_type": "ecm.member_type", "status": "ecm.status"} {
		if value := strings.TrimSpace(filters[key]); value != "" {
			args = append(args, "%"+strings.ToLower(value)+"%")
			where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
		}
	}
	return "where " + strings.Join(where, " and "), args
}

func committeeSortColumn(value string) string {
	switch value {
	case "committee_code":
		return "ec.committee_code"
	case "school_year":
		return "ec.school_year"
	case "committee_type":
		return "ec.committee_type"
	case "title":
		return "ec.title"
	case "status":
		return "ec.status"
	case "starts_on":
		return "ec.starts_on"
	default:
		return "ec.starts_on"
	}
}

func committeeMemberSortColumn(value string) string {
	switch value {
	case "full_name":
		return "ecm.full_name"
	case "role_name":
		return "ecm.role_name"
	case "member_type":
		return "ecm.member_type"
	case "status":
		return "ecm.status"
	case "appointed_on":
		return "ecm.appointed_on"
	default:
		return "ecm.appointed_on"
	}
}

func normalizeCommitteeRequest(req *CreateCommitteeRecordRequest) {
	req.SchoolYear = strings.TrimSpace(req.SchoolYear)
	req.CommitteeType = strings.TrimSpace(req.CommitteeType)
	req.Title = strings.TrimSpace(req.Title)
	req.Status = strings.TrimSpace(req.Status)
	req.DecisionReference = strings.TrimSpace(req.DecisionReference)
	req.StartsOn = strings.TrimSpace(req.StartsOn)
	req.EndsOn = strings.TrimSpace(req.EndsOn)
	req.Notes = strings.TrimSpace(req.Notes)
}

func normalizeCommitteeMemberRequest(req *CreateCommitteeMemberRequest) {
	req.FullName = strings.TrimSpace(req.FullName)
	req.RoleName = strings.TrimSpace(req.RoleName)
	req.MemberType = strings.TrimSpace(req.MemberType)
	req.Status = strings.TrimSpace(req.Status)
	req.AppointedOn = strings.TrimSpace(req.AppointedOn)
	req.ReleasedOn = strings.TrimSpace(req.ReleasedOn)
	req.Notes = strings.TrimSpace(req.Notes)
}
