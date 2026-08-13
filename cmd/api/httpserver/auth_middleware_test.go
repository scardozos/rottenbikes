package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/scardozos/rottenbikes/cmd/api/email"
	"github.com/scardozos/rottenbikes/internal/domain"
)

func TestAuthMiddleware(t *testing.T) {
	mockService := &MockService{
		GetPosterByAPITokenFunc: func(ctx context.Context, token string) (*domain.AuthPoster, error) {
			if token == "good-token" {
				return &domain.AuthPoster{PosterID: 42, Email: "u@example.com", Username: "u"}, nil
			}
			return nil, errors.New("invalid token")
		},
	}

	srv, err := New(mockService, &email.NoopSender{}, ":8080")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	get := func(auth string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/auth/verify", nil)
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		w := httptest.NewRecorder()
		srv.server.Handler.ServeHTTP(w, req)
		return w
	}

	t.Run("missing_authorization", func(t *testing.T) {
		if w := get(""); w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})

	t.Run("wrong_scheme", func(t *testing.T) {
		if w := get("Basic abc"); w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 for non-Bearer scheme, got %d", w.Code)
		}
	})

	t.Run("empty_bearer_token", func(t *testing.T) {
		if w := get("Bearer "); w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 for empty bearer, got %d", w.Code)
		}
	})

	t.Run("invalid_token", func(t *testing.T) {
		if w := get("Bearer not-a-real-token"); w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 for invalid token, got %d", w.Code)
		}
	})

	t.Run("valid_token", func(t *testing.T) {
		// The middleware passes the raw bearer to the service; hashing happens
		// inside the store (mocked here), so the mock sees the raw token.
		if w := get("Bearer good-token"); w.Code != http.StatusOK {
			t.Errorf("expected 200 for valid token, got %d (body %s)", w.Code, w.Body.String())
		}
	})
}
