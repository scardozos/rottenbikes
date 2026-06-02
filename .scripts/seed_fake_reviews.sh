#!/bin/bash

# Usage: .scripts/seed_fake_reviews.sh <env_file>

ENV_FILE="$1"

if [ -z "$ENV_FILE" ]; then
  echo "Usage: $0 <env_file>"
  exit 1
fi

if [ ! -f "$ENV_FILE" ]; then
  echo "Error: Environment file '$ENV_FILE' not found."
  exit 1
fi

echo "Loading config from $ENV_FILE"
export $(grep -v '^#' "$ENV_FILE" | xargs)

# Construct DATABASE_URL if not set
if [ -z "$DATABASE_URL" ]; then
  DB_USER="${DB_USER:-rottenbikes}"
  DB_PASSWORD="${DB_PASSWORD:-rottenbikes}"
  DB_HOST="${DB_HOST:-localhost}"
  DB_PORT="${DB_PORT:-5432}"
  DB_NAME="${DB_NAME:-rottenbikes}"
  DB_SSLMODE="${DB_SSLMODE:-disable}"
  
  DATABASE_URL="postgres://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME?sslmode=$DB_SSLMODE"
fi

echo "Seeding 1-10 fake reviews for each bike in database..."

psql "$DATABASE_URL" <<'EOF'
DO $$
DECLARE
    bike_rec RECORD;
    poster_rec RECORD;
    num_reviews INT;
    i INT;
    random_score INT;
    random_comment TEXT;
    new_review_id INT;
    comments TEXT[] := ARRAY[
        'Great ride, very smooth!',
        'Seat is a bit uncomfortable but mechanical parts are good.',
        'Brakes are squeaking, needs maintenance.',
        'Electric motor works perfectly. Highly recommended.',
        'Sturdy bike, handled potholes well.',
        'Pedals feel loose, overall average.',
        'Awesome bike, zero issues!',
        'Good value, simple and reliable.',
        'Handlebars are misaligned.',
        'Comfortable seat and solid frame.'
    ];
    subcategories TEXT[] := ARRAY['overall', 'breaks', 'seat', 'sturdiness', 'power', 'pedals'];
    subcat TEXT;
BEGIN
    -- Clear existing reviews and review ratings
    DELETE FROM reviews;

    FOR bike_rec IN SELECT numerical_id FROM bikes LOOP
        -- Generate random number of reviews between 1 and 10
        num_reviews := floor(random() * 10) + 1;
        
        FOR i IN 1..num_reviews LOOP
            -- Select a random poster_id from posters table
            SELECT poster_id INTO poster_rec FROM posters ORDER BY random() LIMIT 1;
            
            -- Pick a random comment
            random_comment := comments[floor(random() * array_length(comments, 1)) + 1];
            
            -- Insert the review
            INSERT INTO reviews (poster_id, bike_numerical_id, bike_img, comment, created_ts)
            VALUES (
                poster_rec.poster_id, 
                bike_rec.numerical_id, 
                NULL, 
                random_comment, 
                NOW() - (random() * INTERVAL '30 days')
            )
            RETURNING review_id INTO new_review_id;
            
            -- Insert ratings for all subcategories
            FOREACH subcat IN ARRAY subcategories LOOP
                random_score := floor(random() * 5) + 1;
                INSERT INTO review_ratings (review_id, subcategory, score)
                VALUES (new_review_id, subcat::rating_subcategory, random_score);
            END LOOP;
        END LOOP;
    END LOOP;
END;
$$;

-- Clean and recompute rating aggregates for all bikes
DELETE FROM rating_aggregates;
INSERT INTO rating_aggregates (
	bike_numerical_id, subcategory, rating_sum, rating_count, average_rating
)
SELECT
	r.bike_numerical_id,
	rr.subcategory,
	SUM(rr.score)                        AS rating_sum,
	COUNT(*)                             AS rating_count,
	ROUND(AVG(rr.score)::numeric, 2)     AS average_rating
FROM review_ratings rr
JOIN reviews r ON rr.review_id = r.review_id
GROUP BY r.bike_numerical_id, rr.subcategory;
EOF

if [ $? -eq 0 ]; then
  echo "Successfully seeded fake reviews and recomputed aggregates."
else
  echo "Failed to seed fake reviews."
  exit 1
fi
