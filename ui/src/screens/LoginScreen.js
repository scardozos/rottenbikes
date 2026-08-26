import React, { useContext, useState, useEffect } from 'react';
import { View, Text, TextInput, Button, StyleSheet, ActivityIndicator, Platform, Modal, Alert, KeyboardAvoidingView } from 'react-native';
import HCaptchaView from '../components/HCaptchaView';
import { AuthContext } from '../context/AuthContext';
import { ThemeContext } from '../context/ThemeContext';
import { useToast } from '../context/ToastContext';
import { LanguageContext } from '../context/LanguageContext';

const LoginScreen = ({ navigation }) => {
    const [identifier, setIdentifier] = useState('');
    const [step, setStep] = useState(1); // 1: Email, 2: Done
    const [pendingMagicToken, setPendingMagicToken] = useState(null);
    const { requestLogin, checkLoginStatus } = useContext(AuthContext);
    const { showToast } = useToast();
    const { theme } = useContext(ThemeContext);
    const { t } = useContext(LanguageContext);
    const [loading, setLoading] = useState(false);
    const [showCaptcha, setShowCaptcha] = useState(false);
    const { lastUsername } = useContext(AuthContext);

    // Replace with your real sitekey
    const HCAPTCHA_SITEKEY = (typeof window !== 'undefined' && window.EXPO_PUBLIC_HCAPTCHA_SITEKEY) || process.env.EXPO_PUBLIC_HCAPTCHA_SITEKEY;

    useEffect(() => {
        // Pre-fill username if session expired and standard identifier is empty
        if (lastUsername && !identifier) {
            setIdentifier(lastUsername);
        }
    }, [lastUsername, identifier]);

    const [pollingTimeout, setPollingTimeout] = useState(false);

    useEffect(() => {
        let interval;
        let attempts = 0;
        const maxAttempts = 24; // 2 minutes at 5s intervals

        if (step === 2 && pendingMagicToken && Platform.OS !== 'web') {
            setPollingTimeout(false);
            interval = setInterval(async () => {
                attempts++;
                if (attempts >= maxAttempts) {
                    clearInterval(interval);
                    setPollingTimeout(true);
                    return;
                }
                const confirmed = await checkLoginStatus(pendingMagicToken);
                if (confirmed) {
                    clearInterval(interval);
                }
            }, 5000);
        }
        return () => {
            if (interval) clearInterval(interval);
        };
    }, [step, pendingMagicToken, checkLoginStatus]);

    const handleRequestLink = async () => {
        if (!identifier) {
            showToast(t('email_or_username'), "error");
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
            setPollingTimeout(false);
        } catch (e) {
            const errMsg = e.message || t('error');
            showToast(errMsg, 'error');
        } finally {
            setLoading(false);
        }
    };

    const styles = React.useMemo(() => createStyles(theme), [theme]);

    return (
        <KeyboardAvoidingView
            style={styles.container}
            behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
        >
            <Text style={styles.title}>RottenBikes {t('login')}</Text>

            {step === 1 ? (
                <>
                    <TextInput
                        style={styles.input}
                        placeholder={t('email_or_username')}
                        placeholderTextColor={theme.colors.placeholder}
                        value={identifier}
                        onChangeText={setIdentifier}
                        autoCapitalize="none"
                        keyboardType="email-address"
                    />
                    {loading ? (
                        <ActivityIndicator size="large" color={theme.colors.primary} />
                    ) : (
                        <Button title={t('get_magic_link')} onPress={handleRequestLink} color={theme.colors.primary} />
                    )}

                    <Modal visible={showCaptcha} animationType="slide">
                        <View style={{ flex: 1, backgroundColor: theme.colors.background, paddingVertical: 50 }}>
                            <Text style={{ textAlign: 'center', fontSize: 18, marginBottom: 20, color: theme.colors.text }}>
                                {t('complete_challenge_login')}
                            </Text>
                            <HCaptchaView
                                siteKey={HCAPTCHA_SITEKEY}
                                onVerify={completeRequestLink}
                                onExpired={() => {
                                    setShowCaptcha(false);
                                    showToast(t('captcha_expired'), "error");
                                }}
                                onError={() => {
                                    setShowCaptcha(false);
                                    showToast(t('captcha_failed'), "error");
                                }}
                            />
                            <View style={{ marginTop: 40, paddingHorizontal: 20 }}>
                                <Button title={t('cancel')} onPress={() => setShowCaptcha(false)} color={theme.colors.error} />
                            </View>
                        </View>
                    </Modal>
                </>
            ) : (
                <>
                    <Text style={{ marginBottom: 20, textAlign: 'center', fontSize: 16, color: theme.colors.text }}>
                        {t('magic_link_requested', { identifier })}!{'\n\n'}
                        {t('check_email')}
                    </Text>
                    {pollingTimeout && (
                        <View style={{ marginBottom: 20 }}>
                            <Text style={{ textAlign: 'center', color: theme.colors.error, marginBottom: 10 }}>
                                {t('polling_timeout') || 'Waiting for confirmation timed out.'}
                            </Text>
                            <Button title={t('resend_link') || 'Resend Link'} onPress={handleRequestLink} color={theme.colors.primary} />
                        </View>
                    )}
                    <Button title={t('back')} onPress={() => setStep(1)} color={theme.colors.subtext} />
                </>
            )}

            <View style={{ marginTop: 20 }}>
                <Button
                    title={t('register')}
                    onPress={() => navigation.navigate('Register')}
                    color={theme.colors.secondary}
                />
            </View>
        </KeyboardAvoidingView>
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
