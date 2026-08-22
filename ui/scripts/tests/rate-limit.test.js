'use strict';

const { describe, it, eq } = require('./helpers');
const { loadModule } = require('../load-module');

const { formatRetryAfter } = loadModule('src/utils/rate-limit.js');

describe('formatRetryAfter', () => {
    it('returns null for a missing header', () => {
        eq(formatRetryAfter(null), null);
        eq(formatRetryAfter(undefined), null);
    });

    it('returns null for a non-numeric header', () => {
        eq(formatRetryAfter('abc'), null);
        eq(formatRetryAfter(''), null);
    });

    it('converts seconds to ceiling minutes', () => {
        eq(formatRetryAfter('60'), 'Rate limited. Please try again in 1 minute(s).');
        eq(formatRetryAfter('61'), 'Rate limited. Please try again in 2 minute(s).');
        eq(formatRetryAfter('120'), 'Rate limited. Please try again in 2 minute(s).');
        eq(formatRetryAfter('600'), 'Rate limited. Please try again in 10 minute(s).');
    });

    it('uses a custom base error message', () => {
        eq(
            formatRetryAfter('120', 'Too many reviews'),
            'Too many reviews. Please try again in 2 minute(s).'
        );
    });

    it('uses the default base error when none provided', () => {
        eq(formatRetryAfter('30'), 'Rate limited. Please try again in 1 minute(s).');
    });

    it('handles numeric 0 seconds', () => {
        eq(formatRetryAfter('0'), 'Rate limited. Please try again in 0 minute(s).');
    });
});

require('./helpers').finish();
