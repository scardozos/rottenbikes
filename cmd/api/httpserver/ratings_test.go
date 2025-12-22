package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/scardozos/rottenbikes/cmd/api/email"
	"github.com/scardozos/rottenbikes/internal/domain"
)

func TestHandleListAllBikeRatings(t *testing.T) {
	mockService := &MockService{
		ListRatingAggregatesFunc: func(ctx context.Context) ([]domain.RatingAggregate, error) {
			return []domain.RatingAggregate{
				{BikeNumericalID: 1, Subcategory: "overall", AverageRating: 4.5},
				{BikeNumericalID: 1, Subcategory: "breaks", AverageRating: 4.0},
			}, nil
		},
	}

	srv, err := New(mockService, &email.NoopSender{}, ":8080")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/bikes/ratings", nil)
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})
}

func TestHandleBikeRatings(t *testing.T) {
	mockService := &MockService{
		ListRatingAggregatesByBikeFunc: func(ctx context.Context, bikeID int64) ([]domain.RatingAggregate, error) {
			return []domain.RatingAggregate{
				{BikeNumericalID: 1, Subcategory: "overall", AverageRating: 4.5},
			}, nil
		},
	}

	srv, err := New(mockService, &email.NoopSender{}, ":8080")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/bikes/1/ratings", nil)
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})
}
