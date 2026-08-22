'use strict';

const { describe, it, eq } = require('./helpers');
const { loadModule } = require('../load-module');

const { sortReviews, getPreviewReviews, uniqueBy } = loadModule('src/utils/reviews.js');

const reviews = [
    { review_id: 1, created_at: '2025-01-03T10:00:00Z', ratings: { overall: 4 } },
    { review_id: 2, created_at: '2025-01-01T10:00:00Z', ratings: { overall: 5 } },
    { review_id: 3, created_at: '2025-01-02T10:00:00Z', ratings: { overall: 2 } },
    { review_id: 4, created_at: '2025-01-04T10:00:00Z', ratings: {} },
];

describe('sortReviews', () => {
    it('does not mutate the original array', () => {
        const original = [...reviews];
        sortReviews(reviews, 'date', 'desc');
        eq(reviews, original);
    });

    it('sorts by date descending (newest first) by default', () => {
        const sorted = sortReviews(reviews);
        eq(sorted.map((r) => r.review_id), [4, 1, 3, 2]);
    });

    it('sorts by date ascending (oldest first)', () => {
        const sorted = sortReviews(reviews, 'date', 'asc');
        eq(sorted.map((r) => r.review_id), [2, 3, 1, 4]);
    });

    it('sorts by rating descending (highest first)', () => {
        const sorted = sortReviews(reviews, 'rating', 'desc');
        eq(sorted.map((r) => r.review_id), [2, 1, 3, 4]);
    });

    it('sorts by rating ascending (lowest first)', () => {
        const sorted = sortReviews(reviews, 'rating', 'asc');
        eq(sorted.map((r) => r.review_id), [4, 3, 1, 2]);
    });

    it('treats missing overall rating as 0', () => {
        const sorted = sortReviews(reviews, 'rating', 'asc');
        eq(sorted[0].review_id, 4); // review 4 has no overall -> 0
    });

    it('returns empty array for empty input', () => {
        eq(sortReviews([], 'date', 'desc'), []);
    });
});

describe('getPreviewReviews', () => {
    it('returns the 3 most recent reviews', () => {
        const preview = getPreviewReviews(reviews, 3);
        eq(preview.map((r) => r.review_id), [4, 1, 3]);
    });

    it('does not mutate the original array', () => {
        const original = [...reviews];
        getPreviewReviews(reviews, 3);
        eq(reviews, original);
    });

    it('returns fewer when input is smaller than limit', () => {
        const small = reviews.slice(0, 2);
        eq(getPreviewReviews(small, 3).length, 2);
    });

    it('returns empty array for empty input', () => {
        eq(getPreviewReviews([], 3), []);
    });

    it('respects a custom limit', () => {
        eq(getPreviewReviews(reviews, 2).map((r) => r.review_id), [4, 1]);
    });
});

describe('uniqueBy', () => {
    it('removes duplicates keeping the first occurrence', () => {
        const items = [
            { numerical_id: 1, name: 'a' },
            { numerical_id: 2, name: 'b' },
            { numerical_id: 1, name: 'c' },
            { numerical_id: 3, name: 'd' },
            { numerical_id: 2, name: 'e' },
        ];
        const result = uniqueBy(items, (i) => i.numerical_id);
        eq(result, [
            { numerical_id: 1, name: 'a' },
            { numerical_id: 2, name: 'b' },
            { numerical_id: 3, name: 'd' },
        ]);
    });

    it('returns empty array for empty input', () => {
        eq(uniqueBy([], (i) => i), []);
    });

    it('handles all-unique input', () => {
        const items = [{ id: 1 }, { id: 2 }, { id: 3 }];
        eq(uniqueBy(items, (i) => i.id), items);
    });

    it('handles string keys', () => {
        const items = [{ k: 'a' }, { k: 'b' }, { k: 'a' }];
        eq(uniqueBy(items, (i) => i.k), [{ k: 'a' }, { k: 'b' }]);
    });
});

require('./helpers').finish();
