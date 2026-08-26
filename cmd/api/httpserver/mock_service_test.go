package httpserver

import (
	"context"

	"github.com/scardozos/rottenbikes/internal/domain"
)

type MockService struct {
	RegisterFunc                     func(ctx context.Context, username, email string) (string, string, error)
	CreateMagicLinkFunc              func(ctx context.Context, identifier string) (string, string, string, error)
	ConfirmMagicLinkFunc             func(ctx context.Context, token string) (*domain.ConfirmResult, error)
	GetPosterByAPITokenFunc          func(ctx context.Context, token string) (*domain.AuthPoster, error)
	CheckMagicLinkStatusFunc         func(ctx context.Context, token string) (string, error)
	DeletePosterFunc                 func(ctx context.Context, posterID int64, deleteContent bool) error
	ListBikesFunc                    func(ctx context.Context, searchQuery, sortBy string, limit, offset int) ([]domain.Bike, error)
	CreateBikeFunc                   func(ctx context.Context, numericalID string, hashID *string, isElectric bool, creatorID int64) (*domain.Bike, error)
	GetBikeFunc                      func(ctx context.Context, id string) (*domain.Bike, error)
	GetBikeDetailsFunc               func(ctx context.Context, id string, limit, offset int) (*domain.BikeDetails, error)
	UpdateBikeFunc                   func(ctx context.Context, id string, hashID *string, isElectric *bool, creatorID int64) error
	DeleteBikeFunc                   func(ctx context.Context, id string, creatorID int64) error
	ListRatingAggregatesByBikeFunc   func(ctx context.Context, bikeID string) ([]domain.RatingAggregate, error)
	ListReviewsWithRatingsByBikeFunc func(ctx context.Context, bikeID string, limit, offset int) ([]domain.ReviewWithRatings, error)
	CreateReviewWithRatingsFunc      func(ctx context.Context, in domain.CreateReviewInput) (int64, error)
	UpdateReviewWithRatingsFunc      func(ctx context.Context, in domain.UpdateReviewInput) error
	GetReviewWithRatingsByIDFunc     func(ctx context.Context, reviewID int64) (*domain.ReviewWithRatings, error)
	DeleteReviewFunc                 func(ctx context.Context, reviewID int64, posterID int64) error
	ListReviewsWithRatingsByUserFunc func(ctx context.Context, posterID int64, limit, offset int) ([]domain.ReviewWithRatings, error)
	HealthCheckFunc                  func(ctx context.Context) error
}

func (m *MockService) Register(ctx context.Context, username, email string) (string, string, error) {
	return m.RegisterFunc(ctx, username, email)
}

func (m *MockService) CreateMagicLink(ctx context.Context, identifier string) (string, string, string, error) {
	return m.CreateMagicLinkFunc(ctx, identifier)
}

func (m *MockService) ConfirmMagicLink(ctx context.Context, token string) (*domain.ConfirmResult, error) {
	return m.ConfirmMagicLinkFunc(ctx, token)
}

func (m *MockService) GetPosterByAPIToken(ctx context.Context, token string) (*domain.AuthPoster, error) {
	return m.GetPosterByAPITokenFunc(ctx, token)
}

func (m *MockService) CheckMagicLinkStatus(ctx context.Context, token string) (string, error) {
	return m.CheckMagicLinkStatusFunc(ctx, token)
}

func (m *MockService) DeletePoster(ctx context.Context, posterID int64, deleteContent bool) error {
	if m.DeletePosterFunc != nil {
		return m.DeletePosterFunc(ctx, posterID, deleteContent)
	}
	return nil
}

func (m *MockService) ListBikes(ctx context.Context, searchQuery, sortBy string, limit, offset int) ([]domain.Bike, error) {
	if m.ListBikesFunc != nil {
		return m.ListBikesFunc(ctx, searchQuery, sortBy, limit, offset)
	}
	return nil, nil
}

func (m *MockService) CreateBike(ctx context.Context, numericalID string, hashID *string, isElectric bool, creatorID int64) (*domain.Bike, error) {
	return m.CreateBikeFunc(ctx, numericalID, hashID, isElectric, creatorID)
}

func (m *MockService) GetBike(ctx context.Context, id string) (*domain.Bike, error) {
	return m.GetBikeFunc(ctx, id)
}

func (m *MockService) GetBikeDetails(ctx context.Context, id string, limit, offset int) (*domain.BikeDetails, error) {
	return m.GetBikeDetailsFunc(ctx, id, limit, offset)
}

func (m *MockService) UpdateBike(ctx context.Context, id string, hashID *string, isElectric *bool, creatorID int64) error {
	return m.UpdateBikeFunc(ctx, id, hashID, isElectric, creatorID)
}

func (m *MockService) DeleteBike(ctx context.Context, id string, creatorID int64) error {
	return m.DeleteBikeFunc(ctx, id, creatorID)
}

func (m *MockService) ListRatingAggregatesByBike(ctx context.Context, bikeID string) ([]domain.RatingAggregate, error) {
	return m.ListRatingAggregatesByBikeFunc(ctx, bikeID)
}

func (m *MockService) ListReviewsWithRatingsByBike(ctx context.Context, bikeID string, limit, offset int) ([]domain.ReviewWithRatings, error) {
	if m.ListReviewsWithRatingsByBikeFunc != nil {
		return m.ListReviewsWithRatingsByBikeFunc(ctx, bikeID, limit, offset)
	}
	return nil, nil
}

func (m *MockService) ListReviewsWithRatingsByUser(ctx context.Context, posterID int64, limit, offset int) ([]domain.ReviewWithRatings, error) {
	if m.ListReviewsWithRatingsByUserFunc != nil {
		return m.ListReviewsWithRatingsByUserFunc(ctx, posterID, limit, offset)
	}
	return nil, nil
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

func (m *MockService) HealthCheck(ctx context.Context) error {
	if m.HealthCheckFunc != nil {
		return m.HealthCheckFunc(ctx)
	}
	return nil
}
