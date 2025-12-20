package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/scardozos/rottenbikes/internal/domain"
)

func TestHandleRequestMagicLink(t *testing.T) {
	mockService := &MockService{
		RegisterFunc: func(ctx context.Context, username, email string) (string, error) {
			return "magic-token-for-" + email, nil
		},
		CreateMagicLinkFunc: func(ctx context.Context, email string) (string, error) {
			if email == "test@example.com" {
				return "magic-token-for-" + email, nil
			}
			return "", fmt.Errorf("user not found")
		},
	}

	srv, err := New(mockService, ":8080")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	t.Run("success", func(t *testing.T) {
		email := "test@example.com"
		reqBody, _ := json.Marshal(map[string]string{
			"email": email,
		})

		req := httptest.NewRequest(http.MethodPost, "/auth/request-magic-link", bytes.NewReader(reqBody))
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("user_not_found", func(t *testing.T) {
		email := "unknown@example.com"
		reqBody, _ := json.Marshal(map[string]string{
			"email": email,
		})

		req := httptest.NewRequest(http.MethodPost, "/auth/request-magic-link", bytes.NewReader(reqBody))
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("missing_fields", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]string{})

		req := httptest.NewRequest(http.MethodPost, "/auth/request-magic-link", bytes.NewReader(reqBody))
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})
}

func TestHandleConfirmMagicLink(t *testing.T) {
	mockService := &MockService{
		ConfirmMagicLinkFunc: func(ctx context.Context, token string) (*domain.ConfirmResult, error) {
			return &domain.ConfirmResult{
				APIToken:          "new-api-token",
				Email:             "test@example.com",
				APITokenExpiresAt: time.Now().Add(24 * time.Hour),
			}, nil
		},
	}

	srv, err := New(mockService, ":8080")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	t.Run("success", func(t *testing.T) {
		token := "magic_token"

		req := httptest.NewRequest(http.MethodGet, "/auth/confirm/"+token, nil)
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("missing_token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/confirm/", nil)
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})
}

func TestHandleVerifyToken(t *testing.T) {
	mockService := &MockService{
		GetPosterByAPITokenFunc: func(ctx context.Context, token string) (*domain.AuthPoster, error) {
			if token == "valid-token" {
				return &domain.AuthPoster{PosterID: 123}, nil
			}
			return nil, fmt.Errorf("invalid token")
		},
	}

	srv, err := New(mockService, ":8080")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/verify", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		var resp map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp["status"] != "ok" {
			t.Errorf("expected status ok, got %v", resp["status"])
		}
		if resp["poster_id"] != float64(123) { // json numbers are float64
			t.Errorf("expected poster_id 123, got %v", resp["poster_id"])
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/verify", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
	})
}
