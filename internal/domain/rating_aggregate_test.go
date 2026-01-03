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
	bikeID := "0101"

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
		if aggs[0].Window != "overall" {
			t.Errorf("expected window 'overall', got %s", aggs[0].Window)
		}
	})
}

func TestListWindowedRatingAggregatesByBike(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	ctx := context.Background()
	bikeID := "0101"

	t.Run("success", func(t *testing.T) {
		// Mock rows:
		// subcategory, avg_1w, avg_2w, avg_overall
		rows := sqlmock.NewRows([]string{"subcategory", "avg_1w", "avg_2w", "avg_overall"}).
			AddRow("quality", 4.0, 4.2, 4.5). // All valid
			AddRow("price", nil, 3.0, 3.5)    // 1w nil

		mock.ExpectQuery("SELECT rr.subcategory").
			WithArgs(bikeID).
			WillReturnRows(rows)

		store := NewStore(db)
		aggs, err := store.ListWindowedRatingAggregatesByBike(ctx, bikeID)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Expected:
		// quality: 1w(4.0), 2w(4.2), overall(4.5) -> 3 items
		// price: 2w(3.0), overall(3.5) -> 2 items (1w skipped)
		// Total 5 items
		if len(aggs) != 5 {
			t.Errorf("expected 5 aggregates, got %d", len(aggs))
		}

		// Basic check of one item
		found := false
		for _, a := range aggs {
			if a.Subcategory == "quality" && a.Window == "1w" {
				if a.AverageRating != 4.0 {
					t.Errorf("expected rating 4.0, got %f", a.AverageRating)
				}
				found = true
			}
		}
		if !found {
			t.Errorf("quality 1w aggregate not found")
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
	bikeID := "0101"

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
