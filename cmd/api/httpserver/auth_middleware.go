package httpserver

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/scardozos/rottenbikes/internal/domain"
)

type contextKey string

const contextKeyPosterID contextKey = "poster_id"
const contextKeyUsername contextKey = "username"

func posterIDFromContext(ctx context.Context) (int64, bool) {
	v := ctx.Value(contextKeyPosterID)
	if v == nil {
		return 0, false
	}
	id, ok := v.(int64)
	return id, ok
}

func usernameFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(contextKeyUsername)
	if v == nil {
		return "", false
	}
	u, ok := v.(string)
	return u, ok
}

// middlewareAuth enforces a valid Bearer API token and injects poster_id into context.
func (s *HTTPServer) middlewareAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if h == "" {
			s.sendError(w, "missing Authorization header", http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(h, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			s.sendError(w, "invalid Authorization header", http.StatusUnauthorized)
			return
		}
		token := strings.TrimSpace(parts[1])
		if token == "" {
			s.sendError(w, "empty bearer token", http.StatusUnauthorized)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		poster, err := s.service.GetPosterByAPIToken(ctx, token)
		if err != nil {
			if errors.Is(err, domain.ErrInvalidToken) || errors.Is(err, domain.ErrTokenExpired) || errors.Is(err, domain.ErrEmailNotVerified) {
				s.sendError(w, "invalid or expired api token", http.StatusUnauthorized)
			} else {
				// We don't want to log the user out if the database connection failed
				s.sendError(w, "internal server error", http.StatusInternalServerError)
			}
			return
		}

		ctx = context.WithValue(ctx, contextKeyPosterID, poster.PosterID)
		ctx = context.WithValue(ctx, contextKeyUsername, poster.Username)

		if rw, ok := w.(*ResponseWriter); ok {
			rw.Username = poster.Username
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
