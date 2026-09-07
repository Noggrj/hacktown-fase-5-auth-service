// Package metrics exposes a Prometheus /metrics endpoint plus a thin HTTP
// middleware recording request count and latency. Scraped by the
// Prometheus Agent deployed alongside the service in EKS.
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests processed, labeled by path/method/status.",
	}, []string{"path", "method", "status"})

	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"path", "method"})

	// Business metrics — exported so internal/auth/usecase can record
	// them directly. Deliberately not derived from httpRequests: those
	// are generic (any path/status), these mean something specific
	// (a user actually got created / actually got a token), and they're
	// what the Grafana dashboard in fiapx-infra graphs.
	UsersRegistered = promauto.NewCounter(prometheus.CounterOpts{
		Name: "fiapx_users_registered_total",
		Help: "Total users successfully registered.",
	})

	Logins = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "fiapx_logins_total",
		Help: "Total login attempts, labeled by result.",
	}, []string{"result"}) // "success" | "invalid_credentials"
)

// Handler serves the /metrics scrape endpoint.
func Handler() http.Handler {
	return promhttp.Handler()
}

// Middleware records one observation per request. Uses r.URL.Path
// directly (not a route-pattern) — acceptable cardinality since this
// service only exposes a handful of fixed, non-parameterized routes.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		httpRequests.WithLabelValues(r.URL.Path, r.Method, strconv.Itoa(sw.status)).Inc()
		httpDuration.WithLabelValues(r.URL.Path, r.Method).Observe(time.Since(start).Seconds())
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
