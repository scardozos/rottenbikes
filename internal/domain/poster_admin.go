package domain

import (
	"context"
	_ "embed"
	"fmt"
	"time"
)

var (
	//go:embed sql/search_posters.sql
	searchPostersQuery string
	//go:embed sql/insert_moderation_action.sql
	insertModerationActionQuery string
)

// PosterSummary is an admin-facing view of a poster, including the amount of
// content they own so the admin can see the blast radius before purging.
type PosterSummary struct {
	PosterID    int64     `db:"poster_id" json:"poster_id"`
	Username    string    `db:"username" json:"username"`
	Email       string    `db:"email" json:"email"`
	CreatedAt   time.Time `db:"created_ts" json:"created_ts"`
	ReviewCount int64     `db:"review_count" json:"review_count"`
	BikeCount   int64     `db:"bike_count" json:"bike_count"`
}

// ModerationAudit is written into moderation_actions when an admin acts.
// It is kept in the same transaction as the action itself.
type ModerationAudit struct {
	AdminPosterID  int64
	Action         string
	TargetPosterID int64
}

// SearchPosters finds posters by email or username substring (admin only).
func (s *Store) SearchPosters(ctx context.Context, query string, limit int) ([]PosterSummary, error) {
	rows, err := s.db.QueryContext(ctx, searchPostersQuery, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search posters: %w", err)
	}
	defer rows.Close()

	posters := []PosterSummary{}
	for rows.Next() {
		var p PosterSummary
		if err := rows.Scan(&p.PosterID, &p.Username, &p.Email, &p.CreatedAt, &p.ReviewCount, &p.BikeCount); err != nil {
			return nil, fmt.Errorf("scan poster: %w", err)
		}
		posters = append(posters, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search posters: %w", err)
	}

	return posters, nil
}

// InsertModerationAction records an admin action in the audit log.
// Best-effort by design: callers decide whether a failure is fatal. The purge
// path passes the audit into DeletePoster so it lands in the same tx instead.
func (s *Store) InsertModerationAction(ctx context.Context, audit ModerationAudit) error {
	if _, err := s.db.ExecContext(ctx, insertModerationActionQuery, audit.AdminPosterID, audit.Action, audit.TargetPosterID); err != nil {
		return fmt.Errorf("insert moderation action: %w", err)
	}
	return nil
}
