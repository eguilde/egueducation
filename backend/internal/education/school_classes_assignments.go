package education

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/eguilde/egueducation/internal/httpx"
)

const schoolEnrolmentColumns = `enrolment.id::text, enrolment.student_id::text, enrolment.class_id::text, concat_ws(' ', student.first_name, student.last_name), class_row.class_name, to_char(enrolment.enrolled_from, 'YYYY-MM-DD'), coalesce(to_char(enrolment.enrolled_until, 'YYYY-MM-DD'), ''), enrolment.status`
const schoolHomeroomColumns = `assignment.id::text, assignment.class_id::text, class_row.class_name, assignment.personnel_id::text, assignment.app_user_id::text, user_row.name, to_char(assignment.assigned_from, 'YYYY-MM-DD'), coalesce(to_char(assignment.assigned_until, 'YYYY-MM-DD'), '')`

func scanSchoolEnrolment(row pgx.Row, item *SchoolEnrolment) error {
	return row.Scan(&item.ID, &item.StudentID, &item.ClassID, &item.StudentName, &item.ClassName, &item.EnrolledFrom, &item.EnrolledUntil, &item.Status)
}
func scanSchoolHomeroom(row pgx.Row, item *SchoolHomeroomAssignment) error {
	return row.Scan(&item.ID, &item.ClassID, &item.ClassName, &item.PersonnelID, &item.AppUserID, &item.TeacherName, &item.AssignedFrom, &item.AssignedUntil)
}

func normalizeSchoolEnrolment(req *CreateSchoolEnrolmentRequest) bool {
	req.StudentID = strings.TrimSpace(req.StudentID)
	req.ClassID = strings.TrimSpace(req.ClassID)
	req.EnrolledFrom = strings.TrimSpace(req.EnrolledFrom)
	req.EnrolledUntil = strings.TrimSpace(req.EnrolledUntil)
	req.Status = strings.TrimSpace(req.Status)
	if req.Status == "" {
		req.Status = "active"
	}
	return req.StudentID != "" && req.ClassID != "" && req.EnrolledFrom != "" && containsString([]string{"active", "transferred", "completed", "withdrawn"}, req.Status)
}
func normalizeSchoolHomeroom(req *CreateSchoolHomeroomAssignmentRequest) bool {
	req.ClassID = strings.TrimSpace(req.ClassID)
	req.PersonnelID = strings.TrimSpace(req.PersonnelID)
	req.AppUserID = strings.TrimSpace(req.AppUserID)
	req.AssignedFrom = strings.TrimSpace(req.AssignedFrom)
	req.AssignedUntil = strings.TrimSpace(req.AssignedUntil)
	return req.ClassID != "" && req.PersonnelID != "" && req.AppUserID != "" && req.AssignedFrom != ""
}
func schoolClassFlowConflict(w http.ResponseWriter, err error) bool {
	var p *pgconn.PgError
	if errors.As(err, &p) && (p.Code == "23505" || p.Code == "23P01") {
		httpx.JSON(w, 409, map[string]any{"code": "education_class_temporal_conflict"})
		return true
	}
	return false
}

func (s *Service) SchoolEnrolments(w http.ResponseWriter, r *http.Request) {
	_, all, ok := s.requireSchoolClassesAccess(w, r, false)
	if !ok {
		return
	}
	q := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{"enrolled_from": {}, "enrolled_until": {}, "status": {}, "student_name": {}, "class_name": {}}, []string{"enrolled_from", "enrolled_until", "status", "student_name", "class_name", "class_id", "student_id"})
	if q.Sort == "" {
		q.Sort = "enrolled_from"
	}
	where := " where enrolment.institution_id=$1 and enrolment.tenant_code=public.current_tenant_code()"
	args := []any{s.institutionID(r)}
	for _, f := range []string{"class_id", "student_id"} {
		if v := strings.TrimSpace(q.Filters[f]); v != "" {
			args = append(args, v)
			where += fmt.Sprintf(" and enrolment.%s=$%d::uuid", f, len(args))
		}
	}
	if !all {
		where += " and public.education_classes_current_roster_enrolment(enrolment.id)"
	}
	if v := strings.TrimSpace(q.Filters["status"]); v != "" {
		args = append(args, "%"+strings.ToLower(v)+"%")
		where += fmt.Sprintf(" and lower(enrolment.status) like $%d", len(args))
	}
	if v := strings.TrimSpace(q.Filters["enrolled_from"]); v != "" {
		args = append(args, v)
		where += fmt.Sprintf(" and enrolment.enrolled_from = $%d::date", len(args))
	}
	if v := strings.TrimSpace(q.Filters["enrolled_until"]); v != "" {
		args = append(args, v)
		where += fmt.Sprintf(" and enrolment.enrolled_until = $%d::date", len(args))
	}
	if v := strings.TrimSpace(q.Filters["student_name"]); v != "" {
		args = append(args, "%"+strings.ToLower(v)+"%")
		where += fmt.Sprintf(" and (lower(concat_ws(' ', student.first_name, student.last_name)) like $%d or lower(concat_ws(' ', student.last_name, student.first_name)) like $%d)", len(args), len(args))
	}
	if v := strings.TrimSpace(q.Filters["class_name"]); v != "" {
		args = append(args, "%"+strings.ToLower(v)+"%")
		where += fmt.Sprintf(" and lower(class_row.class_name) like $%d", len(args))
	}
	base := " from education_student_enrolments enrolment join education_students student on student.id=enrolment.student_id join education_school_classes class_row on class_row.id=enrolment.class_id"
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*)"+base+where, args...).Scan(&total); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_enrolments_list_failed"})
		return
	}
	sort := schoolSortColumn(schoolEnrolmentSortColumns, q.Sort, "enrolled_from")
	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf("select %s%s%s order by %s %s,enrolment.id limit $%d offset $%d", schoolEnrolmentColumns, base, where, sort, q.Direction, len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_enrolments_list_failed"})
		return
	}
	defer rows.Close()
	items := []SchoolEnrolment{}
	for rows.Next() {
		var item SchoolEnrolment
		if scanSchoolEnrolment(rows, &item) != nil {
			httpx.JSON(w, 500, map[string]any{"code": "education_enrolments_list_failed"})
			return
		}
		items = append(items, item)
	}
	if rows.Err() != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_enrolments_list_failed"})
		return
	}
	httpx.WritePage(w, 200, items, total, q.Page, q.PageSize)
}
func (s *Service) SchoolEnrolmentDetail(w http.ResponseWriter, r *http.Request) {
	_, all, ok := s.requireSchoolClassesAccess(w, r, false)
	if !ok {
		return
	}
	base := " from education_student_enrolments enrolment join education_students student on student.id=enrolment.student_id join education_school_classes class_row on class_row.id=enrolment.class_id"
	where := " where enrolment.id=$1::uuid and enrolment.institution_id=$2 and enrolment.tenant_code=public.current_tenant_code()"
	args := []any{chi.URLParam(r, "enrolmentID"), s.institutionID(r)}
	if !all {
		where += " and public.education_classes_current_roster_enrolment(enrolment.id)"
	}
	var item SchoolEnrolment
	err := scanSchoolEnrolment(s.pool.QueryRow(r.Context(), "select "+schoolEnrolmentColumns+base+where, args...), &item)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_enrolment_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_enrolment_detail_failed"})
		return
	}
	httpx.JSON(w, 200, item)
}
func (s *Service) schoolEnrolmentRecord(r *http.Request, id string) (SchoolEnrolment, error) {
	var item SchoolEnrolment
	err := scanSchoolEnrolment(s.pool.QueryRow(r.Context(), "select "+schoolEnrolmentColumns+" from education_student_enrolments enrolment join education_students student on student.id=enrolment.student_id join education_school_classes class_row on class_row.id=enrolment.class_id where enrolment.id=$1::uuid and enrolment.institution_id=$2 and enrolment.tenant_code=public.current_tenant_code()", id, s.institutionID(r)), &item)
	return item, err
}
func (s *Service) CreateSchoolEnrolment(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireSchoolClassesAccess(w, r, true); !ok {
		return
	}
	var req CreateSchoolEnrolmentRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil || !normalizeSchoolEnrolment(&req) {
		httpx.JSON(w, 400, map[string]any{"code": "education_enrolment_payload_invalid"})
		return
	}
	var id string
	err := s.pool.QueryRow(r.Context(), "insert into education_student_enrolments(tenant_code,institution_id,student_id,class_id,enrolled_from,enrolled_until,status) values(public.current_tenant_code(),$1,$2::uuid,$3::uuid,$4::date,nullif($5,'')::date,$6) returning id::text", s.institutionID(r), req.StudentID, req.ClassID, req.EnrolledFrom, req.EnrolledUntil, req.Status).Scan(&id)
	if schoolClassFlowConflict(w, err) {
		return
	}
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_enrolment_create_failed"})
		return
	}
	item, err := s.schoolEnrolmentRecord(r, id)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_enrolment_response_failed"})
		return
	}
	s.logAudit(r, "education.enrolments.create", "education_student_enrolment", item.ID, "Student enrolled in class.", map[string]any{"class_id": item.ClassID, "student_id": item.StudentID})
	httpx.JSON(w, 201, item)
}
func (s *Service) UpdateSchoolEnrolment(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireSchoolClassesAccess(w, r, true); !ok {
		return
	}
	var req CreateSchoolEnrolmentRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil || !normalizeSchoolEnrolment(&req) {
		httpx.JSON(w, 400, map[string]any{"code": "education_enrolment_payload_invalid"})
		return
	}
	var id string
	err := s.pool.QueryRow(r.Context(), "update education_student_enrolments set student_id=$1::uuid,class_id=$2::uuid,enrolled_from=$3::date,enrolled_until=nullif($4,'')::date,status=$5,updated_at=now() where id=$6::uuid and institution_id=$7 and tenant_code=public.current_tenant_code() returning id::text", req.StudentID, req.ClassID, req.EnrolledFrom, req.EnrolledUntil, req.Status, chi.URLParam(r, "enrolmentID"), s.institutionID(r)).Scan(&id)
	if schoolClassFlowConflict(w, err) {
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_enrolment_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_enrolment_update_failed"})
		return
	}
	item, err := s.schoolEnrolmentRecord(r, id)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_enrolment_response_failed"})
		return
	}
	httpx.JSON(w, 200, item)
}
func (s *Service) DeleteSchoolEnrolment(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireSchoolClassesAccess(w, r, true); !ok {
		return
	}
	var id string
	err := s.pool.QueryRow(r.Context(), "update education_student_enrolments set status='withdrawn',enrolled_until=coalesce(enrolled_until,greatest(enrolled_from,current_date)),updated_at=now() where id=$1::uuid and institution_id=$2 and tenant_code=public.current_tenant_code() and status='active' returning id::text", chi.URLParam(r, "enrolmentID"), s.institutionID(r)).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_enrolment_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_enrolment_withdraw_failed"})
		return
	}
	item, err := s.schoolEnrolmentRecord(r, id)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_enrolment_response_failed"})
		return
	}
	s.logAudit(r, "education.enrolments.withdraw", "education_student_enrolment", id, "Student enrolment closed without deleting history.", nil)
	httpx.JSON(w, 200, item)
}

func (s *Service) SchoolHomeroomAssignments(w http.ResponseWriter, r *http.Request) {
	_, all, ok := s.requireSchoolClassesAccess(w, r, false)
	if !ok {
		return
	}
	q := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{"assigned_from": {}, "assigned_until": {}, "teacher_name": {}, "class_name": {}}, []string{"assigned_from", "assigned_until", "teacher_name", "class_name", "class_id"})
	if q.Sort == "" {
		q.Sort = "assigned_from"
	}
	where := " where assignment.institution_id=$1 and assignment.tenant_code=public.current_tenant_code()"
	args := []any{s.institutionID(r)}
	if v := strings.TrimSpace(q.Filters["class_id"]); v != "" {
		args = append(args, v)
		where += fmt.Sprintf(" and assignment.class_id=$%d::uuid", len(args))
	}
	if !all {
		where += " and public.education_classes_actor_owns_current_homeroom(assignment.id)"
	}
	if v := strings.TrimSpace(q.Filters["teacher_name"]); v != "" {
		args = append(args, "%"+strings.ToLower(v)+"%")
		where += fmt.Sprintf(" and lower(user_row.name) like $%d", len(args))
	}
	if v := strings.TrimSpace(q.Filters["class_name"]); v != "" {
		args = append(args, "%"+strings.ToLower(v)+"%")
		where += fmt.Sprintf(" and lower(class_row.class_name) like $%d", len(args))
	}
	if v := strings.TrimSpace(q.Filters["assigned_from"]); v != "" {
		args = append(args, v)
		where += fmt.Sprintf(" and assignment.assigned_from=$%d::date", len(args))
	}
	if v := strings.TrimSpace(q.Filters["assigned_until"]); v != "" {
		args = append(args, v)
		where += fmt.Sprintf(" and assignment.assigned_until = $%d::date", len(args))
	}
	base := " from education_class_homeroom_assignments assignment join app_users user_row on user_row.id=assignment.app_user_id join education_school_classes class_row on class_row.id=assignment.class_id"
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*)"+base+where, args...).Scan(&total); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_homeroom_assignments_list_failed"})
		return
	}
	sort := schoolSortColumn(schoolHomeroomSortColumns, q.Sort, "assigned_from")
	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf("select %s%s%s order by %s %s,assignment.id limit $%d offset $%d", schoolHomeroomColumns, base, where, sort, q.Direction, len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_homeroom_assignments_list_failed"})
		return
	}
	defer rows.Close()
	items := []SchoolHomeroomAssignment{}
	for rows.Next() {
		var item SchoolHomeroomAssignment
		if scanSchoolHomeroom(rows, &item) != nil {
			httpx.JSON(w, 500, map[string]any{"code": "education_homeroom_assignments_list_failed"})
			return
		}
		items = append(items, item)
	}
	httpx.WritePage(w, 200, items, total, q.Page, q.PageSize)
}
func (s *Service) SchoolHomeroomAssignmentDetail(w http.ResponseWriter, r *http.Request) {
	_, all, ok := s.requireSchoolClassesAccess(w, r, false)
	if !ok {
		return
	}
	base := " from education_class_homeroom_assignments assignment join app_users user_row on user_row.id=assignment.app_user_id join education_school_classes class_row on class_row.id=assignment.class_id"
	where := " where assignment.id=$1::uuid and assignment.institution_id=$2 and assignment.tenant_code=public.current_tenant_code()"
	args := []any{chi.URLParam(r, "assignmentID"), s.institutionID(r)}
	if !all {
		where += " and public.education_classes_actor_owns_current_homeroom(assignment.id)"
	}
	var item SchoolHomeroomAssignment
	err := scanSchoolHomeroom(s.pool.QueryRow(r.Context(), "select "+schoolHomeroomColumns+base+where, args...), &item)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_homeroom_assignment_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_homeroom_assignment_detail_failed"})
		return
	}
	httpx.JSON(w, 200, item)
}
func (s *Service) schoolHomeroomRecord(r *http.Request, id string) (SchoolHomeroomAssignment, error) {
	var item SchoolHomeroomAssignment
	err := scanSchoolHomeroom(s.pool.QueryRow(r.Context(), "select "+schoolHomeroomColumns+" from education_class_homeroom_assignments assignment join app_users user_row on user_row.id=assignment.app_user_id join education_school_classes class_row on class_row.id=assignment.class_id where assignment.id=$1::uuid and assignment.institution_id=$2 and assignment.tenant_code=public.current_tenant_code()", id, s.institutionID(r)), &item)
	return item, err
}
func (s *Service) CreateSchoolHomeroomAssignment(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireSchoolClassesAccess(w, r, true); !ok {
		return
	}
	var req CreateSchoolHomeroomAssignmentRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil || !normalizeSchoolHomeroom(&req) {
		httpx.JSON(w, 400, map[string]any{"code": "education_homeroom_assignment_payload_invalid"})
		return
	}
	var id string
	err := s.pool.QueryRow(r.Context(), "insert into education_class_homeroom_assignments(tenant_code,institution_id,class_id,personnel_id,app_user_id,assigned_from,assigned_until) values(public.current_tenant_code(),$1,$2::uuid,$3::uuid,$4::uuid,$5::date,nullif($6,'')::date) returning id::text", s.institutionID(r), req.ClassID, req.PersonnelID, req.AppUserID, req.AssignedFrom, req.AssignedUntil).Scan(&id)
	if schoolClassFlowConflict(w, err) {
		return
	}
	if err != nil {
		httpx.JSON(w, 422, map[string]any{"code": "education_homeroom_assignment_create_failed"})
		return
	}
	item, err := s.schoolHomeroomRecord(r, id)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_homeroom_assignment_response_failed"})
		return
	}
	s.logAudit(r, "education.homeroom.assign", "education_class_homeroom_assignment", item.ID, "Homeroom teacher assigned.", map[string]any{"class_id": item.ClassID, "personnel_id": item.PersonnelID})
	httpx.JSON(w, 201, item)
}
func (s *Service) UpdateSchoolHomeroomAssignment(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireSchoolClassesAccess(w, r, true); !ok {
		return
	}
	var req CreateSchoolHomeroomAssignmentRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil || !normalizeSchoolHomeroom(&req) {
		httpx.JSON(w, 400, map[string]any{"code": "education_homeroom_assignment_payload_invalid"})
		return
	}
	var id string
	err := s.pool.QueryRow(r.Context(), "update education_class_homeroom_assignments set class_id=$1::uuid,personnel_id=$2::uuid,app_user_id=$3::uuid,assigned_from=$4::date,assigned_until=nullif($5,'')::date,updated_at=now() where id=$6::uuid and institution_id=$7 and tenant_code=public.current_tenant_code() returning id::text", req.ClassID, req.PersonnelID, req.AppUserID, req.AssignedFrom, req.AssignedUntil, chi.URLParam(r, "assignmentID"), s.institutionID(r)).Scan(&id)
	if schoolClassFlowConflict(w, err) {
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_homeroom_assignment_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, 422, map[string]any{"code": "education_homeroom_assignment_update_failed"})
		return
	}
	item, err := s.schoolHomeroomRecord(r, id)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_homeroom_assignment_response_failed"})
		return
	}
	httpx.JSON(w, 200, item)
}
func (s *Service) DeleteSchoolHomeroomAssignment(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireSchoolClassesAccess(w, r, true); !ok {
		return
	}
	var id string
	err := s.pool.QueryRow(r.Context(), "update education_class_homeroom_assignments set assigned_until=coalesce(assigned_until,greatest(assigned_from,current_date)),updated_at=now() where id=$1::uuid and institution_id=$2 and tenant_code=public.current_tenant_code() and assigned_until is null returning id::text", chi.URLParam(r, "assignmentID"), s.institutionID(r)).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_homeroom_assignment_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_homeroom_assignment_close_failed"})
		return
	}
	item, err := s.schoolHomeroomRecord(r, id)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_homeroom_assignment_response_failed"})
		return
	}
	s.logAudit(r, "education.homeroom.close", "education_class_homeroom_assignment", id, "Homeroom assignment closed without deleting history.", nil)
	httpx.JSON(w, 200, item)
}
