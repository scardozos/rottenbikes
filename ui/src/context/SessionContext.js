import React, { createContext, useState, useContext, useEffect, useCallback, useMemo } from 'react';
import { DeviceEventEmitter } from 'react-native';

export const SessionContext = createContext();

export const SessionProvider = ({ children }) => {
    // Stores the ID of the bike that has been "checked into" via Scan or Manual Input
    const [validatedBikeId, setValidatedBikeId] = useState(null);

    const validateBike = useCallback((id) => {
        console.log('[SessionContext] Validating bike:', id);
        setValidatedBikeId(id);
    }, []);

    const clearValidation = useCallback(() => {
        setValidatedBikeId(null);
    }, []);

    useEffect(() => {
        const sub = DeviceEventEmitter.addListener('clear_session', () => {
            clearValidation();
        });
        return () => sub.remove();
    }, [clearValidation]);

    const contextValue = useMemo(() => ({ validatedBikeId, validateBike, clearValidation }), [validatedBikeId, validateBike, clearValidation]);

    return (
        <SessionContext.Provider value={contextValue}>
            {children}
        </SessionContext.Provider>
    );
};

export const useSession = () => useContext(SessionContext);
