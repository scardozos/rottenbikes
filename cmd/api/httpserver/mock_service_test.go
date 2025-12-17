package httpserver

import (
	"context"

	"github.com/scardozos/rottenbikes/internal/domain"
)

type MockService struct {
	CreateMagicLinkFunc              func(ctx context.Context, email string) (string, error)
	ConfirmMagicLinkFunc             func(ctx context.Context, token string) (*domain.ConfirmResult, error)
	GetPosterByAPITokenFunc          func(ctx context.Context, token string) (*domain.AuthPoster, error)
	ListBikesFunc                    func(ctx context.Context) ([]domain.Bike, error)
	CreateBikeFunc                   func(ctx context.Context, numericalID int64, hashID *string, isElectric bool, creatorID int64) (*domain.Bike, error)
	GetBikeFunc                      func(ctx context.Context, id int64) (*domain.Bike, error)
	UpdateBikeFunc                   func(ctx context.Context, id int64, hashID *string, isElectric *bool) error
	DeleteBikeFunc                   func(ctx context.Context, id int64) error
	ListRatingAggregatesFunc         func(ctx context.Context) ([]domain.RatingAggregate, error)
	ListRatingAggregatesByBikeFunc   func(ctx context.Context, bikeID int64) ([]domain.RatingAggregate, error)
	ListReviewsWithRatingsFunc       func(ctx context.Context) ([]domain.ReviewWithRatings, error)
	ListReviewsWithRatingsByBikeFunc func(ctx context.Context, bikeID int64) ([]domain.ReviewWithRatings, error)
	CreateReviewWithRatingsFunc      func(ctx context.Context, in domain.CreateReviewInput) (int64, error)
	UpdateReviewWithRatingsFunc      func(ctx context.Context, in domain.UpdateReviewInput) error
	GetReviewWithRatingsByIDFunc     func(ctx context.Context, reviewID int64) (*domain.ReviewWithRatings, error)
	DeleteReviewFunc                 func(ctx context.Context, reviewID int64, posterID int64) error
}

func (m *MockService) CreateMagicLink(ctx context.Context, email string) (string, error) {
	return m.CreateMagicLinkFunc(ctx, email)
}

func (m *MockService) ConfirmMagicLink(ctx context.Context, token string) (*domain.ConfirmResult, error) {
	return m.ConfirmMagicLinkFunc(ctx, token)
}

func (m *MockService) GetPosterByAPIToken(ctx context.Context, token string) (*domain.AuthPoster, error) {
	return m.GetPosterByAPITokenFunc(ctx, token)
}

func (m *MockService) ListBikes(ctx context.Context) ([]domain.Bike, error) {
	return m.ListBikesFunc(ctx)
}

func (m *MockService) CreateBike(ctx context.Context, numericalID int64, hashID *string, isElectric bool, creatorID int64) (*domain.Bike, error) {
	return m.CreateBikeFunc(ctx, numericalID, hashID, isElectric, creatorID)
}

func (m *MockService) GetBike(ctx context.Context, id int64) (*domain.Bike, error) {
	return m.GetBikeFunc(ctx, id)
}

func (m *MockService) UpdateBike(ctx context.Context, id int64, hashID *string, isElectric *bool) error {
	return m.UpdateBikeFunc(ctx, id, hashID, isElectric)
}

func (m *MockService) DeleteBike(ctx context.Context, id int64) error {
	return m.DeleteBikeFunc(ctx, id)
}

func (m *MockService) ListRatingAggregates(ctx context.Context) ([]domain.RatingAggregate, error) {
	return m.ListRatingAggregatesFunc(ctx)
}

func (m *MockService) ListRatingAggregatesByBike(ctx context.Context, bikeID int64) ([]domain.RatingAggregate, error) {
	return m.ListRatingAggregatesByBikeFunc(ctx, bikeID)
}

func (m *MockService) ListReviewsWithRatings(ctx context.Context) ([]domain.ReviewWithRatings, error) {
	return m.ListReviewsWithRatingsFunc(ctx)
}

func (m *MockService) ListReviewsWithRatingsByBike(ctx context.Context, bikeID int64) ([]domain.ReviewWithRatings, error) {
	return m.ListReviewsWithRatingsByBikeFunc(ctx, bikeID)
}

func (m *MockService) CreateReviewWithRatings(ctx context.Context, in domain.CreateReviewInput) (int64, error) {
	return m.CreateReviewWithRatingsFunc(ctx, in)
}

func (m *MockService) UpdateReviewWithRatings(ctx context.Context, in domain.UpdateReviewInput) error {
	return m.UpdateReviewWithRatingsFunc(ctx, in)
}

func (m *MockService) GetReviewWithRatingsByID(ctx context.Context, reviewID int64) (*domain.ReviewWithRatings, error) {
	return m.GetReviewWithRatingsByIDFunc(ctx, reviewID)
}

func (m *MockService) DeleteReview(ctx context.Context, reviewID int64, posterID int64) error {
	return m.DeleteReviewFunc(ctx, reviewID, posterID)
}
