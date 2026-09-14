package health_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jasonmiller-cc/parameters-core/pkg/health"
)

func TestChecker_AllOK(t *testing.T) {
	c := health.New("test-service", "v0.1.0")
	c.Add("db", func(ctx context.Context) error { return nil })
	c.Add("cache", func(ctx context.Context) error { return nil })

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	c.Handler()(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body)
	}
	if !strings.Contains(w.Body.String(), `"status":"ok"`) {
		t.Errorf("expected ok status in body: %s", w.Body)
	}
}

func TestChecker_CheckFails(t *testing.T) {
	c := health.New("test-service", "v0.1.0")
	c.Add("db", func(ctx context.Context) error { return errors.New("connection refused") })

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	c.Handler()(w, req)

	if w.Code != 503 {
		t.Fatalf("expected 503, got %d: %s", w.Code, w.Body)
	}
}
