package domain

import (
	"context"
	"database/sql"
	_ "embed"
)

var (
	//go:embed sql/list_rating_aggregates.sql
	listRatingAggregatesQuery string
	//go:embed sql/list_windowed_rating_aggregates.sql
	listWindowedRatingAggregatesQuery string
	//go:embed sql/delete_rating_aggregates.sql
	deleteRatingAggregatesQuery string
	//go:embed sql/recompute_rating_aggregates.sql
	recomputeRatingAggregatesQuery string
)

type RatingAggregate struct {
	BikeNumericalID string            `db:"bike_numerical_id" json:"bike_numerical_id"`
	Subcategory     RatingSubcategory `db:"subcategory"        json:"subcategory"`
	AverageRating   float32           `db:"average_rating"     json:"average_rating"`
	Window          string            `json:"window,omitempty"` // "1w", "2w", "overall"
}

func (s *Store) ListRatingAggregatesByBike(ctx context.Context, bikeID string) ([]RatingAggregate, error) {
	rows, err := s.db.QueryContext(ctx, listRatingAggregatesQuery, bikeID)
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

func (s *Store) ListWindowedRatingAggregatesByBike(ctx context.Context, bikeID string) ([]RatingAggregate, error) {
	rows, err := s.db.QueryContext(ctx, listWindowedRatingAggregatesQuery, bikeID)
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

func RecomputeAggregatesForBike(ctx context.Context, tx *sql.Tx, bikeID string) error {
	// Remove old aggregates for this bike
	if _, err := tx.ExecContext(ctx, deleteRatingAggregatesQuery, bikeID); err != nil {
		return err
	}

	// Recompute from review_ratings + reviews
	_, err := tx.ExecContext(ctx, recomputeRatingAggregatesQuery, bikeID)
	return err
}
