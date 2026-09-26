package httpserver

import (
	"net/http"
	"os"
	"strings"
)

// Admin authorization: the authenticated poster's email must be listed in the
// ADMIN_EMAILS environment variable (comma-separated, trimmed, lowercased).
// The env is read per request (like CORS_ALLOWED_ORIGINS), so removing an
// email revokes admin access immediately without a restart.
func adminEmails() []string {
	raw := strings.TrimSpace(os.Getenv("ADMIN_EMAILS"))
	if raw == "" {
		return nil
	}
	var out []string
	for _, e := range strings.Split(raw, ",") {
		if e = strings.TrimSpace(strings.ToLower(e)); e != "" {
			out = append(out, e)
		}
	}
	return out
}

func isAdminEmail(email string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return false
	}
	for _, a := range adminEmails() {
		if a == email {
			return true
		}
	}
	return false
}

// middlewareAdminAuth = middlewareAuth + admin allowlist check.
// Must wrap the handler after auth so the poster email is in the context.
func (s *HTTPServer) middlewareAdminAuth(next http.Handler) http.Handler {
	return s.middlewareAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		email, _ := emailFromContext(r.Context())
		if !isAdminEmail(email) {
			s.sendError(w, "forbidden: admin access required", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}))
}
