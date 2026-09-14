package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	corelog "github.com/jasonmiller-cc/parameters-core/pkg/log"
	"github.com/jasonmiller-cc/parameters-core/pkg/middleware"
)

func TestRequestID_GeneratesAndPropagates(t *testing.T) {
	var idFromCtx string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idFromCtx = middleware.GetRequestID(r.Context())
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	middleware.RequestID(next).ServeHTTP(w, req)

	header := w.Header().Get("X-Request-ID")
	if header == "" {
		t.Fatal("expected X-Request-ID header to be set")
	}
	if idFromCtx != header {
		t.Errorf("context request ID %q does not match header %q", idFromCtx, header)
	}
}

func TestRequestID_PreservesIncoming(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-ID", "fixed-id")
	w := httptest.NewRecorder()
	middleware.RequestID(next).ServeHTTP(w, req)

	if got := w.Header().Get("X-Request-ID"); got != "fixed-id" {
		t.Errorf("X-Request-ID = %q, want fixed-id", got)
	}
}

func TestRecover_CatchesPanic(t *testing.T) {
	log := corelog.New(corelog.LevelError, corelog.FormatJSON)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic escaped Recover middleware: %v", r)
			}
		}()
		middleware.Recover(log)(next).ServeHTTP(w, req)
	}()

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

func TestLogger_CapturesStatusAndSize(t *testing.T) {
	log := corelog.New(corelog.LevelInfo, corelog.FormatJSON)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("hello"))
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	middleware.Logger(log)(next).ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", w.Code)
	}
	if w.Body.String() != "hello" {
		t.Errorf("body = %q, want hello", w.Body.String())
	}
}

func TestCORS_Wildcard(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	handler := middleware.CORS("*")(next)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want https://example.com", got)
	}
}

func TestCORS_Allowlist_Rejects(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	handler := middleware.CORS("https://allowed.example.com")(next)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin = %q, want empty for disallowed origin", got)
	}
}

func TestCORS_PreflightOptions(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called for OPTIONS preflight")
	})
	handler := middleware.CORS("*")(next)

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", w.Code)
	}
}

func TestChain_OrdersOutermostFirst(t *testing.T) {
	var order []string

	mw := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name)
				next.ServeHTTP(w, r)
			})
		}
	}

	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "final")
	})

	handler := middleware.Chain(mw("first"), mw("second"))(final)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	want := []string{"first", "second", "final"}
	if len(order) != len(want) {
		t.Fatalf("order = %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Errorf("order = %v, want %v", order, want)
			break
		}
	}
}
