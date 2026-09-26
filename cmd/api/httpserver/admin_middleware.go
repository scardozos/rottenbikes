package httpserver

import (
	"net/http"

	"github.com/scardozos/rottenbikes/internal/domain"
)

// Admin authorization: the authenticated poster must have the 'admin' role
// (posters.role). Roles are managed out-of-band via cmd/adminctl, so no admin
// identity ever lives in git or env files — and since the role
// is read from the database on every token lookup, promote/demote takes effect
// immediately without a restart or redeploy.
func (s *HTTPServer) middlewareAdminAuth(next http.Handler) http.Handler {
	return s.middlewareAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, _ := roleFromContext(r.Context())
		if role != domain.PosterRoleAdmin {
			s.sendError(w, "forbidden: admin access required", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}))
}
