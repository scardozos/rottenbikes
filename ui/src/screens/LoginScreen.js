import React, { useContext, useState, useEffect } from 'react';
import { View, Text, StyleSheet, ActivityIndicator, Platform, Modal, KeyboardAvoidingView, ScrollView } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import Button from '../components/Button';
import Input from '../components/Input';
import Icon from '../components/Icon';
import HCaptchaView from '../components/HCaptchaView';
import { AuthContext } from '../context/AuthContext';
import { ThemeContext } from '../context/ThemeContext';
import { useToast } from '../context/ToastContext';
import { LanguageContext } from '../context/LanguageContext';
import { useMagicLinkPolling } from '../hooks/useMagicLinkPolling';

const LoginScreen = ({ navigation }) => {
    const [identifier, setIdentifier] = useState('');
    const [step, setStep] = useState(1); // 1: Form, 2: Waiting confirmation
    const [pendingMagicToken, setPendingMagicToken] = useState(null);
    const { requestLogin, checkLoginStatus, lastUsername } = useContext(AuthContext);
    const { showToast } = useToast();
    const { theme } = useContext(ThemeContext);
    const { t } = useContext(LanguageContext);
    const [loading, setLoading] = useState(false);
    const [showCaptcha, setShowCaptcha] = useState(false);

    const HCAPTCHA_SITEKEY = (typeof window !== 'undefined' && window.EXPO_PUBLIC_HCAPTCHA_SITEKEY) || process.env.EXPO_PUBLIC_HCAPTCHA_SITEKEY;

    // eslint-disable-next-line react-hooks/set-state-in-effect
    useEffect(() => {
        if (lastUsername && !identifier) {
            setIdentifier(lastUsername);
        }
    }, [lastUsername, identifier]);

    const { pollingTimeout, resetPolling } = useMagicLinkPolling(
        step === 2,
        pendingMagicToken,
        checkLoginStatus
    );

    const handleRequestLink = async () => {
        if (!identifier.trim()) {
            showToast(t('email_or_username'), "error");
            return;
        }
        setShowCaptcha(true);
    };

    const completeRequestLink = async (captchaToken) => {
        setShowCaptcha(false);
        setLoading(true);
        try {
            const mToken = await requestLogin(identifier.trim(), captchaToken);
            setPendingMagicToken(mToken);
            setStep(2);
            resetPolling();
        } catch (e) {
            const errMsg = e.message || t('error');
            showToast(errMsg, 'error');
        } finally {
            setLoading(false);
        }
    };

    const styles = React.useMemo(() => createStyles(theme), [theme]);

    return (
        <SafeAreaView style={styles.safeArea} edges={['top', 'bottom', 'left', 'right']}>
            <KeyboardAvoidingView
                style={styles.keyboardView}
                behavior={Platform.OS === 'ios' ? 'padding' : undefined}
            >
                <ScrollView
                    contentContainerStyle={styles.scrollContainer}
                    keyboardShouldPersistTaps="handled"
                >
                    {/* Brand Header */}
                    <View style={styles.brandContainer}>
                        <View style={styles.logoCircle}>
                            <Icon name="bicycle" size={38} color={theme.colors.primary} />
                        </View>
                        <Text style={styles.brandName}>RottenBikes</Text>
                        <Text style={styles.subtitle}>{t('login')}</Text>
                    </View>

                    {step === 1 ? (
                        <View style={styles.formCard}>
                            <Input
                                label={t('email_or_username')}
                                placeholder={t('email_or_username')}
                                value={identifier}
                                onChangeText={setIdentifier}
                                autoCapitalize="none"
                                keyboardType="email-address"
                                leftIcon={<Icon name="person-outline" size={20} color={theme.colors.subtext} />}
                            />

                            <View style={styles.actionSection}>
                                {loading ? (
                                    <ActivityIndicator size="large" color={theme.colors.primary} style={styles.loader} />
                                ) : (
                                    <Button
                                        title={t('get_magic_link')}
                                        onPress={handleRequestLink}
                                        variant="primary"
                                        size="lg"
                                        style={styles.fullWidthBtn}
                                    />
                                )}
                            </View>

                            <View style={styles.switchAuthRow}>
                                <Text style={styles.switchAuthPrompt}>Don&apos;t have an account?</Text>
                                <Button
                                    title={t('register')}
                                    onPress={() => navigation.navigate('Register')}
                                    variant="ghost"
                                    size="sm"
                                    style={styles.switchAuthBtn}
                                />
                            </View>
                        </View>
                    ) : (
                        <View style={styles.waitingCard}>
                            <View style={styles.mailIconCircle}>
                                <Icon name="mail-unread-outline" size={44} color={theme.colors.primary} />
                            </View>

                            <Text style={styles.waitingTitle}>
                                {t('magic_link_requested', { identifier })}!
                            </Text>
                            <Text style={styles.waitingDescription}>
                                {t('check_email')}
                            </Text>

                            {pollingTimeout && (
                                <View style={styles.timeoutContainer}>
                                    <Text style={styles.timeoutText}>
                                        {t('polling_timeout') || 'Waiting for confirmation timed out.'}
                                    </Text>
                                    <Button
                                        title={t('resend_link') || 'Resend Link'}
                                        onPress={handleRequestLink}
                                        variant="primary"
                                        size="md"
                                        style={styles.fullWidthBtn}
                                    />
                                </View>
                            )}

                            <Button
                                title={t('back')}
                                onPress={() => setStep(1)}
                                variant="ghost"
                                size="md"
                                style={styles.backBtn}
                            />
                        </View>
                    )}
                </ScrollView>
            </KeyboardAvoidingView>

            {/* HCaptcha Modal with SafeAreaView */}
            <Modal visible={showCaptcha} animationType="slide">
                <SafeAreaView style={styles.captchaSafeArea} edges={['top', 'bottom']}>
                    <View style={styles.captchaContainer}>
                        <Text style={styles.captchaTitle}>
                            {t('complete_challenge_login')}
                        </Text>
                        <View style={styles.captchaWrapper}>
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
                        </View>
                        <View style={styles.captchaAction}>
                            <Button
                                title={t('cancel')}
                                onPress={() => setShowCaptcha(false)}
                                variant="danger"
                                size="md"
                                style={styles.fullWidthBtn}
                            />
                        </View>
                    </View>
                </SafeAreaView>
            </Modal>
        </SafeAreaView>
    );
};

const createStyles = (theme) => StyleSheet.create({
    safeArea: {
        flex: 1,
        backgroundColor: theme.colors.background,
    },
    keyboardView: {
        flex: 1,
    },
    scrollContainer: {
        flexGrow: 1,
        justifyContent: 'center',
        paddingHorizontal: theme?.metrics?.spacing?.xl || 24,
        paddingVertical: theme?.metrics?.spacing?.xxl || 32,
    },
    brandContainer: {
        alignItems: 'center',
        marginBottom: theme?.metrics?.spacing?.xl || 28,
    },
    logoCircle: {
        width: 72,
        height: 72,
        borderRadius: 36,
        backgroundColor: theme.colors.primary + '18',
        alignItems: 'center',
        justifyContent: 'center',
        marginBottom: 14,
        borderWidth: 1,
        borderColor: theme.colors.primary + '30',
    },
    brandName: {
        fontSize: 28,
        fontWeight: 'bold',
        color: theme.colors.text,
        letterSpacing: -0.5,
    },
    subtitle: {
        fontSize: 16,
        color: theme.colors.subtext,
        marginTop: 4,
    },
    formCard: {
        backgroundColor: theme.colors.card,
        borderRadius: theme?.metrics?.radii?.lg || 16,
        padding: theme?.metrics?.spacing?.xl || 24,
        borderWidth: 1,
        borderColor: theme.colors.border,
        ...(theme?.shadows?.md || {}),
    },
    actionSection: {
        marginTop: theme?.metrics?.spacing?.md || 14,
    },
    fullWidthBtn: {
        width: '100%',
    },
    loader: {
        paddingVertical: 10,
    },
    switchAuthRow: {
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'center',
        marginTop: 20,
        gap: 8,
    },
    switchAuthPrompt: {
        fontSize: 14,
        color: theme.colors.subtext,
    },
    switchAuthBtn: {
        paddingVertical: 4,
        paddingHorizontal: 12,
    },
    waitingCard: {
        backgroundColor: theme.colors.card,
        borderRadius: theme?.metrics?.radii?.lg || 16,
        padding: theme?.metrics?.spacing?.xl || 24,
        alignItems: 'center',
        borderWidth: 1,
        borderColor: theme.colors.border,
        ...(theme?.shadows?.md || {}),
    },
    mailIconCircle: {
        width: 80,
        height: 80,
        borderRadius: 40,
        backgroundColor: theme.colors.primary + '18',
        alignItems: 'center',
        justifyContent: 'center',
        marginBottom: 16,
    },
    waitingTitle: {
        fontSize: 18,
        fontWeight: 'bold',
        color: theme.colors.text,
        textAlign: 'center',
        marginBottom: 8,
    },
    waitingDescription: {
        fontSize: 15,
        color: theme.colors.subtext,
        textAlign: 'center',
        lineHeight: 22,
        marginBottom: 20,
    },
    timeoutContainer: {
        width: '100%',
        alignItems: 'center',
        marginBottom: 16,
    },
    timeoutText: {
        color: theme.colors.error,
        textAlign: 'center',
        marginBottom: 10,
        fontSize: 14,
    },
    backBtn: {
        width: '100%',
    },
    captchaSafeArea: {
        flex: 1,
        backgroundColor: theme.colors.background,
    },
    captchaContainer: {
        flex: 1,
        padding: theme?.metrics?.spacing?.lg || 16,
    },
    captchaTitle: {
        textAlign: 'center',
        fontSize: 18,
        fontWeight: 'bold',
        marginVertical: 16,
        color: theme.colors.text,
    },
    captchaWrapper: {
        flex: 1,
    },
    captchaAction: {
        paddingTop: 16,
    },
});

export default LoginScreen;
