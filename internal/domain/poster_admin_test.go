package domain

import (
	"context"
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
		rows := sqlmock.NewRows([]string{"poster_id", "username", "email", "created_ts", "review_count", "bike_count"}).
			AddRow(int64(2), "troll", "troll@example.com", time.Now(), int64(12), int64(3))

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
		if posters[0].ReviewCount != 12 || posters[0].BikeCount != 3 {
			t.Errorf("expected counts 12/3, got %d/%d", posters[0].ReviewCount, posters[0].BikeCount)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("empty_results", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"poster_id", "username", "email", "created_ts", "review_count", "bike_count"})

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

	t.Run("success_audited_in_tx", func(t *testing.T) {
		mock.ExpectBegin()

		// deleteContent = true path: list reviews for aggregate recompute
		rows := sqlmock.NewRows([]string{"bike_numerical_id"}).
			AddRow("0101")
		mock.ExpectQuery("SELECT DISTINCT bike_numerical_id FROM reviews").
			WithArgs(targetID).
			WillReturnRows(rows)

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

		mock.ExpectCommit()

		store := NewService(NewStore(db))
		if err := store.PurgePoster(ctx, adminID, targetID); err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}
