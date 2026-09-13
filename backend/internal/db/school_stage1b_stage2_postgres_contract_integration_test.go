package db

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestSchoolStage1BAndStage2PostgresContracts exercises the database boundary
// with a fresh database and a NOINHERIT/NOBYPASSRLS application role.  It is
// intentionally DSN-gated: ordinary unit-test runs compile it but do not need
// a PostgreSQL server; CI supplies a disposable TEST_DATABASE_URL.
func TestSchoolStage1BAndStage2PostgresContracts(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("EGUEDUCATION_TEST_DATABASE_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	}
	if dsn == "" {
		t.Skip("Stage 1B/Stage 2 PostgreSQL contract test requires TEST_DATABASE_URL")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	it := newTenantGrantRLSIntegrationDatabase(t, ctx, dsn)
	admin, err := pgxpool.NewWithConfig(ctx, it.databaseConfig.Copy())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	if err := Migrate(ctx, admin); err != nil {
		t.Fatalf("migrate fresh Stage 1B/Stage 2 database: %v", err)
	}

	role := tenantGrantQuoteIdentifier(it.roleName)
	for _, statement := range []string{
		"grant usage on schema public to " + role,
		"grant select on app_parties to " + role,
		// Profile/contract writes append tenant-scoped version evidence through
		// invoker triggers; keep RLS enabled and grant only the required ledger operations.
		"grant select, insert on app_entity_versions to " + role,
		"grant select, insert, update on school_institution_profiles to " + role,
		"grant select, insert on school_policy_evaluations to " + role,
		"grant select, insert on school_regulatory_sources, school_institution_profiles_v2, school_profile_sources, school_operation_policy_inputs to " + role,
		"grant select, insert, update, delete on school_contracts, school_contract_versions, school_operations_outbox to " + role,
	} {
		if _, err := admin.Exec(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}

	config := it.databaseConfig.Copy()
	config.ConnConfig.User = it.roleName
	config.ConnConfig.Password = it.rolePassword
	config.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	session := NewSessionPool(pool)

	type scope struct{ tenant, institution string }
	scopes := []scope{{"tenant-egueducation", "inst-001"}, {"tenant-balotesti", "inst-balotesti"}}
	suppliers := make([]uuid.UUID, len(scopes))
	for n, s := range scopes {
		suppliers[n] = uuid.New()
		if _, err := admin.Exec(ctx, `insert into app_parties(id,tenant_code,institution_id,code,party_type,display_name,active) values($1,$2,$3,$4,'legal',$5,true)`, suppliers[n], s.tenant, s.institution, "stage2-supplier-"+s.tenant, "Supplier "+s.tenant); err != nil {
			t.Fatalf("seed scoped supplier %s: %v", s.tenant, err)
		}
	}

	for n, s := range scopes {
		rctx, release, err := AcquireRequestConn(ctx, pool, SessionConfig{TenantID: s.tenant, InstitutionID: s.institution, ActorSubject: "stage1b-stage2-contract-test"})
		if err != nil {
			t.Fatal(err)
		}
		func() {
			defer release()
			profileV2, policyID := stage1BStage2PolicyFixture(t, rctx, session, s)

			// The composite FK must bind both the profile identity and its exact
			// version; a made-up revision must never be silently accepted.
			if _, err := session.Exec(rctx, `insert into school_operation_policy_inputs(tenant_code,institution_id,operation_code,profile_id,profile_version,context,checksum_sha256,created_by_subject) values($1,$2,'contract.create',$3,2,'{}',repeat('a',64),'contract-test')`, s.tenant, s.institution, profileV2); !isPostgresCode(err, "23503") {
				t.Fatalf("%s: policy input accepted wrong profile version: %v", s.tenant, err)
			}

			other := (n + 1) % len(scopes)
			contractID := uuid.New()
			if _, err := session.Exec(rctx, `insert into school_contracts(id,tenant_code,institution_id,supplier_party_id,contract_number,title,starts_on,policy_evaluation_id,created_by_subject,updated_by_subject) values($1,$2,$3,$4,$5,'Scoped contract',current_date,$6,'contract-test','contract-test')`, contractID, s.tenant, s.institution, suppliers[n], "STAGE2-"+s.tenant, policyID); err != nil {
				t.Fatalf("%s: insert scoped supplier contract: %v", s.tenant, err)
			}
			if _, err := session.Exec(rctx, `insert into school_contracts(tenant_code,institution_id,supplier_party_id,contract_number,title,starts_on,policy_evaluation_id,created_by_subject,updated_by_subject) values($1,$2,$3,$4,'Forged supplier',current_date,$5,'contract-test','contract-test')`, s.tenant, s.institution, suppliers[other], "FORGED-"+s.tenant, policyID); err == nil {
				t.Fatalf("%s: cross-scope supplier was accepted", s.tenant)
			}

			// The service's compare-and-swap predicate is deliberately exercised
			// here: one current revision succeeds and the stale revision affects no
			// row. The history trigger then records both snapshots.
			tag, err := session.Exec(rctx, `update school_contracts set title='Amended contract', expected_version=expected_version+1, updated_by_subject='contract-test' where id=$1 and expected_version=1`, contractID)
			if err != nil || tag.RowsAffected() != 1 {
				t.Fatalf("%s: current contract compare-and-swap: rows=%d err=%v", s.tenant, tag.RowsAffected(), err)
			}
			tag, err = session.Exec(rctx, `update school_contracts set title='Stale overwrite', expected_version=expected_version+1 where id=$1 and expected_version=1`, contractID)
			if err != nil || tag.RowsAffected() != 0 {
				t.Fatalf("%s: stale contract update rows=%d err=%v", s.tenant, tag.RowsAffected(), err)
			}
			var versions int
			if err := session.QueryRow(rctx, `select count(*) from school_contract_versions where contract_id=$1`, contractID).Scan(&versions); err != nil || versions != 2 {
				t.Fatalf("%s: immutable contract history=%d err=%v, want 2", s.tenant, versions, err)
			}

			if _, err := session.Exec(rctx, `update school_contracts set lifecycle_status='signed', expected_version=3 where id=$1 and expected_version=2`, contractID); err != nil {
				t.Fatalf("%s: finalize contract: %v", s.tenant, err)
			}
			if _, err := session.Exec(rctx, `delete from school_contracts where id=$1`, contractID); err == nil {
				t.Fatalf("%s: final contract hard-delete was accepted", s.tenant)
			}

			var outboxID uuid.UUID
			if err := session.QueryRow(rctx, `insert into school_operations_outbox(tenant_code,institution_id,aggregate_type,aggregate_id,event_type,payload) values($1,$2,'contract',$3::uuid,'contract.archive_pending',jsonb_build_object('contract_id',$3::uuid)) returning id`, s.tenant, s.institution, contractID).Scan(&outboxID); err != nil {
				t.Fatalf("%s: persist scoped outbox intent: %v", s.tenant, err)
			}
			if _, err := session.Exec(rctx, `delete from school_operations_outbox where id=$1`, outboxID); err == nil {
				t.Fatalf("%s: durable outbox intent was hard-deleted", s.tenant)
			}
			var leaked int
			if err := session.QueryRow(rctx, `select count(*) from school_contracts where tenant_code=$1 and institution_id=$2`, scopes[other].tenant, scopes[other].institution).Scan(&leaked); err != nil || leaked != 0 {
				t.Fatalf("%s: cross-scope contract visibility=%d err=%v", s.tenant, leaked, err)
			}
		}()
	}
}

func stage1BStage2PolicyFixture(t *testing.T, ctx context.Context, session *SessionPool, s struct{ tenant, institution string }) (uuid.UUID, uuid.UUID) {
	t.Helper()
	var sourceID, profileV2, profileV1, evaluationID uuid.UUID
	if err := session.QueryRow(ctx, `insert into school_regulatory_sources(tenant_code,institution_id,source_kind,citation,source_url,checksum_sha256,verified_at,verified_by_subject,revalidation_owner_subject,created_by_subject,updated_by_subject) values($1,$2,'law',$3,'https://example.test/law',repeat('c',64),now(),'contract-test','contract-test','contract-test','contract-test') returning id`, s.tenant, s.institution, "stage1b-source-"+s.tenant).Scan(&sourceID); err != nil {
		t.Fatal(err)
	}
	profileV2 = uuid.New()
	if _, err := session.Exec(ctx, `insert into school_institution_profiles_v2(id,tenant_code,institution_id,version,status,legal_form,effective_from,profile_series_id,approved_by_subject,approved_at,created_by_subject,updated_by_subject) values($1,$2,$3,1,'active','public',current_date,$1,'contract-test',now(),'contract-test','contract-test')`, profileV2, s.tenant, s.institution); err != nil {
		t.Fatal(err)
	}
	if _, err := session.Exec(ctx, `insert into school_profile_sources(tenant_code,institution_id,profile_id,source_id,purpose,created_by_subject) values($1,$2,$3,$4,'classification','contract-test')`, s.tenant, s.institution, profileV2, sourceID); err != nil {
		t.Fatalf("bind Stage 1B profile v2 to regulatory source: %v", err)
	}
	if err := session.QueryRow(ctx, `insert into school_institution_profiles(tenant_code,institution_id,version,status,school_legal_form,regulatory_profile,effective_from,source_reference,approved_by_subject,approved_at,created_by_subject,updated_by_subject) values($1,$2,2,'active','public','ro.public.preuniversity',current_date,$3,'contract-test',now(),'contract-test','contract-test') returning id`, s.tenant, s.institution, sourceID.String()).Scan(&profileV1); err != nil {
		t.Fatal(err)
	}
	if err := session.QueryRow(ctx, `insert into school_policy_evaluations(tenant_code,institution_id,profile_id,profile_version,evaluated_by_subject,capabilities,blocked,checksum_sha256,created_by_subject,updated_by_subject) values($1,$2,$3,2,'contract-test','[]',false,repeat('b',64),'contract-test','contract-test') returning id`, s.tenant, s.institution, profileV1).Scan(&evaluationID); err != nil {
		t.Fatal(err)
	}
	return profileV2, evaluationID
}
