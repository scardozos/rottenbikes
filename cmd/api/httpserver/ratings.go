package httpserver

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// GET /bikes/ratings → rating aggregates for all bikes
func (s *HTTPServer) handleListAllBikeRatings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	aggs, err := s.service.ListRatingAggregates(ctx)
	if err != nil {
		log.Printf("list all bike ratings error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(aggs); err != nil {
		log.Printf("encode all bike ratings error: %v", err)
	}
}

// GET /bikes/{id}/ratings → rating aggregates for a single bike
func (s *HTTPServer) handleBikeRatings(w http.ResponseWriter, r *http.Request, bikeID int64) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	aggs, err := s.service.ListRatingAggregatesByBike(ctx, bikeID)
	if err != nil {
		log.Printf("list bike %d ratings error: %v", bikeID, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(aggs); err != nil {
		log.Printf("encode bike %d ratings error: %v", bikeID, err)
	}
}
