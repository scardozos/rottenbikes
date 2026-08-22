'use strict';

const { describe, it, eq, ok } = require('./helpers');
const { loadModule } = require('../load-module');

const v = loadModule('src/utils/validation.js');

describe('isValidUsername', () => {
    it('accepts letters and numbers', () => {
        ok(v.isValidUsername('alice123'));
        ok(v.isValidUsername('Bob'));
        ok(v.isValidUsername('x'));
    });

    it('accepts dots', () => {
        ok(v.isValidUsername('john.doe'));
        ok(v.isValidUsername('a.b.c'));
    });

    it('rejects spaces and special characters', () => {
        eq(v.isValidUsername('john doe'), false);
        eq(v.isValidUsername('john_doe'), false);
        eq(v.isValidUsername('john@doe'), false);
        eq(v.isValidUsername('john-doe'), false);
    });

    it('rejects empty string', () => {
        eq(v.isValidUsername(''), false);
    });
});

describe('isValidEmail', () => {
    it('accepts well-formed emails', () => {
        ok(v.isValidEmail('user@example.com'));
        ok(v.isValidEmail('a.b@c.d.e'));
        ok(v.isValidEmail('x@y.io'));
    });

    it('rejects missing @', () => {
        eq(v.isValidEmail('userexample.com'), false);
    });

    it('rejects missing domain', () => {
        eq(v.isValidEmail('user@'), false);
        eq(v.isValidEmail('user@example'), false);
    });

    it('rejects spaces', () => {
        eq(v.isValidEmail('user @example.com'), false);
        eq(v.isValidEmail('user@ example.com'), false);
    });

    it('rejects empty string', () => {
        eq(v.isValidEmail(''), false);
    });
});

describe('isValidNumericalId', () => {
    it('accepts 4-digit ids', () => {
        ok(v.isValidNumericalId('1234'));
        ok(v.isValidNumericalId('9999'));
    });

    it('accepts 5-digit ids', () => {
        ok(v.isValidNumericalId('12345'));
        ok(v.isValidNumericalId('99999'));
    });

    it('rejects fewer than 4 digits', () => {
        eq(v.isValidNumericalId('1'), false);
        eq(v.isValidNumericalId('12'), false);
        eq(v.isValidNumericalId('123'), false);
    });

    it('rejects more than 5 digits', () => {
        eq(v.isValidNumericalId('123456'), false);
        eq(v.isValidNumericalId('1234567'), false);
    });

    it('rejects non-numeric characters', () => {
        eq(v.isValidNumericalId('12a4'), false);
        eq(v.isValidNumericalId('abcd'), false);
    });

    it('trims surrounding whitespace', () => {
        ok(v.isValidNumericalId('  1234  '));
    });
});

describe('isValidHashId', () => {
    it('accepts empty string (optional field)', () => {
        ok(v.isValidHashId(''));
        ok(v.isValidHashId('   '));
    });

    it('accepts alphanumeric values', () => {
        ok(v.isValidHashId('frame123'));
        ok(v.isValidHashId('ABC'));
        ok(v.isValidHashId('a1b2c3'));
    });

    it('rejects special characters', () => {
        eq(v.isValidHashId('frame-123'), false);
        eq(v.isValidHashId('frame_123'), false);
        eq(v.isValidHashId('frame 123'), false);
        eq(v.isValidHashId('frame!'), false);
    });
});

describe('isNumeric', () => {
    it('accepts digit-only strings', () => {
        ok(v.isNumeric('0'));
        ok(v.isNumeric('123'));
        ok(v.isNumeric('9999999999'));
    });

    it('rejects empty string', () => {
        eq(v.isNumeric(''), false);
    });

    it('rejects non-digit characters', () => {
        eq(v.isNumeric('12a'), false);
        eq(v.isNumeric(' 12'), false);
        eq(v.isNumeric('12 '), false);
        eq(v.isNumeric('-12'), false);
        eq(v.isNumeric('1.2'), false);
    });
});

require('./helpers').finish();
