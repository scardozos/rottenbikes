import React, { createContext, useState, useEffect, useContext } from 'react';
import { Platform } from 'react-native';
import storage from '../utils/storage';
import api from '../services/api';
import { useToast } from './ToastContext';

export const AuthContext = createContext();

export const AuthProvider = ({ children }) => {
    const [isLoading, setIsLoading] = useState(true);
    const [userToken, setUserToken] = useState(null);
    const { showToast } = useToast();

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
            throw e;
        }
    };

    const confirmAttempt = async (magicToken) => {
        try {
            console.log(`Confirming magic link attempt (only) for token: ${magicToken}`);
            await api.get(`/auth/confirm/${magicToken}`);
            showToast('Confirmation Successful!', 'success');
        } catch (e) {
            console.log('confirmAttempt error', e);
            throw e;
        }
    };

    const completeLogin = async (magicToken) => {
        try {
            console.log(`Exchanging magic token for API token...`);
            const response = await api.get(`/auth/confirm/${magicToken}`);
            const { api_token } = response.data;

            console.log(`Login confirmed, storing API token.`);
            showToast('Login Confirmed! Success!', 'success');
            setUserToken(api_token);
            await storage.setItem('userToken', api_token);
        } catch (e) {
            console.log('completeLogin error', e);
            throw e;
        }
    };

    const checkLoginStatus = async (magicToken) => {
        try {
            const response = await api.get(`/auth/poll?token=${magicToken}`);
            const { api_token } = response.data;
            if (api_token) {
                showToast('Login Confirmed! Success!', 'success');
                setUserToken(api_token);
                await storage.setItem('userToken', api_token);
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
        await storage.deleteItem('userToken');
    };

    const isLoggedIn = async () => {
        try {
            let token = await storage.getItem('userToken');
            if (token) {
                setUserToken(token);
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
            userToken
        }}>
            {children}
        </AuthContext.Provider>
    );
};
