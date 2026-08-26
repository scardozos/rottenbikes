package domain

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCheckMagicLinkStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	ctx := context.Background()
	pollToken := "the-poll-token"

	t.Run("confirmed_returns_api_token", func(t *testing.T) {
		// The poll token is stored hashed, so the store hashes the incoming
		// token before lookup.
		mock.ExpectQuery("WITH target AS").
			WithArgs(HashToken(pollToken)).
			WillReturnRows(sqlmock.NewRows([]string{"api_token"}).AddRow("returned-api-token"))

		store := NewService(NewStore(db))
		got, err := store.CheckMagicLinkStatus(ctx, pollToken)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "returned-api-token" {
			t.Errorf("expected returned-api-token, got %q", got)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("not_confirmed_returns_empty", func(t *testing.T) {
		// No row: link not consumed yet, expired, or poll token unknown.
		mock.ExpectQuery("WITH target AS").
			WithArgs(HashToken(pollToken)).
			WillReturnError(sql.ErrNoRows)

		store := NewService(NewStore(db))
		got, err := store.CheckMagicLinkStatus(ctx, pollToken)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "" {
			t.Errorf("expected empty token for unconfirmed link, got %q", got)
		}
	})

	t.Run("the_emailed_magic_token_cannot_poll", func(t *testing.T) {
		// A client that only holds the emailed MAGIC token (not the poll token)
		// hashes to a different value, which the poll column does not match.
		magicToken := "the-emailed-magic-token"
		mock.ExpectQuery("WITH target AS").
			WithArgs(HashToken(magicToken)).
			WillReturnError(sql.ErrNoRows)

		store := NewService(NewStore(db))
		got, err := store.CheckMagicLinkStatus(ctx, magicToken)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "" {
			t.Errorf("magic token must not be able to poll; got %q", got)
		}
	})

	t.Run("db_error_propagates", func(t *testing.T) {
		boom := errors.New("connection lost")
		mock.ExpectQuery("WITH target AS").
			WithArgs(HashToken(pollToken)).
			WillReturnError(boom)

		store := NewService(NewStore(db))
		_, err := store.CheckMagicLinkStatus(ctx, pollToken)
		if err == nil {
			t.Fatal("expected error to propagate")
		}
	})
}
