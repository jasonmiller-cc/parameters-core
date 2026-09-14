package ui

import (
	"encoding/json"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/jasonmiller-cc/parameters-core/pkg/version"
)

// DashboardConfig configures the dashboard HTTP handler.
type DashboardConfig struct {
	Registry    *Registry
	PollInterval time.Duration
}

// Handler returns an http.Handler that serves the dashboard UI and its API.
// Mount it at "/" or any prefix.
func (cfg *DashboardConfig) Handler(mux *http.ServeMux) {
	// Static assets served from the embedded filesystem.
	mux.Handle("GET /", http.HandlerFunc(serveIndex))
	mux.Handle("GET /assets/", http.FileServer(http.FS(assets)))

	// Dashboard API.
	mux.HandleFunc("GET /api/platform/summary", cfg.handleSummary)
	mux.HandleFunc("GET /api/platform/services", cfg.handleServices)
	mux.HandleFunc("POST /api/platform/poll", cfg.handlePoll)
	mux.HandleFunc("GET /api/version", cfg.handleVersion)

	// Per-service proxy: forwards requests to the backing service.
	// e.g. GET /proxy/dns/api/v1/zones → parameters-dns /api/v1/zones
	mux.HandleFunc("GET /proxy/", cfg.handleProxy)
	mux.HandleFunc("POST /proxy/", cfg.handleProxy)
	mux.HandleFunc("PUT /proxy/", cfg.handleProxy)
	mux.HandleFunc("DELETE /proxy/", cfg.handleProxy)
	mux.HandleFunc("PATCH /proxy/", cfg.handleProxy)
}

type summaryResponse struct {
	Status    string    `json:"status"`
	Services  int       `json:"services"`
	Healthy   int       `json:"healthy"`
	Degraded  int       `json:"degraded"`
	Down      int       `json:"down"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (cfg *DashboardConfig) handleSummary(w http.ResponseWriter, r *http.Request) {
	statuses := cfg.Registry.Statuses()
	resp := summaryResponse{
		Status:    cfg.Registry.Summary(),
		Services:  len(statuses),
		UpdatedAt: time.Now(),
	}
	for _, s := range statuses {
		switch s.Status {
		case "ok":
			resp.Healthy++
		case "degraded":
			resp.Degraded++
		default:
			resp.Down++
		}
	}
	writeJSON(w, resp)
}

func (cfg *DashboardConfig) handleServices(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, cfg.Registry.Statuses())
}

func (cfg *DashboardConfig) handlePoll(w http.ResponseWriter, r *http.Request) {
	go cfg.Registry.Poll(r.Context())
	writeJSON(w, map[string]string{"status": "polling"})
}

func (cfg *DashboardConfig) handleVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, version.Get())
}

func (cfg *DashboardConfig) handleProxy(w http.ResponseWriter, r *http.Request) {
	// /proxy/{service-name}/{rest...}
	path := strings.TrimPrefix(r.URL.Path, "/proxy/")
	slash := strings.IndexByte(path, '/')
	if slash < 0 {
		http.Error(w, "missing service name", http.StatusBadRequest)
		return
	}
	svcName := path[:slash]
	rest := path[slash:]

	// Find the service URL in the registry.
	cfg.Registry.mu.RLock()
	var target string
	for _, s := range cfg.Registry.services {
		if s.Name == svcName {
			target = s.URL
			break
		}
	}
	cfg.Registry.mu.RUnlock()

	if target == "" {
		http.Error(w, "unknown service: "+svcName, http.StatusNotFound)
		return
	}

	u, err := url.Parse(target)
	if err != nil {
		http.Error(w, "invalid service URL", http.StatusInternalServerError)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(u)
	r2 := r.Clone(r.Context())
	r2.URL.Path = rest
	r2.URL.RawPath = rest
	proxy.ServeHTTP(w, r2)
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	f, err := assets.Open("index.html")
	if err != nil {
		http.Error(w, "dashboard not found", http.StatusInternalServerError)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeContent(w, r, "index.html", time.Time{}, f.(interface {
		Read([]byte) (int, error)
		Seek(int64, int) (int64, error)
	}))
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
