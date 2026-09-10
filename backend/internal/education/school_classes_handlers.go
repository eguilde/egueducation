package education

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/eguilde/egueducation/internal/httpx"
)

const schoolClassColumns = `id::text, tenant_code, institution_id, class_code, class_name, school_year, grade_level, study_shift, active`
const schoolStudentColumns = `id::text, tenant_code, institution_id, student_code, first_name, last_name, status, coalesce(to_char(birth_date, 'YYYY-MM-DD'), '')`

func scanSchoolClass(row pgx.Row, item *SchoolClass) error {
	return row.Scan(&item.ID, &item.TenantCode, &item.InstitutionID, &item.ClassCode, &item.ClassName, &item.SchoolYear, &item.GradeLevel, &item.StudyShift, &item.Active)
}
func scanSchoolStudent(row pgx.Row, item *SchoolStudent) error {
	return row.Scan(&item.ID, &item.TenantCode, &item.InstitutionID, &item.StudentCode, &item.FirstName, &item.LastName, &item.Status, &item.BirthDate)
}

func (s *Service) SchoolClasses(w http.ResponseWriter, r *http.Request) {
	_, all, ok := s.requireSchoolClassesAccess(w, r, false)
	if !ok {
		return
	}
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{"class_code": {}, "class_name": {}, "school_year": {}, "grade_level": {}, "active": {}}, []string{"class_code", "class_name", "school_year", "grade_level", "active"})
	if query.Sort == "" {
		query.Sort = "class_name"
	}
	where := " where class_row.institution_id = $1 and class_row.tenant_code = public.current_tenant_code()"
	args := []any{s.institutionID(r)}
	if !all {
		// Keep the application predicate identical to the FORCE-RLS predicate.
		// The helper resolves the authenticated subject, tenant, effective
		// membership and effective homeroom assignment in one authoritative place.
		where += " and public.education_classes_actor_is_assigned(class_row.id)"
	}
	for _, field := range []string{"class_code", "class_name", "school_year", "grade_level"} {
		if value := strings.TrimSpace(query.Filters[field]); value != "" {
			args = append(args, "%"+strings.ToLower(value)+"%")
			where += fmt.Sprintf(" and lower(class_row.%s) like $%d", field, len(args))
		}
	}
	if value := strings.TrimSpace(query.Filters["active"]); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "education_classes_active_filter_invalid"})
			return
		}
		args = append(args, parsed)
		where += fmt.Sprintf(" and class_row.active = $%d", len(args))
	}
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_school_classes class_row"+where, args...).Scan(&total); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_classes_list_failed"})
		return
	}
	sort := schoolSortColumn(schoolClassSortColumns, query.Sort, "class_name")
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf("select %s from education_school_classes class_row%s order by %s %s, class_row.id limit $%d offset $%d", schoolClassColumns, where, sort, query.Direction, len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_classes_list_failed"})
		return
	}
	defer rows.Close()
	items := make([]SchoolClass, 0, query.PageSize)
	for rows.Next() {
		var item SchoolClass
		if err := scanSchoolClass(rows, &item); err != nil {
			httpx.JSON(w, 500, map[string]any{"code": "education_classes_list_failed"})
			return
		}
		items = append(items, item)
	}
	if rows.Err() != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_classes_list_failed"})
		return
	}
	httpx.WritePage(w, 200, items, total, query.Page, query.PageSize)
}

func (s *Service) SchoolClassDetail(w http.ResponseWriter, r *http.Request) {
	_, all, ok := s.requireSchoolClassesAccess(w, r, false)
	if !ok {
		return
	}
	id := strings.TrimSpace(chi.URLParam(r, "classID"))
	where := "where class_row.id=$1::uuid and class_row.institution_id=$2 and class_row.tenant_code=public.current_tenant_code()"
	args := []any{id, s.institutionID(r)}
	if !all {
		where += " and public.education_classes_actor_is_assigned(class_row.id)"
	}
	var item SchoolClass
	err := scanSchoolClass(s.pool.QueryRow(r.Context(), "select "+schoolClassColumns+" from education_school_classes class_row "+where, args...), &item)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_class_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_class_detail_failed"})
		return
	}
	httpx.JSON(w, 200, item)
}

func normalizeSchoolClassRequest(req *CreateSchoolClassRequest) bool {
	req.ClassCode = strings.TrimSpace(req.ClassCode)
	req.ClassName = strings.TrimSpace(req.ClassName)
	req.SchoolYear = strings.TrimSpace(req.SchoolYear)
	req.GradeLevel = strings.TrimSpace(req.GradeLevel)
	req.StudyShift = strings.TrimSpace(req.StudyShift)
	if req.StudyShift == "" {
		req.StudyShift = "day"
	}
	return req.ClassCode != "" && req.ClassName != "" && req.SchoolYear != "" && req.GradeLevel != "" && containsString([]string{"day", "afternoon", "evening"}, req.StudyShift)
}
func (s *Service) CreateSchoolClass(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireSchoolClassesAccess(w, r, true); !ok {
		return
	}
	var req CreateSchoolClassRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil || !normalizeSchoolClassRequest(&req) {
		httpx.JSON(w, 400, map[string]any{"code": "education_class_payload_invalid"})
		return
	}
	var item SchoolClass
	err := scanSchoolClass(s.pool.QueryRow(r.Context(), "insert into education_school_classes(tenant_code,institution_id,class_code,class_name,school_year,grade_level,study_shift,active) values(public.current_tenant_code(),$1,$2,$3,$4,$5,$6,$7) returning "+schoolClassColumns, s.institutionID(r), req.ClassCode, req.ClassName, req.SchoolYear, req.GradeLevel, req.StudyShift, req.Active), &item)
	if schoolClassConflict(w, err) {
		return
	}
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_class_create_failed"})
		return
	}
	s.logAudit(r, "education.classes.create", "education_school_class", item.ID, "School class created.", map[string]any{"class_code": item.ClassCode})
	httpx.JSON(w, 201, item)
}
func (s *Service) UpdateSchoolClass(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireSchoolClassesAccess(w, r, true); !ok {
		return
	}
	var req CreateSchoolClassRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil || !normalizeSchoolClassRequest(&req) {
		httpx.JSON(w, 400, map[string]any{"code": "education_class_payload_invalid"})
		return
	}
	var item SchoolClass
	err := scanSchoolClass(s.pool.QueryRow(r.Context(), "update education_school_classes set class_code=$1,class_name=$2,school_year=$3,grade_level=$4,study_shift=$5,active=$6,updated_at=now() where id=$7::uuid and institution_id=$8 and tenant_code=public.current_tenant_code() returning "+schoolClassColumns, req.ClassCode, req.ClassName, req.SchoolYear, req.GradeLevel, req.StudyShift, req.Active, chi.URLParam(r, "classID"), s.institutionID(r)), &item)
	if schoolClassConflict(w, err) {
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_class_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_class_update_failed"})
		return
	}
	s.logAudit(r, "education.classes.update", "education_school_class", item.ID, "School class updated.", nil)
	httpx.JSON(w, 200, item)
}
func (s *Service) DeleteSchoolClass(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireSchoolClassesAccess(w, r, true); !ok {
		return
	}
	var item SchoolClass
	err := scanSchoolClass(s.pool.QueryRow(r.Context(), "update education_school_classes set active=false,updated_at=now() where id=$1::uuid and institution_id=$2 and tenant_code=public.current_tenant_code() and active returning "+schoolClassColumns, chi.URLParam(r, "classID"), s.institutionID(r)), &item)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_class_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_class_retire_failed"})
		return
	}
	s.logAudit(r, "education.classes.retire", "education_school_class", item.ID, "School class retired without deleting its history.", nil)
	httpx.JSON(w, http.StatusOK, item)
}
func schoolClassConflict(w http.ResponseWriter, err error) bool {
	var p *pgconn.PgError
	if errors.As(err, &p) && p.Code == "23505" {
		httpx.JSON(w, 409, map[string]any{"code": "education_class_conflict"})
		return true
	}
	return false
}

func normalizeSchoolStudentRequest(req *CreateSchoolStudentRequest) bool {
	req.StudentCode = strings.TrimSpace(req.StudentCode)
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)
	req.Status = strings.TrimSpace(req.Status)
	if req.Status == "" {
		req.Status = "active"
	}
	return req.StudentCode != "" && req.FirstName != "" && req.LastName != "" && containsString([]string{"active", "transferred", "graduated", "withdrawn"}, req.Status)
}
func (s *Service) SchoolStudents(w http.ResponseWriter, r *http.Request) {
	_, all, ok := s.requireSchoolClassesAccess(w, r, false)
	if !ok {
		return
	}
	q := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{"student_code": {}, "first_name": {}, "last_name": {}, "status": {}, "birth_date": {}}, []string{"student_code", "first_name", "last_name", "status", "birth_date", "class_id"})
	if q.Sort == "" {
		q.Sort = "last_name"
	}
	where := " where student.institution_id=$1 and student.tenant_code=public.current_tenant_code()"
	args := []any{s.institutionID(r)}
	if classID := strings.TrimSpace(q.Filters["class_id"]); classID != "" {
		args = append(args, classID)
		if all {
			where += fmt.Sprintf(" and exists(select 1 from education_student_enrolments enrolment where enrolment.student_id=student.id and enrolment.class_id=$%d::uuid)", len(args))
		} else {
			where += fmt.Sprintf(" and exists(select 1 from education_student_enrolments enrolment where enrolment.student_id=student.id and enrolment.class_id=$%d::uuid and public.education_classes_current_roster_enrolment(enrolment.id))", len(args))
		}
	}
	if !all {
		where += " and exists(select 1 from education_student_enrolments enrolment where enrolment.student_id=student.id and public.education_classes_current_roster_enrolment(enrolment.id))"
	}
	for _, f := range []string{"student_code", "first_name", "last_name", "status"} {
		if v := strings.TrimSpace(q.Filters[f]); v != "" {
			args = append(args, "%"+strings.ToLower(v)+"%")
			where += fmt.Sprintf(" and lower(student.%s) like $%d", f, len(args))
		}
	}
	if value := strings.TrimSpace(q.Filters["birth_date"]); value != "" {
		args = append(args, value)
		where += fmt.Sprintf(" and student.birth_date = $%d::date", len(args))
	}
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_students student"+where, args...).Scan(&total); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_students_list_failed"})
		return
	}
	sort := schoolSortColumn(schoolStudentSortColumns, q.Sort, "last_name")
	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf("select %s from education_students student%s order by %s %s,student.id limit $%d offset $%d", schoolStudentColumns, where, sort, q.Direction, len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_students_list_failed"})
		return
	}
	defer rows.Close()
	items := []SchoolStudent{}
	for rows.Next() {
		var x SchoolStudent
		if scanSchoolStudent(rows, &x) != nil {
			httpx.JSON(w, 500, map[string]any{"code": "education_students_list_failed"})
			return
		}
		items = append(items, x)
	}
	httpx.WritePage(w, 200, items, total, q.Page, q.PageSize)
}
func (s *Service) SchoolStudentDetail(w http.ResponseWriter, r *http.Request) {
	_, all, ok := s.requireSchoolClassesAccess(w, r, false)
	if !ok {
		return
	}
	where := " where student.id=$1::uuid and student.institution_id=$2 and student.tenant_code=public.current_tenant_code()"
	args := []any{chi.URLParam(r, "studentID"), s.institutionID(r)}
	if !all {
		where += " and exists(select 1 from education_student_enrolments enrolment where enrolment.student_id=student.id and public.education_classes_current_roster_enrolment(enrolment.id))"
	}
	var item SchoolStudent
	err := scanSchoolStudent(s.pool.QueryRow(r.Context(), "select "+schoolStudentColumns+" from education_students student"+where, args...), &item)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_student_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_student_detail_failed"})
		return
	}
	httpx.JSON(w, 200, item)
}
func (s *Service) CreateSchoolStudent(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireSchoolClassesAccess(w, r, true); !ok {
		return
	}
	var req CreateSchoolStudentRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil || !normalizeSchoolStudentRequest(&req) {
		httpx.JSON(w, 400, map[string]any{"code": "education_student_payload_invalid"})
		return
	}
	var item SchoolStudent
	err := scanSchoolStudent(s.pool.QueryRow(r.Context(), "insert into education_students(tenant_code,institution_id,student_code,first_name,last_name,status,birth_date) values(public.current_tenant_code(),$1,$2,$3,$4,$5,nullif($6,'')::date) returning "+schoolStudentColumns, s.institutionID(r), req.StudentCode, req.FirstName, req.LastName, req.Status, strings.TrimSpace(req.BirthDate)), &item)
	if schoolClassConflict(w, err) {
		return
	}
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_student_create_failed"})
		return
	}
	s.logAudit(r, "education.students.create", "education_student", item.ID, "School student created.", map[string]any{"student_code": item.StudentCode})
	httpx.JSON(w, 201, item)
}
func (s *Service) UpdateSchoolStudent(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireSchoolClassesAccess(w, r, true); !ok {
		return
	}
	var req CreateSchoolStudentRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil || !normalizeSchoolStudentRequest(&req) {
		httpx.JSON(w, 400, map[string]any{"code": "education_student_payload_invalid"})
		return
	}
	var item SchoolStudent
	err := scanSchoolStudent(s.pool.QueryRow(r.Context(), "update education_students set student_code=$1,first_name=$2,last_name=$3,status=$4,birth_date=nullif($5,'')::date,updated_at=now() where id=$6::uuid and institution_id=$7 and tenant_code=public.current_tenant_code() returning "+schoolStudentColumns, req.StudentCode, req.FirstName, req.LastName, req.Status, strings.TrimSpace(req.BirthDate), chi.URLParam(r, "studentID"), s.institutionID(r)), &item)
	if schoolClassConflict(w, err) {
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_student_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_student_update_failed"})
		return
	}
	s.logAudit(r, "education.students.update", "education_student", item.ID, "School student updated.", nil)
	httpx.JSON(w, 200, item)
}
func (s *Service) DeleteSchoolStudent(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireSchoolClassesAccess(w, r, true); !ok {
		return
	}
	var item SchoolStudent
	err := scanSchoolStudent(s.pool.QueryRow(r.Context(), "update education_students set status='withdrawn',updated_at=now() where id=$1::uuid and institution_id=$2 and tenant_code=public.current_tenant_code() and status <> 'withdrawn' returning "+schoolStudentColumns, chi.URLParam(r, "studentID"), s.institutionID(r)), &item)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_student_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_student_withdraw_failed"})
		return
	}
	s.logAudit(r, "education.students.withdraw", "education_student", item.ID, "School student withdrawn without deleting history.", nil)
	httpx.JSON(w, http.StatusOK, item)
}
