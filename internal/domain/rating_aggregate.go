package domain

import (
	"context"
	"database/sql"
)

type RatingAggregate struct {
	BikeNumericalID int64             `db:"bike_numerical_id" json:"bike_numerical_id"`
	Subcategory     RatingSubcategory `db:"subcategory"        json:"subcategory"`
	AverageRating   float32           `db:"average_rating"     json:"average_rating"`
	Window          string            `json:"window,omitempty"` // "1w", "2w", "overall"
}

func (s *Store) ListRatingAggregatesByBike(ctx context.Context, bikeID int64) ([]RatingAggregate, error) {
	rows, err := s.db.QueryContext(ctx, `
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
		a.Window = "overall" // Default to overall for backward compatibility/precomputed
		aggs = append(aggs, a)
	}
	return aggs, rows.Err()
}

func (s *Store) ListWindowedRatingAggregatesByBike(ctx context.Context, bikeID int64) ([]RatingAggregate, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			rr.subcategory,
			ROUND(AVG(CASE WHEN r.created_ts >= NOW() - INTERVAL '1 week' THEN rr.score END)::numeric, 2) as avg_1w,
			ROUND(AVG(CASE WHEN r.created_ts >= NOW() - INTERVAL '2 weeks' THEN rr.score END)::numeric, 2) as avg_2w,
			ROUND(AVG(rr.score)::numeric, 2) as avg_overall
		FROM review_ratings rr
		JOIN reviews r ON rr.review_id = r.review_id
		WHERE r.bike_numerical_id = $1
		GROUP BY rr.subcategory
		ORDER BY rr.subcategory
	`, bikeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var aggs []RatingAggregate
	for rows.Next() {
		var sub RatingSubcategory
		var avg1w, avg2w, avgOverall sql.NullFloat64

		if err := rows.Scan(&sub, &avg1w, &avg2w, &avgOverall); err != nil {
			return nil, err
		}

		// Helper to append if valid
		appendAgg := func(window string, val sql.NullFloat64) {
			if val.Valid {
				aggs = append(aggs, RatingAggregate{
					BikeNumericalID: bikeID,
					Subcategory:     sub,
					AverageRating:   float32(val.Float64),
					Window:          window,
				})
			}
		}

		appendAgg("1w", avg1w)
		appendAgg("2w", avg2w)
		appendAgg("overall", avgOverall)
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
