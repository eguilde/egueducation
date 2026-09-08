package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestParsePageQueryFailsClosedForSortAndNormalizesInput(t *testing.T) {
	values := url.Values{
		"page": {"0"}, "pageSize": {"999"}, "sort": {"unsafe_sql"}, "direction": {"DESC"},
		"filter.status": {"  active  "}, "filter.ignored": {"must-not-pass"},
	}
	query := ParsePageQuery(values, map[string]struct{}{"name": {}}, []string{"status"})
	if query.Page != 1 || query.PageSize != 100 || query.Sort != "" || query.Direction != "desc" {
		t.Fatalf("unexpected normalized query: %#v", query)
	}
	if len(query.Filters) != 1 || query.Filters["status"] != "active" {
		t.Fatalf("filters must be allow-listed and trimmed: %#v", query.Filters)
	}
}

func TestParsePageQueryUsesSafeDefaults(t *testing.T) {
	query := ParsePageQuery(url.Values{"page": {"2"}, "pageSize": {"10"}, "sort": {"name"}, "direction": {"not-a-direction"}}, map[string]struct{}{"name": {}}, nil)
	if query.Page != 2 || query.PageSize != 10 || query.Sort != "name" || query.Direction != "asc" {
		t.Fatalf("unexpected safe defaults: %#v", query)
	}
}

func TestJSONAndWritePageExposeStableJSONContract(t *testing.T) {
	recorder := httptest.NewRecorder()
	WritePage(recorder, http.StatusCreated, []string{"one"}, 3, 2, 25)
	if got := recorder.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("content type = %q", got)
	}
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d", recorder.Code)
	}
	var body PageResponse[string]
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Total != 3 || body.Page != 2 || body.PageSize != 25 || len(body.Items) != 1 || body.Items[0] != "one" {
		t.Fatalf("unexpected response: %#v", body)
	}
}
