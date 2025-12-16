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
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req createReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid JSON"))
		return
	}

	posterID, ok := posterIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("unauthorized"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	reviewID, err := domain.CreateReviewWithRatings(ctx, s.db, domain.CreateReviewInput{
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
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte("you can only review this bike every 10 minutes"))
			return
		}

		log.Printf("create review error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"review_id": reviewID,
	})
}

// GET /bikes/reviews → all bikes' reviews with ratings
func (s *HTTPServer) handleListAllReviewsWithRatings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	reviews, err := domain.ListReviewsWithRatings(ctx, s.db)
	if err != nil {
		log.Printf("list all reviews error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(reviews); err != nil {
		log.Printf("encode all reviews error: %v", err)
	}
}

// GET /bikes/{id}/reviews → reviews with ratings for a single bike
func (s *HTTPServer) handleBikeReviews(w http.ResponseWriter, r *http.Request, bikeID int64) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	reviews, err := domain.ListReviewsWithRatingsByBike(ctx, s.db, bikeID)
	if err != nil {
		log.Printf("list bike %d reviews error: %v", bikeID, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(reviews); err != nil {
		log.Printf("encode bike %d reviews error: %v", bikeID, err)
	}
}

// PUT /reviews/{id}
func (s *HTTPServer) handleUpdateReview(w http.ResponseWriter, r *http.Request, reviewID int64) {
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req createReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid JSON"))
		return
	}

	posterID, ok := posterIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("unauthorized"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	err := domain.UpdateReviewWithRatings(ctx, s.db, domain.UpdateReviewInput{
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
			w.WriteHeader(http.StatusNotFound)
			return
		}
		log.Printf("update review %d error: %v", reviewID, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GET /reviews/{id} → single review with ratings
func (s *HTTPServer) handleGetReview(w http.ResponseWriter, r *http.Request, reviewID int64) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	review, err := domain.GetReviewWithRatingsByID(ctx, s.db, reviewID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		log.Printf("get review %d error: %v", reviewID, err)
		w.WriteHeader(http.StatusInternalServerError)
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
		w.WriteHeader(http.StatusMethodNotAllowed)
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

	if err := domain.DeleteReview(ctx, s.db, reviewID, posterID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		log.Printf("delete review %d error: %v", reviewID, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
