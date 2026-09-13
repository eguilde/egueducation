//go:build integration

package earchiva

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	appdb "github.com/eguilde/egueducation/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPortfolioCustodyRecoveryOperationsRLSAndRetryIntegration(t *testing.T) {
	it := newArchiveIntegrationDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	admin := openArchiveIntegrationPool(t, ctx, it.databaseConfig)
	defer admin.Close()
	if err := appdb.Migrate(ctx, admin); err != nil {
		t.Fatalf("migrate recovery integration database: %v", err)
	}
	grantArchiveIntegrationAccess(t, ctx, admin)
	seedRecoveryOperationActor(t, ctx, admin)
	sessions := appdb.NewSessionPool(it.readerPool)

	portfolioA := seedCustodyIntentPortfolio(t, ctx, admin, "inst-001")
	portfolioB := seedCustodyIntentPortfolio(t, ctx, admin, "inst-balotesti")
	ctxA, releaseA := archiveTenantContext(t, ctx, it.readerPool, "tenant-egueducation", "inst-001")
	defer releaseA()
	ctxB, releaseB := archiveTenantContext(t, ctx, it.readerPool, "tenant-balotesti", "inst-balotesti")
	defer releaseB()
	intentA := seedCustodyUploadIntent(t, ctxA, sessions, "tenant-egueducation", "inst-001", portfolioA)
	intentB := seedCustodyUploadIntent(t, ctxB, sessions, "tenant-balotesti", "inst-balotesti", portfolioB)
	markCustodyIntentStored(t, ctxA, sessions, intentA)
	markCustodyIntentStored(t, ctxB, sessions, intentB)

	forgedCtx, releaseForged, err := appdb.AcquireRequestConn(ctx, it.readerPool, appdb.SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: "recovery-forged-actor"})
	if err != nil {
		t.Fatalf("bind forged actor context: %v", err)
	}
	defer releaseForged()
	if _, err := sessions.Exec(forgedCtx, `insert into portfolio_custody_recovery_operations(intent_id,tenant_code,institution_id,portfolio_id,requested_by_subject,disposition,reason,title,original_file_name,document_date,expected_fingerprint) values($1::uuid,'tenant-egueducation','inst-001',$2::uuid,'recovery-forged-actor','institution_archive_only','forged direct SQL','Recovered custody evidence','evidence.pdf','2026-09-12',$3)`, intentA.intentID, portfolioA, intentA.expectedFingerprint); err == nil || !strings.Contains(err.Error(), "current dual authority") {
		t.Fatalf("forged recovery insert must fail DB dual authority check, got %v", err)
	}
	opA := insertRecoveryOperation(t, ctxA, sessions, intentA, "tenant-egueducation", "inst-001", portfolioA, "institution_archive_only")
	opB := insertRecoveryOperation(t, ctxB, sessions, intentB, "tenant-balotesti", "inst-balotesti", portfolioB, "institution_archive_only")

	var visible int
	if err := sessions.QueryRow(ctxA, `select count(*) from portfolio_custody_recovery_operations where id=$1::uuid`, opB).Scan(&visible); err != nil || visible != 0 {
		t.Fatalf("tenant A sees foreign recovery operation: count=%d err=%v", visible, err)
	}
	if tag, err := sessions.Exec(ctxA, `update portfolio_custody_recovery_operations set status='blocked' where id=$1::uuid`, opB); err != nil || tag.RowsAffected() != 0 {
		t.Fatalf("tenant A changed foreign recovery operation: rows=%d err=%v", tag.RowsAffected(), err)
	}
	if err := sessions.QueryRow(ctxB, `select count(*) from portfolio_custody_recovery_attempts where operation_id=$1::uuid`, opA).Scan(&visible); err != nil || visible != 0 {
		t.Fatalf("tenant B sees foreign recovery attempts: count=%d err=%v", visible, err)
	}
	if _, err := sessions.Exec(ctxB, `update portfolio_custody_recovery_operations set status='leased',attempts=1,lease_owner='fixture-worker',lease_expires_at=now()+interval '5 minutes' where id=$1::uuid`, opB); err == nil || !strings.Contains(err.Error(), "requires bound worker") {
		t.Fatalf("tenant SQL session forged a worker lease: %v", err)
	}
	workerBContext, releaseWorkerB, err := appdb.AcquireRequestConn(ctx, it.readerPool, appdb.SessionConfig{TenantID: "tenant-balotesti", InstitutionID: "inst-balotesti", ActorSubject: "portfolio-custody-recovery-worker"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sessions.Exec(workerBContext, `select set_config('app.portfolio_custody_recovery_worker_id','fixture-worker',false)`); err != nil {
		t.Fatal(err)
	}
	if _, err := sessions.Exec(workerBContext, `update portfolio_custody_recovery_operations set status='leased',attempts=1,lease_owner='fixture-worker',lease_expires_at=now()+interval '5 minutes' where id=$1::uuid`, opB); err != nil {
		t.Fatalf("lease foreign fixture before terminal transition: %v", err)
	}
	if _, err := sessions.Exec(workerBContext, `insert into portfolio_custody_recovery_attempts(operation_id,attempt_no,worker_id,outcome,error_code) values($1::uuid,1,'fixture-worker','blocked','fixture_complete')`, opB); err != nil {
		t.Fatalf("append fixture worker evidence: %v", err)
	}
	if _, err := sessions.Exec(workerBContext, `update portfolio_custody_recovery_operations set status='blocked',lease_owner='',lease_expires_at=null,last_error_code='fixture_complete' where id=$1::uuid`, opB); err != nil {
		t.Fatalf("remove foreign fixture from worker queue: %v", err)
	}
	releaseWorkerB()
	workerAContext, releaseWorkerA, err := appdb.AcquireRequestConn(ctx, it.readerPool, appdb.SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: "portfolio-custody-recovery-worker"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sessions.Exec(workerAContext, `select set_config('app.portfolio_custody_recovery_worker_id','abandoned-worker',false)`); err != nil {
		t.Fatal(err)
	}
	if _, err := sessions.Exec(workerAContext, `update portfolio_custody_recovery_operations set status='leased',attempts=1,lease_owner='abandoned-worker',lease_expires_at=now()-interval '1 minute' where id=$1::uuid`, opA); err != nil {
		t.Fatalf("seed expired recovery lease: %v", err)
	}
	releaseWorkerA()

	worker := NewPortfolioCustodyRecoveryWorker(sessions, nil, nil, time.Millisecond, 3)
	claim, err := worker.claim(ctx)
	if err != nil || claim == nil || claim.ID != opA || claim.Attempts != 2 {
		t.Fatalf("claim queued recovery operation: claim=%#v err=%v", claim, err)
	}
	if err := worker.fail(ctx, claim, errors.New("temporary S3 outage")); err != nil {
		t.Fatalf("record retryable recovery failure: %v", err)
	}
	claim, err = worker.claim(ctx)
	if err != nil || claim == nil || claim.ID != opA || claim.Attempts != 3 {
		t.Fatalf("reclaim retryable recovery operation: claim=%#v err=%v", claim, err)
	}
	if err := worker.fail(ctx, claim, errors.New("temporary S3 outage")); err != nil {
		t.Fatalf("deadletter exhausted recovery operation: %v", err)
	}
	var status, failure string
	if err := sessions.QueryRow(ctxA, `select status,last_error_code from portfolio_custody_recovery_operations where id=$1::uuid`, opA).Scan(&status, &failure); err != nil || status != "deadletter" || failure != "recovery_failed" {
		t.Fatalf("retry/deadletter state status=%q code=%q err=%v", status, failure, err)
	}
	if err := sessions.QueryRow(ctxA, `select count(*) from portfolio_custody_recovery_deadletters where operation_id=$1::uuid`, opA).Scan(&visible); err != nil || visible != 1 {
		t.Fatalf("deadletter row count=%d err=%v", visible, err)
	}
	if _, err := sessions.Exec(ctxA, `update portfolio_custody_recovery_operations set last_error_code='rewritten' where id=$1::uuid`, opA); err == nil || !strings.Contains(err.Error(), "terminal portfolio custody recovery is immutable") {
		t.Fatalf("terminal recovery mutation must be rejected, got %v", err)
	}
	if _, err := sessions.Exec(ctxA, `delete from portfolio_custody_recovery_attempts where operation_id=$1::uuid`, opA); err == nil || !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("recovery attempt deletion must be rejected, got %v", err)
	}
	if _, err := sessions.Exec(ctxA, `delete from portfolio_custody_recovery_deadletters where operation_id=$1::uuid`, opA); err == nil || !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("recovery deadletter deletion must be rejected, got %v", err)
	}

	var corruptOperation string
	corruptCtx, releaseCorrupt, err := appdb.AcquireRequestConn(ctx, admin, appdb.SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: "recovery-privileged-fixture", IsSuperAdmin: true})
	if err != nil {
		t.Fatalf("bind privileged corruption context: %v", err)
	}
	defer releaseCorrupt()
	if err := appdb.NewSessionPool(admin).QueryRow(corruptCtx, `insert into portfolio_custody_recovery_operations(intent_id,tenant_code,institution_id,portfolio_id,requested_by_subject,disposition,reason,title,original_file_name,document_date,expected_fingerprint) values($1::uuid,'tenant-egueducation','inst-001',$2::uuid,'archive-integration-test','institution_archive_only','privileged fixture corruption','Recovered custody evidence','evidence.pdf','2026-09-12',$3) returning id::text`, intentA.intentID, portfolioA, strings.Repeat("b", 64)).Scan(&corruptOperation); err != nil {
		t.Fatalf("seed privileged fingerprint-mismatch fixture: %v", err)
	}
	claim, err = worker.claim(ctx)
	if err != nil || claim == nil || claim.ID != corruptOperation {
		t.Fatalf("claim fingerprint mismatch operation: claim=%#v err=%v", claim, err)
	}
	if err := worker.process(ctxA, claim); err == nil || !strings.Contains(err.Error(), "recovery_operation_fingerprint_mismatch") {
		t.Fatalf("worker must reject operation fingerprint mismatch before storage work, got %v", err)
	}
	if err := worker.fail(ctx, claim, errors.New("recovery_operation_fingerprint_mismatch")); err != nil {
		t.Fatalf("record fingerprint mismatch block: %v", err)
	}
	if err := sessions.QueryRow(ctxA, `select status,last_error_code from portfolio_custody_recovery_operations where id=$1::uuid`, corruptOperation).Scan(&status, &failure); err != nil || status != "blocked" || failure != "recovery_operation_fingerprint_mismatch" {
		t.Fatalf("fingerprint mismatch must block operation status=%q code=%q err=%v", status, failure, err)
	}
}

func insertRecoveryOperation(t *testing.T, ctx context.Context, sessions *appdb.SessionPool, intent custodyIntentFixture, tenant, institution, portfolio, disposition string) string {
	t.Helper()
	var id string
	if err := sessions.QueryRow(ctx, `insert into portfolio_custody_recovery_operations(intent_id,tenant_code,institution_id,portfolio_id,requested_by_subject,disposition,reason,title,original_file_name,document_date,expected_fingerprint) values($1::uuid,$2,$3,$4::uuid,'archive-integration-test',$5,'integration recovery','Recovered custody evidence','evidence.pdf','2026-09-12',$6) returning id::text`, intent.intentID, tenant, institution, portfolio, disposition, intent.expectedFingerprint).Scan(&id); err != nil {
		t.Fatalf("insert %s recovery operation: %v", institution, err)
	}
	return id
}

func seedRecoveryOperationActor(t *testing.T, ctx context.Context, admin *pgxpool.Pool) {
	t.Helper()
	var actorID string
	if err := admin.QueryRow(ctx, `insert into app_users(sub,name,email,locale,status) values('archive-integration-test','Archive Integration Test','archive-integration-test@example.test','ro','active') returning id::text`).Scan(&actorID); err != nil {
		t.Fatalf("seed recovery operation actor: %v", err)
	}
	for _, scope := range []struct{ tenant, org string }{{"tenant-egueducation", "unit-root"}, {"tenant-balotesti", "unit-balotesti-root"}} {
		if _, err := admin.Exec(ctx, `insert into app_memberships(user_id,tenant_code,position_code,org_unit_code,organization_name,is_primary,active,start_date) values($1::uuid,$2,'profesor',$3,'Recovery Integration',true,true,current_date)`, actorID, scope.tenant, scope.org); err != nil {
			t.Fatalf("seed recovery operation membership for %s: %v", scope.tenant, err)
		}
		for _, permission := range []string{"earchiva.manage", "education.portfolios.custody.manage"} {
			if _, err := admin.Exec(ctx, `insert into app_user_permissions(user_id,tenant_code,permission_code) values($1::uuid,$2,$3)`, actorID, scope.tenant, permission); err != nil {
				t.Fatalf("grant recovery operation %s for %s: %v", permission, scope.tenant, err)
			}
		}
	}
}
