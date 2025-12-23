package httpserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/scardozos/rottenbikes/internal/domain"
)

type createReviewRequest struct {
	PosterID int64   `json:"poster_id"` // ignored/overridden by auth
	Comment  *string `json:"comment"`
	BikeImg  *string `json:"bike_img"`

	Overall    *int16 `json:"overall"`
	Breaks     *int16 `json:"breaks"`
	Seat       *int16 `json:"seat"`
	Sturdiness *int16 `json:"sturdiness"`
	Power      *int16 `json:"power"`
	Pedals     *int16 `json:"pedals"`
}

// POST /bikes/{id}/reviews → create a review with optional subcategory ratings
func (s *HTTPServer) handleCreateBikeReview(w http.ResponseWriter, r *http.Request, bikeID int64) {
	if r.Method != http.MethodPost {
		s.sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req createReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	posterID, ok := posterIDFromContext(r.Context())
	if !ok {
		s.sendError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	reviewID, err := s.service.CreateReviewWithRatings(ctx, domain.CreateReviewInput{
		PosterID:   posterID,
		BikeID:     bikeID,
		Comment:    req.Comment,
		BikeImg:    req.BikeImg,
		Overall:    req.Overall,
		Breaks:     req.Breaks,
		Seat:       req.Seat,
		Sturdiness: req.Sturdiness,
		Power:      req.Power,
		Pedals:     req.Pedals,
	})
	if err != nil {
		if errors.Is(err, domain.ErrTooFrequentReview) {
			s.sendError(w, "you can only review this bike every 10 minutes", http.StatusTooManyRequests)
			return
		}

		log.Printf("create review error: %v", err)
		s.sendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"review_id": reviewID,
	})
}

// PUT /reviews/{id}
func (s *HTTPServer) handleUpdateReview(w http.ResponseWriter, r *http.Request, reviewID int64) {
	if r.Method != http.MethodPut {
		s.sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req createReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	posterID, ok := posterIDFromContext(r.Context())
	if !ok {
		s.sendError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	err := s.service.UpdateReviewWithRatings(ctx, domain.UpdateReviewInput{
		ReviewID:   reviewID,
		PosterID:   posterID,
		Comment:    req.Comment,
		BikeImg:    req.BikeImg,
		Overall:    req.Overall,
		Breaks:     req.Breaks,
		Seat:       req.Seat,
		Sturdiness: req.Sturdiness,
		Power:      req.Power,
		Pedals:     req.Pedals,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.sendError(w, "review not found", http.StatusNotFound)
			return
		}
		log.Printf("update review %d error: %v", reviewID, err)
		s.sendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GET /reviews/{id} → single review with ratings
func (s *HTTPServer) handleGetReview(w http.ResponseWriter, r *http.Request, reviewID int64) {
	if r.Method != http.MethodGet {
		s.sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	review, err := s.service.GetReviewWithRatingsByID(ctx, reviewID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.sendError(w, "review not found", http.StatusNotFound)
			return
		}
		log.Printf("get review %d error: %v", reviewID, err)
		s.sendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(review); err != nil {
		log.Printf("encode review %d error: %v", reviewID, err)
	}
}

type deleteReviewRequest struct {
	PosterID int64 `json:"poster_id"` // ignored; auth used instead
}

// DELETE /reviews/{id}
func (s *HTTPServer) handleDeleteReview(w http.ResponseWriter, r *http.Request, reviewID int64) {
	if r.Method != http.MethodDelete {
		s.sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	posterID, ok := posterIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("unauthorized"))
		return
	}

	// Body is not needed anymore, but accept and ignore to stay backward compatible.
	_ = json.NewDecoder(r.Body).Decode(&deleteReviewRequest{})

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := s.service.DeleteReview(ctx, reviewID, posterID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.sendError(w, "review not found", http.StatusNotFound)
			return
		}
		log.Printf("delete review %d error: %v", reviewID, err)
		s.sendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
