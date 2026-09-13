//go:build integration

package education

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eguilde/egueducation/internal/db"
	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPortfolioLifecycleOperationsListsWithinReadBoundaryAndAppliesFiltersPagingAndUTCDate(t *testing.T) {
	it := newGovernanceIntegrationDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	adminPool := openGovernanceIntegrationPool(t, ctx, it.databaseConfig)
	defer adminPool.Close()
	if err := db.Migrate(ctx, adminPool); err != nil {
		t.Fatalf("migrate disposable lifecycle-list database: %v", err)
	}
	grantGovernanceIntegrationAccess(t, ctx, adminPool, it.roleName)
	fixture := seedGovernanceAuthorizationFixture(t, ctx, adminPool)
	grantPortfolioLifecycleListReadPermissions(t, ctx, adminPool, fixture)
	archiveDocumentIDs := []string{
		seedGovernancePortfolioArchiveAttachments(t, ctx, adminPool, fixture.institutionA, fixture.memberUserID),
		seedGovernancePortfolioArchiveAttachments(t, ctx, adminPool, fixture.institutionA, fixture.memberUserID),
		seedGovernancePortfolioArchiveAttachments(t, ctx, adminPool, fixture.institutionA, fixture.memberUserID),
	}
	operations := seedPortfolioLifecycleListOperations(t, ctx, adminPool, fixture, archiveDocumentIDs)
	service := NewService(db.NewSessionPool(it.readerPool))
	typeAscendingIDs := []string{operations.pendingLegalHold, operations.blockedLegalHold}
	if typeAscendingIDs[0] > typeAscendingIDs[1] {
		typeAscendingIDs[0], typeAscendingIDs[1] = typeAscendingIDs[1], typeAscendingIDs[0]
	}
	typeAscendingIDs = append([]string{operations.cessation}, typeAscendingIDs...)

	ownerCtx, ownerRelease := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	defer ownerRelease()
	for _, tc := range []struct {
		name          string
		query         string
		wantTotal     int
		wantPage      int
		wantIDs       []string
		wantRequested string
	}{
		{name: "default newest first", wantTotal: 3, wantPage: 1, wantIDs: []string{operations.blockedLegalHold, operations.pendingLegalHold, operations.cessation}},
		{name: "status filter", query: "filter.status=pending", wantTotal: 1, wantPage: 1, wantIDs: []string{operations.pendingLegalHold}},
		{name: "type filter", query: "filter.type=cessation_retention", wantTotal: 1, wantPage: 1, wantIDs: []string{operations.cessation}},
		{name: "status sort", query: "sort=status&direction=asc", wantTotal: 3, wantPage: 1, wantIDs: []string{operations.blockedLegalHold, operations.cessation, operations.pendingLegalHold}},
		{name: "type sort has stable identifier tie break", query: "sort=type&direction=asc", wantTotal: 3, wantPage: 1, wantIDs: typeAscendingIDs},
		{name: "UTC date filter", query: "filter.requested_at=2026-03-01&sort=requested_at&direction=asc", wantTotal: 2, wantPage: 1, wantIDs: []string{operations.cessation, operations.pendingLegalHold}, wantRequested: "2026-03-01T00:30:00.000Z"},
		{name: "ascending requested timestamp second page", query: "sort=requested_at&direction=asc&page=2&pageSize=1", wantTotal: 3, wantPage: 2, wantIDs: []string{operations.pendingLegalHold}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := portfolioLifecycleListRequest(ownerCtx, fixture, fixture.portfolioID, tc.query)
			service.PortfolioLifecycleOperations(recorder, request)
			if recorder.Code != http.StatusOK {
				t.Fatalf("list lifecycle operations: status=%d body=%s", recorder.Code, recorder.Body.String())
			}
			var page httpx.PageResponse[PortfolioLifecycleOperation]
			if err := json.Unmarshal(recorder.Body.Bytes(), &page); err != nil {
				t.Fatalf("decode lifecycle operation page: %v", err)
			}
			if page.Total != tc.wantTotal || page.Page != tc.wantPage || len(page.Items) != len(tc.wantIDs) {
				t.Fatalf("page metadata/items total=%d page=%d items=%#v, want total=%d page=%d ids=%v", page.Total, page.Page, page.Items, tc.wantTotal, tc.wantPage, tc.wantIDs)
			}
			for index, wantID := range tc.wantIDs {
				if page.Items[index].ID != wantID {
					t.Fatalf("item %d id=%q, want %q; page=%#v", index, page.Items[index].ID, wantID, page.Items)
				}
			}
			if tc.wantRequested != "" && page.Items[0].RequestedAt != tc.wantRequested {
				t.Fatalf("requested_at=%q, want UTC serialization %q", page.Items[0].RequestedAt, tc.wantRequested)
			}
			if tc.name == "UTC date filter" && (page.Items[0].TotalVersions != 2 || page.Items[0].CompletedVersions != 1 || page.Items[0].BlockedVersions != 1) {
				t.Fatalf("cessation transition aggregates=%#v, want total=2 completed=1 blocked=1", page.Items[0])
			}
			if tc.name == "status filter" && (page.Items[0].TotalVersions != 1 || page.Items[0].CompletedVersions != 1 || page.Items[0].BlockedVersions != 0) {
				t.Fatalf("pending transition aggregates=%#v, want total=1 completed=1 blocked=0", page.Items[0])
			}
		})
	}
	ownerRelease()

	foreignSubject := subjectForUser(t, ctx, adminPool, fixture.foreignMemberUserID)
	foreignCtx, foreignRelease := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, foreignSubject)
	defer foreignRelease()
	nonOwner := httptest.NewRecorder()
	foreignFixture := fixture
	foreignFixture.memberSubject = foreignSubject
	service.PortfolioLifecycleOperations(nonOwner, portfolioLifecycleListRequest(foreignCtx, foreignFixture, fixture.portfolioID, ""))
	assertHandlerCode(t, nonOwner, http.StatusForbidden, "education_portfolio_access_denied")
	foreignRelease()

	crossTenantCtx, crossTenantRelease := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantB, fixture.institutionB, fixture.memberSubject)
	defer crossTenantRelease()
	crossTenant := httptest.NewRecorder()
	crossTenantFixture := fixture
	crossTenantFixture.tenantA = fixture.tenantB
	crossTenantFixture.institutionA = fixture.institutionB
	service.PortfolioLifecycleOperations(crossTenant, portfolioLifecycleListRequest(crossTenantCtx, crossTenantFixture, fixture.portfolioID, ""))
	assertHandlerCode(t, crossTenant, http.StatusForbidden, "education_portfolio_access_denied")
	crossTenantRelease()

	grantPortfolioLifecycleListSchoolReadPermissionForTenant(t, ctx, adminPool, fixture.memberUserID, fixture.tenantB, fixture.institutionB)
	crossTenantSchoolReaderCtx, crossTenantSchoolReaderRelease := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantB, fixture.institutionB, fixture.memberSubject)
	defer crossTenantSchoolReaderRelease()
	crossTenantSchoolReader := httptest.NewRecorder()
	service.PortfolioLifecycleOperations(crossTenantSchoolReader, portfolioLifecycleListRequest(crossTenantSchoolReaderCtx, crossTenantFixture, fixture.portfolioID, ""))
	assertHandlerCode(t, crossTenantSchoolReader, http.StatusNotFound, "education_portfolio_not_found")
	crossTenantSchoolReaderRelease()

	grantPortfolioLifecycleListSchoolReadPermission(t, ctx, adminPool, fixture)
	privilegedCtx, privilegedRelease := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	defer privilegedRelease()
	missing := httptest.NewRecorder()
	service.PortfolioLifecycleOperations(missing, portfolioLifecycleListRequest(privilegedCtx, fixture, uuid.NewString(), ""))
	assertHandlerCode(t, missing, http.StatusNotFound, "education_portfolio_not_found")
}

func TestPortfolioLifecycleOperationsRejectsMalformedAndHugePageQueries(t *testing.T) {
	it := newGovernanceIntegrationDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	adminPool := openGovernanceIntegrationPool(t, ctx, it.databaseConfig)
	defer adminPool.Close()
	if err := db.Migrate(ctx, adminPool); err != nil {
		t.Fatalf("migrate disposable lifecycle-list query database: %v", err)
	}
	grantGovernanceIntegrationAccess(t, ctx, adminPool, it.roleName)
	fixture := seedGovernanceAuthorizationFixture(t, ctx, adminPool)
	grantPortfolioLifecycleListSchoolReadPermission(t, ctx, adminPool, fixture)
	service := NewService(db.NewSessionPool(it.readerPool))
	requestCtx, release := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	defer release()

	for _, tc := range []struct {
		name  string
		query string
	}{
		{name: "unknown sort", query: "sort=operation_type"},
		{name: "invalid direction", query: "direction=sideways"},
		{name: "invalid status filter", query: "filter.status=unknown"},
		{name: "invalid type filter", query: "filter.type=delete"},
		{name: "invalid UTC date filter", query: "filter.requested_at=2026-02-29"},
		{name: "non-numeric page", query: "page=not-a-number"},
		{name: "zero page", query: "page=0"},
		{name: "huge page", query: "page=1000001"},
		{name: "space padded page", query: "page=%202%20"},
		{name: "space padded sort", query: "sort=%20status%20"},
		{name: "space padded direction", query: "direction=%20desc%20"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			service.PortfolioLifecycleOperations(recorder, portfolioLifecycleListRequest(requestCtx, fixture, fixture.portfolioID, tc.query))
			assertHandlerCode(t, recorder, http.StatusBadRequest, "invalid_portfolio_lifecycle_operation")
		})
	}
}

type portfolioLifecycleListFixture struct {
	cessation, pendingLegalHold, blockedLegalHold string
}

func seedPortfolioLifecycleListOperations(t *testing.T, ctx context.Context, pool *pgxpool.Pool, fixture governanceAuthorizationFixture, archiveDocumentIDs []string) portfolioLifecycleListFixture {
	t.Helper()
	if len(archiveDocumentIDs) != 3 {
		t.Fatalf("lifecycle list fixture archive documents=%d, want 3", len(archiveDocumentIDs))
	}
	operations := portfolioLifecycleListFixture{cessation: uuid.NewString(), pendingLegalHold: uuid.NewString(), blockedLegalHold: uuid.NewString()}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin lifecycle list fixture: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, `select set_config('app.is_super_admin','true',true), set_config('app.tenant_id',$1,true), set_config('app.institution_id',$2,true)`, fixture.tenantA, fixture.institutionA); err != nil {
		t.Fatalf("bind lifecycle list fixture session: %v", err)
	}
	for _, operation := range []struct {
		id, operationType, status, requestedAt string
		requestedHold                          any
	}{
		{id: operations.cessation, operationType: "cessation_retention", status: "completed", requestedAt: "2026-03-01T00:30:00Z", requestedHold: nil},
		{id: operations.pendingLegalHold, operationType: "legal_hold_reconcile", status: "pending", requestedAt: "2026-03-01T01:30:00Z", requestedHold: true},
		{id: operations.blockedLegalHold, operationType: "legal_hold_reconcile", status: "blocked", requestedAt: "2026-03-02T00:00:00Z", requestedHold: false},
	} {
		if _, err := tx.Exec(ctx, `insert into education_portfolio_lifecycle_operations(
			id,tenant_code,institution_id,portfolio_id,operation_type,activity_ceased_on,retention_through,
			requested_legal_hold,reason,requested_by_subject,status,requested_at,completed_at,last_error)
			values($1::uuid,$2,$3,$4::uuid,$5,
				case when $5='cessation_retention' then '2026-02-28'::date else null end,
				case when $5='cessation_retention' then '2029-02-28'::date else null end,
				$6,'lifecycle list integration fixture',$7,$8,$9::timestamptz,
				case when $8='completed' then $9::timestamptz else null end,
				case when $8='blocked' then 'storage blocked' else '' end)`,
			operation.id, fixture.tenantA, fixture.institutionA, fixture.portfolioID, operation.operationType,
			operation.requestedHold, fixture.memberSubject, operation.status, operation.requestedAt); err != nil {
			t.Fatalf("seed %s lifecycle operation: %v", operation.id, err)
		}
	}
	for _, transition := range []struct {
		operationID, archiveDocumentID, status string
	}{
		{operationID: operations.cessation, archiveDocumentID: archiveDocumentIDs[0], status: "completed"},
		{operationID: operations.cessation, archiveDocumentID: archiveDocumentIDs[1], status: "blocked"},
		{operationID: operations.pendingLegalHold, archiveDocumentID: archiveDocumentIDs[2], status: "completed"},
	} {
		var archiveVersionID string
		if err := tx.QueryRow(ctx, `select id::text from archive_document_versions where institution_id=$1 and document_id=$2::uuid`, fixture.institutionA, transition.archiveDocumentID).Scan(&archiveVersionID); err != nil {
			t.Fatalf("load archive version for lifecycle transition fixture: %v", err)
		}
		if _, err := tx.Exec(ctx, `insert into education_portfolio_storage_transitions(
			operation_id,tenant_code,institution_id,portfolio_id,archive_document_id,archive_version_id,
			source_bucket,source_object_key,source_object_version_id,source_object_etag,source_sha256,source_size_bytes,
			status,completed_at,last_error)
			values($1::uuid,$2,$3,$4::uuid,$5::uuid,$6::uuid,'archive','evidence.pdf','fixture-version-1',
				'fixture-etag-1',$7,3,$8,case when $8='completed' then now() else null end,
				case when $8='blocked' then 'storage_reconciliation_failed' else '' end)`,
			transition.operationID, fixture.tenantA, fixture.institutionA, fixture.portfolioID, transition.archiveDocumentID,
			archiveVersionID, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", transition.status); err != nil {
			t.Fatalf("seed lifecycle transition fixture: %v", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit lifecycle list fixture: %v", err)
	}
	return operations
}

func grantPortfolioLifecycleListReadPermissions(t *testing.T, ctx context.Context, pool *pgxpool.Pool, fixture governanceAuthorizationFixture) {
	t.Helper()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin lifecycle list read permissions: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, `select set_config('app.is_super_admin','true',true), set_config('app.tenant_id',$1,true), set_config('app.institution_id',$2,true)`, fixture.tenantA, fixture.institutionA); err != nil {
		t.Fatalf("bind lifecycle list permission session: %v", err)
	}
	if _, err := tx.Exec(ctx, `insert into app_user_permissions(user_id,permission_code,tenant_code) values
		($1::uuid,'education.portfolios.read_own',$2),
		($3::uuid,'education.portfolios.read_own',$2)
		on conflict do nothing`, fixture.memberUserID, fixture.tenantA, fixture.foreignMemberUserID); err != nil {
		t.Fatalf("grant lifecycle list own-read permissions: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit lifecycle list read permissions: %v", err)
	}
}

func grantPortfolioLifecycleListSchoolReadPermission(t *testing.T, ctx context.Context, pool *pgxpool.Pool, fixture governanceAuthorizationFixture) {
	t.Helper()
	grantPortfolioLifecycleListSchoolReadPermissionForTenant(t, ctx, pool, fixture.memberUserID, fixture.tenantA, fixture.institutionA)
}

func grantPortfolioLifecycleListSchoolReadPermissionForTenant(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID, tenantID, institutionID string) {
	t.Helper()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin lifecycle list school-read permission: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, `select set_config('app.is_super_admin','true',true), set_config('app.tenant_id',$1,true), set_config('app.institution_id',$2,true)`, tenantID, institutionID); err != nil {
		t.Fatalf("bind lifecycle list school-read permission session: %v", err)
	}
	if _, err := tx.Exec(ctx, `insert into app_user_permissions(user_id,permission_code,tenant_code)
		values($1::uuid,'education.portfolios.school.read',$2) on conflict do nothing`, userID, tenantID); err != nil {
		t.Fatalf("grant lifecycle list school-read permission: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit lifecycle list school-read permission: %v", err)
	}
}

func portfolioLifecycleListRequest(ctx context.Context, fixture governanceAuthorizationFixture, recordID, query string) *http.Request {
	request := legalWorkflowRequest(ctx, fixture, http.MethodGet, "", map[string]string{"recordID": recordID})
	request.URL.RawQuery = query
	return request
}
