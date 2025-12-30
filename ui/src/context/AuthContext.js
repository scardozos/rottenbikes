import React, { createContext, useState, useEffect, useContext } from 'react';
import { Platform } from 'react-native';
import storage from '../utils/storage';
import api from '../services/api';
import { useToast } from './ToastContext';
import { LanguageContext } from './LanguageContext';

export const AuthContext = createContext();

export const AuthProvider = ({ children }) => {
    const [isLoading, setIsLoading] = useState(true);
    const [userToken, setUserToken] = useState(null);
    const [userId, setUserId] = useState(null);
    const [username, setUsername] = useState(null);
    const [lastUsername, setLastUsername] = useState(null);
    const { showToast } = useToast();
    const { t } = useContext(LanguageContext);

    const fetchCurrentUser = async () => {
        try {
            const res = await api.get('/auth/verify');
            if (res.data && res.data.poster_id) {
                console.log('[AuthContext] Fetched current user:', res.data);
                setUserId(res.data.poster_id);
                setUsername(res.data.username);
            }
        } catch (e) {
            console.log('[AuthContext] Failed to fetch current user:', e);
            if (e.response && e.response.status === 401) {
                console.log('[AuthContext] Session expired (401). Logging out.');
                setLastUsername(username); // Store current username before clearing
                logout();
                showToast(t('session_expired'), 'info');
            }
        }
    };

    const register = async (username, email, captcha) => {
        try {
            console.log(`Registering user: ${username} with email: ${email}`);
            const data = {
                username: username,
                email: email,
                captcha_token: captcha
            };
            if (Platform.OS !== 'web') {
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
    };

    const requestLogin = async (identifier, captcha) => {
        try {
            console.log(`Requesting magic link for identifier: ${identifier}`);
            const isEmail = identifier.includes('@');
            const data = {};
            if (isEmail) {
                data.email = identifier;
            } else {
                data.username = identifier;
            }
            if (Platform.OS !== 'web') {
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
    };

    const confirmAttempt = async (magicToken) => {
        try {
            console.log(`Confirming magic link attempt (only) for token: ${magicToken}`);
            await api.get(`/auth/confirm/${magicToken}`);
            showToast(t('confirmation_successful'), 'success');
        } catch (e) {
            console.log('confirmAttempt error', e);
            if (e.response && e.response.data && e.response.data.error) {
                throw new Error(e.response.data.error);
            }
            throw e;
        }
    };

    const completeLogin = async (magicToken) => {
        try {
            console.log(`Exchanging magic token for API token...`);
            const response = await api.get(`/auth/confirm/${magicToken}`);
            const { api_token } = response.data;

            console.log(`Login confirmed, storing API token.`);
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
    };

    const checkLoginStatus = async (magicToken) => {
        try {
            const response = await api.get(`/auth/poll?token=${magicToken}`);
            const { api_token } = response.data;
            if (api_token) {
                showToast(t('login_confirmed_success'), 'success');
                setUserToken(api_token);
                await storage.setItem('userToken', api_token);
                fetchCurrentUser();
                return true;
            }
        } catch (e) {
            if (e.response && e.response.status === 404) {
                // silenty retry
            }
            return false;
        }
        return false;
    };

    const logout = async () => {
        setUserToken(null);
        setUserId(null);
        setUsername(null);
        await storage.deleteItem('userToken');
    };

    const isLoggedIn = async () => {
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
    };

    useEffect(() => {
        isLoggedIn();
    }, []);

    return (
        <AuthContext.Provider value={{
            register,
            requestLogin,
            confirmAttempt,
            completeLogin,
            checkLoginStatus,
            logout,
            isLoading,
            userToken,
            userId,
            username,
            lastUsername
        }}>
            {children}
        </AuthContext.Provider>
    );
};
