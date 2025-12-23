import React, { createContext, useState, useEffect, useContext } from 'react';
import storage from '../utils/storage';
import { useColorScheme } from 'react-native';

export const ThemeContext = createContext();

export const themes = {
    dark: {
        dark: true,
        colors: {
            primary: '#BB86FC',
            secondary: '#03DAC6',
            background: '#121212',
            card: '#1E1E1E',
            text: '#FFFFFF',
            subtext: '#A0A0A0',
            border: '#2C2C2C',
            notification: '#CF6679',
            inputBackground: '#2C2C2C',
            placeholder: '#A0A0A0',
            error: '#CF6679',
            success: '#03DAC6',
            buttonText: '#000000',
        }
    },
    light: {
        dark: false,
        colors: {
            primary: '#6200EE',
            secondary: '#03DAC6',
            background: '#FFFFFF',
            card: '#F5F5F5',
            text: '#000000',
            subtext: '#666666',
            border: '#E0E0E0',
            notification: '#B00020',
            inputBackground: '#F0F0F0',
            placeholder: '#999999',
            error: '#B00020',
            success: '#00C853',
            buttonText: '#FFFFFF',
        }
    }
};

export const ThemeProvider = ({ children }) => {
    // Default to dark mode as requested
    const [themeName, setThemeName] = useState('dark');
    const [isLoading, setIsLoading] = useState(true);

    useEffect(() => {
        loadTheme();
    }, []);

    const loadTheme = async () => {
        try {
            const savedTheme = await storage.getItem('userTheme');
            if (savedTheme) {
                setThemeName(savedTheme);
            }
        } catch (e) {
            console.log('Failed to load theme', e);
        } finally {
            setIsLoading(false);
        }
    };

    const toggleTheme = async () => {
        const newTheme = themeName === 'dark' ? 'light' : 'dark';
        setThemeName(newTheme);
        try {
            await storage.setItem('userTheme', newTheme);
        } catch (e) {
            console.log('Failed to save theme', e);
        }
    };

    const theme = themes[themeName];

    return (
        <ThemeContext.Provider value={{ theme, themeName, toggleTheme, isDark: themeName === 'dark' }}>
            {children}
        </ThemeContext.Provider>
    );
};

export const useTheme = () => useContext(ThemeContext);
