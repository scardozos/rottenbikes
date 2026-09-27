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
	//go:embed sql/get_bike_by_hash.sql
	getBikeByHashQuery string
	//go:embed sql/upsert_scan_event.sql
	upsertScanEventQuery string
	//go:embed sql/update_bike.sql
	updateBikeQuery string
	//go:embed sql/delete_bike.sql
	deleteBikeQuery string
	//go:embed sql/admin_delete_bike.sql
	adminDeleteBikeQuery string
	//go:embed sql/count_reviews_by_bike.sql
	countReviewsByBikeQuery string
	//go:embed sql/get_bike_details.sql
	getBikeDetailsQuery string
)

type Bike struct {
	NumericalID   string    `db:"numerical_id" json:"numerical_id"` // PK
	HashID        *string   `db:"hash_id" json:"hash_id"`
	IsElectric    bool      `db:"is_electric" json:"is_electric"`
	WasScanned    bool      `db:"was_scanned" json:"was_scanned"`
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

// ListBikes lists bikes. Test bikes (created by E2E test accounts) are only
// included when includeTest is set, i.e. for a test account.
func (s *Store) ListBikes(ctx context.Context, searchQuery, sortBy string, limit, offset int, includeTest bool) ([]Bike, error) {
	if sortBy == "" {
		sortBy = "recent" // default sort
	}

	rows, err := s.db.QueryContext(ctx, listBikesQuery, limit, offset, searchQuery, sortBy, includeTest)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bikes []Bike
	for rows.Next() {
		var b Bike
		var avgRating sql.NullFloat64
		if err := rows.Scan(&b.NumericalID, &b.HashID, &b.IsElectric, &b.WasScanned, &b.CreatedAt, &b.UpdatedAt, &avgRating); err != nil {
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

func (s *Store) CreateBike(ctx context.Context, numericalID string, hashID *string, isElectric, wasScanned bool, creatorID int64) (*Bike, error) {
	var b Bike
	err := s.db.QueryRowContext(ctx, createBikeQuery, numericalID, hashID, isElectric, wasScanned, creatorID).Scan(
		&b.NumericalID,
		&b.HashID,
		&b.IsElectric,
		&b.WasScanned,
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
	err := s.db.QueryRowContext(ctx, getBikeQuery, id).Scan(&b.NumericalID, &b.HashID, &b.IsElectric, &b.WasScanned, &b.CreatedAt, &b.UpdatedAt, &avgRating)
	if err != nil {
		return nil, err
	}
	if avgRating.Valid {
		b.AverageRating = &avgRating.Float64
	}
	return &b, nil
}

// GetBikeByHash looks up a bike by its QR hash_id and records a scan event for
// the authenticated poster. The scan event is what review creation later checks
// to set reviews.was_scanned, so the flag cannot be spoofed by clients.
func (s *Store) GetBikeByHash(ctx context.Context, hashID string, posterID int64) (*Bike, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var b Bike
	var avgRating sql.NullFloat64
	err = tx.QueryRowContext(ctx, getBikeByHashQuery, hashID).Scan(&b.NumericalID, &b.HashID, &b.IsElectric, &b.WasScanned, &b.CreatedAt, &b.UpdatedAt, &avgRating)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("get bike by hash: %w", err)
	}

	if _, err := tx.ExecContext(ctx, upsertScanEventQuery, posterID, b.NumericalID); err != nil {
		return nil, fmt.Errorf("record scan event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
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
		&bd.Bike.WasScanned,
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

// AdminDeleteBike deletes a bike without checking creator ownership. When
// audit is non-nil, a moderation_actions row is written in the same
// transaction.
func (s *Store) AdminDeleteBike(ctx context.Context, id string, audit *ModerationAudit) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, adminDeleteBikeQuery, id)
	if err != nil {
		return fmt.Errorf("delete bike: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}

	if audit != nil {
		if _, err := tx.ExecContext(ctx, insertBikeModerationActionQuery, audit.AdminPosterID, audit.Action, audit.TargetBikeID); err != nil {
			return fmt.Errorf("insert moderation action: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}
