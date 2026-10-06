package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSwaggerDocumentation(t *testing.T) {
	mux := NewRouter(NewOrderHandler(&OrderServiceMock{}))
	ui := httptest.NewRecorder()
	mux.ServeHTTP(ui, httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil))
	if ui.Code != http.StatusOK || !strings.Contains(ui.Body.String(), "doc.json") {
		t.Fatalf("Swagger UI failed: status %d", ui.Code)
	}
	doc := httptest.NewRecorder()
	mux.ServeHTTP(doc, httptest.NewRequest(http.MethodGet, "/swagger/doc.json", nil))
	if doc.Code != http.StatusOK {
		t.Fatalf("Swagger document returned %d: %s", doc.Code, doc.Body.String())
	}
	var spec struct {
		Swagger string
		Paths   map[string]map[string]json.RawMessage
	}
	if err := json.Unmarshal(doc.Body.Bytes(), &spec); err != nil {
		t.Fatal(err)
	}
	if spec.Swagger != "2.0" {
		t.Fatalf("unexpected Swagger version: %s", spec.Swagger)
	}
	for path, methods := range map[string][]string{
		"/health": {"get"}, "/orders": {"get", "post"}, "/orders/{id}": {"get", "put", "delete"},
	} {
		for _, method := range methods {
			if len(spec.Paths[path][method]) == 0 {
				t.Errorf("missing %s %s documentation", method, path)
			}
		}
	}
}
