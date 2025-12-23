package httpserver

import (
	"context"
	"net/http"
	"strings"
	"time"
)

type contextKey string

const contextKeyPosterID contextKey = "poster_id"

func posterIDFromContext(ctx context.Context) (int64, bool) {
	v := ctx.Value(contextKeyPosterID)
	if v == nil {
		return 0, false
	}
	id, ok := v.(int64)
	return id, ok
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
			s.sendError(w, "invalid or expired api token", http.StatusUnauthorized)
			return
		}

		ctx = context.WithValue(ctx, contextKeyPosterID, poster.PosterID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
