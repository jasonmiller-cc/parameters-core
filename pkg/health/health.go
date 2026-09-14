// Package health provides a composable health-check framework for HTTP services.
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type Status string

const (
	StatusOK      Status = "ok"
	StatusDegraded Status = "degraded"
	StatusDown    Status = "down"
)

// CheckFn is a function that returns whether a dependency is healthy.
type CheckFn func(ctx context.Context) error

type check struct {
	name string
	fn   CheckFn
}

// Checker runs named health checks and exposes an HTTP handler.
type Checker struct {
	mu       sync.RWMutex
	checks   []check
	timeout  time.Duration
	service  string
	version  string
}

// New returns a Checker with a 5 s per-check timeout.
func New(service, version string) *Checker {
	return &Checker{
		service: service,
		version: version,
		timeout: 5 * time.Second,
	}
}

// Add registers a named check.
func (c *Checker) Add(name string, fn CheckFn) *Checker {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.checks = append(c.checks, check{name: name, fn: fn})
	return c
}

// WithTimeout overrides the per-check deadline.
func (c *Checker) WithTimeout(d time.Duration) *Checker {
	c.timeout = d
	return c
}

type Report struct {
	Status  Status            `json:"status"`
	Service string            `json:"service"`
	Version string            `json:"version"`
	Checks  map[string]string `json:"checks"`
}

func (c *Checker) run(ctx context.Context) Report {
	c.mu.RLock()
	checks := c.checks
	c.mu.RUnlock()

	report := Report{
		Status:  StatusOK,
		Service: c.service,
		Version: c.version,
		Checks:  make(map[string]string, len(checks)),
	}

	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, ch := range checks {
		wg.Add(1)
		go func(ch check) {
			defer wg.Done()
			ctx2, cancel := context.WithTimeout(ctx, c.timeout)
			defer cancel()

			var result string
			if err := ch.fn(ctx2); err != nil {
				result = "error: " + err.Error()
			} else {
				result = string(StatusOK)
			}

			mu.Lock()
			report.Checks[ch.name] = result
			if result != string(StatusOK) {
				report.Status = StatusDegraded
			}
			mu.Unlock()
		}(ch)
	}
	wg.Wait()
	return report
}

// Handler returns an http.HandlerFunc for /healthz or /readyz.
func (c *Checker) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		report := c.run(r.Context())
		status := http.StatusOK
		if report.Status != StatusOK {
			status = http.StatusServiceUnavailable
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(report)
	}
}

// LiveHandler is a trivial liveness check that always returns 200.
func LiveHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}
}
