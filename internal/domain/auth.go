package domain

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	//go:embed sql/get_poster.sql
	getPosterQuery string
	//go:embed sql/check_magic_link_rate_limit.sql
	checkMagicLinkRateLimitQuery string
	//go:embed sql/insert_magic_link.sql
	insertMagicLinkQuery string
	//go:embed sql/create_poster.sql
	createPosterQuery string
	//go:embed sql/update_poster_token.sql
	updatePosterTokenQuery string
	//go:embed sql/update_poster_token_expiry.sql
	updatePosterTokenExpiryQuery string
	//go:embed sql/get_magic_link.sql
	getMagicLinkQuery string
	//go:embed sql/update_poster_verified_new_token.sql
	updatePosterVerifiedNewTokenQuery string
	//go:embed sql/consume_magic_link.sql
	consumeMagicLinkQuery string
	//go:embed sql/get_poster_by_token.sql
	getPosterByTokenQuery string
	//go:embed sql/check_magic_link_status.sql
	checkMagicLinkStatusQuery string
	//go:embed sql/list_user_reviews_for_delete.sql
	listUserReviewsForDeleteQuery string
	//go:embed sql/delete_user_ratings.sql
	deleteUserRatingsQuery string
	//go:embed sql/delete_user_reviews.sql
	deleteUserReviewsQuery string
	//go:embed sql/delete_user_bikes.sql
	deleteUserBikesQuery string
	//go:embed sql/orphan_bikes.sql
	orphanBikesQuery string
	//go:embed sql/orphan_reviews.sql
	orphanReviewsQuery string
	//go:embed sql/delete_user_magic_links.sql
	deleteUserMagicLinksQuery string
	//go:embed sql/delete_user_poster.sql
	deleteUserPosterQuery string
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

func HashToken(token string) string {
	h := sha256.New()
	h.Write([]byte(token))
	return hex.EncodeToString(h.Sum(nil))
}

type Poster struct {
	PosterID          int64
	Email             string
	Username          string
	APIToken          *string
	APITokenExpiresAt *time.Time
	EmailVerified     bool
}

// CreateMagicLink issues a magic link for the poster identified by email OR
// username. It returns:
//   - magicToken: the RAW one-time token embedded in the emailed confirm link
//     (only the email recipient ever sees it; it confirm()s the link).
//   - pollToken: the RAW per-request token returned to the requesting device,
//     used to poll for the api token once the link is confirmed on another
//     device. The poll token is decoupled from the magic token so that whoever
//     requests a link cannot confirm it, and the email recipient cannot poll.
func (s *Store) CreateMagicLink(ctx context.Context, identifier string) (magicToken string, pollToken string, email string, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", "", "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var posterID int64
	var apiToken *string
	var apiTokenExpires sql.NullTime
	var userEmail string

	// SELECT poster strictly by email OR username
	err = tx.QueryRowContext(ctx, getPosterQuery, identifier).Scan(&posterID, &apiToken, &apiTokenExpires, &userEmail)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", "", ErrUserNotFound
		}
		return "", "", "", fmt.Errorf("query poster: %w", err)
	}

	// Rate limit: max 2 links per user per 24 hours
	var count int
	err = tx.QueryRowContext(ctx, checkMagicLinkRateLimitQuery, posterID).Scan(&count)
	if err != nil {
		return "", "", "", fmt.Errorf("check rate limit: %w", err)
	}
	if count >= 2 {
		return "", "", "", ErrRateLimitExceeded
	}

	magicToken, pollToken, err = s.issueMagicLink(ctx, tx, posterID, apiToken, apiTokenExpires)
	if err != nil {
		return "", "", "", err
	}

	if err := tx.Commit(); err != nil {
		return "", "", "", fmt.Errorf("commit tx: %w", err)
	}

	return magicToken, pollToken, userEmail, nil
}

func (s *Store) Register(ctx context.Context, username, email string) (string, string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var posterID int64
	var apiToken *string
	var apiTokenExpires sql.NullTime

	// Create poster
	err = tx.QueryRowContext(ctx, createPosterQuery, email, username).Scan(&posterID, &apiToken, &apiTokenExpires)
	if err != nil {
		if strings.Contains(err.Error(), "posters_email_key") {
			return "", "", fmt.Errorf("email already exists")
		}
		if strings.Contains(err.Error(), "posters_username_key") {
			return "", "", fmt.Errorf("username already exists")
		}
		return "", "", fmt.Errorf("insert poster: %w", err)
	}

	magicToken, pollToken, err := s.issueMagicLink(ctx, tx, posterID, apiToken, apiTokenExpires)
	if err != nil {
		return "", "", err
	}

	if err := tx.Commit(); err != nil {
		return "", "", fmt.Errorf("commit tx: %w", err)
	}

	return magicToken, pollToken, nil
}

func (s *Store) issueMagicLink(ctx context.Context, tx *sql.Tx, posterID int64, apiToken *string, apiTokenExpires sql.NullTime) (magicToken string, pollToken string, err error) {

	now := time.Now()
	needNewToken := true
	if apiToken != nil && apiTokenExpires.Valid && apiTokenExpires.Time.After(now) {
		// existing token still valid; keep it but refresh expiry
		needNewToken = false
	}

	if needNewToken {
		tok, err := randomToken(32)
		if err != nil {
			return "", "", fmt.Errorf("generate api token: %w", err)
		}
		exp := now.AddDate(0, 2, 0) // +2 months
		// Store the SHA-256 hash of the api token; the raw token is only ever
		// handed to a client at confirm time and is never persisted in posters.
		if _, err := tx.ExecContext(ctx, updatePosterTokenQuery, HashToken(tok), exp, posterID); err != nil {
			return "", "", fmt.Errorf("set api token: %w", err)
		}
		apiToken = &tok
	} else {
		// refresh expiry on existing token
		exp := now.AddDate(0, 2, 0)
		if _, err := tx.ExecContext(ctx, updatePosterTokenExpiryQuery, exp, posterID); err != nil {
			return "", "", fmt.Errorf("refresh api token expiry: %w", err)
		}
	}

	// issue one-time magic token (emailed) and a separate poll token (returned to
	// the requesting device). Both are stored SHA-256 hashed.
	magicToken, err = randomToken(32)
	if err != nil {
		return "", "", fmt.Errorf("generate magic token: %w", err)
	}
	pollToken, err = randomToken(32)
	if err != nil {
		return "", "", fmt.Errorf("generate poll token: %w", err)
	}

	hashedMagic := HashToken(magicToken)
	hashedPoll := HashToken(pollToken)
	expires := now.Add(30 * time.Minute)
	if _, err := tx.ExecContext(ctx, insertMagicLinkQuery, posterID, hashedMagic, hashedPoll, expires); err != nil {
		return "", "", fmt.Errorf("insert magic link: %w", err)
	}

	return magicToken, pollToken, nil
}

// Consume magic link, verify, and return api_token.
type ConfirmResult struct {
	APIToken          string
	Email             string
	APITokenExpiresAt time.Time
}

// ConfirmMagicLink consumes the one-time emailed magic token, rotates the
// poster's api token (always issuing a fresh one so we never have to return a
// token we only have on file as a hash), and returns the RAW api token.
//
// The raw token is stored transiently in magic_links.api_token (gated by the
// poll token + expiry) so the requesting device can retrieve it via
// CheckMagicLinkStatus. posters.api_token only ever stores its SHA-256 hash.
func (s *Store) ConfirmMagicLink(ctx context.Context, token string) (*ConfirmResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var posterID int64
	var expires time.Time
	var consumed sql.NullTime

	hashedToken := HashToken(token)

	err = tx.QueryRowContext(ctx, getMagicLinkQuery, hashedToken).Scan(&posterID, &expires, &consumed)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid token")
		}
		return nil, fmt.Errorf("load magic link: %w", err)
	}

	if (consumed.Valid && !consumed.Time.IsZero()) || time.Now().After(expires) {
		return nil, fmt.Errorf("token expired or already used")
	}

	// Always rotate: generate a fresh raw api token, persist only its hash.
	tok, err := randomToken(32)
	if err != nil {
		return nil, fmt.Errorf("generate api token: %w", err)
	}
	now := time.Now()
	exp := now.AddDate(0, 2, 0)

	var apiTokenExpiresAt time.Time
	var email string
	if err := tx.QueryRowContext(ctx, updatePosterVerifiedNewTokenQuery, HashToken(tok), exp, posterID).Scan(&apiTokenExpiresAt, &email); err != nil {
		return nil, fmt.Errorf("rotate api token: %w", err)
	}

	// Make the raw api token available for one-time poll retrieval (gated by the
	// poll token + expiry). consume_magic_link matches on magic_links.token and
	// marks the link consumed.
	if _, err := tx.ExecContext(ctx, consumeMagicLinkQuery, tok, hashedToken); err != nil {
		return nil, fmt.Errorf("consume magic link: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &ConfirmResult{
		APIToken:          tok,
		Email:             email,
		APITokenExpiresAt: apiTokenExpiresAt,
	}, nil
}

type AuthPoster struct {
	PosterID int64
	Email    string
	Username string
}

// GetPosterByAPIToken returns the poster for a valid, non-expired token.
// The stored token is a SHA-256 hash, so the incoming bearer is hashed before lookup.
func (s *Store) GetPosterByAPIToken(ctx context.Context, token string) (*AuthPoster, error) {
	var p AuthPoster
	var expires sql.NullTime
	var emailVerified bool

	err := s.db.QueryRowContext(ctx, getPosterByTokenQuery, HashToken(token)).Scan(&p.PosterID, &p.Email, &p.Username, &expires, &emailVerified)
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

// CheckMagicLinkStatus returns the api_token if the link was confirmed using the
// given poll token, otherwise an empty string. The poll token is stored hashed,
// so the incoming token is hashed before lookup. The link must also be unexpired.
func (s *Store) CheckMagicLinkStatus(ctx context.Context, token string) (string, error) {
	var apiToken sql.NullString
	err := s.db.QueryRowContext(ctx, checkMagicLinkStatusQuery, HashToken(token)).Scan(&apiToken)
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
		rows, err := tx.QueryContext(ctx, listUserReviewsForDeleteQuery, posterID)
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
		if _, err := tx.ExecContext(ctx, deleteUserRatingsQuery, posterID); err != nil {
			return fmt.Errorf("delete user ratings: %w", err)
		}

		// 3. Delete reviews
		if _, err := tx.ExecContext(ctx, deleteUserReviewsQuery, posterID); err != nil {
			return fmt.Errorf("delete user reviews: %w", err)
		}

		// 4. Recompute aggregates for affected bikes
		for _, bid := range bikeIDs {
			if err := RecomputeAggregatesForBike(ctx, tx, bid); err != nil {
				return fmt.Errorf("recompute aggregates for bike %s: %w", bid, err)
			}
		}

		// 5. Delete bikes created by user
		if _, err := tx.ExecContext(ctx, deleteUserBikesQuery, posterID); err != nil {
			return fmt.Errorf("delete user bikes: %w", err)
		}

	} else {
		// Orchid mode: set poster_id/creator_id to NULL
		if _, err := tx.ExecContext(ctx, orphanBikesQuery, posterID); err != nil {
			return fmt.Errorf("orphan bikes: %w", err)
		}

		if _, err := tx.ExecContext(ctx, orphanReviewsQuery, posterID); err != nil {
			return fmt.Errorf("orphan reviews: %w", err)
		}
	}

	// Always delete magic links
	if _, err := tx.ExecContext(ctx, deleteUserMagicLinksQuery, posterID); err != nil {
		return fmt.Errorf("delete magic links: %w", err)
	}

	// Always delete poster
	if _, err := tx.ExecContext(ctx, deleteUserPosterQuery, posterID); err != nil {
		return fmt.Errorf("delete poster: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}
