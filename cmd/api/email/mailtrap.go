package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type MailtrapSender struct {
	Token     string
	FromEmail string
	FromName  string
	Category  string
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

	req, err := http.NewRequest("POST", "https://send.api.mailtrap.io/api/send", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create mailtrap request: %w", err)
	}

	req.Header.Set("Api-Token", s.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
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
