'use strict';

const { describe, it, eq, ok } = require('./helpers');
const { loadModule } = require('../load-module');

function makeStorage(platformOS, withLocalStorage) {
    const store = {};
    const mocks = {
        'expo-secure-store': {
            setItemAsync: async (k, v) => { store[k] = String(v); },
            getItemAsync: async (k) => (k in store ? store[k] : null),
            deleteItemAsync: async (k) => { delete store[k]; },
        },
        'react-native': { Platform: { OS: platformOS } },
    };
    const globals = {};
    if (withLocalStorage) {
        globals.localStorage = {
            setItem: (k, v) => { store[k] = String(v); },
            getItem: (k) => (k in store ? store[k] : null),
            removeItem: (k) => { delete store[k]; },
        };
    }
    return { storage: loadModule('src/utils/storage.js', { mocks, globals }), store };
}

describe('storage (web path)', () => {
    it('setItem/getItem round-trips via localStorage', async () => {
        const { storage } = makeStorage('web', true);
        await storage.default.setItem('token', 'abc123');
        eq(await storage.default.getItem('token'), 'abc123');
    });

    it('deleteItem removes the value', async () => {
        const { storage } = makeStorage('web', true);
        await storage.default.setItem('foo', 'bar');
        await storage.default.deleteItem('foo');
        eq(await storage.default.getItem('foo'), null);
    });

    it('getItem returns null for a missing key', async () => {
        const { storage } = makeStorage('web', true);
        eq(await storage.default.getItem('nope'), null);
    });

    it('does not throw when localStorage is undefined (silent failure)', async () => {
        const { storage } = makeStorage('web', false);
        await storage.default.setItem('x', 'y'); // should not throw
        eq(await storage.default.getItem('x'), null);
    });
});

describe('storage (native path)', () => {
    it('setItem/getItem round-trips via SecureStore', async () => {
        const { storage, store } = makeStorage('ios', false);
        await storage.default.setItem('token', 'secret');
        eq(store['token'], 'secret');
        eq(await storage.default.getItem('token'), 'secret');
    });

    it('deleteItem removes the value', async () => {
        const { storage } = makeStorage('android', false);
        await storage.default.setItem('foo', 'bar');
        await storage.default.deleteItem('foo');
        eq(await storage.default.getItem('foo'), null);
    });

    it('getItem returns null for a missing key', async () => {
        const { storage } = makeStorage('ios', false);
        eq(await storage.default.getItem('nope'), null);
    });
});

require('./helpers').finish();
