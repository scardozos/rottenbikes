package httpserver

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/scardozos/rottenbikes/cmd/api/email"
	"github.com/scardozos/rottenbikes/internal/domain"
)

func strPtr(s string) *string {
	return &s
}

func TestHandleListBikes(t *testing.T) {
	mockService := &MockService{
		ListBikesFunc: func(ctx context.Context) ([]domain.Bike, error) {
			return []domain.Bike{
				{NumericalID: 1, HashID: strPtr("hash1"), IsElectric: true},
				{NumericalID: 2, HashID: strPtr("hash2"), IsElectric: false},
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
				HashID:      hashID,
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
		// numericalID := int64(123)
		numericalIDStr := "0123"
		hashID := "hash123"
		isElectric := true

		reqBody, _ := json.Marshal(map[string]interface{}{
			"numerical_id": numericalIDStr,
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

	t.Run("bad_request_invalid_json", func(t *testing.T) {
		token := "valid_token"
		req := httptest.NewRequest(http.MethodPost, "/bikes", bytes.NewReader([]byte("invalid-json")))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("bad_request_missing_id", func(t *testing.T) {
		token := "valid_token"
		reqBody, _ := json.Marshal(map[string]interface{}{
			"hash_id": "hash",
		})
		req := httptest.NewRequest(http.MethodPost, "/bikes", bytes.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("conflict_numerical_id", func(t *testing.T) {
		mockService.CreateBikeFunc = func(ctx context.Context, numericalID int64, hashID *string, isElectric bool, creatorID int64) (*domain.Bike, error) {
			return nil, &pq.Error{Code: "23505", Constraint: "bikes_pkey"}
		}

		token := "valid_token"
		reqBody, _ := json.Marshal(map[string]interface{}{
			"numerical_id": "0123",
		})
		req := httptest.NewRequest(http.MethodPost, "/bikes", bytes.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Errorf("expected status 409, got %d", w.Code)
		}
	})

	t.Run("internal_error", func(t *testing.T) {
		mockService.CreateBikeFunc = func(ctx context.Context, numericalID int64, hashID *string, isElectric bool, creatorID int64) (*domain.Bike, error) {
			return nil, errors.New("db error")
		}

		token := "valid_token"
		reqBody, _ := json.Marshal(map[string]interface{}{
			"numerical_id": "0123",
		})
		req := httptest.NewRequest(http.MethodPost, "/bikes", bytes.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
	})

	t.Run("bad_request_invalid_id_range", func(t *testing.T) {
		token := "valid_token"
		// ID len < 4
		reqBody, _ := json.Marshal(map[string]interface{}{
			"numerical_id": "123",
		})
		req := httptest.NewRequest(http.MethodPost, "/bikes", bytes.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}

		// ID len > 6
		reqBody2, _ := json.Marshal(map[string]interface{}{
			"numerical_id": "123456",
		})
		req2 := httptest.NewRequest(http.MethodPost, "/bikes", bytes.NewReader(reqBody2))
		req2.Header.Set("Authorization", "Bearer "+token)
		w2 := httptest.NewRecorder()
		srv.server.Handler.ServeHTTP(w2, req2)
		if w2.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w2.Code)
		}
	})

	t.Run("bad_request_invalid_hash", func(t *testing.T) {
		token := "valid_token"
		// Non-alphanumeric hash
		reqBody, _ := json.Marshal(map[string]interface{}{
			"numerical_id": "12345",
			"hash_id":      "hash!",
		})
		req := httptest.NewRequest(http.MethodPost, "/bikes", bytes.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})
}

func TestHandleGetBike(t *testing.T) {
	mockService := &MockService{
		GetBikeFunc: func(ctx context.Context, id int64) (*domain.Bike, error) {
			if id == 1 {
				return &domain.Bike{NumericalID: 1, HashID: strPtr("hash1"), IsElectric: true}, nil
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
		hashID := "newhash"
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

	t.Run("bad_request_invalid_hash", func(t *testing.T) {
		token := "valid_token"
		reqBody, _ := json.Marshal(map[string]interface{}{
			"hash_id": "inv@lid",
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

func TestHandleDeleteBike(t *testing.T) {
	mockService := &MockService{
		DeleteBikeFunc: func(ctx context.Context, id int64) error {
			if id == 1 {
				return nil
			}
			return errors.New("delete error")
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
		req := httptest.NewRequest(http.MethodDelete, "/bikes/1", nil)
		req.Header.Set("Authorization", "Bearer valid_token")
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected status 204, got %d", w.Code)
		}
	})

	t.Run("internal_error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/bikes/2", nil)
		req.Header.Set("Authorization", "Bearer valid_token")
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
	})
}

func TestHandleGetBikeDetails(t *testing.T) {
	mockService := &MockService{
		GetBikeDetailsFunc: func(ctx context.Context, id int64) (*domain.BikeDetails, error) {
			if id == 1 {
				return &domain.BikeDetails{
					Bike: domain.Bike{NumericalID: 1},
				}, nil
			} else if id == 404 {
				return nil, sql.ErrNoRows
			}
			return nil, errors.New("db error")
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
		req := httptest.NewRequest(http.MethodGet, "/bikes/1/details", nil)
		req.Header.Set("Authorization", "Bearer valid_token")
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/bikes/404/details", nil)
		req.Header.Set("Authorization", "Bearer valid_token")
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("internal_error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/bikes/500/details", nil)
		req.Header.Set("Authorization", "Bearer valid_token")
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
	})
}
