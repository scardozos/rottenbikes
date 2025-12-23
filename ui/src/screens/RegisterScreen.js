import React, { useState, useEffect, useContext } from 'react';
import { View, Text, TextInput, Button, StyleSheet, Alert, Modal, Platform } from 'react-native';
import { WebView } from 'react-native-webview';
import { AuthContext } from '../context/AuthContext';
import { ThemeContext } from '../context/ThemeContext';
import HCaptchaView from '../components/HCaptchaView';
import { useToast } from '../context/ToastContext';

const RegisterScreen = ({ navigation }) => {
  const [username, setUsername] = useState('');
  const [usernameError, setUsernameError] = useState('');
  const [email, setEmail] = useState('');
  const [emailError, setEmailError] = useState('');
  const [showCaptcha, setShowCaptcha] = useState(false);
  const [step, setStep] = useState(1); // 1: Form, 2: Waiting
  const [pendingMagicToken, setPendingMagicToken] = useState(null);
  const { register, checkLoginStatus } = useContext(AuthContext);
  const { showToast } = useToast();
  const { theme } = useContext(ThemeContext);

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

  const handleRegister = () => {
    setEmailError('');
    setUsernameError('');
    if (!email || !username) {
      showToast("Please fill in all fields", "error");
      return;
    }

    // Validate username (alphanumeric and dots only)
    const usernameRegex = /^[a-zA-Z0-9.]+$/;
    if (!usernameRegex.test(username)) {
      setUsernameError('Username can only contain letters, numbers and dots');
      return;
    }

    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!emailRegex.test(email)) {
      setEmailError('Please enter a valid email address');
      return;
    }
    setShowCaptcha(true);
  };

  const completeRegistration = async (token) => {
    console.log("completeRegistration called with token length:", token ? token.length : 0);
    try {
      console.log("Sending registration request for:", username, email);
      const mToken = await register(username, email, token);
      setPendingMagicToken(mToken);
      setStep(2);
      setShowCaptcha(false);
    } catch (e) {
      console.log("Registration failed error:", e);
      setShowCaptcha(false);

      const errMsg = e.message || "Registration failed. Please try again.";
      showToast(errMsg, 'error');
    }
  };

  const styles = createStyles(theme);

  return (
    <View style={styles.container}>
      <Text style={styles.title}>Register</Text>

      {step === 1 ? (
        <>
          <TextInput
            style={[styles.input, usernameError ? styles.inputError : null]}
            placeholder="Username"
            placeholderTextColor={theme.colors.placeholder}
            value={username}
            onChangeText={(text) => {
              setUsername(text);
              if (usernameError) setUsernameError('');
            }}
            autoCapitalize="none"
          />
          {usernameError ? <Text style={styles.errorText}>{usernameError}</Text> : null}

          <TextInput
            style={[styles.input, emailError ? styles.inputError : null]}
            placeholder="Email"
            placeholderTextColor={theme.colors.placeholder}
            value={email}
            onChangeText={(text) => {
              setEmail(text);
              if (emailError) setEmailError('');
            }}
            autoCapitalize="none"
            keyboardType="email-address"
          />
          {emailError ? <Text style={styles.errorText}>{emailError}</Text> : null}

          <Button title="Register" onPress={handleRegister} color={theme.colors.primary} />

          <Modal visible={showCaptcha} animationType="slide">
            <View style={{ flex: 1, backgroundColor: theme.colors.background, paddingVertical: 50 }}>
              <Text style={{ textAlign: 'center', fontSize: 18, marginBottom: 20, color: theme.colors.text }}>
                Complete the challenge to register
              </Text>
              <HCaptchaView
                siteKey={HCAPTCHA_SITEKEY}
                onVerify={completeRegistration}
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
            Registration successful!{'\n\n'}
            We've sent a magic link to {email}.{'\n\n'}
            Check your email and click the link to confirm your account and log in automatically.
          </Text>
          <Button title="Back" onPress={() => setStep(1)} color={theme.colors.subtext} />
        </>
      )}
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
  inputError: {
    borderColor: theme.colors.error,
  },
  errorText: {
    color: theme.colors.error,
    fontSize: 12,
    marginBottom: 8,
    marginTop: -8,
  },
});

export default RegisterScreen;
