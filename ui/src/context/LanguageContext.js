import React, { createContext, useState, useEffect, useContext } from 'react';
import storage from '../utils/storage';
import { interpolate } from '../utils/i18n';

import ca from '../translations/ca';
import es from '../translations/es';
import en from '../translations/en';

export const LanguageContext = createContext();

const translations = {
    ca,
    es,
    en
};

export const LanguageProvider = ({ children }) => {
    const [language, setLanguage] = useState('ca'); // Default to Catalan

    useEffect(() => {
        const loadLanguage = async () => {
            try {
                const storedLanguage = await storage.getItem('userLanguage');
                if (storedLanguage && translations[storedLanguage]) {
                    setLanguage(storedLanguage);
                }
            } catch (e) {
                console.warn("Failed to load language preference", e);
            }
        };

        loadLanguage();
    }, []);

    const changeLanguage = async (lang) => {
        if (!translations[lang]) return;
        setLanguage(lang);

        try {
            await storage.setItem('userLanguage', lang);
        } catch (e) {
            console.warn("Failed to save language preference", e);
        }
    };

    const t = (key, params = {}) => {
        const text = translations[language][key] || key;
        return interpolate(text, params);
    };

    return (
        <LanguageContext.Provider value={{ language, changeLanguage, t }}>
            {children}
        </LanguageContext.Provider>
    );
};

export const useLanguage = () => useContext(LanguageContext);
