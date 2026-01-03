package domain

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"
)

var (
	ErrRateLimitExceeded = errors.New("daily magic link limit reached")
	ErrUserNotFound      = errors.New("user not found")
)

func randomToken(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

type Poster struct {
	PosterID          int64
	Email             string
	Username          string
	APIToken          *string
	APITokenExpiresAt *time.Time
	EmailVerified     bool
}

// Create or load poster by email or username, ensure a long-lived api_token exists,
// and issue a single-use magic link token.
func (s *Store) CreateMagicLink(ctx context.Context, identifier string) (magicToken string, email string, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var posterID int64
	var apiToken *string
	var apiTokenExpires sql.NullTime
	var userEmail string

	// SELECT poster strictly by email OR username
	err = tx.QueryRowContext(ctx, `
		SELECT poster_id, api_token, api_token_expires_ts, email
		FROM posters
		WHERE email = $1 OR username = $1
	`, identifier).Scan(&posterID, &apiToken, &apiTokenExpires, &userEmail)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", ErrUserNotFound
		}
		return "", "", fmt.Errorf("query poster: %w", err)
	}

	// Rate limit: max 2 links per user per 24 hours
	var count int
	err = tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM magic_links
		WHERE poster_id = $1 AND created_ts > NOW() - INTERVAL '24 hours'
	`, posterID).Scan(&count)
	if err != nil {
		return "", "", fmt.Errorf("check rate limit: %w", err)
	}
	if count >= 2 {
		return "", "", ErrRateLimitExceeded
	}

	magicToken, err = s.issueMagicLink(ctx, tx, posterID, apiToken, apiTokenExpires)
	if err != nil {
		return "", "", err
	}

	if err := tx.Commit(); err != nil {
		return "", "", fmt.Errorf("commit tx: %w", err)
	}

	return magicToken, userEmail, nil
}

func (s *Store) Register(ctx context.Context, username, email string) (string, error) {
	// Validate email format
	_, err := mail.ParseAddress(email)
	if err != nil {
		return "", fmt.Errorf("invalid email format")
	}

	// Validate username format (alphanumeric and dots only)
	validUsername := regexp.MustCompile(`^[a-zA-Z0-9.]+$`)
	if !validUsername.MatchString(username) {
		return "", fmt.Errorf("username can only contain letters, numbers and dots")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var posterID int64
	var apiToken *string
	var apiTokenExpires sql.NullTime

	// Create poster
	err = tx.QueryRowContext(ctx, `
		INSERT INTO posters (email, username)
		VALUES ($1, $2)
		RETURNING poster_id, api_token, api_token_expires_ts
	`, email, username).Scan(&posterID, &apiToken, &apiTokenExpires)
	if err != nil {
		if strings.Contains(err.Error(), "posters_email_key") {
			return "", fmt.Errorf("email already exists")
		}
		if strings.Contains(err.Error(), "posters_username_key") {
			return "", fmt.Errorf("username already exists")
		}
		return "", fmt.Errorf("insert poster: %w", err)
	}

	magicToken, err := s.issueMagicLink(ctx, tx, posterID, apiToken, apiTokenExpires)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit tx: %w", err)
	}

	return magicToken, nil
}

func (s *Store) issueMagicLink(ctx context.Context, tx *sql.Tx, posterID int64, apiToken *string, apiTokenExpires sql.NullTime) (string, error) {

	now := time.Now()
	needNewToken := true
	if apiToken != nil && apiTokenExpires.Valid && apiTokenExpires.Time.After(now) {
		// existing token still valid; keep it but refresh expiry
		needNewToken = false
	}

	if needNewToken {
		tok, err := randomToken(32)
		if err != nil {
			return "", fmt.Errorf("generate api token: %w", err)
		}
		exp := now.AddDate(0, 2, 0) // +2 months
		if _, err := tx.ExecContext(ctx, `
			UPDATE posters
			SET api_token = $1, api_token_expires_ts = $2
			WHERE poster_id = $3
		`, tok, exp, posterID); err != nil {
			return "", fmt.Errorf("set api token: %w", err)
		}
		apiToken = &tok
	} else {
		// refresh expiry on existing token
		exp := now.AddDate(0, 2, 0)
		if _, err := tx.ExecContext(ctx, `
			UPDATE posters
			SET api_token_expires_ts = $1
			WHERE poster_id = $2
		`, exp, posterID); err != nil {
			return "", fmt.Errorf("refresh api token expiry: %w", err)
		}
	}

	// issue one-time magic token
	magicToken, err := randomToken(32)
	if err != nil {
		return "", fmt.Errorf("generate magic token: %w", err)
	}

	expires := now.Add(30 * time.Minute)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO magic_links (poster_id, token, expires_ts)
		VALUES ($1, $2, $3)
	`, posterID, magicToken, expires); err != nil {
		return "", fmt.Errorf("insert magic link: %w", err)
	}

	return magicToken, nil
}

// Consume magic link, verify, and return api_token.
type ConfirmResult struct {
	APIToken          string
	Email             string
	APITokenExpiresAt time.Time
}

// Consume magic link, verify, and return api_token.
func (s *Store) ConfirmMagicLink(ctx context.Context, token string) (*ConfirmResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var posterID int64
	var expires time.Time
	var consumed sql.NullTime

	err = tx.QueryRowContext(ctx, `
		SELECT poster_id, expires_ts, consumed_ts
		FROM magic_links
		WHERE token = $1
		FOR UPDATE
	`, token).Scan(&posterID, &expires, &consumed)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid token")
		}
		return nil, fmt.Errorf("load magic link: %w", err)
	}

	if (consumed.Valid && !consumed.Time.IsZero()) || time.Now().After(expires) {
		return nil, fmt.Errorf("token expired or already used")
	}

	// We will update the magic_links table with the api_token AFTER we retrieve/generate it.
	// This happens after line 210 in the original file.

	// Get current token info
	var apiToken string
	var email string
	var apiTokenExpires sql.NullTime

	err = tx.QueryRowContext(ctx, `
		SELECT api_token, email, api_token_expires_ts
		FROM posters
		WHERE poster_id = $1
		FOR UPDATE
	`, posterID).Scan(&apiToken, &email, &apiTokenExpires)
	if err != nil {
		return nil, fmt.Errorf("load poster: %w", err)
	}

	now := time.Now()
	if apiToken == "" || !apiTokenExpires.Valid || apiTokenExpires.Time.Before(now) {
		// issue a new token valid for 2 months
		tok, err := randomToken(32)
		if err != nil {
			return nil, fmt.Errorf("generate api token: %w", err)
		}
		exp := now.AddDate(0, 2, 0)
		if err := tx.QueryRowContext(ctx, `
			UPDATE posters
			SET api_token = $1, api_token_expires_ts = $2, email_verified = TRUE
			WHERE poster_id = $3
			RETURNING api_token, api_token_expires_ts, email
		`, tok, exp, posterID).Scan(&apiToken, &apiTokenExpires.Time, &email); err != nil {
			return nil, fmt.Errorf("update poster with new token: %w", err)
		}
		apiTokenExpires.Valid = true
	} else {
		// token exists and is valid; ensure email_verified is set
		if err := tx.QueryRowContext(ctx, `
			UPDATE posters
			SET email_verified = TRUE
			WHERE poster_id = $1
			RETURNING api_token_expires_ts, email
		`, posterID).Scan(&apiTokenExpires.Time, &email); err != nil {
			return nil, fmt.Errorf("update poster verified: %w", err)
		}
		apiTokenExpires.Valid = true
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	// Update magic_links table to store the api_token AND mark as consumed
	// This makes it available for the polling endpoint.
	if _, err := s.db.ExecContext(ctx, `
		UPDATE magic_links
		SET consumed_ts = NOW(), api_token = $1
		WHERE token = $2
	`, apiToken, token); err != nil {
	}

	return &ConfirmResult{
		APIToken:          apiToken,
		Email:             email,
		APITokenExpiresAt: apiTokenExpires.Time,
	}, nil
}

type AuthPoster struct {
	PosterID int64
	Email    string
	Username string
}

// GetPosterByAPIToken returns the poster for a valid, non-expired token.
func (s *Store) GetPosterByAPIToken(ctx context.Context, token string) (*AuthPoster, error) {
	var p AuthPoster
	var expires sql.NullTime
	var emailVerified bool

	err := s.db.QueryRowContext(ctx, `
		SELECT poster_id, email, username, api_token_expires_ts, email_verified
		FROM posters
		WHERE api_token = $1
	`, token).Scan(&p.PosterID, &p.Email, &p.Username, &expires, &emailVerified)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid token")
		}
		return nil, fmt.Errorf("load poster by token: %w", err)
	}

	if !emailVerified {
		return nil, fmt.Errorf("email not verified")
	}

	if !expires.Valid || time.Now().After(expires.Time) {
		return nil, fmt.Errorf("token expired")
	}

	return &p, nil
}

// CheckMagicLinkStatus returns the api_token if the link was confirmed, otherwise empty.
func (s *Store) CheckMagicLinkStatus(ctx context.Context, token string) (string, error) {
	var apiToken sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT api_token
		FROM magic_links
		WHERE token = $1 AND consumed_ts IS NOT NULL
	`, token).Scan(&apiToken)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return apiToken.String, nil
}

func (s *Store) DeletePoster(ctx context.Context, posterID int64, deleteContent bool) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Identify bikes that will need aggregate recomputation
	// (Only if we are deleting content OR if we want to be safe, but actually
	// if we orphan reviews, aggregates don't strictly change unless we remove ratings.
	// If deleteContent=false, we keep reviews/ratings but unset poster_id.
	// Aggregates remain valid as they are sum of ratings.
	// If deleteContent=true, we delete ratings, so we MUST recompute.
	if deleteContent {
		rows, err := tx.QueryContext(ctx, `
			SELECT DISTINCT bike_numerical_id
			FROM reviews
			WHERE poster_id = $1
		`, posterID)
		if err != nil {
			return fmt.Errorf("list user reviews: %w", err)
		}
		var bikeIDs []string
		for rows.Next() {
			var bid string
			if err := rows.Scan(&bid); err != nil {
				rows.Close()
				return err
			}
			bikeIDs = append(bikeIDs, bid)
		}
		rows.Close()

		// 2. Delete review ratings
		if _, err := tx.ExecContext(ctx, `
			DELETE FROM review_ratings
			WHERE review_id IN (SELECT review_id FROM reviews WHERE poster_id = $1)
		`, posterID); err != nil {
			return fmt.Errorf("delete user ratings: %w", err)
		}

		// 3. Delete reviews
		if _, err := tx.ExecContext(ctx, `
			DELETE FROM reviews
			WHERE poster_id = $1
		`, posterID); err != nil {
			return fmt.Errorf("delete user reviews: %w", err)
		}

		// 4. Recompute aggregates for affected bikes
		for _, bid := range bikeIDs {
			if err := RecomputeAggregatesForBike(ctx, tx, bid); err != nil {
				return fmt.Errorf("recompute aggregates for bike %s: %w", bid, err)
			}
		}

		// 5. Delete bikes created by user
		if _, err := tx.ExecContext(ctx, `
			DELETE FROM bikes
			WHERE creator_id = $1
		`, posterID); err != nil {
			return fmt.Errorf("delete user bikes: %w", err)
		}

	} else {
		// Orchid mode: set poster_id/creator_id to NULL
		if _, err := tx.ExecContext(ctx, `
			UPDATE bikes
			SET creator_id = NULL
			WHERE creator_id = $1
		`, posterID); err != nil {
			return fmt.Errorf("orphan bikes: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE reviews
			SET poster_id = NULL
			WHERE poster_id = $1
		`, posterID); err != nil {
			return fmt.Errorf("orphan reviews: %w", err)
		}
	}

	// Always delete magic links
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM magic_links
		WHERE poster_id = $1
	`, posterID); err != nil {
		return fmt.Errorf("delete magic links: %w", err)
	}

	// Always delete poster
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM posters
		WHERE poster_id = $1
	`, posterID); err != nil {
		return fmt.Errorf("delete poster: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}
