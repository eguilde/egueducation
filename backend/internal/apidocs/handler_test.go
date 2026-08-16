package apidocs

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSpecServesEmbeddedOpenAPI(t *testing.T) {
	recorder := httptest.NewRecorder()
	Spec().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/openapi.json", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"openapi":"3.1.1"`) && !strings.Contains(recorder.Body.String(), `"openapi": "3.1.1"`) {
		t.Fatal("embedded document is not the generated OpenAPI 3.1.1 contract")
	}
}

func TestSwaggerUIUsesLocalContractAndDisablesPersistentAuthorization(t *testing.T) {
	recorder := httptest.NewRecorder()
	SwaggerUI().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/docs", nil))
	body := recorder.Body.String()
	if !strings.Contains(body, `src="/api/docs/init.js"`) {
		t.Fatal("Swagger UI is not configured with the safe local contract")
	}
	if recorder.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("Swagger UI must declare a content security policy")
	}
	initRecorder := httptest.NewRecorder()
	SwaggerInit().ServeHTTP(initRecorder, httptest.NewRequest(http.MethodGet, "/api/docs/init.js", nil))
	if !strings.Contains(initRecorder.Body.String(), "url:'/api/openapi.json'") || !strings.Contains(initRecorder.Body.String(), "persistAuthorization:false") {
		t.Fatal("Swagger initializer is not configured with the safe local contract")
	}
}
