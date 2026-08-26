package domain

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"time"
)

var (
	//go:embed sql/list_bikes.sql
	listBikesQuery string
	//go:embed sql/create_bike.sql
	createBikeQuery string
	//go:embed sql/get_bike_by_id.sql
	getBikeQuery string
	//go:embed sql/update_bike.sql
	updateBikeQuery string
	//go:embed sql/delete_bike.sql
	deleteBikeQuery string
	//go:embed sql/count_reviews_by_bike.sql
	countReviewsByBikeQuery string
	//go:embed sql/get_bike_details.sql
	getBikeDetailsQuery string
)

type Bike struct {
	NumericalID   string    `db:"numerical_id" json:"numerical_id"` // PK
	HashID        *string   `db:"hash_id" json:"hash_id"`
	IsElectric    bool      `db:"is_electric" json:"is_electric"`
	AverageRating *float64  `db:"average_rating" json:"average_rating"`
	CreatedAt     time.Time `db:"created_ts" json:"created_ts"`
	UpdatedAt     time.Time `db:"updated_ts" json:"updated_ts"`
}

type BikeDetails struct {
	Bike
	Ratings      []RatingAggregate   `json:"ratings"`
	Reviews      []ReviewWithRatings `json:"reviews"`
	TotalReviews int                 `json:"total_reviews"`
}

func (s *Store) ListBikes(ctx context.Context, searchQuery, sortBy string, limit, offset int) ([]Bike, error) {
	if sortBy == "" {
		sortBy = "recent" // default sort
	}

	rows, err := s.db.QueryContext(ctx, listBikesQuery, limit, offset, searchQuery, sortBy)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bikes []Bike
	for rows.Next() {
		var b Bike
		var avgRating sql.NullFloat64
		if err := rows.Scan(&b.NumericalID, &b.HashID, &b.IsElectric, &b.CreatedAt, &b.UpdatedAt, &avgRating); err != nil {
			return nil, err
		}
		if avgRating.Valid {
			b.AverageRating = &avgRating.Float64
		}
		bikes = append(bikes, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return bikes, nil
}

func (s *Store) CreateBike(ctx context.Context, numericalID string, hashID *string, isElectric bool, creatorID int64) (*Bike, error) {
	var b Bike
	err := s.db.QueryRowContext(ctx, createBikeQuery, numericalID, hashID, isElectric, creatorID).Scan(
		&b.NumericalID,
		&b.HashID,
		&b.IsElectric,
		&b.CreatedAt,
		&b.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert bike: %w", err)
	}
	return &b, nil
}

func (s *Store) GetBike(ctx context.Context, id string) (*Bike, error) {
	var b Bike
	var avgRating sql.NullFloat64
	err := s.db.QueryRowContext(ctx, getBikeQuery, id).Scan(&b.NumericalID, &b.HashID, &b.IsElectric, &b.CreatedAt, &b.UpdatedAt, &avgRating)
	if err != nil {
		return nil, err
	}
	if avgRating.Valid {
		b.AverageRating = &avgRating.Float64
	}
	return &b, nil
}

func (s *Store) GetBikeDetails(ctx context.Context, id string, limit, offset int) (*BikeDetails, error) {
	var bd BikeDetails
	var avgRating sql.NullFloat64
	var reviewsJSON []byte

	err := s.db.QueryRowContext(ctx, getBikeDetailsQuery, id, limit, offset).Scan(
		&bd.Bike.NumericalID,
		&bd.Bike.HashID,
		&bd.Bike.IsElectric,
		&bd.Bike.CreatedAt,
		&bd.Bike.UpdatedAt,
		&avgRating,
		&bd.TotalReviews,
		&reviewsJSON,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("failed to fetch bike details: %w", err)
	}

	if avgRating.Valid {
		bd.Bike.AverageRating = &avgRating.Float64
	}

	aggs, err := s.ListWindowedRatingAggregatesByBike(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch windowed ratings: %w", err)
	}
	bd.Ratings = aggs

	if err := json.Unmarshal(reviewsJSON, &bd.Reviews); err != nil {
		return nil, fmt.Errorf("failed to decode reviews json: %w", err)
	}

	return &bd, nil
}

// UpdateBike updates a bike owned by creatorID. Only the creator may update a
// bike; a non-creator (or a missing bike) yields sql.ErrNoRows so callers can
// map both to a single 404 without leaking whether the bike exists.
func (s *Store) UpdateBike(ctx context.Context, id string, hashID *string, isElectric *bool, creatorID int64) error {
	res, err := s.db.ExecContext(ctx, updateBikeQuery, hashID, isElectric, id, creatorID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// DeleteBike deletes a bike owned by creatorID. Only the creator may delete a
// bike (which cascades to all its reviews + aggregates); a non-creator (or a
// missing bike) yields sql.ErrNoRows.
func (s *Store) DeleteBike(ctx context.Context, id string, creatorID int64) error {
	res, err := s.db.ExecContext(ctx, deleteBikeQuery, id, creatorID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
