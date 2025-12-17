import axios from 'axios';
import storage from '../utils/storage';

// Replace with your API URL. For Android Emulator use 10.0.2.2 usually, for iOS localhost.
// For web, localhost:8080 is fine.
// We can make this dynamic or env based later.
// For mobile phone use the IP of the machine running the API.
const API_URL = 'http://localhost:8080';

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

export default api;
