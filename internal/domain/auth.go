package domain

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/lib/pq"
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
	//go:embed sql/check_username_exists.sql
	checkUsernameExistsQuery string
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
	//go:embed sql/get_magic_link_by_poll_token.sql
	getMagicLinkByPollTokenQuery string
	//go:embed sql/increment_login_code_attempts.sql
	incrementLoginCodeAttemptsQuery string
	//go:embed sql/mark_login_code_used.sql
	markLoginCodeUsedQuery string
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
	//go:embed sql/delete_poster_token.sql
	deletePosterTokenQuery string
)

var (
	ErrRateLimitExceeded = errors.New("daily magic link limit reached")
	ErrUserNotFound      = errors.New("user not found")
	ErrInvalidToken      = errors.New("invalid token")
	ErrTokenExpired      = errors.New("token expired")
	ErrEmailNotVerified  = errors.New("email not verified")
	// ErrInvalidLoginCode covers every reason a login code is refused (wrong
	// code, unknown request token, used or expired link, too many attempts),
	// so a response never tells them apart.
	ErrInvalidLoginCode = errors.New("invalid or expired code")
)

// maxLoginCodeAttempts is how many wrong codes a link accepts before it stops
// accepting codes. With a 6-digit code, 5 attempts give a 1 in 200,000 chance
// of guessing it per link, and links are rate limited per account.
const maxLoginCodeAttempts = 5

func randomToken(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// randomLoginCode returns a uniformly random 6-digit code, zero padded.
func randomLoginCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
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

// MagicLink is a freshly issued magic link, with its secrets in the clear.
// Only their hashes are stored.
type MagicLink struct {
	// MagicToken goes in the emailed confirm link. Opening the link logs in
	// the device that opens it.
	MagicToken string
	// Code is the 6-digit login code in the same email. Entered on the
	// requesting device together with PollToken, it logs that device in.
	// The link and the code are each usable once, independently.
	Code string
	// PollToken is returned to the requesting device (the API calls it
	// magic_token for compatibility). On its own it grants nothing: the
	// device also needs the code from the email.
	PollToken string
}

// CreateMagicLink issues a magic link for the poster identified by email OR
// username, and returns it together with the poster's email address.
func (s *Store) CreateMagicLink(ctx context.Context, identifier string) (link MagicLink, email string, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return MagicLink{}, "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var posterID int64
	var userEmail string

	// SELECT poster strictly by email OR username
	err = tx.QueryRowContext(ctx, getPosterQuery, identifier).Scan(&posterID, &userEmail)
	if err != nil {
		if err == sql.ErrNoRows {
			return MagicLink{}, "", ErrUserNotFound
		}
		return MagicLink{}, "", fmt.Errorf("query poster: %w", err)
	}

	// Rate limit: max 2 links per user per 24 hours
	var count int
	err = tx.QueryRowContext(ctx, checkMagicLinkRateLimitQuery, posterID).Scan(&count)
	if err != nil {
		return MagicLink{}, "", fmt.Errorf("check rate limit: %w", err)
	}
	if count >= 2 {
		return MagicLink{}, "", ErrRateLimitExceeded
	}

	link, err = s.issueMagicLink(ctx, tx, posterID)
	if err != nil {
		return MagicLink{}, "", err
	}

	if err := tx.Commit(); err != nil {
		return MagicLink{}, "", fmt.Errorf("commit tx: %w", err)
	}

	return link, userEmail, nil
}

var (
	ErrEmailExists    = errors.New("email already exists")
	ErrUsernameExists = errors.New("username already exists")
)

func (s *Store) Register(ctx context.Context, username, email string) (MagicLink, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return MagicLink{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Check the username first: usernames are public, emails are not. When
	// both are taken the caller must only learn about the username, so that
	// the response never depends on whether the email has an account.
	var usernameTaken bool
	if err := tx.QueryRowContext(ctx, checkUsernameExistsQuery, username).Scan(&usernameTaken); err != nil {
		return MagicLink{}, fmt.Errorf("check username: %w", err)
	}
	if usernameTaken {
		return MagicLink{}, ErrUsernameExists
	}

	var posterID int64

	// Create poster
	err = tx.QueryRowContext(ctx, createPosterQuery, email, username).Scan(&posterID)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			if pqErr.Constraint == "posters_email_key" {
				return MagicLink{}, ErrEmailExists
			}
			if pqErr.Constraint == "posters_username_key" {
				return MagicLink{}, ErrUsernameExists
			}
		}
		return MagicLink{}, fmt.Errorf("insert poster: %w", err)
	}

	link, err := s.issueMagicLink(ctx, tx, posterID)
	if err != nil {
		return MagicLink{}, err
	}

	if err := tx.Commit(); err != nil {
		return MagicLink{}, fmt.Errorf("commit tx: %w", err)
	}

	return link, nil
}

func (s *Store) issueMagicLink(ctx context.Context, tx *sql.Tx, posterID int64) (MagicLink, error) {
	// Issue the one-time magic token and login code (both emailed) and a
	// separate poll token (returned to the requesting device). All three are
	// stored SHA-256 hashed.
	var link MagicLink
	var err error
	if link.MagicToken, err = randomToken(32); err != nil {
		return MagicLink{}, fmt.Errorf("generate magic token: %w", err)
	}
	if link.PollToken, err = randomToken(32); err != nil {
		return MagicLink{}, fmt.Errorf("generate poll token: %w", err)
	}
	if link.Code, err = randomLoginCode(); err != nil {
		return MagicLink{}, fmt.Errorf("generate login code: %w", err)
	}

	expires := time.Now().Add(30 * time.Minute)
	if _, err := tx.ExecContext(ctx, insertMagicLinkQuery, posterID, HashToken(link.MagicToken), HashToken(link.PollToken), HashToken(link.Code), expires); err != nil {
		return MagicLink{}, fmt.Errorf("insert magic link: %w", err)
	}

	return link, nil
}

// Consume magic link, verify, and return api_token.
type ConfirmResult struct {
	APIToken          string
	Email             string
	APITokenExpiresAt time.Time
}

// ConfirmMagicLink consumes the one-time emailed magic token and starts a
// session for the device that opened the link.
func (s *Store) ConfirmMagicLink(ctx context.Context, token string) (*ConfirmResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var linkID, posterID int64
	var expires time.Time
	var consumed sql.NullTime

	err = tx.QueryRowContext(ctx, getMagicLinkQuery, HashToken(token)).Scan(&linkID, &posterID, &expires, &consumed)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("load magic link: %w", err)
	}

	if (consumed.Valid && !consumed.Time.IsZero()) || time.Now().After(expires) {
		return nil, ErrTokenExpired
	}

	res, err := s.startSession(ctx, tx, posterID)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, consumeMagicLinkQuery, linkID); err != nil {
		return nil, fmt.Errorf("consume magic link: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return res, nil
}

// VerifyLoginCode logs in the device that requested a magic link: it holds
// the poll token, and the user types in the code from the email. The code
// works once, whether or not the link was opened. Wrong codes are counted,
// and after maxLoginCodeAttempts the link accepts no more codes. Every
// refusal is ErrInvalidLoginCode.
func (s *Store) VerifyLoginCode(ctx context.Context, pollToken, code string) (*ConfirmResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var linkID, posterID int64
	var codeHash sql.NullString
	var attempts int
	var expires time.Time
	var codeUsed sql.NullTime

	err = tx.QueryRowContext(ctx, getMagicLinkByPollTokenQuery, HashToken(pollToken)).Scan(&linkID, &posterID, &codeHash, &attempts, &expires, &codeUsed)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrInvalidLoginCode
		}
		return nil, fmt.Errorf("load magic link: %w", err)
	}

	// Links issued before login codes existed have no code_hash.
	if !codeHash.Valid || codeUsed.Valid || time.Now().After(expires) || attempts >= maxLoginCodeAttempts {
		return nil, ErrInvalidLoginCode
	}

	if subtle.ConstantTimeCompare([]byte(HashToken(code)), []byte(codeHash.String)) != 1 {
		if _, err := tx.ExecContext(ctx, incrementLoginCodeAttemptsQuery, linkID); err != nil {
			return nil, fmt.Errorf("count login code attempt: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit tx: %w", err)
		}
		return nil, ErrInvalidLoginCode
	}

	res, err := s.startSession(ctx, tx, posterID)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, markLoginCodeUsedQuery, linkID); err != nil {
		return nil, fmt.Errorf("mark login code used: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return res, nil
}

// startSession verifies the poster's email and starts a new session in tx.
// It returns the RAW api token; only its SHA-256 hash is stored (in
// poster_tokens).
func (s *Store) startSession(ctx context.Context, tx *sql.Tx, posterID int64) (*ConfirmResult, error) {
	tok, err := randomToken(32)
	if err != nil {
		return nil, fmt.Errorf("generate api token: %w", err)
	}
	exp := time.Now().AddDate(0, 2, 0)

	var apiTokenExpiresAt time.Time
	var email string
	if err := tx.QueryRowContext(ctx, updatePosterVerifiedNewTokenQuery, HashToken(tok), exp, posterID).Scan(&apiTokenExpiresAt, &email); err != nil {
		return nil, fmt.Errorf("start session: %w", err)
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
	Role     PosterRole
	// IsTest marks accounts used by the E2E suite; their bikes are hidden
	// from everyone else's listings.
	IsTest bool
}

// GetPosterByAPIToken returns the poster for a valid, non-expired token.
// The stored token is a SHA-256 hash, so the incoming bearer is hashed before lookup.
func (s *Store) GetPosterByAPIToken(ctx context.Context, token string) (*AuthPoster, error) {
	var p AuthPoster
	var role string
	var expires sql.NullTime
	var emailVerified bool

	err := s.db.QueryRowContext(ctx, getPosterByTokenQuery, HashToken(token)).Scan(&p.PosterID, &p.Email, &p.Username, &role, &expires, &emailVerified, &p.IsTest)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("load poster by token: %w", err)
	}
	p.Role = PosterRole(role)

	if !emailVerified {
		return nil, ErrEmailNotVerified
	}

	if !expires.Valid || time.Now().After(expires.Time) {
		return nil, ErrTokenExpired
	}

	return &p, nil
}

// RevokeAPIToken deletes the given API token session from poster_tokens.
func (s *Store) RevokeAPIToken(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, deletePosterTokenQuery, HashToken(token))
	if err != nil {
		return fmt.Errorf("revoke token: %w", err)
	}
	return nil
}

// DeletePoster removes a poster and (depending on deleteContent) their
// content. When audit is non-nil, a moderation_actions row is written inside
// the same transaction, so admin purges are logged atomically.
func (s *Store) DeletePoster(ctx context.Context, posterID int64, deleteContent bool, audit *ModerationAudit) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Admin purge path (audit != nil): refuse to purge another admin, and
	// surface a missing target as sql.ErrNoRows (→ 404) instead of a silent
	// no-op. The row lock serializes with concurrent promote/demote, so there
	// is no TOCTOU window on the role check. The self-service path (audit ==
	// nil) keeps its current behavior: you may always delete your own account.
	if audit != nil {
		var targetRole string
		if err := tx.QueryRowContext(ctx, getPosterRoleForPurgeQuery, posterID).Scan(&targetRole); err != nil {
			if err == sql.ErrNoRows {
				return sql.ErrNoRows
			}
			return fmt.Errorf("load target poster role: %w", err)
		}
		if PosterRole(targetRole) == PosterRoleAdmin {
			return ErrCannotPurgeAdmin
		}
	}

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

	// Optional audit row, inside the same tx (admin purge path)
	if audit != nil {
		if _, err := tx.ExecContext(ctx, insertModerationActionQuery, audit.AdminPosterID, audit.Action, audit.TargetPosterID); err != nil {
			return fmt.Errorf("insert moderation action: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}
