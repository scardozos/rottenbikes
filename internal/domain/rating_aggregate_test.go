package domain

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestListRatingAggregatesByBike(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	ctx := context.Background()
	bikeID := int64(1)

	t.Run("success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"bike_numerical_id", "subcategory", "average_rating"}).
			AddRow(bikeID, "overall", 4.5)

		mock.ExpectQuery("SELECT bike_numerical_id, subcategory, average_rating FROM rating_aggregates").
			WithArgs(bikeID).
			WillReturnRows(rows)

		store := NewStore(db)
		aggs, err := store.ListRatingAggregatesByBike(ctx, bikeID)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(aggs) != 1 {
			t.Errorf("expected 1 aggregate, got %d", len(aggs))
		}
	})
}

func TestRecomputeAggregatesForBike(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	ctx := context.Background()
	bikeID := int64(1)

	t.Run("success", func(t *testing.T) {
		mock.ExpectBegin()
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("failed to begin tx: %v", err)
		}

		mock.ExpectExec("DELETE FROM rating_aggregates").
			WithArgs(bikeID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectExec("INSERT INTO rating_aggregates").
			WithArgs(bikeID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err = RecomputeAggregatesForBike(ctx, tx, bikeID)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}
