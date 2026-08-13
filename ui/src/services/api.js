import axios from 'axios';
import storage from '../utils/storage';

const API_URL = window.EXPO_PUBLIC_API_URL || process.env.EXPO_PUBLIC_API_URL || (__DEV__ ? 'http://localhost:8080' : '');

const api = axios.create({
    baseURL: API_URL,
    headers: {
        'Content-Type': 'application/json',
    },
});

api.interceptors.request.use(
    async (config) => {
        const token = await storage.getItem('userToken');
        if (token) {
            config.headers.Authorization = `Bearer ${token}`;
        }
        return config;
    },
    (error) => {
        return Promise.reject(error);
    }
);

import { DeviceEventEmitter } from 'react-native';

api.interceptors.response.use(
    (response) => response,
    (error) => {
        if (error.response && error.response.status === 401) {
            DeviceEventEmitter.emit('session_expired');
        } else if (error.response && error.response.status === 429) {
            const retryAfter = error.response.headers['retry-after'];
            if (retryAfter) {
                // If it's a number of seconds
                const seconds = parseInt(retryAfter, 10);
                if (!isNaN(seconds)) {
                    const minutes = Math.ceil(seconds / 60);
                    if (error.response.data) {
                        error.response.data.error = `${error.response.data.error || 'Rate limited'}. Please try again in ${minutes} minute(s).`;
                    }
                }
            }
        }
        return Promise.reject(error);
    }
);

export default api;
