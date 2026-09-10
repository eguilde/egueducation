//go:build integration

package education

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appdb "github.com/eguilde/egueducation/internal/db"
	"github.com/go-chi/chi/v5"
)

// TestSchoolReportsCompileAgainstMigratedSchema prevents report definitions
// from drifting to browser-owned or nonexistent database columns. The report
// service runs each catalog query against a freshly migrated database.
func TestSchoolReportsCompileAgainstMigratedSchema(t *testing.T) {
	it := newGovernanceIntegrationDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	admin := openGovernanceIntegrationPool(t, ctx, it.databaseConfig)
	defer admin.Close()
	if err := appdb.Migrate(ctx, admin); err != nil {
		t.Fatalf("migrate school report database: %v", err)
	}
	bound, release, err := appdb.AcquireRequestConn(ctx, admin, appdb.SessionConfig{
		TenantID:      "tenant-egueducation",
		InstitutionID: "inst-001",
		ActorSubject:  "school-report-contract-test",
		IsSuperAdmin:  true,
	})
	if err != nil {
		t.Fatalf("bind school report database session: %v", err)
	}
	defer release()
	service := NewService(appdb.NewSessionPool(admin))
	request := requestWithContext(bound, "tenant-egueducation", "inst-001", "school-report-contract-test")
	for code, spec := range schoolReportSpecs {
		rows, total, err := service.loadSchoolReport(request, spec, nil, spec.defaultSort, "asc", 5, 0)
		if err != nil {
			t.Errorf("report %q does not compile against migrated schema: %v", code, err)
			continue
		}
		if total < len(rows) {
			t.Errorf("report %q total %d is smaller than returned rows %d", code, total, len(rows))
		}
	}
}

func TestSchoolReportExportRequiresReadAndSensitiveExportPermissions(t *testing.T) {
	it := newGovernanceIntegrationDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	admin := openGovernanceIntegrationPool(t, ctx, it.databaseConfig)
	defer admin.Close()
	if err := appdb.Migrate(ctx, admin); err != nil {
		t.Fatalf("migrate school report RBAC database: %v", err)
	}
	grantGovernanceIntegrationAccess(t, ctx, admin, it.roleName)
	fixture := seedGovernanceAuthorizationFixture(t, ctx, admin)
	if _, err := admin.Exec(ctx, `insert into app_user_permissions(user_id,permission_code,tenant_code) values($1::uuid,'education.compliance.read',$2) on conflict do nothing`, fixture.memberUserID, fixture.tenantA); err != nil {
		t.Fatalf("grant report read permission: %v", err)
	}
	bound, release := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	defer release()
	service := NewService(appdb.NewSessionPool(it.readerPool))

	request := requestWithContext(bound, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	route := chi.NewRouteContext()
	route.URLParams.Add("reportCode", "publication-backlog")
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, route))
	readOnly := httptest.NewRecorder()
	service.SchoolReportCSV(readOnly, request)
	if readOnly.Code != http.StatusForbidden {
		t.Fatalf("read-only export status=%d body=%s, want 403", readOnly.Code, readOnly.Body.String())
	}

	if _, err := admin.Exec(ctx, `insert into app_user_permissions(user_id,permission_code,tenant_code) values($1::uuid,'education.reports.export_sensitive',$2) on conflict do nothing`, fixture.memberUserID, fixture.tenantA); err != nil {
		t.Fatalf("grant sensitive export permission: %v", err)
	}
	authorized := httptest.NewRecorder()
	service.SchoolReportCSV(authorized, request)
	if authorized.Code != http.StatusOK || authorized.Header().Get("Content-Type") != "text/csv; charset=utf-8" {
		t.Fatalf("authorized export status=%d type=%q body=%s", authorized.Code, authorized.Header().Get("Content-Type"), authorized.Body.String())
	}
}
