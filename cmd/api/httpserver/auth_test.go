package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/scardozos/rottenbikes/cmd/api/email"
	"github.com/scardozos/rottenbikes/internal/domain"
)

func TestHandleRequestMagicLink(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	mockService := &MockService{
		RegisterFunc: func(ctx context.Context, username, email string) (domain.MagicLink, error) {
			return domain.MagicLink{MagicToken: "magic-token-for-" + email, PollToken: "poll-token-for-" + email, Code: "123456"}, nil
		},
		CreateMagicLinkFunc: func(ctx context.Context, identifier string) (domain.MagicLink, string, error) {
			if identifier == "test@example.com" || identifier == "testuser" {
				return domain.MagicLink{MagicToken: "magic-token-for-" + identifier, PollToken: "poll-token-for-" + identifier, Code: "123456"}, "test@example.com", nil
			}
			return domain.MagicLink{}, "", domain.ErrUserNotFound
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

	// Unknown accounts get the same response as known ones (with a decoy poll
	// token), so the endpoint does not reveal which emails are registered.
	t.Run("user_not_found_is_indistinguishable", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]string{
			"email":         "nonexistent@example.com",
			"captcha_token": "valid-captcha",
		})

		req := httptest.NewRequest(http.MethodPost, "/auth/request-magic-link", bytes.NewReader(reqBody))
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
		var resp map[string]string
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp["message"] != "magic link email sent" || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(resp["magic_token"]) {
			t.Errorf("expected the same shape as a real response, got %v", resp)
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

		// The opaque credential in the response is the poll token (raw), NOT the
		// hash of the emailed magic token. The requester polls with this.
		expectedPollToken := "poll-token-for-" + username
		if resp["magic_token"] != expectedPollToken {
			t.Errorf("expected magic_token (poll token) %s, got %s", expectedPollToken, resp["magic_token"])
		}
		// The poll token must NOT equal the hash of the emailed magic token (the
		// old, vulnerable behavior). That hash should not even appear here.
		if resp["magic_token"] == domain.HashToken("magic-token-for-"+username) {
			t.Errorf("poll token must not be deriveable from the emailed magic token")
		}
	})

	t.Run("rate_limit_exceeded", func(t *testing.T) {
		mockService.CreateMagicLinkFunc = func(ctx context.Context, identifier string) (domain.MagicLink, string, error) {
			return domain.MagicLink{}, "", domain.ErrRateLimitExceeded
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

func TestDecoyPollToken(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		tok := decoyPollToken()
		if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(tok) {
			t.Fatalf("decoy %q does not look like a real poll token (64 hex chars)", tok)
		}
		if seen[tok] {
			t.Fatalf("decoy %q repeated: decoys must be random, not a fixed or reused value", tok)
		}
		seen[tok] = true
	}
}

// Every unknown-account request gets its own decoy: a fixed or reused value
// would make fake responses recognizable.
func TestUnknownAccountDecoysDiffer(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	t.Setenv("HCAPTCHA_SECRET", "")
	svc := &MockService{
		CreateMagicLinkFunc: func(ctx context.Context, identifier string) (domain.MagicLink, string, error) {
			return domain.MagicLink{}, "", domain.ErrUserNotFound
		},
	}
	srv, _ := New(svc, &email.NoopSender{}, ":8080")

	tokens := map[string]bool{}
	for i := 0; i < 2; i++ {
		body, _ := json.Marshal(map[string]string{"email": "ghost@example.com", "captcha_token": "x"})
		w := httptest.NewRecorder()
		srv.server.Handler.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/auth/request-magic-link", bytes.NewReader(body)))
		var resp map[string]string
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		tokens[resp["magic_token"]] = true
	}
	if len(tokens) != 2 {
		t.Errorf("expected two different decoys for two requests, got %v", tokens)
	}
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

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", w.Code)
		}
	})
}

func TestHandleVerifyToken(t *testing.T) {
	mockService := &MockService{
		GetPosterByAPITokenFunc: func(ctx context.Context, token string) (*domain.AuthPoster, error) {
			if token == "valid-token" {
				return &domain.AuthPoster{PosterID: 123}, nil
			}
			return nil, domain.ErrInvalidToken
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

func TestHandleLogout(t *testing.T) {
	var revoked []string
	revokeErr := error(nil)
	mockService := &MockService{
		GetPosterByAPITokenFunc: func(ctx context.Context, token string) (*domain.AuthPoster, error) {
			t.Error("logout must not require a valid session")
			return nil, domain.ErrInvalidToken
		},
		RevokeAPITokenFunc: func(ctx context.Context, token string) error {
			revoked = append(revoked, token)
			return revokeErr
		},
	}

	srv, err := New(mockService, &email.NoopSender{}, ":8080")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	logout := func(authHeader string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
		if authHeader != "" {
			req.Header.Set("Authorization", authHeader)
		}
		w := httptest.NewRecorder()
		srv.server.Handler.ServeHTTP(w, req)
		return w
	}

	t.Run("revokes the session", func(t *testing.T) {
		revoked = nil
		if w := logout("Bearer active-token"); w.Code != http.StatusNoContent {
			t.Errorf("expected status 204, got %d", w.Code)
		}
		if len(revoked) != 1 || revoked[0] != "active-token" {
			t.Errorf("expected active-token to be revoked, got %v", revoked)
		}
	})

	// A dead session (already revoked, expired, or deleted with its account)
	// is still logged out: 204, never 401.
	t.Run("already invalid token is still 204", func(t *testing.T) {
		revoked = nil
		if w := logout("Bearer dead-token"); w.Code != http.StatusNoContent {
			t.Errorf("expected status 204, got %d", w.Code)
		}
		if len(revoked) != 1 {
			t.Errorf("expected a revoke attempt, got %v", revoked)
		}
	})

	for name, header := range map[string]string{
		"missing token": "",
		"wrong scheme":  "Basic dXNlcjpwYXNz",
		"empty bearer":  "Bearer ",
	} {
		t.Run(name+" is 204 without revoking", func(t *testing.T) {
			revoked = nil
			if w := logout(header); w.Code != http.StatusNoContent {
				t.Errorf("expected status 204, got %d", w.Code)
			}
			if len(revoked) != 0 {
				t.Errorf("nothing should be revoked, got %v", revoked)
			}
		})
	}

	t.Run("revoke failure is 500", func(t *testing.T) {
		revokeErr = errors.New("db down")
		defer func() { revokeErr = nil }()
		if w := logout("Bearer active-token"); w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
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
