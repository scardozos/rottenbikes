import React, { useContext, useState, useEffect } from 'react';
import { View, Text, TextInput, Button, StyleSheet, ActivityIndicator, Platform, Modal, Alert } from 'react-native';
import HCaptchaView from '../components/HCaptchaView';
import { AuthContext } from '../context/AuthContext';
import { ThemeContext } from '../context/ThemeContext';
import { useToast } from '../context/ToastContext';

const LoginScreen = ({ navigation }) => {
    const [identifier, setIdentifier] = useState('');
    const [step, setStep] = useState(1); // 1: Email, 2: Done
    const [pendingMagicToken, setPendingMagicToken] = useState(null);
    const { requestLogin, checkLoginStatus } = useContext(AuthContext);
    const { showToast } = useToast();
    const { theme } = useContext(ThemeContext);
    const [loading, setLoading] = useState(false);
    const [showCaptcha, setShowCaptcha] = useState(false);

    // Replace with your real sitekey
    const HCAPTCHA_SITEKEY = window.EXPO_PUBLIC_HCAPTCHA_SITEKEY || process.env.EXPO_PUBLIC_HCAPTCHA_SITEKEY || "10000000-ffff-ffff-ffff-000000000001";

    useEffect(() => {
        let interval;
        // Polling MUST only take place if it happens from within the mobile application, not the web.
        if (step === 2 && pendingMagicToken && Platform.OS !== 'web') {
            interval = setInterval(async () => {
                const confirmed = await checkLoginStatus(pendingMagicToken);
                if (confirmed) {
                    clearInterval(interval);
                }
            }, 5000);
        }
        return () => {
            if (interval) clearInterval(interval);
        };
    }, [step, pendingMagicToken]);

    const handleRequestLink = async () => {
        if (!identifier) {
            showToast("Please enter your email or username", "error");
            return;
        }
        setShowCaptcha(true);
    };

    const completeRequestLink = async (captchaToken) => {
        setShowCaptcha(false);
        setLoading(true);
        try {
            const data = {
                captcha_token: captchaToken
            };
            const mToken = await requestLogin(identifier, captchaToken);
            setPendingMagicToken(mToken);
            setStep(2);
        } catch (e) {
            const errMsg = e.message || 'Failed to request magic link';
            showToast(errMsg, 'error');
        } finally {
            setLoading(false);
        }
    };

    const styles = createStyles(theme);

    return (
        <View style={styles.container}>
            <Text style={styles.title}>RottenBikes Login</Text>

            {step === 1 ? (
                <>
                    <TextInput
                        style={styles.input}
                        placeholder="Email or Username"
                        placeholderTextColor={theme.colors.placeholder}
                        value={identifier}
                        onChangeText={setIdentifier}
                        autoCapitalize="none"
                        keyboardType="email-address"
                    />
                    {loading ? (
                        <ActivityIndicator size="large" color={theme.colors.primary} />
                    ) : (
                        <Button title="Get Magic Link" onPress={handleRequestLink} color={theme.colors.primary} />
                    )}

                    <Modal visible={showCaptcha} animationType="slide">
                        <View style={{ flex: 1, backgroundColor: theme.colors.background, paddingVertical: 50 }}>
                            <Text style={{ textAlign: 'center', fontSize: 18, marginBottom: 20, color: theme.colors.text }}>
                                Complete the challenge to login
                            </Text>
                            <HCaptchaView
                                siteKey={HCAPTCHA_SITEKEY}
                                onVerify={completeRequestLink}
                                onExpired={() => {
                                    setShowCaptcha(false);
                                    showToast("Captcha expired", "error");
                                }}
                                onError={() => {
                                    setShowCaptcha(false);
                                    showToast("Captcha failed", "error");
                                }}
                            />
                            <View style={{ marginTop: 40, paddingHorizontal: 20 }}>
                                <Button title="Cancel" onPress={() => setShowCaptcha(false)} color={theme.colors.error} />
                            </View>
                        </View>
                    </Modal>
                </>
            ) : (
                <>
                    <Text style={{ marginBottom: 20, textAlign: 'center', fontSize: 16, color: theme.colors.text }}>
                        Magic link requested for {identifier}!{'\n\n'}
                        Check your email for the link and click it to log in.
                    </Text>
                    <Button title="Back" onPress={() => setStep(1)} color={theme.colors.subtext} />
                </>
            )}

            <View style={{ marginTop: 20 }}>
                <Button
                    title="Register"
                    onPress={() => navigation.navigate('Register')}
                    color={theme.colors.secondary}
                />
            </View>
        </View>
    );
};

const createStyles = (theme) => StyleSheet.create({
    container: { flex: 1, justifyContent: 'center', padding: 20, backgroundColor: theme.colors.background },
    title: { fontSize: 24, marginBottom: 20, textAlign: 'center', color: theme.colors.text },
    input: {
        height: 40,
        borderColor: theme.colors.border,
        borderWidth: 1,
        marginBottom: 12,
        paddingHorizontal: 8,
        borderRadius: 4,
        color: theme.colors.text,
        backgroundColor: theme.colors.inputBackground,
    },
});

export default LoginScreen;
