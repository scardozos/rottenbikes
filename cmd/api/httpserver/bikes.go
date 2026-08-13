package httpserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
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

	limitVal := -1
	offsetVal := -1
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val >= 0 {
			limitVal = val
			if limitVal > 100 {
				limitVal = 100 // Cap to prevent abuse
			}
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if val, err := strconv.Atoi(offsetStr); err == nil && val >= 0 {
			offsetVal = val
		}
	}

	searchQuery := r.URL.Query().Get("q")
	sortBy := r.URL.Query().Get("sort")

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	bikes, err := s.service.ListBikes(ctx, searchQuery, sortBy, limitVal, offsetVal)
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

	// Validate numerical_id consistency (4-5 digits)
	// We don't parse to int64 anymore to preserve leading zeros, but strictly check format.
	// Allow 0-9, length 4-5.
	validID := true
	if len(req.NumericalID) < 4 || len(req.NumericalID) > 5 {
		validID = false
	} else {
		for _, r := range req.NumericalID {
			if r < '0' || r > '9' {
				validID = false
				break
			}
		}
	}

	if !validID {
		s.sendError(w, "numerical_id must be 4-5 digits", http.StatusBadRequest)
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

	bike, err := s.service.CreateBike(ctx, numericalID, req.HashID, req.IsElectric, creatorID)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			// Distinguish by constraint/index name
			switch pqErr.Constraint {
			case "bikes_pkey":
				s.sendError(w, "bike with this numerical_id already exists", http.StatusConflict)
				return
			case "bikes_hash_id_key":
				s.sendError(w, "bike with this hash_id already exists", http.StatusConflict)
				return
			default:
				s.sendError(w, "bike already exists (duplicate key)", http.StatusConflict)
				return
			}
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
func (s *HTTPServer) handleUpdateBike(w http.ResponseWriter, r *http.Request, bikeID string) {
	if r.Method != http.MethodPut {
		s.sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

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
		zerolog.Ctx(r.Context()).Error().Err(err).Str("bike_id", bikeID).Msg("update bike error")
		s.sendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GET /bikes/{id} → single bike
func (s *HTTPServer) handleGetBike(w http.ResponseWriter, r *http.Request, bikeID string) {
	if r.Method != http.MethodGet {
		s.sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

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

// DELETE /bikes/{id}
func (s *HTTPServer) handleDeleteBike(w http.ResponseWriter, r *http.Request, bikeID string) {
	if r.Method != http.MethodDelete {
		s.sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !isNumeric(bikeID) {
		s.sendError(w, "invalid bike id", http.StatusBadRequest)
		return
	}

	posterID, ok := posterIDFromContext(r.Context())
	if !ok {
		s.sendError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if err := s.service.DeleteBike(ctx, bikeID, posterID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.sendError(w, "bike not found", http.StatusNotFound)
			return
		}
		zerolog.Ctx(r.Context()).Error().Err(err).Str("bike_id", bikeID).Msg("delete bike error")
		s.sendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GET /bikes/{id}/details → single bike + ratings + reviews
func (s *HTTPServer) handleGetBikeDetails(w http.ResponseWriter, r *http.Request, bikeID string) {
	if r.Method != http.MethodGet {
		s.sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !isNumeric(bikeID) {
		s.sendError(w, "invalid bike id", http.StatusBadRequest)
		return
	}

	limitVal := -1
	offsetVal := -1
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val >= 0 {
			limitVal = val
			if limitVal > 100 {
				limitVal = 100 // Cap to prevent abuse
			}
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if val, err := strconv.Atoi(offsetStr); err == nil && val >= 0 {
			offsetVal = val
		}
	}

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
func (s *HTTPServer) handleListBikeReviews(w http.ResponseWriter, r *http.Request, bikeID string) {
	if r.Method != http.MethodGet {
		s.sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !isNumeric(bikeID) {
		s.sendError(w, "invalid bike id", http.StatusBadRequest)
		return
	}

	limitVal := 20
	offsetVal := 0
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			limitVal = val
			if limitVal > 100 {
				limitVal = 100 // Cap to prevent abuse
			}
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if val, err := strconv.Atoi(offsetStr); err == nil && val >= 0 {
			offsetVal = val
		}
	}

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
