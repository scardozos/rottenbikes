import React, { createContext, useState, useEffect } from 'react';
import storage from '../utils/storage';
import api from '../services/api';

export const AuthContext = createContext();

export const AuthProvider = ({ children }) => {
    const [isLoading, setIsLoading] = useState(true);
    const [userToken, setUserToken] = useState(null);

    const requestLogin = async (email) => {
        try {
            console.log(`Requesting magic link for email: ${email}`);
            await api.post('/auth/request-magic-link', {
                email: email,
                captcha_token: "dev-dummy"
            });
            // Don't set token yet, wait for user to enter it
            return true;
        } catch (e) {
            console.log('requestLogin error', e);
            throw e;
        }
    };

    const completeLogin = async (token) => {
        try {
            console.log(`Completing login with token: ${token}`);
            // Verify token works (optional, but good practice, or just store it)
            // For now, assume it's valid if provided, but typically we'd use it immediately.
            // But wait, the token IS the auth credential.

            setUserToken(token);
            await storage.setItem('userToken', token);
        } catch (e) {
            console.log('completeLogin error', e);
            throw e;
        }
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
        <AuthContext.Provider value={{ requestLogin, completeLogin, logout, isLoading, userToken }}>
            {children}
        </AuthContext.Provider>
    );
};
