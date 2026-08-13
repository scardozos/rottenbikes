package domain

import (
	"context"
	"database/sql"
	"errors"

	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCreateMagicLink(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	ctx := context.Background()
	email := "test@example.com"

	t.Run("success_existing_valid_token", func(t *testing.T) {
		validToken := "existing_token"
		validExpires := time.Now().Add(24 * time.Hour)

		mock.ExpectBegin()
		mock.ExpectQuery("SELECT poster_id, api_token, api_token_expires_ts, email FROM posters").
			WithArgs(email).
			WillReturnRows(sqlmock.NewRows([]string{"poster_id", "api_token", "api_token_expires_ts", "email"}).
				AddRow(1, validToken, validExpires, email))

		// Rate limit check
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM magic_links").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		// issueMagicLink: existing token still valid -> refresh expiry only
		mock.ExpectExec("UPDATE posters").
			WithArgs(sqlmock.AnyArg(), 1).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Insert magic link (now carries a poll_token: 4 args)
		mock.ExpectExec("INSERT INTO magic_links").
			WithArgs(1, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectCommit()

		store := NewService(NewStore(db))
		magicToken, pollToken, _, err := store.CreateMagicLink(ctx, email)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if magicToken == "" {
			t.Error("expected magic token to be generated")
		}
		// The poll token is separate and distinct from the magic token.
		if pollToken == "" {
			t.Error("expected poll token to be generated")
		}
		if pollToken == magicToken {
			t.Error("poll token must differ from the magic token")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("user_not_found", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT poster_id, api_token, api_token_expires_ts, email FROM posters").
			WithArgs(email).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectRollback()

		store := NewService(NewStore(db))
		_, _, _, err := store.CreateMagicLink(ctx, email)
		if err == nil {
			t.Error("expected error user not found")
		}
	})

	t.Run("rate_limit_exceeded", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT poster_id, api_token, api_token_expires_ts, email FROM posters").
			WithArgs(email).
			WillReturnRows(sqlmock.NewRows([]string{"poster_id", "api_token", "api_token_expires_ts", "email"}).
				AddRow(1, "some_token", time.Now().Add(time.Hour), email))

		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM magic_links").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

		mock.ExpectRollback()

		store := NewService(NewStore(db))
		_, _, _, err := store.CreateMagicLink(ctx, email)
		if err == nil || !errors.Is(err, ErrRateLimitExceeded) {
			t.Errorf("expected ErrRateLimitExceeded, got %v", err)
		}
	})
}

func TestRegister(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	ctx := context.Background()
	email := "new@example.com"
	username := "newuser"

	t.Run("success", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery("INSERT INTO posters").
			WithArgs(email, username).
			WillReturnRows(sqlmock.NewRows([]string{"poster_id", "api_token", "api_token_expires_ts"}).
				AddRow(1, nil, nil))

		// issueMagicLink: no existing token -> set a new (hashed) api token
		mock.ExpectExec("UPDATE posters").
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 1).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Insert magic link (4 args: poster_id, magic_hash, poll_hash, expires)
		mock.ExpectExec("INSERT INTO magic_links").
			WithArgs(1, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectCommit()

		store := NewService(NewStore(db))
		magicToken, pollToken, err := store.Register(ctx, username, email)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if magicToken == "" {
			t.Error("expected magic token to be generated")
		}
		if pollToken == "" {
			t.Error("expected poll token to be generated")
		}
		if pollToken == magicToken {
			t.Error("poll token must differ from the magic token")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("invalid_email", func(t *testing.T) {
		username := "testuser"
		email := "invalid-email" // Missing @ and domain

		store := NewService(NewStore(db))
		_, _, err := store.Register(ctx, username, email)
		if err == nil {
			t.Error("expected error for invalid email, got nil")
		}
		if err != nil && err.Error() != "invalid email format" {
			t.Errorf("expected 'invalid email format' error, got: %v", err)
		}
	})

	t.Run("invalid_username", func(t *testing.T) {
		username := "test@user" // Contains special character
		email := "test@example.com"

		store := NewService(NewStore(db))
		_, _, err := store.Register(ctx, username, email)
		if err == nil {
			t.Error("expected error for invalid username, got nil")
		}
		if err != nil && err.Error() != "username can only contain letters, numbers and dots" {
			t.Errorf("expected 'username can only contain letters, numbers and dots' error, got: %v", err)
		}
	})
}

func TestConfirmMagicLink(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	ctx := context.Background()
	token := "magic_token"

	t.Run("success_always_rotates", func(t *testing.T) {
		mock.ExpectBegin()

		// Load magic link (FOR UPDATE)
		mock.ExpectQuery("SELECT poster_id, expires_ts, consumed_ts FROM magic_links").
			WithArgs(HashToken(token)).
			WillReturnRows(sqlmock.NewRows([]string{"poster_id", "expires_ts", "consumed_ts"}).
				AddRow(1, time.Now().Add(time.Hour), nil))

		// Always rotate: issue a fresh api token, store only its hash, return
		// expires + email.
		mock.ExpectQuery("UPDATE posters").
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 1).
			WillReturnRows(sqlmock.NewRows([]string{"api_token_expires_ts", "email"}).
				AddRow(time.Now().Add(time.Hour), "test@example.com"))

		// Make the raw api token available for one-time poll retrieval.
		mock.ExpectExec("UPDATE magic_links").
			WithArgs(sqlmock.AnyArg(), HashToken(token)).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectCommit()

		store := NewService(NewStore(db))
		res, err := store.ConfirmMagicLink(ctx, token)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res == nil {
			t.Fatal("expected result")
		}
		// The returned (and poll-retrievable) api token is the RAW 32-byte token
		// (64 hex chars); posters only stores its hash.
		if len(res.APIToken) != 64 {
			t.Errorf("expected raw api token of 64 hex chars, got %d (%q)", len(res.APIToken), res.APIToken)
		}
		if res.Email != "test@example.com" {
			t.Errorf("expected email test@example.com, got %s", res.Email)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("already_consumed", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT poster_id, expires_ts, consumed_ts FROM magic_links").
			WithArgs(HashToken(token)).
			WillReturnRows(sqlmock.NewRows([]string{"poster_id", "expires_ts", "consumed_ts"}).
				AddRow(1, time.Now().Add(time.Hour), time.Now().Add(-time.Minute)))
		mock.ExpectRollback()

		store := NewService(NewStore(db))
		_, err := store.ConfirmMagicLink(ctx, token)
		if err == nil {
			t.Error("expected error for already-consumed token")
		}
	})

	t.Run("expired", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT poster_id, expires_ts, consumed_ts FROM magic_links").
			WithArgs(HashToken(token)).
			WillReturnRows(sqlmock.NewRows([]string{"poster_id", "expires_ts", "consumed_ts"}).
				AddRow(1, time.Now().Add(-time.Hour), nil))
		mock.ExpectRollback()

		store := NewService(NewStore(db))
		_, err := store.ConfirmMagicLink(ctx, token)
		if err == nil {
			t.Error("expected error for expired token")
		}
	})

	t.Run("invalid_token", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT poster_id, expires_ts, consumed_ts FROM magic_links").
			WithArgs(HashToken(token)).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectRollback()

		store := NewService(NewStore(db))
		_, err := store.ConfirmMagicLink(ctx, token)
		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestGetPosterByAPIToken(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	ctx := context.Background()
	token := "api_token"

	t.Run("success", func(t *testing.T) {
		// The bearer is hashed before lookup (posters.api_token stores the hash).
		mock.ExpectQuery("SELECT poster_id, email, username, api_token_expires_ts, email_verified FROM posters").
			WithArgs(HashToken(token)).
			WillReturnRows(sqlmock.NewRows([]string{"poster_id", "email", "username", "api_token_expires_ts", "email_verified"}).
				AddRow(1, "test@example.com", "testuser", time.Now().Add(time.Hour), true))

		store := NewService(NewStore(db))
		poster, err := store.GetPosterByAPIToken(ctx, token)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if poster == nil {
			t.Fatalf("expected poster")
		}
		if poster.Email != "test@example.com" {
			t.Errorf("expected email test@example.com, got %s", poster.Email)
		}
	})

	t.Run("expired_token", func(t *testing.T) {
		mock.ExpectQuery("SELECT poster_id, email, username, api_token_expires_ts, email_verified FROM posters").
			WithArgs(HashToken(token)).
			WillReturnRows(sqlmock.NewRows([]string{"poster_id", "email", "username", "api_token_expires_ts", "email_verified"}).
				AddRow(1, "test@example.com", "testuser", time.Now().Add(-time.Hour), true))

		store := NewService(NewStore(db))
		_, err := store.GetPosterByAPIToken(ctx, token)
		if err == nil {
			t.Error("expected error for expired token")
		}
	})

	t.Run("unverified_email", func(t *testing.T) {
		mock.ExpectQuery("SELECT poster_id, email, username, api_token_expires_ts, email_verified FROM posters").
			WithArgs(HashToken(token)).
			WillReturnRows(sqlmock.NewRows([]string{"poster_id", "email", "username", "api_token_expires_ts", "email_verified"}).
				AddRow(1, "test@example.com", "testuser", time.Now().Add(time.Hour), false))

		store := NewService(NewStore(db))
		_, err := store.GetPosterByAPIToken(ctx, token)
		if err == nil {
			t.Error("expected error for unverified email")
		}
	})

	// A token whose hash is not in the DB (typo / revoked) must not authenticate.
	t.Run("unknown_token", func(t *testing.T) {
		mock.ExpectQuery("SELECT poster_id, email, username, api_token_expires_ts, email_verified FROM posters").
			WithArgs(HashToken("some-other-token")).
			WillReturnError(sql.ErrNoRows)

		store := NewService(NewStore(db))
		_, err := store.GetPosterByAPIToken(ctx, "some-other-token")
		if err == nil {
			t.Error("expected error for unknown token")
		}
	})
}

func TestDeletePoster(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	ctx := context.Background()
	posterID := int64(1)

	t.Run("success_delete_content", func(t *testing.T) {
		mock.ExpectBegin()

		// 1. List user reviews (returns 2 bikes)
		mock.ExpectQuery("SELECT DISTINCT bike_numerical_id FROM reviews").
			WithArgs(posterID).
			WillReturnRows(sqlmock.NewRows([]string{"bike_numerical_id"}).AddRow("101").AddRow("102"))

		// 2. Delete ratings
		mock.ExpectExec("DELETE FROM review_ratings").
			WithArgs(posterID).
			WillReturnResult(sqlmock.NewResult(0, 5))

		// 3. Delete reviews
		mock.ExpectExec("DELETE FROM reviews").
			WithArgs(posterID).
			WillReturnResult(sqlmock.NewResult(0, 2))

		// 4. Recompute aggregates (for bike 101)
		mock.ExpectExec("DELETE FROM rating_aggregates").
			WithArgs("101").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("INSERT INTO rating_aggregates").
			WithArgs("101").
			WillReturnResult(sqlmock.NewResult(0, 1))

		// 4. Recompute aggregates (for bike 102)
		mock.ExpectExec("DELETE FROM rating_aggregates").
			WithArgs("102").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("INSERT INTO rating_aggregates").
			WithArgs("102").
			WillReturnResult(sqlmock.NewResult(0, 1))

		// 5. Delete user bikes
		mock.ExpectExec("DELETE FROM bikes").
			WithArgs(posterID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// 6. Delete magic links
		mock.ExpectExec("DELETE FROM magic_links").
			WithArgs(posterID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// 7. Delete poster
		mock.ExpectExec("DELETE FROM posters").
			WithArgs(posterID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectCommit()

		store := NewService(NewStore(db))
		// deleteContent = true
		err := store.DeletePoster(ctx, posterID, true)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("success_orphan_content", func(t *testing.T) {
		mock.ExpectBegin()

		// 1. Orphan bikes
		mock.ExpectExec("UPDATE bikes SET creator_id = NULL").
			WithArgs(posterID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// 2. Orphan reviews
		mock.ExpectExec("UPDATE reviews SET poster_id = NULL").
			WithArgs(posterID).
			WillReturnResult(sqlmock.NewResult(0, 2))

		// 3. Delete magic links
		mock.ExpectExec("DELETE FROM magic_links").
			WithArgs(posterID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// 4. Delete poster
		mock.ExpectExec("DELETE FROM posters").
			WithArgs(posterID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectCommit()

		store := NewService(NewStore(db))
		// deleteContent = false
		err := store.DeletePoster(ctx, posterID, false)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}
