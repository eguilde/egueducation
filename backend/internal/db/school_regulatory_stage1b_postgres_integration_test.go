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

// TestSchoolRegulatoryStage1BPostgresIntegration is intentionally a real
// NOBYPASSRLS test. CI provides a disposable DSN; local unit runs skip it.
func TestSchoolRegulatoryStage1BPostgresIntegration(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("EGUEDUCATION_TEST_DATABASE_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	}
	if dsn == "" {
		t.Skip("Stage 1B PostgreSQL integration requires TEST_DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	it := newTenantGrantRLSIntegrationDatabase(t, ctx, dsn)
	admin, err := pgxpool.NewWithConfig(ctx, it.databaseConfig.Copy())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	if err = Migrate(ctx, admin); err != nil {
		t.Fatalf("migrate Stage 1B: %v", err)
	}
	role := tenantGrantQuoteIdentifier(it.roleName)
	for _, q := range []string{"grant usage on schema public to " + role, "grant select,insert,update,delete on school_regulatory_sources,school_institution_profiles_v2,school_locations,school_education_offerings,school_offering_authorizations,school_funding_instruments,school_funding_eligibility_evaluations,school_procurement_applicability_assessments,school_operation_policy_inputs,school_operation_policy_evaluations_v2 to " + role} {
		if _, err = admin.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	cfg := it.databaseConfig.Copy()
	cfg.ConnConfig.User = it.roleName
	cfg.ConnConfig.Password = it.rolePassword
	cfg.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	session := NewSessionPool(pool)
	var firstPolicyID string
	for i, scope := range []struct{ tenant, institution, form string }{{"tenant-egueducation", "inst-001", "public"}, {"tenant-balotesti", "inst-balotesti", "private"}} {
		rctx, release, err := AcquireRequestConn(ctx, pool, SessionConfig{TenantID: scope.tenant, InstitutionID: scope.institution, ActorSubject: "stage1b-integration"})
		if err != nil {
			t.Fatal(err)
		}
		var source, profile, offering, location, authorization string
		if err = session.QueryRow(rctx, `insert into school_regulatory_sources(tenant_code,institution_id,source_kind,citation,source_url,checksum_sha256,verified_at,verified_by_subject,revalidation_owner_subject,created_by_subject,updated_by_subject) values($1,$2,'law',$3,'https://legislatie.just.ro/',repeat('c',64),now(),'stage1b','stage1b','stage1b','stage1b') returning id::text`, scope.tenant, scope.institution, "stage1b-source-"+scope.tenant).Scan(&source); err != nil {
			release()
			t.Fatal(err)
		}
		profile = uuid.NewString()
		if _, err = session.Exec(rctx, `insert into school_institution_profiles_v2(id,tenant_code,institution_id,version,status,legal_form,effective_from,profile_series_id,approved_by_subject,approved_at,created_by_subject,updated_by_subject) values($1,$2,$3,1,'active',$4,current_date,$1,'stage1b',now(),'stage1b','stage1b')`, profile, scope.tenant, scope.institution, scope.form); err != nil {
			release()
			t.Fatal(err)
		}
		if err = session.QueryRow(rctx, `insert into school_education_offerings(tenant_code,institution_id,code,education_level,title,effective_from,created_by_subject,updated_by_subject) values($1,$2,'gimnazial','gimnazial','Stage 1B',current_date,'stage1b','stage1b') returning id::text`, scope.tenant, scope.institution).Scan(&offering); err != nil {
			release()
			t.Fatal(err)
		}
		if err = session.QueryRow(rctx, `insert into school_locations(tenant_code,institution_id,code,name,effective_from,created_by_subject,updated_by_subject) values($1,$2,'sediu','Sediu',current_date,'stage1b','stage1b') returning id::text`, scope.tenant, scope.institution).Scan(&location); err != nil {
			release()
			t.Fatal(err)
		}
		if err = session.QueryRow(rctx, `insert into school_offering_authorizations(tenant_code,institution_id,offering_id,location_id,status,authority_name,decision_reference,effective_from,source_id,created_by_subject,updated_by_subject) values($1,$2,$3,$4,'accredited','ARACIP','stage1b',current_date,$5,'stage1b','stage1b') returning id::text`, scope.tenant, scope.institution, offering, location, source).Scan(&authorization); err != nil {
			release()
			t.Fatal(err)
		}
		var count int
		if err = session.QueryRow(rctx, `select count(*) from school_offering_authorizations where id=$1`, authorization).Scan(&count); err != nil || count != 1 {
			release()
			t.Fatalf("authorized offer missing: %v %d", err, count)
		}
		if _, err = session.Exec(rctx, `update school_offering_authorizations set effective_to=current_date+29,expected_version=expected_version+1,updated_by_subject='stage1b' where id=$1`, authorization); err != nil {
			release()
			t.Fatalf("close predecessor authorization: %v", err)
		}
		var replacement string
		if err = session.QueryRow(rctx, `insert into school_offering_authorizations(tenant_code,institution_id,offering_id,location_id,status,authority_name,decision_reference,effective_from,source_id,replaces_authorization_id,created_by_subject,updated_by_subject) values($1,$2,$3,$4,'suspended','ARACIP','stage1b-suspension',current_date+30,$5,$6,'stage1b','stage1b') returning id::text`, scope.tenant, scope.institution, offering, location, source, authorization).Scan(&replacement); err != nil {
			release()
			t.Fatalf("append prospective authorization replacement: %v", err)
		}
		if _, err = session.Exec(rctx, `insert into school_offering_authorizations(tenant_code,institution_id,offering_id,location_id,status,authority_name,decision_reference,effective_from,source_id,created_by_subject,updated_by_subject) values($1,$2,$3,$4,'withdrawn','ARACIP','overlap',current_date+30,$5,'stage1b','stage1b')`, scope.tenant, scope.institution, offering, location, source); err == nil {
			release()
			t.Fatal("overlapping authorization interval was accepted")
		}
		if err = session.QueryRow(rctx, `select count(*) from school_offering_authorizations where id=$1 and replaces_authorization_id=$2`, replacement, authorization).Scan(&count); err != nil || count != 1 {
			release()
			t.Fatalf("authorization lineage missing: %v %d", err, count)
		}
		for _, guard := range []struct {
			name       string
			statement  string
			id         string
			constraint string
		}{
			{name: "location inactive without end", statement: `update school_locations set active=false where id=$1`, id: location, constraint: "school_location_inactive_requires_effective_to"},
			{name: "offering inactive without end", statement: `update school_education_offerings set active=false where id=$1`, id: offering, constraint: "education_offering_inactive_requires_effective_to"},
			{name: "location shortened across authorization", statement: `update school_locations set effective_to=current_date where id=$1`, id: location, constraint: "school_location_authorization_dependency_conflict"},
			{name: "offering shortened across authorization", statement: `update school_education_offerings set effective_to=current_date where id=$1`, id: offering, constraint: "education_offering_authorization_dependency_conflict"},
		} {
			_, guardErr := session.Exec(rctx, guard.statement, guard.id)
			var pgErr *pgconn.PgError
			if !errors.As(guardErr, &pgErr) || pgErr.Code != "23514" || pgErr.ConstraintName != guard.constraint {
				release()
				t.Fatalf("%s: got %v, want check constraint %s", guard.name, guardErr, guard.constraint)
			}
		}
		var locationActive, offeringActive bool
		var locationTo, offeringTo *time.Time
		if err = session.QueryRow(rctx, `select location.active,location.effective_to,offering.active,offering.effective_to from school_locations location cross join school_education_offerings offering where location.id=$1 and offering.id=$2`, location, offering).Scan(&locationActive, &locationTo, &offeringActive, &offeringTo); err != nil || !locationActive || !offeringActive || locationTo != nil || offeringTo != nil {
			release()
			t.Fatalf("rejected parent updates changed catalog state: %v active=%v/%v end=%v/%v", err, locationActive, offeringActive, locationTo, offeringTo)
		}
		if err = session.QueryRow(rctx, `insert into school_operation_policy_inputs(tenant_code,institution_id,operation_code,profile_id,profile_version,offering_id,location_id,context,checksum_sha256,created_by_subject) values($1,$2,'enrollment.create',$3,1,$4,$5,'{}',repeat('a',64),'stage1b') returning id::text`, scope.tenant, scope.institution, profile, offering, location).Scan(&firstPolicyID); err != nil {
			release()
			t.Fatal(err)
		}
		if _, err = session.Exec(rctx, `insert into school_operation_policy_evaluations_v2(tenant_code,institution_id,input_id,allowed,capabilities,obligations,checksum_sha256,evaluated_by_subject) values($1,$2,$3,true,'[]','[]',repeat('b',64),'stage1b')`, scope.tenant, scope.institution, firstPolicyID); err != nil {
			release()
			t.Fatal(err)
		}
		if _, err = session.Exec(rctx, `update school_operation_policy_inputs set operation_code='forged' where id=$1`, firstPolicyID); err == nil {
			release()
			t.Fatal("immutable policy input updated")
		}
		otherTenant, otherInst := "tenant-balotesti", "inst-balotesti"
		if i == 1 {
			otherTenant, otherInst = "tenant-egueducation", "inst-001"
		}
		if _, err = session.Exec(rctx, `insert into school_institution_profiles_v2(id,tenant_code,institution_id,version,status,legal_form,effective_from,created_by_subject,updated_by_subject) values($1,$2,$3,99,'draft','private',current_date,'forged','forged')`, uuid.New(), otherTenant, otherInst); !isInsufficientPrivilege(err) {
			release()
			t.Fatalf("cross-tenant profile write allowed: %v", err)
		}
		release()
	}
}
