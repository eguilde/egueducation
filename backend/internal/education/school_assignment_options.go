package education

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/eguilde/egueducation/internal/httpx"
)

const (
	schoolAssignmentOptionClasses  = "classes"
	schoolAssignmentOptionStudents = "students"
	schoolAssignmentOptionTeachers = "teachers"
)

type SchoolAssignmentOption struct {
	Kind        string `json:"kind"`
	ClassID     string `json:"class_id,omitempty"`
	StudentID   string `json:"student_id,omitempty"`
	PersonnelID string `json:"personnel_id,omitempty"`
	AppUserID   string `json:"app_user_id,omitempty"`
	Code        string `json:"code"`
	Name        string `json:"name"`
}

type schoolAssignmentOptionsQuery struct {
	Kind     string
	Search   string
	Page     int
	PageSize int
}

type schoolAssignmentOptionPlan struct {
	selectColumns string
	fromWhere     string
	searchClause  string
	orderBy       string
}

func parseSchoolAssignmentOptionsQuery(values url.Values) (schoolAssignmentOptionsQuery, error) {
	page := httpx.ParsePageQuery(values, map[string]struct{}{}, nil)
	query := schoolAssignmentOptionsQuery{
		Kind:     strings.ToLower(strings.TrimSpace(values.Get("kind"))),
		Search:   strings.TrimSpace(values.Get("q")),
		Page:     page.Page,
		PageSize: page.PageSize,
	}
	if _, err := schoolAssignmentOptionsPlan(query.Kind); err != nil {
		return schoolAssignmentOptionsQuery{}, err
	}
	return query, nil
}

func schoolAssignmentOptionsPlan(kind string) (schoolAssignmentOptionPlan, error) {
	switch kind {
	case schoolAssignmentOptionClasses:
		return schoolAssignmentOptionPlan{
			selectColumns: `'classes', class_row.id::text, '', '', '', class_row.class_code, class_row.class_name`,
			fromWhere: `from education_school_classes class_row
				where class_row.tenant_code = public.current_tenant_code()
				and class_row.institution_id = $1 and class_row.active`,
			searchClause: ` and (lower(class_row.class_code) like $2 or lower(class_row.class_name) like $2)`,
			orderBy:      `lower(class_row.class_name), lower(class_row.class_code), class_row.id`,
		}, nil
	case schoolAssignmentOptionStudents:
		return schoolAssignmentOptionPlan{
			selectColumns: `'students', '', student.id::text, '', '', student.student_code, concat_ws(' ', student.last_name, student.first_name)`,
			fromWhere: `from education_students student
				where student.tenant_code = public.current_tenant_code()
				and student.institution_id = $1 and student.status = 'active'`,
			searchClause: ` and (lower(student.student_code) like $2 or lower(student.first_name) like $2 or lower(student.last_name) like $2 or lower(concat_ws(' ', student.last_name, student.first_name)) like $2)`,
			orderBy:      `lower(student.last_name), lower(student.first_name), student.id`,
		}, nil
	case schoolAssignmentOptionTeachers:
		return schoolAssignmentOptionPlan{
			selectColumns: `'teachers', '', '', person.id::text, person.app_user_id::text, person.employee_code, person.full_name`,
			fromWhere: `from education_personnel person
				where person.tenant_code = public.current_tenant_code()
				and person.institution_id = $1 and person.status = 'active'
				and person.employment_type in ('titular','suplinitor','plata_cu_ora')
				and person.app_user_id is not null
				and public.education_membership_is_eligible(person.app_user_id, public.current_tenant_code(), $1, null)`,
			searchClause: ` and (lower(person.employee_code) like $2 or lower(person.full_name) like $2)`,
			orderBy:      `lower(person.full_name), lower(person.employee_code), person.id`,
		}, nil
	default:
		return schoolAssignmentOptionPlan{}, errors.New("unsupported assignment option kind")
	}
}

// SchoolAssignmentOptions returns display-ready, institution-scoped choices for
// class administration. Tenant and institution are always session-derived.
func (s *Service) SchoolAssignmentOptions(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireSchoolClassesAccess(w, r, true); !ok {
		return
	}
	query, err := parseSchoolAssignmentOptionsQuery(r.URL.Query())
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "education_assignment_options_kind_invalid"})
		return
	}
	plan, _ := schoolAssignmentOptionsPlan(query.Kind)
	args := []any{s.institutionID(r)}
	where := plan.fromWhere
	if query.Search != "" {
		args = append(args, "%"+strings.ToLower(query.Search)+"%")
		where += plan.searchClause
	}
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) "+where, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_assignment_options_failed"})
		return
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf("select %s %s order by %s limit $%d offset $%d", plan.selectColumns, where, plan.orderBy, len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_assignment_options_failed"})
		return
	}
	defer rows.Close()
	items := make([]SchoolAssignmentOption, 0, query.PageSize)
	for rows.Next() {
		var item SchoolAssignmentOption
		if err := rows.Scan(&item.Kind, &item.ClassID, &item.StudentID, &item.PersonnelID, &item.AppUserID, &item.Code, &item.Name); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_assignment_options_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_assignment_options_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}
