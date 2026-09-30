'use strict';

const { describe, it, eq, ok, finish } = require('./helpers');
const { loadModule } = require('../load-module');

const c = loadModule('src/utils/loginCode.js');

describe('normalizeLoginCode', () => {
    it('keeps a plain code', () => {
        eq(c.normalizeLoginCode('123456'), '123456');
    });

    it('drops spaces and dashes people type or paste', () => {
        eq(c.normalizeLoginCode(' 123 456 '), '123456');
        eq(c.normalizeLoginCode('123-456'), '123456');
    });

    it('keeps leading zeros', () => {
        eq(c.normalizeLoginCode('007 042'), '007042');
    });

    it('drops anything that is not a digit', () => {
        eq(c.normalizeLoginCode('12a3b4'), '1234');
        eq(c.normalizeLoginCode('１２３４５６'), '');
    });

    it('caps the code at 6 digits', () => {
        eq(c.normalizeLoginCode('1234567'), '123456');
    });

    it('handles empty input', () => {
        eq(c.normalizeLoginCode(''), '');
        eq(c.normalizeLoginCode(undefined), '');
        eq(c.normalizeLoginCode(null), '');
    });
});

describe('isCompleteLoginCode', () => {
    it('accepts exactly 6 digits', () => {
        ok(c.isCompleteLoginCode('123456'));
        ok(c.isCompleteLoginCode('000000'));
    });

    it('rejects anything else', () => {
        ok(!c.isCompleteLoginCode('12345'));
        ok(!c.isCompleteLoginCode('1234567'));
        ok(!c.isCompleteLoginCode('12 456'));
        ok(!c.isCompleteLoginCode(''));
        ok(!c.isCompleteLoginCode(undefined));
        ok(!c.isCompleteLoginCode(123456));
    });
});

finish();
