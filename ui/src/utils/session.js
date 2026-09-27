// Session helpers shared by the API client and AuthContext.

const LOGOUT_PATH = '/auth/logout';

// Whether a request config targets the logout endpoint (relative or absolute
// URL, with or without a query string).
export const isLogoutRequest = (config) => {
    const url = config && config.url;
    if (!url) return false;
    const path = url.split('?')[0].replace(/\/+$/, '');
    return path === LOGOUT_PATH || path.endsWith(LOGOUT_PATH);
};

// Whether a failed API call means the stored session is no longer valid.
// A failed logout never counts: it already ends in a local logout, and
// treating it as an expiry would trigger another logout (and so on).
export const isSessionExpiredError = (error) => {
    if (!error || !error.response || error.response.status !== 401) return false;
    return !isLogoutRequest(error.config);
};

// Wraps an async function so that calls made while one is still running
// share that call's promise instead of starting another one.
export const singleFlight = (fn) => {
    let inFlight = null;
    return (...args) => {
        if (inFlight) return inFlight;
        inFlight = Promise.resolve()
            .then(() => fn(...args))
            .finally(() => {
                inFlight = null;
            });
        return inFlight;
    };
};
