package db

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSchoolRegulatoryPolicyRLSIntegration(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("EGUEDUCATION_TEST_DATABASE_URL"))
	if databaseURL == "" {
		databaseURL = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	}
	if databaseURL == "" {
		t.Skip("school regulatory PostgreSQL integration test requires TEST_DATABASE_URL")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	it := newTenantGrantRLSIntegrationDatabase(t, ctx, databaseURL)
	adminPool, err := pgxpool.NewWithConfig(ctx, it.databaseConfig.Copy())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(adminPool.Close)
	if err := Migrate(ctx, adminPool); err != nil {
		t.Fatalf("apply migrations including regulatory foundation: %v", err)
	}

	quotedRole := tenantGrantQuoteIdentifier(it.roleName)
	for _, statement := range []string{
		"grant usage on schema public to " + quotedRole,
		"grant select, insert, update, delete on school_institution_profiles, school_policy_pack_versions, school_policy_assignments, school_policy_overrides, school_policy_evaluations to " + quotedRole,
		"grant select, insert, update, delete on education_publications to " + quotedRole,
	} {
		if _, err := adminPool.Exec(ctx, statement); err != nil {
			t.Fatalf("grant restricted regulatory access: %v", err)
		}
	}

	applicationConfig := it.databaseConfig.Copy()
	applicationConfig.ConnConfig.User = it.roleName
	applicationConfig.ConnConfig.Password = it.rolePassword
	applicationConfig.MaxConns = 1
	applicationPool, err := pgxpool.NewWithConfig(ctx, applicationConfig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(applicationPool.Close)
	sessionPool := NewSessionPool(applicationPool)

	for _, scope := range []struct{ tenant, institution string }{{"tenant-egueducation", "inst-001"}, {"tenant-balotesti", "inst-balotesti"}} {
		requestContext, release, err := AcquireRequestConn(ctx, applicationPool, SessionConfig{TenantID: scope.tenant, InstitutionID: scope.institution, ActorSubject: "rls-test"})
		if err != nil {
			t.Fatal(err)
		}
		var profileID string
		if err := sessionPool.QueryRow(requestContext, `insert into school_institution_profiles(tenant_code,institution_id,version,status,school_legal_form,regulatory_profile,effective_from,source_reference,approved_by_subject,approved_at,created_by_subject,updated_by_subject) values($1,$2,2,'active','public','ro.public.preuniversity',current_date,'test-approval','rls-test',now(),'rls-test','rls-test') returning id::text`, scope.tenant, scope.institution).Scan(&profileID); err != nil {
			release()
			t.Fatal(err)
		}
		if _, err := sessionPool.Exec(requestContext, `insert into school_institution_profiles(tenant_code,institution_id,version,status,school_legal_form,regulatory_profile,effective_from,source_reference,approved_by_subject,approved_at,created_by_subject,updated_by_subject) values($1,$2,3,'approved','private','ro.private.preuniversity',current_date,'overlap','rls-test',now(),'rls-test','rls-test')`, scope.tenant, scope.institution); !isPostgresCode(err, "23P01") {
			release()
			t.Fatalf("overlapping effective profile was not rejected: %v", err)
		}
		var assignmentID string
		if err := sessionPool.QueryRow(requestContext, `insert into school_policy_assignments(tenant_code,institution_id,policy_pack_version_id,profile_id,profile_version,pack_code,assignment_kind,status,effective_from,assigned_by_subject,created_by_subject,updated_by_subject) select $1,$2,id,$3,2,pack_code,'common','active',current_date,'rls-test','rls-test','rls-test' from school_policy_pack_versions where pack_code='common.ro' returning id::text`, scope.tenant, scope.institution, profileID).Scan(&assignmentID); err != nil {
			release()
			t.Fatal(err)
		}
		if _, err := sessionPool.Exec(requestContext, `insert into school_policy_overrides(tenant_code,institution_id,policy_assignment_id,key,value,justification,status,created_by_subject,updated_by_subject) values($1,$2,$3,'capabilities.education.publication.manage.reason','"test"','integration','draft','rls-test','rls-test')`, scope.tenant, scope.institution, assignmentID); err != nil {
			release()
			t.Fatal(err)
		}
		var evaluationID string
		if err := sessionPool.QueryRow(requestContext, `insert into school_policy_evaluations(tenant_code,institution_id,profile_id,profile_version,evaluated_by_subject,capabilities,blocked,checksum_sha256,created_by_subject,updated_by_subject) values($1,$2,$3,2,'rls-test','[]',false,repeat('a',64),'rls-test','rls-test') returning id::text`, scope.tenant, scope.institution, profileID).Scan(&evaluationID); err != nil {
			release()
			t.Fatal(err)
		}
		if _, err := sessionPool.Exec(requestContext, `update school_policy_evaluations set blocked=true where id=$1`, evaluationID); err == nil {
			release()
			t.Fatal("immutable evaluation accepted update")
		}
		publicationCode := "PUB-" + scope.tenant
		if _, err := sessionPool.Exec(requestContext, `insert into education_publications(publication_code,domain,entity_type,entity_label,publication_channel,publication_status,anonymization_status,institution_id,tenant_code,policy_evaluation_id) values($1,'conformitate','anunt','RLS policy proof','intranet','pregatit','nu_este_necesara',$2,$3,$4)`, publicationCode, scope.institution, scope.tenant, evaluationID); err != nil {
			release()
			t.Fatalf("scoped policy-evaluated publication insert failed: %v", err)
		}
		if _, err := sessionPool.Exec(requestContext, `insert into education_publications(publication_code,domain,entity_type,entity_label,publication_channel,publication_status,anonymization_status,institution_id,tenant_code) values($1,'conformitate','anunt','Missing evaluation','intranet','pregatit','nu_este_necesara',$2,$3)`, "PUB-NO-POLICY-"+scope.tenant, scope.institution, scope.tenant); err == nil {
			release()
			t.Fatal("publication without policy evaluation was accepted")
		}

		var profiles, packs, assignments, overrides, evaluations int
		if err := sessionPool.QueryRow(requestContext, `select count(*) from school_institution_profiles where status <> 'superseded'`).Scan(&profiles); err != nil {
			release()
			t.Fatal(err)
		}
		if err := sessionPool.QueryRow(requestContext, `select count(*) from school_policy_pack_versions`).Scan(&packs); err != nil {
			release()
			t.Fatal(err)
		}
		if err := sessionPool.QueryRow(requestContext, `select count(*) from school_policy_assignments`).Scan(&assignments); err != nil {
			release()
			t.Fatal(err)
		}
		if err := sessionPool.QueryRow(requestContext, `select count(*) from school_policy_overrides`).Scan(&overrides); err != nil {
			release()
			t.Fatal(err)
		}
		if err := sessionPool.QueryRow(requestContext, `select count(*) from school_policy_evaluations`).Scan(&evaluations); err != nil {
			release()
			t.Fatal(err)
		}
		if profiles != 2 || packs != 5 || assignments != 1 || overrides != 1 || evaluations != 1 {
			release()
			t.Fatalf("scope %s sees profiles=%d packs=%d assignments=%d overrides=%d evaluations=%d", scope.tenant, profiles, packs, assignments, overrides, evaluations)
		}

		otherTenant, otherInstitution := "tenant-balotesti", "inst-balotesti"
		if scope.tenant == otherTenant {
			otherTenant, otherInstitution = "tenant-egueducation", "inst-001"
		}
		var leaked int
		if err := sessionPool.QueryRow(requestContext, `select count(*) from school_institution_profiles where tenant_code=$1 or institution_id=$2`, otherTenant, otherInstitution).Scan(&leaked); err != nil {
			release()
			t.Fatal(err)
		}
		if leaked != 0 {
			release()
			t.Fatalf("cross-scope profile leak from %s", scope.tenant)
		}
		for _, table := range []string{"school_policy_pack_versions", "school_policy_assignments", "school_policy_overrides", "school_policy_evaluations"} {
			if err := sessionPool.QueryRow(requestContext, `select count(*) from `+table+` where tenant_code=$1 or institution_id=$2`, otherTenant, otherInstitution).Scan(&leaked); err != nil {
				release()
				t.Fatal(err)
			}
			if leaked != 0 {
				release()
				t.Fatalf("cross-scope %s leak from %s", table, scope.tenant)
			}
		}
		if err := sessionPool.QueryRow(requestContext, `select count(*) from education_publications where tenant_code=$1 or institution_id=$2`, otherTenant, otherInstitution).Scan(&leaked); err != nil {
			release()
			t.Fatal(err)
		}
		if leaked != 0 {
			release()
			t.Fatalf("cross-scope education_publications leak from %s", scope.tenant)
		}
		_, err = sessionPool.Exec(requestContext, `insert into school_institution_profiles(id,tenant_code,institution_id,version,status,created_by_subject,updated_by_subject) values($1,$2,$3,99,'unclassified','forged','forged')`, uuid.New(), otherTenant, otherInstitution)
		if !isInsufficientPrivilege(err) {
			release()
			t.Fatalf("cross-scope insert was not denied: %v", err)
		}

		if _, err := sessionPool.Exec(requestContext, `update school_institution_profiles set effective_to=current_date where id=$1`, profileID); err != nil {
			release()
			t.Fatalf("close current policy period: %v", err)
		}
		if _, err := sessionPool.Exec(requestContext, `insert into school_institution_profiles(tenant_code,institution_id,version,status,school_legal_form,regulatory_profile,effective_from,source_reference,approved_by_subject,approved_at,created_by_subject,updated_by_subject) values($1,$2,3,'active','private','ro.private.preuniversity',current_date+1,'future-policy','rls-test',now(),'rls-test','rls-test')`, scope.tenant, scope.institution); err != nil {
			release()
			t.Fatalf("create later policy revision: %v", err)
		}
		var retainedEvaluationID string
		if err := sessionPool.QueryRow(requestContext, `select policy_evaluation_id::text from education_publications where publication_code=$1`, publicationCode).Scan(&retainedEvaluationID); err != nil {
			release()
			t.Fatalf("read publication policy snapshot: %v", err)
		}
		if retainedEvaluationID != evaluationID {
			release()
			t.Fatalf("later policy revision rewrote publication evaluation: got %s want %s", retainedEvaluationID, evaluationID)
		}
		release()
	}
}

func isInsufficientPrivilege(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "42501"
}

func isPostgresCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}
