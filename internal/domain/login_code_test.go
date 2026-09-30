package domain

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

// capturedArg is a sqlmock argument matcher that accepts any value and
// records it, for asserting on generated values (tokens, hashes) afterwards.
type capturedArg struct{ value any }

func (c *capturedArg) Match(v driver.Value) bool {
	c.value = v
	return true
}

// expectIssuedLink checks a freshly issued magic link against the hashes
// that were stored for it: three distinct secrets, only their hashes stored.
func expectIssuedLink(t *testing.T, link MagicLink, magicHash, pollHash, codeHash *capturedArg) {
	t.Helper()
	if len(link.MagicToken) != 64 || len(link.PollToken) != 64 {
		t.Errorf("expected 64-hex-char tokens, got magic %q, poll %q", link.MagicToken, link.PollToken)
	}
	if link.PollToken == link.MagicToken {
		t.Error("poll token must differ from the magic token")
	}
	if !regexp.MustCompile(`^\d{6}$`).MatchString(link.Code) {
		t.Errorf("expected a 6-digit login code, got %q", link.Code)
	}
	if magicHash.value != HashToken(link.MagicToken) {
		t.Error("expected the magic token to be stored hashed")
	}
	if pollHash.value != HashToken(link.PollToken) {
		t.Error("expected the poll token to be stored hashed")
	}
	if codeHash.value != HashToken(link.Code) {
		t.Error("expected the login code to be stored hashed")
	}
}

func TestRandomLoginCode(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		code, err := randomLoginCode()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !regexp.MustCompile(`^\d{6}$`).MatchString(code) {
			t.Fatalf("expected 6 digits, got %q", code)
		}
		seen[code] = true
	}
	// 200 draws from a million: a handful of collisions at most.
	if len(seen) < 190 {
		t.Errorf("codes do not look random: %d distinct out of 200", len(seen))
	}
}

func TestVerifyLoginCode(t *testing.T) {
	ctx := context.Background()
	const pollToken = "the-poll-token"
	const code = "123456"

	linkColumns := []string{"id", "poster_id", "code_hash", "code_attempts", "expires_ts", "code_used_ts"}
	const lookup = "SELECT id, poster_id, code_hash, code_attempts, expires_ts, code_used_ts FROM magic_links WHERE poll_token = \\$1 FOR UPDATE"

	newMock := func(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
		t.Helper()
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("open sqlmock: %v", err)
		}
		t.Cleanup(func() {
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
			db.Close()
		})
		return db, mock
	}

	t.Run("right_code_starts_a_session", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin()
		// The poll token is stored hashed, so it is hashed before lookup.
		mock.ExpectQuery(lookup).
			WithArgs(HashToken(pollToken)).
			WillReturnRows(sqlmock.NewRows(linkColumns).
				AddRow(7, 1, HashToken(code), 2, time.Now().Add(time.Minute), nil))
		tokenHash := &capturedArg{}
		mock.ExpectQuery("WITH updated_poster AS").
			WithArgs(tokenHash, sqlmock.AnyArg(), 1).
			WillReturnRows(sqlmock.NewRows([]string{"expires_ts", "email"}).
				AddRow(time.Now().Add(time.Hour), "test@example.com"))
		// Only the code is marked used: the link keeps working, and the other
		// way round (see TestConfirmMagicLink).
		mock.ExpectExec("UPDATE magic_links SET code_used_ts = NOW\\(\\) WHERE id = \\$1").
			WithArgs(7).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		res, err := NewService(NewStore(db)).VerifyLoginCode(ctx, pollToken, code)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res.APIToken) != 64 || tokenHash.value != HashToken(res.APIToken) {
			t.Errorf("expected a raw 64-hex-char api token whose hash was stored, got %q", res.APIToken)
		}
		if res.Email != "test@example.com" {
			t.Errorf("expected email test@example.com, got %q", res.Email)
		}
	})

	t.Run("wrong_code_counts_an_attempt", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(lookup).
			WithArgs(HashToken(pollToken)).
			WillReturnRows(sqlmock.NewRows(linkColumns).
				AddRow(7, 1, HashToken(code), 0, time.Now().Add(time.Minute), nil))
		mock.ExpectExec("UPDATE magic_links SET code_attempts = code_attempts \\+ 1 WHERE id = \\$1").
			WithArgs(7).
			WillReturnResult(sqlmock.NewResult(0, 1))
		// The attempt must be committed, not rolled back with the refusal.
		mock.ExpectCommit()

		_, err := NewService(NewStore(db)).VerifyLoginCode(ctx, pollToken, "654321")
		if !errors.Is(err, ErrInvalidLoginCode) {
			t.Errorf("expected ErrInvalidLoginCode, got %v", err)
		}
	})

	// Every refusal below happens before any write: no attempt is counted
	// and no session is started.
	refusals := []struct {
		name string
		row  []driver.Value
	}{
		{"too_many_attempts", []driver.Value{7, 1, HashToken(code), maxLoginCodeAttempts, time.Now().Add(time.Minute), nil}},
		{"expired", []driver.Value{7, 1, HashToken(code), 0, time.Now().Add(-time.Minute), nil}},
		{"code_already_used", []driver.Value{7, 1, HashToken(code), 0, time.Now().Add(time.Minute), time.Now().Add(-time.Minute)}},
		{"link_from_before_login_codes", []driver.Value{7, 1, nil, 0, time.Now().Add(time.Minute), nil}},
	}
	for _, tc := range refusals {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := newMock(t)
			mock.ExpectBegin()
			mock.ExpectQuery(lookup).
				WithArgs(HashToken(pollToken)).
				WillReturnRows(sqlmock.NewRows(linkColumns).AddRow(tc.row...))
			mock.ExpectRollback()

			// Even the right code is refused.
			_, err := NewService(NewStore(db)).VerifyLoginCode(ctx, pollToken, code)
			if !errors.Is(err, ErrInvalidLoginCode) {
				t.Errorf("expected ErrInvalidLoginCode, got %v", err)
			}
		})
	}

	t.Run("unknown_poll_token", func(t *testing.T) {
		// Also what a decoy poll token (unknown account) or the emailed magic
		// token (hashes to a different value) gets.
		db, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(lookup).
			WithArgs(HashToken("some-other-token")).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectRollback()

		_, err := NewService(NewStore(db)).VerifyLoginCode(ctx, "some-other-token", code)
		if !errors.Is(err, ErrInvalidLoginCode) {
			t.Errorf("expected ErrInvalidLoginCode, got %v", err)
		}
	})

	t.Run("db_error_propagates", func(t *testing.T) {
		db, mock := newMock(t)
		boom := errors.New("connection lost")
		mock.ExpectBegin()
		mock.ExpectQuery(lookup).
			WithArgs(HashToken(pollToken)).
			WillReturnError(boom)
		mock.ExpectRollback()

		_, err := NewService(NewStore(db)).VerifyLoginCode(ctx, pollToken, code)
		if !errors.Is(err, boom) || errors.Is(err, ErrInvalidLoginCode) {
			t.Errorf("expected the db error, got %v", err)
		}
	})
}
