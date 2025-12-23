import React, { createContext, useState, useEffect, useContext } from 'react';
import storage from '../utils/storage';
import { useColorScheme } from 'react-native';

export const ThemeContext = createContext();

export const themes = {
    dark: {
        dark: true,
        colors: {
            primary: '#60A5FA', // Blue 400
            secondary: '#34D399', // Emerald 400
            background: '#0F172A', // Slate 900
            card: '#1E293B', // Slate 800
            text: '#F1F5F9', // Slate 100
            subtext: '#94A3B8', // Slate 400
            border: '#334155', // Slate 700
            notification: '#F87171', // Red 400
            inputBackground: '#1E293B', // Slate 800 (Card color) or slightly lighter
            placeholder: '#64748B', // Slate 500
            error: '#EF4444', // Red 500
            success: '#10B981', // Emerald 500
            buttonText: '#FFFFFF',
        },
        fonts: { regular: { fontFamily: 'System', fontWeight: '400' } } // Fix for potential missing fonts prop
    },
    light: {
        dark: false,
        colors: {
            primary: '#2563EB', // Blue 600
            secondary: '#059669', // Emerald 600
            background: '#FFFFFF',
            card: '#F8FAFC', // Slate 50
            text: '#0F172A', // Slate 900
            subtext: '#64748B', // Slate 500
            border: '#E2E8F0', // Slate 200
            notification: '#DC2626', // Red 600
            inputBackground: '#F1F5F9', // Slate 100
            placeholder: '#94A3B8', // Slate 400
            error: '#DC2626', // Red 600
            success: '#059669', // Emerald 600
            buttonText: '#FFFFFF',
        },
        fonts: { regular: { fontFamily: 'System', fontWeight: '400' } } // Fix for potential missing fonts prop
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
