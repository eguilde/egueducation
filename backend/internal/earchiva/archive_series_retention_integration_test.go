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
	"github.com/eguilde/egueducation/internal/regulatorysource"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// This is deliberately a handler + real PostgreSQL test. Authentication is
// injected with WithSessionContextForIntegration (not OIDC); authorization is
// still checked against the request-scoped PostgreSQL session on every call.
func TestArchiveSeriesRetentionHTTPPostgresIntegration(t *testing.T) {
	it := newArchiveIntegrationDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	adminPool := openArchiveIntegrationPool(t, ctx, it.databaseConfig)
	t.Cleanup(adminPool.Close)
	if err := appdb.Migrate(ctx, adminPool); err != nil {
		t.Fatalf("migrate fresh archive retention database: %v", err)
	}
	grantArchiveIntegrationAccess(t, ctx, adminPool)

	adminSessions := appdb.NewSessionPool(adminPool)
	adminCtx, releaseAdmin := retentionScopedDB(t, ctx, adminPool, "retention-fixture-admin", true)
	defer releaseAdmin()
	proposer := seedArchiveRetentionActor(t, adminCtx, adminSessions, "director")
	approver := seedArchiveRetentionActor(t, adminCtx, adminSessions, "director")
	reader := seedArchiveRetentionActor(t, adminCtx, adminSessions, "arhivar")
	revoked := seedArchiveRetentionActor(t, adminCtx, adminSessions, "director")

	readerSessions := appdb.NewSessionPool(it.readerPool)
	fixtureCtx, releaseFixture := retentionScopedDB(t, ctx, it.readerPool, "retention-fixture-source", false)
	mainTaxonomy := seedArchiveRetentionTaxonomy(t, fixtureCtx, readerSessions, "main")
	otherTaxonomy := seedArchiveRetentionTaxonomy(t, fixtureCtx, readerSessions, "other")
	revokedTaxonomy := seedArchiveRetentionTaxonomy(t, fixtureCtx, readerSessions, "revoked")
	sourceID := createActivatedRetentionSource(t, ctx, it.readerPool, readerSessions, proposer, approver)
	releaseFixture()

	service := NewArchiveSeriesRetentionService(readerSessions)
	router := chi.NewRouter()
	router.Get("/earchiva/retention-rules", service.ListArchiveSeriesRetentionRules)
	router.Post("/earchiva/retention-rules", service.ProposeArchiveSeriesRetentionRule)
	router.Post("/earchiva/retention-rules/{ruleID}/approve", service.ApproveArchiveSeriesRetentionRule)
	router.Post("/earchiva/retention-rules/{ruleID}/retire", service.RetireArchiveSeriesRetentionRule)

	send := func(actor archiveRetentionHTTPActor, method, path, key string, payload any) *httptest.ResponseRecorder {
		t.Helper()
		requestCtx, release := retentionScopedDB(t, ctx, it.readerPool, actor.Subject, false)
		defer release()
		// These client-side permissions intentionally remain stale in the later
		// revocation assertion. The service must not trust them.
		requestCtx = auth.WithSessionContextForIntegration(requestCtx, auth.SessionContext{
			TenantCode: "tenant-egueducation", InstitutionID: "inst-001",
			User:        auth.SessionUser{ID: actor.ID, Sub: actor.Subject},
			Permissions: []string{"earchiva.retention.read", "earchiva.retention.manage", "earchiva.retention.approve"},
		})
		var body *bytes.Reader
		if payload == nil {
			body = bytes.NewReader(nil)
		} else {
			encoded, err := json.Marshal(payload)
			if err != nil {
				t.Fatal(err)
			}
			body = bytes.NewReader(encoded)
		}
		request := httptest.NewRequest(method, path, body).WithContext(requestCtx)
		if payload != nil {
			request.Header.Set("Content-Type", "application/json")
		}
		if key != "" {
			request.Header.Set("Idempotency-Key", key)
		}
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		return response
	}

	days := 365
	effectiveTo := time.Now().UTC().AddDate(0, 1, 0).Format("2006-01-02")
	proposal := ProposeArchiveSeriesRetentionRuleRequest{
		TaxonomyNodeID: mainTaxonomy, SourceID: sourceID, EffectiveFrom: time.Now().UTC().Format("2006-01-02"),
		EffectiveTo: &effectiveTo, AnchorKind: "intake_received_at", DurationModel: "minimum_days", MinimumRetentionDays: &days,
	}
	// A legacy active source with browser-supplied checksum metadata has no
	// activation/evidence chain and must not become a retention authority.
	var legacySourceID string
	if err := adminSessions.QueryRow(adminCtx, `insert into school_regulatory_sources(tenant_code,institution_id,source_kind,citation,source_url,checksum_sha256,verified_at,verified_by_subject,revalidation_owner_subject,created_by_subject,updated_by_subject) values('tenant-egueducation','inst-001','law',$1,'https://publisher.test/legacy.pdf',repeat('a',64),now(),'legacy','legacy','legacy','legacy') returning id::text`, "legacy retention source "+uuid.NewString()).Scan(&legacySourceID); err != nil {
		t.Fatalf("seed legacy source: %v", err)
	}
	legacyProposal := proposal
	legacyProposal.SourceID = legacySourceID
	legacyResponse := send(proposer, http.MethodPost, "/earchiva/retention-rules", "legacy-metadata-only", legacyProposal)
	assertArchiveRetentionHTTPCode(t, legacyResponse, http.StatusConflict, "legacy metadata-only retention source")
	var legacyRules int
	if err := adminSessions.QueryRow(adminCtx, "select count(*) from archive_series_retention_rules where source_id=$1::uuid", legacySourceID).Scan(&legacyRules); err != nil || legacyRules != 0 {
		t.Fatalf("legacy source created retention rules=%d err=%v", legacyRules, err)
	}
	created := send(proposer, http.MethodPost, "/earchiva/retention-rules", "main-proposal", proposal)
	if created.Code != http.StatusCreated {
		t.Fatalf("propose rule: status=%d body=%s", created.Code, created.Body.String())
	}
	var proposed ArchiveSeriesRetentionRule
	decodeArchiveRetentionHTTP(t, created, &proposed)
	if proposed.ID == "" || proposed.Status != "proposed" || proposed.ExpectedVersion != 1 || proposed.ProposedBySubject != proposer.Subject || proposed.ApprovedAt != nil || proposed.SourceID != sourceID || proposed.TaxonomyNodeID != mainTaxonomy || proposed.MinimumRetentionDays == nil || *proposed.MinimumRetentionDays != days {
		t.Fatalf("propose did not return the full expected DTO: %#v", proposed)
	}

	replay := send(proposer, http.MethodPost, "/earchiva/retention-rules", "main-proposal", proposal)
	if replay.Code != http.StatusOK {
		t.Fatalf("same proposal replay: status=%d body=%s", replay.Code, replay.Body.String())
	}
	var replayed ArchiveSeriesRetentionRule
	decodeArchiveRetentionHTTP(t, replay, &replayed)
	if !bytes.Equal(replay.Body.Bytes(), created.Body.Bytes()) {
		t.Fatalf("replay DTO changed: got=%s want=%s", replay.Body.String(), created.Body.String())
	}
	var proposalAudits int
	if err := adminSessions.QueryRow(adminCtx, `select count(*) from app_audit_log where actor_subject=$1 and action='earchiva.retention.propose' and target_id=$2`, proposer.Subject, proposed.ID).Scan(&proposalAudits); err != nil || proposalAudits != 1 {
		t.Fatalf("proposal replay audit count=%d err=%v", proposalAudits, err)
	}

	changedDays := 366
	changedPayload := proposal
	changedPayload.MinimumRetentionDays = &changedDays
	assertArchiveRetentionHTTPCode(t, send(proposer, http.MethodPost, "/earchiva/retention-rules", "main-proposal", changedPayload), http.StatusConflict, "changed payload with same key")
	otherPayload := proposal
	otherPayload.TaxonomyNodeID = otherTaxonomy
	assertArchiveRetentionHTTPCode(t, send(proposer, http.MethodPost, "/earchiva/retention-rules", "main-proposal", otherPayload), http.StatusConflict, "same key on another resource")

	// The database lifecycle guard is the explicit rejection boundary here. It
	// currently maps its distinct-actor violation to the documented 409 contract.
	selfApproval := send(proposer, http.MethodPost, "/earchiva/retention-rules/"+proposed.ID+"/approve", "self-approval", ApproveArchiveSeriesRetentionRuleRequest{ExpectedVersion: 1})
	assertArchiveRetentionHTTPCode(t, selfApproval, http.StatusConflict, "same actor approval")
	var afterSelf ArchiveSeriesRetentionRule
	readRuleArchiveRetentionHTTP(t, send, proposer, proposed.ID, &afterSelf)
	if afterSelf.Status != "proposed" || afterSelf.ExpectedVersion != 1 || afterSelf.ApprovedBySubject != "" || afterSelf.ApprovedAt != nil {
		t.Fatalf("self approval changed lifecycle state: %#v", afterSelf)
	}

	approvedResponse := send(approver, http.MethodPost, "/earchiva/retention-rules/"+proposed.ID+"/approve", "approve-main", ApproveArchiveSeriesRetentionRuleRequest{ExpectedVersion: 1})
	if approvedResponse.Code != http.StatusOK {
		t.Fatalf("distinct approver: status=%d body=%s", approvedResponse.Code, approvedResponse.Body.String())
	}
	var approved ArchiveSeriesRetentionRule
	decodeArchiveRetentionHTTP(t, approvedResponse, &approved)
	if approved.Status != "active" || approved.ExpectedVersion != 2 || approved.ApprovedBySubject != approver.Subject || approved.ApprovedAt == nil {
		t.Fatalf("approval provenance missing: %#v", approved)
	}

	retiredResponse := send(proposer, http.MethodPost, "/earchiva/retention-rules/"+proposed.ID+"/retire", "retire-main", RetireArchiveSeriesRetentionRuleRequest{ExpectedVersion: 2, Reason: "superseded"})
	if retiredResponse.Code != http.StatusOK {
		t.Fatalf("retire approved rule: status=%d body=%s", retiredResponse.Code, retiredResponse.Body.String())
	}
	var retired ArchiveSeriesRetentionRule
	decodeArchiveRetentionHTTP(t, retiredResponse, &retired)
	if retired.Status != "retired" || retired.ExpectedVersion != 3 || retired.ApprovedBySubject != approver.Subject || retired.ApprovedAt == nil || retired.RetiredBySubject != proposer.Subject || retired.RetiredAt == nil || retired.RetirementReason != "superseded" {
		t.Fatalf("retirement lost lifecycle provenance: %#v", retired)
	}

	otherDays := 90
	otherPayload.MinimumRetentionDays = &otherDays
	second := send(proposer, http.MethodPost, "/earchiva/retention-rules", "other-proposal", otherPayload)
	if second.Code != http.StatusCreated {
		t.Fatalf("propose list fixture: status=%d body=%s", second.Code, second.Body.String())
	}
	list := send(reader, http.MethodGet, "/earchiva/retention-rules?filter.status=proposed&sort=minimum_retention_days&direction=asc&page=1&pageSize=1", "", nil)
	if list.Code != http.StatusOK {
		t.Fatalf("list filtered/sorted retention rules: status=%d body=%s", list.Code, list.Body.String())
	}
	var page struct {
		Items []ArchiveSeriesRetentionRule `json:"items"`
		Total int                          `json:"total"`
		Page  int                          `json:"page"`
		Size  int                          `json:"pageSize"`
	}
	decodeArchiveRetentionHTTP(t, list, &page)
	if page.Total != 1 || page.Page != 1 || page.Size != 1 || len(page.Items) != 1 || page.Items[0].TaxonomyNodeID != otherTaxonomy || page.Items[0].MinimumRetentionDays == nil || *page.Items[0].MinimumRetentionDays != otherDays {
		t.Fatalf("unexpected filtered/sorted page: %#v", page)
	}

	// The injected auth context still declares retention.manage, but a live
	// membership revocation must deny the following handler invocation.
	if _, err := adminSessions.Exec(adminCtx, `update app_memberships set active=false where user_id=$1::uuid and tenant_code='tenant-egueducation'`, revoked.ID); err != nil {
		t.Fatalf("revoke live retention membership: %v", err)
	}
	revokedPayload := proposal
	revokedPayload.TaxonomyNodeID = revokedTaxonomy
	assertArchiveRetentionHTTPCode(t, send(revoked, http.MethodPost, "/earchiva/retention-rules", "revoked-stale-session", revokedPayload), http.StatusForbidden, "live RBAC revocation with stale injected permission")
}

type archiveRetentionHTTPActor struct{ ID, Subject string }

func retentionScopedDB(t *testing.T, ctx context.Context, pool *pgxpool.Pool, actor string, admin bool) (context.Context, func()) {
	t.Helper()
	requestCtx, release, err := appdb.AcquireRequestConn(ctx, pool, appdb.SessionConfig{
		TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: actor, IsSuperAdmin: admin,
	})
	if err != nil {
		t.Fatalf("acquire scoped database session: %v", err)
	}
	return requestCtx, release
}

func seedArchiveRetentionActor(t *testing.T, ctx context.Context, sessions *appdb.SessionPool, position string) archiveRetentionHTTPActor {
	t.Helper()
	actor := archiveRetentionHTTPActor{Subject: "retention-http-" + position + "-" + uuid.NewString()}
	if err := sessions.QueryRow(ctx, `insert into app_users(sub,name,email,locale,status)
		values($1,$2,$3,'ro','active') returning id::text`, actor.Subject, "Retention HTTP "+position, uuid.NewString()+"@example.test").Scan(&actor.ID); err != nil {
		t.Fatalf("seed %s actor: %v", position, err)
	}
	// Copy the real tenant's director organization scope. The tested authority
	// comes from the migration's director/arhivar position-permission mappings,
	// not a test-only direct grant in app_user_permissions.
	if _, err := sessions.Exec(ctx, `insert into app_memberships(user_id,tenant_code,position_code,org_unit_code,organization_name,is_primary,active,start_date)
		select $1::uuid,'tenant-egueducation',$2,m.org_unit_code,m.organization_name,false,true,current_date
		from app_memberships m join app_users u on u.id=m.user_id
		where u.sub='usr-002' and m.tenant_code='tenant-egueducation' and m.position_code='director'
		order by m.is_primary desc limit 1`, actor.ID, position); err != nil {
		t.Fatalf("seed %s membership: %v", position, err)
	}
	var hasPositionGrant bool
	permission := "earchiva.retention.manage"
	if position == "director" {
		permission = "earchiva.retention.approve"
	}
	if err := sessions.QueryRow(ctx, `select exists(select 1 from app_position_permissions where position_code=$1 and permission_code=$2)`, position, permission).Scan(&hasPositionGrant); err != nil || !hasPositionGrant {
		t.Fatalf("missing real %s position permission %q: exists=%t err=%v", position, permission, hasPositionGrant, err)
	}
	return actor
}

func seedArchiveRetentionTaxonomy(t *testing.T, ctx context.Context, sessions *appdb.SessionPool, suffix string) string {
	t.Helper()
	var id string
	if err := sessions.QueryRow(ctx, `insert into archive_taxonomy_nodes(institution_id,code,label)
		values('inst-001',$1,$2) returning id::text`, "retention-http-"+suffix+"-"+uuid.NewString(), "Retention HTTP "+suffix).Scan(&id); err != nil {
		t.Fatalf("seed %s taxonomy: %v", suffix, err)
	}
	return id
}

func seedArchiveRetentionSource(t *testing.T, ctx context.Context, sessions *appdb.SessionPool) string {
	t.Helper()
	var id string
	if err := sessions.QueryRow(ctx, `insert into school_regulatory_sources(
		tenant_code,institution_id,source_kind,citation,source_url,checksum_sha256,status,effective_from,effective_to,
		verified_at,verified_by_subject,revalidation_owner_subject,created_by_subject,updated_by_subject)
		values('tenant-egueducation','inst-001','law',$1,'https://example.test/retention-http',repeat('a',64),'active',
		current_date-1,current_date+365,now(),'retention-fixture-source','retention-fixture-source','retention-fixture-source','retention-fixture-source')
		returning id::text`, "retention http legacy active source "+uuid.NewString()).Scan(&id); err != nil {
		t.Fatalf("seed legacy active source window: %v", err)
	}
	return id
}

func createActivatedRetentionSource(t *testing.T, ctx context.Context, pool *pgxpool.Pool, sessions *appdb.SessionPool, creator, approver archiveRetentionHTTPActor) string {
	t.Helper()
	service := regulatorysource.NewService(sessions, regulatoryFetcherFake{bytes: []byte("retention evidence")})
	router := chi.NewRouter()
	router.Post("/sources", service.Register)
	router.Post("/sources/{sourceID}/verify", service.Verify)
	router.Post("/sources/{sourceID}/activate", service.Activate)
	send := func(actor archiveRetentionHTTPActor, path, key string, payload any) *httptest.ResponseRecorder {
		requestCtx, release := retentionScopedDB(t, ctx, pool, actor.Subject, false)
		defer release()
		requestCtx = auth.WithSessionContextForIntegration(requestCtx, auth.SessionContext{TenantCode: "tenant-egueducation", InstitutionID: "inst-001", User: auth.SessionUser{ID: actor.ID, Sub: actor.Subject}})
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body)).WithContext(requestCtx)
		req.Header.Set("Idempotency-Key", key)
		req.Header.Set("Content-Type", "application/json")
		out := httptest.NewRecorder()
		router.ServeHTTP(out, req)
		return out
	}
	registered := send(creator, "/sources", "retention-source-register", map[string]any{"citation": "retention evidence source", "publisher_url": "https://publisher.test/retention.pdf", "issuer": "test", "source_kind": "law", "applicable_from": "2026-01-01"})
	if registered.Code != http.StatusCreated {
		t.Fatalf("register source %d %s", registered.Code, registered.Body.String())
	}
	var source regulatorysource.RegulatorySource
	if err := json.Unmarshal(registered.Body.Bytes(), &source); err != nil {
		t.Fatal(err)
	}
	verified := send(creator, "/sources/"+source.ID+"/verify", "retention-source-verify", map[string]any{"expected_version": 1})
	if verified.Code != http.StatusOK {
		t.Fatalf("verify source %d %s", verified.Code, verified.Body.String())
	}
	if err := json.Unmarshal(verified.Body.Bytes(), &source); err != nil || source.LatestEvidenceID == nil {
		t.Fatalf("verify dto %v %#v", err, source)
	}
	activated := send(approver, "/sources/"+source.ID+"/activate", "retention-source-activate", map[string]any{"expected_version": 2, "evidence_id": *source.LatestEvidenceID, "assessment": "retention applicable"})
	if activated.Code != http.StatusOK {
		t.Fatalf("activate source %d %s", activated.Code, activated.Body.String())
	}
	return source.ID
}

func assertArchiveRetentionHTTPCode(t *testing.T, response *httptest.ResponseRecorder, want int, label string) {
	t.Helper()
	if response.Code != want {
		t.Fatalf("%s: status=%d want=%d body=%s", label, response.Code, want, response.Body.String())
	}
}

func decodeArchiveRetentionHTTP(t *testing.T, response *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.Unmarshal(response.Body.Bytes(), target); err != nil {
		t.Fatalf("decode HTTP response %q: %v", response.Body.String(), err)
	}
}

func readRuleArchiveRetentionHTTP(t *testing.T, send func(archiveRetentionHTTPActor, string, string, string, any) *httptest.ResponseRecorder, actor archiveRetentionHTTPActor, id string, target *ArchiveSeriesRetentionRule) {
	t.Helper()
	// There is no single-rule HTTP endpoint. The filtered paginated collection
	// is the public read surface and also exercises filter handling.
	response := send(actor, http.MethodGet, "/earchiva/retention-rules?filter.status=proposed&pageSize=100", "", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("list rule after rejected approval: status=%d body=%s", response.Code, response.Body.String())
	}
	var page struct {
		Items []ArchiveSeriesRetentionRule `json:"items"`
	}
	decodeArchiveRetentionHTTP(t, response, &page)
	for _, item := range page.Items {
		if item.ID == id {
			*target = item
			return
		}
	}
	t.Fatalf("proposed rule %s absent from filtered collection: %#v", id, page.Items)
}
