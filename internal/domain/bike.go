package domain

import (
	"context"
	"database/sql"
	_ "embed"
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
	Ratings []RatingAggregate   `json:"ratings"`
	Reviews []ReviewWithRatings `json:"reviews"`
}

func (s *Store) ListBikes(ctx context.Context) ([]Bike, error) {
	rows, err := s.db.QueryContext(ctx, listBikesQuery)
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
	return bikes, rows.Err()
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

func (s *Store) GetBikeDetails(ctx context.Context, id string) (*BikeDetails, error) {
	b, err := s.GetBike(ctx, id)
	if err != nil {
		return nil, err
	}

	ratings, err := s.ListWindowedRatingAggregatesByBike(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ratings: %w", err)
	}

	reviews, err := s.ListReviewsWithRatingsByBike(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch reviews: %w", err)
	}

	return &BikeDetails{
		Bike:    *b,
		Ratings: ratings,
		Reviews: reviews,
	}, nil
}

func (s *Store) UpdateBike(ctx context.Context, id string, hashID *string, isElectric *bool) error {
	_, err := s.db.ExecContext(ctx, updateBikeQuery, hashID, isElectric, id)
	return err
}

func (s *Store) DeleteBike(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, deleteBikeQuery, id)
	return err
}
