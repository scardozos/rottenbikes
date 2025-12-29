import React, { useState, useEffect, useContext } from 'react';
import { View, Text, TextInput, Button, StyleSheet, Alert, Modal, Platform, Switch, TouchableOpacity } from 'react-native';
import { WebView } from 'react-native-webview';
import { AuthContext } from '../context/AuthContext';
import { ThemeContext } from '../context/ThemeContext';
import { LanguageContext } from '../context/LanguageContext';
import HCaptchaView from '../components/HCaptchaView';
import { useToast } from '../context/ToastContext';

const RegisterScreen = ({ navigation }) => {
  const [username, setUsername] = useState('');
  const [usernameError, setUsernameError] = useState('');
  const [email, setEmail] = useState('');
  const [emailError, setEmailError] = useState('');
  const [acceptedTerms, setAcceptedTerms] = useState(false);
  const [showCaptcha, setShowCaptcha] = useState(false);
  const [step, setStep] = useState(1); // 1: Form, 2: Waiting
  const [pendingMagicToken, setPendingMagicToken] = useState(null);
  const { register, checkLoginStatus } = useContext(AuthContext);
  const { showToast } = useToast();
  const { theme } = useContext(ThemeContext);
  const { t } = useContext(LanguageContext);

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
      showToast(t('please_fill_all'), "error");
      return;
    }

    // Validate username (alphanumeric and dots only)
    const usernameRegex = /^[a-zA-Z0-9.]+$/;
    if (!usernameRegex.test(username)) {
      setUsernameError(t('username_invalid'));
      return;
    }

    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!emailRegex.test(email)) {
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

      const errMsg = e.message || t('registration_failed');
      showToast(errMsg, 'error');
    }
  };

  const styles = createStyles(theme);

  return (
    <View style={styles.container}>
      <Text style={styles.title}>{t('register')}</Text>

      {step === 1 ? (
        <>
          <TextInput
            style={[styles.input, usernameError ? styles.inputError : null]}
            placeholder={t('username')}
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
            placeholder={t('email')}
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

          <View style={styles.checkboxContainer}>
            <Switch
              trackColor={{ false: "#767577", true: theme.colors.primary }}
              thumbColor={acceptedTerms ? "#f4f3f4" : "#f4f3f4"}
              ios_backgroundColor="#3e3e3e"
              onValueChange={setAcceptedTerms}
              value={acceptedTerms}
            />
            <View style={styles.checkboxTextContainer}>
              <Text style={styles.checkboxLabel}>{t('i_agree_to')}{' '}</Text>
              <TouchableOpacity onPress={() => navigation.navigate('Privacy')}>
                <Text style={styles.linkText}>{t('privacy_and_terms_title')}</Text>
              </TouchableOpacity>
            </View>
          </View>

          <TouchableOpacity
            style={[
              styles.registerButton,
              !acceptedTerms && styles.registerButtonDisabled
            ]}
            onPress={handleRegister}
          >
            <Text style={styles.registerButtonText}>{t('register')}</Text>
          </TouchableOpacity>

          <Modal visible={showCaptcha} animationType="slide">
            <View style={{ flex: 1, backgroundColor: theme.colors.background, paddingVertical: 50 }}>
              <Text style={{ textAlign: 'center', fontSize: 18, marginBottom: 20, color: theme.colors.text }}>
                {t('complete_challenge_register')}
              </Text>
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
              <View style={{ marginTop: 40, paddingHorizontal: 20 }}>
                <Button title={t('cancel')} onPress={() => setShowCaptcha(false)} color={theme.colors.error} />
              </View>
            </View>
          </Modal>
        </>
      ) : (
        <>
          <Text style={{ marginBottom: 20, textAlign: 'center', fontSize: 16, color: theme.colors.text }}>
            {t('registration_successful')}!{'\n\n'}
            {t('magic_link_sent', { email })}.{'\n\n'}
            {t('check_email')}
          </Text>
          <Button title={t('back')} onPress={() => setStep(1)} color={theme.colors.subtext} />
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
  checkboxContainer: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 20,
  },
  checkboxTextContainer: {
    flexDirection: 'row',
    alignItems: 'center',
    flexWrap: 'wrap',
    marginLeft: 8,
  },
  checkboxLabel: {
    color: theme.colors.text,
  },
  linkText: {
    color: theme.colors.primary,
    textDecorationLine: 'underline',
  },
  registerButton: {
    backgroundColor: theme.colors.primary,
    padding: 12,
    borderRadius: 4,
    alignItems: 'center',
    marginBottom: 20,
  },
  registerButtonDisabled: {
    opacity: 0.5,
  },
  registerButtonText: {
    color: 'white', // Assuming primary button text is white; adjust if theme differs
    fontSize: 16,
    fontWeight: 'bold',
  },
});

export default RegisterScreen;
