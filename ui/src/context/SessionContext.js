import React, { createContext, useState, useContext, useEffect, useCallback, useMemo } from 'react';
import { DeviceEventEmitter } from 'react-native';

export const SessionContext = createContext();

// How a bike was "checked into": an actual QR scan, or manual ID entry.
// Used to set the review/bike was_scanned moderation flag.
export const BIKE_ORIGIN = {
    SCAN: 'scan',
    MANUAL: 'manual',
};

export const SessionProvider = ({ children }) => {
    // Stores the ID of the bike that has been "checked into" via Scan or Manual Input
    const [validatedBikeId, setValidatedBikeId] = useState(null);
    // Stores how the bike was validated ('scan' | 'manual')
    const [validatedBikeOrigin, setValidatedBikeOrigin] = useState(null);

    const validateBike = useCallback((id, origin = BIKE_ORIGIN.MANUAL) => {
        console.log('[SessionContext] Validating bike:', id, 'origin:', origin);
        setValidatedBikeId(id);
        setValidatedBikeOrigin(origin);
    }, []);

    const clearValidation = useCallback(() => {
        setValidatedBikeId(null);
        setValidatedBikeOrigin(null);
    }, []);

    useEffect(() => {
        const sub = DeviceEventEmitter.addListener('clear_session', () => {
            clearValidation();
        });
        return () => sub.remove();
    }, [clearValidation]);

    const contextValue = useMemo(() => ({ validatedBikeId, validatedBikeOrigin, validateBike, clearValidation }), [validatedBikeId, validatedBikeOrigin, validateBike, clearValidation]);

    return (
        <SessionContext.Provider value={contextValue}>
            {children}
        </SessionContext.Provider>
    );
};

export const useSession = () => useContext(SessionContext);
