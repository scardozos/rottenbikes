package domain

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSearchPosters(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"poster_id", "username", "email", "role", "created_ts", "review_count", "bike_count"}).
			AddRow(int64(2), "troll", "troll@example.com", "user", time.Now(), int64(12), int64(3))

		mock.ExpectQuery(regexp.QuoteMeta(searchPostersQuery)).
			WithArgs("troll", 20).
			WillReturnRows(rows)

		store := NewService(NewStore(db))
		posters, err := store.SearchPosters(ctx, "troll", 20)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(posters) != 1 {
			t.Fatalf("expected 1 poster, got %d", len(posters))
		}
		if posters[0].Username != "troll" {
			t.Errorf("expected username troll, got %s", posters[0].Username)
		}
		if posters[0].Role != PosterRoleUser {
			t.Errorf("expected role user, got %s", posters[0].Role)
		}
		if posters[0].ReviewCount != 12 || posters[0].BikeCount != 3 {
			t.Errorf("expected counts 12/3, got %d/%d", posters[0].ReviewCount, posters[0].BikeCount)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("empty_results", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"poster_id", "username", "email", "role", "created_ts", "review_count", "bike_count"})

		mock.ExpectQuery(regexp.QuoteMeta(searchPostersQuery)).
			WithArgs("nobody", 20).
			WillReturnRows(rows)

		store := NewService(NewStore(db))
		posters, err := store.SearchPosters(ctx, "nobody", 20)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(posters) != 0 {
			t.Errorf("expected 0 posters, got %d", len(posters))
		}
	})
}

func TestPurgePoster(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	ctx := context.Background()
	adminID := int64(1)
	targetID := int64(2)

	expectPurgeSequence := func(targetRole string) {
		// Guard: lock + load the target's role inside the tx
		mock.ExpectQuery(regexp.QuoteMeta(getPosterRoleForPurgeQuery)).
			WithArgs(targetID).
			WillReturnRows(sqlmock.NewRows([]string{"role"}).AddRow(targetRole))

		// deleteContent = true path: list reviews for aggregate recompute
		mock.ExpectQuery("SELECT DISTINCT bike_numerical_id FROM reviews").
			WithArgs(targetID).
			WillReturnRows(sqlmock.NewRows([]string{"bike_numerical_id"}).AddRow("0101"))

		mock.ExpectExec("DELETE FROM review_ratings").
			WithArgs(targetID).
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectExec("DELETE FROM reviews").
			WithArgs(targetID).
			WillReturnResult(sqlmock.NewResult(0, 2))

		// Recompute aggregates for the affected bike
		mock.ExpectExec("DELETE FROM rating_aggregates").
			WithArgs("0101").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("INSERT INTO rating_aggregates").
			WithArgs("0101").
			WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectExec("DELETE FROM bikes").
			WithArgs(targetID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectExec("DELETE FROM magic_links").
			WithArgs(targetID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectExec("DELETE FROM posters").
			WithArgs(targetID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// Audit row lands inside the same transaction
		mock.ExpectExec(regexp.QuoteMeta(insertModerationActionQuery)).
			WithArgs(adminID, "purge_poster", targetID).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}

	t.Run("success_audited_in_tx", func(t *testing.T) {
		mock.ExpectBegin()
		expectPurgeSequence("user")
		mock.ExpectCommit()

		store := NewService(NewStore(db))
		if err := store.PurgePoster(ctx, adminID, targetID); err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("guard_refuses_admin_target", func(t *testing.T) {
		// One compromised admin must not be able to purge another admin.
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(getPosterRoleForPurgeQuery)).
			WithArgs(targetID).
			WillReturnRows(sqlmock.NewRows([]string{"role"}).AddRow("admin"))
		mock.ExpectRollback()

		store := NewService(NewStore(db))
		err := store.PurgePoster(ctx, adminID, targetID)
		if err != ErrCannotPurgeAdmin {
			t.Errorf("expected ErrCannotPurgeAdmin, got %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("guard_missing_target", func(t *testing.T) {
		// A purge of a nonexistent poster surfaces as sql.ErrNoRows (→ 404)
		// instead of a silent no-op.
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(getPosterRoleForPurgeQuery)).
			WithArgs(targetID).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectRollback()

		store := NewService(NewStore(db))
		err := store.PurgePoster(ctx, adminID, targetID)
		if err != sql.ErrNoRows {
			t.Errorf("expected sql.ErrNoRows, got %v", err)
		}
	})
}

func TestSetPosterRole(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	ctx := context.Background()

	t.Run("promote_by_username", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(setPosterRoleQuery)).
			WithArgs("admin", "alice").
			WillReturnRows(sqlmock.NewRows([]string{"poster_id", "email", "username", "role"}).
				AddRow(1, "alice@example.com", "alice", "admin"))

		store := NewStore(db)
		p, err := store.SetPosterRole(ctx, "alice", PosterRoleAdmin)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if p.Role != PosterRoleAdmin {
			t.Errorf("expected role admin, got %s", p.Role)
		}
		if p.Username != "alice" {
			t.Errorf("expected username alice, got %s", p.Username)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("unknown_identifier", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(setPosterRoleQuery)).
			WithArgs("admin", "nobody").
			WillReturnError(sql.ErrNoRows)

		store := NewStore(db)
		_, err := store.SetPosterRole(ctx, "nobody", PosterRoleAdmin)
		if err != sql.ErrNoRows {
			t.Errorf("expected sql.ErrNoRows, got %v", err)
		}
	})
}

func TestListAdmins(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"poster_id", "email", "username", "role", "created_ts"}).
			AddRow(int64(1), "alice@example.com", "alice", "admin", time.Now())

		mock.ExpectQuery(regexp.QuoteMeta(listAdminPostersQuery)).
			WillReturnRows(rows)

		store := NewStore(db)
		admins, err := store.ListAdmins(ctx)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(admins) != 1 {
			t.Fatalf("expected 1 admin, got %d", len(admins))
		}
		if admins[0].Username != "alice" || admins[0].Role != PosterRoleAdmin {
			t.Errorf("expected admin alice, got %+v", admins[0])
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("none", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"poster_id", "email", "username", "role", "created_ts"})

		mock.ExpectQuery(regexp.QuoteMeta(listAdminPostersQuery)).
			WillReturnRows(rows)

		store := NewStore(db)
		admins, err := store.ListAdmins(ctx)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(admins) != 0 {
			t.Errorf("expected 0 admins, got %d", len(admins))
		}
	})
}
