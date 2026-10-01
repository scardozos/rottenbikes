package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/scardozos/rottenbikes/cmd/api/email"
	"github.com/scardozos/rottenbikes/internal/domain"
)

func TestHandleVerifyLoginCode(t *testing.T) {
	var gotToken, gotCode string
	mockService := &MockService{
		VerifyLoginCodeFunc: func(ctx context.Context, pollToken, code string) (*domain.ConfirmResult, error) {
			gotToken, gotCode = pollToken, code
			switch {
			case pollToken == "boom":
				return nil, errors.New("db down")
			case pollToken == "good-poll" && code == "123456":
				return &domain.ConfirmResult{APIToken: "new-api-token", Email: "u@example.com", APITokenExpiresAt: time.Now().Add(time.Hour)}, nil
			default:
				return nil, domain.ErrInvalidLoginCode
			}
		},
	}

	srv, err := New(mockService, &email.NoopSender{}, ":8080")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	post := func(body string) *httptest.ResponseRecorder {
		gotToken, gotCode = "", ""
		req := httptest.NewRequest(http.MethodPost, "/auth/verify-code", strings.NewReader(body))
		w := httptest.NewRecorder()
		srv.server.Handler.ServeHTTP(w, req)
		return w
	}

	t.Run("right_code_returns_api_token", func(t *testing.T) {
		w := post(`{"token":"good-poll","code":"123456"}`)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body)
		}
		var resp map[string]any
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp["api_token"] != "new-api-token" || resp["email"] != "u@example.com" || resp["api_token_expires_at"] == nil {
			t.Errorf("expected the same shape as /auth/confirm, got %v", resp)
		}
	})

	t.Run("spaces_and_dashes_are_ignored", func(t *testing.T) {
		if w := post(`{"token":"good-poll","code":" 123-456 "}`); w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body)
		}
		if gotCode != "123456" {
			t.Errorf("expected the service to get 123456, got %q", gotCode)
		}
	})

	t.Run("wrong_code", func(t *testing.T) {
		w := post(`{"token":"good-poll","code":"000000"}`)
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "invalid or expired code") {
			t.Errorf("expected 400 invalid or expired code, got %d: %s", w.Code, w.Body)
		}
	})

	// Malformed codes are refused like wrong ones, without using up one of
	// the link's attempts.
	t.Run("malformed_code_is_not_sent_to_the_service", func(t *testing.T) {
		for _, code := range []string{"12345", "1234567", "12345a", "１２３４５６"} {
			w := post(`{"token":"good-poll","code":"` + code + `"}`)
			if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "invalid or expired code") {
				t.Errorf("code %q: expected 400 invalid or expired code, got %d: %s", code, w.Code, w.Body)
			}
			if gotToken != "" {
				t.Errorf("code %q: the service must not be called", code)
			}
		}
	})

	t.Run("missing_fields", func(t *testing.T) {
		for _, body := range []string{`{}`, `{"token":"good-poll"}`, `{"code":"123456"}`, `{"token":"good-poll","code":" "}`} {
			if w := post(body); w.Code != http.StatusBadRequest {
				t.Errorf("%s: expected 400, got %d", body, w.Code)
			}
		}
	})

	t.Run("invalid_body", func(t *testing.T) {
		if w := post(`not json`); w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("internal_error", func(t *testing.T) {
		if w := post(`{"token":"boom","code":"123456"}`); w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", w.Code)
		}
	})

	t.Run("wrong_method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/verify-code", nil)
		w := httptest.NewRecorder()
		srv.server.Handler.ServeHTTP(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", w.Code)
		}
	})

	// Polling used to hand the session to whoever requested the link.
	t.Run("poll_endpoint_is_gone", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/poll?token=good-poll", nil)
		w := httptest.NewRecorder()
		srv.server.Handler.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", w.Code)
		}
	})
}

func TestHandleRegister(t *testing.T) {
	t.Setenv("APP_ENV", "local") // skips hCaptcha when HCAPTCHA_SECRET unset

	mockService := &MockService{
		GetPosterByAPITokenFunc: func(ctx context.Context, token string) (*domain.AuthPoster, error) {
			return &domain.AuthPoster{PosterID: 1}, nil
		},
		CreateMagicLinkFunc: func(ctx context.Context, identifier string) (domain.MagicLink, string, error) {
			return domain.MagicLink{MagicToken: "login-magic-for-" + identifier, PollToken: "owner-poll-token", Code: "123456"}, identifier, nil
		},
		RegisterFunc: func(ctx context.Context, username, eml string) (domain.MagicLink, error) {
			if eml == "taken@example.com" {
				return domain.MagicLink{}, domain.ErrEmailExists
			}
			if eml == "baduser@example.com" {
				return domain.MagicLink{}, domain.ErrUsernameExists
			}
			if eml == "dbfail@example.com" {
				return domain.MagicLink{}, errors.New("db error")
			}
			return domain.MagicLink{MagicToken: "emailed-magic-token", PollToken: "client-poll-token", Code: "123456"}, nil
		},
	}

	sender := &recordingSender{}
	srv, err := New(mockService, sender, ":8080")
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
		srv.pendingEmails.Wait()
		mail := sender.last(t)
		if link := extractLink(t, mail.Body); link.Path != "/confirm/emailed-magic-token" {
			t.Errorf("unexpected confirm link %s", link)
		}
		if !strings.Contains(mail.Body, "123456") {
			t.Errorf("expected the login code in the welcome email, got %q", mail.Body)
		}
	})

	t.Run("missing_fields", func(t *testing.T) {
		w := post(map[string]string{"username": "x"})
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	// A taken email is answered like a new registration, so the endpoint does
	// not reveal which emails are registered. The owner gets a login link; the
	// requester gets a decoy poll token, never the owner's.
	t.Run("email_conflict_is_indistinguishable", func(t *testing.T) {
		w := post(map[string]string{
			"username":      "a",
			"email":         "taken@example.com",
			"captcha_token": "x",
		})
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body)
		}
		var resp map[string]string
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["message"] != "confirmation email sent" || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(resp["magic_token"]) {
			t.Errorf("expected the same shape as a real response, got %v", resp)
		}
		if resp["magic_token"] == "owner-poll-token" {
			t.Error("the requester must not get the owner's poll token")
		}
		again := map[string]string{}
		_ = json.Unmarshal(post(map[string]string{"username": "a", "email": "taken@example.com", "captcha_token": "x"}).Body.Bytes(), &again)
		if again["magic_token"] == resp["magic_token"] {
			t.Error("each request must get a fresh decoy")
		}
		srv.pendingEmails.Wait()
		mail := sender.last(t)
		if mail.To != "taken@example.com" || !strings.Contains(mail.Body, "already has an account") {
			t.Errorf("expected an existing-account email to the owner, got %+v", mail)
		}
		if link := extractLink(t, mail.Body); link.Path != "/confirm/login-magic-for-taken@example.com" {
			t.Errorf("unexpected login link %s", link)
		}
		if !strings.Contains(mail.Body, "123456") {
			t.Errorf("expected the login code in the existing-account email, got %q", mail.Body)
		}
	})

	t.Run("email_conflict_rate_limited_sends_nothing", func(t *testing.T) {
		mockService.CreateMagicLinkFunc = func(ctx context.Context, identifier string) (domain.MagicLink, string, error) {
			return domain.MagicLink{}, "", domain.ErrRateLimitExceeded
		}
		defer func() {
			mockService.CreateMagicLinkFunc = func(ctx context.Context, identifier string) (domain.MagicLink, string, error) {
				return domain.MagicLink{MagicToken: "login-magic-for-" + identifier, PollToken: "owner-poll-token", Code: "123456"}, identifier, nil
			}
		}()
		before := len(sender.sent)
		w := post(map[string]string{"username": "a", "email": "taken@example.com", "captcha_token": "x"})
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
		srv.pendingEmails.Wait()
		if len(sender.sent) != before {
			t.Error("no email should be sent when the daily limit is reached")
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
