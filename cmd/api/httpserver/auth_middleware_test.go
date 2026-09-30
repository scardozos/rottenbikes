package httpserver

import (
	"context"
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
			return nil, domain.ErrInvalidToken
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

type dummyWrapper struct {
	http.ResponseWriter
}

func (d *dummyWrapper) Unwrap() http.ResponseWriter {
	return d.ResponseWriter
}

func TestAuthMiddlewareUnwrap(t *testing.T) {
	mockService := &MockService{
		GetPosterByAPITokenFunc: func(ctx context.Context, token string) (*domain.AuthPoster, error) {
			return &domain.AuthPoster{PosterID: 42, Username: "testuser"}, nil
		},
	}
	srv := &HTTPServer{service: mockService}

	handler := srv.middlewareAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid")

	rec := httptest.NewRecorder()

	// Create custom ResponseWriter that we want to extract
	customRW := &ResponseWriter{ResponseWriter: rec}
	// Wrap it in dummyWrapper
	wrapped := &dummyWrapper{ResponseWriter: customRW}

	handler.ServeHTTP(wrapped, req)

	if customRW.Username != "testuser" {
		t.Errorf("expected username 'testuser', got '%s'", customRW.Username)
	}
}

// The middleware's lookup timeout must not leak into the handler: handlers
// set their own (longer) timeouts, and a child context cannot outlive its
// parent's deadline.
func TestAuthMiddlewareDoesNotBoundHandlerContext(t *testing.T) {
	var lookupHadDeadline bool
	mockService := &MockService{
		GetPosterByAPITokenFunc: func(ctx context.Context, token string) (*domain.AuthPoster, error) {
			_, lookupHadDeadline = ctx.Deadline()
			return &domain.AuthPoster{PosterID: 42, Username: "u"}, nil
		},
	}
	srv := &HTTPServer{service: mockService}

	var handlerHadDeadline, handlerCtxDone bool
	handler := srv.middlewareAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, handlerHadDeadline = r.Context().Deadline()
		handlerCtxDone = r.Context().Err() != nil
		if id, ok := posterIDFromContext(r.Context()); !ok || id != 42 {
			t.Errorf("expected poster_id 42 in context, got %d (ok=%v)", id, ok)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if !lookupHadDeadline {
		t.Error("expected the token lookup to run with a deadline")
	}
	if handlerHadDeadline {
		t.Error("handler context must not inherit the token lookup deadline")
	}
	if handlerCtxDone {
		t.Error("handler context must not be cancelled when the lookup finishes")
	}
}
