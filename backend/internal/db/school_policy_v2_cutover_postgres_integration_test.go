package db

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestSchoolPolicyV2CutoverUpgradePostgresIntegration proves the actual
// deployed upgrade shape: a database with data through 0144 is upgraded by
// Migrate through the current admission migrations. It intentionally does not simulate the upgrade by
// removing objects from a current schema, because that misses dependencies
// created while the legacy and Stage 1B models coexist.
//
// The test requires an administrator DSN capable of creating a disposable
// database; ordinary unit runs skip it.
func TestSchoolPolicyV2CutoverUpgradePostgresIntegration(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("EGUEDUCATION_TEST_DATABASE_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	}
	if dsn == "" {
		t.Skip("0145 cutover PostgreSQL upgrade test requires EGUEDUCATION_TEST_DATABASE_URL or TEST_DATABASE_URL")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	it := newTenantGrantRLSIntegrationDatabase(t, ctx, dsn)
	admin, err := pgxpool.NewWithConfig(ctx, it.databaseConfig.Copy())
	if err != nil {
		t.Fatalf("open disposable 0145 upgrade database: %v", err)
	}
	t.Cleanup(admin.Close)

	if err := migrateSchoolPolicyCutoverThrough0144(ctx, admin); err != nil {
		t.Fatalf("apply migrations through 0144: %v", err)
	}
	var pre0145MigrationCount int
	if err := admin.QueryRow(ctx, `select count(*) from schema_migrations`).Scan(&pre0145MigrationCount); err != nil {
		t.Fatalf("count pre-0145 migration ledger: %v", err)
	}

	const tenantA, institutionA = "tenant-egueducation", "inst-001"
	const tenantB, institutionB = "tenant-balotesti", "inst-balotesti"
	adminCtx, releaseAdmin, err := AcquireRequestConn(ctx, admin, SessionConfig{
		TenantID: tenantA, InstitutionID: institutionA, ActorSubject: "0145-upgrade-admin", IsSuperAdmin: true,
	})
	if err != nil {
		t.Fatalf("bind upgrade admin session: %v", err)
	}
	adminSession := NewSessionPool(admin)

	v2ProfileID := uuid.NewString()
	legacyProfileID := uuid.NewString()
	if _, err = adminSession.Exec(adminCtx, `
		insert into school_institution_profiles_v2
			(id, tenant_code, institution_id, version, status, legal_form, effective_from, profile_series_id, approved_by_subject, approved_at, created_by_subject, updated_by_subject)
		values ($1,$2,$3,1,'active','public',current_date,$1,'0145-upgrade-admin',now(),'0145-upgrade-admin','0145-upgrade-admin')
	`, v2ProfileID, tenantA, institutionA); err != nil {
		t.Fatalf("seed pre-0145 v2 profile: %v", err)
	}
	if _, err = adminSession.Exec(adminCtx, `
		insert into school_institution_profiles
			(id, tenant_code, institution_id, version, status, school_legal_form, regulatory_profile, effective_from, approved_by_subject, approved_at, created_by_subject, updated_by_subject)
		values ($1,$2,$3,2,'active','public','ro.public.preuniversity',current_date,'0145-upgrade-admin',now(),'0145-upgrade-admin','0145-upgrade-admin')
	`, legacyProfileID, tenantA, institutionA); err != nil {
		t.Fatalf("seed pre-0145 legacy profile: %v", err)
	}

	var packID, packChecksum string
	if err = adminSession.QueryRow(adminCtx, `
		select id::text, checksum_sha256
		from school_policy_pack_versions
		where tenant_code=$1 and institution_id=$2 and pack_code='common.ro' and version=1
	`, tenantA, institutionA).Scan(&packID, &packChecksum); err != nil {
		t.Fatalf("load pre-0145 policy pack: %v", err)
	}
	legacyAssignmentID := uuid.NewString()
	if _, err = adminSession.Exec(adminCtx, `
		insert into school_policy_assignments
			(id,tenant_code,institution_id,policy_pack_version_id,profile_id,profile_version,pack_code,assignment_kind,status,effective_from,assigned_by_subject,created_by_subject,updated_by_subject)
		values ($1,$2,$3,$4,$5,2,'common.ro','common','active',current_date,'0145-upgrade-admin','0145-upgrade-admin','0145-upgrade-admin')
	`, legacyAssignmentID, tenantA, institutionA, packID, legacyProfileID); err != nil {
		t.Fatalf("seed pre-0145 legacy assignment: %v", err)
	}
	legacyOverrideID := uuid.NewString()
	if _, err = adminSession.Exec(adminCtx, `
		insert into school_policy_overrides
			(id,tenant_code,institution_id,policy_assignment_id,key,value,justification,status,created_by_subject,updated_by_subject)
		values ($1,$2,$3,$4,'cutover.fixture','true'::jsonb,'upgrade fixture','draft','0145-upgrade-admin','0145-upgrade-admin')
	`, legacyOverrideID, tenantA, institutionA, legacyAssignmentID); err != nil {
		t.Fatalf("seed pre-0145 legacy override: %v", err)
	}
	legacyEvaluationID := uuid.NewString()
	if _, err = adminSession.Exec(adminCtx, `
		insert into school_policy_evaluations
			(id,tenant_code,institution_id,profile_id,profile_version,evaluated_by_subject,policy_pack_version_ids,capabilities,blocked,checksum_sha256,created_by_subject,updated_by_subject)
		values ($1,$2,$3,$4,2,'0145-upgrade-admin',array[$5::uuid],'[]'::jsonb,false,repeat('1',64),'0145-upgrade-admin','0145-upgrade-admin')
	`, legacyEvaluationID, tenantA, institutionA, legacyProfileID, packID); err != nil {
		t.Fatalf("seed pre-0145 legacy evaluation: %v", err)
	}

	inputID, evaluationID := uuid.NewString(), uuid.NewString()
	if _, err = adminSession.Exec(adminCtx, `
		insert into school_operation_policy_inputs
			(id,tenant_code,institution_id,operation_code,profile_id,profile_version,context,checksum_sha256,created_by_subject)
		values ($1,$2,$3,'contract.create',$4,1,'{}'::jsonb,repeat('2',64),'0145-upgrade-admin')
	`, inputID, tenantA, institutionA, v2ProfileID); err != nil {
		t.Fatalf("seed pre-0145 v2 input: %v", err)
	}
	if _, err = adminSession.Exec(adminCtx, `
		insert into school_operation_policy_evaluations_v2
			(id,tenant_code,institution_id,input_id,allowed,capabilities,obligations,checksum_sha256,evaluated_by_subject)
		values ($1,$2,$3,$4,true,'[]'::jsonb,'[]'::jsonb,repeat('3',64),'0145-upgrade-admin')
	`, evaluationID, tenantA, institutionA, inputID); err != nil {
		t.Fatalf("seed pre-0145 v2 evaluation: %v", err)
	}

	// Migrate needs to acquire its own connection for the migration advisory
	// lock. Release the bound request connection before invoking it.
	releaseAdmin()
	if err = Migrate(ctx, admin); err != nil {
		t.Fatalf("upgrade pre-0145 database through 0145: %v", err)
	}
	var post0145MigrationCount int
	var migrationsAfter0144 []string
	if err = admin.QueryRow(ctx, `
		select count(*), coalesce(array_agg(version order by version) filter (where version > '0144_education_portfolio_review_evidence.sql'), '{}')
		from schema_migrations
	`).Scan(&post0145MigrationCount, &migrationsAfter0144); err != nil {
		t.Fatalf("read post-0145 migration ledger: %v", err)
	}
	expectedNewer := []string{
		"0145_school_policy_v2_cutover_expand.sql",
		"0148_school_admissions.sql",
		"0149_school_admission_dss_bindings.sql",
		"0150_school_admission_legal_authority.sql",
		"0151_school_admission_signed_payload_contract.sql",
		"0152_school_admission_legal_prepare_finalize.sql",
		"0153_education_signed_artifact_provenance_reconciliation.sql",
		"0154_school_admission_preparation_retention.sql",
		"0155_school_admission_worm_ingestion_intents.sql",
		"0156_earchiva_series_retention_rules.sql",
		"0157_school_regulatory_source_evidence.sql",
		"0158_archive_retention_activation_evidence.sql",
		"0160_portfolio_active_custody_hold.sql",
		"0161_portfolio_retention_calendar_years.sql",
		"0162_portfolio_custody_upload_intents.sql",
		"0163_portfolio_storage_lifecycle.sql",
		"0164_portfolio_custody_recovery.sql",
		"0165_portfolio_retention_expiry_review.sql",
	}
	if post0145MigrationCount != pre0145MigrationCount+len(expectedNewer) || strings.Join(migrationsAfter0144, "\n") != strings.Join(expectedNewer, "\n") {
		t.Fatalf("0145 incremental ledger changed unexpectedly: before=%d after=%d newer=%v", pre0145MigrationCount, post0145MigrationCount, migrationsAfter0144)
	}
	adminCtx, releaseAdmin, err = AcquireRequestConn(ctx, admin, SessionConfig{
		TenantID: tenantA, InstitutionID: institutionA, ActorSubject: "0145-upgrade-admin", IsSuperAdmin: true,
	})
	if err != nil {
		t.Fatalf("rebind upgrade admin session after migration: %v", err)
	}
	defer releaseAdmin()
	var applied int
	if err = adminSession.QueryRow(adminCtx, `select count(*) from schema_migrations where version='0145_school_policy_v2_cutover_expand.sql'`).Scan(&applied); err != nil || applied != 1 {
		t.Fatalf("0145 migration ledger = %d, %v; want exactly one", applied, err)
	}

	apiProfileID := uuid.NewString()
	if _, err = adminSession.Exec(adminCtx, `
		insert into school_profile_cutover_identity
			(tenant_code,institution_id,profile_v2_id,profile_v2_version,api_profile_id,api_version,legacy_profile_id,legacy_profile_version,mapping_kind,created_by_subject)
		values ($1,$2,$3,1,$4,2,$5,2,'dual_write','0145-upgrade-admin')
	`, tenantA, institutionA, v2ProfileID, apiProfileID, legacyProfileID); err != nil {
		t.Fatalf("insert exact profile cutover identity: %v", err)
	}
	bindingID := uuid.NewString()
	if _, err = adminSession.Exec(adminCtx, `
		insert into school_operation_policy_bindings_v2
			(id,tenant_code,institution_id,profile_v2_id,profile_v2_version,profile_series_id,policy_pack_version_id,pack_code,policy_pack_version,assignment_kind,status,effective_from,legacy_assignment_id,created_by_subject,updated_by_subject)
		values ($1,$2,$3,$4,1,$4,$5,'common.ro',1,'common','active',current_date,$6,'0145-upgrade-admin','0145-upgrade-admin')
	`, bindingID, tenantA, institutionA, v2ProfileID, packID, legacyAssignmentID); err != nil {
		t.Fatalf("insert exact v2 policy binding: %v", err)
	}
	if _, err = adminSession.Exec(adminCtx, `
		insert into school_operation_policy_input_bindings
			(tenant_code,institution_id,input_id,binding_id,profile_v2_id,profile_v2_version,profile_series_id,policy_pack_version_id,policy_pack_checksum_sha256,override_snapshot,override_checksum_sha256,created_by_subject)
		values ($1,$2,$3,$4,$5,1,$5,$6,$7,'[]'::jsonb,repeat('4',64),'0145-upgrade-admin')
	`, tenantA, institutionA, inputID, bindingID, v2ProfileID, packID, packChecksum); err != nil {
		t.Fatalf("record exact input provenance: %v", err)
	}
	if _, err = adminSession.Exec(adminCtx, `
		insert into school_policy_evaluation_cutover_identity
			(tenant_code,institution_id,legacy_evaluation_id,v2_evaluation_id,input_id,conversion_kind,created_by_subject)
		values ($1,$2,$3,$4,$5,'dual_write','0145-upgrade-admin')
	`, tenantA, institutionA, legacyEvaluationID, evaluationID, inputID); err != nil {
		t.Fatalf("record v1/v2 evaluation bridge: %v", err)
	}

	mismatchInputID := uuid.NewString()
	if _, err = adminSession.Exec(adminCtx, `
		insert into school_operation_policy_inputs
			(id,tenant_code,institution_id,operation_code,profile_id,profile_version,context,checksum_sha256,created_by_subject)
		values ($1,$2,$3,'contract.create',$4,1,'{}'::jsonb,repeat('5',64),'0145-upgrade-admin')
	`, mismatchInputID, tenantA, institutionA, v2ProfileID); err != nil {
		t.Fatalf("seed mismatch-policy input: %v", err)
	}
	var packChecksumFK string
	if err = adminSession.QueryRow(adminCtx, `
		select conname
		from pg_constraint
		where conrelid='school_operation_policy_input_bindings'::regclass
			and contype='f'
			and pg_get_constraintdef(oid) like '%policy_pack_version_id, policy_pack_checksum_sha256%'
	`).Scan(&packChecksumFK); err != nil {
		t.Fatalf("locate exact pack-checksum provenance FK: %v", err)
	}
	_, err = adminSession.Exec(adminCtx, `
		insert into school_operation_policy_input_bindings
			(tenant_code,institution_id,input_id,binding_id,profile_v2_id,profile_v2_version,profile_series_id,policy_pack_version_id,policy_pack_checksum_sha256,override_snapshot,override_checksum_sha256,created_by_subject)
		values ($1,$2,$3,$4,$5,1,$5,$6,repeat('f',64),'[]'::jsonb,repeat('4',64),'0145-upgrade-admin')
	`, tenantA, institutionA, mismatchInputID, bindingID, v2ProfileID, packID)
	assertPGFailure(t, err, "23503", packChecksumFK, "")
	_, err = adminSession.Exec(adminCtx, `update school_profile_cutover_identity set api_version=3 where tenant_code=$1 and institution_id=$2 and profile_v2_id=$3`, tenantA, institutionA, v2ProfileID)
	assertPGFailure(t, err, "P0001", "", "school regulatory snapshots are immutable")
	_, err = adminSession.Exec(adminCtx, `delete from school_operation_policy_input_bindings where tenant_code=$1 and institution_id=$2 and input_id=$3 and binding_id=$4`, tenantA, institutionA, inputID, bindingID)
	assertPGFailure(t, err, "P0001", "", "school regulatory snapshots are immutable")

	v2ProfileB := uuid.NewString()
	if _, err = adminSession.Exec(adminCtx, `
		insert into school_institution_profiles_v2
			(id,tenant_code,institution_id,version,status,legal_form,effective_from,profile_series_id,approved_by_subject,approved_at,created_by_subject,updated_by_subject)
		values ($1,$2,$3,1,'active','private',current_date,$1,'0145-upgrade-admin',now(),'0145-upgrade-admin','0145-upgrade-admin')
	`, v2ProfileB, tenantB, institutionB); err != nil {
		t.Fatalf("seed tenant-B v2 profile: %v", err)
	}
	if _, err = adminSession.Exec(adminCtx, `
		insert into school_profile_cutover_identity
			(tenant_code,institution_id,profile_v2_id,profile_v2_version,api_profile_id,api_version,mapping_kind,created_by_subject)
		values ($1,$2,$3,1,$4,1,'v2_native','0145-upgrade-admin')
	`, tenantB, institutionB, v2ProfileB, uuid.NewString()); err != nil {
		t.Fatalf("seed tenant-B cutover identity: %v", err)
	}

	role := tenantGrantQuoteIdentifier(it.roleName)
	for _, statement := range []string{
		"grant usage on schema public to " + role,
		"grant select, insert on school_profile_cutover_identity to " + role,
		"grant insert on school_operation_policy_bindings_v2 to " + role,
		"grant insert on school_operation_policy_input_bindings to " + role,
	} {
		if _, err = adminSession.Exec(adminCtx, statement); err != nil {
			t.Fatalf("grant RLS test access: %v", err)
		}
	}
	userConfig := it.databaseConfig.Copy()
	userConfig.ConnConfig.User, userConfig.ConnConfig.Password, userConfig.MaxConns = it.roleName, it.rolePassword, 1
	userPool, err := pgxpool.NewWithConfig(ctx, userConfig)
	if err != nil {
		t.Fatalf("open non-bypass RLS pool: %v", err)
	}
	t.Cleanup(userPool.Close)
	userSession := NewSessionPool(userPool)
	userCtx, releaseUser, err := AcquireRequestConn(ctx, userPool, SessionConfig{TenantID: tenantB, InstitutionID: institutionB, ActorSubject: "0145-rls"})
	if err != nil {
		t.Fatalf("bind non-bypass RLS session: %v", err)
	}
	defer releaseUser()
	var visible, invisible int
	if err = userSession.QueryRow(userCtx, `select count(*) from school_profile_cutover_identity where tenant_code=$1 and institution_id=$2`, tenantB, institutionB).Scan(&visible); err != nil {
		t.Fatalf("query same-scope cutover identity: %v", err)
	}
	if visible != 1 {
		t.Fatalf("RLS did not expose tenant-B identity to its own session: %d", visible)
	}
	if err = userSession.QueryRow(userCtx, `select count(*) from school_profile_cutover_identity where tenant_code=$1 and institution_id=$2`, tenantA, institutionA).Scan(&invisible); err != nil {
		t.Fatalf("query cross-scope cutover identity: %v", err)
	}
	if invisible != 0 {
		t.Fatalf("RLS exposed %d foreign cutover identities", invisible)
	}
	_, err = userSession.Exec(userCtx, `
		insert into school_operation_policy_bindings_v2
			(id,tenant_code,institution_id,profile_v2_id,profile_v2_version,profile_series_id,policy_pack_version_id,pack_code,policy_pack_version,assignment_kind,status,effective_from,created_by_subject,updated_by_subject)
		values ($1,$2,$3,$4,1,$4,$5,'common.ro',1,'common','active',current_date,'0145-rls','0145-rls')
	`, uuid.NewString(), tenantA, institutionA, v2ProfileID, packID)
	assertPGFailure(t, err, "42501", "", `new row violates row-level security policy for table "school_operation_policy_bindings_v2"`)
	_, err = userSession.Exec(userCtx, `
		insert into school_operation_policy_input_bindings
			(tenant_code,institution_id,input_id,binding_id,profile_v2_id,profile_v2_version,profile_series_id,policy_pack_version_id,policy_pack_checksum_sha256,override_snapshot,override_checksum_sha256,created_by_subject)
		values ($1,$2,$3,$4,$5,1,$5,$6,$7,'[]'::jsonb,repeat('4',64),'0145-rls')
	`, tenantA, institutionA, mismatchInputID, bindingID, v2ProfileID, packID, packChecksum)
	assertPGFailure(t, err, "42501", "", `new row violates row-level security policy for table "school_operation_policy_input_bindings"`)
}

func assertPGFailure(t *testing.T, err error, wantCode, wantConstraint, wantMessage string) {
	t.Helper()
	if err == nil {
		t.Fatal("statement unexpectedly succeeded")
	}
	pgErr, ok := err.(*pgconn.PgError)
	if !ok {
		t.Fatalf("statement failed with non-PostgreSQL error %T: %v", err, err)
	}
	if pgErr.Code != wantCode {
		t.Fatalf("SQLSTATE=%s, want %s (error: %v)", pgErr.Code, wantCode, err)
	}
	if wantConstraint != "" && pgErr.ConstraintName != wantConstraint {
		t.Fatalf("constraint=%q, want %q (error: %v)", pgErr.ConstraintName, wantConstraint, err)
	}
	if wantMessage != "" && pgErr.Message != wantMessage {
		t.Fatalf("message=%q, want %q", pgErr.Message, wantMessage)
	}
}

// migrateSchoolPolicyCutoverThrough0144 executes the embedded historical
// migrations exactly once and records each in the normal migration ledger.
// Migrate can then exercise the genuine 0145-only incremental path.
func migrateSchoolPolicyCutoverThrough0144(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `create table if not exists schema_migrations (version text primary key, applied_at timestamptz not null default now())`); err != nil {
		return fmt.Errorf("create migration ledger: %w", err)
	}
	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read embedded migrations: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") && entry.Name() <= "0144_education_portfolio_review_evidence.sql" {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		body, err := migrationFiles.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin %s: %w", name, err)
		}
		if _, err = tx.Exec(ctx, string(body)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("apply %s: %w", name, err)
		}
		if _, err = tx.Exec(ctx, `insert into schema_migrations(version) values ($1)`, name); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record %s: %w", name, err)
		}
		if err = tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit %s: %w", name, err)
		}
	}
	return nil
}
