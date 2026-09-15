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
            primary: '#A3E635', // Lime 400
            secondary: '#2DD4BF', // Teal 400
            background: '#0C1210', // Green-tinted near-black
            card: '#151E19', // Deep green-charcoal
            text: '#EAF2E6', // Soft green-white
            subtext: '#8CA392', // Desaturated green-gray
            border: '#26332B', // Muted green-gray
            notification: '#F87171', // Red 400
            inputBackground: '#1A241E', // Slightly lighter than card
            placeholder: '#5A6B60', // Muted green-gray
            error: '#F87171', // Red 400
            danger: '#F87171',
            success: '#34D399', // Emerald 400
            warning: '#FBBF24', // Amber 400
            ghostBackground: 'rgba(255, 255, 255, 0.1)',
            // Lime is light enough that white text fails contrast; buttons use dark text
            buttonText: '#101606',
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
            primary: '#65A30D', // Lime 600
            secondary: '#0D9488', // Teal 600
            background: '#F7F9F2', // Warm off-white
            card: '#FFFFFF',
            text: '#182415', // Green-tinted charcoal
            subtext: '#5C6B58', // Green-gray
            border: '#DFE6DA', // Soft green-gray
            notification: '#DC2626', // Red 600
            inputBackground: '#EEF2E8', // Pale green-tinted
            placeholder: '#98A693', // Muted green-gray
            error: '#DC2626', // Red 600
            danger: '#DC2626',
            success: '#047857', // Emerald 700
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
