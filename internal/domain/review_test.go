package domain

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCreateReviewWithRatings(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	ctx := context.Background()
	posterID := int64(1)
	bikeID := int64(1)
	comment := "great bike"
	bikeImg := "img.jpg"
	score := int16(5)

	in := CreateReviewInput{
		PosterID: posterID,
		BikeID:   bikeID,
		Comment:  &comment,
		BikeImg:  &bikeImg,
		Overall:  &score,
	}

	t.Run("success", func(t *testing.T) {
		// New: Check global hourly limit
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM reviews").
			WithArgs(posterID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		// Check last review time
		mock.ExpectQuery("SELECT created_ts FROM reviews").
			WithArgs(posterID, bikeID).
			WillReturnError(sql.ErrNoRows)

		mock.ExpectBegin()

		// Insert review
		mock.ExpectQuery("INSERT INTO reviews").
			WithArgs(posterID, bikeID, bikeImg, comment).
			WillReturnRows(sqlmock.NewRows([]string{"review_id"}).AddRow(1))

		// Insert rating
		mock.ExpectExec("INSERT INTO review_ratings").
			WithArgs(1, RatingSubcategoryOverall, score).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Recompute aggregates
		mock.ExpectExec("DELETE FROM rating_aggregates").
			WithArgs(bikeID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("INSERT INTO rating_aggregates").
			WithArgs(bikeID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectCommit()

		store := NewStore(db)
		id, err := store.CreateReviewWithRatings(ctx, in)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if id != 1 {
			t.Errorf("expected review id 1, got %d", id)
		}
	})

	t.Run("hourly_limit_exceeded", func(t *testing.T) {
		// Expect count >= 5
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM reviews").
			WithArgs(posterID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		store := NewStore(db)
		_, err := store.CreateReviewWithRatings(ctx, in)
		if err != ErrHourlyRateLimitExceeded {
			t.Errorf("expected error %v, got %v", ErrHourlyRateLimitExceeded, err)
		}
	})

	t.Run("rate_limit_per_bike", func(t *testing.T) {
		// Hourly limit is fine
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM reviews").
			WithArgs(posterID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		// Last review was recent (per bike)
		mock.ExpectQuery("SELECT created_ts FROM reviews").
			WithArgs(posterID, bikeID).
			WillReturnRows(sqlmock.NewRows([]string{"created_ts"}).AddRow(time.Now()))

		store := NewStore(db)
		_, err := store.CreateReviewWithRatings(ctx, in)
		if err != ErrTooFrequentReview {
			t.Errorf("expected error %v, got %v", ErrTooFrequentReview, err)
		}
	})

	t.Run("invalid_rating", func(t *testing.T) {
		// Hourly limit is fine
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM reviews").
			WithArgs(posterID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		// Last review was long ago
		mock.ExpectQuery("SELECT created_ts FROM reviews").
			WithArgs(posterID, bikeID).
			WillReturnRows(sqlmock.NewRows([]string{"created_ts"}).AddRow(time.Now().Add(-24 * time.Hour)))

		mock.ExpectBegin()

		// Insert review
		mock.ExpectQuery("INSERT INTO reviews").
			WithArgs(posterID, bikeID, bikeImg, comment).
			WillReturnRows(sqlmock.NewRows([]string{"review_id"}).AddRow(1))

		// Invalid score
		invalidScore := int16(6)
		inInvalid := in
		inInvalid.Overall = &invalidScore

		store := NewStore(db)
		_, err := store.CreateReviewWithRatings(ctx, inInvalid)
		if err == nil {
			t.Error("expected error for invalid rating, got nil")
		}
		// Expect rollback implicitly on error
	})

}

func TestUpdateReviewWithRatings(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	ctx := context.Background()
	reviewID := int64(1)
	posterID := int64(1)
	bikeID := int64(1)
	comment := "updated comment"
	score := int16(4)

	in := UpdateReviewInput{
		ReviewID: reviewID,
		PosterID: posterID,
		Comment:  &comment,
		Overall:  &score,
	}

	t.Run("success", func(t *testing.T) {
		mock.ExpectBegin()

		// Check ownership
		mock.ExpectQuery("SELECT bike_numerical_id FROM reviews").
			WithArgs(reviewID, posterID).
			WillReturnRows(sqlmock.NewRows([]string{"bike_numerical_id"}).AddRow(bikeID))

		// Update review
		mock.ExpectExec("UPDATE reviews").
			WithArgs(comment, sqlmock.AnyArg(), reviewID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// Update rating
		mock.ExpectExec("INSERT INTO review_ratings").
			WithArgs(reviewID, RatingSubcategoryOverall, score).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// Recompute aggregates
		mock.ExpectExec("DELETE FROM rating_aggregates").
			WithArgs(bikeID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("INSERT INTO rating_aggregates").
			WithArgs(bikeID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectCommit()

		store := NewStore(db)
		err := store.UpdateReviewWithRatings(ctx, in)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT bike_numerical_id FROM reviews").
			WithArgs(reviewID, posterID).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectRollback()

		store := NewStore(db)
		err := store.UpdateReviewWithRatings(ctx, in)
		if err != sql.ErrNoRows {
			t.Errorf("expected error %v, got %v", sql.ErrNoRows, err)
		}
	})

}

func TestDeleteReview(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	ctx := context.Background()
	reviewID := int64(1)
	posterID := int64(1)
	bikeID := int64(1)

	t.Run("success", func(t *testing.T) {
		mock.ExpectBegin()

		// Check ownership
		mock.ExpectQuery("SELECT bike_numerical_id FROM reviews").
			WithArgs(reviewID, posterID).
			WillReturnRows(sqlmock.NewRows([]string{"bike_numerical_id"}).AddRow(bikeID))

		// Delete ratings
		mock.ExpectExec("DELETE FROM review_ratings").
			WithArgs(reviewID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// Delete review
		mock.ExpectExec("DELETE FROM reviews").
			WithArgs(reviewID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// Recompute aggregates
		mock.ExpectExec("DELETE FROM rating_aggregates").
			WithArgs(bikeID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("INSERT INTO rating_aggregates").
			WithArgs(bikeID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectCommit()

		store := NewStore(db)
		err := store.DeleteReview(ctx, reviewID, posterID)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT bike_numerical_id FROM reviews").
			WithArgs(reviewID, posterID).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectRollback()

		store := NewStore(db)
		err := store.DeleteReview(ctx, reviewID, posterID)
		if err != sql.ErrNoRows {
			t.Errorf("expected error %v, got %v", sql.ErrNoRows, err)
		}
	})

}

func TestGetReviewWithRatingsByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	ctx := context.Background()
	reviewID := int64(1)

	t.Run("success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"review_id", "poster_id", "username", "bike_numerical_id", "comment", "created_ts", "subcategory", "score", "bike_img",
		}).
			AddRow(reviewID, 1, "user1", 1, "comment", time.Now(), "overall", 5, "img.jpg")

		mock.ExpectQuery("SELECT .* FROM reviews r JOIN posters p .* JOIN review_ratings rr .*").
			WithArgs(reviewID).
			WillReturnRows(rows)

		store := NewStore(db)
		review, err := store.GetReviewWithRatingsByID(ctx, reviewID)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if review.ReviewID != reviewID {
			t.Errorf("expected review id %d, got %d", reviewID, review.ReviewID)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		mock.ExpectQuery("SELECT .* FROM reviews r JOIN posters p .* JOIN review_ratings rr .*").
			WithArgs(reviewID).
			WillReturnError(sql.ErrNoRows)

		store := NewStore(db)
		_, err := store.GetReviewWithRatingsByID(ctx, reviewID)
		if err != sql.ErrNoRows {
			t.Errorf("expected error %v, got %v", sql.ErrNoRows, err)
		}
	})
}
