import React, { createContext, useState, useEffect, useContext, useMemo } from 'react';
import storage from '../utils/storage';

export const ThemeContext = createContext();

const metrics = {
    spacing: { xs: 4, sm: 8, md: 12, lg: 16, xl: 24, xxl: 32 },
    radii: { sm: 8, md: 12, lg: 16, xl: 24, pill: 9999 },
};

const typography = {
    h1: { fontSize: 32, fontWeight: 'bold' },
    h2: { fontSize: 24, fontWeight: 'bold' },
    h3: { fontSize: 20, fontWeight: '600' },
    body: { fontSize: 16, fontWeight: '400' },
    bodyBold: { fontSize: 16, fontWeight: 'bold' },
    caption: { fontSize: 14, fontWeight: '400' },
    captionBold: { fontSize: 14, fontWeight: '600' },
    small: { fontSize: 12, fontWeight: '400' },
};

const baseTheme = {
    metrics,
    typography,
    fonts: { regular: { fontFamily: 'System', fontWeight: '400' } }
};

export const themes = {
    dark: {
        ...baseTheme,
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
            danger: '#EF4444',
            success: '#10B981', // Emerald 500
            warning: '#F59E0B', // Amber 500
            ghostBackground: 'rgba(255, 255, 255, 0.1)',
            buttonText: '#FFFFFF',
        },
        shadows: {
            sm: { shadowColor: '#000', shadowOffset: { width: 0, height: 1 }, shadowOpacity: 0.3, shadowRadius: 2, elevation: 2 },
            md: { shadowColor: '#000', shadowOffset: { width: 0, height: 2 }, shadowOpacity: 0.4, shadowRadius: 4, elevation: 4 },
            lg: { shadowColor: '#000', shadowOffset: { width: 0, height: 4 }, shadowOpacity: 0.5, shadowRadius: 8, elevation: 8 },
        }
    },
    light: {
        ...baseTheme,
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
            danger: '#DC2626',
            success: '#059669', // Emerald 600
            warning: '#D97706', // Amber 600
            ghostBackground: 'rgba(0, 0, 0, 0.05)',
            buttonText: '#FFFFFF',
        },
        shadows: {
            sm: { shadowColor: '#000', shadowOffset: { width: 0, height: 1 }, shadowOpacity: 0.1, shadowRadius: 2, elevation: 2 },
            md: { shadowColor: '#000', shadowOffset: { width: 0, height: 2 }, shadowOpacity: 0.1, shadowRadius: 4, elevation: 4 },
            lg: { shadowColor: '#000', shadowOffset: { width: 0, height: 4 }, shadowOpacity: 0.15, shadowRadius: 8, elevation: 8 },
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
    
    // eslint-disable-next-line react-hooks/exhaustive-deps
    const contextValue = useMemo(() => ({ theme, themeName, toggleTheme, isDark: themeName === 'dark' }), [themeName]);

    return (
        <ThemeContext.Provider value={contextValue}>
            {children}
        </ThemeContext.Provider>
    );
};

export const useTheme = () => useContext(ThemeContext);
