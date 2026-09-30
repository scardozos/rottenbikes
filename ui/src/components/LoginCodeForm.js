import React, { useContext, useState } from 'react';
import { View, StyleSheet } from 'react-native';
import Button from './Button';
import Input from './Input';
import Icon from './Icon';
import { ThemeContext } from '../context/ThemeContext';
import { LanguageContext } from '../context/LanguageContext';
import { LOGIN_CODE_LENGTH, normalizeLoginCode, isCompleteLoginCode } from '../utils/loginCode';

// Entry for the 6-digit code from the login email, for logging in on the
// device that asked for the email. onSubmit(code) logs in, or throws an
// error whose status is 400 when the code is wrong or expired.
const LoginCodeForm = ({ onSubmit }) => {
    const { theme } = useContext(ThemeContext);
    const { t } = useContext(LanguageContext);
    const [code, setCode] = useState('');
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState('');

    const submit = async (value) => {
        if (!isCompleteLoginCode(value) || submitting) return;
        setSubmitting(true);
        setError('');
        try {
            // On success the app switches to the logged-in screens.
            await onSubmit(value);
        } catch (e) {
            setError(e.status === 400 ? t('login_code_invalid') : t('error'));
            setSubmitting(false);
        }
    };

    const handleChange = (text) => {
        const next = normalizeLoginCode(text);
        setCode(next);
        if (error) setError('');
        // Submit as soon as the code is complete (typed, pasted or autofilled).
        if (isCompleteLoginCode(next) && next !== code) {
            submit(next);
        }
    };

    return (
        <View style={styles.container}>
            <Input
                label={t('login_code_label')}
                placeholder={t('login_code_placeholder')}
                value={code}
                onChangeText={handleChange}
                keyboardType="number-pad"
                textContentType="oneTimeCode"
                autoComplete="one-time-code"
                maxLength={LOGIN_CODE_LENGTH + 2}
                error={error || undefined}
                inputStyle={styles.codeInput}
                leftIcon={<Icon name="keypad-outline" size={20} color={theme.colors.subtext} />}
            />
            <Button
                title={t('login_with_code')}
                onPress={() => submit(code)}
                disabled={!isCompleteLoginCode(code) || submitting}
                loading={submitting}
                variant="primary"
                size="md"
                style={styles.button}
            />
        </View>
    );
};

const styles = StyleSheet.create({
    container: {
        width: '100%',
        marginBottom: 12,
    },
    codeInput: {
        fontSize: 20,
        letterSpacing: 6,
    },
    button: {
        width: '100%',
    },
});

export default LoginCodeForm;
