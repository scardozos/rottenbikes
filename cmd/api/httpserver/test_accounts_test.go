package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/scardozos/rottenbikes/cmd/api/email"
	"github.com/scardozos/rottenbikes/internal/domain"
)

// Bikes created by E2E test accounts are only listed for test accounts, so
// running the suite against a shared environment doesn't show them to users.
// GET /bikes stays public: a bad token is ignored, never rejected.
func TestListBikesIncludesTestBikesOnlyForTestAccounts(t *testing.T) {
	var gotIncludeTest bool
	svc := &MockService{
		GetPosterByAPITokenFunc: func(ctx context.Context, token string) (*domain.AuthPoster, error) {
			switch token {
			case "test-account":
				return &domain.AuthPoster{PosterID: 1, IsTest: true}, nil
			case "real-account":
				return &domain.AuthPoster{PosterID: 2}, nil
			}
			return nil, domain.ErrInvalidToken
		},
		ListBikesFunc: func(ctx context.Context, searchQuery, sortBy string, limit, offset int, includeTest bool) ([]domain.Bike, error) {
			gotIncludeTest = includeTest
			return nil, nil
		},
	}
	srv, _ := New(svc, &email.NoopSender{}, ":0")

	cases := []struct {
		name        string
		auth        string
		includeTest bool
	}{
		{"anonymous", "", false},
		{"regular account", "Bearer real-account", false},
		{"test account", "Bearer test-account", true},
		{"invalid token", "Bearer nope", false},
		{"malformed header", "Basic abc", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotIncludeTest = !c.includeTest
			req := httptest.NewRequest(http.MethodGet, "/bikes", nil)
			if c.auth != "" {
				req.Header.Set("Authorization", c.auth)
			}
			w := httptest.NewRecorder()
			srv.server.Handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", w.Code, w.Body)
			}
			if gotIncludeTest != c.includeTest {
				t.Errorf("includeTest = %v, want %v", gotIncludeTest, c.includeTest)
			}
		})
	}
}

// Test accounts create bikes only in the reserved 6-digit range, and regular
// accounts only with real 4-5 digit numbers, so E2E runs against a shared
// environment never take (or later clean up) a real bike number.
func TestCreateBikeNumberRangeDependsOnAccount(t *testing.T) {
	svc := &MockService{
		GetPosterByAPITokenFunc: func(ctx context.Context, token string) (*domain.AuthPoster, error) {
			return &domain.AuthPoster{PosterID: 1, IsTest: token == "test-account"}, nil
		},
	}
	var gotIsTest *bool
	svc.CreateBikeFunc = func(ctx context.Context, numericalID string, hashID *string, isElectric, wasScanned bool, creatorID int64, creatorIsTest bool) (*domain.Bike, error) {
		gotIsTest = &creatorIsTest
		return &domain.Bike{NumericalID: numericalID}, nil
	}
	srv, _ := New(svc, &email.NoopSender{}, ":0")

	cases := []struct {
		account, id string
		want        int
	}{
		{"real-account", "1234", http.StatusCreated},
		{"real-account", "01234", http.StatusCreated},
		{"real-account", "123456", http.StatusBadRequest},
		{"test-account", "123456", http.StatusCreated},
		{"test-account", "012345", http.StatusCreated},
		{"test-account", "1234", http.StatusBadRequest},
		{"test-account", "12345", http.StatusBadRequest},
		{"test-account", "1234567", http.StatusBadRequest},
		{"test-account", "12a456", http.StatusBadRequest},
	}
	for _, c := range cases {
		t.Run(c.account+"/"+c.id, func(t *testing.T) {
			gotIsTest = nil
			req := httptest.NewRequest(http.MethodPost, "/bikes", strings.NewReader(`{"numerical_id":"`+c.id+`"}`))
			req.Header.Set("Authorization", "Bearer "+c.account)
			w := httptest.NewRecorder()
			srv.server.Handler.ServeHTTP(w, req)

			if w.Code != c.want {
				t.Fatalf("expected %d, got %d: %s", c.want, w.Code, w.Body)
			}
			if c.want == http.StatusCreated && (gotIsTest == nil || *gotIsTest != (c.account == "test-account")) {
				t.Errorf("service got creatorIsTest=%v", gotIsTest)
			}
			if c.want != http.StatusCreated && gotIsTest != nil {
				t.Error("service must not be called for a rejected number")
			}
		})
	}
}
