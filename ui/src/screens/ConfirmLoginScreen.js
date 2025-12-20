import React, { useEffect, useContext, useState } from 'react';
import { View, Text, ActivityIndicator, StyleSheet, Button, Platform } from 'react-native';
import { AuthContext } from '../context/AuthContext';

const ConfirmLoginScreen = ({ route, navigation }) => {
    const { token } = route.params || {};
    // Extract origin from query parameters (if it exists)
    // React Navigation puts query params in route.params as well
    const { origin } = route.params || {};

    const { completeLogin, confirmAttempt } = useContext(AuthContext);
    const [status, setStatus] = useState('loading'); // loading, success, error
    const [errorMsg, setErrorMsg] = useState('');
    const [isCrossDevice, setIsCrossDevice] = useState(false);

    useEffect(() => {
        if (!token) {
            setStatus('error');
            setErrorMsg('No token provided');
            return;
        }

        const confirmToken = async () => {
            try {
                // Determine if we should only confirm (cross-device) or fully login
                const shouldOnlyConfirm = origin === 'mobile' && Platform.OS === 'web';

                if (shouldOnlyConfirm) {
                    await confirmAttempt(token);
                    setIsCrossDevice(true);
                } else {
                    await completeLogin(token);
                }
                setStatus('success');
            } catch (e) {
                setStatus('error');
                setErrorMsg('Invalid or expired token');
            }
        };

        confirmToken();
    }, [token, origin]);

    if (status === 'loading') {
        return (
            <View style={styles.container}>
                <ActivityIndicator size="large" />
                <Text style={styles.text}>Confirming your login...</Text>
            </View>
        );
    }

    if (status === 'error') {
        return (
            <View style={styles.container}>
                <Text style={[styles.text, { color: 'red' }]}>{errorMsg}</Text>
                <Button title="Back to Login" onPress={() => navigation.navigate('Login')} />
            </View>
        );
    }

    return (
        <View style={styles.container}>
            <Text style={[styles.text, styles.successText]}>
                {isCrossDevice ? 'Login Confirmed!' : 'Login Confirmed! Success!'}
            </Text>
            <Text style={styles.subText}>
                {isCrossDevice
                    ? 'Your mobile app will log you in automatically. You can close this window.'
                    : 'You can now close this window or wait to be redirected.'}
            </Text>
            <View style={{ marginTop: 20 }}>
                <Button
                    title="Continue to App"
                    onPress={() => {
                        navigation.reset({
                            index: 0,
                            routes: [{ name: 'Home' }],
                        });
                    }}
                />
            </View>
        </View>
    );
};

const styles = StyleSheet.create({
    container: { flex: 1, justifyContent: 'center', alignItems: 'center', padding: 20 },
    text: { fontSize: 18, marginTop: 10, textAlign: 'center' },
    successText: { color: 'green', fontWeight: 'bold', fontSize: 24 },
    subText: { fontSize: 14, color: 'gray', marginTop: 10 }
});

export default ConfirmLoginScreen;
