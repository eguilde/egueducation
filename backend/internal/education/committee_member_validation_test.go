package education

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestCommitteeMemberHandlersRejectInvalidAppointedOnBeforePersistence(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name    string
		method  string
		handler func(*Service, http.ResponseWriter, *http.Request)
	}{
		{name: "create", method: http.MethodPost, handler: (*Service).CreateCommitteeMember},
		{name: "update", method: http.MethodPatch, handler: (*Service).UpdateCommitteeMember},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			request := httptest.NewRequest(testCase.method, "/", io.NopCloser(strings.NewReader(`{
				"full_name":"Ana Pop",
				"role_name":"Președinte",
				"member_type":"presedinte",
				"status":"active",
				"appointed_on":"not-a-date"
			}`)))
			route := chi.NewRouteContext()
			route.URLParams.Add("recordID", "committee-1")
			route.URLParams.Add("itemID", "member-1")
			request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, route))
			response := httptest.NewRecorder()

			testCase.handler(&Service{}, response, request)

			if response.Code != http.StatusBadRequest {
				t.Fatalf("expected invalid appointed_on to return 400, got %d: %s", response.Code, response.Body.String())
			}
			if !strings.Contains(response.Body.String(), `"code":"invalid_committee_member_appointed_on"`) {
				t.Fatalf("expected stable appointed_on validation code, got %s", response.Body.String())
			}
		})
	}
}
