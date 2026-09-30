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

	// Usernames are public, so a taken one is a 409, even if the email is
	// taken too (the response must not depend on the email).
	t.Run("taken username conflicts", func(t *testing.T) {
		requireCaptchaPass(t)
		u := seedPoster(t)
		for _, email := range []string{uniqueUsername() + "@example.com", u.Email} {
			r := call(t, "POST", "/auth/register", "", map[string]string{
				"username": u.Username, "email": email, "captcha_token": passingCaptcha(),
			})
			if expectStatus(t, r, http.StatusConflict) {
				expectJSONError(t, r)
			}
		}
	})

	// A taken email must not be revealed: the response is the same as for a
	// new registration, the owner gets a login link, and the requester's poll
	// token can never be turned into the owner's session.
	t.Run("taken email is indistinguishable from a new registration", func(t *testing.T) {
		requireCaptchaPass(t)
		owner := seedPoster(t)

		fresh := call(t, "POST", "/auth/register", "", map[string]string{
			"username": uniqueUsername(), "email": uniqueUsername() + "@example.com", "captcha_token": passingCaptcha(),
		})
		taken := call(t, "POST", "/auth/register", "", map[string]string{
			"username": uniqueUsername(), "email": owner.Email, "captcha_token": passingCaptcha(),
		})
		mustStatus(t, fresh, http.StatusOK)
		if !expectStatus(t, taken, http.StatusOK) {
			return
		}
		expectSameAuthResponse(t, fresh, taken)
		decoy := decode[map[string]string](t, taken)["magic_token"]

		// The owner can log in with the emailed link...
		tok := confirm(t, magicTokenForPoster(t, owner))
		r := call(t, "GET", "/auth/verify", tok, nil)
		if expectStatus(t, r, http.StatusOK) && decode[verifyResponse](t, r).PosterID != owner.ID {
			t.Errorf("the emailed link should log in the existing account: %s", r)
		}
		// ...but whoever tried to register cannot pick up that session.
		expectDecoyUnusable(t, decoy)
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

// Registration through the API end to end: register -> email -> the
// registering device enters the emailed code. On dev the email is read from
// the Mailtrap sandbox inbox.
func TestRegisterViaAPI(t *testing.T) {
	u, poll := registerViaAPI(t)
	_, code := loginEmailFor(t, u.Email, poll)
	tok := loginWithCode(t, poll, code)
	r := call(t, "GET", "/auth/verify", tok, nil)
	if expectStatus(t, r, http.StatusOK) && decode[verifyResponse](t, r).Username != u.Username {
		t.Errorf("verify: %s", r)
	}
}

func TestConfirmFlow(t *testing.T) {
	u := seedPoster(t)
	magic, poll, code := seedMagicLink(t, u.ID)

	// The poll token must not be usable as the confirm (emailed) token.
	r := call(t, "GET", "/auth/confirm/"+poll, "", nil)
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

	// Opening the link (say, on a phone) leaves the code usable on the device
	// that asked for the email, once. It gets its own session.
	fromCode := loginWithCode(t, poll, code)
	if fromCode == conf.APIToken {
		t.Error("the code must start its own session")
	}
	expectStatus(t, call(t, "GET", "/auth/verify", fromCode, nil), http.StatusOK)
	expectStatus(t, verifyCode(t, poll, code), http.StatusBadRequest)

	// Polling used to hand the requesting device the confirmed session.
	expectStatus(t, call(t, "GET", "/auth/poll?token="+poll, "", nil), http.StatusNotFound)

	r = call(t, "GET", "/auth/verify", conf.APIToken, nil)
	mustStatus(t, r, http.StatusOK)
	v := decode[verifyResponse](t, r)
	if v.PosterID != u.ID || v.Username != u.Username || v.IsAdmin || v.Status != "ok" {
		t.Errorf("verify response: %s", r)
	}

	// Multiple sessions: a second login keeps the first token valid.
	magic2, _, _ := seedMagicLink(t, u.ID)
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

// The requesting device logs in with the code from the email, instead of
// the link.
func TestLoginCodeFlow(t *testing.T) {
	u := seedPoster(t)
	magic, poll, code := seedMagicLink(t, u.ID)

	wrong := otherLoginCode(code)
	for _, c := range []struct {
		name       string
		poll, code string
	}{
		{"wrong code", poll, wrong},
		{"emailed token instead of the poll token", magic, code},
		{"unknown poll token", randomHex(32), code},
		{"malformed code", poll, "12345"},
	} {
		r := verifyCode(t, c.poll, c.code)
		if expectStatus(t, r, http.StatusBadRequest) {
			expectJSONError(t, r)
		}
	}
	expectStatus(t, call(t, "POST", "/auth/verify-code", "", map[string]string{"token": poll}), http.StatusBadRequest)

	// Spaces, as people may type them, are fine.
	r := verifyCode(t, poll, code[:3]+" "+code[3:])
	mustStatus(t, r, http.StatusOK)
	conf := decode[struct {
		APIToken  string    `json:"api_token"`
		Email     string    `json:"email"`
		ExpiresAt time.Time `json:"api_token_expires_at"`
	}](t, r)
	if conf.APIToken == "" || conf.Email != u.Email || !conf.ExpiresAt.After(time.Now().Add(24*time.Hour)) {
		t.Errorf("verify-code response: %s", r)
	}
	r = call(t, "GET", "/auth/verify", conf.APIToken, nil)
	if expectStatus(t, r, http.StatusOK) && decode[verifyResponse](t, r).PosterID != u.ID {
		t.Errorf("verify: %s", r)
	}

	// One-time. The link is independent and still works, once.
	expectStatus(t, verifyCode(t, poll, code), http.StatusBadRequest)
	confirm(t, magic)
	expectStatus(t, call(t, "GET", "/auth/confirm/"+magic, "", nil), http.StatusBadRequest)
}

// After 5 wrong codes a link accepts no code, not even the right one.
func TestLoginCodeAttemptsAreLimited(t *testing.T) {
	u := seedPoster(t)
	_, poll, code := seedMagicLink(t, u.ID)
	wrong := otherLoginCode(code)
	for i := 0; i < 5; i++ {
		expectStatus(t, verifyCode(t, poll, wrong), http.StatusBadRequest)
	}
	expectStatus(t, verifyCode(t, poll, code), http.StatusBadRequest)
}

func TestExpiredMagicLink(t *testing.T) {
	u := seedPoster(t)
	magic, poll, code := seedMagicLink(t, u.ID)
	if _, err := db.Exec(`UPDATE magic_links SET expires_ts = NOW() - INTERVAL '1 minute' WHERE poll_token = $1`, sha256Hex(poll)); err != nil {
		t.Fatal(err)
	}
	expectStatus(t, call(t, "GET", "/auth/confirm/"+magic, "", nil), http.StatusBadRequest)
	expectStatus(t, verifyCode(t, poll, code), http.StatusBadRequest)
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

	// Unknown accounts get the same response as known ones, so the endpoint
	// does not reveal which emails are registered.
	t.Run("unknown user is indistinguishable", func(t *testing.T) {
		known := seedPoster(t)
		ref := call(t, "POST", "/auth/request-magic-link", "", map[string]string{
			"email": known.Email, "captcha_token": passingCaptcha(),
		})
		mustStatus(t, ref, http.StatusOK)
		for _, body := range []map[string]string{
			{"email": uniqueUsername() + "@example.com", "captcha_token": passingCaptcha()},
			{"username": uniqueUsername(), "captcha_token": passingCaptcha()},
		} {
			r := call(t, "POST", "/auth/request-magic-link", "", body)
			if !expectStatus(t, r, http.StatusOK) {
				continue
			}
			expectSameAuthResponse(t, ref, r)
			expectDecoyUnusable(t, decode[map[string]string](t, r)["magic_token"])
		}
	})

	// Login by username, entering the emailed code on this device.
	r := call(t, "POST", "/auth/request-magic-link", "", map[string]string{
		"username": u.Username, "captcha_token": passingCaptcha(),
	})
	mustStatus(t, r, http.StatusOK)
	poll := decode[map[string]string](t, r)["magic_token"]
	_, code := loginEmailFor(t, u.Email, poll)
	second := loginWithCode(t, poll, code)
	r = call(t, "GET", "/auth/verify", second, nil)
	if expectStatus(t, r, http.StatusOK) && decode[verifyResponse](t, r).PosterID != u.ID {
		t.Errorf("the code should log in %s: %s", u.Username, r)
	}

	// Third link within 24h (by email this time) is rate limited.
	r = call(t, "POST", "/auth/request-magic-link", "", map[string]string{
		"email": u.Email, "captcha_token": passingCaptcha(),
	})
	if expectStatus(t, r, http.StatusTooManyRequests) {
		expectJSONError(t, r)
	}
}
