import React, { useContext, useState, useEffect } from 'react';
import { View, Text, TextInput, Button, StyleSheet, ActivityIndicator } from 'react-native';
import { AuthContext } from '../context/AuthContext';

const LoginScreen = ({ navigation }) => {
    const [identifier, setIdentifier] = useState('');
    const [step, setStep] = useState(1); // 1: Email, 2: Done
    const [pendingMagicToken, setPendingMagicToken] = useState(null);
    const { requestLogin, checkLoginStatus } = useContext(AuthContext);
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        let interval;
        if (step === 2 && pendingMagicToken) {
            interval = setInterval(async () => {
                const confirmed = await checkLoginStatus(pendingMagicToken);
                if (confirmed) {
                    clearInterval(interval);
                }
            }, 2000);
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
        setLoading(true);
        try {
            const mToken = await requestLogin(identifier);
            setPendingMagicToken(mToken);
            setStep(2);
        } catch (e) {
            alert('Failed to request magic link');
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
