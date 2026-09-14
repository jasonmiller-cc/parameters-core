package ui_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jasonmiller-cc/parameters-core/pkg/ui"
)

func TestNewRegistry_InitialStatusUnknown(t *testing.T) {
	reg := ui.NewRegistry([]ui.ServiceEntry{
		{Name: "dns", DisplayName: "DNS", URL: "http://localhost:9999"},
	})

	statuses := reg.Statuses()
	if len(statuses) != 1 {
		t.Fatalf("len(statuses) = %d, want 1", len(statuses))
	}
	if statuses[0].Status != "unknown" {
		t.Errorf("Status = %q, want unknown", statuses[0].Status)
	}
}

func TestPoll_HealthyService(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			t.Errorf("path = %s, want /healthz", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","version":"v1.2.3","checks":{"db":"ok"}}`))
	}))
	defer srv.Close()

	reg := ui.NewRegistry([]ui.ServiceEntry{{Name: "dns", URL: srv.URL}})
	reg.Poll(context.Background())

	statuses := reg.Statuses()
	if len(statuses) != 1 {
		t.Fatalf("len(statuses) = %d, want 1", len(statuses))
	}
	st := statuses[0]
	if st.Status != "ok" {
		t.Errorf("Status = %q, want ok", st.Status)
	}
	if st.Version != "v1.2.3" {
		t.Errorf("Version = %q, want v1.2.3", st.Version)
	}
	if st.Checks["db"] != "ok" {
		t.Errorf("Checks[db] = %q, want ok", st.Checks["db"])
	}
}

func TestPoll_UnreachableService(t *testing.T) {
	reg := ui.NewRegistry([]ui.ServiceEntry{{Name: "dns", URL: "http://127.0.0.1:1"}})
	reg.Poll(context.Background())

	statuses := reg.Statuses()
	if statuses[0].Status != "down" {
		t.Errorf("Status = %q, want down", statuses[0].Status)
	}
	if statuses[0].Error == "" {
		t.Error("expected Error to be set for unreachable service")
	}
}

func TestPoll_5xxMarksDown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	reg := ui.NewRegistry([]ui.ServiceEntry{{Name: "dns", URL: srv.URL}})
	reg.Poll(context.Background())

	if got := reg.Statuses()[0].Status; got != "down" {
		t.Errorf("Status = %q, want down (5xx should override body status)", got)
	}
}

func TestSummary(t *testing.T) {
	tests := []struct {
		name     string
		statuses []string
		want     string
	}{
		{"empty", nil, "unknown"},
		{"all ok", []string{"ok", "ok"}, "ok"},
		{"one degraded", []string{"ok", "degraded"}, "degraded"},
		{"one down", []string{"ok", "down"}, "degraded"},
		{"all down", []string{"down", "down"}, "down"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var entries []ui.ServiceEntry
			for i := range tt.statuses {
				entries = append(entries, ui.ServiceEntry{Name: string(rune('a' + i))})
			}
			reg := ui.NewRegistry(entries)

			// Poll against a server returning the desired status per service,
			// or skip polling entirely for the empty case.
			if len(tt.statuses) == 0 {
				if got := reg.Summary(); got != tt.want {
					t.Errorf("Summary() = %q, want %q", got, tt.want)
				}
				return
			}

			srvs := make([]*httptest.Server, len(tt.statuses))
			for i, status := range tt.statuses {
				status := status
				srvs[i] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = w.Write([]byte(`{"status":"` + status + `"}`))
				}))
				entries[i].URL = srvs[i].URL
			}
			defer func() {
				for _, s := range srvs {
					s.Close()
				}
			}()

			reg = ui.NewRegistry(entries)
			reg.Poll(context.Background())

			if got := reg.Summary(); got != tt.want {
				t.Errorf("Summary() = %q, want %q", got, tt.want)
			}
		})
	}
}
