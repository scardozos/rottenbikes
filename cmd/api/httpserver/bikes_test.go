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

func TestHandleListBikes(t *testing.T) {
	mockService := &MockService{
		ListBikesFunc: func(ctx context.Context) ([]domain.Bike, error) {
			return []domain.Bike{
				{NumericalID: 1, HashID: "hash1", IsElectric: true},
				{NumericalID: 2, HashID: "hash2", IsElectric: false},
			}, nil
		},
		GetPosterByAPITokenFunc: func(ctx context.Context, token string) (*domain.AuthPoster, error) {
			return &domain.AuthPoster{PosterID: 1}, nil
		},
	}

	srv, err := New(mockService, &email.NoopSender{}, ":8080")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/bikes", nil)
		req.Header.Set("Authorization", "Bearer valid_token")
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})
}

func TestHandleCreateBike(t *testing.T) {
	mockService := &MockService{
		GetPosterByAPITokenFunc: func(ctx context.Context, token string) (*domain.AuthPoster, error) {
			return &domain.AuthPoster{
				PosterID: 1,
				Email:    "test@example.com",
			}, nil
		},
		CreateBikeFunc: func(ctx context.Context, numericalID int64, hashID *string, isElectric bool, creatorID int64) (*domain.Bike, error) {
			return &domain.Bike{
				NumericalID: numericalID,
				HashID:      *hashID,
				IsElectric:  isElectric,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}, nil
		},
	}

	srv, err := New(mockService, &email.NoopSender{}, ":8080")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	t.Run("success", func(t *testing.T) {
		token := "valid_token"
		numericalID := int64(123)
		hashID := "hash_123"
		isElectric := true

		reqBody, _ := json.Marshal(map[string]interface{}{
			"numerical_id": numericalID,
			"hash_id":      hashID,
			"is_electric":  isElectric,
		})

		req := httptest.NewRequest(http.MethodPost, "/bikes", bytes.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d", w.Code)
		}
	})
}

func TestHandleGetBike(t *testing.T) {
	mockService := &MockService{
		GetBikeFunc: func(ctx context.Context, id int64) (*domain.Bike, error) {
			if id == 1 {
				return &domain.Bike{NumericalID: 1, HashID: "hash1", IsElectric: true}, nil
			}
			return nil, sql.ErrNoRows
		},
		GetPosterByAPITokenFunc: func(ctx context.Context, token string) (*domain.AuthPoster, error) {
			return &domain.AuthPoster{PosterID: 1}, nil
		},
	}

	srv, err := New(mockService, &email.NoopSender{}, ":8080")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/bikes/1", nil)
		req.Header.Set("Authorization", "Bearer valid_token")
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})
}

func TestHandleUpdateBike(t *testing.T) {
	mockService := &MockService{
		GetPosterByAPITokenFunc: func(ctx context.Context, token string) (*domain.AuthPoster, error) {
			return &domain.AuthPoster{PosterID: 1}, nil
		},
		UpdateBikeFunc: func(ctx context.Context, id int64, hashID *string, isElectric *bool) error {
			return nil
		},
	}

	srv, err := New(mockService, &email.NoopSender{}, ":8080")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	t.Run("success", func(t *testing.T) {
		token := "valid_token"
		hashID := "new_hash"
		reqBody, _ := json.Marshal(map[string]interface{}{
			"hash_id": hashID,
		})

		req := httptest.NewRequest(http.MethodPut, "/bikes/1", bytes.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected status 204, got %d", w.Code)
		}
	})

	t.Run("prevent_numerical_id_update", func(t *testing.T) {
		token := "valid_token"
		reqBody, _ := json.Marshal(map[string]interface{}{
			"numerical_id": 123,
		})

		req := httptest.NewRequest(http.MethodPut, "/bikes/1", bytes.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})
}
