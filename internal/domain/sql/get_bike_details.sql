WITH bike_base AS (
    SELECT 
        b.numerical_id, b.hash_id, b.is_electric, b.created_ts, b.updated_ts,
        ra.average_rating,
        (SELECT COUNT(*) FROM reviews r WHERE r.bike_numerical_id = b.numerical_id) as total_reviews
    FROM bikes b
    LEFT JOIN rating_aggregates ra ON b.numerical_id = ra.bike_numerical_id AND ra.subcategory = 'overall'
    WHERE b.numerical_id = $1
),
reviews_list AS (
    SELECT 
        COALESCE(json_agg(json_build_object(
            'review_id', r.review_id,
            'poster_id', COALESCE(r.poster_id, 0),
            'poster_username', COALESCE(p.username, ''),
            'bike_numerical_id', r.bike_numerical_id,
            'comment', r.comment,
            'created_at', r.created_ts,
            'bike_img', r.bike_img,
            'ratings', COALESCE((
                SELECT json_object_agg(rr.subcategory, rr.score)
                FROM review_ratings rr
                WHERE rr.review_id = r.review_id
            ), '{}'::json)
        )), '[]'::json) as reviews
    FROM (
        SELECT * FROM reviews 
        WHERE bike_numerical_id = $1 
        ORDER BY review_id DESC 
        LIMIT (CASE WHEN $2::int >= 0 THEN $2::int ELSE NULL END)
        OFFSET (CASE WHEN $3::int >= 0 THEN $3::int ELSE 0 END)
    ) r
    LEFT JOIN posters p ON p.poster_id = r.poster_id
)
SELECT 
    b.numerical_id, b.hash_id, b.is_electric, b.created_ts, b.updated_ts, b.average_rating, b.total_reviews,
    r.reviews
FROM bike_base b
CROSS JOIN reviews_list r;
