package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/scardozos/rottenbikes/internal/domain"
)

type magicLinkRequest struct {
	Email   string `json:"email"`
	Captcha string `json:"captcha_token"`
}

// POST /auth/request-magic-link
func (s *HTTPServer) handleRequestMagicLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req magicLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid JSON"))
		return
	}
	if req.Email == "" || req.Captcha == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("email and captcha_token are required"))
		return
	}

	// TODO: verify CAPTCHA with your provider (reCAPTCHA, hCaptcha, etc.).
	// If invalid, return 400/403.

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	token, err := domain.CreateMagicLink(ctx, s.db, req.Email)
	if err != nil {
		log.Printf("CreateMagicLink error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Build confirmation URL and send email (pseudo-code).
	confirmURL := fmt.Sprintf("http://localhost:8080/auth/confirm?token=%s", token)

	// TODO: integrate with real email sender.
	log.Printf("Magic link for %s: %s", req.Email, confirmURL)

	w.WriteHeader(http.StatusAccepted)
}

type confirmResponse struct {
	APIToken        string    `json:"api_token"`
	Email           string    `json:"email"`
	APITokenExpires time.Time `json:"api_token_expires_at"`
}

// GET /auth/confirm?token=...
func (s *HTTPServer) handleConfirmMagicLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	token := r.URL.Query().Get("token")
	if token == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("token is required"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	res, err := domain.ConfirmMagicLink(ctx, s.db, token)
	if err != nil {
		log.Printf("ConfirmMagicLink error: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid or expired token"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(confirmResponse{
		APIToken:        res.APIToken,
		Email:           res.Email,
		APITokenExpires: res.APITokenExpiresAt,
	})
}
