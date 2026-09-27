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
            primary: '#14B8A6', // Electric Teal
            secondary: '#3B82F6', // Cobalt Blue
            background: '#09090B', // True Carbon
            card: '#18181B', // Elevated Carbon
            text: '#F4F4F5', // Off-white
            subtext: '#A1A1AA', // Zinc 400
            border: '#27272A', // Zinc 800
            notification: '#14B8A6',
            inputBackground: '#27272A',
            placeholder: '#71717A', // Zinc 500
            error: '#F87171',
            danger: '#F87171',
            success: '#34D399',
            warning: '#FBBF24',
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
            primary: '#0F766E', // Electric Teal
            secondary: '#3B82F6', // Cobalt Blue
            background: '#F4F4F5', // Soft gray
            card: '#FFFFFF',
            text: '#18181B', // Near black
            subtext: '#52525B', // Zinc 600
            border: '#E4E4E7', // Zinc 200
            notification: '#0F766E',
            inputBackground: '#F4F4F5',
            placeholder: '#A1A1AA', // Zinc 400
            error: '#DC2626',
            danger: '#DC2626',
            success: '#059669',
            warning: '#D97706',
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
