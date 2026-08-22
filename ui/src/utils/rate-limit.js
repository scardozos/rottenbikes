// Rate-limit response helpers.

// Formats a Retry-After header value into a human readable message.
// Returns null when the header is missing or not a number of seconds.
export const formatRetryAfter = (retryAfterHeader, baseError = 'Rate limited') => {
    if (retryAfterHeader == null) return null;
    const seconds = parseInt(retryAfterHeader, 10);
    if (isNaN(seconds)) return null;
    const minutes = Math.ceil(seconds / 60);
    return `${baseError}. Please try again in ${minutes} minute(s).`;
};
