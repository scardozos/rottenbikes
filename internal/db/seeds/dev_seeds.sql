-- Sample posters
INSERT INTO posters (email, username)
VALUES
    ('alice@example.com', 'alice'),
    ('bob@example.com',   'bob'),
    ('carol@example.com', 'carol')
ON CONFLICT DO NOTHING;

-- Sample bikes
-- NumericalID must be 4-5 digits (1000-99999) but stored as text
INSERT INTO bikes (numerical_id, hash_id, is_electric, creator_id)
VALUES
    ('1001', 'bike1hash', FALSE, (SELECT poster_id FROM posters WHERE username = 'alice' LIMIT 1)),
    ('1002', 'bike2hash', TRUE,  (SELECT poster_id FROM posters WHERE username = 'alice' LIMIT 1)),
    ('1003', 'bike3hash', FALSE, (SELECT poster_id FROM posters WHERE username = 'alice' LIMIT 1))
ON CONFLICT (numerical_id) DO NOTHING;

-- Reset sequence to avoid conflicts with future inserts
-- We must cast numerical_id to integer for MAX and for setval

-- Sample reviews
-- Alice reviews bike 1
INSERT INTO reviews (poster_id, bike_numerical_id, bike_img, comment)
SELECT
    (SELECT poster_id FROM posters WHERE username = 'alice' LIMIT 1),
    '1001',
    'https://example.com/bike1_alice.jpg',
    'Solid bike, comfy seat'
WHERE NOT EXISTS (
    SELECT 1 FROM reviews
    WHERE poster_id = (SELECT poster_id FROM posters WHERE username = 'alice' LIMIT 1)
      AND bike_numerical_id = '1001'
);

-- Bob reviews bike 1
INSERT INTO reviews (poster_id, bike_numerical_id, bike_img, comment)
SELECT
    (SELECT poster_id FROM posters WHERE username = 'bob' LIMIT 1),
    '1001',
    'https://example.com/bike1_bob.jpg',
    'Good power but weak breaks'
WHERE NOT EXISTS (
    SELECT 1 FROM reviews
    WHERE poster_id = (SELECT poster_id FROM posters WHERE username = 'bob' LIMIT 1)
      AND bike_numerical_id = '1001'
);

-- Carol reviews bike 2
INSERT INTO reviews (poster_id, bike_numerical_id, bike_img, comment)
SELECT
    (SELECT poster_id FROM posters WHERE username = 'carol' LIMIT 1),
    '1002',
    'https://example.com/bike2_carol.jpg',
    'Great for commuting'
WHERE NOT EXISTS (
    SELECT 1 FROM reviews
    WHERE poster_id = (SELECT poster_id FROM posters WHERE username = 'carol' LIMIT 1)
      AND bike_numerical_id = '1002'
);

-- Sample review_ratings (per subcategory)
INSERT INTO review_ratings (review_id, subcategory, score)
SELECT r.review_id, 'overall', 4
FROM reviews r
JOIN posters p ON r.poster_id = p.poster_id
WHERE p.username = 'alice' AND r.bike_numerical_id = '1001'
ON CONFLICT (review_id, subcategory) DO NOTHING;

INSERT INTO review_ratings (review_id, subcategory, score)
SELECT r.review_id, 'seat', 5
FROM reviews r
JOIN posters p ON r.poster_id = p.poster_id
WHERE p.username = 'alice' AND r.bike_numerical_id = '1001'
ON CONFLICT (review_id, subcategory) DO NOTHING;

INSERT INTO review_ratings (review_id, subcategory, score)
SELECT r.review_id, 'sturdiness', 4
FROM reviews r
JOIN posters p ON r.poster_id = p.poster_id
WHERE p.username = 'alice' AND r.bike_numerical_id = '1001'
ON CONFLICT (review_id, subcategory) DO NOTHING;

INSERT INTO review_ratings (review_id, subcategory, score)
SELECT r.review_id, 'overall', 3
FROM reviews r
JOIN posters p ON r.poster_id = p.poster_id
WHERE p.username = 'bob' AND r.bike_numerical_id = '1001'
ON CONFLICT (review_id, subcategory) DO NOTHING;

INSERT INTO review_ratings (review_id, subcategory, score)
SELECT r.review_id, 'breaks', 2
FROM reviews r
JOIN posters p ON r.poster_id = p.poster_id
WHERE p.username = 'bob' AND r.bike_numerical_id = '1001'
ON CONFLICT (review_id, subcategory) DO NOTHING;

INSERT INTO review_ratings (review_id, subcategory, score)
SELECT r.review_id, 'power', 4
FROM reviews r
JOIN posters p ON r.poster_id = p.poster_id
WHERE p.username = 'bob' AND r.bike_numerical_id = '1001'
ON CONFLICT (review_id, subcategory) DO NOTHING;

INSERT INTO review_ratings (review_id, subcategory, score)
SELECT r.review_id, 'overall', 5
FROM reviews r
JOIN posters p ON r.poster_id = p.poster_id
WHERE p.username = 'carol' AND r.bike_numerical_id = '1002'
ON CONFLICT (review_id, subcategory) DO NOTHING;

INSERT INTO review_ratings (review_id, subcategory, score)
SELECT r.review_id, 'seat', 4
FROM reviews r
JOIN posters p ON r.poster_id = p.poster_id
WHERE p.username = 'carol' AND r.bike_numerical_id = '1002'
ON CONFLICT (review_id, subcategory) DO NOTHING;

INSERT INTO review_ratings (review_id, subcategory, score)
SELECT r.review_id, 'power', 5
FROM reviews r
JOIN posters p ON r.poster_id = p.poster_id
WHERE p.username = 'carol' AND r.bike_numerical_id = '1002'
ON CONFLICT (review_id, subcategory) DO NOTHING;


-- Seed rating_aggregates from review_ratings
INSERT INTO rating_aggregates (bike_numerical_id, subcategory, rating_sum, rating_count, average_rating)
SELECT
    r.bike_numerical_id,
    rr.subcategory,
    SUM(rr.score)                    AS rating_sum,
    COUNT(*)                         AS rating_count,
    ROUND(AVG(rr.score)::numeric, 2) AS average_rating
FROM review_ratings rr
JOIN reviews r ON rr.review_id = r.review_id
GROUP BY r.bike_numerical_id, rr.subcategory
ON CONFLICT (bike_numerical_id, subcategory) DO UPDATE SET
    rating_sum = EXCLUDED.rating_sum,
    rating_count = EXCLUDED.rating_count,
    average_rating = EXCLUDED.average_rating;
