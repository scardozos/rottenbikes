package httpserver

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog/log"
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
	Body     bytes.Buffer // Capture body for error logging
}

func (rw *ResponseWriter) WriteHeader(status int) {
	rw.Status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *ResponseWriter) Write(b []byte) (int, error) {
	if rw.Status == 0 {
		rw.Status = http.StatusOK
	}
	// Capture body if error, up to a limit
	if rw.Status >= 400 && rw.Body.Len() < 512 {
		toWrite := b
		if rw.Body.Len()+len(b) > 512 {
			toWrite = b[:512-rw.Body.Len()]
		}
		rw.Body.Write(toWrite)
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

		requestID := generateRequestID()
		logger := log.With().Str("request_id", requestID).Logger()
		ctx := logger.WithContext(r.Context())

		next.ServeHTTP(rw, r.WithContext(ctx))

		duration := time.Since(start).Seconds()

		username := rw.Username
		if username == "" {
			username = "-"
		}

		// Extract error message if applicable
		var errMsg string
		if rw.Status >= 400 {
			// Try to parse JSON error
			var errBody struct {
				Error string `json:"error"`
			}
			if json.Unmarshal(rw.Body.Bytes(), &errBody) == nil && errBody.Error != "" {
				errMsg = errBody.Error
			} else {
				// Fallback to raw body or generic message if parsing fails or empty
				errMsg = rw.Body.String()
				if errMsg == "" {
					errMsg = "unknown error"
				}
			}
		} else {
			errMsg = "-"
		}

		// Log the request
		event := logger.Info()
		if errMsg != "-" {
			event.Str("error", errMsg)
		} else {
			event.Str("error", "-")
		}

		event.
			Str("type", "access").
			Str("method", r.Method).
			Str("path", r.RequestURI).
			Int("status", rw.Status).
			Float64("duration", duration).
			Int("size", rw.Size).
			Str("username", username).
			Msg("request_completed")

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

func generateRequestID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback if rand fails
		return "fallback-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return hex.EncodeToString(b)
}
