package domain

import (
	"context"
	"database/sql"
	"fmt"
	"net/mail"
	"regexp"
)

type Service interface {
	// Auth
	Register(ctx context.Context, username, email string) (string, string, error)
	CreateMagicLink(ctx context.Context, identifier string) (string, string, string, error)
	ConfirmMagicLink(ctx context.Context, token string) (*ConfirmResult, error)
	GetPosterByAPIToken(ctx context.Context, token string) (*AuthPoster, error)
	CheckMagicLinkStatus(ctx context.Context, token string) (string, error)
	DeletePoster(ctx context.Context, posterID int64, deleteContent bool) error

	// Bike
	ListBikes(ctx context.Context, limit, offset int) ([]Bike, error)
	CreateBike(ctx context.Context, numericalID string, hashID *string, isElectric bool, creatorID int64) (*Bike, error)
	GetBike(ctx context.Context, id string) (*Bike, error)
	GetBikeDetails(ctx context.Context, id string, limit, offset int) (*BikeDetails, error)
	UpdateBike(ctx context.Context, id string, hashID *string, isElectric *bool, creatorID int64) error
	DeleteBike(ctx context.Context, id string, creatorID int64) error

	// Rating Aggregate
	ListRatingAggregatesByBike(ctx context.Context, bikeID string) ([]RatingAggregate, error)

	// Review
	ListReviewsWithRatingsByBike(ctx context.Context, bikeID string, limit, offset int) ([]ReviewWithRatings, error)
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

type service struct {
	store *Store
}

func NewService(store *Store) Service {
	return &service{store: store}
}

// Auth

func (s *service) Register(ctx context.Context, username, email string) (string, string, error) {
	// Validate email format
	_, err := mail.ParseAddress(email)
	if err != nil {
		return "", "", fmt.Errorf("invalid email format")
	}

	// Validate username format (alphanumeric and dots only)
	validUsername := regexp.MustCompile(`^[a-zA-Z0-9.]+$`)
	if !validUsername.MatchString(username) {
		return "", "", fmt.Errorf("username can only contain letters, numbers and dots")
	}

	return s.store.Register(ctx, username, email)
}

func (s *service) CreateMagicLink(ctx context.Context, identifier string) (string, string, string, error) {
	return s.store.CreateMagicLink(ctx, identifier)
}

func (s *service) ConfirmMagicLink(ctx context.Context, token string) (*ConfirmResult, error) {
	return s.store.ConfirmMagicLink(ctx, token)
}

func (s *service) GetPosterByAPIToken(ctx context.Context, token string) (*AuthPoster, error) {
	return s.store.GetPosterByAPIToken(ctx, token)
}

func (s *service) CheckMagicLinkStatus(ctx context.Context, token string) (string, error) {
	return s.store.CheckMagicLinkStatus(ctx, token)
}

func (s *service) DeletePoster(ctx context.Context, posterID int64, deleteContent bool) error {
	return s.store.DeletePoster(ctx, posterID, deleteContent)
}

// Bike

func (s *service) ListBikes(ctx context.Context, limit, offset int) ([]Bike, error) {
	return s.store.ListBikes(ctx, limit, offset)
}

func (s *service) CreateBike(ctx context.Context, numericalID string, hashID *string, isElectric bool, creatorID int64) (*Bike, error) {
	// Validate numerical_id consistency (4-5 digits)
	validID := true
	if len(numericalID) < 4 || len(numericalID) > 5 {
		validID = false
	} else {
		for _, r := range numericalID {
			if r < '0' || r > '9' {
				validID = false
				break
			}
		}
	}
	if !validID {
		return nil, fmt.Errorf("numerical_id must be 4-5 digits")
	}

	// Treat empty string hash_id as nil
	if hashID != nil && *hashID == "" {
		hashID = nil
	}

	if hashID != nil && !isDomainAlphanumeric(*hashID) {
		return nil, fmt.Errorf("hash_id must be alphanumeric")
	}

	return s.store.CreateBike(ctx, numericalID, hashID, isElectric, creatorID)
}

func (s *service) GetBike(ctx context.Context, id string) (*Bike, error) {
	return s.store.GetBike(ctx, id)
}

func (s *service) GetBikeDetails(ctx context.Context, id string, limit, offset int) (*BikeDetails, error) {
	return s.store.GetBikeDetails(ctx, id, limit, offset)
}

func (s *service) UpdateBike(ctx context.Context, id string, hashID *string, isElectric *bool, creatorID int64) error {
	if hashID != nil && *hashID != "" && !isDomainAlphanumeric(*hashID) {
		return fmt.Errorf("hash_id must be alphanumeric")
	}
	return s.store.UpdateBike(ctx, id, hashID, isElectric, creatorID)
}

func (s *service) DeleteBike(ctx context.Context, id string, creatorID int64) error {
	return s.store.DeleteBike(ctx, id, creatorID)
}

// Rating Aggregate

func (s *service) ListRatingAggregatesByBike(ctx context.Context, bikeID string) ([]RatingAggregate, error) {
	return s.store.ListRatingAggregatesByBike(ctx, bikeID)
}

// Review

func (s *service) ListReviewsWithRatingsByBike(ctx context.Context, bikeID string, limit, offset int) ([]ReviewWithRatings, error) {
	return s.store.ListReviewsWithRatingsByBike(ctx, bikeID, limit, offset)
}

func (s *service) CreateReviewWithRatings(ctx context.Context, in CreateReviewInput) (int64, error) {
	if err := validateReviewScores(in.Overall, in.Breaks, in.Seat, in.Sturdiness, in.Power, in.Pedals); err != nil {
		return 0, err
	}
	return s.store.CreateReviewWithRatings(ctx, in)
}

func (s *service) UpdateReviewWithRatings(ctx context.Context, in UpdateReviewInput) error {
	if err := validateReviewScores(in.Overall, in.Breaks, in.Seat, in.Sturdiness, in.Power, in.Pedals); err != nil {
		return err
	}
	return s.store.UpdateReviewWithRatings(ctx, in)
}

func (s *service) GetReviewWithRatingsByID(ctx context.Context, reviewID int64) (*ReviewWithRatings, error) {
	return s.store.GetReviewWithRatingsByID(ctx, reviewID)
}

func (s *service) DeleteReview(ctx context.Context, reviewID int64, posterID int64) error {
	return s.store.DeleteReview(ctx, reviewID, posterID)
}

// Helpers

func validateScore(sub RatingSubcategory, val *int16) error {
	if val == nil {
		return nil
	}
	if *val < 1 || *val > 5 {
		return fmt.Errorf("invalid score %d for %s", *val, sub)
	}
	return nil
}

func validateReviewScores(overall, breaks, seat, sturdiness, power, pedals *int16) error {
	if err := validateScore(RatingSubcategoryOverall, overall); err != nil {
		return err
	}
	if err := validateScore(RatingSubcategoryBreaks, breaks); err != nil {
		return err
	}
	if err := validateScore(RatingSubcategorySeat, seat); err != nil {
		return err
	}
	if err := validateScore(RatingSubcategorySturdiness, sturdiness); err != nil {
		return err
	}
	if err := validateScore(RatingSubcategoryPower, power); err != nil {
		return err
	}
	if err := validateScore(RatingSubcategoryPedals, pedals); err != nil {
		return err
	}
	return nil
}

func isDomainAlphanumeric(s string) bool {
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}
