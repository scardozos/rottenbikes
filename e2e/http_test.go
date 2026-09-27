//go:build e2e

package e2e

import (
	"net/http"
	"testing"
)

func TestMethodNotAllowedIsJSON(t *testing.T) {
	for _, c := range []struct{ method, path string }{
		{"PATCH", "/bikes"},
		{"POST", "/reviews/1"},
		{"GET", "/auth/register"},
	} {
		r := call(t, c.method, c.path, "", nil)
		if expectStatus(t, r, http.StatusMethodNotAllowed) {
			expectJSONError(t, r)
		}
	}
}

func TestCORS(t *testing.T) {
	t.Run("allowed origin", func(t *testing.T) {
		if corsOrigin == "" {
			t.Skip("E2E_CORS_ORIGIN not set")
		}
		r := call(t, "OPTIONS", "/bikes", "", nil,
			withHeader("Origin", corsOrigin),
			withHeader("Access-Control-Request-Method", "POST"))
		expectStatus(t, r, http.StatusNoContent)
		if got := r.Header.Get("Access-Control-Allow-Origin"); got != corsOrigin {
			t.Errorf("allowed origin not echoed, got %q", got)
		}
	})

	r := call(t, "OPTIONS", "/bikes", "", nil, withHeader("Origin", "https://evil.example"))
	expectStatus(t, r, http.StatusForbidden)
	if got := r.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("disallowed origin got Access-Control-Allow-Origin %q", got)
	}

	r = call(t, "GET", "/bikes?limit=1", "", nil, withHeader("Origin", "https://evil.example"))
	if expectStatus(t, r, http.StatusOK) && r.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("disallowed origin got CORS headers on GET")
	}
}
