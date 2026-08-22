'use strict';

const { describe, it, eq } = require('./helpers');
const { loadModule } = require('../load-module');

const r = loadModule('src/utils/ratings.js');

describe('getBorderColor', () => {
    it('returns transparent for null/undefined', () => {
        eq(r.getBorderColor(null), 'transparent');
        eq(r.getBorderColor(undefined), 'transparent');
    });

    it('returns green for ratings >= 4', () => {
        eq(r.getBorderColor(4), '#2ecc71');
        eq(r.getBorderColor(4.5), '#2ecc71');
        eq(r.getBorderColor(5), '#2ecc71');
    });

    it('returns yellow for ratings >= 3 and < 4', () => {
        eq(r.getBorderColor(3), '#f1c40f');
        eq(r.getBorderColor(3.5), '#f1c40f');
        eq(r.getBorderColor(3.99), '#f1c40f');
    });

    it('returns red for ratings < 3', () => {
        eq(r.getBorderColor(2.9), '#e74c3c');
        eq(r.getBorderColor(1), '#e74c3c');
        eq(r.getBorderColor(0), '#e74c3c');
    });

    it('treats 0 as red (not null)', () => {
        eq(r.getBorderColor(0), '#e74c3c');
    });
});

describe('computeTrend', () => {
    it('returns null when either window is missing', () => {
        eq(r.computeTrend(null, 3), null);
        eq(r.computeTrend(3, null), null);
        eq(r.computeTrend(undefined, undefined), null);
    });

    it('returns "improving" when w1 exceeds w2 by more than threshold', () => {
        eq(r.computeTrend(4, 3), 'improving');
        eq(r.computeTrend(4.3, 4), 'improving');
    });

    it('returns "degrading" when w1 is below w2 by more than threshold', () => {
        eq(r.computeTrend(3, 4), 'degrading');
        eq(r.computeTrend(3.7, 4), 'degrading');
    });

    it('returns "stable" when difference is within threshold', () => {
        eq(r.computeTrend(4, 4), 'stable');
        eq(r.computeTrend(4.1, 4), 'stable');
        eq(r.computeTrend(3.9, 4), 'stable');
        // Exactly at threshold (0.2) is NOT improving/degrading (uses strict inequality)
        eq(r.computeTrend(4.2, 4), 'stable');
        eq(r.computeTrend(3.8, 4), 'stable');
    });
});

require('./helpers').finish();
