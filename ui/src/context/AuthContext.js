import React, { createContext, useState, useEffect, useContext, useRef, useMemo, useCallback } from 'react';
import { Platform, DeviceEventEmitter } from 'react-native';
import storage from '../utils/storage';
import api from '../services/api';
import { useToast } from './ToastContext';
import { LanguageContext } from './LanguageContext';
import { singleFlight } from '../utils/session';

export const AuthContext = createContext();

// Login emails for requests from the app contain only the 6-digit code: the
// emailed link would open the browser, not the app. Web requests get the link
// and the code. The API tells them apart by origin: 'mobile'.
export const emailsOnlyCode = Platform.OS !== 'web';

export const AuthProvider = ({ children }) => {
    const [isLoading, setIsLoading] = useState(true);
    const [userToken, setUserToken] = useState(null);
    const [userId, setUserId] = useState(null);
    const [username, setUsername] = useState(null);
    const [isAdmin, setIsAdmin] = useState(false);
    const [lastUsername, setLastUsername] = useState(null);
    const lastUsernameRef = useRef(null);

    useEffect(() => {
        if (username) {
            lastUsernameRef.current = username;
            setLastUsername(username);
        }
    }, [username]);


    const { showToast } = useToast();
    const { t } = useContext(LanguageContext);

    const fetchCurrentUser = useCallback(async () => {
        try {
            const res = await api.get('/auth/verify');
            if (res.data && res.data.poster_id) {
                __DEV__ && console.log('[AuthContext] Fetched current user:', res.data);
                setUserId(res.data.poster_id);
                setUsername(res.data.username);
                setIsAdmin(res.data.is_admin === true);
            }
        } catch (e) {
            console.log('[AuthContext] Failed to fetch current user:', e);
            if (e.response && e.response.status === 401) {
                // The global interceptor will catch this and emit session_expired
            }
        }
    }, []);

    const register = useCallback(async (username, email, captcha) => {
        try {
            __DEV__ && console.log(`Registering user: ${username} with email: ${email}`);
            const data = {
                username: username,
                email: email,
                captcha_token: captcha
            };
            if (emailsOnlyCode) {
                data.origin = 'mobile';
            }
            const response = await api.post('/auth/register', data);
            return response.data.magic_token;
        } catch (e) {
            console.log('register error', e);
            if (e.response && e.response.data && e.response.data.error) {
                throw new Error(e.response.data.error);
            }
            throw e;
        }
    }, []);

    const requestLogin = useCallback(async (identifier, captcha) => {
        try {
            __DEV__ && console.log(`Requesting magic link for identifier: ${identifier}`);
            const isEmail = identifier.includes('@');
            const data = {};
            if (isEmail) {
                data.email = identifier;
            } else {
                data.username = identifier;
            }
            if (emailsOnlyCode) {
                data.origin = 'mobile';
            }
            if (captcha) {
                data.captcha_token = captcha;
            }

            const response = await api.post('/auth/request-magic-link', data);
            // Return magic_token for polling
            return response.data.magic_token;
        } catch (e) {
            console.log('requestLogin error', e);
            if (e.response && e.response.data && e.response.data.error) {
                throw new Error(e.response.data.error);
            }
            throw e;
        }
    }, []);

    const completeLogin = useCallback(async (magicToken) => {
        try {
            __DEV__ && console.log(`Exchanging magic token for API token...`);
            const response = await api.get(`/auth/confirm/${magicToken}`);
            const { api_token } = response.data;

            __DEV__ && console.log(`Login confirmed, storing API token.`);
            showToast(t('login_confirmed_success'), 'success');
            setUserToken(api_token);
            await storage.setItem('userToken', api_token);
            fetchCurrentUser();
        } catch (e) {
            console.log('completeLogin error', e);
            if (e.response && e.response.data && e.response.data.error) {
                throw new Error(e.response.data.error);
            }
            throw e;
        }
    }, [fetchCurrentUser, showToast, t]);

    // Logs in the device that asked for the login email, with the poll
    // token it got back and the 6-digit code from the email. Throws an Error
    // whose status is the HTTP status (400: wrong or expired code).
    const verifyLoginCode = useCallback(async (pollToken, code) => {
        try {
            const response = await api.post('/auth/verify-code', { token: pollToken, code });
            const { api_token } = response.data;

            showToast(t('login_confirmed_success'), 'success');
            setUserToken(api_token);
            await storage.setItem('userToken', api_token);
            fetchCurrentUser();
        } catch (e) {
            console.log('verifyLoginCode error', e);
            const err = new Error(e.response?.data?.error || e.message);
            err.status = e.response?.status;
            throw err;
        }
    }, [fetchCurrentUser, showToast, t]);

    // Pass { revokeSession: false } when the server-side session is already
    // gone (expired, or deleted with the account): there is nothing to revoke.
    // Concurrent calls share the one in progress.
    const logout = useMemo(() => singleFlight(async ({ revokeSession = true } = {}) => {
        try {
            if (revokeSession) {
                await api.post('/auth/logout');
            }
        } catch (e) {
            // Silently continue local logout even if network / server fails
            __DEV__ && console.log('Error invalidating token on logout:', e);
        } finally {
            setUserToken(null);
            setUserId(null);
            setUsername(null);
            setIsAdmin(false);
            await storage.deleteItem('userToken');
            DeviceEventEmitter.emit('clear_session');
        }
    }), []);

    const isLoggedIn = useCallback(async () => {
        try {
            let token = await storage.getItem('userToken');
            if (token) {
                setUserToken(token);
                fetchCurrentUser();
            }
        } catch (e) {
            console.log(`isLoggedIn error ${e}`);
        } finally {
            setIsLoading(false);
        }
    }, [fetchCurrentUser]);

    useEffect(() => {
        isLoggedIn();
    }, [isLoggedIn]);

    useEffect(() => {
        const sub = DeviceEventEmitter.addListener('session_expired', () => {
            console.log('[AuthContext] Session expired event received.');
            if (lastUsernameRef.current) {
                setLastUsername(lastUsernameRef.current);
            }
            logout({ revokeSession: false });
            showToast(t('session_expired'), 'info');
        });
        return () => sub.remove();
    }, [logout, showToast, t]);

    const contextValue = useMemo(() => ({
        register,
        requestLogin,
        completeLogin,
        verifyLoginCode,
        logout,
        isLoading,
        userToken,
        userId,
        username,
        isAdmin,
        lastUsername
    }), [
        register,
        requestLogin,
        completeLogin,
        verifyLoginCode,
        logout,
        isLoading,
        userToken,
        userId,
        username,
        isAdmin,
        lastUsername
    ]);

    return (
        <AuthContext.Provider value={contextValue}>
            {children}
        </AuthContext.Provider>
    );
};
