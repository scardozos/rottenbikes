package httpserver

import (
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

func TestHandleAdminSearchPosters(t *testing.T) {
	mockService := &MockService{
		GetPosterByAPITokenFunc: func(ctx context.Context, token string) (*domain.AuthPoster, error) {
			// The bearer decides whether the caller is an admin.
			if token == "admin_token" {
				return &domain.AuthPoster{PosterID: 1, Email: "admin@example.com", Role: domain.PosterRoleAdmin}, nil
			}
			return &domain.AuthPoster{PosterID: 2, Email: "user@example.com", Role: domain.PosterRoleUser}, nil
		},
		SearchPostersFunc: func(ctx context.Context, query string, limit int) ([]domain.PosterSummary, error) {
			return []domain.PosterSummary{
				{PosterID: 2, Username: "troll", Email: "troll@example.com", Role: domain.PosterRoleUser, CreatedAt: time.Now(), ReviewCount: 12, BikeCount: 3},
			}, nil
		},
	}

	srv, err := New(mockService, &email.NoopSender{}, ":8080")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	t.Run("admin_success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/users?q=troll", nil)
		req.Header.Set("Authorization", "Bearer admin_token")
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		var posters []domain.PosterSummary
		if err := json.Unmarshal(w.Body.Bytes(), &posters); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(posters) != 1 || posters[0].Username != "troll" {
			t.Errorf("expected 1 poster (troll), got %+v", posters)
		}
		if posters[0].Role != domain.PosterRoleUser {
			t.Errorf("expected role user, got %s", posters[0].Role)
		}
	})

	t.Run("non_admin_forbidden", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/users?q=troll", nil)
		req.Header.Set("Authorization", "Bearer user_token")
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status 403, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/users?q=troll", nil)
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
	})
}

func TestHandleAdminPurgePoster(t *testing.T) {
	var purgedTarget int64
	var called bool

	mockService := &MockService{
		GetPosterByAPITokenFunc: func(ctx context.Context, token string) (*domain.AuthPoster, error) {
			if token == "admin_token" {
				return &domain.AuthPoster{PosterID: 1, Email: "admin@example.com", Role: domain.PosterRoleAdmin}, nil
			}
			return &domain.AuthPoster{PosterID: 2, Email: "user@example.com", Role: domain.PosterRoleUser}, nil
		},
		PurgePosterFunc: func(ctx context.Context, adminPosterID, targetPosterID int64) error {
			called = true
			purgedTarget = targetPosterID
			return nil
		},
	}

	srv, err := New(mockService, &email.NoopSender{}, ":8080")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	t.Run("admin_success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/admin/users/2", nil)
		req.Header.Set("Authorization", "Bearer admin_token")
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected status 204, got %d", w.Code)
		}
		if !called {
			t.Error("expected PurgePoster to be called")
		}
		if purgedTarget != 2 {
			t.Errorf("expected target 2, got %d", purgedTarget)
		}
	})

	t.Run("purge_gets_its_own_timeout", func(t *testing.T) {
		// A purge may recompute aggregates for many bikes; it must get the
		// handler's 30s budget, not the auth middleware's 3s lookup timeout.
		var remaining time.Duration
		mockService.PurgePosterFunc = func(ctx context.Context, adminPosterID, targetPosterID int64) error {
			if deadline, ok := ctx.Deadline(); ok {
				remaining = time.Until(deadline)
			}
			return nil
		}
		req := httptest.NewRequest(http.MethodDelete, "/admin/users/2", nil)
		req.Header.Set("Authorization", "Bearer admin_token")
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Fatalf("expected status 204, got %d", w.Code)
		}
		if remaining < 20*time.Second {
			t.Errorf("expected purge deadline ~30s away, got %v", remaining)
		}
	})

	t.Run("admin_target_rejected", func(t *testing.T) {
		mockService.PurgePosterFunc = func(ctx context.Context, adminPosterID, targetPosterID int64) error {
			return domain.ErrCannotPurgeAdmin
		}

		req := httptest.NewRequest(http.MethodDelete, "/admin/users/9", nil)
		req.Header.Set("Authorization", "Bearer admin_token")
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400 for admin target, got %d", w.Code)
		}
	})

	t.Run("missing_target", func(t *testing.T) {
		mockService.PurgePosterFunc = func(ctx context.Context, adminPosterID, targetPosterID int64) error {
			return sql.ErrNoRows
		}

		req := httptest.NewRequest(http.MethodDelete, "/admin/users/404", nil)
		req.Header.Set("Authorization", "Bearer admin_token")
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404 for missing target, got %d", w.Code)
		}
	})

	t.Run("self_purge_rejected", func(t *testing.T) {
		mockService.PurgePosterFunc = func(ctx context.Context, adminPosterID, targetPosterID int64) error {
			t.Error("PurgePoster must not be called for self-purge")
			return nil
		}

		req := httptest.NewRequest(http.MethodDelete, "/admin/users/1", nil)
		req.Header.Set("Authorization", "Bearer admin_token")
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("non_admin_forbidden", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/admin/users/2", nil)
		req.Header.Set("Authorization", "Bearer user_token")
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status 403, got %d", w.Code)
		}
	})

	t.Run("invalid_id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/admin/users/notanumber", nil)
		req.Header.Set("Authorization", "Bearer admin_token")
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})
}

func TestHandleVerifyTokenIsAdmin(t *testing.T) {
	mockService := &MockService{
		GetPosterByAPITokenFunc: func(ctx context.Context, token string) (*domain.AuthPoster, error) {
			if token == "admin_token" {
				return &domain.AuthPoster{PosterID: 1, Email: "admin@example.com", Username: "admin", Role: domain.PosterRoleAdmin}, nil
			}
			return &domain.AuthPoster{PosterID: 2, Email: "user@example.com", Username: "user", Role: domain.PosterRoleUser}, nil
		},
	}

	srv, err := New(mockService, &email.NoopSender{}, ":8080")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	decode := func(body []byte) map[string]any {
		var m map[string]any
		if err := json.Unmarshal(body, &m); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		return m
	}

	t.Run("admin_flag_true_for_admin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/verify", nil)
		req.Header.Set("Authorization", "Bearer admin_token")
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
		m := decode(w.Body.Bytes())
		if m["is_admin"] != true {
			t.Errorf("expected is_admin true, got %v", m["is_admin"])
		}
	})

	t.Run("admin_flag_false_for_regular_user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/verify", nil)
		req.Header.Set("Authorization", "Bearer user_token")
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
		m := decode(w.Body.Bytes())
		if m["is_admin"] != false {
			t.Errorf("expected is_admin false, got %v", m["is_admin"])
		}
	})
}

func TestHandleAdminDeleteBike(t *testing.T) {
	var deletedBikeID string
	var called bool

	mockService := &MockService{
		GetPosterByAPITokenFunc: func(ctx context.Context, token string) (*domain.AuthPoster, error) {
			if token == "admin_token" {
				return &domain.AuthPoster{PosterID: 1, Email: "admin@example.com", Role: domain.PosterRoleAdmin}, nil
			}
			return &domain.AuthPoster{PosterID: 2, Email: "user@example.com", Role: domain.PosterRoleUser}, nil
		},
		AdminDeleteBikeFunc: func(ctx context.Context, adminPosterID int64, bikeID string) error {
			called = true
			deletedBikeID = bikeID
			return nil
		},
	}

	srv, err := New(mockService, &email.NoopSender{}, ":8080")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	t.Run("admin_success", func(t *testing.T) {
		called = false
		req := httptest.NewRequest(http.MethodDelete, "/admin/bikes/1234", nil)
		req.Header.Set("Authorization", "Bearer admin_token")
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected status 204, got %d", w.Code)
		}
		if !called {
			t.Error("expected AdminDeleteBike to be called")
		}
		if deletedBikeID != "1234" {
			t.Errorf("expected deleted bike id 1234, got %s", deletedBikeID)
		}
	})

	t.Run("non_admin_forbidden", func(t *testing.T) {
		called = false
		req := httptest.NewRequest(http.MethodDelete, "/admin/bikes/1234", nil)
		req.Header.Set("Authorization", "Bearer user_token")
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status 403, got %d", w.Code)
		}
		if called {
			t.Error("expected AdminDeleteBike not to be called")
		}
	})

	t.Run("missing_target", func(t *testing.T) {
		mockService.AdminDeleteBikeFunc = func(ctx context.Context, adminPosterID int64, bikeID string) error {
			return sql.ErrNoRows
		}

		req := httptest.NewRequest(http.MethodDelete, "/admin/bikes/404", nil)
		req.Header.Set("Authorization", "Bearer admin_token")
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404 for missing target, got %d", w.Code)
		}
	})

	t.Run("invalid_id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/admin/bikes/notanumber", nil)
		req.Header.Set("Authorization", "Bearer admin_token")
		w := httptest.NewRecorder()

		srv.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})
}
