package domain

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Bike struct {
	NumericalID   int64     `db:"numerical_id" json:"numerical_id"` // PK
	HashID        string    `db:"hash_id" json:"hash_id"`
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
	rows, err := s.db.QueryContext(ctx, `
		SELECT 
			b.numerical_id, 
			b.hash_id, 
			b.is_electric, 
			b.created_ts, 
			b.updated_ts,
			ra.average_rating
		FROM bikes b
		LEFT JOIN rating_aggregates ra 
			ON b.numerical_id = ra.bike_numerical_id 
			AND ra.subcategory = 'overall'
		ORDER BY b.numerical_id
	`)
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

func (s *Store) CreateBike(ctx context.Context, numericalID int64, hashID *string, isElectric bool, creatorID int64) (*Bike, error) {
	var b Bike
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO bikes (numerical_id, hash_id, is_electric, creator_id)
		VALUES ($1, $2, $3, $4)
		RETURNING numerical_id, hash_id, is_electric, created_ts, updated_ts
	`, numericalID, hashID, isElectric, creatorID).Scan(
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

func (s *Store) GetBike(ctx context.Context, id int64) (*Bike, error) {
	var b Bike
	var avgRating sql.NullFloat64
	err := s.db.QueryRowContext(ctx, `
		SELECT 
			b.numerical_id, 
			b.hash_id, 
			b.is_electric, 
			b.created_ts, 
			b.updated_ts,
			ra.average_rating
		FROM bikes b
		LEFT JOIN rating_aggregates ra 
			ON b.numerical_id = ra.bike_numerical_id 
			AND ra.subcategory = 'overall'
		WHERE b.numerical_id = $1
	`, id).Scan(&b.NumericalID, &b.HashID, &b.IsElectric, &b.CreatedAt, &b.UpdatedAt, &avgRating)
	if err != nil {
		return nil, err
	}
	if avgRating.Valid {
		b.AverageRating = &avgRating.Float64
	}
	return &b, nil
}

func (s *Store) GetBikeDetails(ctx context.Context, id int64) (*BikeDetails, error) {
	b, err := s.GetBike(ctx, id)
	if err != nil {
		return nil, err
	}

	ratings, err := s.ListRatingAggregatesByBike(ctx, id)
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

func (s *Store) UpdateBike(ctx context.Context, id int64, hashID *string, isElectric *bool) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE bikes
		SET
			hash_id     = COALESCE($1, hash_id),
			is_electric = COALESCE($2, is_electric),
			updated_ts  = NOW()
		WHERE numerical_id = $3
	`, hashID, isElectric, id)
	return err
}

func (s *Store) DeleteBike(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM bikes
		WHERE numerical_id = $1
	`, id)
	return err
}
