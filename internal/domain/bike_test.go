package domain

import (
	"context"
	"database/sql"
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
		rows := sqlmock.NewRows([]string{"numerical_id", "hash_id", "is_electric"}).
			AddRow(1, "hash1", true).
			AddRow(2, "hash2", false)

		mock.ExpectQuery("SELECT numerical_id, hash_id, is_electric FROM bikes").
			WillReturnRows(rows)

		store := NewStore(db)
		bikes, err := store.ListBikes(ctx)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(bikes) != 2 {
			t.Errorf("expected 2 bikes, got %d", len(bikes))
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
	numericalID := int64(123)
	hashID := "hash_123"
	isElectric := true
	creatorID := int64(1)

	t.Run("success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"numerical_id", "hash_id", "is_electric", "created_ts", "updated_ts"}).
			AddRow(numericalID, hashID, isElectric, time.Now(), time.Now())

		mock.ExpectQuery("INSERT INTO bikes").
			WithArgs(numericalID, hashID, isElectric, creatorID).
			WillReturnRows(rows)

		store := NewStore(db)
		bike, err := store.CreateBike(ctx, numericalID, &hashID, isElectric, creatorID)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if bike.NumericalID != numericalID {
			t.Errorf("expected numericalID %d, got %d", numericalID, bike.NumericalID)
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
	id := int64(1)

	t.Run("success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"numerical_id", "hash_id", "is_electric", "created_ts", "updated_ts"}).
			AddRow(id, "hash1", true, time.Now(), time.Now())

		mock.ExpectQuery("SELECT numerical_id, hash_id, is_electric, created_ts, updated_ts FROM bikes").
			WithArgs(id).
			WillReturnRows(rows)

		store := NewStore(db)
		bike, err := store.GetBike(ctx, id)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if bike.NumericalID != id {
			t.Errorf("expected id %d, got %d", id, bike.NumericalID)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		mock.ExpectQuery("SELECT numerical_id, hash_id, is_electric, created_ts, updated_ts FROM bikes").
			WithArgs(id).
			WillReturnError(sql.ErrNoRows)

		store := NewStore(db)
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
	id := int64(1)
	hashID := "new_hash"
	isElectric := false

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec("UPDATE bikes").
			WithArgs(hashID, isElectric, id).
			WillReturnResult(sqlmock.NewResult(0, 1))

		store := NewStore(db)
		err := store.UpdateBike(ctx, id, &hashID, &isElectric)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
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
	id := int64(1)

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM bikes").
			WithArgs(id).
			WillReturnResult(sqlmock.NewResult(0, 1))

		store := NewStore(db)
		err := store.DeleteBike(ctx, id)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}
