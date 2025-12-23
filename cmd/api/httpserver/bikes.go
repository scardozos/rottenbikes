package httpserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/lib/pq"
)

// GET /bikes → list (now includes average_rating)
func (s *HTTPServer) handleListBikes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	bikes, err := s.service.ListBikes(ctx)
	if err != nil {
		log.Printf("list bikes error: %v", err)
		s.sendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(bikes); err != nil {
		log.Printf("encode bikes error: %v", err)
	}
}

type createBikeRequest struct {
	NumericalID int64   `json:"numerical_id"`
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

	if req.NumericalID == 0 {
		s.sendError(w, "numerical_id is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	bike, err := s.service.CreateBike(ctx, req.NumericalID, req.HashID, req.IsElectric, creatorID)
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

		log.Printf("create bike error: %v", err)
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
func (s *HTTPServer) handleUpdateBike(w http.ResponseWriter, r *http.Request, bikeID int64) {
	if r.Method != http.MethodPut {
		s.sendError(w, "method not allowed", http.StatusMethodNotAllowed)
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

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if err := s.service.UpdateBike(ctx, bikeID, req.HashID, req.IsElectric); err != nil {
		log.Printf("update bike %d error: %v", bikeID, err)
		s.sendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GET /bikes/{id} → single bike
func (s *HTTPServer) handleGetBike(w http.ResponseWriter, r *http.Request, bikeID int64) {
	if r.Method != http.MethodGet {
		s.sendError(w, "method not allowed", http.StatusMethodNotAllowed)
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
		log.Printf("get bike %d error: %v", bikeID, err)
		s.sendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(bike)
}

// DELETE /bikes/{id}
func (s *HTTPServer) handleDeleteBike(w http.ResponseWriter, r *http.Request, bikeID int64) {
	if r.Method != http.MethodDelete {
		s.sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if err := s.service.DeleteBike(ctx, bikeID); err != nil {
		log.Printf("delete bike %d error: %v", bikeID, err)
		s.sendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GET /bikes/{id}/details → single bike + ratings + reviews
func (s *HTTPServer) handleGetBikeDetails(w http.ResponseWriter, r *http.Request, bikeID int64) {
	if r.Method != http.MethodGet {
		s.sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	details, err := s.service.GetBikeDetails(ctx, bikeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.sendError(w, "bike not found", http.StatusNotFound)
			return
		}
		log.Printf("get bike details %d error: %v", bikeID, err)
		s.sendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(details)
}
