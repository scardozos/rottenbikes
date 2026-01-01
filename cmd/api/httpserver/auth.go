package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/scardozos/rottenbikes/internal/domain"
)

type magicLinkRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Origin   string `json:"origin"`
	Captcha  string `json:"captcha_token"`
}

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Captcha  string `json:"captcha_token"`
	Origin   string `json:"origin"`
}

// POST /auth/register
func (s *HTTPServer) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Username == "" || req.Captcha == "" {
		s.sendError(w, "email, username, and captcha are required", http.StatusBadRequest)
		return
	}

	// Verify hCaptcha
	if err := s.verifyCaptcha(r.Context(), req.Captcha, req.Email); err != nil {
		s.sendError(w, "invalid captcha", http.StatusForbidden)
		return
	}

	magicToken, err := s.service.Register(r.Context(), req.Username, req.Email)
	if err != nil {
		if strings.Contains(err.Error(), "email already exists") || strings.Contains(err.Error(), "username already exists") {
			s.sendError(w, err.Error(), http.StatusConflict)
			return
		}
		s.sendInternalServerError(w, r, err)
		return
	}

	uiHost := os.Getenv("UI_HOST")
	if uiHost == "" {
		uiHost = "localhost"
	}
	uiPort := os.Getenv("UI_PORT")
	if uiPort == "" {
		uiPort = "8081"
	}

	scheme := "http"
	if !isPrivateIP(uiHost) {
		scheme = "https"
	}
	uiURL := fmt.Sprintf("%s://%s:%s/confirm/%s", scheme, uiHost, uiPort, magicToken)
	if req.Origin != "" {
		uiURL = fmt.Sprintf("%s?origin=%s", uiURL, req.Origin)
	}
	if req.Origin != "" {
		uiURL = fmt.Sprintf("%s?origin=%s", uiURL, req.Origin)
	}
	zerolog.Ctx(r.Context()).Info().Str("email", req.Email).Str("url", uiURL).Msg("sending UI confirmation link")

	subject := "Welcome to RottenBikes!"
	body := fmt.Sprintf("Hello %s,\n\nPlease confirm your registration by clicking the following link:\n\n%s\n\nIf you did not request this, please ignore this email.", req.Username, uiURL)

	if err := s.emailSender.SendEmail(req.Email, subject, body); err != nil {
		zerolog.Ctx(r.Context()).Error().Err(err).Str("email", req.Email).Msg("failed to send registration email")
		s.sendError(w, "failed to send confirmation email", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message":     "confirmation email sent",
		"magic_token": magicToken,
	})
}

// POST /auth/request-magic-link
func (s *HTTPServer) handleRequestMagicLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req magicLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	identifier := req.Email
	if identifier == "" {
		identifier = req.Username
	}

	if identifier == "" || req.Captcha == "" {
		s.sendError(w, "email or username, and captcha, are required", http.StatusBadRequest)
		return
	}

	// Verify hCaptcha
	if err := s.verifyCaptcha(r.Context(), req.Captcha, identifier); err != nil {
		s.sendError(w, "invalid captcha", http.StatusForbidden)
		return
	}

	magicToken, targetEmail, err := s.service.CreateMagicLink(r.Context(), identifier)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			s.sendError(w, "user not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, domain.ErrRateLimitExceeded) {
			s.sendError(w, "daily magic link limit reached", http.StatusTooManyRequests)
			return
		}
		s.sendInternalServerError(w, r, err)
		return
	}

	uiHost := os.Getenv("UI_HOST")
	if uiHost == "" {
		uiHost = "localhost"
	}
	uiPort := os.Getenv("UI_PORT")
	if uiPort == "" {
		uiPort = "8081"
	}

	scheme := "http"
	if !isPrivateIP(uiHost) {
		scheme = "https"
	}
	uiURL := fmt.Sprintf("%s://%s:%s/confirm/%s", scheme, uiHost, uiPort, magicToken)
	if req.Origin != "" {
		uiURL = fmt.Sprintf("%s?origin=%s", uiURL, req.Origin)
	}
	if req.Origin != "" {
		uiURL = fmt.Sprintf("%s?origin=%s", uiURL, req.Origin)
	}
	zerolog.Ctx(r.Context()).Info().Str("email", targetEmail).Str("url", uiURL).Msg("sending magic link")

	subject := "Your RottenBikes Magic Link"
	body := fmt.Sprintf("Hello,\n\nYou requested a magic link to log in to RottenBikes. Click the following link to continue:\n\n%s\n\nIf you did not request this, please ignore this email.", uiURL)

	if err := s.emailSender.SendEmail(targetEmail, subject, body); err != nil {
		zerolog.Ctx(r.Context()).Error().Err(err).Str("email", targetEmail).Msg("failed to send magic link email")
		s.sendError(w, "failed to send magic link email", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message":     "magic link email sent",
		"magic_token": magicToken,
	})
}

func (s *HTTPServer) verifyCaptcha(ctx context.Context, token, email string) error {
	secret := strings.TrimSpace(os.Getenv("HCAPTCHA_SECRET"))

	if secret != "" {
		vreq, err := http.PostForm("https://api.hcaptcha.com/siteverify", url.Values{
			"secret":   {secret},
			"response": {token},
		})
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Str("email", email).Msg("hCaptcha request error")
			return err
		}
		defer vreq.Body.Close()

		var vres struct {
			Success     bool     `json:"success"`
			ErrorCodes  []string `json:"error-codes"`
			Hostname    string   `json:"hostname"`
			ChallengeTS string   `json:"challenge_ts"`
		}
		if err := json.NewDecoder(vreq.Body).Decode(&vres); err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("hCaptcha decode error")
			return err
		}

		if !vres.Success {
			zerolog.Ctx(ctx).Warn().Str("email", email).Strs("errors", vres.ErrorCodes).Msg("hCaptcha verification FAILED")
			return fmt.Errorf("invalid captcha")
		}
		zerolog.Ctx(ctx).Info().Str("email", email).Msg("hCaptcha verification SUCCESS")
	} else {
		zerolog.Ctx(ctx).Warn().Msg("HCAPTCHA_SECRET not set, skipping verification request (DEV mode?)")
	}
	return nil
}

type confirmResponse struct {
	APIToken        string    `json:"api_token"`
	Email           string    `json:"email"`
	APITokenExpires time.Time `json:"api_token_expires_at"`
}

// GET /auth/confirm?token=...
func (s *HTTPServer) handleConfirmMagicLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	token := r.URL.Query().Get("token")
	if token == "" {
		// New path-based format: /auth/confirm/TOKEN
		token = strings.TrimPrefix(r.URL.Path, "/auth/confirm/")
		token = strings.Trim(token, "/")
	}
	if token == "" {
		s.sendError(w, "token is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	res, err := s.service.ConfirmMagicLink(ctx, token)
	if err != nil {
		s.sendError(w, "invalid or expired token", http.StatusBadRequest)
		return
	}

	zerolog.Ctx(r.Context()).Info().Str("email", res.Email).Msg("magic link confirmed")

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(confirmResponse{
		APIToken:        res.APIToken,
		Email:           res.Email,
		APITokenExpires: res.APITokenExpiresAt,
	})
}

// GET /auth/verify
func (s *HTTPServer) handleVerifyToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	posterID, ok := posterIDFromContext(r.Context())
	if !ok {
		s.sendError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	username, _ := usernameFromContext(r.Context())

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"poster_id": posterID,
		"username":  username,
		"status":    "ok",
	})
}

// GET /auth/poll?token=...
func (s *HTTPServer) handlePollMagicLink(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		s.sendError(w, "token is required", http.StatusBadRequest)
		return
	}

	apiToken, err := s.service.CheckMagicLinkStatus(r.Context(), token)
	if err != nil {
		s.sendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if apiToken == "" {
		s.sendError(w, "not confirmed", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"api_token": apiToken,
	})
}

// DELETE /auth/user
func (s *HTTPServer) handleDeletePoster(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		s.sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	posterID, ok := posterIDFromContext(r.Context())
	if !ok {
		s.sendError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Optional: safety check if username provided matches context
	// Leaving strictly context-based for now as token is proof of ownership.

	// Parse optional delete config
	var req struct {
		DeletePosterSubresources bool `json:"delete_poster_subresources"`
	}
	// We allow empty body (defaults to false)
	if r.Body != http.NoBody {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	// Pass the parsed flag
	if err := s.service.DeletePoster(ctx, posterID, req.DeletePosterSubresources); err != nil {
		s.sendInternalServerError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func isPrivateIP(host string) bool {
	if host == "localhost" {
		return true
	}
	// Check for private IPv4 ranges
	// 10.0.0.0/8
	// 172.16.0.0/12
	// 192.168.0.0/16
	// 127.0.0.0/8
	ip := net.ParseIP(host)
	if ip == nil {
		return false // It's a domain name or invalid IP, assume public/HTTPS unless specifically localhost
	}

	if ip.IsLoopback() {
		return true
	}
	if ip.IsPrivate() {
		return true
	}
	return false
}
