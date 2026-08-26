package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/scardozos/rottenbikes/cmd/api/email"
	"github.com/scardozos/rottenbikes/internal/domain"
)

func TestHandlePollMagicLink(t *testing.T) {
	mockService := &MockService{
		CheckMagicLinkStatusFunc: func(ctx context.Context, token string) (string, error) {
			if token == "good-poll" {
				return "recovered-api-token", nil
			}
			if token == "pending" {
				return "", nil
			}
			if token == "boom" {
				return "", errors.New("db down")
			}
			return "", nil
		},
	}

	srv, err := New(mockService, &email.NoopSender{}, ":8080")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	t.Run("confirmed_returns_api_token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/poll?token=good-poll", nil)
		w := httptest.NewRecorder()
		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var resp map[string]string
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp["api_token"] != "recovered-api-token" {
			t.Errorf("expected recovered-api-token, got %q", resp["api_token"])
		}
	})

	t.Run("not_confirmed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/poll?token=pending", nil)
		w := httptest.NewRecorder()
		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404 for unconfirmed link, got %d", w.Code)
		}
	})

	t.Run("missing_token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/poll", nil)
		w := httptest.NewRecorder()
		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for missing token, got %d", w.Code)
		}
	})

	t.Run("internal_error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/poll?token=boom", nil)
		w := httptest.NewRecorder()
		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", w.Code)
		}
	})
}

func TestHandleRegister(t *testing.T) {
	t.Setenv("APP_ENV", "local") // skips hCaptcha when HCAPTCHA_SECRET unset

	mockService := &MockService{
		GetPosterByAPITokenFunc: func(ctx context.Context, token string) (*domain.AuthPoster, error) {
			return &domain.AuthPoster{PosterID: 1}, nil
		},
		RegisterFunc: func(ctx context.Context, username, eml string) (string, string, error) {
			if eml == "taken@example.com" {
				return "", "", domain.ErrEmailExists
			}
			if eml == "baduser@example.com" {
				return "", "", domain.ErrUsernameExists
			}
			if eml == "dbfail@example.com" {
				return "", "", errors.New("db error")
			}
			return "emailed-magic-token", "client-poll-token", nil
		},
	}

	srv, err := New(mockService, &email.NoopSender{}, ":8080")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	post := func(body map[string]string) *httptest.ResponseRecorder {
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(b))
		w := httptest.NewRecorder()
		srv.server.Handler.ServeHTTP(w, req)
		return w
	}

	t.Run("success", func(t *testing.T) {
		w := post(map[string]string{
			"username":      "newuser",
			"email":         "new@example.com",
			"captcha_token": "x",
		})
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]string
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		// The response credential is the poll token (raw), NOT anything derivable
		// from the emailed magic token.
		if resp["magic_token"] != "client-poll-token" {
			t.Errorf("expected poll token in magic_token, got %q", resp["magic_token"])
		}
		if resp["magic_token"] == domain.HashToken("emailed-magic-token") {
			t.Error("poll token must not equal the hash of the emailed magic token")
		}
	})

	t.Run("missing_fields", func(t *testing.T) {
		w := post(map[string]string{"username": "x"})
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("email_conflict", func(t *testing.T) {
		w := post(map[string]string{
			"username":      "a",
			"email":         "taken@example.com",
			"captcha_token": "x",
		})
		if w.Code != http.StatusConflict {
			t.Errorf("expected 409, got %d", w.Code)
		}
	})

	t.Run("username_conflict", func(t *testing.T) {
		w := post(map[string]string{
			"username":      "a",
			"email":         "baduser@example.com",
			"captcha_token": "x",
		})
		if w.Code != http.StatusConflict {
			t.Errorf("expected 409, got %d", w.Code)
		}
	})

	t.Run("internal_error", func(t *testing.T) {
		w := post(map[string]string{
			"username":      "a",
			"email":         "dbfail@example.com",
			"captcha_token": "x",
		})
		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", w.Code)
		}
	})

	t.Run("wrong_method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/register", nil)
		w := httptest.NewRecorder()
		srv.server.Handler.ServeHTTP(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", w.Code)
		}
	})
}

func TestHandleDeletePoster(t *testing.T) {
	var capturedPoster int64
	var capturedDeleteContent bool
	mockService := &MockService{
		GetPosterByAPITokenFunc: func(ctx context.Context, token string) (*domain.AuthPoster, error) {
			// Middleware passes the raw bearer; hashing is a store-layer detail
			// hidden behind this service boundary, so the mock sees the raw token.
			if token == "good" {
				return &domain.AuthPoster{PosterID: 7}, nil
			}
			return nil, errors.New("invalid")
		},
		DeletePosterFunc: func(ctx context.Context, posterID int64, deleteContent bool) error {
			capturedPoster = posterID
			capturedDeleteContent = deleteContent
			if posterID == 999 {
				return errors.New("db error")
			}
			return nil
		},
	}

	srv, err := New(mockService, &email.NoopSender{}, ":8080")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	t.Run("unauth_without_token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/auth/user", nil)
		w := httptest.NewRecorder()
		srv.server.Handler.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})

	t.Run("success_204_empty_body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/auth/user", nil)
		req.Header.Set("Authorization", "Bearer good")
		w := httptest.NewRecorder()
		srv.server.Handler.ServeHTTP(w, req)
		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", w.Code)
		}
		if capturedPoster != 7 {
			t.Errorf("expected poster 7, got %d", capturedPoster)
		}
		if capturedDeleteContent {
			t.Error("expected delete_poster_subresources=false by default")
		}
	})

	t.Run("success_with_delete_content_flag", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/auth/user", bytes.NewReader([]byte(`{"delete_poster_subresources":true}`)))
		req.Header.Set("Authorization", "Bearer good")
		w := httptest.NewRecorder()
		srv.server.Handler.ServeHTTP(w, req)
		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", w.Code)
		}
		if !capturedDeleteContent {
			t.Error("expected delete_poster_subresources=true to propagate")
		}
	})

	t.Run("wrong_method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/user", nil)
		req.Header.Set("Authorization", "Bearer good")
		w := httptest.NewRecorder()
		srv.server.Handler.ServeHTTP(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", w.Code)
		}
	})

	// Use a poster whose DeletePoster fails.
	t.Run("internal_error", func(t *testing.T) {
		mockService.GetPosterByAPITokenFunc = func(ctx context.Context, token string) (*domain.AuthPoster, error) {
			return &domain.AuthPoster{PosterID: 999}, nil
		}
		req := httptest.NewRequest(http.MethodDelete, "/auth/user", nil)
		req.Header.Set("Authorization", "Bearer good")
		w := httptest.NewRecorder()
		srv.server.Handler.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", w.Code)
		}
	})
}

func TestHealthz(t *testing.T) {
	srv, err := New(&MockService{}, &email.NoopSender{}, ":8080")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "ok" {
		t.Errorf("expected body ok, got %q", w.Body.String())
	}
}
