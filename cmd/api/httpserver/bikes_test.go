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
	var lastLimit, lastOffset int
	mockService := &MockService{
		ListBikesFunc: func(ctx context.Context, searchQuery, sortBy string, limit, offset int) ([]domain.Bike, error) {
			lastLimit = limit
			lastOffset = offset
			return []domain.Bike{
				{NumericalID: "1", HashID: strPtr("hash1"), IsElectric: true},
				{NumericalID: "2", HashID: strPtr("hash2"), IsElectric: false},
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

	t.Run("empty_list", func(t *testing.T) {
		mockService.ListBikesFunc = func(ctx context.Context, searchQuery, sortBy string, limit, offset int) ([]domain.Bike, error) {
			return nil, nil // Simulate empty DB returning nil
		}

		req := httptest.NewRequest(http.MethodGet, "/bikes", nil)
		req.Header.Set("Authorization", "Bearer valid_token")
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		// Verify response body is []
		if w.Body.String() != "[]\n" {
			t.Errorf("expected body [], got %q", w.Body.String())
		}
	})

	t.Run("pagination_params", func(t *testing.T) {
		mockService.ListBikesFunc = func(ctx context.Context, searchQuery, sortBy string, limit, offset int) ([]domain.Bike, error) {
			lastLimit = limit
			lastOffset = offset
			return []domain.Bike{}, nil
		}

		req := httptest.NewRequest(http.MethodGet, "/bikes?limit=15&offset=5", nil)
		req.Header.Set("Authorization", "Bearer valid_token")
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
		if lastLimit != 15 {
			t.Errorf("expected limit 15, got %d", lastLimit)
		}
		if lastOffset != 5 {
			t.Errorf("expected offset 5, got %d", lastOffset)
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
		CreateBikeFunc: func(ctx context.Context, numericalID string, hashID *string, isElectric bool, creatorID int64) (*domain.Bike, error) {
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
		mockService.CreateBikeFunc = func(ctx context.Context, numericalID string, hashID *string, isElectric bool, creatorID int64) (*domain.Bike, error) {
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
		mockService.CreateBikeFunc = func(ctx context.Context, numericalID string, hashID *string, isElectric bool, creatorID int64) (*domain.Bike, error) {
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
		GetBikeFunc: func(ctx context.Context, id string) (*domain.Bike, error) {
			if id == "1" {
				return &domain.Bike{NumericalID: "1", HashID: strPtr("hash1"), IsElectric: true}, nil
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
	var capturedCreatorID int64
	mockService := &MockService{
		GetPosterByAPITokenFunc: func(ctx context.Context, token string) (*domain.AuthPoster, error) {
			return &domain.AuthPoster{PosterID: 1}, nil
		},
		UpdateBikeFunc: func(ctx context.Context, id string, hashID *string, isElectric *bool, creatorID int64) error {
			capturedCreatorID = creatorID
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
		// Ownership must be threaded: the authenticated poster's id is passed as
		// creator_id to the service so the store can enforce it in SQL.
		if capturedCreatorID != 1 {
			t.Errorf("expected creatorID 1 to be passed to service, got %d", capturedCreatorID)
		}
	})

	t.Run("not_owner_returns_404", func(t *testing.T) {
		// Non-owner (or nonexistent bike) -> service returns sql.ErrNoRows ->
		// handler maps to 404 (no enumeration of whether the bike exists).
		mockService.UpdateBikeFunc = func(ctx context.Context, id string, hashID *string, isElectric *bool, creatorID int64) error {
			return sql.ErrNoRows
		}

		token := "valid_token"
		reqBody, _ := json.Marshal(map[string]interface{}{"hash_id": "x"})
		req := httptest.NewRequest(http.MethodPut, "/bikes/1", bytes.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404 for non-owner, got %d", w.Code)
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
	var capturedCreatorID int64
	mockService := &MockService{
		DeleteBikeFunc: func(ctx context.Context, id string, creatorID int64) error {
			capturedCreatorID = creatorID
			if id == "1" {
				return nil
			}
			if id == "404" {
				return sql.ErrNoRows
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
		if capturedCreatorID != 1 {
			t.Errorf("expected creatorID 1 to be passed to service, got %d", capturedCreatorID)
		}
	})

	t.Run("not_owner_returns_404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/bikes/404", nil)
		req.Header.Set("Authorization", "Bearer valid_token")
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404 for non-owner, got %d", w.Code)
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
	var lastLimit, lastOffset int
	mockService := &MockService{
		GetBikeDetailsFunc: func(ctx context.Context, id string, limit, offset int) (*domain.BikeDetails, error) {
			lastLimit = limit
			lastOffset = offset
			if id == "1" {
				return &domain.BikeDetails{
					Bike: domain.Bike{NumericalID: "1"},
				}, nil
			} else if id == "404" {
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

	t.Run("pagination_params", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/bikes/1/details?limit=10&offset=2", nil)
		req.Header.Set("Authorization", "Bearer valid_token")
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
		if lastLimit != 10 {
			t.Errorf("expected limit 10, got %d", lastLimit)
		}
		if lastOffset != 2 {
			t.Errorf("expected offset 2, got %d", lastOffset)
		}
	})
}
