package domain

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"time"
)

var (
	//go:embed sql/list_reviews_with_ratings.sql
	listReviewsWithRatingsQuery string
	//go:embed sql/create_review_check_rate_limit.sql
	createReviewCheckRateLimitQuery string
	//go:embed sql/create_review_check_frequency.sql
	createReviewCheckFrequencyQuery string
	//go:embed sql/insert_review.sql
	insertReviewQuery string
	//go:embed sql/insert_review_rating.sql
	insertReviewRatingQuery string
	//go:embed sql/update_review_check_ownership.sql
	updateReviewCheckOwnershipQuery string
	//go:embed sql/update_review.sql
	updateReviewQuery string
	//go:embed sql/upsert_review_rating.sql
	upsertReviewRatingQuery string
	//go:embed sql/get_review_by_id.sql
	getReviewByIDQuery string
	//go:embed sql/delete_review_check_ownership.sql
	deleteReviewCheckOwnershipQuery string
	//go:embed sql/delete_review_ratings.sql
	deleteReviewRatingsQuery string
	//go:embed sql/delete_review.sql
	deleteReviewQuery string
)

type ReviewWithRatings struct {
	ReviewID        int64                       `json:"review_id"`
	PosterID        int64                       `json:"poster_id"`
	PosterUsername  string                      `json:"poster_username"`
	BikeNumericalID string                      `json:"bike_numerical_id"`
	Comment         *string                     `json:"comment"`
	CreatedAt       time.Time                   `json:"created_at"`
	Ratings         map[RatingSubcategory]int16 `json:"ratings"`
	BikeImg         *string                     `json:"bike_img"`
}

type reviewRatingRow struct {
	ReviewID        int64
	PosterID        sql.NullInt64
	PosterUsername  string
	BikeNumericalID string
	Comment         *string
	CreatedAt       time.Time
	Subcategory     RatingSubcategory
	Score           int16
	BikeImg         *string
}

// all bikes

// single bike
func (s *Store) ListReviewsWithRatingsByBike(ctx context.Context, bikeID string) ([]ReviewWithRatings, error) {
	rows, err := s.db.QueryContext(ctx, listReviewsWithRatingsQuery, bikeID)
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
				PosterID:        row.PosterID.Int64, // Corrected based on instruction interpretation
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
var ErrHourlyRateLimitExceeded = errors.New("hourly review limit exceeded")

type CreateReviewInput struct {
	PosterID int64
	BikeID   string
	Comment  *string
	BikeImg  *string

	Overall    *int16
	Breaks     *int16
	Seat       *int16
	Sturdiness *int16
	Power      *int16
	Pedals     *int16
}

func (s *Store) CreateReviewWithRatings(ctx context.Context, in CreateReviewInput) (int64, error) {
	const minInterval = 10 * time.Minute
	const maxHourlyReviews = 5

	// 1. Check global hourly limit
	var hourlyCount int
	if err := s.db.QueryRowContext(ctx, createReviewCheckRateLimitQuery, in.PosterID).Scan(&hourlyCount); err != nil {
		return 0, fmt.Errorf("check hourly limit: %w", err)
	}
	if hourlyCount >= maxHourlyReviews {
		return 0, ErrHourlyRateLimitExceeded
	}

	// 2. Check per-bike frequency

	var lastCreated time.Time
	err := s.db.QueryRowContext(ctx, createReviewCheckFrequencyQuery, in.PosterID, in.BikeID).Scan(&lastCreated)

	if err == nil {
		if time.Since(lastCreated) < minInterval {
			return 0, ErrTooFrequentReview
		}
	} else if err != sql.ErrNoRows {
		return 0, fmt.Errorf("check last review time: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Insert review, now including bike_img
	var reviewID int64
	if err := tx.QueryRowContext(ctx, insertReviewQuery, in.PosterID, in.BikeID, in.BikeImg, in.Comment).Scan(&reviewID); err != nil {
		return 0, fmt.Errorf("insert review: %w", err)
	}

	insertRating := func(sub RatingSubcategory, val *int16) error {
		if val == nil {
			return nil
		}
		if *val < 1 || *val > 5 {
			return fmt.Errorf("invalid score %d for %s", *val, sub)
		}
		_, err := tx.ExecContext(ctx, insertReviewRatingQuery, reviewID, sub, *val)
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

func (s *Store) UpdateReviewWithRatings(ctx context.Context, in UpdateReviewInput) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// ensure review belongs to poster
	var bikeID string
	if err := tx.QueryRowContext(ctx, updateReviewCheckOwnershipQuery, in.ReviewID, in.PosterID).Scan(&bikeID); err != nil {
		if err == sql.ErrNoRows {
			return sql.ErrNoRows
		}
		return fmt.Errorf("load review: %w", err)
	}

	// update main review row
	if _, err := tx.ExecContext(ctx, updateReviewQuery, in.Comment, in.BikeImg, in.ReviewID); err != nil {
		return fmt.Errorf("update review: %w", err)
	}

	updateRating := func(sub RatingSubcategory, val *int16) error {
		if val == nil {
			return nil
		}
		if *val < 1 || *val > 5 {
			return fmt.Errorf("invalid score %d for %s", *val, sub)
		}
		_, err := tx.ExecContext(ctx, upsertReviewRatingQuery, in.ReviewID, sub, *val)
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

func (s *Store) GetReviewWithRatingsByID(ctx context.Context, reviewID int64) (*ReviewWithRatings, error) {
	rows, err := s.db.QueryContext(ctx, getReviewByIDQuery, reviewID)
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

func (s *Store) DeleteReview(ctx context.Context, reviewID int64, posterID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// ensure review exists and belongs to poster, and get bike id for recompute
	var bikeID string
	if err := tx.QueryRowContext(ctx, deleteReviewCheckOwnershipQuery, reviewID, posterID).Scan(&bikeID); err != nil {
		if err == sql.ErrNoRows {
			return sql.ErrNoRows
		}
		return fmt.Errorf("load review: %w", err)
	}

	// delete ratings first due to FK
	if _, err := tx.ExecContext(ctx, deleteReviewRatingsQuery, reviewID); err != nil {
		return fmt.Errorf("delete review_ratings: %w", err)
	}

	// delete review
	if _, err := tx.ExecContext(ctx, deleteReviewQuery, reviewID); err != nil {
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
