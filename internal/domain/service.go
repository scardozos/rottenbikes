package domain

import (
	"context"
	"database/sql"
)

type Service interface {
	// Auth
	Register(ctx context.Context, username, email string) (string, string, error)
	CreateMagicLink(ctx context.Context, identifier string) (string, string, string, error)
	ConfirmMagicLink(ctx context.Context, token string) (*ConfirmResult, error)
	GetPosterByAPIToken(ctx context.Context, token string) (*AuthPoster, error)
	CheckMagicLinkStatus(ctx context.Context, token string) (string, error)
	RevokeAPIToken(ctx context.Context, token string) error
	DeletePoster(ctx context.Context, posterID int64, deleteContent bool) error

	// Admin (moderation)
	SearchPosters(ctx context.Context, query string, limit int) ([]PosterSummary, error)
	PurgePoster(ctx context.Context, adminPosterID, targetPosterID int64) error
	AdminDeleteBike(ctx context.Context, adminPosterID int64, bikeID string) error

	// Infrastructure
	HealthCheck(ctx context.Context) error

	// Bike
	ListBikes(ctx context.Context, searchQuery, sortBy string, limit, offset int) ([]Bike, error)
	CreateBike(ctx context.Context, numericalID string, hashID *string, isElectric, wasScanned bool, creatorID int64) (*Bike, error)
	GetBike(ctx context.Context, id string) (*Bike, error)
	GetBikeByHash(ctx context.Context, hashID string, posterID int64) (*Bike, error)
	GetBikeDetails(ctx context.Context, id string, limit, offset int) (*BikeDetails, error)
	UpdateBike(ctx context.Context, id string, hashID *string, isElectric *bool, creatorID int64) error
	DeleteBike(ctx context.Context, id string, creatorID int64) error

	// Rating Aggregate
	ListRatingAggregatesByBike(ctx context.Context, bikeID string) ([]RatingAggregate, error)

	// Review
	ListReviewsWithRatingsByBike(ctx context.Context, bikeID string, limit, offset int) ([]ReviewWithRatings, error)
	ListReviewsWithRatingsByUser(ctx context.Context, posterID int64, limit, offset int) ([]ReviewWithRatings, error)
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
	if err := validateEmail(email); err != nil {
		return "", "", err
	}
	if err := validateUsername(username); err != nil {
		return "", "", err
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

func (s *service) RevokeAPIToken(ctx context.Context, token string) error {
	return s.store.RevokeAPIToken(ctx, token)
}

func (s *service) DeletePoster(ctx context.Context, posterID int64, deleteContent bool) error {
	return s.store.DeletePoster(ctx, posterID, deleteContent, nil)
}

// Admin (moderation)

// SearchPosters finds posters by email or username substring (admin only).
func (s *service) SearchPosters(ctx context.Context, query string, limit int) ([]PosterSummary, error) {
	return s.store.SearchPosters(ctx, query, limit)
}

// PurgePoster removes a malicious poster and all their content (reviews,
// ratings, created bikes, sessions) in one transaction, and writes an audit
// row so the action is traceable. Deleting created bikes cascades to other
// posters' reviews on those bikes.
func (s *service) PurgePoster(ctx context.Context, adminPosterID, targetPosterID int64) error {
	return s.store.DeletePoster(ctx, targetPosterID, true, &ModerationAudit{
		AdminPosterID:  adminPosterID,
		Action:         "purge_poster",
		TargetPosterID: targetPosterID,
	})
}

// AdminDeleteBike allows an admin to delete a bike regardless of who created
// it, and writes an audit row so the action is traceable.
func (s *service) AdminDeleteBike(ctx context.Context, adminPosterID int64, bikeID string) error {
	return s.store.AdminDeleteBike(ctx, bikeID, &ModerationAudit{
		AdminPosterID: adminPosterID,
		Action:        "delete_bike",
		TargetBikeID:  bikeID,
	})
}

// Infrastructure

func (s *service) HealthCheck(ctx context.Context) error {
	return s.store.db.PingContext(ctx)
}

// Bike

func (s *service) ListBikes(ctx context.Context, searchQuery, sortBy string, limit, offset int) ([]Bike, error) {
	return s.store.ListBikes(ctx, searchQuery, sortBy, limit, offset)
}

func (s *service) CreateBike(ctx context.Context, numericalID string, hashID *string, isElectric, wasScanned bool, creatorID int64) (*Bike, error) {
	if err := validateNumericalID(numericalID); err != nil {
		return nil, err
	}

	// Treat empty string hash_id as nil
	if hashID != nil && *hashID == "" {
		hashID = nil
	}

	if err := validateHashID(hashID); err != nil {
		return nil, err
	}

	return s.store.CreateBike(ctx, numericalID, hashID, isElectric, wasScanned, creatorID)
}

// GetBikeByHash looks up a bike by QR hash and records the scan server-side
// (see Store.GetBikeByHash). wasScanned on bike creation is client-declared
// (there is nothing to verify an unknown QR against), but scans of existing
// bikes are recorded here and later prove a review came from a real scan.
func (s *service) GetBikeByHash(ctx context.Context, hashID string, posterID int64) (*Bike, error) {
	return s.store.GetBikeByHash(ctx, hashID, posterID)
}

func (s *service) GetBike(ctx context.Context, id string) (*Bike, error) {
	return s.store.GetBike(ctx, id)
}

func (s *service) GetBikeDetails(ctx context.Context, id string, limit, offset int) (*BikeDetails, error) {
	return s.store.GetBikeDetails(ctx, id, limit, offset)
}

func (s *service) UpdateBike(ctx context.Context, id string, hashID *string, isElectric *bool, creatorID int64) error {
	if err := validateHashID(hashID); err != nil {
		return err
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

func (s *service) ListReviewsWithRatingsByUser(ctx context.Context, posterID int64, limit, offset int) ([]ReviewWithRatings, error) {
	return s.store.ListReviewsWithRatingsByUser(ctx, posterID, limit, offset)
}

func (s *service) CreateReviewWithRatings(ctx context.Context, in CreateReviewInput) (int64, error) {
	if err := validateComment(in.Comment); err != nil {
		return 0, err
	}
	if err := validateReviewScores(in.Overall, in.Breaks, in.Seat, in.Sturdiness, in.Power, in.Pedals); err != nil {
		return 0, err
	}
	return s.store.CreateReviewWithRatings(ctx, in)
}

func (s *service) UpdateReviewWithRatings(ctx context.Context, in UpdateReviewInput) error {
	if err := validateComment(in.Comment); err != nil {
		return err
	}
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
