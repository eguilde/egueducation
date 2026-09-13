//go:build integration

package earchiva

import (
	"context"
	"strings"
	"testing"
	"time"

	appdb "github.com/eguilde/egueducation/internal/db"
	"github.com/google/uuid"
)

func TestPortfolioStorageLifecycleAggregatesCustodyRetentionAndLegalHoldsIntegration(t *testing.T) {
	it := newArchiveIntegrationDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	admin := openArchiveIntegrationPool(t, ctx, it.databaseConfig)
	defer admin.Close()
	if err := appdb.Migrate(ctx, admin); err != nil {
		t.Fatalf("migrate lifecycle integration database: %v", err)
	}
	grantArchiveIntegrationAccess(t, ctx, admin)

	portfolioA := seedCustodyIntentPortfolio(t, ctx, admin, "inst-001")
	portfolioB := seedCustodyIntentPortfolio(t, ctx, admin, "inst-001")
	sessions := appdb.NewSessionPool(it.readerPool)
	tenantCtx, release := archiveTenantContext(t, ctx, it.readerPool, "tenant-egueducation", "inst-001")
	defer release()
	workerCtx, workerRelease, err := appdb.AcquireRequestConn(ctx, it.readerPool, appdb.SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: "portfolio-storage-lifecycle-worker"})
	if err != nil {
		t.Fatalf("bind lifecycle worker session: %v", err)
	}
	defer workerRelease()
	intent := seedCustodyUploadIntent(t, tenantCtx, sessions, "tenant-egueducation", "inst-001", portfolioA)
	markCustodyIntentStored(t, tenantCtx, sessions, intent)
	seedCustodyArchiveDocument(t, tenantCtx, sessions, "inst-001", intent.documentID)
	if _, err := sessions.Exec(tenantCtx, custodyVersionInsertSQL, intent.versionID, "inst-001", intent.documentID,
		intent.bucket, intent.objectKey, strings.Repeat("a", 64), int64(3), intent.storageVersion, intent.etag, intent.intentID); err != nil {
		t.Fatalf("seed custody-held version: %v", err)
	}
	if _, err := sessions.Exec(tenantCtx, `update archive_documents set status='ready',current_version_no=1 where id=$1::uuid`, intent.documentID); err != nil {
		t.Fatalf("activate custody archive document: %v", err)
	}
	attachLifecycleVersion(t, tenantCtx, sessions, portfolioA, intent)
	attachLifecycleVersion(t, tenantCtx, sessions, portfolioB, intent)

	if _, err := sessions.Exec(tenantCtx, `update education_portfolios set activity_ceased_on='2023-10-01',activity_cessation_reason='cessation A' where id=$1::uuid`, portfolioA); err != nil {
		t.Fatalf("cease first portfolio: %v", err)
	}
	firstOperation := seedLifecycleTransition(t, tenantCtx, sessions, portfolioA, intent, "cessation_retention", nil)
	storage := &memoryPortfolioLifecycleStorage{bucket: intent.bucket, hold: true}
	worker := newPortfolioStorageLifecycleWorker(sessions, storage, nil, time.Millisecond, 3)
	processNextPortfolioLifecycleTransition(t, ctx, workerCtx, worker, firstOperation)
	if len(storage.requests) != 1 || !storage.requests[0].LegalHoldActive || storage.requests[0].RetentionUntil.IsZero() {
		t.Fatalf("active shared portfolio must preserve hold while applying retention: %#v", storage.requests)
	}

	if _, err := sessions.Exec(tenantCtx, `update education_portfolios set activity_ceased_on='2024-02-29',activity_cessation_reason='cessation B' where id=$1::uuid`, portfolioB); err != nil {
		t.Fatalf("cease shared portfolio: %v", err)
	}
	if _, err := sessions.Exec(tenantCtx, `update education_portfolios set legal_hold_active=true,legal_hold_reason='litigation',legal_hold_set_at=now(),legal_hold_set_by_subject='archive-integration-test' where id=$1::uuid`, portfolioA); err != nil {
		t.Fatalf("activate real legal hold: %v", err)
	}
	secondOperation := seedLifecycleTransition(t, tenantCtx, sessions, portfolioB, intent, "cessation_retention", nil)
	processNextPortfolioLifecycleTransition(t, ctx, workerCtx, worker, secondOperation)
	second := storage.requests[1]
	if !second.LegalHoldActive || !second.RetentionUntil.After(storage.requests[0].RetentionUntil) {
		t.Fatalf("real hold and maximum shared deadline not preserved: first=%#v second=%#v", storage.requests[0], second)
	}

	if _, err := sessions.Exec(tenantCtx, `update education_portfolios set legal_hold_active=false,legal_hold_reason='released',legal_hold_set_at=now(),legal_hold_set_by_subject='archive-integration-test' where id=$1::uuid`, portfolioA); err != nil {
		t.Fatalf("record explicit legal-hold release: %v", err)
	}
	releaseState := false
	thirdOperation := seedLifecycleTransition(t, tenantCtx, sessions, portfolioA, intent, "legal_hold_reconcile", &releaseState)
	processNextPortfolioLifecycleTransition(t, ctx, workerCtx, worker, thirdOperation)
	third := storage.requests[2]
	if !third.LegalHoldActive || third.RetentionUntil.Before(second.RetentionUntil) {
		t.Fatalf("pre-expiry retention disposition hold and maximum verified deadline must remain active: %#v", third)
	}

	var retention *time.Time
	var custody, legal, retentionDisposition bool
	if err := sessions.QueryRow(tenantCtx, `select retention_until,custody_hold_active,legal_hold_active,retention_disposition_hold_active from archive_document_versions where id=$1::uuid`, intent.versionID).Scan(&retention, &custody, &legal, &retentionDisposition); err != nil {
		t.Fatalf("read final archive lifecycle projection: %v", err)
	}
	if retention == nil || retention.Before(second.RetentionUntil) || custody || legal || !retentionDisposition {
		t.Fatalf("final lifecycle projection retention=%v custody=%v legal=%v retention_disposition=%v", retention, custody, legal, retentionDisposition)
	}

	portfolioC := seedCustodyIntentPortfolio(t, ctx, admin, "inst-001")
	_, err = sessions.Exec(tenantCtx, lifecycleAttachmentSQL, portfolioC, intent.documentID, intent.documentID, intent.versionID,
		intent.bucket, intent.objectKey, strings.Repeat("a", 64))
	if err == nil || !strings.Contains(err.Error(), "requires verified custody hold") {
		t.Fatalf("released version was attachable to a new active portfolio: %v", err)
	}
}

func attachLifecycleVersion(t *testing.T, ctx context.Context, sessions *appdb.SessionPool, portfolioID string, intent custodyIntentFixture) {
	t.Helper()
	if _, err := sessions.Exec(ctx, lifecycleAttachmentSQL, portfolioID, intent.documentID, intent.documentID, intent.versionID,
		intent.bucket, intent.objectKey, strings.Repeat("a", 64)); err != nil {
		t.Fatalf("attach shared lifecycle version: %v", err)
	}
}

const lifecycleAttachmentSQL = `insert into education_portfolio_documents(
		portfolio_id,section_code,component_code,document_title,description,school_year,subject_discipline,applicable_class,competencies,
		source_scope,evidence_type,issued_on,added_on,chronological_index,sensitive_data,authenticity_status,file_reference,institution_id,
		archive_document_id,archive_version_id,archive_version_no,archive_source_bucket,archive_source_object_key,archive_sha256,last_change_reason)
		values($1::uuid,'identificare_profesionala','structura_cadru','Lifecycle evidence','Immutable lifecycle evidence','2026-2027',
		'Test','Class',array['test'],'portofoliu','document',current_date,current_date,1,false,'declarat','archive://' || $2,'inst-001',
		$3::uuid,$4::uuid,1,$5,$6,$7,'lifecycle fixture')`

func seedLifecycleTransition(t *testing.T, ctx context.Context, sessions *appdb.SessionPool, portfolioID string, intent custodyIntentFixture, operationType string, requestedHold *bool) string {
	t.Helper()
	operationID := uuid.NewString()
	cessationDate := any(nil)
	retentionThrough := any(nil)
	if operationType == "cessation_retention" {
		if err := sessions.QueryRow(ctx, `select activity_ceased_on,retention_until from education_portfolios where id=$1::uuid`, portfolioID).Scan(&cessationDate, &retentionThrough); err != nil {
			t.Fatalf("load cessation state: %v", err)
		}
	}
	if _, err := sessions.Exec(ctx, `insert into education_portfolio_lifecycle_operations(
		id,tenant_code,institution_id,portfolio_id,operation_type,activity_ceased_on,retention_through,requested_legal_hold,reason,requested_by_subject)
		values($1::uuid,'tenant-egueducation','inst-001',$2::uuid,$3,$4,$5,$6,'integration lifecycle','archive-integration-test')`,
		operationID, portfolioID, operationType, cessationDate, retentionThrough, requestedHold); err != nil {
		t.Fatalf("seed lifecycle operation: %v", err)
	}
	if _, err := sessions.Exec(ctx, `insert into education_portfolio_storage_transitions(
		operation_id,tenant_code,institution_id,portfolio_id,archive_document_id,archive_version_id,source_bucket,source_object_key,
		source_object_version_id,source_object_etag,source_sha256,source_size_bytes,required_retention_until)
		values($1::uuid,'tenant-egueducation','inst-001',$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,$8,$9,3,
		case when $10='cessation_retention' then (select (retention_until+1)::timestamp at time zone 'UTC' from education_portfolios where id=$2::uuid) else null end)`,
		operationID, portfolioID, intent.documentID, intent.versionID, intent.bucket, intent.objectKey, intent.storageVersion, intent.etag, strings.Repeat("a", 64), operationType); err != nil {
		t.Fatalf("seed lifecycle transition: %v", err)
	}
	return operationID
}

func processNextPortfolioLifecycleTransition(t *testing.T, claimCtx, tenantCtx context.Context, worker *PortfolioStorageLifecycleWorker, operationID string) {
	t.Helper()
	transition, err := worker.claimTransition(claimCtx)
	if err != nil || transition == nil {
		t.Fatalf("claim lifecycle transition: %#v %v", transition, err)
	}
	if transition.OperationID != operationID {
		t.Fatalf("claimed operation %s, want %s", transition.OperationID, operationID)
	}
	if err := worker.processTransition(tenantCtx, transition); err != nil {
		t.Fatalf("process lifecycle transition: %v", err)
	}
}

type memoryPortfolioLifecycleStorage struct {
	bucket    string
	hold      bool
	retention time.Time
	requests  []PortfolioCustodyLifecycleRequest
}

func (s *memoryPortfolioLifecycleStorage) Enabled() bool  { return true }
func (s *memoryPortfolioLifecycleStorage) Bucket() string { return s.bucket }
func (s *memoryPortfolioLifecycleStorage) ReconcilePortfolioCustodyLifecycle(_ context.Context, request PortfolioCustodyLifecycleRequest) (PortfolioCustodyLifecycleState, error) {
	s.requests = append(s.requests, request)
	if request.RetentionUntil.After(s.retention) {
		s.retention = request.RetentionUntil
	}
	s.hold = request.LegalHoldActive
	return PortfolioCustodyLifecycleState{RetentionUntil: s.retention, LegalHoldActive: s.hold}, nil
}
