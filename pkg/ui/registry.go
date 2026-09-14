// Package ui provides the parameters platform dashboard — a central web UI
// that aggregates health, version, and status from all registered services.
package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// ServiceEntry describes one parameters service in the registry.
type ServiceEntry struct {
	Name        string `yaml:"name" json:"name"`
	DisplayName string `yaml:"display_name" json:"display_name"`
	URL         string `yaml:"url" json:"url"`
	Description string `yaml:"description" json:"description"`
	Group       string `yaml:"group" json:"group"`
}

// ServiceStatus is the last-known health result for a service.
type ServiceStatus struct {
	ServiceEntry
	Status    string            `json:"status"`
	Version   string            `json:"version,omitempty"`
	Checks    map[string]string `json:"checks,omitempty"`
	LastCheck time.Time         `json:"last_check"`
	Latency   int64             `json:"latency_ms"`
	Error     string            `json:"error,omitempty"`
}

// Registry holds the list of registered services and their last polled status.
type Registry struct {
	mu       sync.RWMutex
	services []ServiceEntry
	statuses map[string]*ServiceStatus
	client   *http.Client
}

// NewRegistry creates a Registry from a list of service entries.
func NewRegistry(services []ServiceEntry) *Registry {
	r := &Registry{
		services: services,
		statuses: make(map[string]*ServiceStatus, len(services)),
		client:   &http.Client{Timeout: 5 * time.Second},
	}
	for _, s := range services {
		r.statuses[s.Name] = &ServiceStatus{ServiceEntry: s, Status: "unknown"}
	}
	return r
}

// Poll checks every service's /healthz endpoint concurrently.
func (r *Registry) Poll(ctx context.Context) {
	r.mu.RLock()
	services := r.services
	r.mu.RUnlock()

	var wg sync.WaitGroup
	for _, svc := range services {
		wg.Add(1)
		go func(svc ServiceEntry) {
			defer wg.Done()
			status := r.probe(ctx, svc)
			r.mu.Lock()
			r.statuses[svc.Name] = status
			r.mu.Unlock()
		}(svc)
	}
	wg.Wait()
}

// StartPolling runs Poll on the given interval until ctx is cancelled.
func (r *Registry) StartPolling(ctx context.Context, interval time.Duration) {
	r.Poll(ctx)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.Poll(ctx)
		}
	}
}

// Statuses returns a snapshot of all service statuses.
func (r *Registry) Statuses() []*ServiceStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*ServiceStatus, 0, len(r.statuses))
	for _, s := range r.services {
		if st, ok := r.statuses[s.Name]; ok {
			cp := *st
			out = append(out, &cp)
		}
	}
	return out
}

// Summary returns overall platform health: ok | degraded | down.
func (r *Registry) Summary() string {
	statuses := r.Statuses()
	if len(statuses) == 0 {
		return "unknown"
	}
	down, degraded := 0, 0
	for _, s := range statuses {
		switch s.Status {
		case "down", "error":
			down++
		case "degraded":
			degraded++
		}
	}
	switch {
	case down == len(statuses):
		return "down"
	case down > 0 || degraded > 0:
		return "degraded"
	default:
		return "ok"
	}
}

type healthResponse struct {
	Status  string            `json:"status"`
	Version string            `json:"version,omitempty"`
	Service string            `json:"service,omitempty"`
	Checks  map[string]string `json:"checks,omitempty"`
}

func (r *Registry) probe(ctx context.Context, svc ServiceEntry) *ServiceStatus {
	st := &ServiceStatus{ServiceEntry: svc, LastCheck: time.Now()}

	url := svc.URL + "/healthz"
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		st.Status = "error"
		st.Error = fmt.Sprintf("build request: %v", err)
		return st
	}

	resp, err := r.client.Do(req)
	st.Latency = time.Since(start).Milliseconds()
	if err != nil {
		st.Status = "down"
		st.Error = err.Error()
		return st
	}
	defer func() { _ = resp.Body.Close() }()

	var h healthResponse
	if err := json.NewDecoder(resp.Body).Decode(&h); err != nil {
		st.Status = "degraded"
		st.Error = fmt.Sprintf("decode response: %v", err)
		return st
	}

	st.Status = h.Status
	st.Version = h.Version
	st.Checks = h.Checks
	if resp.StatusCode >= 500 {
		st.Status = "down"
	}
	return st
}
