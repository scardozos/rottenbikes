package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type magicLinkRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Origin   string `json:"origin"`
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
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Username == "" || req.Captcha == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Verify hCaptcha
	if err := s.verifyCaptcha(req.Captcha, req.Email); err != nil {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("invalid captcha"))
		return
	}

	magicToken, err := s.service.Register(r.Context(), req.Username, req.Email)
	if err != nil {
		if strings.Contains(err.Error(), "email already exists") || strings.Contains(err.Error(), "username already exists") {
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// In a real app, send the token via email.
	// For now, return it in the response (BAD for security, but okay for this prototype).
	uiHost := os.Getenv("UI_HOST")
	if uiHost == "" {
		uiHost = "localhost"
	}
	uiPort := os.Getenv("UI_PORT")
	if uiPort == "" {
		uiPort = "8081"
	}
	uiURL := fmt.Sprintf("http://%s:%s/confirm/%s", uiHost, uiPort, magicToken)
	if req.Origin != "" {
		uiURL = fmt.Sprintf("%s?origin=%s", uiURL, req.Origin)
	}
	log.Printf("UI confirmation link for %s: %s", req.Email, uiURL)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"magic_token": magicToken,
	})
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
		return
	}

	identifier := req.Email
	if identifier == "" {
		identifier = req.Username
	}

	if identifier == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("email or username is required"))
		return
	}

	magicToken, err := s.service.CreateMagicLink(r.Context(), identifier)
	if err != nil {
		if err.Error() == "user not found" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// In a real app, send the token via email.
	// For now, return it in the response (BAD for security, but okay for this prototype).
	uiHost := os.Getenv("UI_HOST")
	if uiHost == "" {
		uiHost = "localhost"
	}
	uiPort := os.Getenv("UI_PORT")
	if uiPort == "" {
		uiPort = "8081"
	}
	uiURL := fmt.Sprintf("http://%s:%s/confirm/%s", uiHost, uiPort, magicToken)
	if req.Origin != "" {
		uiURL = fmt.Sprintf("%s?origin=%s", uiURL, req.Origin)
	}
	log.Printf("UI confirmation link for %s: %s", identifier, uiURL)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"magic_token": magicToken,
	})
}

func (s *HTTPServer) verifyCaptcha(token, email string) error {
	secret := os.Getenv("HCAPTCHA_SECRET")
	if secret != "" {
		log.Printf("Verifying hCaptcha with secret (first 9 chars): %s...", secret[:9])
		vreq, err := http.PostForm("https://hcaptcha.com/siteverify", url.Values{
			"secret":   {secret},
			"response": {token},
		})
		if err != nil {
			log.Printf("hCaptcha request error: %v", err)
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
			log.Printf("hCaptcha decode error: %v", err)
			return err
		}

		if !vres.Success {
			log.Printf("hCaptcha verification FAILED for %s. Errors: %v", email, vres.ErrorCodes)
			return fmt.Errorf("invalid captcha")
		}
		log.Printf("hCaptcha verification SUCCESS for %s", email)
	} else {
		log.Println("WARNING: HCAPTCHA_SECRET not set, skipping verification (DEV mode?)")
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
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	token := r.URL.Query().Get("token")
	if token == "" {
		// New path-based format: /auth/confirm/TOKEN
		token = strings.TrimPrefix(r.URL.Path, "/auth/confirm/")
		token = strings.Trim(token, "/")
	}
	if token == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("token is required"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	res, err := s.service.ConfirmMagicLink(ctx, token)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid or expired token"))
		return
	}

	log.Printf("Magic link confirmed for %s", res.Email)

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
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	posterID, ok := posterIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"poster_id": posterID,
		"status":    "ok",
	})
}

// GET /auth/poll?token=...
func (s *HTTPServer) handlePollMagicLink(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	apiToken, err := s.service.CheckMagicLinkStatus(r.Context(), token)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if apiToken == "" {
		w.WriteHeader(http.StatusNotFound) // Not confirmed yet
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"api_token": apiToken,
	})
}
