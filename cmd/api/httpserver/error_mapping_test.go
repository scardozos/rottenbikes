package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lib/pq"

	"github.com/scardozos/rottenbikes/cmd/api/email"
	"github.com/scardozos/rottenbikes/internal/domain"
)

// Domain errors caused by client input must map to 4xx, never 500.

func authedRequest(srv *HTTPServer, method, path string, body any) *httptest.ResponseRecorder {
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer valid_token")
	w := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(w, req)
	return w
}

func authedMock() *MockService {
	return &MockService{
		GetPosterByAPITokenFunc: func(ctx context.Context, token string) (*domain.AuthPoster, error) {
			return &domain.AuthPoster{PosterID: 1}, nil
		},
	}
}

func TestCreateReviewErrorMapping(t *testing.T) {
	cases := []struct {
		name       string
		path       string
		err        error
		wantStatus int
		wantMsg    string
	}{
		{"validation", "/bikes/1234/reviews", &domain.ValidationError{Msg: "invalid score 6 for overall: must be between 1 and 5"}, http.StatusBadRequest, "invalid score 6"},
		{"unknown bike", "/bikes/1234/reviews", domain.ErrBikeNotFound, http.StatusNotFound, "bike not found"},
		{"non-numeric bike id", "/bikes/abc/reviews", nil, http.StatusBadRequest, "invalid bike id"},
		{"unexpected error", "/bikes/1234/reviews", fmt.Errorf("boom"), http.StatusInternalServerError, "internal server error"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc := authedMock()
			called := false
			svc.CreateReviewWithRatingsFunc = func(ctx context.Context, in domain.CreateReviewInput) (int64, error) {
				called = true
				return 0, c.err
			}
			srv, _ := New(svc, &email.NoopSender{}, ":0")

			w := authedRequest(srv, http.MethodPost, c.path, map[string]any{"overall": 6})
			if w.Code != c.wantStatus || !strings.Contains(w.Body.String(), c.wantMsg) {
				t.Errorf("expected %d %q, got %d %s", c.wantStatus, c.wantMsg, w.Code, w.Body)
			}
			if c.err == nil && called {
				t.Error("service must not be called for an invalid bike id")
			}
		})
	}

	t.Run("success response is JSON", func(t *testing.T) {
		svc := authedMock()
		svc.CreateReviewWithRatingsFunc = func(ctx context.Context, in domain.CreateReviewInput) (int64, error) {
			return 7, nil
		}
		srv, _ := New(svc, &email.NoopSender{}, ":0")
		w := authedRequest(srv, http.MethodPost, "/bikes/1234/reviews", map[string]any{"overall": 5})
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", w.Code)
		}
		if ct := w.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type %q, want application/json", ct)
		}
		if w.Body.String() != "{\"review_id\":7}\n" {
			t.Errorf("unexpected body %s", w.Body)
		}
	})
}

func TestUpdateReviewValidationError(t *testing.T) {
	svc := authedMock()
	svc.UpdateReviewWithRatingsFunc = func(ctx context.Context, in domain.UpdateReviewInput) error {
		return &domain.ValidationError{Msg: "comment must be at most 500 characters"}
	}
	srv, _ := New(svc, &email.NoopSender{}, ":0")
	w := authedRequest(srv, http.MethodPut, "/reviews/1", map[string]any{"comment": "x"})
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "at most 500") {
		t.Errorf("expected 400 with message, got %d %s", w.Code, w.Body)
	}
}

func TestUpdateBikeErrorMapping(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantMsg    string
	}{
		{"hash_id taken", &pq.Error{Code: "23505", Constraint: "bikes_hash_id_key"}, http.StatusConflict, "hash_id already exists"},
		{"wrapped duplicate", fmt.Errorf("update: %w", &pq.Error{Code: "23505", Constraint: "other"}), http.StatusConflict, "already exists"},
		{"validation", &domain.ValidationError{Msg: "hash_id must be alphanumeric"}, http.StatusBadRequest, "alphanumeric"},
		{"other db error", &pq.Error{Code: "40001"}, http.StatusInternalServerError, "internal server error"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc := authedMock()
			svc.UpdateBikeFunc = func(ctx context.Context, id string, hashID *string, isElectric *bool, creatorID int64) error {
				return c.err
			}
			srv, _ := New(svc, &email.NoopSender{}, ":0")
			w := authedRequest(srv, http.MethodPut, "/bikes/1234", map[string]any{"hash_id": "abc"})
			if w.Code != c.wantStatus || !strings.Contains(w.Body.String(), c.wantMsg) {
				t.Errorf("expected %d %q, got %d %s", c.wantStatus, c.wantMsg, w.Code, w.Body)
			}
		})
	}

	t.Run("empty hash_id is passed through to clear it", func(t *testing.T) {
		svc := authedMock()
		var got *string
		svc.UpdateBikeFunc = func(ctx context.Context, id string, hashID *string, isElectric *bool, creatorID int64) error {
			got = hashID
			return nil
		}
		srv, _ := New(svc, &email.NoopSender{}, ":0")
		w := authedRequest(srv, http.MethodPut, "/bikes/1234", map[string]any{"hash_id": ""})
		if w.Code != http.StatusNoContent || got == nil || *got != "" {
			t.Errorf("expected 204 with hash_id \"\" passed to the service, got %d, %v", w.Code, got)
		}
	})
}

func TestCreateBikeValidationError(t *testing.T) {
	svc := authedMock()
	svc.CreateBikeFunc = func(ctx context.Context, numericalID string, hashID *string, isElectric, wasScanned bool, creatorID int64, creatorIsTest bool) (*domain.Bike, error) {
		return nil, &domain.ValidationError{Msg: "numerical_id must be 4-5 digits"}
	}
	srv, _ := New(svc, &email.NoopSender{}, ":0")
	w := authedRequest(srv, http.MethodPost, "/bikes", map[string]any{"numerical_id": "1234"})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d %s", w.Code, w.Body)
	}
}

func TestConfirmMagicLinkOnlyViaPath(t *testing.T) {
	var got string
	svc := &MockService{
		ConfirmMagicLinkFunc: func(ctx context.Context, token string) (*domain.ConfirmResult, error) {
			got = token
			return &domain.ConfirmResult{APIToken: "api", Email: "a@example.com"}, nil
		},
	}
	srv, _ := New(svc, &email.NoopSender{}, ":0")

	w := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/confirm/abc123?token=other", nil))
	if w.Code != http.StatusOK || got != "abc123" {
		t.Errorf("expected the path token to be used, got %d token=%q", w.Code, got)
	}

	w = httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/confirm?token=abc123", nil))
	if w.Code != http.StatusNotFound {
		t.Errorf("query form is not a route, expected 404, got %d", w.Code)
	}
}
