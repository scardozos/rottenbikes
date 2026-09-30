//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

// When the target API delivers email to a Mailtrap sandbox inbox (dev), the
// suite reads magic links and login codes from that inbox, covering
// register -> email -> confirm / verify-code with no DB shortcut. Otherwise
// it falls back to swapping the stored hashes in the DB (interceptMagicLink).
var (
	mailtrapToken   = os.Getenv("E2E_MAILTRAP_API_TOKEN")
	mailtrapAccount = os.Getenv("E2E_MAILTRAP_ACCOUNT_ID")
	mailtrapInbox   = os.Getenv("E2E_MAILTRAP_INBOX_ID")
	mailtrapBaseURL = strings.TrimRight(envOr("E2E_MAILTRAP_API_BASE", "https://mailtrap.io"), "/")
)

var (
	magicLinkRe = regexp.MustCompile(`https?://\S+/confirm/([0-9a-f]{64})\S*`)
	loginCodeRe = regexp.MustCompile(`(?s)Enter this code.*?\b(\d{6})\b`)
)

func inboxConfigured() bool {
	return mailtrapToken != "" && mailtrapAccount != "" && mailtrapInbox != ""
}

func mailSource() string {
	if inboxConfigured() {
		return "mailtrap-inbox"
	}
	return "db-intercept"
}

// loginEmailFor returns the raw magic token and login code emailed to `to`
// for the link identified by pollToken.
func loginEmailFor(t *testing.T, to, pollToken string) (magic, code string) {
	t.Helper()
	if !inboxConfigured() {
		return interceptMagicLink(t, pollToken)
	}
	return readLoginEmailFromInbox(t, to)
}

// magicTokenForPoster returns the raw magic token of the latest link emailed
// to the poster, when the suite has no poll token for it.
func magicTokenForPoster(t *testing.T, u user) string {
	t.Helper()
	if !inboxConfigured() {
		return interceptLatestMagicLink(t, u.ID)
	}
	magic, _ := readLoginEmailFromInbox(t, u.Email)
	return magic
}

type mailtrapMessage struct {
	ID      int64  `json:"id"`
	ToEmail string `json:"to_email"`
	Subject string `json:"subject"`
}

func mailtrapRequest(t *testing.T, method, path string) []byte {
	t.Helper()
	req, err := http.NewRequest(method, fmt.Sprintf("%s/api/accounts/%s/inboxes/%s%s", mailtrapBaseURL, mailtrapAccount, mailtrapInbox, path), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+mailtrapToken)
	req.Header.Set("Accept", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("mailtrap %s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		t.Fatalf("mailtrap %s %s: %d %s", method, path, resp.StatusCode, body)
	}
	return body
}

// readLoginEmailFromInbox waits for the email sent to `to`, checks the link
// is well formed, deletes the message and returns the magic token and the
// login code.
func readLoginEmailFromInbox(t *testing.T, to string) (magic, code string) {
	t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	for {
		var msgs []mailtrapMessage
		body := mailtrapRequest(t, "GET", "/messages?search="+url.QueryEscape(to))
		if err := json.Unmarshal(body, &msgs); err != nil {
			t.Fatalf("mailtrap messages: %v: %s", err, body)
		}
		for _, m := range msgs {
			if !strings.EqualFold(m.ToEmail, to) {
				continue
			}
			text := string(mailtrapRequest(t, "GET", fmt.Sprintf("/messages/%d/body.txt", m.ID)))
			mailtrapRequest(t, "DELETE", fmt.Sprintf("/messages/%d", m.ID))

			match := magicLinkRe.FindStringSubmatch(text)
			if match == nil {
				t.Fatalf("email %q to %s has no /confirm/{token} link:\n%s", m.Subject, to, text)
			}
			link, err := url.Parse(match[0])
			if err != nil || link.Host == "" {
				t.Fatalf("malformed confirm link %q: %v", match[0], err)
			}
			if link.Scheme != "https" && envName != "local" {
				t.Errorf("confirm link should be https on %s: %s", envName, link)
			}
			codeMatch := loginCodeRe.FindStringSubmatch(text)
			if codeMatch == nil {
				t.Fatalf("email %q to %s has no login code:\n%s", m.Subject, to, text)
			}
			t.Logf("magic link and login code read from the Mailtrap inbox (%s)", link.Host)
			return match[1], codeMatch[1]
		}
		if time.Now().After(deadline) {
			t.Fatalf("no email to %s arrived in the Mailtrap inbox within 60s", to)
		}
		time.Sleep(2 * time.Second)
	}
}
