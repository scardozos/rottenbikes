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

	"github.com/scardozos/rottenbikes/cmd/api/email"
	"github.com/scardozos/rottenbikes/internal/domain"
)

func TestHandleRequestMagicLink(t *testing.T) {
	mockService := &MockService{
		RegisterFunc: func(ctx context.Context, username, email string) (string, error) {
			return "magic-token-for-" + email, nil
		},
		CreateMagicLinkFunc: func(ctx context.Context, email string) (string, string, error) {
			if email == "test@example.com" || email == "testuser" {
				return "magic-token-for-" + email, "test@example.com", nil
			}
			return "", "", domain.ErrUserNotFound
		},
	}

	srv, err := New(mockService, &email.NoopSender{}, ":8080")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	t.Run("success", func(t *testing.T) {
		email := "test@example.com"
		reqBody, _ := json.Marshal(map[string]string{
			"email":         email,
			"captcha_token": "valid-captcha",
		})

		req := httptest.NewRequest(http.MethodPost, "/auth/request-magic-link", bytes.NewReader(reqBody))
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("user_not_found", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]string{
			"email":         "nonexistent@example.com",
			"captcha_token": "valid-captcha",
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
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("success_by_username", func(t *testing.T) {
		username := "testuser"
		reqBody, _ := json.Marshal(map[string]string{
			"username":      username,
			"captcha_token": "valid-captcha",
		})

		req := httptest.NewRequest(http.MethodPost, "/auth/request-magic-link", bytes.NewReader(reqBody))
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp map[string]string
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp["magic_token"] != "magic-token-for-"+username {
			t.Errorf("expected magic_token %s, got %s", "magic-token-for-"+username, resp["magic_token"])
		}
	})

	t.Run("rate_limit_exceeded", func(t *testing.T) {
		mockService.CreateMagicLinkFunc = func(ctx context.Context, email string) (string, string, error) {
			return "", "", domain.ErrRateLimitExceeded
		}

		reqBody, _ := json.Marshal(map[string]string{
			"email":         "test@example.com",
			"captcha_token": "valid-captcha",
		})

		req := httptest.NewRequest(http.MethodPost, "/auth/request-magic-link", bytes.NewReader(reqBody))
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusTooManyRequests {
			t.Errorf("expected status 429, got %d", w.Code)
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

	srv, err := New(mockService, &email.NoopSender{}, ":8080")
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

	srv, err := New(mockService, &email.NoopSender{}, ":8080")
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

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		host     string
		expected bool
	}{
		{"localhost", true},
		{"127.0.0.1", true},
		{"192.168.1.1", true},
		{"10.0.0.5", true},
		{"172.16.0.1", true},
		{"172.31.255.255", true},
		{"example.com", false},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			if got := isPrivateIP(tt.host); got != tt.expected {
				t.Errorf("isPrivateIP(%q) = %v; want %v", tt.host, got, tt.expected)
			}
		})
	}
}
