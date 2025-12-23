import React, { useEffect, useContext, useState } from 'react';
import { View, Text, ActivityIndicator, StyleSheet, Button, Platform } from 'react-native';
import { AuthContext } from '../context/AuthContext';
import { ThemeContext } from '../context/ThemeContext';

const ConfirmLoginScreen = ({ route, navigation }) => {
    const { token } = route.params || {};
    // Extract origin from query parameters (if it exists)
    // React Navigation puts query params in route.params as well
    const { origin } = route.params || {};

    const { completeLogin, confirmAttempt, userToken } = useContext(AuthContext);
    const { theme } = useContext(ThemeContext);
    const [status, setStatus] = useState('loading'); // loading, ready, success, error
    const [errorMsg, setErrorMsg] = useState('');
    const [isCrossDevice, setIsCrossDevice] = useState(false);
    const [submitting, setSubmitting] = useState(false);

    useEffect(() => {
        // Only trigger automatic redirection if we just successfully logged in on this device
        if (status === 'success' && !isCrossDevice) {
            navigation.navigate('Home');
        }
    }, [status, isCrossDevice, navigation]);

    useEffect(() => {
        if (!token) {
            setStatus('error');
            setErrorMsg('No token provided');
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
                navigation.navigate('Home');
            }
        } catch (e) {
            setStatus('error');
            setErrorMsg('Invalid or expired token');
        } finally {
            setSubmitting(false);
        }
    };

    const styles = createStyles(theme);

    if (status === 'ready') {
        return (
            <View style={styles.container}>
                <Text style={styles.title}>Finish Logging In</Text>
                <Text style={styles.text}>Click below to complete your login request.</Text>
                <View style={{ marginTop: 20, width: '100%' }}>
                    <Button title="Confirm Login" onPress={handleConfirm} disabled={submitting} color={theme.colors.primary} />
                </View>
                <View style={{ marginTop: 10 }}>
                    <Button title="Cancel" onPress={() => navigation.navigate('Login')} color={theme.colors.subtext} disabled={submitting} />
                </View>
            </View>
        );
    }

    if (status === 'loading') {
        return (
            <View style={styles.container}>
                <ActivityIndicator size="large" color={theme.colors.primary} />
                <Text style={styles.text}>Confirming your login...</Text>
            </View>
        );
    }

    if (status === 'error') {
        return (
            <View style={styles.container}>
                <Text style={[styles.text, { color: theme.colors.error }]}>{errorMsg}</Text>
                <Button title="Back to Login" onPress={() => navigation.navigate('Login')} color={theme.colors.primary} />
            </View>
        );
    }

    if (status === 'success') {
        return (
            <View style={styles.container}>
                <Text style={[styles.text, styles.successText]}>
                    {isCrossDevice ? 'Login Confirmed!' : 'Login Confirmed! Success!'}
                </Text>
                <Text style={styles.subText}>
                    {isCrossDevice
                        ? 'Your mobile app will log you in automatically. You can close this window.'
                        : 'Redirecting you to the app...'}
                </Text>
                {isCrossDevice && (
                    <View style={{ marginTop: 20 }}>
                        <Button
                            title="Continue to App"
                            onPress={() => {
                                navigation.reset({
                                    index: 0,
                                    routes: [{ name: 'Home' }],
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
