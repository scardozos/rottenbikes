package domain

import (
	"context"
	"database/sql"
)

type RatingAggregate struct {
	BikeNumericalID int64             `db:"bike_numerical_id" json:"bike_numerical_id"`
	Subcategory     RatingSubcategory `db:"subcategory"        json:"subcategory"`
	AverageRating   float32           `db:"average_rating"     json:"average_rating"`
}

func ListRatingAggregates(ctx context.Context, db *sql.DB) ([]RatingAggregate, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT bike_numerical_id, subcategory, average_rating
		FROM rating_aggregates
		ORDER BY bike_numerical_id, subcategory
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var aggs []RatingAggregate
	for rows.Next() {
		var a RatingAggregate
		if err := rows.Scan(&a.BikeNumericalID, &a.Subcategory, &a.AverageRating); err != nil {
			return nil, err
		}
		aggs = append(aggs, a)
	}
	return aggs, rows.Err()
}

func ListRatingAggregatesByBike(ctx context.Context, db *sql.DB, bikeID int64) ([]RatingAggregate, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT bike_numerical_id, subcategory, average_rating
		FROM rating_aggregates
		WHERE bike_numerical_id = $1
		ORDER BY subcategory
	`, bikeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var aggs []RatingAggregate
	for rows.Next() {
		var a RatingAggregate
		if err := rows.Scan(&a.BikeNumericalID, &a.Subcategory, &a.AverageRating); err != nil {
			return nil, err
		}
		aggs = append(aggs, a)
	}
	return aggs, rows.Err()
}

func RecomputeAggregatesForBike(ctx context.Context, tx *sql.Tx, bikeID int64) error {
	// Remove old aggregates for this bike
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM rating_aggregates
		WHERE bike_numerical_id = $1
	`, bikeID); err != nil {
		return err
	}

	// Recompute from review_ratings + reviews
	_, err := tx.ExecContext(ctx, `
		INSERT INTO rating_aggregates (
			bike_numerical_id, subcategory, rating_sum, rating_count, average_rating
		)
		SELECT
			r.bike_numerical_id,
			rr.subcategory,
			SUM(rr.score)                        AS rating_sum,
			COUNT(*)                             AS rating_count,
			ROUND(AVG(rr.score)::numeric, 2)     AS average_rating
		FROM review_ratings rr
		JOIN reviews r ON rr.review_id = r.review_id
		WHERE r.bike_numerical_id = $1
		GROUP BY r.bike_numerical_id, rr.subcategory
	`, bikeID)
	return err
}
