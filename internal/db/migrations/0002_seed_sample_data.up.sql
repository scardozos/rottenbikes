-- 0002_seed_sample_data.up.sql

-- Sample posters
INSERT INTO posters (email, username)
VALUES
    ('alice@example.com', 'alice'),
    ('bob@example.com',   'bob'),
    ('carol@example.com', 'carol');

-- Sample bikes (creator_id references posters.poster_id)
INSERT INTO bikes (hash_id, is_electric, creator_id)
VALUES
    ('bike_1_hash', FALSE,
        (SELECT poster_id FROM posters WHERE username = 'alice')),
    ('bike_2_hash', TRUE,
        (SELECT poster_id FROM posters WHERE username = 'alice')),
    ('bike_3_hash', FALSE,
        (SELECT poster_id FROM posters WHERE username = 'alice'));

-- Sample reviews
-- Alice reviews bike 1
INSERT INTO reviews (poster_id, bike_numerical_id, bike_img, comment)
VALUES (
    (SELECT poster_id FROM posters WHERE username = 'alice'),
    (SELECT numerical_id FROM bikes WHERE hash_id = 'bike_1_hash'),
    'https://example.com/bike1_alice.jpg',
    'Solid bike, comfy seat'
);

-- Bob reviews bike 1
INSERT INTO reviews (poster_id, bike_numerical_id, bike_img, comment)
VALUES (
    (SELECT poster_id FROM posters WHERE username = 'bob'),
    (SELECT numerical_id FROM bikes WHERE hash_id = 'bike_1_hash'),
    'https://example.com/bike1_bob.jpg',
    'Good power but weak breaks'
);

-- Carol reviews bike 2
INSERT INTO reviews (poster_id, bike_numerical_id, bike_img, comment)
VALUES (
    (SELECT poster_id FROM posters WHERE username = 'carol'),
    (SELECT numerical_id FROM bikes WHERE hash_id = 'bike_2_hash'),
    'https://example.com/bike2_carol.jpg',
    'Great for commuting'
);

-- Sample review_ratings (per subcategory)
-- Alice on bike 1
INSERT INTO review_ratings (review_id, subcategory, score)
VALUES
    ((SELECT review_id FROM reviews r JOIN posters p ON r.poster_id = p.poster_id
      WHERE p.username = 'alice'
        AND r.bike_numerical_id = (SELECT numerical_id FROM bikes WHERE hash_id = 'bike_1_hash')),
     'overall', 4),
    ((SELECT review_id FROM reviews r JOIN posters p ON r.poster_id = p.poster_id
      WHERE p.username = 'alice'
        AND r.bike_numerical_id = (SELECT numerical_id FROM bikes WHERE hash_id = 'bike_1_hash')),
     'seat', 5),
    ((SELECT review_id FROM reviews r JOIN posters p ON r.poster_id = p.poster_id
      WHERE p.username = 'alice'
        AND r.bike_numerical_id = (SELECT numerical_id FROM bikes WHERE hash_id = 'bike_1_hash')),
     'sturdiness', 4);

-- Bob on bike 1
INSERT INTO review_ratings (review_id, subcategory, score)
VALUES
    ((SELECT review_id FROM reviews r JOIN posters p ON r.poster_id = p.poster_id
      WHERE p.username = 'bob'
        AND r.bike_numerical_id = (SELECT numerical_id FROM bikes WHERE hash_id = 'bike_1_hash')),
     'overall', 3),
    ((SELECT review_id FROM reviews r JOIN posters p ON r.poster_id = p.poster_id
      WHERE p.username = 'bob'
        AND r.bike_numerical_id = (SELECT numerical_id FROM bikes WHERE hash_id = 'bike_1_hash')),
     'breaks', 2),
    ((SELECT review_id FROM reviews r JOIN posters p ON r.poster_id = p.poster_id
      WHERE p.username = 'bob'
        AND r.bike_numerical_id = (SELECT numerical_id FROM bikes WHERE hash_id = 'bike_1_hash')),
     'power', 4);

-- Carol on bike 2
INSERT INTO review_ratings (review_id, subcategory, score)
VALUES
    ((SELECT review_id FROM reviews r JOIN posters p ON r.poster_id = p.poster_id
      WHERE p.username = 'carol'
        AND r.bike_numerical_id = (SELECT numerical_id FROM bikes WHERE hash_id = 'bike_2_hash')),
     'overall', 5),
    ((SELECT review_id FROM reviews r JOIN posters p ON r.poster_id = p.poster_id
      WHERE p.username = 'carol'
        AND r.bike_numerical_id = (SELECT numerical_id FROM bikes WHERE hash_id = 'bike_2_hash')),
     'seat', 4),
    ((SELECT review_id FROM reviews r JOIN posters p ON r.poster_id = p.poster_id
      WHERE p.username = 'carol'
        AND r.bike_numerical_id = (SELECT numerical_id FROM bikes WHERE hash_id = 'bike_2_hash')),
     'power', 5);

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
GROUP BY r.bike_numerical_id, rr.subcategory;
