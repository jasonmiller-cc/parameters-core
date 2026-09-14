package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jasonmiller-cc/parameters-core/pkg/metrics"
)

func TestNew_HandlerServesPrometheusFormat(t *testing.T) {
	reg := metrics.New("testsvc")

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	reg.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "go_goroutines") {
		t.Errorf("expected default Go collector metrics in output, got: %s", w.Body.String())
	}
}

func TestMiddleware_InstrumentsRequests(t *testing.T) {
	reg := metrics.New("testsvc2")

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := reg.Middleware(next)

	req := httptest.NewRequest("GET", "/foo", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	metricsReq := httptest.NewRequest("GET", "/metrics", nil)
	metricsW := httptest.NewRecorder()
	reg.Handler().ServeHTTP(metricsW, metricsReq)

	if !strings.Contains(metricsW.Body.String(), "parameters_testsvc2_http_requests_total") {
		t.Errorf("expected instrumented counter in metrics output, got: %s", metricsW.Body.String())
	}
}
