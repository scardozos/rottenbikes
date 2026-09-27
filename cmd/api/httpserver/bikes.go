package httpserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/lib/pq"
	"github.com/rs/zerolog"
	"github.com/scardozos/rottenbikes/internal/domain"
)

// GET /bikes → list (now includes average_rating)
func (s *HTTPServer) handleListBikes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limitVal, offsetVal := parsePagination(r, -1, 100)

	searchQuery := r.URL.Query().Get("q")
	sortBy := r.URL.Query().Get("sort")

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	// Bikes created by E2E test accounts are only listed for test accounts,
	// so running the suite against a shared environment doesn't show them to
	// real users.
	viewer := s.optionalPoster(ctx, r)
	includeTest := viewer != nil && viewer.IsTest

	bikes, err := s.service.ListBikes(ctx, searchQuery, sortBy, limitVal, offsetVal, includeTest)
	if err != nil {
		zerolog.Ctx(r.Context()).Error().Err(err).Msg("list bikes error")
		s.sendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if bikes == nil {
		bikes = []domain.Bike{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(bikes); err != nil {
		zerolog.Ctx(r.Context()).Error().Err(err).Msg("encode bikes error")
	}
}

type createBikeRequest struct {
	NumericalID string  `json:"numerical_id"`
	HashID      *string `json:"hash_id"`
	IsElectric  bool    `json:"is_electric"`
	// WasScanned is client-declared: true when the creation flow started from
	// an actual QR scan of an unknown bike (there is nothing to verify an
	// unknown QR against server-side). Used as a moderation signal only.
	WasScanned bool `json:"was_scanned"`
}

// POST /bikes → create a bike
func (s *HTTPServer) handleCreateBike(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	creatorID, ok := posterIDFromContext(r.Context())
	if !ok {
		s.sendError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req createBikeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Kept as a string to preserve leading zeros. 4-5 digits for real bikes;
	// test accounts must use the reserved 6-digit range instead.
	isTest := isTestAccountFromContext(r.Context())
	if err := domain.ValidateNumericalID(req.NumericalID, isTest); err != nil {
		s.sendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	numericalID := req.NumericalID

	// Treat empty string hash_id as nil to allow multiple bikes without hash_ids
	if req.HashID != nil && *req.HashID == "" {
		req.HashID = nil
	}

	if req.HashID != nil && !isAlphanumeric(*req.HashID) {
		s.sendError(w, "hash_id must be alphanumeric", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	bike, err := s.service.CreateBike(ctx, numericalID, req.HashID, req.IsElectric, req.WasScanned, creatorID, isTest)
	if err != nil {
		if msg, ok := bikeConflictMessage(err); ok {
			s.sendError(w, msg, http.StatusConflict)
			return
		}
		if errors.Is(err, domain.ErrValidation) {
			s.sendError(w, err.Error(), http.StatusBadRequest)
			return
		}

		zerolog.Ctx(r.Context()).Error().Err(err).Msg("create bike error")
		s.sendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(bike)
}

type updateBikeRequest struct {
	NumericalID *int64  `json:"numerical_id"`
	HashID      *string `json:"hash_id"`
	IsElectric  *bool   `json:"is_electric"`
}

// PUT /bikes/{id} → update hash_id/is_electric
func (s *HTTPServer) handleUpdateBike(w http.ResponseWriter, r *http.Request) {
	bikeID := r.PathValue("id")
	if !isNumeric(bikeID) {
		s.sendError(w, "invalid bike id", http.StatusBadRequest)
		return
	}

	var req updateBikeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if req.NumericalID != nil {
		s.sendError(w, "numerical_id cannot be updated", http.StatusBadRequest)
		return
	}

	if req.HashID != nil && *req.HashID != "" && !isAlphanumeric(*req.HashID) {
		s.sendError(w, "hash_id must be alphanumeric", http.StatusBadRequest)
		return
	}

	posterID, ok := posterIDFromContext(r.Context())
	if !ok {
		s.sendError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if err := s.service.UpdateBike(ctx, bikeID, req.HashID, req.IsElectric, posterID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.sendError(w, "bike not found", http.StatusNotFound)
			return
		}
		if msg, ok := bikeConflictMessage(err); ok {
			s.sendError(w, msg, http.StatusConflict)
			return
		}
		if errors.Is(err, domain.ErrValidation) {
			s.sendError(w, err.Error(), http.StatusBadRequest)
			return
		}
		zerolog.Ctx(r.Context()).Error().Err(err).Str("bike_id", bikeID).Msg("update bike error")
		s.sendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GET /bikes/{id} → single bike
func (s *HTTPServer) handleGetBike(w http.ResponseWriter, r *http.Request) {
	bikeID := r.PathValue("id")
	if !isNumeric(bikeID) {
		s.sendError(w, "invalid bike id", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	bike, err := s.service.GetBike(ctx, bikeID)
	if err != nil {
		if err == sql.ErrNoRows {
			s.sendError(w, "bike not found", http.StatusNotFound)
			return
		}
		zerolog.Ctx(r.Context()).Error().Err(err).Str("bike_id", bikeID).Msg("get bike error")
		s.sendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(bike)
}

// GET /scan/{hash} → single bike looked up by its QR hash_id.
// Auth required: the lookup records a server-side scan event for the poster,
// which is what later proves a review came from an actual scan.
func (s *HTTPServer) handleGetBikeByHash(w http.ResponseWriter, r *http.Request) {
	hashID := r.PathValue("hash")
	if !isAlphanumeric(hashID) {
		s.sendError(w, "invalid hash id", http.StatusBadRequest)
		return
	}

	posterID, ok := posterIDFromContext(r.Context())
	if !ok {
		s.sendError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	bike, err := s.service.GetBikeByHash(ctx, hashID, posterID)
	if err != nil {
		if err == sql.ErrNoRows {
			s.sendError(w, "bike not found", http.StatusNotFound)
			return
		}
		zerolog.Ctx(r.Context()).Error().Err(err).Str("hash_id", hashID).Msg("get bike by hash error")
		s.sendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(bike)
}

// GET /bikes/{id}/details → single bike + ratings + reviews
func (s *HTTPServer) handleGetBikeDetails(w http.ResponseWriter, r *http.Request) {
	bikeID := r.PathValue("id")
	if !isNumeric(bikeID) {
		s.sendError(w, "invalid bike id", http.StatusBadRequest)
		return
	}

	limitVal, offsetVal := parsePagination(r, -1, 100)

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	details, err := s.service.GetBikeDetails(ctx, bikeID, limitVal, offsetVal)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.sendError(w, "bike not found", http.StatusNotFound)
			return
		}
		zerolog.Ctx(r.Context()).Error().Err(err).Str("bike_id", bikeID).Msg("get bike details error")
		s.sendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(details)
}

// bikeConflictMessage maps a unique violation on bikes to a client message.
func bikeConflictMessage(err error) (string, bool) {
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) || pqErr.Code != "23505" {
		return "", false
	}
	switch pqErr.Constraint {
	case "bikes_pkey":
		return "bike with this numerical_id already exists", true
	case "bikes_hash_id_key":
		return "bike with this hash_id already exists", true
	default:
		return "bike already exists (duplicate key)", true
	}
}

func isAlphanumeric(s string) bool {
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

func isNumeric(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// GET /bikes/{id}/reviews → list reviews for a bike
func (s *HTTPServer) handleListBikeReviews(w http.ResponseWriter, r *http.Request) {
	bikeID := r.PathValue("id")
	if !isNumeric(bikeID) {
		s.sendError(w, "invalid bike id", http.StatusBadRequest)
		return
	}

	limitVal, offsetVal := parsePagination(r, 20, 100)

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	reviews, err := s.service.ListReviewsWithRatingsByBike(ctx, bikeID, limitVal, offsetVal)
	if err != nil {
		zerolog.Ctx(r.Context()).Error().Err(err).Str("bike_id", bikeID).Msg("list bike reviews error")
		s.sendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if reviews == nil {
		reviews = []domain.ReviewWithRatings{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"reviews": reviews,
	})
}
