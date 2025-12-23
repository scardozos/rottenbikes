package httpserver

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/scardozos/rottenbikes/cmd/api/email"
	"github.com/scardozos/rottenbikes/internal/domain"
)

func TestHandleCreateBikeReview(t *testing.T) {
	mockService := &MockService{
		GetPosterByAPITokenFunc: func(ctx context.Context, token string) (*domain.AuthPoster, error) {
			return &domain.AuthPoster{PosterID: 1}, nil
		},
		CreateReviewWithRatingsFunc: func(ctx context.Context, in domain.CreateReviewInput) (int64, error) {
			return 1, nil
		},
	}

	srv, err := New(mockService, &email.NoopSender{}, ":8080")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	t.Run("success", func(t *testing.T) {
		token := "valid_token"
		comment := "great bike"

		reqBody, _ := json.Marshal(map[string]interface{}{
			"comment": comment,
		})

		req := httptest.NewRequest(http.MethodPost, "/bikes/1/reviews", bytes.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d", w.Code)
		}
	})
}

func TestHandleUpdateReview(t *testing.T) {
	mockService := &MockService{
		GetPosterByAPITokenFunc: func(ctx context.Context, token string) (*domain.AuthPoster, error) {
			return &domain.AuthPoster{PosterID: 1}, nil
		},
		UpdateReviewWithRatingsFunc: func(ctx context.Context, in domain.UpdateReviewInput) error {
			return nil
		},
	}

	srv, err := New(mockService, &email.NoopSender{}, ":8080")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	t.Run("success", func(t *testing.T) {
		token := "valid_token"
		comment := "updated comment"

		reqBody, _ := json.Marshal(map[string]interface{}{
			"comment": comment,
		})

		req := httptest.NewRequest(http.MethodPut, "/reviews/1", bytes.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected status 204, got %d", w.Code)
		}
	})
}

func TestHandleGetReview(t *testing.T) {
	mockService := &MockService{
		GetReviewWithRatingsByIDFunc: func(ctx context.Context, reviewID int64) (*domain.ReviewWithRatings, error) {
			if reviewID == 1 {
				comment := "comment"
				bikeImg := "img.jpg"
				return &domain.ReviewWithRatings{
					ReviewID:        1,
					PosterID:        1,
					PosterUsername:  "user1",
					BikeNumericalID: 1,
					Comment:         &comment,
					CreatedAt:       time.Now(),
					Ratings: map[domain.RatingSubcategory]int16{
						"overall": 5,
					},
					BikeImg: &bikeImg,
				}, nil
			}
			return nil, sql.ErrNoRows
		},
	}

	srv, err := New(mockService, &email.NoopSender{}, ":8080")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/reviews/1", nil)
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})
}

func TestHandleDeleteReview(t *testing.T) {
	mockService := &MockService{
		GetPosterByAPITokenFunc: func(ctx context.Context, token string) (*domain.AuthPoster, error) {
			return &domain.AuthPoster{PosterID: 1}, nil
		},
		DeleteReviewFunc: func(ctx context.Context, reviewID int64, posterID int64) error {
			return nil
		},
	}

	srv, err := New(mockService, &email.NoopSender{}, ":8080")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	t.Run("success", func(t *testing.T) {
		token := "valid_token"

		req := httptest.NewRequest(http.MethodDelete, "/reviews/1", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected status 204, got %d", w.Code)
		}
	})
}
