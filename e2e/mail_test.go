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
// suite reads magic links from that inbox, covering register -> email ->
// confirm -> poll with no DB shortcut. Otherwise it falls back to swapping
// the stored token hash in the DB (interceptMagicLink).
var (
	mailtrapToken   = os.Getenv("E2E_MAILTRAP_API_TOKEN")
	mailtrapAccount = os.Getenv("E2E_MAILTRAP_ACCOUNT_ID")
	mailtrapInbox   = os.Getenv("E2E_MAILTRAP_INBOX_ID")
	mailtrapBaseURL = strings.TrimRight(envOr("E2E_MAILTRAP_API_BASE", "https://mailtrap.io"), "/")
)

var magicLinkRe = regexp.MustCompile(`https?://\S+/confirm/([0-9a-f]{64})\S*`)

func inboxConfigured() bool {
	return mailtrapToken != "" && mailtrapAccount != "" && mailtrapInbox != ""
}

func mailSource() string {
	if inboxConfigured() {
		return "mailtrap-inbox"
	}
	return "db-intercept"
}

// magicTokenFor returns the raw magic token emailed to `to` for the link
// identified by pollToken.
func magicTokenFor(t *testing.T, to, pollToken string) string {
	t.Helper()
	if !inboxConfigured() {
		return interceptMagicLink(t, pollToken)
	}
	return readMagicLinkFromInbox(t, to)
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

// readMagicLinkFromInbox waits for the email sent to `to`, checks the link
// is well formed, deletes the message and returns the magic token.
func readMagicLinkFromInbox(t *testing.T, to string) string {
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
			t.Logf("magic link read from the Mailtrap inbox (%s)", link.Host)
			return match[1]
		}
		if time.Now().After(deadline) {
			t.Fatalf("no email to %s arrived in the Mailtrap inbox within 60s", to)
		}
		time.Sleep(2 * time.Second)
	}
}
