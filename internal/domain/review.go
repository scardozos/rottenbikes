package domain

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type ReviewWithRatings struct {
	ReviewID        int64                       `json:"review_id"`
	PosterID        int64                       `json:"poster_id"`
	PosterUsername  string                      `json:"poster_username"`
	BikeNumericalID int64                       `json:"bike_numerical_id"`
	Comment         *string                     `json:"comment"`
	CreatedAt       time.Time                   `json:"created_at"`
	Ratings         map[RatingSubcategory]int16 `json:"ratings"`
	BikeImg         *string                     `json:"bike_img"`
}

type reviewRatingRow struct {
	ReviewID        int64
	PosterID        int64
	PosterUsername  string
	BikeNumericalID int64
	Comment         *string
	CreatedAt       time.Time
	Subcategory     RatingSubcategory
	Score           int16
	BikeImg         *string
}

// all bikes
func ListReviewsWithRatings(ctx context.Context, db *sql.DB) ([]ReviewWithRatings, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT
			r.review_id,
			r.poster_id,
			p.username,
			r.bike_numerical_id,
			r.comment,
			r.created_ts,
			rr.subcategory,
			rr.score,
			r.bike_img
		FROM reviews r
		JOIN posters p       ON p.poster_id = r.poster_id
		JOIN review_ratings rr ON rr.review_id = r.review_id
		ORDER BY r.bike_numerical_id, r.review_id, rr.subcategory
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return buildReviewWithRatingsFromRows(rows)
}

// single bike
func ListReviewsWithRatingsByBike(ctx context.Context, db *sql.DB, bikeID int64) ([]ReviewWithRatings, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT
			r.review_id,
			r.poster_id,
			p.username,
			r.bike_numerical_id,
			r.comment,
			r.created_ts,
			rr.subcategory,
			rr.score,
			r.bike_img
		FROM reviews r
		JOIN posters p       ON p.poster_id = r.poster_id
		JOIN review_ratings rr ON rr.review_id = r.review_id
		WHERE r.bike_numerical_id = $1
		ORDER BY r.review_id, rr.subcategory
	`, bikeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return buildReviewWithRatingsFromRows(rows)
}

func buildReviewWithRatingsFromRows(rows *sql.Rows) ([]ReviewWithRatings, error) {
	reviewsMap := make(map[int64]*ReviewWithRatings)

	for rows.Next() {
		var row reviewRatingRow
		if err := rows.Scan(
			&row.ReviewID,
			&row.PosterID,
			&row.PosterUsername,
			&row.BikeNumericalID,
			&row.Comment,
			&row.CreatedAt,
			&row.Subcategory,
			&row.Score,
			&row.BikeImg,
		); err != nil {
			return nil, err
		}

		r, ok := reviewsMap[row.ReviewID]
		if !ok {
			r = &ReviewWithRatings{
				ReviewID:        row.ReviewID,
				PosterID:        row.PosterID,
				PosterUsername:  row.PosterUsername,
				BikeNumericalID: row.BikeNumericalID,
				Comment:         row.Comment,
				CreatedAt:       row.CreatedAt,
				Ratings:         make(map[RatingSubcategory]int16),
				BikeImg:         row.BikeImg,
			}
			reviewsMap[row.ReviewID] = r
		}
		r.Ratings[row.Subcategory] = row.Score
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make([]ReviewWithRatings, 0, len(reviewsMap))
	for _, r := range reviewsMap {
		result = append(result, *r)
	}
	return result, nil
}

var ErrTooFrequentReview = errors.New("review too frequent")

type CreateReviewInput struct {
	PosterID int64
	BikeID   int64
	Comment  *string
	BikeImg  *string

	Overall    *int16
	Breaks     *int16
	Seat       *int16
	Sturdiness *int16
	Power      *int16
	Pedals     *int16
}

func CreateReviewWithRatings(ctx context.Context, db *sql.DB, in CreateReviewInput) (int64, error) {
	const minInterval = 10 * time.Minute

	var lastCreated time.Time
	err := db.QueryRowContext(ctx, `
		SELECT created_ts
		FROM reviews
		WHERE poster_id = $1 AND bike_numerical_id = $2
		ORDER BY created_ts DESC
		LIMIT 1
	`, in.PosterID, in.BikeID).Scan(&lastCreated)

	if err == nil {
		if time.Since(lastCreated) < minInterval {
			return 0, ErrTooFrequentReview
		}
	} else if err != sql.ErrNoRows {
		return 0, fmt.Errorf("check last review time: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Insert review, now including bike_img
	var reviewID int64
	if err := tx.QueryRowContext(ctx, `
		INSERT INTO reviews (poster_id, bike_numerical_id, bike_img, comment)
		VALUES ($1, $2, $3, $4)
		RETURNING review_id
	`, in.PosterID, in.BikeID, in.BikeImg, in.Comment).Scan(&reviewID); err != nil {
		return 0, fmt.Errorf("insert review: %w", err)
	}

	insertRating := func(sub RatingSubcategory, val *int16) error {
		if val == nil {
			return nil
		}
		if *val < 1 || *val > 5 {
			return fmt.Errorf("invalid score %d for %s", *val, sub)
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO review_ratings (review_id, subcategory, score)
			VALUES ($1, $2, $3)
		`, reviewID, sub, *val)
		return err
	}

	if err := insertRating(RatingSubcategoryOverall, in.Overall); err != nil {
		return 0, fmt.Errorf("insert overall rating: %w", err)
	}
	if err := insertRating(RatingSubcategoryBreaks, in.Breaks); err != nil {
		return 0, fmt.Errorf("insert breaks rating: %w", err)
	}
	if err := insertRating(RatingSubcategorySeat, in.Seat); err != nil {
		return 0, fmt.Errorf("insert seat rating: %w", err)
	}
	if err := insertRating(RatingSubcategorySturdiness, in.Sturdiness); err != nil {
		return 0, fmt.Errorf("insert sturdiness rating: %w", err)
	}
	if err := insertRating(RatingSubcategoryPower, in.Power); err != nil {
		return 0, fmt.Errorf("insert power rating: %w", err)
	}
	if err := insertRating(RatingSubcategoryPedals, in.Pedals); err != nil {
		return 0, fmt.Errorf("insert pedals rating: %w", err)
	}

	if err := RecomputeAggregatesForBike(ctx, tx, in.BikeID); err != nil {
		return 0, fmt.Errorf("recompute aggregates: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit tx: %w", err)
	}

	return reviewID, nil
}

type UpdateReviewInput struct {
	ReviewID int64
	PosterID int64 // for ownership check

	Comment *string
	BikeImg *string

	Overall    *int16
	Breaks     *int16
	Seat       *int16
	Sturdiness *int16
	Power      *int16
	Pedals     *int16
}

func UpdateReviewWithRatings(ctx context.Context, db *sql.DB, in UpdateReviewInput) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// ensure review belongs to poster
	var bikeID int64
	if err := tx.QueryRowContext(ctx, `
		SELECT bike_numerical_id
		FROM reviews
		WHERE review_id = $1 AND poster_id = $2
	`, in.ReviewID, in.PosterID).Scan(&bikeID); err != nil {
		if err == sql.ErrNoRows {
			return sql.ErrNoRows
		}
		return fmt.Errorf("load review: %w", err)
	}

	// update main review row
	if _, err := tx.ExecContext(ctx, `
		UPDATE reviews
		SET comment = COALESCE($1, comment),
		    bike_img = COALESCE($2, bike_img)
		WHERE review_id = $3
	`, in.Comment, in.BikeImg, in.ReviewID); err != nil {
		return fmt.Errorf("update review: %w", err)
	}

	updateRating := func(sub RatingSubcategory, val *int16) error {
		if val == nil {
			return nil
		}
		if *val < 1 || *val > 5 {
			return fmt.Errorf("invalid score %d for %s", *val, sub)
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO review_ratings (review_id, subcategory, score)
			VALUES ($1, $2, $3)
			ON CONFLICT (review_id, subcategory)
			DO UPDATE SET score = EXCLUDED.score
		`, in.ReviewID, sub, *val)
		return err
	}

	if err := updateRating(RatingSubcategoryOverall, in.Overall); err != nil {
		return fmt.Errorf("update overall rating: %w", err)
	}
	if err := updateRating(RatingSubcategoryBreaks, in.Breaks); err != nil {
		return fmt.Errorf("update breaks rating: %w", err)
	}
	if err := updateRating(RatingSubcategorySeat, in.Seat); err != nil {
		return fmt.Errorf("update seat rating: %w", err)
	}
	if err := updateRating(RatingSubcategorySturdiness, in.Sturdiness); err != nil {
		return fmt.Errorf("update sturdiness rating: %w", err)
	}
	if err := updateRating(RatingSubcategoryPower, in.Power); err != nil {
		return fmt.Errorf("update power rating: %w", err)
	}
	if err := updateRating(RatingSubcategoryPedals, in.Pedals); err != nil {
		return fmt.Errorf("update pedals rating: %w", err)
	}

	if err := RecomputeAggregatesForBike(ctx, tx, bikeID); err != nil {
		return fmt.Errorf("recompute aggregates: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func GetReviewWithRatingsByID(ctx context.Context, db *sql.DB, reviewID int64) (*ReviewWithRatings, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT
			r.review_id,
			r.poster_id,
			p.username,
			r.bike_numerical_id,
			r.comment,
			r.created_ts,
			rr.subcategory,
			rr.score,
			r.bike_img
		FROM reviews r
		JOIN posters p       ON p.poster_id = r.poster_id
		LEFT JOIN review_ratings rr ON rr.review_id = r.review_id
		WHERE r.review_id = $1
		ORDER BY rr.subcategory
	`, reviewID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reviews, err := buildReviewWithRatingsFromRows(rows)
	if err != nil {
		return nil, err
	}
	if len(reviews) == 0 {
		return nil, sql.ErrNoRows
	}
	return &reviews[0], nil
}

func DeleteReview(ctx context.Context, db *sql.DB, reviewID int64, posterID int64) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// ensure review exists and belongs to poster, and get bike id for recompute
	var bikeID int64
	if err := tx.QueryRowContext(ctx, `
		SELECT bike_numerical_id
		FROM reviews
		WHERE review_id = $1 AND poster_id = $2
	`, reviewID, posterID).Scan(&bikeID); err != nil {
		if err == sql.ErrNoRows {
			return sql.ErrNoRows
		}
		return fmt.Errorf("load review: %w", err)
	}

	// delete ratings first due to FK
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM review_ratings
		WHERE review_id = $1
	`, reviewID); err != nil {
		return fmt.Errorf("delete review_ratings: %w", err)
	}

	// delete review
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM reviews
		WHERE review_id = $1
	`, reviewID); err != nil {
		return fmt.Errorf("delete review: %w", err)
	}

	if err := RecomputeAggregatesForBike(ctx, tx, bikeID); err != nil {
		return fmt.Errorf("recompute aggregates: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}
