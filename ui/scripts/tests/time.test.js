'use strict';

const { describe, it, eq, finish } = require('./helpers');
const { loadModule } = require('../load-module');

const { getRelativeTime } = loadModule('src/utils/time.js');

// A fake translator that returns a readable label for each key.
const t = (key, params = {}) => {
    const labels = {
        just_now: 'just now',
        m_ago: '{minutes}m ago',
        h_ago: '{hours}h ago',
        d_ago: '{days}d ago',
        w_ago: '{weeks}w ago',
        mo_ago: '{months}mo ago',
        y_ago: '{years}y ago',
    };
    let text = labels[key] || key;
    Object.keys(params).forEach((p) => {
        text = text.replace(`{${p}}`, params[p]);
    });
    return text;
};

// Build a date string `ms` milliseconds in the past.
const ago = (ms) => new Date(Date.now() - ms).toISOString();

describe('getRelativeTime', () => {
    it('returns empty string for null/undefined input', () => {
        eq(getRelativeTime(null, t), '');
        eq(getRelativeTime(undefined, t), '');
        eq(getRelativeTime('', t), '');
    });

    it('returns "just now" for less than 60 seconds', () => {
        eq(getRelativeTime(ago(0), t), 'just now');
        eq(getRelativeTime(ago(30 * 1000), t), 'just now');
        eq(getRelativeTime(ago(59 * 1000), t), 'just now');
    });

    it('returns minutes for less than 60 minutes', () => {
        eq(getRelativeTime(ago(60 * 1000), t), '1m ago');
        eq(getRelativeTime(ago(5 * 60 * 1000), t), '5m ago');
        eq(getRelativeTime(ago(59 * 60 * 1000), t), '59m ago');
    });

    it('returns hours for less than 24 hours', () => {
        eq(getRelativeTime(ago(60 * 60 * 1000), t), '1h ago');
        eq(getRelativeTime(ago(3 * 60 * 60 * 1000), t), '3h ago');
        eq(getRelativeTime(ago(23 * 60 * 60 * 1000), t), '23h ago');
    });

    it('returns days for less than 7 days', () => {
        eq(getRelativeTime(ago(24 * 60 * 60 * 1000), t), '1d ago');
        eq(getRelativeTime(ago(6 * 24 * 60 * 60 * 1000), t), '6d ago');
    });

    it('returns weeks for less than 4 weeks', () => {
        eq(getRelativeTime(ago(7 * 24 * 60 * 60 * 1000), t), '1w ago');
        eq(getRelativeTime(ago(3 * 7 * 24 * 60 * 60 * 1000), t), '3w ago');
    });

    it('returns months for less than 12 months', () => {
        eq(getRelativeTime(ago(30 * 24 * 60 * 60 * 1000), t), '1mo ago');
        eq(getRelativeTime(ago(11 * 30 * 24 * 60 * 60 * 1000), t), '11mo ago');
    });

    it('returns years for 365+ days', () => {
        eq(getRelativeTime(ago(365 * 24 * 60 * 60 * 1000), t), '1y ago');
        eq(getRelativeTime(ago(2 * 365 * 24 * 60 * 60 * 1000), t), '2y ago');
    });

    it('passes interpolation params to the translator', () => {
        const captured = {};
        const capturingT = (key, params = {}) => {
            captured.key = key;
            captured.params = params;
            return 'x';
        };
        getRelativeTime(ago(5 * 60 * 1000), capturingT);
        eq(captured.key, 'm_ago');
        eq(captured.params.minutes, 5);
    });
});

finish();
