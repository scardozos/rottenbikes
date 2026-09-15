// Rating display helpers: border colors and trend computation.

export const TREND_THRESHOLD = 0.2;

// Semantic rating colors, aligned with the success/warning/danger tokens in ThemeContext.
export const RATING_COLORS = {
    dark: { good: '#34D399', warn: '#FBBF24', bad: '#F87171' },
    light: { good: '#047857', warn: '#D97706', bad: '#DC2626' },
};
export const NEUTRAL_COLOR = 'transparent';

// Returns a border color for a given aggregate rating.
export const getBorderColor = (rating, isDark = false) => {
    const colors = RATING_COLORS[isDark ? 'dark' : 'light'];
    if (rating == null) return NEUTRAL_COLOR;
    if (rating >= 4) return colors.good;
    if (rating >= 3) return colors.warn;
    return colors.bad;
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
