-- Posters (users)
CREATE TABLE posters (
    poster_id BIGSERIAL PRIMARY KEY,
    email     text      NOT NULL UNIQUE,
    username  text      NOT NULL UNIQUE,
    api_token TEXT,
    api_token_expires_ts TIMESTAMPTZ,
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    created_ts TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT email_valid CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$')
);


-- Bikes
CREATE TABLE bikes (
    numerical_id    BIGSERIAL PRIMARY KEY,
    hash_id         TEXT UNIQUE,
    is_electric     BOOLEAN     NOT NULL DEFAULT FALSE,
    creator_id      BIGINT,
    created_ts      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_ts      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_bikes_creator
        FOREIGN KEY (creator_id) REFERENCES posters (poster_id)
        ON DELETE SET NULL
);


-- magic links for email-based auth
CREATE TABLE magic_links (
    id          BIGSERIAL PRIMARY KEY,
    poster_id   BIGINT      NOT NULL REFERENCES posters(poster_id) ON DELETE CASCADE,
    token       TEXT        NOT NULL UNIQUE,
    api_token   TEXT,
    created_ts  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    consumed_ts TIMESTAMPTZ,
    expires_ts  TIMESTAMPTZ NOT NULL
);

-- reviews (one per bike per poster)
CREATE TABLE reviews (
    review_id          BIGSERIAL PRIMARY KEY,
    poster_id          BIGINT,
    bike_numerical_id  BIGINT      NOT NULL,
    bike_img           TEXT,
    comment            VARCHAR(500),
    created_ts         TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_reviews_poster
        FOREIGN KEY (poster_id) REFERENCES posters (poster_id)
        ON DELETE SET NULL,
    CONSTRAINT fk_reviews_bike
        FOREIGN KEY (bike_numerical_id) REFERENCES bikes (numerical_id)
        ON DELETE CASCADE
);

-- review_ratings (one row per review + subcategory)
CREATE TYPE rating_subcategory AS ENUM (
    'overall',
    'breaks',
    'seat',
    'sturdiness',
    'power',
    'pedals'
);

CREATE TABLE review_ratings (
    review_id   BIGINT              NOT NULL,
    subcategory rating_subcategory  NOT NULL,
    score       SMALLINT            NOT NULL CHECK (score BETWEEN 1 AND 5),

    PRIMARY KEY (review_id, subcategory),
    CONSTRAINT fk_review_ratings_review
        FOREIGN KEY (review_id) REFERENCES reviews (review_id)
        ON DELETE CASCADE
);

-- rating_aggregates (cached aggregates per bike + subcategory)
CREATE TABLE rating_aggregates (
    bike_numerical_id BIGINT             NOT NULL,
    subcategory       rating_subcategory NOT NULL,
    rating_sum        BIGINT             NOT NULL DEFAULT 0,
    rating_count      BIGINT             NOT NULL DEFAULT 0,
    average_rating    NUMERIC(3,2)       NOT NULL DEFAULT 0,

    PRIMARY KEY (bike_numerical_id, subcategory),
    CONSTRAINT fk_rating_agg_bike
        FOREIGN KEY (bike_numerical_id) REFERENCES bikes (numerical_id)
        ON DELETE CASCADE
);

-- Useful indexes for lookups
CREATE INDEX idx_reviews_bike ON reviews (bike_numerical_id);
CREATE INDEX idx_reviews_poster ON reviews (poster_id);
CREATE INDEX idx_review_ratings_subcategory ON review_ratings (subcategory);
CREATE INDEX idx_reviews_bike_created ON reviews (bike_numerical_id, created_ts);
