import * as SecureStore from 'expo-secure-store';
import { Platform } from 'react-native';

// Robust check for web
const isWeb = Platform.OS === 'web';

const setItem = async (key, value) => {
    if (isWeb) {
        try {
            if (typeof localStorage !== 'undefined') {
                localStorage.setItem(key, value);
            }
        } catch (e) {
            console.error('Local storage setItem failed', e);
        }
    } else {
        try {
            await SecureStore.setItemAsync(key, value);
        } catch (e) {
            console.error('SecureStore setItemAsync failed', e);
            throw e;
        }
    }
};

const getItem = async (key) => {
    if (isWeb) {
        try {
            if (typeof localStorage !== 'undefined') {
                return localStorage.getItem(key);
            }
        } catch (e) {
            console.error('Local storage getItem failed', e);
        }
        return null;
    } else {
        try {
            return await SecureStore.getItemAsync(key);
        } catch (e) {
            console.error('SecureStore getItemAsync failed', e);
            return null;
        }
    }
};

const deleteItem = async (key) => {
    if (isWeb) {
        try {
            if (typeof localStorage !== 'undefined') {
                localStorage.removeItem(key);
            }
        } catch (e) {
            console.error('Local storage removeItem failed', e);
        }
    } else {
        try {
            await SecureStore.deleteItemAsync(key);
        } catch (e) {
            console.error('SecureStore deleteItemAsync failed', e);
        }
    }
};

export default { setItem, getItem, deleteItem };
