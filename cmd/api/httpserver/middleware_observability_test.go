package httpserver

import (
	"testing"
)

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/bikes", "/bikes"},
		{"/bikes/1001", "/bikes/{id}"},
		{"/bikes/1001/details", "/bikes/{id}/details"},
		{"/bikes/1001/reviews", "/bikes/{id}/reviews"},
		{"/reviews/7", "/reviews/{id}"},
		{"/auth/confirm/123456", "/auth/confirm/[REDACTED]"},
		{"/auth/confirm/", "/auth/confirm/[REDACTED]"},
	}

	for _, tt := range tests {
		got := sanitizePath(tt.input)
		if got != tt.expected {
			t.Errorf("sanitizePath(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestSanitizeURI(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/bikes", "/bikes"},
		{"/auth/confirm/123456", "/auth/confirm/%5BREDACTED%5D"},
		{"/auth/confirm/123456?origin=web", "/auth/confirm/%5BREDACTED%5D?origin=web"},
		{"/auth/poll?token=some_token", "/auth/poll?token=%5BREDACTED%5D"},
		{"/auth/poll?token=some_token&other=123", "/auth/poll?other=123&token=%5BREDACTED%5D"},
		{"invalid-uri-%%", "invalid-uri-%%"}, // parse error fallback
	}

	for _, tt := range tests {
		got := sanitizeURI(tt.input)
		if got != tt.expected {
			t.Errorf("sanitizeURI(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}
