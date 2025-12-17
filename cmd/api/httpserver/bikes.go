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

func (s *HTTPServer) handleListBikes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	bikes, err := s.service.ListBikes(ctx)
	if err != nil {
		log.Printf("list bikes error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
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
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	creatorID, ok := posterIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("unauthorized"))
		return
	}

	var req createBikeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid JSON"))
		return
	}

	if req.NumericalID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("numerical_id is required"))
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
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte("bike with this numerical_id already exists"))
				return
			case "bikes_hash_id_key":
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte("bike with this hash_id already exists"))
				return
			default:
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte("bike already exists (duplicate key)"))
				return
			}
		}

		log.Printf("create bike error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
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
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req updateBikeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid JSON"))
		return
	}

	if req.NumericalID != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("numerical_id cannot be updated"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if err := s.service.UpdateBike(ctx, bikeID, req.HashID, req.IsElectric); err != nil {
		log.Printf("update bike %d error: %v", bikeID, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GET /bikes/{id} → single bike
func (s *HTTPServer) handleGetBike(w http.ResponseWriter, r *http.Request, bikeID int64) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	bike, err := s.service.GetBike(ctx, bikeID)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		log.Printf("get bike %d error: %v", bikeID, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(bike)
}

// DELETE /bikes/{id}
func (s *HTTPServer) handleDeleteBike(w http.ResponseWriter, r *http.Request, bikeID int64) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if err := s.service.DeleteBike(ctx, bikeID); err != nil {
		log.Printf("delete bike %d error: %v", bikeID, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
