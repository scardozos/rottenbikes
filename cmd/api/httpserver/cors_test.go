package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/scardozos/rottenbikes/cmd/api/email"
)

func TestCORS(t *testing.T) {
	// CORS + /healthz never touch the service; an all-nil mock is safe and will
	// panic (failing the test) if a path unexpectedly calls into it.
	srv, err := New(&MockService{}, &email.NoopSender{}, ":8080")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	t.Run("allowed_origin_is_echoed_not_wildcard", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		req.Header.Set("Origin", "http://localhost:8081")
		w := httptest.NewRecorder()
		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:8081" {
			t.Errorf("expected ACAO to echo the allowed origin, got %q", got)
		}
		if got := w.Header().Get("Vary"); got != "Origin" {
			t.Errorf("expected Vary: Origin, got %q", got)
		}
	})

	t.Run("disallowed_origin_gets_no_acao", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		req.Header.Set("Origin", "https://evil.example")
		w := httptest.NewRecorder()
		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("request still succeeds (CORs only gates browsers), got %d", w.Code)
		}
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("disallowed origin must not receive ACAO, got %q", got)
		}
	})

	t.Run("preflight_allowed_origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/bikes", nil)
		req.Header.Set("Origin", "http://localhost:8081")
		req.Header.Set("Access-Control-Request-Method", "POST")
		w := httptest.NewRecorder()
		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected 204 for allowed preflight, got %d", w.Code)
		}
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:8081" {
			t.Errorf("expected echoed ACAO, got %q", got)
		}
		if got := w.Header().Get("Access-Control-Allow-Headers"); got == "" {
			t.Error("expected Allow-Headers on preflight")
		}
	})

	t.Run("preflight_disallowed_origin_rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/bikes", nil)
		req.Header.Set("Origin", "https://evil.example")
		w := httptest.NewRecorder()
		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403 for disallowed preflight, got %d", w.Code)
		}
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("disallowed preflight must not receive ACAO, got %q", got)
		}
	})

	t.Run("no_origin_same_origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()
		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("same-origin request must not receive ACAO, got %q", got)
		}
	})

	t.Run("custom_env_overrides_defaults", func(t *testing.T) {
		t.Setenv("CORS_ALLOWED_ORIGINS", "https://rottenbik.es")

		// configured origin is allowed
		req := httptest.NewRequest(http.MethodOptions, "/bikes", nil)
		req.Header.Set("Origin", "https://rottenbik.es")
		w := httptest.NewRecorder()
		srv.server.Handler.ServeHTTP(w, req)
		if w.Code != http.StatusNoContent {
			t.Errorf("expected configured origin allowed (204), got %d", w.Code)
		}

		// default dev origin is no longer allowed once env is set
		req2 := httptest.NewRequest(http.MethodOptions, "/bikes", nil)
		req2.Header.Set("Origin", "http://localhost:8081")
		w2 := httptest.NewRecorder()
		srv.server.Handler.ServeHTTP(w2, req2)
		if w2.Code != http.StatusForbidden {
			t.Errorf("expected default origin rejected when env overrides (403), got %d", w2.Code)
		}
	})
}
