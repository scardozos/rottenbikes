package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/scardozos/rottenbikes/internal/domain"
)

func TestHandleRequestMagicLink(t *testing.T) {
	mockService := &MockService{
		CreateMagicLinkFunc: func(ctx context.Context, email string) (string, error) {
			return "magic-token-for-" + email, nil
		},
	}

	srv, err := New(mockService, ":8080")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	t.Run("success", func(t *testing.T) {
		email := "test@example.com"
		reqBody, _ := json.Marshal(map[string]string{
			"email":         email,
			"captcha_token": "token",
		})

		req := httptest.NewRequest(http.MethodPost, "/auth/request-magic-link", bytes.NewReader(reqBody))
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusAccepted {
			t.Errorf("expected status 202, got %d", w.Code)
		}
	})

	t.Run("missing_email", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]string{
			"captcha_token": "token",
		})

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

		req := httptest.NewRequest(http.MethodGet, "/auth/confirm?token="+token, nil)
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("missing_token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/confirm", nil)
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})
}
