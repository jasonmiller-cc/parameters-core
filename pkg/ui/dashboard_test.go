package ui_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jasonmiller-cc/parameters-core/pkg/ui"
)

func newTestMux(reg *ui.Registry) *http.ServeMux {
	mux := http.NewServeMux()
	dash := &ui.DashboardConfig{Registry: reg, PollInterval: 30 * time.Second}
	dash.Handler(mux)
	return mux
}

func TestDashboard_ServesIndex(t *testing.T) {
	mux := newTestMux(ui.NewRegistry(nil))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Parameters Dashboard") {
		t.Errorf("expected dashboard HTML in body, got: %.200s", w.Body.String())
	}
}

func TestDashboard_ServesStaticAssets(t *testing.T) {
	mux := newTestMux(ui.NewRegistry(nil))

	req := httptest.NewRequest("GET", "/assets/app.js", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if w.Body.Len() == 0 {
		t.Error("expected non-empty app.js body")
	}
}

func TestDashboard_APIVersion(t *testing.T) {
	mux := newTestMux(ui.NewRegistry(nil))

	req := httptest.NewRequest("GET", "/api/version", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if _, ok := body["version"]; !ok {
		t.Error("expected version field in response")
	}
}

func TestDashboard_APISummary(t *testing.T) {
	reg := ui.NewRegistry([]ui.ServiceEntry{{Name: "dns"}, {Name: "ldap"}})
	mux := newTestMux(reg)

	req := httptest.NewRequest("GET", "/api/platform/summary", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var body struct {
		Services int `json:"services"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Services != 2 {
		t.Errorf("Services = %d, want 2", body.Services)
	}
}

func TestDashboard_APIServices(t *testing.T) {
	reg := ui.NewRegistry([]ui.ServiceEntry{{Name: "dns"}})
	mux := newTestMux(reg)

	req := httptest.NewRequest("GET", "/api/platform/services", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var body []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(body) != 1 {
		t.Errorf("len(body) = %d, want 1", len(body))
	}
}

func TestDashboard_APIPoll(t *testing.T) {
	reg := ui.NewRegistry(nil)
	mux := newTestMux(reg)

	req := httptest.NewRequest("POST", "/api/platform/poll", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestDashboard_Proxy(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/zones" {
			t.Errorf("backend path = %s, want /api/v1/zones", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("zones"))
	}))
	defer backend.Close()

	reg := ui.NewRegistry([]ui.ServiceEntry{{Name: "dns", URL: backend.URL}})
	mux := newTestMux(reg)

	req := httptest.NewRequest("GET", "/proxy/dns/api/v1/zones", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if w.Body.String() != "zones" {
		t.Errorf("body = %q, want zones", w.Body.String())
	}
}

func TestDashboard_Proxy_UnknownService(t *testing.T) {
	reg := ui.NewRegistry(nil)
	mux := newTestMux(reg)

	req := httptest.NewRequest("GET", "/proxy/nonexistent/api/v1/zones", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestDashboard_Proxy_MissingServiceName(t *testing.T) {
	reg := ui.NewRegistry(nil)
	mux := newTestMux(reg)

	req := httptest.NewRequest("GET", "/proxy/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}
