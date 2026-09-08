import React, { useState, useEffect, useContext } from 'react';
import { View, Text, StyleSheet, Switch, Alert } from 'react-native';
import Button from '../components/Button';
import Input from '../components/Input';
import ErrorBoundary from '../components/ErrorBoundary';
import api from '../services/api';
import { useToast } from '../context/ToastContext';
import { ThemeContext } from '../context/ThemeContext';
import { LanguageContext } from '../context/LanguageContext';

import { useSession } from '../context/SessionContext';
import { isValidNumericalId, isValidHashId } from '../utils/validation';

const CreateBikeScreen = ({ route, navigation }) => {
    const { initialNumericalId, initialHashId } = route.params || {};
    const { showToast } = useToast();
    const { theme } = useContext(ThemeContext);
    const { t } = useContext(LanguageContext);
    const { validateBike } = useSession();

    const [numericalId, setNumericalId] = useState(initialNumericalId || '');
    const [hashId, setHashId] = useState(initialHashId || '');
    const [isElectric, setIsElectric] = useState(false);
    const [loading, setLoading] = useState(false);

    // Ensure we update state if params change while component is mounted (though usually it's a new mount)
    useEffect(() => {
        if (initialNumericalId) setNumericalId(initialNumericalId);
        if (initialHashId) setHashId(initialHashId);
    }, [initialNumericalId, initialHashId]);

    const handleSubmit = async () => {
        setLoading(true);

        const numId = String(numericalId).trim();
        if (!isValidNumericalId(numId)) {
            showToast(t('invalid_numerical_id'), "error");
            setLoading(false);
            return;
        }

        const hId = hashId.trim();
        if (!isValidHashId(hId)) {
            showToast(t('invalid_hash_id'), "error");
            setLoading(false);
            return;
        }

        try {
            const response = await api.post('/bikes', {
                numerical_id: numId,
                hash_id: hId === '' ? null : hId,
                is_electric: isElectric
            });
            showToast(t('success'), "success");

            // Validate the newly created bike so we can review it
            validateBike(response.data.numerical_id);

            // Navigate to CreateReview (replacing CreateBike screen)
            navigation.replace('CreateReview', { bikeId: response.data.numerical_id });
        } catch (e) {

            console.error(e);
            const errMsg = e.response?.data?.error || t('error');
            showToast(errMsg, "error");
        } finally {
            setLoading(false);
        }
    };

    const styles = React.useMemo(() => createStyles(theme), [theme]);

    return (
        <ErrorBoundary>
            <CreateBikeContent {...{ theme, styles, numericalId, setNumericalId, hashId, setHashId, isElectric, setIsElectric, loading, handleSubmit, t }} />
        </ErrorBoundary>
    );
};

const CreateBikeContent = ({ theme, styles, numericalId, setNumericalId, hashId, setHashId, isElectric, setIsElectric, loading, handleSubmit, t }) => (
    <View style={styles.container}>
        <Text style={styles.title}>{t('add_new_bike')}</Text>
        <View style={styles.formCard}>
            <Input
                label={t('numerical_id')}
                placeholder={t('numerical_id_placeholder')}
                value={numericalId ? String(numericalId) : ''}
                onChangeText={setNumericalId}
                keyboardType="numeric"
            />

            <Input
                label={t('hash_id_input_label') || 'Hash ID'}
                placeholder={t('hash_id_placeholder')}
                value={hashId}
                onChangeText={setHashId}
            />

            <View style={styles.switchContainer}>
                <Text style={styles.switchText}>{t('electric_bike')}</Text>
                <Switch
                    value={isElectric}
                    onValueChange={setIsElectric}
                    trackColor={{ false: theme.colors.border, true: theme.colors.primary }}
                    thumbColor={isElectric ? (theme.colors.buttonText || '#FFFFFF') : '#f4f3f4'}
                />
            </View>

            <Button
                title={t('create_bike_btn')}
                onPress={handleSubmit}
                disabled={loading}
                loading={loading}
                variant="primary"
                size="lg"
                style={{ marginTop: 8 }}
            />
        </View>
    </View>
);

const createStyles = (theme) => StyleSheet.create({
    container: {
        flex: 1,
        padding: theme?.metrics?.spacing?.lg || 16,
        backgroundColor: theme.colors.background,
    },
    title: {
        fontSize: theme?.typography?.h2?.fontSize || 24,
        fontWeight: 'bold',
        marginBottom: theme?.metrics?.spacing?.lg || 16,
        color: theme.colors.text,
    },
    formCard: {
        backgroundColor: theme.colors.card,
        borderRadius: theme?.metrics?.radii?.lg || 16,
        padding: theme?.metrics?.spacing?.lg || 16,
        borderWidth: 1,
        borderColor: theme.colors.border,
        ...(theme?.shadows?.sm || {}),
    },
    switchContainer: {
        flexDirection: 'row',
        alignItems: 'center',
        marginVertical: 12,
        justifyContent: 'space-between',
    },
    switchText: {
        fontSize: 15,
        fontWeight: '600',
        color: theme.colors.text,
    },
});

export default CreateBikeScreen;
