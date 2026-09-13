//go:build integration

package earchiva

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eguilde/egueducation/internal/auth"
	appdb "github.com/eguilde/egueducation/internal/db"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestPortfolioCustodyRecoveryHTTPCurrentRBACReplayAndListIntegration(t *testing.T) {
	it := newArchiveIntegrationDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	admin := openArchiveIntegrationPool(t, ctx, it.databaseConfig)
	defer admin.Close()
	if err := appdb.Migrate(ctx, admin); err != nil {
		t.Fatalf("migrate recovery HTTP database: %v", err)
	}
	grantArchiveIntegrationAccess(t, ctx, admin)
	sessions := appdb.NewSessionPool(it.readerPool)
	adminSessions := appdb.NewSessionPool(admin)
	adminCtx, releaseAdmin := retentionScopedDB(t, ctx, admin, "recovery-http-fixture", true)
	defer releaseAdmin()

	portfolioID := seedCustodyIntentPortfolio(t, ctx, admin, "inst-001")
	tenantCtx, releaseTenant := archiveTenantContext(t, ctx, it.readerPool, "tenant-egueducation", "inst-001")
	defer releaseTenant()
	intent := seedCustodyUploadIntent(t, tenantCtx, sessions, "tenant-egueducation", "inst-001", portfolioID)
	markCustodyIntentStored(t, tenantCtx, sessions, intent)

	both := seedRecoveryHTTPActor(t, adminCtx, adminSessions, "both", "earchiva.manage", "education.portfolios.custody.manage")
	onlyArchive := seedRecoveryHTTPActor(t, adminCtx, adminSessions, "archive", "earchiva.manage")
	onlyPortfolio := seedRecoveryHTTPActor(t, adminCtx, adminSessions, "portfolio", "education.portfolios.custody.manage")

	service := NewRecoveryAdmin(sessions)
	router := chi.NewRouter()
	router.Get("/intents", service.List)
	router.Post("/intents/{intentID}/reconcile", service.Reconcile)
	router.Get("/intents/{intentID}/operations/{operationID}", service.GetOperation)
	send := func(actor archiveRetentionHTTPActor, method, path string, payload any) *httptest.ResponseRecorder {
		requestCtx, release := retentionScopedDB(t, ctx, it.readerPool, actor.Subject, false)
		defer release()
		requestCtx = auth.WithSessionContextForIntegration(requestCtx, auth.SessionContext{
			TenantCode: "tenant-egueducation", InstitutionID: "inst-001",
			User: auth.SessionUser{ID: actor.ID, Sub: actor.Subject},
		})
		var body bytes.Buffer
		if payload != nil {
			if err := json.NewEncoder(&body).Encode(payload); err != nil {
				t.Fatal(err)
			}
		}
		request := httptest.NewRequest(method, path, &body).WithContext(requestCtx)
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		return response
	}

	for name, actor := range map[string]archiveRetentionHTTPActor{"only_archive": onlyArchive, "only_portfolio": onlyPortfolio} {
		t.Run("requires_both_"+name, func(t *testing.T) {
			response := send(actor, http.MethodGet, "/intents", nil)
			if response.Code != http.StatusForbidden {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}

	listed := send(both, http.MethodGet, "/intents?page=1&pageSize=1&status=stored&intent_id="+intent.intentID+"&portfolio_id="+portfolioID+"&sort=intent_id&direction=asc", nil)
	if listed.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listed.Code, listed.Body.String())
	}
	var page PortfolioCustodyRecoveryPage
	if err := json.Unmarshal(listed.Body.Bytes(), &page); err != nil || page.Total != 1 || len(page.Items) != 1 || page.Items[0].IntentID != intent.intentID {
		t.Fatalf("list page=%#v err=%v", page, err)
	}

	payload := ReconcilePortfolioCustodyRequest{
		Reason:      "Recover exact held version after interrupted response",
		Disposition: "institution_archive_only", Title: "Recovered custody evidence", OriginalFileName: "evidence.pdf",
	}
	created := send(both, http.MethodPost, "/intents/"+intent.intentID+"/reconcile", payload)
	if created.Code != http.StatusAccepted {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	var operation PortfolioCustodyRecoveryOperation
	if err := json.Unmarshal(created.Body.Bytes(), &operation); err != nil || operation.ID == "" || operation.Status != "queued" {
		t.Fatalf("created operation=%#v err=%v", operation, err)
	}
	replay := send(both, http.MethodPost, "/intents/"+intent.intentID+"/reconcile", payload)
	var replayed PortfolioCustodyRecoveryOperation
	if replay.Code != http.StatusAccepted || json.Unmarshal(replay.Body.Bytes(), &replayed) != nil || replayed.ID != operation.ID {
		t.Fatalf("replay status=%d operation=%#v body=%s", replay.Code, replayed, replay.Body.String())
	}
	conflicting := payload
	conflicting.Reason = "Different operator justification"
	if response := send(both, http.MethodPost, "/intents/"+intent.intentID+"/reconcile", conflicting); response.Code != http.StatusConflict {
		t.Fatalf("active conflict status=%d body=%s", response.Code, response.Body.String())
	}
	detail := send(both, http.MethodGet, "/intents/"+intent.intentID+"/operations/"+operation.ID, nil)
	if detail.Code != http.StatusOK {
		t.Fatalf("detail status=%d body=%s", detail.Code, detail.Body.String())
	}

	if _, err := adminSessions.Exec(adminCtx, `delete from app_user_permissions where tenant_code='tenant-egueducation' and user_id=$1::uuid and permission_code='earchiva.manage'`, both.ID); err != nil {
		t.Fatalf("revoke live archive permission: %v", err)
	}
	if response := send(both, http.MethodGet, "/intents", nil); response.Code != http.StatusForbidden {
		t.Fatalf("stale browser identity survived live DB revocation: status=%d body=%s", response.Code, response.Body.String())
	}
}

func seedRecoveryHTTPActor(t *testing.T, ctx context.Context, sessions *appdb.SessionPool, label string, permissions ...string) archiveRetentionHTTPActor {
	t.Helper()
	actor := archiveRetentionHTTPActor{ID: uuid.NewString(), Subject: "recovery-http-" + label + "-" + uuid.NewString()}
	if _, err := sessions.Exec(ctx, `insert into app_users(id,sub,name,email,locale,status) values($1::uuid,$2,$3,$4,'ro','active')`, actor.ID, actor.Subject, "Recovery HTTP "+label, uuid.NewString()+"@example.test"); err != nil {
		t.Fatalf("seed %s user: %v", label, err)
	}
	if _, err := sessions.Exec(ctx, `insert into app_memberships(user_id,tenant_code,position_code,org_unit_code,organization_name,is_primary,active,start_date) values($1::uuid,'tenant-egueducation','profesor','unit-root',$2,true,true,current_date)`, actor.ID, "Recovery HTTP "+label); err != nil {
		t.Fatalf("seed %s membership: %v", label, err)
	}
	for _, permission := range permissions {
		if _, err := sessions.Exec(ctx, `insert into app_user_permissions(tenant_code,user_id,permission_code) values('tenant-egueducation',$1::uuid,$2)`, actor.ID, permission); err != nil {
			t.Fatalf("grant %s permission %s: %v", label, permission, err)
		}
	}
	return actor
}
