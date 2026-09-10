//go:build integration

package education

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	appdb "github.com/eguilde/egueducation/internal/db"
)

// TestSchoolClassesNOBYPASSRLS proves that the teacher read-assigned policy
// remains authoritative even if a future HTTP handler accidentally omits its
// contextual predicate. It also exercises the canonical personnel/user pair.
func TestSchoolClassesNOBYPASSRLS(t *testing.T) {
	it := newGovernanceIntegrationDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	admin := openGovernanceIntegrationPool(t, ctx, it.databaseConfig)
	defer admin.Close()
	if err := appdb.Migrate(ctx, admin); err != nil {
		t.Fatalf("migrate School classes database: %v", err)
	}
	grantGovernanceIntegrationAccess(t, ctx, admin, it.roleName)
	for _, table := range []string{"education_school_classes", "education_students", "education_student_enrolments", "education_class_homeroom_assignments"} {
		if _, err := admin.Exec(ctx, "grant select, insert, update, delete on "+table+" to "+quoteGovernanceIdentifier(it.roleName)); err != nil {
			t.Fatalf("grant %s: %v", table, err)
		}
	}
	fixture := seedGovernanceAuthorizationFixture(t, ctx, admin)
	assignedClass, unassignedClass, pastClass, futureClass := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	currentStudent, expiredStudent, futureStudent := uuid.NewString(), uuid.NewString(), uuid.NewString()
	transferredStudent, withdrawnStudent, withdrawnPupilStudent, unassignedStudent := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	if _, err := admin.Exec(ctx, `
		insert into education_school_classes(id,tenant_code,institution_id,class_code,class_name,school_year,grade_level)
		values
			($1::uuid,$12,$13,'CLS-ASSIGNED','IV A','2026-2027','IV'),
			($2::uuid,$12,$13,'CLS-UNASSIGNED','IV B','2026-2027','IV'),
			($3::uuid,$12,$13,'CLS-PAST','IV C','2026-2027','IV'),
			($4::uuid,$12,$13,'CLS-FUTURE','IV D','2026-2027','IV');
		insert into education_students(id,tenant_code,institution_id,student_code,first_name,last_name,status) values
			($5::uuid,$12,$13,'STU-CURRENT','Ana','Current','active'),
			($6::uuid,$12,$13,'STU-EXPIRED','Bela','Expired','active'),
			($7::uuid,$12,$13,'STU-FUTURE','Cora','Future','active'),
			($8::uuid,$12,$13,'STU-TRANSFERRED','Dan','Transferred','active'),
			($9::uuid,$12,$13,'STU-WITHDRAWN','Eva','Withdrawn','active'),
			($10::uuid,$12,$13,'STU-PUPIL-WITHDRAWN','Fia','PupilWithdrawn','withdrawn'),
			($11::uuid,$12,$13,'STU-UNASSIGNED','Gheorghe','Unassigned','active');
		insert into education_student_enrolments(tenant_code,institution_id,student_id,class_id,enrolled_from,enrolled_until,status) values
			($12,$13,$5::uuid,$1::uuid,current_date,null,'active'),
			($12,$13,$6::uuid,$1::uuid,current_date-10,current_date-1,'active'),
			($12,$13,$7::uuid,$1::uuid,current_date+1,null,'active'),
			($12,$13,$8::uuid,$1::uuid,current_date,null,'transferred'),
			($12,$13,$9::uuid,$1::uuid,current_date,null,'withdrawn'),
			($12,$13,$10::uuid,$1::uuid,current_date,null,'active'),
			($12,$13,$11::uuid,$2::uuid,current_date,null,'active');
		insert into education_class_homeroom_assignments(tenant_code,institution_id,class_id,personnel_id,app_user_id,assigned_from,assigned_until) values
			($12,$13,$1::uuid,$14::uuid,$15::uuid,current_date,null),
			($12,$13,$3::uuid,$14::uuid,$15::uuid,current_date-10,current_date-1),
			($12,$13,$4::uuid,$14::uuid,$15::uuid,current_date+1,null)
	`, assignedClass, unassignedClass, pastClass, futureClass, currentStudent, expiredStudent, futureStudent, transferredStudent, withdrawnStudent, withdrawnPupilStudent, unassignedStudent, fixture.tenantA, fixture.institutionA, fixture.memberPersonnelID, fixture.memberUserID); err != nil {
		t.Fatalf("seed class/student/homeroom fixture: %v", err)
	}

	pool := appdb.NewSessionPool(it.readerPool)
	teacherCtx, release := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	defer release()
	var visibleClasses, visibleStudents, visibleEnrolments, visibleAssignments int
	if err := pool.QueryRow(teacherCtx, "select count(*) from education_school_classes").Scan(&visibleClasses); err != nil || visibleClasses != 1 {
		t.Fatalf("assigned teacher class visibility: count=%d err=%v want=1", visibleClasses, err)
	}
	if err := pool.QueryRow(teacherCtx, "select count(*) from education_students").Scan(&visibleStudents); err != nil || visibleStudents != 1 {
		t.Fatalf("assigned teacher student visibility: count=%d err=%v want=1", visibleStudents, err)
	}
	if err := pool.QueryRow(teacherCtx, "select count(*) from education_student_enrolments").Scan(&visibleEnrolments); err != nil || visibleEnrolments != 1 {
		t.Fatalf("assigned teacher current roster visibility: count=%d err=%v want=1", visibleEnrolments, err)
	}
	if err := pool.QueryRow(teacherCtx, "select count(*) from education_class_homeroom_assignments").Scan(&visibleAssignments); err != nil || visibleAssignments != 1 {
		t.Fatalf("assigned teacher current homeroom visibility: count=%d err=%v want=1", visibleAssignments, err)
	}
	for _, hiddenStudentID := range []string{expiredStudent, futureStudent, transferredStudent, withdrawnStudent, withdrawnPupilStudent, unassignedStudent} {
		var count int
		if err := pool.QueryRow(teacherCtx, "select count(*) from education_students where id=$1::uuid", hiddenStudentID).Scan(&count); err != nil || count != 0 {
			t.Fatalf("assigned teacher must not read non-current student %s: count=%d err=%v", hiddenStudentID, count, err)
		}
		if err := pool.QueryRow(teacherCtx, "select count(*) from education_student_enrolments where student_id=$1::uuid", hiddenStudentID).Scan(&count); err != nil || count != 0 {
			t.Fatalf("assigned teacher must not read non-current enrolment for %s: count=%d err=%v", hiddenStudentID, count, err)
		}
	}
	for _, hiddenClassID := range []string{unassignedClass, pastClass, futureClass} {
		var count int
		if err := pool.QueryRow(teacherCtx, "select count(*) from education_school_classes where id=$1::uuid", hiddenClassID).Scan(&count); err != nil || count != 0 {
			t.Fatalf("assigned teacher must not read non-current or unassigned class %s: count=%d err=%v", hiddenClassID, count, err)
		}
	}
	if _, err := pool.Exec(teacherCtx, `insert into education_school_classes(tenant_code,institution_id,class_code,class_name,school_year,grade_level) values($1,$2,'CLS-CROSS','Cross','2026-2027','IV')`, fixture.tenantB, fixture.institutionB); err == nil {
		t.Fatal("NOBYPASSRLS teacher must not insert a foreign-tenant class")
	}
	if _, err := admin.Exec(ctx, `insert into education_class_homeroom_assignments(tenant_code,institution_id,class_id,personnel_id,app_user_id,assigned_from) values($1,$2,$3::uuid,$4::uuid,$5::uuid,current_date)`, fixture.tenantA, fixture.institutionA, unassignedClass, fixture.foreignPersonnelID, fixture.memberUserID); err == nil || !strings.Contains(err.Error(), "canonical") {
		t.Fatalf("spoofed personnel/user homeroom pair must be rejected, err=%v", err)
	}
}
