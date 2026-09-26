package domain

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestListBikes(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"numerical_id", "hash_id", "is_electric", "was_scanned", "created_ts", "updated_ts", "average_rating"}).
			AddRow("01", "hash1", true, true, time.Now(), time.Now(), 4.5).
			AddRow("02", "hash2", false, false, time.Now(), time.Now(), nil)

		mock.ExpectQuery(regexp.QuoteMeta(listBikesQuery)).
			WithArgs(10, 0, "", "recent").
			WillReturnRows(rows)

		store := NewService(NewStore(db))
		bikes, err := store.ListBikes(ctx, "", "recent", 10, 0)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(bikes) != 2 {
			t.Errorf("expected 2 bikes, got %d", len(bikes))
		}
	})
}

func TestGetBikeByHash(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	ctx := context.Background()
	hashID := "qrhash123"
	posterID := int64(1)

	t.Run("success_records_scan_event", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"numerical_id", "hash_id", "is_electric", "was_scanned", "created_ts", "updated_ts", "average_rating"}).
			AddRow("0123", hashID, true, false, time.Now(), time.Now(), 4.5)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(getBikeByHashQuery)).
			WithArgs(hashID).
			WillReturnRows(rows)
		mock.ExpectExec(regexp.QuoteMeta(upsertScanEventQuery)).
			WithArgs(posterID, "0123").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		store := NewService(NewStore(db))
		bike, err := store.GetBikeByHash(ctx, hashID, posterID)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if bike.NumericalID != "0123" {
			t.Errorf("expected numerical_id 0123, got %s", bike.NumericalID)
		}
		if bike.AverageRating == nil || *bike.AverageRating != 4.5 {
			t.Errorf("expected average rating 4.5, got %v", bike.AverageRating)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("not_found_no_scan_event", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(getBikeByHashQuery)).
			WithArgs(hashID).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectRollback()

		store := NewService(NewStore(db))
		_, err := store.GetBikeByHash(ctx, hashID, posterID)
		if err != sql.ErrNoRows {
			t.Errorf("expected sql.ErrNoRows, got %v", err)
		}

		// The scan event upsert must not have run for an unknown hash.
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}

func TestCreateBike(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	ctx := context.Background()
	numericalID := "0123"
	hashID := "hash123"
	isElectric := true
	wasScanned := true
	creatorID := int64(1)

	t.Run("success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"numerical_id", "hash_id", "is_electric", "was_scanned", "created_ts", "updated_ts"}).
			AddRow(numericalID, hashID, isElectric, wasScanned, time.Now(), time.Now())

		mock.ExpectQuery("INSERT INTO bikes").
			WithArgs(numericalID, &hashID, isElectric, wasScanned, creatorID).
			WillReturnRows(rows)

		store := NewService(NewStore(db))
		bike, err := store.CreateBike(ctx, numericalID, &hashID, isElectric, wasScanned, creatorID)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if bike.NumericalID != numericalID {
			t.Errorf("expected numericalID %s, got %s", numericalID, bike.NumericalID)
		}
		if bike.WasScanned != wasScanned {
			t.Errorf("expected was_scanned %v, got %v", wasScanned, bike.WasScanned)
		}
	})
}

func TestGetBike(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	ctx := context.Background()
	id := "01"

	t.Run("success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"numerical_id", "hash_id", "is_electric", "was_scanned", "created_ts", "updated_ts", "average_rating"}).
			AddRow(id, "hash1", true, false, time.Now(), time.Now(), 4.5)

		mock.ExpectQuery("SELECT b.numerical_id, b.hash_id, b.is_electric, b.was_scanned, b.created_ts, b.updated_ts, ra.average_rating FROM bikes b LEFT JOIN rating_aggregates ra ON b.numerical_id = ra.bike_numerical_id AND ra.subcategory = 'overall' WHERE b.numerical_id = \\$1").
			WithArgs(id).
			WillReturnRows(rows)

		store := NewService(NewStore(db))
		bike, err := store.GetBike(ctx, id)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if bike.NumericalID != id {
			t.Errorf("expected id %s, got %s", id, bike.NumericalID)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		mock.ExpectQuery("SELECT b.numerical_id, b.hash_id, b.is_electric, b.was_scanned, b.created_ts, b.updated_ts, ra.average_rating FROM bikes b LEFT JOIN rating_aggregates ra ON b.numerical_id = ra.bike_numerical_id AND ra.subcategory = 'overall' WHERE b.numerical_id = \\$1").
			WithArgs(id).
			WillReturnError(sql.ErrNoRows)

		store := NewService(NewStore(db))
		_, err := store.GetBike(ctx, id)
		if err != sql.ErrNoRows {
			t.Errorf("expected sql.ErrNoRows, got %v", err)
		}
	})
}

func TestUpdateBike(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	ctx := context.Background()
	id := "01"
	hashID := "newhash"
	isElectric := false
	creatorID := int64(1)

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec("UPDATE bikes").
			WithArgs(&hashID, &isElectric, id, creatorID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		store := NewService(NewStore(db))
		err := store.UpdateBike(ctx, id, &hashID, &isElectric, creatorID)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("not_owner_or_missing", func(t *testing.T) {
		// 0 rows affected means the WHERE clause (creator_id match) failed: the
		// caller is not the owner, or the bike does not exist. Both collapse to
		// sql.ErrNoRows so the handler can map to a single 404.
		mock.ExpectExec("UPDATE bikes").
			WithArgs(&hashID, &isElectric, id, creatorID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		store := NewService(NewStore(db))
		err := store.UpdateBike(ctx, id, &hashID, &isElectric, creatorID)
		if err != sql.ErrNoRows {
			t.Errorf("expected sql.ErrNoRows for non-owner, got %v", err)
		}
	})
}

func TestDeleteBike(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	ctx := context.Background()
	id := "01"
	creatorID := int64(1)

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM bikes").
			WithArgs(id, creatorID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		store := NewService(NewStore(db))
		err := store.DeleteBike(ctx, id, creatorID)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("not_owner_or_missing", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM bikes").
			WithArgs(id, creatorID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		store := NewService(NewStore(db))
		err := store.DeleteBike(ctx, id, creatorID)
		if err != sql.ErrNoRows {
			t.Errorf("expected sql.ErrNoRows for non-owner, got %v", err)
		}
	})
}
