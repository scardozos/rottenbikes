import { useState, useEffect } from 'react';
import { Platform } from 'react-native';

/**
 * Hook to poll for magic-link confirmation status.
 *
 * @param {boolean} isWaiting - Whether the user is on the step awaiting confirmation.
 * @param {string|null} pendingMagicToken - Token returned when magic link was requested.
 * @param {Function} checkLoginStatus - Async function from AuthContext to check login status.
 * @param {number} [maxAttempts=24] - Max polling attempts (default 24 * 5s = 2 minutes).
 * @param {number} [intervalMs=5000] - Interval between checks in ms (default 5000ms).
 */
export const useMagicLinkPolling = (
    isWaiting,
    pendingMagicToken,
    checkLoginStatus,
    maxAttempts = 24,
    intervalMs = 5000
) => {
    const [pollingTimeout, setPollingTimeout] = useState(false);

    useEffect(() => {
        let interval;
        let attempts = 0;

        if (isWaiting && pendingMagicToken && Platform.OS !== 'web') {
            setPollingTimeout(false);
            interval = setInterval(async () => {
                attempts++;
                if (attempts >= maxAttempts) {
                    clearInterval(interval);
                    setPollingTimeout(true);
                    return;
                }
                try {
                    const confirmed = await checkLoginStatus(pendingMagicToken);
                    if (confirmed) {
                        clearInterval(interval);
                    }
                } catch (err) {
                    // Suppress or handle errors during background polling
                    console.warn('[useMagicLinkPolling] Polling error:', err);
                }
            }, intervalMs);
        }

        return () => {
            if (interval) clearInterval(interval);
        };
    }, [isWaiting, pendingMagicToken, checkLoginStatus, maxAttempts, intervalMs]);

    return {
        pollingTimeout,
        setPollingTimeout,
        resetPolling: () => setPollingTimeout(false),
    };
};

export default useMagicLinkPolling;
