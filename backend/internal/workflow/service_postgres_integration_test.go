//go:build integration

package workflow

// This test is deliberately a PostgreSQL test rather than a mocked handler
// test.  The workflow service relies on RLS session variables, generated
// entity versions and the append-only audit table; mocks cannot prove those
// contracts.  CI supplies TEST_DATABASE_URL and each run creates a separate
// database plus a non-BYPASSRLS application role.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	appdb "github.com/eguilde/egueducation/internal/db"
)

func TestWorkflowPostgresHandlersAndTenantContractsIntegration(t *testing.T) {
	it := newWorkflowIntegrationDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	adminPool := openWorkflowIntegrationPool(t, ctx, it.databaseConfig)
	defer adminPool.Close()
	if err := appdb.Migrate(ctx, adminPool); err != nil {
		t.Fatalf("migrate disposable workflow database: %v", err)
	}
	if err := appdb.ValidateSchemaContract(ctx, adminPool); err != nil {
		t.Fatalf("validate workflow schema contract: %v", err)
	}
	grantWorkflowIntegrationAccess(t, ctx, adminPool, it.roleName)

	fixture := seedWorkflowFixture(t, ctx, adminPool)
	service := NewService(appdb.NewSessionPool(it.readerPool))
	ctxA, releaseA := workflowTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, "workflow-it-a")
	defer releaseA()
	ctxB, releaseB := workflowTenantContext(t, ctx, it.readerPool, fixture.tenantB, fixture.institutionB, "workflow-it-b")
	defer releaseB()

	// Definitions are global catalog data, while all instances are tenant-bound.
	definitionsRecorder := httptest.NewRecorder()
	service.ListDefinitions(definitionsRecorder, httptest.NewRequest(http.MethodGet, "/workflow/definitions", nil).WithContext(ctxA))
	if definitionsRecorder.Code != http.StatusOK {
		t.Fatalf("definitions status = %d: %s", definitionsRecorder.Code, definitionsRecorder.Body.String())
	}
	var definitions []Definition
	if err := json.Unmarshal(definitionsRecorder.Body.Bytes(), &definitions); err != nil || len(definitions) == 0 {
		t.Fatalf("decode global definitions: len=%d err=%v body=%s", len(definitions), err, definitionsRecorder.Body.String())
	}

	// PostgreSQL performs the actual filter, allow-listed sort and pagination
	// contract used by ListTasks.  Tenant B's similarly named task must not
	// contribute to either the count or the page.
	where, args := buildTaskFilters(fixture.institutionA, map[string]string{"title": "integration", "status": "new"})
	var total int
	if err := service.pool.QueryRow(ctxA, `select count(*) from workflow_instances wi `+where, args...).Scan(&total); err != nil {
		t.Fatalf("filtered workflow count: %v", err)
	}
	if total != 3 {
		t.Fatalf("filtered tenant-A workflow count = %d, want 3", total)
	}
	rows, err := service.pool.Query(ctxA, `select title from workflow_instances wi `+where+` order by `+sortColumn("title")+` desc limit $4 offset $5`, append(args, 2, 0)...)
	if err != nil {
		t.Fatalf("paged/sorted workflow query: %v", err)
	}
	var page []string
	for rows.Next() {
		var title string
		if err := rows.Scan(&title); err != nil {
			t.Fatalf("scan workflow page: %v", err)
		}
		page = append(page, title)
	}
	rows.Close()
	if len(page) != 2 || page[0] != "Integration task 3" || page[1] != "Integration task 2" {
		t.Fatalf("workflow server page = %#v, want descending first page", page)
	}

	// Dashboard readiness is evaluated from real dossier requirement rows, not
	// a client-provided flag.  The source-less task is ready; the linked task is
	// intentionally blocked because it has no supporting document.
	ready, blocked, err := service.readinessStats(ctxA, fixture.institutionA)
	if err != nil {
		t.Fatalf("workflow dashboard readiness: %v", err)
	}
	if ready < 1 || blocked < 1 {
		t.Fatalf("workflow readiness = ready:%d blocked:%d, want both categories", ready, blocked)
	}

	// TaskFilters reads through the restricted connection; RLS must hide the
	// tenant-B-only assignee from the public handler response.
	filtersRecorder := httptest.NewRecorder()
	service.TaskFilters(filtersRecorder, httptest.NewRequest(http.MethodGet, "/workflow/tasks/filters", nil).WithContext(ctxA))
	if filtersRecorder.Code != http.StatusOK {
		t.Fatalf("task filters status = %d: %s", filtersRecorder.Code, filtersRecorder.Body.String())
	}
	var filters FiltersResponse
	if err := json.Unmarshal(filtersRecorder.Body.Bytes(), &filters); err != nil {
		t.Fatalf("decode task filters: %v", err)
	}
	if contains(filters.Assignees, "Tenant B only") {
		t.Fatalf("RLS leaked tenant-B assignee through task filters: %#v", filters.Assignees)
	}

	// A valid transition writes an immutable version and tenant-scoped audit
	// event.  An invalid state/action is rejected without modifying that row.
	valid := transitionWorkflowRequest(ctxA, fixture.transitionTaskID, "start")
	validRecorder := httptest.NewRecorder()
	service.TransitionTask(validRecorder, valid)
	if validRecorder.Code != http.StatusOK {
		t.Fatalf("valid transition status = %d: %s", validRecorder.Code, validRecorder.Body.String())
	}
	var transitioned Task
	if err := json.Unmarshal(validRecorder.Body.Bytes(), &transitioned); err != nil {
		t.Fatalf("decode transitioned task: %v", err)
	}
	if transitioned.Status != "in_progress" || transitioned.CurrentStep != "Verificare și completare" {
		t.Fatalf("valid transition result = %#v", transitioned)
	}
	invalidRecorder := httptest.NewRecorder()
	service.TransitionTask(invalidRecorder, transitionWorkflowRequest(ctxA, fixture.transitionTaskID, "approve"))
	if invalidRecorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid transition status = %d: %s", invalidRecorder.Code, invalidRecorder.Body.String())
	}
	var versionCount, auditCount int
	if err := service.pool.QueryRow(ctxA, `select count(*) from app_entity_versions where entity_table='workflow_instances' and entity_id=$1::uuid`, fixture.transitionTaskID).Scan(&versionCount); err != nil {
		t.Fatalf("read workflow entity versions: %v", err)
	}
	if versionCount < 2 {
		t.Fatalf("workflow version history count = %d, want insert + valid update", versionCount)
	}
	if err := service.pool.QueryRow(ctxA, `select count(*) from app_audit_log where action='workflow.tasks.transition' and target_id=$1`, fixture.transitionTaskID).Scan(&auditCount); err != nil {
		t.Fatalf("read transition audit: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("workflow transition audit count = %d, want 1", auditCount)
	}

	// The same task ID is opaque across tenants: RLS turns it into the public
	// not-found response and no audit/version can be written by tenant B.
	foreignRecorder := httptest.NewRecorder()
	service.TransitionTask(foreignRecorder, transitionWorkflowRequest(ctxB, fixture.transitionTaskID, "archive"))
	if foreignRecorder.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant transition status = %d: %s", foreignRecorder.Code, foreignRecorder.Body.String())
	}
	var foreignVisible int
	if err := service.pool.QueryRow(ctxB, `select count(*) from workflow_instances where id=$1::uuid`, fixture.transitionTaskID).Scan(&foreignVisible); err != nil {
		t.Fatalf("verify cross-tenant task invisibility: %v", err)
	}
	if foreignVisible != 0 {
		t.Fatalf("tenant B can see %d tenant-A workflow task(s)", foreignVisible)
	}
}

func transitionWorkflowRequest(ctx context.Context, taskID, action string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/workflow/tasks/"+taskID+"/transition", strings.NewReader(`{"action":"`+action+`"}`)).WithContext(ctx)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("taskID", taskID)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

type workflowIntegrationDatabase struct {
	databaseConfig *pgxpool.Config
	readerPool     *pgxpool.Pool
	roleName       string
}

func newWorkflowIntegrationDatabase(t *testing.T) workflowIntegrationDatabase {
	t.Helper()
	baseURL := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if baseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set; workflow PostgreSQL integration test is intentionally skipped")
	}
	ctx := context.Background()
	baseConfig, err := pgx.ParseConfig(baseURL)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}
	admin, err := pgx.ConnectConfig(ctx, baseConfig)
	if err != nil {
		t.Fatalf("connect TEST_DATABASE_URL: %v", err)
	}
	databaseName := "workflow_it_" + strings.ReplaceAll(uuid.NewString()[:12], "-", "")
	roleName := "workflow_it_" + strings.ReplaceAll(uuid.NewString()[:12], "-", "")
	rolePassword := uuid.NewString()
	if _, err := admin.Exec(ctx, "create database "+quoteWorkflowIdentifier(databaseName)+" template template0"); err != nil {
		admin.Close(ctx)
		t.Fatalf("create disposable workflow database: %v", err)
	}
	if _, err := admin.Exec(ctx, "create role "+quoteWorkflowIdentifier(roleName)+" login nosuperuser nobypassrls password "+quoteWorkflowLiteral(rolePassword)); err != nil {
		_, _ = admin.Exec(ctx, "drop database "+quoteWorkflowIdentifier(databaseName))
		admin.Close(ctx)
		t.Fatalf("create restricted workflow role: %v", err)
	}
	admin.Close(ctx)
	targetConfig, err := pgxpool.ParseConfig(baseURL)
	if err != nil {
		t.Fatalf("parse workflow target config: %v", err)
	}
	targetConfig.ConnConfig.Database = databaseName
	readerConfig := targetConfig.Copy()
	readerConfig.ConnConfig.User = roleName
	readerConfig.ConnConfig.Password = rolePassword
	readerConfig.MaxConns = 2
	readerPool, err := pgxpool.NewWithConfig(ctx, readerConfig)
	if err != nil {
		t.Fatalf("open restricted workflow pool: %v", err)
	}
	t.Cleanup(func() {
		readerPool.Close()
		cleanup, cleanupErr := pgx.ConnectConfig(context.Background(), baseConfig)
		if cleanupErr != nil {
			t.Errorf("connect to remove workflow database: %v", cleanupErr)
			return
		}
		defer cleanup.Close(context.Background())
		if _, err := cleanup.Exec(context.Background(), "drop database if exists "+quoteWorkflowIdentifier(databaseName)+" with (force)"); err != nil {
			t.Errorf("drop workflow database: %v", err)
		}
		if _, err := cleanup.Exec(context.Background(), "drop role if exists "+quoteWorkflowIdentifier(roleName)); err != nil {
			t.Errorf("drop workflow role: %v", err)
		}
	})
	return workflowIntegrationDatabase{targetConfig, readerPool, roleName}
}

func openWorkflowIntegrationPool(t *testing.T, ctx context.Context, cfg *pgxpool.Config) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.NewWithConfig(ctx, cfg.Copy())
	if err != nil {
		t.Fatalf("open workflow integration pool: %v", err)
	}
	return pool
}

func grantWorkflowIntegrationAccess(t *testing.T, ctx context.Context, pool *pgxpool.Pool, roleName string) {
	t.Helper()
	for _, statement := range []string{"grant usage on schema public to " + quoteWorkflowIdentifier(roleName), "grant select, insert, update, delete on all tables in schema public to " + quoteWorkflowIdentifier(roleName), "grant usage, select on all sequences in schema public to " + quoteWorkflowIdentifier(roleName), "grant execute on all functions in schema public to " + quoteWorkflowIdentifier(roleName)} {
		if _, err := pool.Exec(ctx, statement); err != nil {
			t.Fatalf("grant workflow integration access: %v", err)
		}
	}
}

type workflowFixture struct{ tenantA, institutionA, tenantB, institutionB, transitionTaskID string }

func seedWorkflowFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool) workflowFixture {
	t.Helper()
	const tenantA, institutionA = "tenant-egueducation", "inst-001"
	const tenantB, institutionB = "tenant-balotesti", "inst-balotesti"
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin workflow seed: %v", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `select set_config('app.tenant_id',$1,true), set_config('app.institution_id',$2,true), set_config('app.is_super_admin','true',true), set_config('app.actor_subject','workflow-seed',true)`, tenantA, institutionA); err != nil {
		t.Fatalf("bind workflow seed session: %v", err)
	}
	transitionID := uuid.NewString()
	for i, title := range []string{"Integration task 1", "Integration task 2", "Integration task 3"} {
		id := uuid.NewString()
		if i == 0 {
			id = transitionID
		}
		if _, err := tx.Exec(ctx, `insert into workflow_instances(id, definition_code, title, status, priority, assigned_to, current_step, institution_id, source_module) values ($1::uuid,'incoming-document',$2,'new','high','Tenant A only','Înregistrare',$3,'registratura')`, id, title, institutionA); err != nil {
			t.Fatalf("seed tenant-A workflow task: %v", err)
		}
	}
	// A source record forces dossier requirements; without document links this
	// record must be reported blocked by dashboard readiness.
	if _, err := tx.Exec(ctx, `insert into workflow_instances(id, definition_code, title, status, priority, assigned_to, current_step, institution_id, source_module, source_record_id) values ($1::uuid,'incoming-document','Blocked dossier task','in_progress','medium','Tenant A only','Verificare',$2,'education.portfolios',$3::uuid)`, uuid.NewString(), institutionA, uuid.NewString()); err != nil {
		t.Fatalf("seed blocked dossier task: %v", err)
	}
	if _, err := tx.Exec(ctx, `select set_config('app.tenant_id',$1,true), set_config('app.institution_id',$2,true)`, tenantB, institutionB); err != nil {
		t.Fatalf("bind tenant-B workflow seed: %v", err)
	}
	if _, err := tx.Exec(ctx, `insert into workflow_instances(definition_code,title,status,priority,assigned_to,current_step,institution_id,source_module) values ('incoming-document','Integration tenant B task','new','high','Tenant B only','Înregistrare',$1,'registratura')`, institutionB); err != nil {
		t.Fatalf("seed tenant-B workflow task: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit workflow seed: %v", err)
	}
	return workflowFixture{tenantA, institutionA, tenantB, institutionB, transitionID}
}

func workflowTenantContext(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, institutionID, actor string) (context.Context, func()) {
	t.Helper()
	bound, release, err := appdb.AcquireRequestConn(ctx, pool, appdb.SessionConfig{TenantID: tenantID, InstitutionID: institutionID, ActorSubject: actor})
	if err != nil {
		t.Fatalf("bind workflow tenant session: %v", err)
	}
	return bound, release
}
func quoteWorkflowIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}
func quoteWorkflowLiteral(value string) string {
	return `'` + strings.ReplaceAll(value, `'`, `''`) + `'`
}
