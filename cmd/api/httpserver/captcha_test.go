package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"github.com/scardozos/rottenbikes/cmd/api/email"
	"github.com/scardozos/rottenbikes/internal/domain"
)

func counterValue(t *testing.T, c prometheus.Counter) float64 {
	t.Helper()
	var m dto.Metric
	if err := c.Write(&m); err != nil {
		t.Fatalf("read counter: %v", err)
	}
	return m.GetCounter().GetValue()
}

// recordingSender captures sent emails and can be told to fail.
type recordingSender struct {
	mu   sync.Mutex
	sent []sentEmail
	err  error
}

type sentEmail struct{ To, Subject, Body string }

func (s *recordingSender) SendEmail(to, subject, body string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	s.sent = append(s.sent, sentEmail{to, subject, body})
	return nil
}

func (s *recordingSender) Name() string { return "RECORDING" }

func (s *recordingSender) last(t *testing.T) sentEmail {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.sent) == 0 {
		t.Fatal("no email was sent")
	}
	return s.sent[len(s.sent)-1]
}

func authMockService() *MockService {
	return &MockService{
		RegisterFunc: func(ctx context.Context, username, email string) (string, string, error) {
			return "magic-" + username, "poll-" + username, nil
		},
		CreateMagicLinkFunc: func(ctx context.Context, identifier string) (string, string, string, error) {
			return "magic-" + identifier, "poll-" + identifier, identifier + "@example.com", nil
		},
	}
}

func postJSON(srv *HTTPServer, path string, body any) *httptest.ResponseRecorder {
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	w := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(w, req)
	return w
}

// fakeHCaptcha stands in for api.hcaptcha.com/siteverify.
func fakeHCaptcha(t *testing.T, handler func(form url.Values) (int, string)) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("hCaptcha called with %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		status, body := handler(r.PostForm)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(ts.Close)
	return ts
}

func TestCaptchaVerification(t *testing.T) {
	const secret = "test-secret"
	t.Setenv("HCAPTCHA_SECRET", secret)
	t.Setenv("APP_ENV", "production")

	endpoints := []struct {
		path string
		body map[string]string
	}{
		{"/auth/register", map[string]string{"username": "alice", "email": "alice@example.com", "captcha_token": "tok"}},
		{"/auth/request-magic-link", map[string]string{"username": "alice", "captcha_token": "tok"}},
	}

	cases := []struct {
		name        string
		hcaptcha    func(form url.Values) (int, string)
		unreachable bool
		wantStatus  int
		wantResult  string
		wantEmail   bool
	}{
		{
			name: "success",
			hcaptcha: func(form url.Values) (int, string) {
				if form.Get("secret") != secret || form.Get("response") != "tok" {
					t.Errorf("unexpected siteverify form: %v", form)
				}
				return http.StatusOK, `{"success": true}`
			},
			wantStatus: http.StatusOK,
			wantResult: "success",
			wantEmail:  true,
		},
		{
			name: "rejected token",
			hcaptcha: func(url.Values) (int, string) {
				return http.StatusOK, `{"success": false, "error-codes": ["invalid-input-response"]}`
			},
			wantStatus: http.StatusForbidden,
			wantResult: "failure",
		},
		{
			name:        "hCaptcha unreachable",
			unreachable: true,
			wantStatus:  http.StatusServiceUnavailable,
			wantResult:  "error",
		},
		{
			name:       "hCaptcha server error",
			hcaptcha:   func(url.Values) (int, string) { return http.StatusBadGateway, "bad gateway" },
			wantStatus: http.StatusServiceUnavailable,
			wantResult: "error",
		},
		{
			name:       "malformed hCaptcha response",
			hcaptcha:   func(url.Values) (int, string) { return http.StatusOK, "<html>" },
			wantStatus: http.StatusServiceUnavailable,
			wantResult: "error",
		},
	}

	for _, ep := range endpoints {
		for _, c := range cases {
			t.Run(ep.path+"/"+c.name, func(t *testing.T) {
				sender := &recordingSender{}
				srv, _ := New(authMockService(), sender, ":0")
				if c.unreachable {
					ts := httptest.NewServer(http.NotFoundHandler())
					srv.captchaVerifyURL = ts.URL
					ts.Close() // connection refused
				} else {
					srv.captchaVerifyURL = fakeHCaptcha(t, c.hcaptcha).URL
				}

				counter := captchaVerificationsTotal.WithLabelValues(c.wantResult)
				before := counterValue(t, counter)

				w := postJSON(srv, ep.path, ep.body)
				srv.pendingEmails.Wait()
				if w.Code != c.wantStatus {
					t.Errorf("expected %d, got %d: %s", c.wantStatus, w.Code, w.Body)
				}
				if got := counterValue(t, counter) - before; got != 1 {
					t.Errorf("captcha_verifications_total{result=%q} increased by %v, want 1", c.wantResult, got)
				}
				if sent := len(sender.sent) > 0; sent != c.wantEmail {
					t.Errorf("email sent = %v, want %v", sent, c.wantEmail)
				}
			})
		}
	}

	t.Run("secret missing in production", func(t *testing.T) {
		t.Setenv("HCAPTCHA_SECRET", "")
		srv, _ := New(authMockService(), &recordingSender{}, ":0")
		srv.captchaVerifyURL = fakeHCaptcha(t, func(url.Values) (int, string) {
			t.Error("hCaptcha must not be called without a secret")
			return http.StatusOK, `{"success": true}`
		}).URL
		counter := captchaVerificationsTotal.WithLabelValues("not_configured")
		before := counterValue(t, counter)

		w := postJSON(srv, "/auth/register", endpoints[0].body)
		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected 503, got %d: %s", w.Code, w.Body)
		}
		if counterValue(t, counter)-before != 1 {
			t.Error("not_configured was not counted")
		}
	})

	t.Run("secret missing in development is skipped", func(t *testing.T) {
		t.Setenv("HCAPTCHA_SECRET", "")
		t.Setenv("APP_ENV", "development")
		srv, _ := New(authMockService(), &recordingSender{}, ":0")
		w := postJSON(srv, "/auth/register", endpoints[0].body)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body)
		}
	})
}

var confirmLinkRe = regexp.MustCompile(`https?://\S+`)

func extractLink(t *testing.T, body string) *url.URL {
	t.Helper()
	links := confirmLinkRe.FindAllString(body, -1)
	if len(links) != 1 {
		t.Fatalf("expected exactly one link in the email body, found %d:\n%s", len(links), body)
	}
	u, err := url.Parse(links[0])
	if err != nil {
		t.Fatalf("malformed link %q: %v", links[0], err)
	}
	return u
}

func TestEmailContainsConfirmLink(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	t.Setenv("HCAPTCHA_SECRET", "")

	cases := []struct {
		name, host, port, wantScheme string
	}{
		{"public host uses https", "rottenbik.es", "443", "https"},
		{"private host uses http", "192.168.1.10", "8081", "http"},
	}
	endpoints := []struct {
		path, kind, wantToken, wantTo string
		body                          map[string]string
	}{
		{"/auth/register", "register", "magic-alice", "alice@example.com",
			map[string]string{"username": "alice", "email": "alice@example.com", "captcha_token": "x", "origin": "web&x=1"}},
		{"/auth/request-magic-link", "magic_link", "magic-bob", "bob@example.com",
			map[string]string{"username": "bob", "captcha_token": "x", "origin": "web&x=1"}},
	}

	for _, c := range cases {
		for _, ep := range endpoints {
			t.Run(c.name+ep.path, func(t *testing.T) {
				t.Setenv("UI_HOST", c.host)
				t.Setenv("UI_PORT", c.port)
				sender := &recordingSender{}
				srv, _ := New(authMockService(), sender, ":0")

				counter := emailsSentTotal.WithLabelValues("RECORDING", ep.kind, "success")
				before := counterValue(t, counter)

				w := postJSON(srv, ep.path, ep.body)
				if w.Code != http.StatusOK {
					t.Fatalf("expected 200, got %d: %s", w.Code, w.Body)
				}
				srv.pendingEmails.Wait()
				if counterValue(t, counter)-before != 1 {
					t.Errorf("emails_sent_total{kind=%q,result=success} not incremented", ep.kind)
				}

				mail := sender.last(t)
				if mail.To != ep.wantTo {
					t.Errorf("email sent to %q, want %q", mail.To, ep.wantTo)
				}
				link := extractLink(t, mail.Body)
				if link.Scheme != c.wantScheme || link.Hostname() != c.host || link.Port() != c.port {
					t.Errorf("link %s: want %s://%s:%s", link, c.wantScheme, c.host, c.port)
				}
				if link.Path != "/confirm/"+ep.wantToken {
					t.Errorf("link path %q, want /confirm/%s", link.Path, ep.wantToken)
				}
				if got := link.Query().Get("origin"); got != "web&x=1" {
					t.Errorf("origin round-trip: got %q", got)
				}

				// The raw magic token goes to the email only; the response
				// carries the (different) poll token.
				resp := map[string]string{}
				_ = json.Unmarshal(w.Body.Bytes(), &resp)
				if resp["magic_token"] == ep.wantToken || strings.Contains(w.Body.String(), ep.wantToken) {
					t.Errorf("response leaks the emailed magic token: %s", w.Body)
				}
			})
		}
	}
}

// Emails are sent in the background, so a failed send cannot change the
// response (that would reveal the account exists); it is counted and logged.
func TestEmailSendFailure(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	t.Setenv("HCAPTCHA_SECRET", "")

	for _, ep := range []struct {
		path, kind string
		body       map[string]string
	}{
		{"/auth/register", "register", map[string]string{"username": "alice", "email": "alice@example.com", "captcha_token": "x"}},
		{"/auth/request-magic-link", "magic_link", map[string]string{"username": "alice", "captcha_token": "x"}},
	} {
		t.Run(ep.path, func(t *testing.T) {
			srv, _ := New(authMockService(), &recordingSender{err: context.DeadlineExceeded}, ":0")
			counter := emailsSentTotal.WithLabelValues("RECORDING", ep.kind, "failure")
			before := counterValue(t, counter)

			w := postJSON(srv, ep.path, ep.body)
			if w.Code != http.StatusOK {
				t.Errorf("expected 200, got %d: %s", w.Code, w.Body)
			}
			srv.pendingEmails.Wait()
			if counterValue(t, counter)-before != 1 {
				t.Errorf("emails_sent_total{kind=%q,result=failure} not incremented", ep.kind)
			}
		})
	}
}

func TestRegisterValidationError(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	t.Setenv("HCAPTCHA_SECRET", "")
	svc := authMockService()
	svc.RegisterFunc = func(ctx context.Context, username, email string) (string, string, error) {
		return "", "", &domain.ValidationError{Msg: "invalid email format"}
	}
	srv, _ := New(svc, &email.NoopSender{}, ":0")

	w := postJSON(srv, "/auth/register", map[string]string{"username": "alice", "email": "nope", "captcha_token": "x"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body)
	}
	if !strings.Contains(w.Body.String(), "invalid email format") {
		t.Errorf("expected the validation message, got %s", w.Body)
	}
}
