package email

import (
	"fmt"
	"os"
)

// EmailSender defines the interface for sending emails and identifying the sender.
type EmailSender interface {
	SendEmail(to string, subject string, body string) error
	Name() string
}

// GetToken returns the authentication token for a given sender by name.
// It looks for an environment variable named EMAIL_SENDER_TOKEN_$NAME.
func GetToken(senderName string) string {
	envVar := fmt.Sprintf("EMAIL_SENDER_TOKEN_%s", senderName)
	return os.Getenv(envVar)
}

// NoopSender is a placeholder sender that does nothing.
type NoopSender struct{}

func (s *NoopSender) SendEmail(to string, subject string, body string) error {
	return nil
}

func (s *NoopSender) Name() string {
	return "NOOP"
}
