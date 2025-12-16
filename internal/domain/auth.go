package domain

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
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

// Create or load poster by email, ensure a long-lived api_token exists,
// and issue a single-use magic link token.
func CreateMagicLink(ctx context.Context, db *sql.DB, email string) (magicToken string, err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var posterID int64
	var apiToken *string
	var apiTokenExpires sql.NullTime

	// Upsert or select poster
	err = tx.QueryRowContext(ctx, `
		INSERT INTO posters (email, username)
		VALUES ($1, $1)
		ON CONFLICT (email) DO UPDATE SET email = EXCLUDED.email
		RETURNING poster_id, api_token, api_token_expires_ts
	`, email).Scan(&posterID, &apiToken, &apiTokenExpires)
	if err != nil {
		return "", fmt.Errorf("upsert poster: %w", err)
	}

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

	// Invalidate all previous unconsumed magic links for this poster
	if _, err := tx.ExecContext(ctx, `
		UPDATE magic_links
		SET consumed_ts = NOW()
		WHERE poster_id = $1
		  AND consumed_ts IS NULL
	`, posterID); err != nil {
		return "", fmt.Errorf("invalidate old magic links: %w", err)
	}

	// issue one-time magic token
	magicToken, err = randomToken(32)
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

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit tx: %w", err)
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
func ConfirmMagicLink(ctx context.Context, db *sql.DB, token string) (*ConfirmResult, error) {
	tx, err := db.BeginTx(ctx, nil)
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

	if _, err := tx.ExecContext(ctx, `
		UPDATE magic_links
		SET consumed_ts = NOW()
		WHERE token = $1
	`, token); err != nil {
		return nil, fmt.Errorf("consume magic link: %w", err)
	}

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

	return &ConfirmResult{
		APIToken:          apiToken,
		Email:             email,
		APITokenExpiresAt: apiTokenExpires.Time,
	}, nil
}

type AuthPoster struct {
	PosterID int64
	Email    string
}

// GetPosterByAPIToken returns the poster for a valid, non-expired token.
func GetPosterByAPIToken(ctx context.Context, db *sql.DB, token string) (*AuthPoster, error) {
	var p AuthPoster
	var expires sql.NullTime
	var emailVerified bool

	err := db.QueryRowContext(ctx, `
		SELECT poster_id, email, api_token_expires_ts, email_verified
		FROM posters
		WHERE api_token = $1
	`, token).Scan(&p.PosterID, &p.Email, &expires, &emailVerified)
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
