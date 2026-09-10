package education

var schoolClassSortColumns = map[string]string{
	"class_code": "class_row.class_code", "class_name": "class_row.class_name", "school_year": "class_row.school_year", "grade_level": "class_row.grade_level", "active": "class_row.active",
}
var schoolStudentSortColumns = map[string]string{
	"student_code": "student.student_code", "first_name": "student.first_name", "last_name": "student.last_name", "status": "student.status", "birth_date": "student.birth_date",
}
var schoolEnrolmentSortColumns = map[string]string{
	"enrolled_from": "enrolment.enrolled_from", "enrolled_until": "enrolment.enrolled_until", "status": "enrolment.status", "student_name": "student.last_name", "class_name": "class_row.class_name",
}
var schoolHomeroomSortColumns = map[string]string{
	"assigned_from": "assignment.assigned_from", "assigned_until": "assignment.assigned_until", "teacher_name": "user_row.name", "class_name": "class_row.class_name",
}

func schoolSortColumn(mapping map[string]string, requested, fallback string) string {
	if column, ok := mapping[requested]; ok {
		return column
	}
	return mapping[fallback]
}
