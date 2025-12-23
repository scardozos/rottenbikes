package domain

import (
	"context"
	"database/sql"
)

type Service interface {
	// Auth
	Register(ctx context.Context, username, email string) (string, error)
	CreateMagicLink(ctx context.Context, identifier string) (string, string, error)
	ConfirmMagicLink(ctx context.Context, token string) (*ConfirmResult, error)
	GetPosterByAPIToken(ctx context.Context, token string) (*AuthPoster, error)
	CheckMagicLinkStatus(ctx context.Context, token string) (string, error)

	// Bike
	ListBikes(ctx context.Context) ([]Bike, error)
	CreateBike(ctx context.Context, numericalID int64, hashID *string, isElectric bool, creatorID int64) (*Bike, error)
	GetBike(ctx context.Context, id int64) (*Bike, error)
	GetBikeDetails(ctx context.Context, id int64) (*BikeDetails, error)
	UpdateBike(ctx context.Context, id int64, hashID *string, isElectric *bool) error
	DeleteBike(ctx context.Context, id int64) error

	// Rating Aggregate

	ListRatingAggregatesByBike(ctx context.Context, bikeID int64) ([]RatingAggregate, error)

	// Review

	ListReviewsWithRatingsByBike(ctx context.Context, bikeID int64) ([]ReviewWithRatings, error)
	CreateReviewWithRatings(ctx context.Context, in CreateReviewInput) (int64, error)
	UpdateReviewWithRatings(ctx context.Context, in UpdateReviewInput) error
	GetReviewWithRatingsByID(ctx context.Context, reviewID int64) (*ReviewWithRatings, error)
	DeleteReview(ctx context.Context, reviewID int64, posterID int64) error
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}
