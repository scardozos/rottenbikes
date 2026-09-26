import React, { useState, useEffect, useContext, useCallback } from 'react';
import { Modal } from 'react-native';
import { AuthContext } from '../context/AuthContext';
import storage from '../utils/storage';
import OnboardingScreen from '../screens/OnboardingScreen';

const ONBOARDING_FLAG_KEY = 'hasCompletedOnboarding';

// Shows the onboarding flow once per device: whenever the user is logged in
// and the completion flag has not been set yet.
export const OnboardingGate = ({ children }) => {
    const { isLoading, userToken } = useContext(AuthContext);
    const [showOnboarding, setShowOnboarding] = useState(false);

    useEffect(() => {
        if (isLoading || userToken == null) return;

        let cancelled = false;
        (async () => {
            try {
                const completed = await storage.getItem(ONBOARDING_FLAG_KEY);
                if (!cancelled && completed !== 'true') {
                    setShowOnboarding(true);
                }
            } catch (e) {
                console.log('[OnboardingGate] Failed to read onboarding flag', e);
            }
        })();

        return () => { cancelled = true; };
    }, [isLoading, userToken]);

    const finishOnboarding = useCallback(async () => {
        try {
            await storage.setItem(ONBOARDING_FLAG_KEY, 'true');
        } catch (e) {
            console.log('[OnboardingGate] Failed to persist onboarding flag', e);
        } finally {
            // Always close the modal so the user can never get stuck,
            // even if persisting the flag fails.
            setShowOnboarding(false);
        }
    }, []);

    return (
        <>
            {children}
            <Modal
                visible={showOnboarding}
                animationType="slide"
                onRequestClose={finishOnboarding}
            >
                <OnboardingScreen onFinish={finishOnboarding} />
            </Modal>
        </>
    );
};

export default OnboardingGate;
