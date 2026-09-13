package admission

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/eguilde/egueducation/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// This regression executes the actual 0152 preparation/hold rows against a
// migrated PostgreSQL database. It specifically proves expiry can consume the
// UPDATE ... RETURNING rows and then issue the allocation UPDATE on the same
// transaction without pgx's "conn busy" failure.
func TestExpireAdmissionPreparationsTxReleasesExpiredHoldPostgres(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("EGUEDUCATION_TEST_DATABASE_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	}
	if dsn == "" {
		t.Skip("real PostgreSQL expiry regression requires TEST_DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	pool := newAdmissionExpiryDatabase(t, ctx, dsn)
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate legal preparation expiry database: %v", err)
	}
	sessions := db.NewSessionPool(pool)
	requestCtx, release, err := db.AcquireRequestConn(ctx, pool, db.SessionConfig{
		TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: "expiry-regression-actor", IsSuperAdmin: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	applicationID, allocationID, policyID := seedAdmissionExpiryGraph(t, requestCtx, sessions)
	seedExpiryRetentionAuthority(t, ctx, pool, sessions)
	preparationID := uuid.NewString()
	if _, err = sessions.Exec(requestCtx, `insert into school_admission_legal_preparations(id,tenant_code,institution_id,artifact_kind,artifact_id,application_id,capacity_allocation_id,policy_evaluation_v2_id,aggregate_expected_version,canonical_payload,canonical_payload_bytes,canonical_payload_sha256,preparation_snapshot,prepared_by_subject,expires_at,retention_policy_id,retention_rule_version_id,retention_source_id,retention_anchor_at,minimum_retention_days,required_retention_until) select $1::uuid,'tenant-egueducation','inst-001','admission_decision',$2::uuid,$3::uuid,$4::uuid,$5::uuid,1,'{}','{}',encode(digest('{}','sha256'),'hex'),'{}','expiry-regression-actor',now()+interval '10 milliseconds',p.id,p.rule_version_id,p.source_id,now()+interval '10 milliseconds',p.minimum_retention_days,now()+interval '10 milliseconds'+make_interval(days=>p.minimum_retention_days) from school_admission_dss_retention_policies p where p.tenant_code='tenant-egueducation' and p.institution_id='inst-001' and p.status='active'`, preparationID, uuid.NewString(), applicationID, allocationID, policyID); err != nil {
		t.Fatalf("seed prepared legal record: %v", err)
	}
	time.Sleep(25 * time.Millisecond)
	tx, err := sessions.Begin(requestCtx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(requestCtx) //nolint:errcheck
	if err = expireAdmissionPreparationsTx(requestCtx, tx, scope{tenant: "tenant-egueducation", institution: "inst-001", actor: "expiry-regression-actor"}); err != nil {
		t.Fatalf("expire preparation with held allocation: %v", err)
	}
	if err = tx.Commit(requestCtx); err != nil {
		t.Fatalf("commit expiry release: %v", err)
	}
	var preparationStatus, allocationStatus string
	if err = sessions.QueryRow(requestCtx, `select status from school_admission_legal_preparations where id=$1::uuid`, preparationID).Scan(&preparationStatus); err != nil {
		t.Fatal(err)
	}
	if err = sessions.QueryRow(requestCtx, `select status from school_admission_capacity_allocations where id=$1::uuid`, allocationID).Scan(&allocationStatus); err != nil {
		t.Fatal(err)
	}
	if preparationStatus != "expired" || allocationStatus != "released" {
		t.Fatalf("expiry statuses preparation=%q allocation=%q, want expired/released", preparationStatus, allocationStatus)
	}
}

func seedExpiryRetentionAuthority(t *testing.T, ctx context.Context, pool *pgxpool.Pool, sessions *db.SessionPool) {
	t.Helper()
	sourceCtx, releaseSource, err := db.AcquireRequestConn(ctx, pool, db.SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: "expiry-retention-source", IsSuperAdmin: true})
	if err != nil {
		t.Fatal(err)
	}
	var sourceID string
	if err = sessions.QueryRow(sourceCtx, `insert into school_regulatory_sources(tenant_code,institution_id,source_kind,citation,source_url,checksum_sha256,effective_from,verified_at,verified_by_subject,revalidation_owner_subject,created_by_subject,updated_by_subject) values('tenant-egueducation','inst-001','law','synthetic retention test','https://example.test/retention',repeat('a',64),current_date,now(),'expiry-retention-source','expiry-retention-source','expiry-retention-source','expiry-retention-source') returning id::text`).Scan(&sourceID); err != nil {
		releaseSource()
		t.Fatal(err)
	}
	releaseSource()
	proposer := "admission-retention-proposer-" + uuid.NewString()
	approver := "admission-retention-approver-" + uuid.NewString()
	for _, actor := range []string{proposer, approver} {
		var userID string
		// This DB-only fixture authenticates by subject. Do not fabricate a
		// phone identity: profile phones require the verified identity workflow.
		if err := pool.QueryRow(ctx, `insert into app_users(sub,name,email,locale,status) values($1,$2,$3,'ro','active') returning id::text`, actor, actor, actor+"@example.test").Scan(&userID); err != nil {
			t.Fatalf("seed retention director %s: %v", actor, err)
		}
		tag, err := pool.Exec(ctx, `insert into app_memberships(user_id,tenant_code,position_code,org_unit_code,organization_name,is_primary,active,start_date)
			select $1::uuid,'tenant-egueducation','director',membership.org_unit_code,membership.organization_name,false,true,current_date
			from app_memberships membership join app_users user_record on user_record.id=membership.user_id
			where user_record.sub='usr-002' and membership.tenant_code='tenant-egueducation' and membership.position_code='director' and membership.active
			order by membership.is_primary desc limit 1`, userID)
		if err != nil || tag.RowsAffected() != 1 {
			t.Fatalf("seed effective retention director membership %s: %v", actor, err)
		}
	}

	proposerCtx, releaseProposer, err := db.AcquireRequestConn(ctx, pool, db.SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: proposer})
	if err != nil {
		t.Fatalf("bind retention proposer: %v", err)
	}
	var ruleID string
	if err = sessions.QueryRow(proposerCtx, `insert into school_admission_retention_rule_versions(tenant_code,institution_id,artifact_kind,status,minimum_retention_days,effective_from,source_id,proposed_by_subject) values('tenant-egueducation','inst-001','admission_dss','proposed',365,current_date,$1::uuid,$2) returning id::text`, sourceID, proposer).Scan(&ruleID); err != nil {
		releaseProposer()
		t.Fatalf("propose admission retention authority: %v", err)
	}
	if _, err = sessions.Exec(proposerCtx, `update school_admission_retention_rule_versions set status='active' where id=$1::uuid`, ruleID); err == nil || !strings.Contains(err.Error(), "distinct approving actor") {
		releaseProposer()
		t.Fatalf("self-approval error=%v, want distinct-actor rejection", err)
	}
	releaseProposer()

	approverCtx, releaseApprover, err := db.AcquireRequestConn(ctx, pool, db.SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: approver})
	if err != nil {
		t.Fatalf("bind retention approver: %v", err)
	}
	if _, err = sessions.Exec(approverCtx, `update school_admission_retention_rule_versions set status='active' where id=$1::uuid`, ruleID); err != nil {
		releaseApprover()
		t.Fatalf("approve admission retention authority: %v", err)
	}
	releaseApprover()

	proposerCtx, releaseProposer, err = db.AcquireRequestConn(ctx, pool, db.SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: proposer})
	if err != nil {
		t.Fatalf("rebind retention proposer: %v", err)
	}
	defer releaseProposer()
	var configuredDays int
	if err = sessions.QueryRow(proposerCtx, `insert into school_admission_dss_retention_policies(tenant_code,institution_id,status,minimum_retention_days,source_id,rule_version_id,effective_from,created_by_subject) values('tenant-egueducation','inst-001','active',1,$1::uuid,$2::uuid,current_date,$3) returning minimum_retention_days`, sourceID, ruleID, proposer).Scan(&configuredDays); err != nil {
		t.Fatalf("activate approved retention authority: %v", err)
	}
	if configuredDays != 365 {
		t.Fatalf("configured retention days=%d, want server-derived legal minimum 365", configuredDays)
	}
}

func newAdmissionExpiryDatabase(t *testing.T, ctx context.Context, dsn string) *pgxpool.Pool {
	t.Helper()
	base, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	admin, err := pgx.ConnectConfig(ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	name := "admission_expiry_" + strings.ReplaceAll(uuid.NewString()[:12], "-", "")
	if _, err = admin.Exec(ctx, `create database `+pgx.Identifier{name}.Sanitize()+` template template0`); err != nil {
		admin.Close(ctx)
		t.Fatal(err)
	}
	admin.Close(ctx)
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.Database = name
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		cleanup, cleanupErr := pgx.ConnectConfig(context.Background(), base)
		if cleanupErr == nil {
			_, _ = cleanup.Exec(context.Background(), `drop database if exists `+pgx.Identifier{name}.Sanitize()+` with (force)`)
			cleanup.Close(context.Background())
		}
	})
	return pool
}

func seedAdmissionExpiryGraph(t *testing.T, ctx context.Context, pool *db.SessionPool) (applicationID, allocationID, policyID string) {
	t.Helper()
	var sourceID, offeringID, locationID, authorizationID, classID, contextID, campaignID, partyID string
	mustRow := func(query string, args ...any) string {
		var id string
		if err := pool.QueryRow(ctx, query, args...).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	sourceID = mustRow(`insert into school_regulatory_sources(tenant_code,institution_id,source_kind,citation,source_url,checksum_sha256,effective_from,verified_at,verified_by_subject,revalidation_owner_subject,created_by_subject,updated_by_subject) values('tenant-egueducation','inst-001','law','expiry test','https://example.test/',repeat('a',64),current_date,now(),'expiry-regression-actor','expiry-regression-actor','expiry-regression-actor','expiry-regression-actor') returning id::text`)
	offeringID = mustRow(`insert into school_education_offerings(tenant_code,institution_id,code,education_level,title,effective_from,created_by_subject,updated_by_subject) values('tenant-egueducation','inst-001',$1,'primary','Expiry offering',current_date,'expiry-regression-actor','expiry-regression-actor') returning id::text`, "OFF-"+uuid.NewString())
	locationID = mustRow(`insert into school_locations(tenant_code,institution_id,code,name,effective_from,created_by_subject,updated_by_subject) values('tenant-egueducation','inst-001',$1,'Expiry location',current_date,'expiry-regression-actor','expiry-regression-actor') returning id::text`, "LOC-"+uuid.NewString())
	authorizationID = mustRow(`insert into school_offering_authorizations(tenant_code,institution_id,offering_id,location_id,status,authority_name,decision_reference,capacity,effective_from,source_id,created_by_subject,updated_by_subject) values('tenant-egueducation','inst-001',$1::uuid,$2::uuid,'provisional','Authority',$3,2,current_date,$4::uuid,'expiry-regression-actor','expiry-regression-actor') returning id::text`, offeringID, locationID, "AUTH-"+uuid.NewString(), sourceID)
	classID = mustRow(`insert into education_school_classes(tenant_code,institution_id,class_code,class_name,school_year,grade_level) values('tenant-egueducation','inst-001',$1,'Expiry class','2035-2036','I') returning id::text`, "CLASS-"+uuid.NewString())
	contextID = mustRow(`insert into school_admission_class_offering_contexts(tenant_code,institution_id,class_id,offering_id,location_id,authorization_id,school_year,shift,effective_from,created_by_subject,updated_by_subject) values('tenant-egueducation','inst-001',$1::uuid,$2::uuid,$3::uuid,$4::uuid,'2035-2036','day',current_date,'expiry-regression-actor','expiry-regression-actor') returning id::text`, classID, offeringID, locationID, authorizationID)
	campaignID = mustRow(`insert into school_admission_campaigns(tenant_code,institution_id,source_id,code,title,school_year,offering_id,location_id,authorization_id,class_offering_context_id,capacity_limit,capacity_unit,student_place_limit,capacity_basis,shift,opens_on,closes_on,created_by_subject,updated_by_subject) values('tenant-egueducation','inst-001',$1::uuid,$2,'Expiry campaign','2035-2036',$3::uuid,$4::uuid,$5::uuid,$6::uuid,2,'students',2,'{}','day',current_date,current_date,'expiry-regression-actor','expiry-regression-actor') returning id::text`, sourceID, "CMP-"+uuid.NewString(), offeringID, locationID, authorizationID, contextID)
	partyID = mustRow(`insert into app_parties(tenant_code,institution_id,code,party_type,display_name) values('tenant-egueducation','inst-001',$1,'physical','Expiry applicant') returning id::text`, "PARTY-"+uuid.NewString())
	applicationID = mustRow(`insert into school_admission_applications(tenant_code,institution_id,campaign_id,application_no,candidate_party_id,created_by_subject,updated_by_subject) values('tenant-egueducation','inst-001',$1::uuid,$2,$3::uuid,'expiry-regression-actor','expiry-regression-actor') returning id::text`, campaignID, "APP-"+uuid.NewString(), partyID)
	allocationID = mustRow(`insert into school_admission_capacity_allocations(tenant_code,institution_id,application_id,campaign_id,class_offering_context_id,authorization_id,capacity_unit,shift,created_by_subject,updated_by_subject) values('tenant-egueducation','inst-001',$1::uuid,$2::uuid,$3::uuid,$4::uuid,'students','day','expiry-regression-actor','expiry-regression-actor') returning id::text`, applicationID, campaignID, contextID, authorizationID)

	profileID := uuid.NewString()
	if _, err := pool.Exec(ctx, `insert into school_institution_profiles_v2(id,tenant_code,institution_id,version,profile_series_id,status,legal_form,effective_from,created_by_subject,updated_by_subject) values($1::uuid,'tenant-egueducation','inst-001',1,$1::uuid,'draft','public',current_date,'expiry-regression-actor','expiry-regression-actor')`, profileID); err != nil {
		t.Fatal(err)
	}
	inputID := mustRow(`insert into school_operation_policy_inputs(tenant_code,institution_id,operation_code,profile_id,profile_version,profile_series_id,context,checksum_sha256,created_by_subject) values('tenant-egueducation','inst-001','admission.expiry',$1::uuid,1,$1::uuid,'{}',repeat('b',64),'expiry-regression-actor') returning id::text`, profileID)
	policyID = mustRow(`insert into school_operation_policy_evaluations_v2(tenant_code,institution_id,input_id,allowed,capabilities,obligations,checksum_sha256,evaluated_by_subject) values('tenant-egueducation','inst-001',$1::uuid,true,'[]','[]',repeat('c',64),'expiry-regression-actor') returning id::text`, inputID)
	return applicationID, allocationID, policyID
}
