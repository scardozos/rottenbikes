'use strict';

const { describe, it, eq, ok, finish } = require('./helpers');
const { loadModule } = require('../load-module');

const { isLogoutRequest, isSessionExpiredError, singleFlight } = loadModule('src/utils/session.js');

const httpError = (status, url) => ({ response: { status }, config: { url } });

(async () => {
    describe('isLogoutRequest', () => {
        it('matches relative, absolute, trailing-slash and query forms', () => {
            ok(isLogoutRequest({ url: '/auth/logout' }));
            ok(isLogoutRequest({ url: 'https://api.rottenbik.es/auth/logout' }));
            ok(isLogoutRequest({ url: '/auth/logout/' }));
            ok(isLogoutRequest({ url: '/auth/logout?x=1' }));
        });

        it('does not match other endpoints or missing configs', () => {
            eq(isLogoutRequest({ url: '/auth/verify' }), false);
            eq(isLogoutRequest({ url: '/auth/logout-all' }), false);
            eq(isLogoutRequest({}), false);
            eq(isLogoutRequest(undefined), false);
        });
    });

    describe('isSessionExpiredError', () => {
        it('treats a 401 from a regular endpoint as an expired session', () => {
            ok(isSessionExpiredError(httpError(401, '/auth/verify')));
            ok(isSessionExpiredError(httpError(401, '/bikes/1234/reviews')));
        });

        it('ignores a 401 from logout (otherwise logout -> 401 -> logout loops)', () => {
            eq(isSessionExpiredError(httpError(401, '/auth/logout')), false);
        });

        it('ignores other statuses and network errors', () => {
            eq(isSessionExpiredError(httpError(403, '/admin/users')), false);
            eq(isSessionExpiredError(httpError(500, '/auth/verify')), false);
            eq(isSessionExpiredError({ message: 'Network Error' }), false);
            eq(isSessionExpiredError(undefined), false);
        });
    });

    // describe() runs its body synchronously, so the async cases are awaited
    // one by one here to make their failures count before finish().
    describe('singleFlight', () => {});
    {
        await it('shares the in-progress call', async () => {
            let calls = 0;
            let release;
            const fn = singleFlight(() => {
                calls++;
                return new Promise((resolve) => { release = resolve; });
            });
            const a = fn();
            const b = fn();
            await Promise.resolve();
            eq(calls, 1);
            ok(a === b, 'concurrent callers get the same promise');
            release('done');
            eq(await a, 'done');
        });

        await it('starts a new call once the previous one settled, even after a failure', async () => {
            let calls = 0;
            const fn = singleFlight(async () => {
                calls++;
                if (calls === 1) throw new Error('boom');
                return calls;
            });
            await fn().catch(() => {});
            eq(await fn(), 2);
        });

        await it('stops a logout that re-triggers itself from looping', async () => {
            // Mirrors the old bug: logout's request fails with 401, the 401
            // handler calls logout again before the first one finished.
            let requests = 0;
            let logout;
            logout = singleFlight(async () => {
                requests++;
                if (requests > 50) throw new Error('logout loop');
                await Promise.resolve();
                logout(); // re-entrant call from the "session expired" handler
            });
            await logout();
            eq(requests, 1);
        });
    }

    finish();
})();
