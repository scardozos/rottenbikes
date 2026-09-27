package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// DefaultMailtrapAPIURL is Mailtrap's production sending endpoint. Point
// APIURL at https://sandbox.api.mailtrap.io/api/send/<inbox_id> to deliver to
// a sandbox inbox instead (e.g. in dev, so E2E tests can read the emails).
const DefaultMailtrapAPIURL = "https://send.api.mailtrap.io/api/send"

type MailtrapSender struct {
	Token     string
	FromEmail string
	FromName  string
	Category  string
	// APIURL defaults to DefaultMailtrapAPIURL.
	APIURL string
	// Client defaults to a client with a 10s timeout.
	Client *http.Client
}

type mailtrapAddress struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type mailtrapRequest struct {
	From     mailtrapAddress   `json:"from"`
	To       []mailtrapAddress `json:"to"`
	Subject  string            `json:"subject"`
	Text     string            `json:"text"`
	Category string            `json:"category,omitempty"`
}

func (s *MailtrapSender) SendEmail(to string, subject string, body string) error {
	reqBody := mailtrapRequest{
		From: mailtrapAddress{
			Email: s.FromEmail,
			Name:  s.FromName,
		},
		To: []mailtrapAddress{
			{Email: to},
		},
		Subject:  subject,
		Text:     body,
		Category: s.Category,
	}

	if reqBody.Category == "" {
		reqBody.Category = "Auth"
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal mailtrap request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	apiURL := s.APIURL
	if apiURL == "" {
		apiURL = DefaultMailtrapAPIURL
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create mailtrap request: %w", err)
	}

	token := strings.TrimSpace(s.Token)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send mailtrap request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(resp.Body)
		return fmt.Errorf("mailtrap API returned status code %d: %s", resp.StatusCode, buf.String())
	}

	return nil
}

func (s *MailtrapSender) Name() string {
	return "MAILTRAP"
}
