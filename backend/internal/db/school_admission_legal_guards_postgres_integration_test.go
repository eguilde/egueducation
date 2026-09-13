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

func TestSchoolAdmissionLegalGuardsPostgresIntegration(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("EGUEDUCATION_TEST_DATABASE_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	}
	if dsn == "" {
		t.Skip("0152 legal guard PostgreSQL integration requires TEST_DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	it := newTenantGrantRLSIntegrationDatabase(t, ctx, dsn)
	pool, err := pgxpool.NewWithConfig(ctx, it.databaseConfig.Copy())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err = Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate 0152 guards: %v", err)
	}
	sessions := NewSessionPool(pool)
	adminCtx, release, err := AcquireRequestConn(ctx, pool, SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: "0152-legal-admin", IsSuperAdmin: true})
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	fixture := seedAdmissionIntegrationFixture(t, adminCtx, sessions)
	testAdmissionRetentionAuthority(t, ctx, pool, sessions, fixture.sourceID)
	if _, err = sessions.Exec(adminCtx, `update school_offering_authorizations set status='provisional' where tenant_code='tenant-egueducation' and institution_id='inst-001' and id=$1::uuid`, fixture.authorizationID); err != nil {
		t.Fatal(err)
	}
	campaignID := uuid.NewString()
	if _, err = sessions.Exec(adminCtx, `insert into school_admission_campaigns(id,tenant_code,institution_id,source_id,code,title,school_year,offering_id,location_id,authorization_id,class_offering_context_id,capacity_limit,capacity_unit,student_place_limit,capacity_basis,shift,opens_on,closes_on,created_by_subject,updated_by_subject) values($1::uuid,'tenant-egueducation','inst-001',$2::uuid,$3,'0152 campaign','2040-2041',$4::uuid,$5::uuid,$6::uuid,$7::uuid,2,'students',2,'{}','day',current_date,current_date,'0152-legal-admin','0152-legal-admin')`, campaignID, fixture.sourceID, "LGL-"+uuid.NewString(), fixture.offeringID, fixture.locationID, fixture.authorizationID, fixture.contextID); err != nil {
		t.Fatalf("seed 0152 campaign: %v", err)
	}
	applicationID := seedAdmissionApplication(t, adminCtx, sessions, campaignID, "0152")
	policyID := seed0152PolicyEvaluation(t, adminCtx, sessions)

	// The guard must reject the bytes/hash mismatch before an application can be
	// turned into an unsigned legal preparation.
	_, err = sessions.Exec(adminCtx, `insert into school_admission_legal_preparations(id,tenant_code,institution_id,artifact_kind,artifact_id,application_id,policy_evaluation_v2_id,aggregate_expected_version,canonical_payload,canonical_payload_bytes,canonical_payload_sha256,preparation_snapshot,prepared_by_subject,expires_at,retention_policy_id,retention_rule_version_id,retention_source_id,retention_anchor_at,minimum_retention_days,required_retention_until) select $1::uuid,'tenant-egueducation','inst-001','admission_decision',$2::uuid,$3::uuid,$4::uuid,1,'{}','{"forged":true}',repeat('0',64),'{}','0152-legal-admin',now()+interval '5 minutes',p.id,p.rule_version_id,p.source_id,now()+interval '5 minutes',p.minimum_retention_days,now()+interval '5 minutes'+make_interval(days=>p.minimum_retention_days) from school_admission_dss_retention_policies p where p.tenant_code='tenant-egueducation' and p.institution_id='inst-001' and p.status='active'`, uuid.NewString(), uuid.NewString(), applicationID, policyID)
	assert0152Check(t, err, "canonical legal preparation payload bytes and hash must match")

	preparationID := uuid.NewString()
	if _, err = sessions.Exec(adminCtx, `insert into school_admission_legal_preparations(id,tenant_code,institution_id,artifact_kind,artifact_id,application_id,policy_evaluation_v2_id,aggregate_expected_version,canonical_payload,canonical_payload_bytes,canonical_payload_sha256,preparation_snapshot,prepared_by_subject,expires_at,retention_policy_id,retention_rule_version_id,retention_source_id,retention_anchor_at,minimum_retention_days,required_retention_until) select $1::uuid,'tenant-egueducation','inst-001','admission_decision',$2::uuid,$3::uuid,$4::uuid,1,'{}','{}',encode(digest('{}','sha256'),'hex'),'{}','0152-legal-admin',now()+interval '5 minutes',p.id,p.rule_version_id,p.source_id,now()+interval '5 minutes',p.minimum_retention_days,now()+interval '5 minutes'+make_interval(days=>p.minimum_retention_days) from school_admission_dss_retention_policies p where p.tenant_code='tenant-egueducation' and p.institution_id='inst-001' and p.status='active'`, preparationID, uuid.NewString(), applicationID, policyID); err != nil {
		t.Fatalf("seed valid legal preparation: %v", err)
	}
	if _, err = sessions.Exec(adminCtx, `update school_admission_legal_preparations set status='cancelled',cancellation_reason='operator_cancelled' where id=$1::uuid`, preparationID); err != nil {
		t.Fatalf("cancel legal preparation: %v", err)
	}
	_, err = sessions.Exec(adminCtx, `update school_admission_legal_preparations set cancellation_reason='rewritten' where id=$1::uuid`, preparationID)
	if err == nil || !strings.Contains(err.Error(), "lifecycle records are immutable") {
		t.Fatalf("terminal preparation provenance rewrite err=%v", err)
	}

	proposer, approver := seed0152Directors(t, ctx, pool)
	proposerCtx, releaseProposer, err := AcquireRequestConn(ctx, pool, SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: proposer})
	if err != nil {
		t.Fatal(err)
	}
	var userID string
	if err = pool.QueryRow(ctx, `select id::text from app_users where sub=$1`, proposer).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	var authorizationID string
	if err = sessions.QueryRow(proposerCtx, `insert into school_admission_signer_authorizations(tenant_code,institution_id,certificate_sha256,user_id,actor_subject,permission_code,valid_from,valid_until,proposed_by_subject) values('tenant-egueducation','inst-001',repeat('e',64),$1::uuid,$2,'education.admissions.decide',now()-interval '2 days',now()-interval '1 day',$2) returning id::text`, userID, proposer).Scan(&authorizationID); err != nil {
		t.Fatalf("seed expired signer proposal: %v", err)
	}
	releaseProposer()
	approverCtx, releaseApprover, err := AcquireRequestConn(ctx, pool, SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: approver})
	if err != nil {
		t.Fatal(err)
	}
	defer releaseApprover()
	_, err = sessions.Exec(approverCtx, `update school_admission_signer_authorizations set status='active' where id=$1::uuid`, authorizationID)
	if err == nil || !strings.Contains(err.Error(), "expired signer authorization cannot be approved") {
		t.Fatalf("expired signer approval err=%v", err)
	}
}

func seed0152PolicyEvaluation(t *testing.T, ctx context.Context, sessions *SessionPool) string {
	t.Helper()
	var profileID, profileSeriesID, inputID, evaluationID string
	var profileVersion int
	if err := sessions.QueryRow(ctx, `select id::text,version,profile_series_id::text from school_institution_profiles_v2 where tenant_code='tenant-egueducation' and institution_id='inst-001' order by version desc limit 1`).Scan(&profileID, &profileVersion, &profileSeriesID); err != nil {
		profileID = uuid.NewString()
		profileSeriesID = profileID
		if _, insertErr := sessions.Exec(ctx, `insert into school_institution_profiles_v2(id,tenant_code,institution_id,version,profile_series_id,status,legal_form,effective_from,created_by_subject,updated_by_subject) values($1::uuid,'tenant-egueducation','inst-001',1,$2::uuid,'draft','public',current_date,'0152-legal-admin','0152-legal-admin')`, profileID, profileSeriesID); insertErr != nil {
			t.Fatalf("seed 0152 profile: %v", insertErr)
		}
		if err = sessions.QueryRow(ctx, `select id::text,version,profile_series_id::text from school_institution_profiles_v2 where tenant_code='tenant-egueducation' and institution_id='inst-001' order by version desc limit 1`).Scan(&profileID, &profileVersion, &profileSeriesID); err != nil {
			t.Fatalf("find seeded 0152 profile: %v", err)
		}
	}
	inputID, evaluationID = uuid.NewString(), uuid.NewString()
	if _, err := sessions.Exec(ctx, `insert into school_operation_policy_inputs(id,tenant_code,institution_id,operation_code,profile_id,profile_version,profile_series_id,context,checksum_sha256,created_by_subject) values($1::uuid,'tenant-egueducation','inst-001','0152.guard',$2::uuid,$3,$4::uuid,'{}',repeat('a',64),'0152-legal-admin')`, inputID, profileID, profileVersion, profileSeriesID); err != nil {
		t.Fatalf("seed 0152 policy input: %v", err)
	}
	if _, err := sessions.Exec(ctx, `insert into school_operation_policy_evaluations_v2(id,tenant_code,institution_id,input_id,allowed,capabilities,obligations,checksum_sha256,evaluated_by_subject) values($1::uuid,'tenant-egueducation','inst-001',$2::uuid,true,'[]','[]',repeat('b',64),'0152-legal-admin')`, evaluationID, inputID); err != nil {
		t.Fatalf("seed 0152 policy evaluation: %v", err)
	}
	return evaluationID
}

func seed0152Directors(t *testing.T, ctx context.Context, pool *pgxpool.Pool) (string, string) {
	t.Helper()
	actors := []string{"0152-proposer-" + uuid.NewString(), "0152-approver-" + uuid.NewString()}
	for _, actor := range actors {
		var userID string
		if err := pool.QueryRow(ctx, `insert into app_users(sub,name,email,locale,status) values($1,$1,$2,'ro','active') returning id::text`, actor, actor+"@example.test").Scan(&userID); err != nil {
			t.Fatal(err)
		}
		tag, err := pool.Exec(ctx, `insert into app_memberships(user_id,tenant_code,position_code,org_unit_code,organization_name,is_primary,active,start_date) select $1::uuid,'tenant-egueducation','director',m.org_unit_code,m.organization_name,false,true,current_date from app_memberships m join app_users u on u.id=m.user_id where u.sub='usr-002' and m.tenant_code='tenant-egueducation' and m.position_code='director' and m.active order by m.is_primary desc limit 1`, userID)
		if err != nil || tag.RowsAffected() != 1 {
			t.Fatalf("seed 0152 director %s err=%v", actor, err)
		}
	}
	return actors[0], actors[1]
}

func assert0152Check(t *testing.T, err error, contains string) {
	t.Helper()
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || (pgErr.Code != "23514" && pgErr.Code != "P0001") || !strings.Contains(pgErr.Message, contains) {
		t.Fatalf("guard err=%v, want PostgreSQL guard rejection containing %q", err, contains)
	}
}
