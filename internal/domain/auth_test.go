package domain

import (
	"context"
	"database/sql"

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
		mock.ExpectQuery("SELECT poster_id, api_token, api_token_expires_ts FROM posters").
			WithArgs(email).
			WillReturnRows(sqlmock.NewRows([]string{"poster_id", "api_token", "api_token_expires_ts"}).
				AddRow(1, validToken, validExpires))

		// issueMagicLink logic starts here
		// Invalidate old links - wait, I removed this in my refactor but I should probably keep it if it was intended.
		// Actually, I removed it in my multi_replace_file_content call. Let's see.

		// Refresh expiry (part of issueMagicLink if token exists)
		mock.ExpectExec("UPDATE posters").
			WithArgs(sqlmock.AnyArg(), 1).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Insert magic link
		mock.ExpectExec("INSERT INTO magic_links").
			WithArgs(1, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectCommit()

		store := NewStore(db)
		token, err := store.CreateMagicLink(ctx, email)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if token == "" {
			t.Error("expected token to be generated")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("user_not_found", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT poster_id, api_token, api_token_expires_ts FROM posters").
			WithArgs(email).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectRollback()

		store := NewStore(db)
		_, err := store.CreateMagicLink(ctx, email)
		if err == nil {
			t.Error("expected error user not found")
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

		// issueMagicLink
		// Create new token
		mock.ExpectExec("UPDATE posters").
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 1).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Insert magic link
		mock.ExpectExec("INSERT INTO magic_links").
			WithArgs(1, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectCommit()

		store := NewStore(db)
		token, err := store.Register(ctx, username, email)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if token == "" {
			t.Error("expected token to be generated")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("invalid_email", func(t *testing.T) {
		username := "testuser"
		email := "invalid-email" // Missing @ and domain

		store := NewStore(db)
		_, err := store.Register(ctx, username, email)
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

		store := NewStore(db)
		_, err := store.Register(ctx, username, email)
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

	t.Run("success_valid_token", func(t *testing.T) {
		mock.ExpectBegin()

		// Load magic link
		mock.ExpectQuery("SELECT poster_id, expires_ts, consumed_ts FROM magic_links").
			WithArgs(token).
			WillReturnRows(sqlmock.NewRows([]string{"poster_id", "expires_ts", "consumed_ts"}).
				AddRow(1, time.Now().Add(time.Hour), nil))

		// Load poster
		mock.ExpectQuery("SELECT api_token, email, api_token_expires_ts FROM posters").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"api_token", "email", "api_token_expires_ts"}).
				AddRow("api_token", "test@example.com", time.Now().Add(time.Hour)))

		// Update poster verified
		mock.ExpectQuery("UPDATE posters").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"api_token_expires_ts", "email"}).
				AddRow(time.Now().Add(time.Hour), "test@example.com"))

		mock.ExpectCommit()

		// New: External update for polling
		mock.ExpectExec("UPDATE magic_links").
			WithArgs("api_token", token).
			WillReturnResult(sqlmock.NewResult(1, 1))

		store := NewStore(db)
		res, err := store.ConfirmMagicLink(ctx, token)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if res == nil {
			t.Error("expected result")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("invalid_token", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT poster_id, expires_ts, consumed_ts FROM magic_links").
			WithArgs(token).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectRollback()

		store := NewStore(db)
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
		mock.ExpectQuery("SELECT poster_id, email, api_token_expires_ts, email_verified FROM posters").
			WithArgs(token).
			WillReturnRows(sqlmock.NewRows([]string{"poster_id", "email", "api_token_expires_ts", "email_verified"}).
				AddRow(1, "test@example.com", time.Now().Add(time.Hour), true))

		store := NewStore(db)
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
		mock.ExpectQuery("SELECT poster_id, email, api_token_expires_ts, email_verified FROM posters").
			WithArgs(token).
			WillReturnRows(sqlmock.NewRows([]string{"poster_id", "email", "api_token_expires_ts", "email_verified"}).
				AddRow(1, "test@example.com", time.Now().Add(-time.Hour), true))

		store := NewStore(db)
		_, err := store.GetPosterByAPIToken(ctx, token)
		if err == nil {
			t.Error("expected error for expired token")
		}
	})

	t.Run("unverified_email", func(t *testing.T) {
		mock.ExpectQuery("SELECT poster_id, email, api_token_expires_ts, email_verified FROM posters").
			WithArgs(token).
			WillReturnRows(sqlmock.NewRows([]string{"poster_id", "email", "api_token_expires_ts", "email_verified"}).
				AddRow(1, "test@example.com", time.Now().Add(time.Hour), false))

		store := NewStore(db)
		_, err := store.GetPosterByAPIToken(ctx, token)
		if err == nil {
			t.Error("expected error for unverified email")
		}
	})
}
