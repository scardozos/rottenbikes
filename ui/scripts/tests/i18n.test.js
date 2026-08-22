'use strict';

const { describe, it, eq } = require('./helpers');
const { loadModule } = require('../load-module');

const { interpolate, translate } = loadModule('src/utils/i18n.js');

describe('interpolate', () => {
    it('replaces a single placeholder', () => {
        eq(interpolate('Hello {name}', { name: 'World' }), 'Hello World');
    });

    it('replaces multiple placeholders', () => {
        eq(interpolate('{a} and {b}', { a: '1', b: '2' }), '1 and 2');
    });

    it('leaves unmatched placeholders as-is', () => {
        eq(interpolate('Hi {name} from {place}', { name: 'Bob' }), 'Hi Bob from {place}');
    });

    it('returns text unchanged when no params given', () => {
        eq(interpolate('no placeholders'), 'no placeholders');
        eq(interpolate('no placeholders', {}), 'no placeholders');
    });

    it('handles numeric param values', () => {
        eq(interpolate('Bike #{id}', { id: 123 }), 'Bike #123');
    });

    it('handles text with no placeholders and extra params', () => {
        eq(interpolate('plain text', { x: 'y' }), 'plain text');
    });
});

describe('translate', () => {
    const dict = {
        hello: 'Hello {name}',
        bye: 'Goodbye',
    };

    it('resolves a key and interpolates params', () => {
        eq(translate(dict, 'hello', { name: 'Alice' }), 'Hello Alice');
    });

    it('resolves a key without params', () => {
        eq(translate(dict, 'bye'), 'Goodbye');
    });

    it('falls back to the key when missing from dictionary', () => {
        eq(translate(dict, 'missing_key'), 'missing_key');
    });

    it('falls back to the key and interpolates when missing', () => {
        eq(translate(dict, 'unknown', { x: '1' }), 'unknown');
    });

    it('works with an empty dictionary', () => {
        eq(translate({}, 'anything'), 'anything');
    });
});

require('./helpers').finish();
