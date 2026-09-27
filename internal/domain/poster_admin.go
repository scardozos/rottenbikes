package domain

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"time"
)

var (
	//go:embed sql/search_posters.sql
	searchPostersQuery string
	//go:embed sql/set_poster_role.sql
	setPosterRoleQuery string
	//go:embed sql/list_admin_posters.sql
	listAdminPostersQuery string
	//go:embed sql/get_poster_role_for_purge.sql
	getPosterRoleForPurgeQuery string
	//go:embed sql/insert_moderation_action.sql
	insertModerationActionQuery string
	//go:embed sql/insert_bike_moderation_action.sql
	insertBikeModerationActionQuery string
)

// PosterRole is a poster's authorization role, stored in posters.role and
// managed out-of-band via cmd/adminctl (promote/demote). The role is read on
// every token lookup, so changes take effect immediately — and no admin
// identity ever lives in git or env files.
type PosterRole string

const (
	PosterRoleUser  PosterRole = "user"
	PosterRoleAdmin PosterRole = "admin"
)

// ErrCannotPurgeAdmin is returned when an admin tries to purge another admin:
// one compromised admin must not be able to wipe the others.
var ErrCannotPurgeAdmin = errors.New("cannot purge an admin")

// PosterSummary is an admin-facing view of a poster, including the amount of
// content they own so the admin can see the blast radius before purging.
type PosterSummary struct {
	PosterID    int64      `db:"poster_id" json:"poster_id"`
	Username    string     `db:"username" json:"username"`
	Email       string     `db:"email" json:"email"`
	Role        PosterRole `db:"role" json:"role"`
	CreatedAt   time.Time  `db:"created_ts" json:"created_ts"`
	ReviewCount int64      `db:"review_count" json:"review_count"`
	BikeCount   int64      `db:"bike_count" json:"bike_count"`
}

// AdminPoster is a poster with its role, as returned by role management.
type AdminPoster struct {
	PosterID  int64      `db:"poster_id" json:"poster_id"`
	Email     string     `db:"email" json:"email"`
	Username  string     `db:"username" json:"username"`
	Role      PosterRole `db:"role" json:"role"`
	CreatedAt time.Time  `db:"created_ts" json:"created_ts"`
}

// ModerationAudit is written into moderation_actions when an admin acts.
// It is kept in the same transaction as the action itself. Exactly one of
// TargetPosterID / TargetBikeID is set, depending on the action.
type ModerationAudit struct {
	AdminPosterID  int64
	Action         string
	TargetPosterID int64
	TargetBikeID   string
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
		var role string
		if err := rows.Scan(&p.PosterID, &p.Username, &p.Email, &role, &p.CreatedAt, &p.ReviewCount, &p.BikeCount); err != nil {
			return nil, fmt.Errorf("scan poster: %w", err)
		}
		p.Role = PosterRole(role)
		posters = append(posters, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search posters: %w", err)
	}

	return posters, nil
}

// SetPosterRole promotes or demotes the poster identified by email or
// username. Returns the updated poster, or sql.ErrNoRows if no poster matches.
func (s *Store) SetPosterRole(ctx context.Context, identifier string, role PosterRole) (*AdminPoster, error) {
	var p AdminPoster
	var roleStr string
	err := s.db.QueryRowContext(ctx, setPosterRoleQuery, string(role), identifier).Scan(
		&p.PosterID, &p.Email, &p.Username, &roleStr,
	)
	if err != nil {
		return nil, err
	}
	p.Role = PosterRole(roleStr)
	return &p, nil
}

// ListAdmins returns all posters with the admin role. Only used out of
// band (via cmd/adminctl) for auditing purposes.
func (s *Store) ListAdmins(ctx context.Context) ([]AdminPoster, error) {
	rows, err := s.db.QueryContext(ctx, listAdminPostersQuery)
	if err != nil {
		return nil, fmt.Errorf("list admins: %w", err)
	}
	defer rows.Close()

	admins := []AdminPoster{}
	for rows.Next() {
		var p AdminPoster
		var roleStr string
		if err := rows.Scan(&p.PosterID, &p.Email, &p.Username, &roleStr, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan admin: %w", err)
		}
		p.Role = PosterRole(roleStr)
		admins = append(admins, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list admins: %w", err)
	}

	return admins, nil
}
