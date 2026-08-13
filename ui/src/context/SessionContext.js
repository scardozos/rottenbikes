import React, { createContext, useState, useContext, useEffect } from 'react';
import { DeviceEventEmitter } from 'react-native';

export const SessionContext = createContext();

export const SessionProvider = ({ children }) => {
    // Stores the ID of the bike that has been "checked into" via Scan or Manual Input
    const [validatedBikeId, setValidatedBikeId] = useState(null);

    const validateBike = (id) => {
        console.log('[SessionContext] Validating bike:', id);
        setValidatedBikeId(id);
    };

    const clearValidation = () => {
        setValidatedBikeId(null);
    };

    useEffect(() => {
        const sub = DeviceEventEmitter.addListener('clear_session', () => {
            clearValidation();
        });
        return () => sub.remove();
    }, []);

    return (
        <SessionContext.Provider value={{ validatedBikeId, validateBike, clearValidation }}>
            {children}
        </SessionContext.Provider>
    );
};

export const useSession = () => useContext(SessionContext);
