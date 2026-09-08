import React, { useState, useContext } from 'react';
import { View, Text, StyleSheet, Modal, Platform, Switch, TouchableOpacity, KeyboardAvoidingView, ScrollView } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import Button from '../components/Button';
import Input from '../components/Input';
import Icon from '../components/Icon';
import { AuthContext } from '../context/AuthContext';
import { ThemeContext } from '../context/ThemeContext';
import { LanguageContext } from '../context/LanguageContext';
import HCaptchaView from '../components/HCaptchaView';
import { useToast } from '../context/ToastContext';
import { isValidUsername, isValidEmail } from '../utils/validation';
import { useMagicLinkPolling } from '../hooks/useMagicLinkPolling';

const RegisterScreen = ({ navigation }) => {
    const [username, setUsername] = useState('');
    const [usernameError, setUsernameError] = useState('');
    const [email, setEmail] = useState('');
    const [emailError, setEmailError] = useState('');
    const [acceptedTerms, setAcceptedTerms] = useState(false);
    const [showCaptcha, setShowCaptcha] = useState(false);
    const [step, setStep] = useState(1); // 1: Form, 2: Waiting confirmation
    const [pendingMagicToken, setPendingMagicToken] = useState(null);
    const [loading, setLoading] = useState(false);
    const { register, checkLoginStatus } = useContext(AuthContext);
    const { showToast } = useToast();
    const { theme } = useContext(ThemeContext);
    const { t } = useContext(LanguageContext);

    const HCAPTCHA_SITEKEY = (typeof window !== 'undefined' && window.EXPO_PUBLIC_HCAPTCHA_SITEKEY) || process.env.EXPO_PUBLIC_HCAPTCHA_SITEKEY;

    const { pollingTimeout, resetPolling } = useMagicLinkPolling(
        step === 2,
        pendingMagicToken,
        checkLoginStatus
    );

    const handleRegister = () => {
        setEmailError('');
        setUsernameError('');

        if (!email.trim() || !username.trim()) {
            showToast(t('please_fill_all'), "error");
            return;
        }

        if (!isValidUsername(username.trim())) {
            setUsernameError(t('username_invalid'));
            return;
        }

        if (!isValidEmail(email.trim())) {
            setEmailError(t('email_invalid'));
            return;
        }

        if (!acceptedTerms) {
            showToast(t('must_accept_terms'), "error");
            return;
        }

        setShowCaptcha(true);
    };

    const completeRegistration = async (token) => {
        setLoading(true);
        try {
            const mToken = await register(username.trim(), email.trim(), token);
            setPendingMagicToken(mToken);
            setStep(2);
            resetPolling();
            setShowCaptcha(false);
        } catch (e) {
            setShowCaptcha(false);
            const errMsg = e.message || t('registration_failed');
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
                        <Text style={styles.subtitle}>{t('register')}</Text>
                    </View>

                    {step === 1 ? (
                        <View style={styles.formCard}>
                            <Input
                                label={t('username')}
                                placeholder={t('username')}
                                value={username}
                                onChangeText={(text) => {
                                    setUsername(text);
                                    if (usernameError) setUsernameError('');
                                }}
                                autoCapitalize="none"
                                error={usernameError}
                                leftIcon={<Icon name="person-outline" size={20} color={theme.colors.subtext} />}
                            />

                            <Input
                                label={t('email')}
                                placeholder={t('email')}
                                value={email}
                                onChangeText={(text) => {
                                    setEmail(text);
                                    if (emailError) setEmailError('');
                                }}
                                autoCapitalize="none"
                                keyboardType="email-address"
                                error={emailError}
                                leftIcon={<Icon name="mail-outline" size={20} color={theme.colors.subtext} />}
                            />

                            <View style={styles.checkboxContainer}>
                                <Switch
                                    trackColor={{ false: theme.colors.border, true: theme.colors.primary }}
                                    thumbColor={acceptedTerms ? (theme.colors.buttonText || '#FFFFFF') : '#f4f3f4'}
                                    ios_backgroundColor={theme.colors.border}
                                    onValueChange={setAcceptedTerms}
                                    value={acceptedTerms}
                                />
                                <View style={styles.checkboxTextContainer}>
                                    <Text style={styles.checkboxLabel}>{t('i_agree_to')}{' '}</Text>
                                    <TouchableOpacity onPress={() => navigation.navigate('Privacy')} activeOpacity={0.7}>
                                        <Text style={styles.linkText}>{t('privacy_and_terms_title')}</Text>
                                    </TouchableOpacity>
                                </View>
                            </View>

                            <View style={styles.actionSection}>
                                <Button
                                    title={t('register')}
                                    onPress={handleRegister}
                                    disabled={!acceptedTerms || loading}
                                    loading={loading}
                                    variant="primary"
                                    size="lg"
                                    style={styles.fullWidthBtn}
                                />
                            </View>

                            <View style={styles.switchAuthRow}>
                                <Text style={styles.switchAuthPrompt}>Already have an account?</Text>
                                <Button
                                    title={t('login')}
                                    onPress={() => navigation.navigate('Login')}
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
                                {t('registration_successful')}!
                            </Text>
                            <Text style={styles.waitingDescription}>
                                {t('magic_link_sent', { email })}.{'\n\n'}
                                {t('check_email')}
                            </Text>

                            {pollingTimeout && (
                                <View style={styles.timeoutContainer}>
                                    <Text style={styles.timeoutText}>
                                        {t('polling_timeout') || 'Waiting for confirmation timed out.'}
                                    </Text>
                                    <Button
                                        title={t('resend_link') || 'Resend Link'}
                                        onPress={handleRegister}
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
                            {t('complete_challenge_register')}
                        </Text>
                        <View style={styles.captchaWrapper}>
                            <HCaptchaView
                                siteKey={HCAPTCHA_SITEKEY}
                                onVerify={completeRegistration}
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
    checkboxContainer: {
        flexDirection: 'row',
        alignItems: 'center',
        marginVertical: theme?.metrics?.spacing?.sm || 8,
    },
    checkboxTextContainer: {
        flexDirection: 'row',
        alignItems: 'center',
        flexWrap: 'wrap',
        marginLeft: 10,
        flex: 1,
    },
    checkboxLabel: {
        fontSize: 14,
        color: theme.colors.text,
    },
    linkText: {
        fontSize: 14,
        color: theme.colors.primary,
        fontWeight: '600',
        textDecorationLine: 'underline',
    },
    actionSection: {
        marginTop: theme?.metrics?.spacing?.md || 16,
    },
    fullWidthBtn: {
        width: '100%',
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

export default RegisterScreen;
