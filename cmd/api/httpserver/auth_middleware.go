package httpserver

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/scardozos/rottenbikes/internal/domain"
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
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("missing Authorization header"))
			return
		}

		parts := strings.SplitN(h, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("invalid Authorization header"))
			return
		}
		token := strings.TrimSpace(parts[1])
		if token == "" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("empty bearer token"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		poster, err := domain.GetPosterByAPIToken(ctx, s.db, token)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("invalid or expired api token"))
			return
		}

		ctx = context.WithValue(ctx, contextKeyPosterID, poster.PosterID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
