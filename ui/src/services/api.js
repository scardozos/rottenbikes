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
        }
        return Promise.reject(error);
    }
);

export default api;
