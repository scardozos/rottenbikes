import React, { useEffect, useContext, useState } from 'react';
import { View, Text, ActivityIndicator, StyleSheet, Button, Platform } from 'react-native';
import { AuthContext } from '../context/AuthContext';
import { ThemeContext } from '../context/ThemeContext';
import { LanguageContext } from '../context/LanguageContext';

const ConfirmLoginScreen = ({ route, navigation }) => {
    const { token } = route.params || {};
    // Extract origin from query parameters (if it exists)
    // React Navigation puts query params in route.params as well
    const { origin } = route.params || {};

    const { completeLogin, confirmAttempt, userToken } = useContext(AuthContext);
    const { theme } = useContext(ThemeContext);
    const { t } = useContext(LanguageContext);
    const [status, setStatus] = useState('loading'); // loading, ready, success, error
    const [errorMsg, setErrorMsg] = useState('');
    const [isCrossDevice, setIsCrossDevice] = useState(false);
    const [submitting, setSubmitting] = useState(false);

    useEffect(() => {
        // Only trigger automatic redirection if we just successfully logged in on this device
        if (status === 'success' && !isCrossDevice) {
            navigation.replace('Main');
        }
    }, [status, isCrossDevice, navigation]);

    useEffect(() => {
        if (!token) {
            setStatus('error');
            setErrorMsg(t('no_token'));
        } else {
            setStatus('ready');
        }
    }, [token]);

    const handleConfirm = async () => {
        setSubmitting(true);
        setStatus('loading');
        try {
            // Determine if we should only confirm (cross-device) or fully login
            const shouldOnlyConfirm = origin === 'mobile' && Platform.OS === 'web';

            if (shouldOnlyConfirm) {
                await confirmAttempt(token);
                setIsCrossDevice(true);
                setStatus('success');
            } else {
                await completeLogin(token);
                setStatus('success');
                // Automatically redirect to Home for same-device logins
                navigation.replace('Main');
            }
        } catch (e) {
            setStatus('error');
            setErrorMsg(t('invalid_token'));
        } finally {
            setSubmitting(false);
        }
    };

    const styles = createStyles(theme);

    if (status === 'ready') {
        return (
            <View style={styles.container}>
                <Text style={styles.title}>{t('finish_login')}</Text>
                <Text style={styles.text}>{t('click_to_complete')}</Text>
                <View style={{ marginTop: 20, width: '100%' }}>
                    <Button title={t('confirm_login_btn')} onPress={handleConfirm} disabled={submitting} color={theme.colors.primary} />
                </View>
                <View style={{ marginTop: 10 }}>
                    <Button title={t('cancel')} onPress={() => navigation.navigate('Login')} color={theme.colors.subtext} disabled={submitting} />
                </View>
            </View>
        );
    }

    if (status === 'loading') {
        return (
            <View style={styles.container}>
                <ActivityIndicator size="large" color={theme.colors.primary} />
                <Text style={styles.text}>{t('confirming_login')}</Text>
            </View>
        );
    }

    if (status === 'error') {
        return (
            <View style={styles.container}>
                <Text style={[styles.text, { color: theme.colors.error }]}>{errorMsg}</Text>
                <Button title={t('back_to_login')} onPress={() => navigation.navigate('Login')} color={theme.colors.primary} />
            </View>
        );
    }

    if (status === 'success') {
        return (
            <View style={styles.container}>
                <Text style={[styles.text, styles.successText]}>
                    {isCrossDevice ? t('login_confirmed') : t('login_confirmed_success')}
                </Text>
                <Text style={styles.subText}>
                    {isCrossDevice
                        ? t('mobile_auto_login')
                        : t('redirecting')}
                </Text>
                {isCrossDevice && (
                    <View style={{ marginTop: 20 }}>
                        <Button
                            title={t('continue_to_app')}
                            onPress={() => {
                                navigation.reset({
                                    index: 0,
                                    routes: [{ name: 'Main' }],
                                });
                            }}
                            color={theme.colors.primary}
                        />
                    </View>
                )}
            </View>
        );
    }

    return null;
};

const createStyles = (theme) => StyleSheet.create({
    container: { flex: 1, justifyContent: 'center', alignItems: 'center', padding: 20, backgroundColor: theme.colors.background },
    title: { fontSize: 24, fontWeight: 'bold', marginBottom: 20, textAlign: 'center', color: theme.colors.text },
    text: { fontSize: 18, marginTop: 10, textAlign: 'center', color: theme.colors.text },
    successText: { color: theme.colors.success, fontWeight: 'bold', fontSize: 24 },
    subText: { fontSize: 14, color: theme.colors.subtext, marginTop: 10 }
});

export default ConfirmLoginScreen;
