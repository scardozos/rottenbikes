import React, { useContext, useState, useEffect } from 'react';
import { View, Text, TextInput, Button, StyleSheet, ActivityIndicator, Platform, Modal, Alert } from 'react-native';
import HCaptchaView from '../components/HCaptchaView';
import { AuthContext } from '../context/AuthContext';

const LoginScreen = ({ navigation }) => {
    const [identifier, setIdentifier] = useState('');
    const [step, setStep] = useState(1); // 1: Email, 2: Done
    const [pendingMagicToken, setPendingMagicToken] = useState(null);
    const { requestLogin, checkLoginStatus } = useContext(AuthContext);
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
            alert("Please enter your email or username");
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
            const errMsg = e.response?.data || 'Failed to request magic link';
            if (Platform.OS === 'web') window.alert(errMsg);
            else Alert.alert("Error", errMsg);
        } finally {
            setLoading(false);
        }
    };

    return (
        <View style={styles.container}>
            <Text style={styles.title}>RottenBikes Login</Text>

            {step === 1 ? (
                <>
                    <TextInput
                        style={styles.input}
                        placeholder="Email or Username"
                        value={identifier}
                        onChangeText={setIdentifier}
                        autoCapitalize="none"
                        keyboardType="email-address"
                    />
                    {loading ? (
                        <ActivityIndicator size="large" />
                    ) : (
                        <Button title="Get Magic Link" onPress={handleRequestLink} />
                    )}

                    <Modal visible={showCaptcha} animationType="slide">
                        <View style={{ flex: 1, backgroundColor: 'white', paddingVertical: 50 }}>
                            <Text style={{ textAlign: 'center', fontSize: 18, marginBottom: 20 }}>
                                Complete the challenge to login
                            </Text>
                            <HCaptchaView
                                siteKey={HCAPTCHA_SITEKEY}
                                onVerify={completeRequestLink}
                                onExpired={() => {
                                    setShowCaptcha(false);
                                    Alert.alert("Error", "Captcha expired");
                                }}
                                onError={() => {
                                    setShowCaptcha(false);
                                    Alert.alert("Error", "Captcha failed");
                                }}
                            />
                            <View style={{ marginTop: 40, paddingHorizontal: 20 }}>
                                <Button title="Cancel" onPress={() => setShowCaptcha(false)} color="red" />
                            </View>
                        </View>
                    </Modal>
                </>
            ) : (
                <>
                    <Text style={{ marginBottom: 20, textAlign: 'center', fontSize: 16 }}>
                        Magic link requested for {identifier}!{'\n\n'}
                        Check the server console for the link and click it to log in.
                    </Text>
                    <Button title="Back" onPress={() => setStep(1)} color="gray" />
                </>
            )}

            <View style={{ marginTop: 20 }}>
                <Button
                    title="Register"
                    onPress={() => navigation.navigate('Register')}
                    color="gray"
                />
            </View>
        </View>
    );
};

const styles = StyleSheet.create({
    container: { flex: 1, justifyContent: 'center', padding: 20 },
    title: { fontSize: 24, marginBottom: 20, textAlign: 'center' },
    input: {
        height: 40,
        borderColor: 'gray',
        borderWidth: 1,
        marginBottom: 12,
        paddingHorizontal: 8,
        borderRadius: 4
    },
});

export default LoginScreen;
