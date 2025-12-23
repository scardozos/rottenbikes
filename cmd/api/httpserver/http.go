package httpserver

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

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
	mux.HandleFunc("/auth/request-magic-link", s.handleRequestMagicLink)
	mux.HandleFunc("/auth/confirm/", s.handleConfirmMagicLink)
	mux.HandleFunc("/auth/poll", s.handlePollMagicLink)
	mux.HandleFunc("/auth/register", s.handleRegister)
	mux.HandleFunc("/auth/verify", s.middlewareAuth(http.HandlerFunc(s.handleVerifyToken)).ServeHTTP)

	// /bikes → list (GET, public) and create (POST, auth)
	mux.HandleFunc("/bikes", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			s.handleListBikes(w, r)
		case http.MethodPost:
			s.middlewareAuth(http.HandlerFunc(s.handleCreateBike)).ServeHTTP(w, r)
		default:
			s.sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// /bikes/{id}, /bikes/{id}/reviews, /bikes/{id}/ratings
	// GETs are public; CUD goes through auth middleware
	mux.HandleFunc("/bikes/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodDelete:
			s.middlewareAuth(http.HandlerFunc(s.handleBikeSubroutes)).ServeHTTP(w, r)
		default:
			s.handleBikeSubroutes(w, r)
		}
	})

	// /bikes/reviews → all reviews with ratings (GET, public)
	mux.HandleFunc("/bikes/reviews", s.handleListAllReviewsWithRatings)

	// /bikes/ratings → aggregates for all bikes (GET, public)
	mux.HandleFunc("/bikes/ratings", s.handleListAllBikeRatings)

	// /reviews/{id} → get (public), update/delete (auth)
	mux.HandleFunc("/reviews/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut, http.MethodDelete:
			s.middlewareAuth(http.HandlerFunc(s.handleReviewSubroutes)).ServeHTTP(w, r)
		default:
			s.handleReviewSubroutes(w, r)
		}
	})

	s.server = &http.Server{
		Addr:    addr,
		Handler: corsMiddleware(mux),
	}

	return s, nil
}

// /bikes/{id}/...
func (s *HTTPServer) handleBikeSubroutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/bikes/")
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) == 1 && parts[0] != "" {
		// /bikes/{id}
		idStr := parts[0]
		bikeID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			s.sendError(w, "invalid bike id", http.StatusBadRequest)
			return
		}
		switch r.Method {
		case http.MethodGet:
			s.handleGetBike(w, r, bikeID)
		case http.MethodPut:
			s.handleUpdateBike(w, r, bikeID)
		case http.MethodDelete:
			s.handleDeleteBike(w, r, bikeID)
		default:
			s.sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	if len(parts) == 2 {
		idStr, sub := parts[0], parts[1]
		bikeID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			s.sendError(w, "invalid bike id", http.StatusBadRequest)
			return
		}

		switch sub {
		case "reviews":
			if r.Method == http.MethodGet {
				s.handleBikeReviews(w, r, bikeID)
				return
			}
			if r.Method == http.MethodPost {
				s.handleCreateBikeReview(w, r, bikeID)
				return
			}
			s.sendError(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		case "ratings":
			if r.Method == http.MethodGet {
				s.handleBikeRatings(w, r, bikeID)
				return
			}
			s.sendError(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
	}

	s.sendError(w, "not found", http.StatusNotFound)
}

// /reviews/{id}...
func (s *HTTPServer) handleReviewSubroutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/reviews/")
	idStr := strings.Trim(path, "/")
	if idStr == "" {
		s.sendError(w, "not found", http.StatusNotFound)
		return
	}

	reviewID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.sendError(w, "invalid review id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetReview(w, r, reviewID)
	case http.MethodPut:
		s.handleUpdateReview(w, r, reviewID)
	case http.MethodDelete:
		s.handleDeleteReview(w, r, reviewID)
	default:
		s.sendError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *HTTPServer) sendError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (s *HTTPServer) Start() error {
	log.Printf("HTTP server listening on %s", s.server.Addr)
	return s.server.ListenAndServe()
}

func (s *HTTPServer) Shutdown(ctx context.Context) error {
	log.Printf("Shutting down HTTP server on %s", s.server.Addr)
	return s.server.Shutdown(ctx)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow all origins for now (dev/web UI on 8081)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
