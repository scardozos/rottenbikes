import React, { createContext, useState, useContext } from 'react';

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

    return (
        <SessionContext.Provider value={{ validatedBikeId, validateBike, clearValidation }}>
            {children}
        </SessionContext.Provider>
    );
};

export const useSession = () => useContext(SessionContext);
