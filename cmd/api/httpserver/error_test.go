package httpserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/scardozos/rottenbikes/cmd/api/email"
	"github.com/scardozos/rottenbikes/internal/domain"
)

func TestSendError(t *testing.T) {
	srv := &HTTPServer{}
	w := httptest.NewRecorder()
	msg := "test error message"
	status := http.StatusTeapot

	srv.sendError(w, msg, status)

	if w.Code != status {
		t.Errorf("expected status %d, got %d", status, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["error"] != msg {
		t.Errorf("expected error message %q, got %q", msg, resp["error"])
	}
}

func TestErrorSchemas(t *testing.T) {
	mockService := &MockService{
		GetBikeFunc: func(ctx context.Context, id int64) (*domain.Bike, error) {
			return nil, sql.ErrNoRows
		},
		GetReviewWithRatingsByIDFunc: func(ctx context.Context, reviewID int64) (*domain.ReviewWithRatings, error) {
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

	tests := []struct {
		name           string
		method         string
		url            string
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:           "bikes_invalid_id",
			method:         http.MethodGet,
			url:            "/bikes/invalid",
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "invalid bike id",
		},
		{
			name:           "bikes_not_found",
			method:         http.MethodGet,
			url:            "/bikes/999",
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "bike not found",
		},
		{
			name:           "reviews_not_found",
			method:         http.MethodGet,
			url:            "/reviews/999",
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "review not found",
		},
		{
			name:           "auth_invalid_method",
			method:         http.MethodGet,
			url:            "/auth/register",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedMsg:    "method not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			if tt.url != "/auth/register" {
				req.Header.Set("Authorization", "Bearer valid_token")
			}
			w := httptest.NewRecorder()

			srv.server.Handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var resp map[string]string
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if resp["error"] != tt.expectedMsg {
				t.Errorf("expected error message %q, got %q", tt.expectedMsg, resp["error"])
			}
		})
	}
}
