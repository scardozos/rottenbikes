package httpserver

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	httpResponseSize = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "Size of HTTP responses in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8), // 100B to ~1GB
		},
	)

	httpRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Current number of HTTP requests being processed",
		},
	)
)

// ResponseWriter wraps http.ResponseWriter to capture status code, size and username
type ResponseWriter struct {
	http.ResponseWriter
	Status   int
	Size     int
	Username string
}

func (rw *ResponseWriter) WriteHeader(status int) {
	rw.Status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *ResponseWriter) Write(b []byte) (int, error) {
	if rw.Status == 0 {
		rw.Status = http.StatusOK
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.Size += n
	return n, err
}

func observabilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &ResponseWriter{ResponseWriter: w}

		httpRequestsInFlight.Inc()
		defer httpRequestsInFlight.Dec()

		next.ServeHTTP(rw, r)

		duration := time.Since(start).Seconds()

		username := rw.Username
		if username == "" {
			username = "-"
		}

		// Log the request
		log.Printf(
			"method=%s path=%s status=%d duration=%.4fs size=%d username=%s",
			r.Method,
			r.RequestURI,
			rw.Status,
			duration,
			rw.Size,
			username,
		)

		// Record metrics
		// Use r.URL.Path instead of RequestURI to avoid high cardinality with query params if any
		path := sanitizePath(r.URL.Path)
		status := strconv.Itoa(rw.Status)

		httpRequestsTotal.WithLabelValues(r.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(r.Method, path).Observe(duration)
		httpResponseSize.Observe(float64(rw.Size))
	})
}

func sanitizePath(path string) string {
	if strings.HasPrefix(path, "/auth/confirm/") {
		return "/auth/confirm/MAGIC_TOKEN_REDACTED"
	}
	return path
}
