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

// TestSchoolAdmissionsPostgresIntegration exercises the database invariants
// that source-level tests cannot prove: a real migration, FORCE RLS, the
// statutory provisional/authorized boundary, per-application capacity, and
// WORM source-version snapshots.  CI supplies a disposable administrator DSN.
func TestSchoolAdmissionsPostgresIntegration(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("EGUEDUCATION_TEST_DATABASE_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	}
	if dsn == "" {
		t.Skip("0148 admission PostgreSQL integration requires EGUEDUCATION_TEST_DATABASE_URL or TEST_DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	it := newTenantGrantRLSIntegrationDatabase(t, ctx, dsn)
	admin, err := pgxpool.NewWithConfig(ctx, it.databaseConfig.Copy())
	if err != nil {
		t.Fatalf("open 0148 admin database: %v", err)
	}
	t.Cleanup(admin.Close)
	if err = Migrate(ctx, admin); err != nil {
		t.Fatalf("apply admission migrations: %v", err)
	}
	var admissionPermissionCount int
	if err = admin.QueryRow(ctx, `
		select count(*)
		from app_permissions
		where code in (
			'education.admissions.read',
			'education.admissions.manage',
			'education.admissions.decide',
			'education.admissions.appeals.manage',
			'education.admissions.exports.generate'
		) and btrim(label) <> ''
	`).Scan(&admissionPermissionCount); err != nil || admissionPermissionCount != 5 {
		t.Fatalf("admission permission labels=%d err=%v, want all five 0148 permissions", admissionPermissionCount, err)
	}

	role := tenantGrantQuoteIdentifier(it.roleName)
	for _, statement := range []string{
		"grant usage on schema public to " + role,
		"grant select on school_admission_campaigns to " + role,
	} {
		if _, err = admin.Exec(ctx, statement); err != nil {
			t.Fatalf("grant admission integration access: %v", err)
		}
	}

	adminPool := NewSessionPool(admin)
	adminCtx, releaseAdmin, err := AcquireRequestConn(ctx, admin, SessionConfig{
		TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: "0148-admission-admin", IsSuperAdmin: true,
	})
	if err != nil {
		t.Fatalf("bind admission admin context: %v", err)
	}
	defer releaseAdmin()
	fixture := seedAdmissionIntegrationFixture(t, adminCtx, adminPool)
	testAdmissionRetentionAuthority(t, ctx, admin, adminPool, fixture.sourceID)
	_, classYearErr := adminPool.Exec(adminCtx, `
		insert into school_admission_class_offering_contexts
			(tenant_code,institution_id,class_id,offering_id,location_id,authorization_id,school_year,shift,effective_from,created_by_subject,updated_by_subject)
		values ('tenant-egueducation','inst-001',$1,$2,$3,$4,'2036-2037','day',current_date,'0148-admission-admin','0148-admission-admin')
	`, fixture.classID, fixture.offeringID, fixture.locationID, fixture.authorizationID)
	assertAdmissionCheckViolation(t, classYearErr, "class offering context")

	if _, err = adminPool.Exec(adminCtx, `
		insert into school_admission_campaigns
			(tenant_code,institution_id,source_id,code,title,school_year,offering_id,location_id,authorization_id,class_offering_context_id,capacity_limit,capacity_unit,student_place_limit,capacity_basis,shift,opens_on,closes_on,created_by_subject,updated_by_subject)
		values ('tenant-egueducation','inst-001',$1,$2,'must fail','2035-2036',$3,$4,$5,$6,1,'students',1,'{}','day',current_date,current_date,'0148-admission-admin','0148-admission-admin')
	`, fixture.sourceID, "ADMISSION-AUTHORIZED-"+uuid.NewString(), fixture.offeringID, fixture.locationID, fixture.authorizationID, fixture.contextID); err == nil {
		t.Fatal("ambiguous authorized status was accepted for an admission campaign")
	}
	if _, err = adminPool.Exec(adminCtx, `update school_offering_authorizations set status='provisional' where id=$1`, fixture.authorizationID); err != nil {
		t.Fatalf("make fixture authorization provisional: %v", err)
	}
	if _, err = adminPool.Exec(adminCtx, `
		insert into school_admission_campaigns
			(tenant_code,institution_id,id,source_id,code,title,school_year,offering_id,location_id,authorization_id,class_offering_context_id,capacity_limit,capacity_unit,student_place_limit,capacity_basis,shift,opens_on,closes_on,created_by_subject,updated_by_subject)
		values ('tenant-egueducation','inst-001',$1,$2,$3,'provisional allowed','2035-2036',$4,$5,$6,$7,2,'students',2,'{}','day',current_date,current_date,'0148-admission-admin','0148-admission-admin')
	`, fixture.campaignID, fixture.sourceID, "ADMISSION-PROVISIONAL-"+uuid.NewString(), fixture.offeringID, fixture.locationID, fixture.authorizationID, fixture.contextID); err != nil {
		t.Fatalf("provisional authorization must permit campaign: %v", err)
	}

	applicationOne := seedAdmissionApplication(t, adminCtx, adminPool, fixture.campaignID, "ONE")
	if _, err = adminPool.Exec(adminCtx, `
		insert into school_admission_capacity_allocations
			(tenant_code,institution_id,application_id,campaign_id,class_offering_context_id,authorization_id,capacity_unit,shift,created_by_subject,updated_by_subject)
		values ('tenant-egueducation','inst-001',$1,$2,$3,$4,'students','day','0148-admission-admin','0148-admission-admin')
	`, applicationOne, fixture.campaignID, fixture.contextID, fixture.authorizationID); err != nil {
		t.Fatalf("hold one application place: %v", err)
	}
	applicationTwo := seedAdmissionApplication(t, adminCtx, adminPool, fixture.campaignID, "TWO")
	if _, err = adminPool.Exec(adminCtx, `
		insert into school_admission_capacity_allocations
			(tenant_code,institution_id,application_id,campaign_id,class_offering_context_id,authorization_id,capacity_unit,shift,created_by_subject,updated_by_subject)
		values ('tenant-egueducation','inst-001',$1,$2,$3,$4,'students','day','0148-admission-admin','0148-admission-admin')
	`, applicationTwo, fixture.campaignID, fixture.contextID, fixture.authorizationID); err != nil {
		t.Fatalf("hold second application place: %v", err)
	}
	applicationThree := seedAdmissionApplication(t, adminCtx, adminPool, fixture.campaignID, "THREE")
	_, capacityErr := adminPool.Exec(adminCtx, `
		insert into school_admission_capacity_allocations
			(tenant_code,institution_id,application_id,campaign_id,class_offering_context_id,authorization_id,capacity_unit,shift,created_by_subject,updated_by_subject)
		values ('tenant-egueducation','inst-001',$1,$2,$3,$4,'students','day','0148-admission-admin','0148-admission-admin')
	`, applicationThree, fixture.campaignID, fixture.contextID, fixture.authorizationID)
	assertAdmissionCheckViolation(t, capacityErr, "campaign student place limit")
	_, campaignLimitErr := adminPool.Exec(adminCtx, `
		update school_admission_campaigns
		set capacity_limit=1, student_place_limit=1
		where id=$1
	`, fixture.campaignID)
	assertAdmissionCheckViolation(t, campaignLimitErr, "cannot be reduced below active allocations")

	var requirementID string
	if err = adminPool.QueryRow(adminCtx, `
		insert into school_admission_document_requirements(tenant_code,institution_id,campaign_id,code,title,ordinal,created_by_subject,updated_by_subject)
		values ('tenant-egueducation','inst-001',$1,'identity','Identity document',1,'0148-admission-admin','0148-admission-admin') returning id::text
	`, fixture.campaignID).Scan(&requirementID); err != nil {
		t.Fatalf("create admission document requirement: %v", err)
	}
	_, wormErr := adminPool.Exec(adminCtx, `
		insert into school_admission_application_documents(tenant_code,institution_id,application_id,document_requirement_id,document_kind,status,created_by_subject,updated_by_subject)
		values ('tenant-egueducation','inst-001',$1,$2,'identity','submitted','0148-admission-admin','0148-admission-admin')
	`, applicationOne, requirementID)
	assertAdmissionCheckViolation(t, wormErr, "")
	archiveDocumentID, archiveVersionID := seedAdmissionWORMSnapshot(t, adminCtx, adminPool)
	if _, err = adminPool.Exec(adminCtx, `
		insert into school_admission_application_documents
			(tenant_code,institution_id,application_id,document_requirement_id,document_kind,status,archive_document_id,archive_version_id,archive_version_no,archive_source_bucket,archive_source_object_key,archive_source_object_version_id,archive_retention_until,archive_sha256,created_by_subject,updated_by_subject)
		select 'tenant-egueducation','inst-001',$1,$2,'identity','submitted',$3,$4,1,'source-bucket','source/key.pdf','source-version-0148',v.retention_until,repeat('d',64),'0148-admission-admin','0148-admission-admin'
		from archive_document_versions v where v.id=$4::uuid
	`, applicationOne, requirementID, archiveDocumentID, archiveVersionID); err != nil {
		t.Fatalf("insert WORM-snapshotted admission evidence: %v", err)
	}

	applicationConfig := it.databaseConfig.Copy()
	applicationConfig.ConnConfig.User = it.roleName
	applicationConfig.ConnConfig.Password = it.rolePassword
	applicationConfig.MaxConns = 1
	applicationPool, err := pgxpool.NewWithConfig(ctx, applicationConfig)
	if err != nil {
		t.Fatalf("open admission NOBYPASSRLS pool: %v", err)
	}
	t.Cleanup(applicationPool.Close)
	appSession := NewSessionPool(applicationPool)
	appCtx, releaseApp, err := AcquireRequestConn(ctx, applicationPool, SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: "0148-admission-reader"})
	if err != nil {
		t.Fatalf("bind admission RLS context: %v", err)
	}
	defer releaseApp()
	var ownCampaigns, foreignCampaigns int
	if err = appSession.QueryRow(appCtx, `select count(*) from school_admission_campaigns where id=$1`, fixture.campaignID).Scan(&ownCampaigns); err != nil || ownCampaigns != 1 {
		t.Fatalf("own admission campaign visibility=%d err=%v", ownCampaigns, err)
	}
	if err = appSession.QueryRow(appCtx, `select count(*) from school_admission_campaigns where tenant_code='tenant-balotesti'`).Scan(&foreignCampaigns); err != nil || foreignCampaigns != 0 {
		t.Fatalf("foreign admission campaign visible=%d err=%v", foreignCampaigns, err)
	}
}

func testAdmissionRetentionAuthority(t *testing.T, ctx context.Context, pool *pgxpool.Pool, sessions *SessionPool, sourceID string) {
	t.Helper()
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

	proposerCtx, releaseProposer, err := AcquireRequestConn(ctx, pool, SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: proposer})
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

	approverCtx, releaseApprover, err := AcquireRequestConn(ctx, pool, SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: approver})
	if err != nil {
		t.Fatalf("bind retention approver: %v", err)
	}
	if _, err = sessions.Exec(approverCtx, `update school_admission_retention_rule_versions set status='active' where id=$1::uuid`, ruleID); err != nil {
		releaseApprover()
		t.Fatalf("approve admission retention authority: %v", err)
	}
	releaseApprover()

	proposerCtx, releaseProposer, err = AcquireRequestConn(ctx, pool, SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: proposer})
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

type admissionIntegrationFixture struct {
	sourceID, classID, offeringID, locationID, authorizationID, contextID, campaignID string
}

func seedAdmissionIntegrationFixture(t *testing.T, ctx context.Context, pool *SessionPool) admissionIntegrationFixture {
	t.Helper()
	fixture := admissionIntegrationFixture{campaignID: uuid.NewString()}
	var sourceID, classID string
	if err := pool.QueryRow(ctx, `insert into school_regulatory_sources(tenant_code,institution_id,source_kind,citation,source_url,checksum_sha256,effective_from,verified_at,verified_by_subject,revalidation_owner_subject,created_by_subject,updated_by_subject) values ('tenant-egueducation','inst-001','law','0148 integration','https://legislatie.just.ro/',repeat('a',64),current_date,now(),'0148-admission-admin','0148-admission-admin','0148-admission-admin','0148-admission-admin') returning id::text`).Scan(&sourceID); err != nil {
		t.Fatalf("seed admission source: %v", err)
	}
	fixture.sourceID = sourceID
	if err := pool.QueryRow(ctx, `insert into school_education_offerings(tenant_code,institution_id,code,education_level,title,effective_from,created_by_subject,updated_by_subject) values ('tenant-egueducation','inst-001',$1,'primary','0148 offer',current_date,'0148-admission-admin','0148-admission-admin') returning id::text`, "OFFER-"+uuid.NewString()).Scan(&fixture.offeringID); err != nil {
		t.Fatalf("seed admission offering: %v", err)
	}
	if err := pool.QueryRow(ctx, `insert into school_locations(tenant_code,institution_id,code,name,effective_from,created_by_subject,updated_by_subject) values ('tenant-egueducation','inst-001',$1,'0148 location',current_date,'0148-admission-admin','0148-admission-admin') returning id::text`, "LOC-"+uuid.NewString()).Scan(&fixture.locationID); err != nil {
		t.Fatalf("seed admission location: %v", err)
	}
	if err := pool.QueryRow(ctx, `insert into school_offering_authorizations(tenant_code,institution_id,offering_id,location_id,status,authority_name,decision_reference,capacity,effective_from,source_id,created_by_subject,updated_by_subject) values ('tenant-egueducation','inst-001',$1,$2,'authorized','ARACIP',$3,2,current_date,$4,'0148-admission-admin','0148-admission-admin') returning id::text`, fixture.offeringID, fixture.locationID, "AUTH-"+uuid.NewString(), sourceID).Scan(&fixture.authorizationID); err != nil {
		t.Fatalf("seed admission authorization: %v", err)
	}
	if err := pool.QueryRow(ctx, `insert into education_school_classes(tenant_code,institution_id,class_code,class_name,school_year,grade_level) values ('tenant-egueducation','inst-001',$1,'0148 class','2035-2036','I') returning id::text`, "CLASS-"+uuid.NewString()).Scan(&classID); err != nil {
		t.Fatalf("seed admission class: %v", err)
	}
	fixture.classID = classID
	if err := pool.QueryRow(ctx, `insert into school_admission_class_offering_contexts(tenant_code,institution_id,class_id,offering_id,location_id,authorization_id,school_year,shift,effective_from,created_by_subject,updated_by_subject) values ('tenant-egueducation','inst-001',$1,$2,$3,$4,'2035-2036','day',current_date,'0148-admission-admin','0148-admission-admin') returning id::text`, classID, fixture.offeringID, fixture.locationID, fixture.authorizationID).Scan(&fixture.contextID); err != nil {
		t.Fatalf("seed admission class context: %v", err)
	}
	return fixture
}

func seedAdmissionApplication(t *testing.T, ctx context.Context, pool *SessionPool, campaignID, suffix string) string {
	t.Helper()
	var partyID, applicationID string
	if err := pool.QueryRow(ctx, `insert into app_parties(tenant_code,institution_id,code,party_type,display_name) values ('tenant-egueducation','inst-001',$1,'physical',$2) returning id::text`, "APPLICANT-"+uuid.NewString(), "Applicant "+suffix).Scan(&partyID); err != nil {
		t.Fatalf("seed applicant party: %v", err)
	}
	if err := pool.QueryRow(ctx, `insert into school_admission_applications(tenant_code,institution_id,campaign_id,application_no,candidate_party_id,created_by_subject,updated_by_subject) values ('tenant-egueducation','inst-001',$1,$2,$3,'0148-admission-admin','0148-admission-admin') returning id::text`, campaignID, "APP-"+uuid.NewString(), partyID).Scan(&applicationID); err != nil {
		t.Fatalf("seed admission application: %v", err)
	}
	return applicationID
}

func seedAdmissionWORMSnapshot(t *testing.T, ctx context.Context, pool *SessionPool) (string, string) {
	t.Helper()
	var documentID, versionID string
	if err := pool.QueryRow(ctx, `insert into archive_documents(institution_id,title,original_file_name,mime_type,source_kind,status) values ('inst-001','0148 evidence','0148.pdf','application/pdf','upload','ready') returning id::text`).Scan(&documentID); err != nil {
		t.Fatalf("seed admission archive document: %v", err)
	}
	if err := pool.QueryRow(ctx, `insert into archive_document_versions(document_id,institution_id,version_no,mime_type,bucket_name,object_key,hash_sha256,status,source_bucket,source_object_key,source_sha256,source_object_version_id,retention_until) values ($1,'inst-001',1,'application/pdf','artifact-bucket','artifact/key.pdf',repeat('c',64),'active','source-bucket','source/key.pdf',repeat('d',64),'source-version-0148',now()+interval '30 days') returning id::text`, documentID).Scan(&versionID); err != nil {
		t.Fatalf("seed admission WORM archive version: %v", err)
	}
	return documentID, versionID
}

func assertAdmissionCheckViolation(t *testing.T, err error, messagePart string) {
	t.Helper()
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23514" || (messagePart != "" && !strings.Contains(pgErr.Message, messagePart)) {
		t.Fatalf("admission check error=%v, want SQLSTATE 23514 containing %q", err, messagePart)
	}
}
