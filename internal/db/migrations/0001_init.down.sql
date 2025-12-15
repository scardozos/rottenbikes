-- Drop rating_aggregates first (depends on bikes)
DROP TABLE IF EXISTS rating_aggregates;

-- Drop review_ratings (depends on reviews)
DROP TABLE IF EXISTS review_ratings;

-- Drop reviews (depends on bikes, posters)
DROP TABLE IF EXISTS reviews;

-- Drop posters
DROP TABLE IF EXISTS posters;

-- Drop bikes
DROP TABLE IF EXISTS bikes;

-- Drop enum type used by review_ratings and rating_aggregates
DROP TYPE IF EXISTS rating_subcategory;
