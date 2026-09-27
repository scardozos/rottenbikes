//go:build e2e

package e2e

import (
	"net/http"
	"testing"
	"time"
)

func TestHealth(t *testing.T) {
	r := call(t, "GET", "/healthz", "", nil)
	if expectStatus(t, r, http.StatusOK) && string(r.Body) != "ok" {
		t.Errorf("healthz body: %s", r)
	}
	r = call(t, "GET", "/readyz", "", nil)
	if expectStatus(t, r, http.StatusOK) && string(r.Body) != "ready" {
		t.Errorf("readyz body: %s", r)
	}

	t.Run("metrics", func(t *testing.T) {
		if metricsURL == "" {
			t.Skip("E2E_METRICS_URL not set (the metrics port is not exposed publicly)")
		}
		resp, err := httpClient.Get(metricsURL + "/metrics")
		if err != nil {
			t.Fatalf("metrics: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("metrics: expected 200, got %d", resp.StatusCode)
		}
	})
}

func TestRegisterValidation(t *testing.T) {
	name := uniqueUsername()
	valid := map[string]string{"username": name, "email": name + "@example.com", "captcha_token": "e2e"}

	// These are rejected before the captcha is checked, so they run everywhere.
	t.Run("missing fields", func(t *testing.T) {
		for _, missing := range []string{"username", "email", "captcha_token"} {
			body := map[string]string{}
			for k, v := range valid {
				if k != missing {
					body[k] = v
				}
			}
			r := call(t, "POST", "/auth/register", "", body)
			if expectStatus(t, r, http.StatusBadRequest) {
				expectJSONError(t, r)
			}
		}
	})

	t.Run("malformed JSON", func(t *testing.T) {
		r := call(t, "POST", "/auth/register", "", "{not json")
		expectStatus(t, r, http.StatusBadRequest)
	})

	t.Run("invalid captcha is rejected", func(t *testing.T) {
		if !captchaEnforced {
			t.Skipf("hCaptcha is not enforced on %s", envName)
		}
		body := map[string]string{"username": name, "email": name + "@example.com", "captcha_token": "e2e-bogus-captcha"}
		r := call(t, "POST", "/auth/register", "", body)
		if expectStatus(t, r, http.StatusForbidden) {
			expectJSONError(t, r)
		}
	})

	t.Run("invalid email format is a client error", func(t *testing.T) {
		requireCaptchaPass(t)
		r := call(t, "POST", "/auth/register", "", map[string]string{
			"username": uniqueUsername(), "email": "not-an-email", "captcha_token": passingCaptcha(),
		})
		expectStatus(t, r, http.StatusBadRequest)
	})

	t.Run("invalid username format is a client error", func(t *testing.T) {
		requireCaptchaPass(t)
		r := call(t, "POST", "/auth/register", "", map[string]string{
			"username": "bad user!", "email": uniqueUsername() + "@example.com", "captcha_token": passingCaptcha(),
		})
		expectStatus(t, r, http.StatusBadRequest)
	})

	t.Run("duplicates conflict", func(t *testing.T) {
		requireCaptchaPass(t)
		u := seedPoster(t)
		r := call(t, "POST", "/auth/register", "", map[string]string{
			"username": uniqueUsername(), "email": u.Email, "captcha_token": passingCaptcha(),
		})
		if expectStatus(t, r, http.StatusConflict) {
			expectJSONError(t, r)
		}
		r = call(t, "POST", "/auth/register", "", map[string]string{
			"username": u.Username, "email": uniqueUsername() + "@example.com", "captcha_token": passingCaptcha(),
		})
		expectStatus(t, r, http.StatusConflict)
	})
}

// TestCaptchaConfiguration checks that the environment's captcha is set up
// the way the suite expects. On prod it is the only proof (short of a real
// sign-up) that sign-up is protected: a bogus token must get a 403. Note a
// 403 cannot tell "hCaptcha rejected it" from nothing else; an unreachable
// hCaptcha yields 503, which fails here too.
func TestCaptchaConfiguration(t *testing.T) {
	switch bogusCaptchaStatus {
	case http.StatusForbidden:
	case http.StatusBadRequest:
		if requireCaptcha {
			t.Errorf("hCaptcha is not enforced on %s: a bogus captcha token got past it", envName)
		}
	default:
		t.Errorf("captcha verification is broken on %s: a bogus token got %d (403 = enforced, 503 = hCaptcha unreachable or HCAPTCHA_SECRET missing)",
			envName, bogusCaptchaStatus)
	}

	if captchaToken != "" && tokenCaptchaStatus != http.StatusBadRequest {
		t.Errorf("E2E_CAPTCHA_TOKEN was not accepted on %s (got %d); is the API using hCaptcha's test secret?", envName, tokenCaptchaStatus)
	}

	t.Run("request-magic-link is protected too", func(t *testing.T) {
		if !captchaEnforced {
			t.Skipf("hCaptcha is not enforced on %s", envName)
		}
		u := seedPoster(t)
		r := call(t, "POST", "/auth/request-magic-link", "", map[string]string{
			"username": u.Username, "captcha_token": "e2e-bogus-captcha",
		})
		if expectStatus(t, r, http.StatusForbidden) {
			expectJSONError(t, r)
		}
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM magic_links WHERE poster_id = $1`, u.ID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Errorf("a magic link was issued despite the rejected captcha")
		}
	})
}

// Registration through the API end to end: register -> email -> confirm ->
// poll. On dev the magic link is read from the Mailtrap sandbox inbox.
func TestRegisterViaAPI(t *testing.T) {
	u, poll := registerViaAPI(t)
	expectStatus(t, call(t, "GET", "/auth/poll?token="+poll, "", nil), http.StatusNotFound)
	tok := confirm(t, magicTokenFor(t, u.Email, poll))
	r := call(t, "GET", "/auth/poll?token="+poll, "", nil)
	if expectStatus(t, r, http.StatusOK) && decode[map[string]string](t, r)["api_token"] != tok {
		t.Errorf("poll returned a different token than confirm")
	}
	r = call(t, "GET", "/auth/verify", tok, nil)
	if expectStatus(t, r, http.StatusOK) && decode[verifyResponse](t, r).Username != u.Username {
		t.Errorf("verify: %s", r)
	}
}

func TestConfirmPollFlow(t *testing.T) {
	u := seedPoster(t)
	magic, poll := seedMagicLink(t, u.ID)

	// Not confirmed yet.
	r := call(t, "GET", "/auth/poll?token="+poll, "", nil)
	expectStatus(t, r, http.StatusNotFound)

	r = call(t, "GET", "/auth/poll", "", nil)
	expectStatus(t, r, http.StatusBadRequest)

	// The poll token must not be usable as the confirm (emailed) token.
	r = call(t, "GET", "/auth/confirm/"+poll, "", nil)
	expectStatus(t, r, http.StatusBadRequest)

	r = call(t, "GET", "/auth/confirm/"+randomHex(32), "", nil)
	if expectStatus(t, r, http.StatusBadRequest) {
		expectJSONError(t, r)
	}

	r = call(t, "GET", "/auth/confirm/"+magic, "", nil)
	mustStatus(t, r, http.StatusOK)
	conf := decode[struct {
		APIToken  string    `json:"api_token"`
		Email     string    `json:"email"`
		ExpiresAt time.Time `json:"api_token_expires_at"`
	}](t, r)
	if conf.APIToken == "" || conf.Email != u.Email {
		t.Errorf("confirm response: %s", r)
	}
	if !conf.ExpiresAt.After(time.Now().Add(24 * time.Hour)) {
		t.Errorf("api token should be long-lived, expires at %v", conf.ExpiresAt)
	}

	// Magic links are one-time.
	r = call(t, "GET", "/auth/confirm/"+magic, "", nil)
	expectStatus(t, r, http.StatusBadRequest)

	// The requesting device picks up the same api token exactly once.
	r = call(t, "GET", "/auth/poll?token="+poll, "", nil)
	if expectStatus(t, r, http.StatusOK) {
		if got := decode[map[string]string](t, r)["api_token"]; got != conf.APIToken {
			t.Errorf("poll returned %q, confirm returned %q", got, conf.APIToken)
		}
	}
	r = call(t, "GET", "/auth/poll?token="+poll, "", nil)
	expectStatus(t, r, http.StatusNotFound)

	r = call(t, "GET", "/auth/verify", conf.APIToken, nil)
	mustStatus(t, r, http.StatusOK)
	v := decode[verifyResponse](t, r)
	if v.PosterID != u.ID || v.Username != u.Username || v.IsAdmin || v.Status != "ok" {
		t.Errorf("verify response: %s", r)
	}

	// Multiple sessions: a second login keeps the first token valid.
	magic2, _ := seedMagicLink(t, u.ID)
	second := confirm(t, magic2)
	expectStatus(t, call(t, "GET", "/auth/verify", conf.APIToken, nil), http.StatusOK)
	expectStatus(t, call(t, "GET", "/auth/verify", second, nil), http.StatusOK)

	// Logout revokes only the current session.
	expectStatus(t, call(t, "POST", "/auth/logout", second, nil), http.StatusNoContent)
	expectStatus(t, call(t, "GET", "/auth/verify", second, nil), http.StatusUnauthorized)
	expectStatus(t, call(t, "GET", "/auth/verify", conf.APIToken, nil), http.StatusOK)

	// Logout is idempotent: a dead or missing session is still a logout.
	expectStatus(t, call(t, "POST", "/auth/logout", second, nil), http.StatusNoContent)
	expectStatus(t, call(t, "POST", "/auth/logout", "", nil), http.StatusNoContent)
}

func TestExpiredMagicLink(t *testing.T) {
	u := seedPoster(t)
	magic, poll := seedMagicLink(t, u.ID)
	if _, err := db.Exec(`UPDATE magic_links SET expires_ts = NOW() - INTERVAL '1 minute' WHERE poll_token = $1`, sha256Hex(poll)); err != nil {
		t.Fatal(err)
	}
	expectStatus(t, call(t, "GET", "/auth/confirm/"+magic, "", nil), http.StatusBadRequest)
	expectStatus(t, call(t, "GET", "/auth/poll?token="+poll, "", nil), http.StatusNotFound)
}

func TestVerifyRejectsBadCredentials(t *testing.T) {
	cases := []struct {
		name   string
		header string
	}{
		{"missing", ""},
		{"wrong scheme", "Basic dXNlcjpwYXNz"},
		{"empty bearer", "Bearer "},
		{"unknown token", "Bearer " + randomHex(32)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var opts []reqOpt
			if c.header != "" {
				opts = append(opts, withHeader("Authorization", c.header))
			}
			r := call(t, "GET", "/auth/verify", "", nil, opts...)
			if expectStatus(t, r, http.StatusUnauthorized) {
				expectJSONError(t, r)
			}
		})
	}
}

func TestRequestMagicLink(t *testing.T) {
	t.Run("validation", func(t *testing.T) {
		r := call(t, "POST", "/auth/request-magic-link", "", map[string]string{"email": "x@example.com"})
		expectStatus(t, r, http.StatusBadRequest)
		r = call(t, "POST", "/auth/request-magic-link", "", map[string]string{"captcha_token": "e2e"})
		expectStatus(t, r, http.StatusBadRequest)
		r = call(t, "POST", "/auth/request-magic-link", "", "{bad")
		expectStatus(t, r, http.StatusBadRequest)
	})

	requireCaptchaPass(t)
	u := newUser(t) // already has 1 of the 2 daily links

	t.Run("unknown user", func(t *testing.T) {
		r := call(t, "POST", "/auth/request-magic-link", "", map[string]string{
			"username": uniqueUsername(), "captcha_token": passingCaptcha(),
		})
		expectStatus(t, r, http.StatusNotFound)
	})

	// Login by username, confirm on "another device", poll on this one.
	r := call(t, "POST", "/auth/request-magic-link", "", map[string]string{
		"username": u.Username, "captcha_token": passingCaptcha(),
	})
	mustStatus(t, r, http.StatusOK)
	poll := decode[map[string]string](t, r)["magic_token"]
	second := confirm(t, magicTokenFor(t, u.Email, poll))
	r = call(t, "GET", "/auth/poll?token="+poll, "", nil)
	if expectStatus(t, r, http.StatusOK) && decode[map[string]string](t, r)["api_token"] != second {
		t.Errorf("poll returned a different token than confirm")
	}

	// Third link within 24h (by email this time) is rate limited.
	r = call(t, "POST", "/auth/request-magic-link", "", map[string]string{
		"email": u.Email, "captcha_token": passingCaptcha(),
	})
	if expectStatus(t, r, http.StatusTooManyRequests) {
		expectJSONError(t, r)
	}
}
