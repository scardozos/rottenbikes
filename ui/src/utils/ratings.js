// Rating display helpers: border colors and trend computation.

export const TREND_THRESHOLD = 0.2;

export const GOOD_COLOR = '#2ecc71'; // Green
export const WARN_COLOR = '#f1c40f'; // Yellow
export const BAD_COLOR = '#e74c3c'; // Red
export const NEUTRAL_COLOR = 'transparent';

// Returns a border color for a given aggregate rating.
export const getBorderColor = (rating) => {
    if (rating == null) return NEUTRAL_COLOR;
    if (rating >= 4) return GOOD_COLOR;
    if (rating >= 3) return WARN_COLOR;
    return BAD_COLOR;
};

// Computes a trend ('improving' | 'degrading' | 'stable') by comparing
// the most recent week (w1) against the previous two weeks (w2).
// Returns null when either window is missing.
export const computeTrend = (w1, w2) => {
    if (w1 == null || w2 == null) return null;
    if (w1 > w2 + TREND_THRESHOLD) return 'improving';
    if (w1 < w2 - TREND_THRESHOLD) return 'degrading';
    return 'stable';
};
