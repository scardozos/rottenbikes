package email

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestMailtrapSender(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var got mailtrapRequest
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/api/send/42" {
				t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			}
			if auth := r.Header.Get("Authorization"); auth != "Bearer tok" {
				t.Errorf("Authorization header %q", auth)
			}
			if ct := r.Header.Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type %q", ct)
			}
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Errorf("decode body: %v", err)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"success": true}`))
		}))
		defer ts.Close()

		s := &MailtrapSender{Token: " tok\n", FromEmail: "hello@rottenbik.es", FromName: "RottenBikes", APIURL: ts.URL + "/api/send/42"}
		if err := s.SendEmail("alice@example.com", "Subject", "Body"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.From.Email != "hello@rottenbik.es" || got.From.Name != "RottenBikes" ||
			len(got.To) != 1 || got.To[0].Email != "alice@example.com" ||
			got.Subject != "Subject" || got.Text != "Body" || got.Category != "Auth" {
			t.Errorf("unexpected payload: %+v", got)
		}
	})

	for _, status := range []int{http.StatusUnauthorized, http.StatusBadRequest, http.StatusInternalServerError} {
		t.Run("non-2xx "+http.StatusText(status), func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"errors": ["Unauthorized"]}`))
			}))
			defer ts.Close()

			s := &MailtrapSender{Token: "tok", APIURL: ts.URL}
			err := s.SendEmail("alice@example.com", "Subject", "Body")
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), strconv.Itoa(status)) {
				t.Errorf("error should mention the status code: %v", err)
			}
			if !strings.Contains(err.Error(), "Unauthorized") {
				t.Errorf("error should include the response body: %v", err)
			}
		})
	}

	t.Run("unreachable", func(t *testing.T) {
		ts := httptest.NewServer(http.NotFoundHandler())
		ts.Close()
		s := &MailtrapSender{Token: "tok", APIURL: ts.URL}
		if err := s.SendEmail("alice@example.com", "Subject", "Body"); err == nil {
			t.Fatal("expected an error")
		}
	})

	t.Run("defaults to the production endpoint", func(t *testing.T) {
		var called string
		s := &MailtrapSender{Token: "tok", Client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			called = r.URL.String()
			return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Header: http.Header{}}, nil
		})}}
		if err := s.SendEmail("alice@example.com", "Subject", "Body"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if called != DefaultMailtrapAPIURL {
			t.Errorf("called %q, want %q", called, DefaultMailtrapAPIURL)
		}
	})
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
