package httpserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

	if err := s.verifyCaptcha(r.Context(), req.Captcha, req.Email); err != nil {
		s.sendCaptchaError(w, err)
		return
	}

	link, err := s.service.Register(r.Context(), req.Username, req.Email)
	if err != nil {
		// Usernames are public, so a taken one can be reported. A taken email
		// is answered exactly like a successful registration (see below).
		if errors.Is(err, domain.ErrUsernameExists) {
			s.sendError(w, err.Error(), http.StatusConflict)
			return
		}
		if errors.Is(err, domain.ErrEmailExists) {
			s.handleRegisterExistingEmail(w, r, req)
			return
		}
		if errors.Is(err, domain.ErrValidation) {
			s.sendError(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.sendInternalServerError(w, r, err)
		return
	}

	zerolog.Ctx(r.Context()).Info().Str("email", req.Email).Str("url", confirmURL("[REDACTED]", req.Origin)).Msg("sending UI confirmation link")

	subject := "Welcome to RottenBikes!"
	body := fmt.Sprintf("Hello %s,\n\nPlease confirm your registration %s\n\nIf you did not request this, please ignore this email.", req.Username, loginInstructions(link, req.Origin))
	s.sendEmailAsync(r, "register", req.Email, subject, body, "failed to send registration email")

	// The requesting device gets the poll token. With the code from the email,
	// it can log in through /auth/verify-code.
	s.sendRegisterResponse(w, link.PollToken)
}

// handleRegisterExistingEmail answers a registration for an email that
// already has an account exactly like a new registration, so the response
// does not reveal which emails are registered. The account's owner gets a
// login link instead of a welcome email. The requesting device gets a decoy
// poll token that no code redeems: whoever tried to register must not be
// able to log in as the owner.
func (s *HTTPServer) handleRegisterExistingEmail(w http.ResponseWriter, r *http.Request, req registerRequest) {
	logger := zerolog.Ctx(r.Context())
	link, targetEmail, err := s.service.CreateMagicLink(r.Context(), req.Email)
	switch {
	case errors.Is(err, domain.ErrRateLimitExceeded):
		logger.Info().Str("email", req.Email).Msg("registration for an existing email: daily magic link limit reached, not emailing")
	case err != nil:
		logger.Error().Err(err).Str("email", req.Email).Msg("registration for an existing email: could not issue a login link")
	default:
		logger.Info().Str("email", targetEmail).Str("url", confirmURL("[REDACTED]", req.Origin)).Msg("registration for an existing email: sending login link")
		subject := "Your RottenBikes account"
		body := fmt.Sprintf("Hello,\n\nSomeone tried to register a new RottenBikes account with this email address, but it already has an account. If it was you, you can log in %s\n\nIf you did not request this, please ignore this email.", loginInstructions(link, req.Origin))
		s.sendEmailAsync(r, "existing_account", targetEmail, subject, body, "failed to send existing account email")
	}

	s.sendRegisterResponse(w, decoyPollToken())
}

func (s *HTTPServer) sendRegisterResponse(w http.ResponseWriter, pollToken string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message":     "confirmation email sent",
		"magic_token": pollToken,
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

	if err := s.verifyCaptcha(r.Context(), req.Captcha, identifier); err != nil {
		s.sendCaptchaError(w, err)
		return
	}

	link, targetEmail, err := s.service.CreateMagicLink(r.Context(), identifier)
	if err != nil {
		// Answer an unknown email/username exactly like a known one, so the
		// response does not reveal which accounts exist. No code redeems the
		// decoy poll token, and nothing is emailed.
		if errors.Is(err, domain.ErrUserNotFound) {
			zerolog.Ctx(r.Context()).Info().Str("identifier", identifier).Msg("magic link requested for an unknown account")
			s.sendMagicLinkResponse(w, decoyPollToken())
			return
		}
		// Accepted leak: only an existing account can hit the daily limit, but
		// probing it takes 3 solved captchas per address, and users need to
		// know why no email arrives.
		if errors.Is(err, domain.ErrRateLimitExceeded) {
			s.sendError(w, "daily magic link limit reached", http.StatusTooManyRequests)
			return
		}
		s.sendInternalServerError(w, r, err)
		return
	}

	zerolog.Ctx(r.Context()).Info().Str("email", targetEmail).Str("url", confirmURL("[REDACTED]", req.Origin)).Msg("sending magic link")

	subject := "Your RottenBikes Magic Link"
	if req.Origin == originApp {
		subject = "Your RottenBikes login code"
	}
	body := fmt.Sprintf("Hello,\n\nYou asked to log in to RottenBikes. You can log in %s\n\nIf you did not request this, please ignore this email.", loginInstructions(link, req.Origin))
	s.sendEmailAsync(r, "magic_link", targetEmail, subject, body, "failed to send magic link email")

	// The requesting device gets the poll token. With the code from the email,
	// it can log in through /auth/verify-code.
	s.sendMagicLinkResponse(w, link.PollToken)
}

// originApp is the origin the mobile app sends with its login requests.
const originApp = "mobile"

// loginInstructions is the part of a login email that says how to log in,
// following "you can log in ...".
//
// Requests from the app get only the code. The user reads the email on the
// same phone, and the link would open the browser, not the app (there are no
// universal links), so it cannot log the app in. Web requests get the link,
// for logging in wherever the email is opened, and the code, for logging in
// the device that asked for the email.
func loginInstructions(link domain.MagicLink, origin string) string {
	if origin == originApp {
		return fmt.Sprintf("by entering this code in the RottenBikes app:\n\n%s\n\nThe code expires in 30 minutes.", link.Code)
	}
	return fmt.Sprintf("by clicking the following link:\n\n%s\n\nUsing a different device? Enter this code in RottenBikes on the device where you asked for this email:\n\n%s\n\nThe link and the code expire in 30 minutes.", confirmURL(link.MagicToken, origin), link.Code)
}

func (s *HTTPServer) sendMagicLinkResponse(w http.ResponseWriter, pollToken string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message":     "magic link email sent",
		"magic_token": pollToken,
	})
}

// decoyPollToken returns a poll token indistinguishable from a real one (same
// length and alphabet) that matches no magic link, so no code redeems it.
func decoyPollToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand: %v", err))
	}
	return hex.EncodeToString(b)
}

const defaultCaptchaVerifyURL = "https://api.hcaptcha.com/siteverify"

var (
	// errCaptchaInvalid: hCaptcha rejected the token (the client's fault).
	errCaptchaInvalid = errors.New("invalid captcha")
	// errCaptchaUnavailable: the token could not be verified (our side).
	errCaptchaUnavailable = errors.New("captcha verification unavailable")
)

// verifyCaptcha checks the token against hCaptcha. It returns nil on success,
// an error wrapping errCaptchaInvalid when the token is rejected, and one
// wrapping errCaptchaUnavailable when it could not be verified.
func (s *HTTPServer) verifyCaptcha(ctx context.Context, token, email string) error {
	secret := strings.TrimSpace(os.Getenv("HCAPTCHA_SECRET"))
	appEnv := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))

	if secret == "" {
		if appEnv == "development" || appEnv == "dev" || appEnv == "local" {
			captchaVerificationsTotal.WithLabelValues("skipped").Inc()
			zerolog.Ctx(ctx).Warn().Msg("HCAPTCHA_SECRET not set, skipping verification request in development/local mode")
			return nil
		}
		captchaVerificationsTotal.WithLabelValues("not_configured").Inc()
		zerolog.Ctx(ctx).Error().Msg("HCAPTCHA_SECRET not set, cannot verify captcha in production")
		return fmt.Errorf("%w: HCAPTCHA_SECRET not set", errCaptchaUnavailable)
	}

	vreq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.captchaVerifyURL, strings.NewReader(url.Values{
		"secret":   {secret},
		"response": {token},
	}.Encode()))
	if err != nil {
		captchaVerificationsTotal.WithLabelValues("error").Inc()
		return fmt.Errorf("%w: %v", errCaptchaUnavailable, err)
	}
	vreq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	vresp, err := s.httpClient.Do(vreq)
	if err != nil {
		captchaVerificationsTotal.WithLabelValues("error").Inc()
		zerolog.Ctx(ctx).Error().Err(err).Str("email", email).Msg("hCaptcha request error")
		return fmt.Errorf("%w: %v", errCaptchaUnavailable, err)
	}
	defer vresp.Body.Close()

	if vresp.StatusCode != http.StatusOK {
		captchaVerificationsTotal.WithLabelValues("error").Inc()
		zerolog.Ctx(ctx).Error().Int("status", vresp.StatusCode).Str("email", email).Msg("hCaptcha request error")
		return fmt.Errorf("%w: hCaptcha returned status %d", errCaptchaUnavailable, vresp.StatusCode)
	}

	var vres struct {
		Success     bool     `json:"success"`
		ErrorCodes  []string `json:"error-codes"`
		Hostname    string   `json:"hostname"`
		ChallengeTS string   `json:"challenge_ts"`
	}
	if err := json.NewDecoder(vresp.Body).Decode(&vres); err != nil {
		captchaVerificationsTotal.WithLabelValues("error").Inc()
		zerolog.Ctx(ctx).Error().Err(err).Msg("hCaptcha decode error")
		return fmt.Errorf("%w: decode response: %v", errCaptchaUnavailable, err)
	}

	if !vres.Success {
		captchaVerificationsTotal.WithLabelValues("failure").Inc()
		zerolog.Ctx(ctx).Warn().Str("email", email).Strs("errors", vres.ErrorCodes).Msg("hCaptcha verification FAILED")
		return errCaptchaInvalid
	}
	captchaVerificationsTotal.WithLabelValues("success").Inc()
	zerolog.Ctx(ctx).Info().Str("email", email).Msg("hCaptcha verification SUCCESS")
	return nil
}

func (s *HTTPServer) sendCaptchaError(w http.ResponseWriter, err error) {
	if errors.Is(err, errCaptchaInvalid) {
		s.sendError(w, "invalid captcha", http.StatusForbidden)
		return
	}
	s.sendError(w, "captcha verification unavailable, please try again later", http.StatusServiceUnavailable)
}

// sendEmailAsync sends an email in the background, so that how long sending
// takes cannot tell apart the "account exists" and "account does not exist"
// responses. Failures are logged (failureMsg) and counted in
// emails_sent_total; Shutdown waits for pending sends.
func (s *HTTPServer) sendEmailAsync(r *http.Request, kind, to, subject, body, failureMsg string) {
	logger := zerolog.Ctx(r.Context())
	s.pendingEmails.Add(1)
	go func() {
		defer s.pendingEmails.Done()
		if err := s.sendEmail(kind, to, subject, body); err != nil {
			logger.Error().Err(err).Str("email", to).Msg(failureMsg)
		}
	}()
}

// sendEmail sends through the configured sender and records the outcome.
func (s *HTTPServer) sendEmail(kind, to, subject, body string) error {
	err := s.emailSender.SendEmail(to, subject, body)
	result := "success"
	if err != nil {
		result = "failure"
	}
	emailsSentTotal.WithLabelValues(s.emailSender.Name(), kind, result).Inc()
	return err
}

// confirmURL builds the UI link that confirms a magic token, as emailed to
// the user.
func confirmURL(token, origin string) string {
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
	link := fmt.Sprintf("%s://%s:%s/confirm/%s", scheme, uiHost, uiPort, token)
	if origin != "" {
		link = fmt.Sprintf("%s?origin=%s", link, url.QueryEscape(origin))
	}
	return link
}

type confirmResponse struct {
	APIToken        string    `json:"api_token"`
	Email           string    `json:"email"`
	APITokenExpires time.Time `json:"api_token_expires_at"`
}

// GET /auth/confirm/{token}
func (s *HTTPServer) handleConfirmMagicLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	token := r.PathValue("token")
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
	role, _ := roleFromContext(r.Context())

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"poster_id": posterID,
		"username":  username,
		"is_admin":  role == domain.PosterRoleAdmin,
		"status":    "ok",
	})
}

type verifyLoginCodeRequest struct {
	Token string `json:"token"`
	Code  string `json:"code"`
}

// POST /auth/verify-code
//
// Logs in the device that requested a magic link: it sends its poll token
// (the magic_token from the request response) and the code from the email.
func (s *HTTPServer) handleVerifyLoginCode(w http.ResponseWriter, r *http.Request) {
	var req verifyLoginCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	// People copy codes with spaces or dashes in them.
	code := strings.NewReplacer(" ", "", "-", "").Replace(req.Code)
	if req.Token == "" || code == "" {
		s.sendError(w, "token and code are required", http.StatusBadRequest)
		return
	}
	if !isLoginCode(code) {
		s.sendError(w, domain.ErrInvalidLoginCode.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	res, err := s.service.VerifyLoginCode(ctx, req.Token, code)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidLoginCode) {
			s.sendError(w, domain.ErrInvalidLoginCode.Error(), http.StatusBadRequest)
			return
		}
		s.sendInternalServerError(w, r, err)
		return
	}

	zerolog.Ctx(r.Context()).Info().Str("email", res.Email).Msg("login code accepted")

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(confirmResponse{
		APIToken:        res.APIToken,
		Email:           res.Email,
		APITokenExpires: res.APITokenExpiresAt,
	})
}

// isLoginCode reports whether code has the shape of a login code (6 digits).
func isLoginCode(code string) bool {
	if len(code) != 6 {
		return false
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// POST /auth/logout
//
// Idempotent: logging out a session that is already gone (revoked, expired,
// or deleted with its account) is still a successful logout, so this always
// returns 204 unless revoking fails. It deliberately does not go through
// middlewareAuth: answering 401 here made clients that react to 401 by
// logging out call logout again, in a loop.
func (s *HTTPServer) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token, problem := bearerToken(r)
	if problem != "" {
		// No session to revoke.
		w.WriteHeader(http.StatusNoContent)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if err := s.service.RevokeAPIToken(ctx, token); err != nil {
		s.sendInternalServerError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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
