//go:build integration

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

func TestArchiveSeriesRetentionLifecyclePostgresIntegration(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("EGUEDUCATION_TEST_DATABASE_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	}
	if dsn == "" {
		t.Skip("archive retention PostgreSQL integration requires TEST_DATABASE_URL")
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
		t.Fatalf("migrate archive retention: %v", err)
	}
	sessions := NewSessionPool(pool)
	proposer := "0156-proposer-" + uuid.NewString()
	approver := "0156-approver-" + uuid.NewString()
	adminCtx, releaseAdmin, err := AcquireRequestConn(ctx, pool, SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: proposer, IsSuperAdmin: true})
	if err != nil {
		t.Fatal(err)
	}
	var taxonomyID, sourceID, ruleID string
	if err = sessions.QueryRow(adminCtx, `insert into archive_taxonomy_nodes(institution_id,code,label) values('inst-001',$1,'0156 retention') returning id::text`, "retention-"+uuid.NewString()).Scan(&taxonomyID); err != nil {
		t.Fatalf("seed taxonomy: %v", err)
	}
	if err = sessions.QueryRow(adminCtx, `insert into school_regulatory_sources(tenant_code,institution_id,source_kind,citation,source_url,checksum_sha256,effective_from,verified_at,verified_by_subject,revalidation_owner_subject,created_by_subject,updated_by_subject,status) values('tenant-egueducation','inst-001','law',$1,'https://example.test/0156',encode(digest('retention evidence','sha256'),'hex'),current_date,now(),$2,$2,$2,$2,'draft') returning id::text`, "0156 source "+uuid.NewString(), proposer).Scan(&sourceID); err != nil {
		t.Fatalf("seed verified source: %v", err)
	}
	// Exercise the actual database guards; this fixture does not assert network
	// retrieval or legal applicability (the HTTP suite covers service commands).
	tx, err := sessions.Begin(adminCtx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(adminCtx) //nolint:errcheck
	var evidenceID string
	if err = tx.QueryRow(adminCtx, `insert into school_regulatory_source_evidence
		(tenant_code,institution_id,source_id,source_version,requested_url,source_snapshot,
		retrieved_url,content_type,content,sha256,retrieved_at,retrieved_by_subject)
		select tenant_code,institution_id,id,2,source_url,to_jsonb(s),source_url,'text/plain',
		convert_to('retention evidence','UTF8'),checksum_sha256,now(),$2
		from school_regulatory_sources s where id=$1::uuid returning id::text`,
		sourceID, proposer).Scan(&evidenceID); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(adminCtx, `update school_regulatory_sources set status='verified',expected_version=2 where id=$1::uuid`, sourceID); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(adminCtx, `update school_regulatory_sources set status='active',expected_version=3 where id=$1::uuid`, sourceID); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(adminCtx, `insert into school_regulatory_source_activations
		(tenant_code,institution_id,source_id,evidence_id,assessment,activated_by_subject)
		values('tenant-egueducation','inst-001',$1::uuid,$2::uuid,'Database fixture assessment',$3)`,
		sourceID, evidenceID, approver); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(adminCtx); err != nil {
		t.Fatal(err)
	}
	if err = sessions.QueryRow(adminCtx, `insert into archive_series_retention_rules(tenant_code,institution_id,taxonomy_node_id,source_id,source_checksum_sha256,anchor_kind,duration_model,minimum_retention_days,effective_from,proposed_by_subject) values('tenant-egueducation','inst-001',$1::uuid,$2::uuid,encode(digest('retention evidence','sha256'),'hex'),'intake_received_at','minimum_days',365,current_date,$3) returning id::text`, taxonomyID, sourceID, proposer).Scan(&ruleID); err != nil {
		t.Fatalf("propose retention rule: %v", err)
	}
	if _, err = sessions.Exec(adminCtx, `update archive_series_retention_rules set status='active' where id=$1::uuid`, ruleID); err == nil || !strings.Contains(err.Error(), "distinct actor") {
		t.Fatalf("self approval err=%v, want distinct actor rejection", err)
	}
	var overlapID string
	if err = sessions.QueryRow(adminCtx, `insert into archive_series_retention_rules(tenant_code,institution_id,taxonomy_node_id,source_id,source_checksum_sha256,anchor_kind,duration_model,minimum_retention_days,effective_from,proposed_by_subject) values('tenant-egueducation','inst-001',$1::uuid,$2::uuid,encode(digest('retention evidence','sha256'),'hex'),'intake_received_at','minimum_days',365,current_date,$3) returning id::text`, taxonomyID, sourceID, proposer).Scan(&overlapID); err != nil {
		t.Fatalf("propose overlap candidate: %v", err)
	}
	releaseAdmin()
	approverCtx, releaseApprover, err := AcquireRequestConn(ctx, pool, SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: approver, IsSuperAdmin: true})
	if err != nil {
		t.Fatal(err)
	}
	defer releaseApprover()
	if _, err = sessions.Exec(approverCtx, `update archive_series_retention_rules set status='active' where id=$1::uuid`, ruleID); err != nil {
		t.Fatalf("approve retention rule: %v", err)
	}
	if _, err = sessions.Exec(approverCtx, `update archive_series_retention_rules set status='active' where id=$1::uuid`, overlapID); err == nil || !strings.Contains(err.Error(), "active windows overlap") {
		t.Fatalf("overlap approval err=%v", err)
	}
	if _, err = sessions.Exec(approverCtx, `update archive_series_retention_rules set status='retired',retirement_reason='superseded rule' where id=$1::uuid`, ruleID); err != nil {
		t.Fatalf("retire retention rule: %v", err)
	}
	var approvedBy string
	var approvedAt time.Time
	var version int
	if err = sessions.QueryRow(approverCtx, `select approved_by_subject,approved_at,expected_version from archive_series_retention_rules where id=$1::uuid`, ruleID).Scan(&approvedBy, &approvedAt, &version); err != nil || approvedBy != approver || approvedAt.IsZero() || version != 3 {
		t.Fatalf("retired rule lost approval provenance: actor=%q at=%v version=%d err=%v", approvedBy, approvedAt, version, err)
	}
	if _, err = sessions.Exec(approverCtx, `update archive_series_retention_rules set minimum_retention_days=1 where id=$1::uuid`, ruleID); err == nil || !strings.Contains(err.Error(), "provenance is immutable") {
		t.Fatalf("immutable retention provenance err=%v", err)
	}
}
