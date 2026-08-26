package httpserver

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/scardozos/rottenbikes/cmd/api/email"
	"github.com/scardozos/rottenbikes/internal/domain"
)

type HTTPServer struct {
	service     domain.Service
	emailSender email.EmailSender
	server      *http.Server
}

func New(service domain.Service, sender email.EmailSender, addr string) (*HTTPServer, error) {
	// Ping check removed as it belongs to the store/db layer, or we can add a HealthCheck method to Service
	// For now, we'll assume the service is ready or check it if we add a method.

	s := &HTTPServer{service: service, emailSender: sender}

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		// Simplified health check
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Auth endpoints (public)
	mux.HandleFunc("POST /auth/request-magic-link", s.handleRequestMagicLink)
	mux.HandleFunc("GET /auth/confirm/{token}", s.handleConfirmMagicLink)
	mux.HandleFunc("GET /auth/poll", s.handlePollMagicLink)
	mux.HandleFunc("POST /auth/register", s.handleRegister)
	mux.HandleFunc("GET /auth/verify", s.middlewareAuth(http.HandlerFunc(s.handleVerifyToken)).ServeHTTP)
	mux.HandleFunc("DELETE /auth/user", s.middlewareAuth(http.HandlerFunc(s.handleDeletePoster)).ServeHTTP)
	mux.HandleFunc("GET /users/me/reviews", s.middlewareAuth(http.HandlerFunc(s.handleListMyReviews)).ServeHTTP)

	// /bikes
	mux.HandleFunc("GET /bikes", s.handleListBikes)
	mux.HandleFunc("POST /bikes", s.middlewareAuth(http.HandlerFunc(s.handleCreateBike)).ServeHTTP)
	
	// /bikes/{id}
	mux.HandleFunc("GET /bikes/{id}", s.handleGetBike)
	mux.HandleFunc("PUT /bikes/{id}", s.middlewareAuth(http.HandlerFunc(s.handleUpdateBike)).ServeHTTP)
	mux.HandleFunc("DELETE /bikes/{id}", s.middlewareAuth(http.HandlerFunc(s.handleDeleteBike)).ServeHTTP)

	// /bikes/{id}/...
	mux.HandleFunc("POST /bikes/{id}/reviews", s.middlewareAuth(http.HandlerFunc(s.handleCreateBikeReview)).ServeHTTP)
	mux.HandleFunc("GET /bikes/{id}/reviews", s.handleListBikeReviews)
	mux.HandleFunc("GET /bikes/{id}/details", s.handleGetBikeDetails)

	// /reviews/{id}
	mux.HandleFunc("GET /reviews/{id}", s.handleGetReview)
	mux.HandleFunc("PUT /reviews/{id}", s.middlewareAuth(http.HandlerFunc(s.handleUpdateReview)).ServeHTTP)
	mux.HandleFunc("DELETE /reviews/{id}", s.middlewareAuth(http.HandlerFunc(s.handleDeleteReview)).ServeHTTP)

	s.server = &http.Server{
		Addr:              addr,
		Handler:           observabilityMiddleware(corsMiddleware(json405Middleware(mux))),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	return s, nil
}

func (s *HTTPServer) sendError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (s *HTTPServer) sendInternalServerError(w http.ResponseWriter, r *http.Request, err error) {
	zerolog.Ctx(r.Context()).Error().Err(err).Msg("internal server error")
	s.sendError(w, "internal server error", http.StatusInternalServerError)
}

func (s *HTTPServer) Start() error {
	log.Info().Msgf("HTTP server listening on %s", s.server.Addr)
	return s.server.ListenAndServe()
}

func (s *HTTPServer) Shutdown(ctx context.Context) error {
	log.Info().Msgf("Shutting down HTTP server on %s", s.server.Addr)
	return s.server.Shutdown(ctx)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := origin != "" && isOriginAllowed(origin)

		w.Header().Add("Vary", "Origin")

		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}

		if r.Method == http.MethodOptions {
			if allowed {
				w.WriteHeader(http.StatusNoContent)
			} else {
				w.WriteHeader(http.StatusForbidden)
			}
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isOriginAllowed(origin string) bool {
	for _, a := range allowedOrigins() {
		if a == origin {
			return true
		}
	}
	
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	
	ip := net.ParseIP(u.Hostname())
	if ip == nil {
		return false
	}
	
	return ip.IsPrivate() || ip.IsLoopback()
}

func allowedOrigins() []string {
	raw := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if raw == "" {
		return defaultAllowedOrigins()
	}
	var out []string
	for _, o := range strings.Split(raw, ",") {
		if o = strings.TrimSpace(o); o != "" {
			out = append(out, o)
		}
	}
	if len(out) == 0 {
		return defaultAllowedOrigins()
	}
	return out
}

func defaultAllowedOrigins() []string {
	return []string{"http://localhost:8081", "http://localhost:8080"}
}

func json405Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &statusInterceptor{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)
		if rw.statusCode == http.StatusMethodNotAllowed {
			// If we haven't written the body yet, or if ServeMux wrote a plaintext Method Not Allowed,
			// unfortunately ServeMux already wrote it. 
			// A better way is to use a response interceptor that prevents writing body on 405.
		}
	})
}

type statusInterceptor struct {
	http.ResponseWriter
	statusCode int
	wroteBody  bool
}

func (i *statusInterceptor) WriteHeader(code int) {
	if code == http.StatusMethodNotAllowed {
		i.statusCode = code
		i.ResponseWriter.Header().Set("Content-Type", "application/json")
		i.ResponseWriter.WriteHeader(code)
		_ = json.NewEncoder(i.ResponseWriter).Encode(map[string]string{"error": "method not allowed"})
		return
	}
	i.statusCode = code
	i.ResponseWriter.WriteHeader(code)
}

func (i *statusInterceptor) Write(b []byte) (int, error) {
	if i.statusCode == http.StatusMethodNotAllowed {
		return len(b), nil // discard the plaintext
	}
	return i.ResponseWriter.Write(b)
}

func parsePagination(r *http.Request, defaultLimit, maxLimit int) (limit, offset int) {
	limit = defaultLimit
	offset = 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil && parsedLimit > 0 {
			if parsedLimit > maxLimit {
				limit = maxLimit
			} else {
				limit = parsedLimit
			}
		}
	}

	if o := r.URL.Query().Get("offset"); o != "" {
		if parsedOffset, err := strconv.Atoi(o); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	return limit, offset
}
