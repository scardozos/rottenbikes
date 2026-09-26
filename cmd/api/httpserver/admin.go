package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/rs/zerolog"

	"github.com/scardozos/rottenbikes/internal/domain"
)

// GET /admin/users?q= → search posters by email/username (admin only).
// Returns each poster with their review/bike counts so the admin can see the
// blast radius before purging: deleting a poster's bikes cascades to every
// review on those bikes.
func (s *HTTPServer) handleAdminSearchPosters(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	limit, _ := parsePagination(r, 20, 50)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	posters, err := s.service.SearchPosters(ctx, query, limit)
	if err != nil {
		zerolog.Ctx(r.Context()).Error().Err(err).Str("q", query).Msg("admin search posters error")
		s.sendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if posters == nil {
		posters = []domain.PosterSummary{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(posters); err != nil {
		zerolog.Ctx(r.Context()).Error().Err(err).Msg("encode posters error")
	}
}

// DELETE /admin/users/{id} → purge a malicious poster and all their content
// (reviews, ratings, created bikes, sessions) in one transaction, with an
// audit row. Deleting their bikes cascades to other posters' reviews on those
// bikes. Normal users are unaffected: self-service deletion keeps its current
// behavior (content kept, attribution orphaned).
func (s *HTTPServer) handleAdminPurgePoster(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	targetID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.sendError(w, "invalid poster id", http.StatusBadRequest)
		return
	}

	adminID, ok := posterIDFromContext(r.Context())
	if !ok {
		s.sendError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// The admin already has a self-service delete path with the same effect.
	if targetID == adminID {
		s.sendError(w, "cannot purge yourself; use account deletion instead", http.StatusBadRequest)
		return
	}

	// Purges can touch many bikes/reviews (aggregate recompute), so allow a
	// longer timeout than regular requests.
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if err := s.service.PurgePoster(ctx, adminID, targetID); err != nil {
		zerolog.Ctx(r.Context()).Error().Err(err).Int64("target_poster_id", targetID).Msg("admin purge poster error")
		s.sendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	zerolog.Ctx(r.Context()).Info().Int64("admin_poster_id", adminID).Int64("target_poster_id", targetID).Msg("poster purged by admin")
	w.WriteHeader(http.StatusNoContent)
}
