// Review sorting/preview helpers.

// Sorts a copy of reviews by either 'date' or 'rating', ascending or descending.
export const sortReviews = (reviews, sortBy = 'date', sortOrder = 'desc') => {
    return [...reviews].sort((a, b) => {
        let diff = 0;
        if (sortBy === 'date') {
            diff = new Date(a.created_at) - new Date(b.created_at);
        } else {
            diff = (a.ratings?.overall || 0) - (b.ratings?.overall || 0);
        }
        return sortOrder === 'asc' ? diff : -diff;
    });
};

// Returns the most recent N reviews (newest first).
export const getPreviewReviews = (reviews, limit = 3) => {
    return [...reviews]
        .sort((a, b) => new Date(b.created_at) - new Date(a.created_at))
        .slice(0, limit);
};

// Deduplicates an array by a key selector, keeping the first occurrence.
export const uniqueBy = (items, keyFn) => {
    const seen = new Set();
    return items.filter((item) => {
        const key = keyFn(item);
        if (seen.has(key)) return false;
        seen.add(key);
        return true;
    });
};
