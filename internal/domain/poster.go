package domain

import "time"

type Poster struct {
	PosterID  int64     `db:"poster_id"` // PK
	Email     string    `db:"email"`
	Username  string    `db:"username"`
	CreatedAt time.Time `db:"created_ts"`
}
