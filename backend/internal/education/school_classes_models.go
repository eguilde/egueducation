package education

type SchoolClass struct {
	ID            string `json:"id"`
	TenantCode    string `json:"tenant_code"`
	InstitutionID string `json:"institution_id"`
	ClassCode     string `json:"class_code"`
	ClassName     string `json:"class_name"`
	SchoolYear    string `json:"school_year"`
	GradeLevel    string `json:"grade_level"`
	StudyShift    string `json:"study_shift"`
	Active        bool   `json:"active"`
}

type SchoolStudent struct {
	ID            string `json:"id"`
	TenantCode    string `json:"tenant_code"`
	InstitutionID string `json:"institution_id"`
	StudentCode   string `json:"student_code"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	Status        string `json:"status"`
	BirthDate     string `json:"birth_date,omitempty"`
}

type SchoolEnrolment struct {
	ID            string `json:"id"`
	StudentID     string `json:"student_id"`
	ClassID       string `json:"class_id"`
	StudentName   string `json:"student_name,omitempty"`
	ClassName     string `json:"class_name,omitempty"`
	EnrolledFrom  string `json:"enrolled_from"`
	EnrolledUntil string `json:"enrolled_until,omitempty"`
	Status        string `json:"status"`
}

type SchoolHomeroomAssignment struct {
	ID            string `json:"id"`
	ClassID       string `json:"class_id"`
	ClassName     string `json:"class_name,omitempty"`
	PersonnelID   string `json:"personnel_id"`
	AppUserID     string `json:"app_user_id"`
	TeacherName   string `json:"teacher_name,omitempty"`
	AssignedFrom  string `json:"assigned_from"`
	AssignedUntil string `json:"assigned_until,omitempty"`
}

type CreateSchoolClassRequest struct {
	ClassCode  string `json:"class_code"`
	ClassName  string `json:"class_name"`
	SchoolYear string `json:"school_year"`
	GradeLevel string `json:"grade_level"`
	StudyShift string `json:"study_shift"`
	Active     bool   `json:"active"`
}

type CreateSchoolStudentRequest struct {
	StudentCode string `json:"student_code"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Status      string `json:"status"`
	BirthDate   string `json:"birth_date"`
}

type CreateSchoolEnrolmentRequest struct {
	StudentID     string `json:"student_id"`
	ClassID       string `json:"class_id"`
	EnrolledFrom  string `json:"enrolled_from"`
	EnrolledUntil string `json:"enrolled_until"`
	Status        string `json:"status"`
}

type CreateSchoolHomeroomAssignmentRequest struct {
	ClassID       string `json:"class_id"`
	PersonnelID   string `json:"personnel_id"`
	AppUserID     string `json:"app_user_id"`
	AssignedFrom  string `json:"assigned_from"`
	AssignedUntil string `json:"assigned_until"`
}
