// Package metrics provides Prometheus instrumentation helpers for parameters services.
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Registry wraps a prometheus.Registry with pre-registered HTTP metrics.
type Registry struct {
	reg         *prometheus.Registry
	reqTotal    *prometheus.CounterVec
	reqDuration *prometheus.HistogramVec
	reqInFlight prometheus.Gauge
}

// New creates a Registry with standard HTTP metrics pre-registered.
func New(service string) *Registry {
	reg := prometheus.NewRegistry()

	labels := []string{"method", "path", "status"}

	reqTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "parameters",
		Subsystem: service,
		Name:      "http_requests_total",
		Help:      "Total number of HTTP requests.",
	}, labels)

	reqDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "parameters",
		Subsystem: service,
		Name:      "http_request_duration_seconds",
		Help:      "HTTP request duration in seconds.",
		Buckets:   prometheus.DefBuckets,
	}, labels[:2])

	reqInFlight := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "parameters",
		Subsystem: service,
		Name:      "http_requests_in_flight",
		Help:      "Current number of in-flight HTTP requests.",
	})

	reg.MustRegister(reqTotal, reqDuration, reqInFlight)
	reg.MustRegister(prometheus.NewGoCollector())
	reg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	return &Registry{
		reg:         reg,
		reqTotal:    reqTotal,
		reqDuration: reqDuration,
		reqInFlight: reqInFlight,
	}
}

// Handler returns an HTTP handler that exposes Prometheus metrics.
func (r *Registry) Handler() http.Handler {
	return promhttp.HandlerFor(r.reg, promhttp.HandlerOpts{})
}

// MustRegister adds additional collectors.
func (r *Registry) MustRegister(cs ...prometheus.Collector) {
	r.reg.MustRegister(cs...)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

// Middleware instruments every handler with request counters and latency histograms.
func (r *Registry) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		sr := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		r.reqInFlight.Inc()
		start := time.Now()

		next.ServeHTTP(sr, req)

		r.reqInFlight.Dec()
		elapsed := time.Since(start).Seconds()
		status := strconv.Itoa(sr.status)
		path := req.URL.Path

		r.reqTotal.WithLabelValues(req.Method, path, status).Inc()
		r.reqDuration.WithLabelValues(req.Method, path).Observe(elapsed)
	})
}
