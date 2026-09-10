//go:build integration

package education

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	appdb "github.com/eguilde/egueducation/internal/db"
)

func TestCommitteeMemberParentIntegrityIntegration(t *testing.T) {
	it := newGovernanceIntegrationDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	adminPool := openGovernanceIntegrationPool(t, ctx, it.databaseConfig)
	defer adminPool.Close()
	if err := appdb.Migrate(ctx, adminPool); err != nil {
		t.Fatalf("migrate disposable committee database: %v", err)
	}
	grantGovernanceIntegrationAccess(t, ctx, adminPool, it.roleName)

	committeeA := uuid.NewString()
	committeeB := uuid.NewString()
	if _, err := adminPool.Exec(ctx, `
		insert into education_committees (
			id, committee_code, school_year, committee_type, title, status, starts_on, institution_id
		) values
			($1::uuid, 'COM-PARENT-A', '2030-2031', 'permanenta', 'Committee A', 'active', current_date, 'inst-001'),
			($2::uuid, 'COM-PARENT-B', '2030-2031', 'permanenta', 'Committee B', 'active', current_date, 'inst-balotesti')
	`, committeeA, committeeB); err != nil {
		t.Fatalf("seed institution-scoped committees: %v", err)
	}

	_, err := adminPool.Exec(ctx, `
		insert into education_committee_members (
			committee_id, full_name, role_name, member_type, status, appointed_on, institution_id
		) values ($1::uuid, 'Cross Tenant', 'Member', 'membru', 'active', current_date, 'inst-001')
	`, committeeB)
	var pgErr *pgconn.PgError
	if err == nil || !strings.Contains(err.Error(), "education_committee_members_parent_institution_fk") {
		t.Fatalf("composite committee parent FK must reject a cross-institution member, got %v", err)
	}
	if !errors.As(err, &pgErr) || pgErr.Code != "23503" {
		t.Fatalf("cross-institution member error must be foreign_key_violation, got %v", err)
	}

	service := NewService(appdb.NewSessionPool(it.readerPool))
	ctxA, releaseA := governanceTenantContext(t, ctx, it.readerPool, "tenant-egueducation", "inst-001", "committee-integrity-test")
	defer releaseA()

	foreignCreate := committeeMemberRequest(ctxA, http.MethodPost, committeeB, "", `{"full_name":"Cross Tenant","role_name":"Member","member_type":"membru","voting_right":true,"status":"active","appointed_on":"2030-09-01"}`)
	foreignResponse := httptest.NewRecorder()
	service.CreateCommitteeMember(foreignResponse, foreignCreate)
	if foreignResponse.Code != http.StatusNotFound {
		t.Fatalf("handler must hide a foreign-institution parent: status=%d body=%s", foreignResponse.Code, foreignResponse.Body.String())
	}

	validCreate := committeeMemberRequest(ctxA, http.MethodPost, committeeA, "", `{"full_name":"Local Member","role_name":"Member","member_type":"membru","voting_right":true,"status":"active","appointed_on":"2030-09-01"}`)
	validResponse := httptest.NewRecorder()
	service.CreateCommitteeMember(validResponse, validCreate)
	if validResponse.Code != http.StatusCreated {
		t.Fatalf("handler must accept a same-institution parent: status=%d body=%s", validResponse.Code, validResponse.Body.String())
	}
	var created CommitteeMember
	if err := json.Unmarshal(validResponse.Body.Bytes(), &created); err != nil || created.ID == "" {
		t.Fatalf("decode created committee member: member=%#v err=%v", created, err)
	}

	foreignUpdate := committeeMemberRequest(ctxA, http.MethodPut, committeeB, created.ID, `{"full_name":"Cross Tenant Update","role_name":"Member","member_type":"membru","voting_right":true,"status":"active","appointed_on":"2030-09-01"}`)
	foreignUpdateResponse := httptest.NewRecorder()
	service.UpdateCommitteeMember(foreignUpdateResponse, foreignUpdate)
	if foreignUpdateResponse.Code != http.StatusNotFound {
		t.Fatalf("handler must reject an update through a foreign-institution parent: status=%d body=%s", foreignUpdateResponse.Code, foreignUpdateResponse.Body.String())
	}

	validUpdate := committeeMemberRequest(ctxA, http.MethodPut, committeeA, created.ID, `{"full_name":"Updated Local Member","role_name":"Member","member_type":"membru","voting_right":true,"status":"active","appointed_on":"2030-09-01"}`)
	updateResponse := httptest.NewRecorder()
	service.UpdateCommitteeMember(updateResponse, validUpdate)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("handler must update a member only through its same-institution parent: status=%d body=%s", updateResponse.Code, updateResponse.Body.String())
	}
}

func committeeMemberRequest(ctx context.Context, method, committeeID, memberID, payload string) *http.Request {
	request := requestWithContext(ctx, "tenant-egueducation", "inst-001", "committee-integrity-test")
	request.Method = method
	request.Body = http.NoBody
	if payload != "" {
		request.Body = io.NopCloser(strings.NewReader(payload))
	}
	route := chi.NewRouteContext()
	route.URLParams.Add("recordID", committeeID)
	if memberID != "" {
		route.URLParams.Add("itemID", memberID)
	}
	return request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, route))
}
