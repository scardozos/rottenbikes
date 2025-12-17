import React, { useContext, useState } from 'react';
import { View, Text, TextInput, Button, StyleSheet, ActivityIndicator } from 'react-native';
import { AuthContext } from '../context/AuthContext';

const LoginScreen = ({ navigation }) => {
    const [email, setEmail] = useState('');
    const [token, setToken] = useState('');
    const [step, setStep] = useState(1); // 1: Email, 2: Token
    const { requestLogin, completeLogin } = useContext(AuthContext);
    const [loading, setLoading] = useState(false);

    const handleRequestLink = async () => {
        if (!email) {
            alert("Please enter an email");
            return;
        }
        setLoading(true);
        try {
            await requestLogin(email);
            setStep(2);
            alert("Magic link sent! Check server console.");
        } catch (e) {
            alert('Failed to request magic link');
        } finally {
            setLoading(false);
        }
    };

    const handleCompleteLogin = async () => {
        if (!token) {
            alert("Please enter the token");
            return;
        }
        setLoading(true);
        try {
            await completeLogin(token);
        } catch (e) {
            alert('Login failed');
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
                        placeholder="Email"
                        value={email}
                        onChangeText={setEmail}
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
                    <Text style={{ marginBottom: 10, textAlign: 'center' }}>
                        Magic link sent to {email}.{'\n'}
                        Check server logs, click the link (or curl it), and copy the 'api_token' from the response JSON here.
                    </Text>
                    <TextInput
                        style={styles.input}
                        placeholder="Paste API Token"
                        value={token}
                        onChangeText={setToken}
                        autoCapitalize="none"
                    />
                    {loading ? (
                        <ActivityIndicator size="large" />
                    ) : (
                        <Button title="Complete Login" onPress={handleCompleteLogin} />
                    )}
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
