package domain

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Bike struct {
	NumericalID int64     `db:"numerical_id" json:"numerical_id"` // PK
	HashID      string    `db:"hash_id" json:"hash_id"`
	IsElectric  bool      `db:"is_electric" json:"is_electric"`
	CreatedAt   time.Time `db:"created_ts" json:"created_ts"`
	UpdatedAt   time.Time `db:"updated_ts" json:"updated_ts"`
}

func ListBikes(ctx context.Context, db *sql.DB) ([]Bike, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT numerical_id, hash_id, is_electric
		FROM bikes
		ORDER BY numerical_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bikes []Bike
	for rows.Next() {
		var b Bike
		if err := rows.Scan(&b.NumericalID, &b.HashID, &b.IsElectric); err != nil {
			return nil, err
		}
		bikes = append(bikes, b)
	}
	return bikes, rows.Err()
}

func CreateBike(ctx context.Context, db *sql.DB, numericalID int64, hashID *string, isElectric bool, creatorID int64) (*Bike, error) {
	var b Bike
	err := db.QueryRowContext(ctx, `
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

func GetBike(ctx context.Context, db *sql.DB, id int64) (*Bike, error) {
	var b Bike
	err := db.QueryRowContext(ctx, `
		SELECT numerical_id, hash_id, is_electric, created_ts, updated_ts
		FROM bikes
		WHERE numerical_id = $1
	`, id).Scan(&b.NumericalID, &b.HashID, &b.IsElectric, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func UpdateBike(ctx context.Context, db *sql.DB, id int64, hashID *string, isElectric *bool) error {
	_, err := db.ExecContext(ctx, `
		UPDATE bikes
		SET
			hash_id     = COALESCE($1, hash_id),
			is_electric = COALESCE($2, is_electric),
			updated_ts  = NOW()
		WHERE numerical_id = $3
	`, hashID, isElectric, id)
	return err
}

func DeleteBike(ctx context.Context, db *sql.DB, id int64) error {
	_, err := db.ExecContext(ctx, `
		DELETE FROM bikes
		WHERE numerical_id = $1
	`, id)
	return err
}
