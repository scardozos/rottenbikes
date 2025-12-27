import React, { useState, useEffect, useContext } from 'react';
import { View, Text, TextInput, Button, StyleSheet, Switch, Alert } from 'react-native';
import api from '../services/api';
import { useToast } from '../context/ToastContext';
import { ThemeContext } from '../context/ThemeContext';
import { LanguageContext } from '../context/LanguageContext';

import { useSession } from '../context/SessionContext';

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
        try {
            const response = await api.post('/bikes', {
                numerical_id: parseInt(numericalId),
                hash_id: hashId.trim() === '' ? null : hashId,
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

    const styles = createStyles(theme);

    return (
        <ErrorBoundary>
            <CreateBikeContent {...{ theme, styles, numericalId, setNumericalId, hashId, setHashId, isElectric, setIsElectric, loading, handleSubmit, t }} />
        </ErrorBoundary>
    );
};

const CreateBikeContent = ({ theme, styles, numericalId, setNumericalId, hashId, setHashId, isElectric, setIsElectric, loading, handleSubmit, t }) => (
    <View style={styles.container}>
        <Text style={styles.title}>{t('add_new_bike')}</Text>

        <TextInput
            placeholder={t('numerical_id_placeholder')}
            placeholderTextColor={theme.colors.placeholder}
            style={styles.input}
            value={numericalId ? String(numericalId) : ''}
            onChangeText={setNumericalId}
            keyboardType="numeric"
        />

        <TextInput
            placeholder={t('hash_id_placeholder')}
            placeholderTextColor={theme.colors.placeholder}
            style={styles.input}
            value={hashId}
            onChangeText={setHashId}
        />

        <View style={styles.switchContainer}>
            <Text style={styles.text}>{t('electric_bike')}</Text>
            <Switch
                value={isElectric}
                onValueChange={setIsElectric}
                trackColor={{ false: "#767577", true: theme.colors.primary }}
                thumbColor={isElectric ? "#f4f3f4" : "#f4f3f4"}
            />
        </View>

        <Button title={t('create_bike_btn')} onPress={handleSubmit} disabled={loading} color={theme.colors.primary} />
    </View>
);

class ErrorBoundary extends React.Component {
    constructor(props) {
        super(props);
        this.state = { hasError: false, error: null };
    }
    static getDerivedStateFromError(error) {
        return { hasError: true, error };
    }
    componentDidCatch(error, errorInfo) {
        console.error("CreateBike ErrorBoundary:", error, errorInfo);
    }
    render() {
        if (this.state.hasError) {
            return <View style={{ flex: 1, justifyContent: 'center', alignItems: 'center' }}><Text style={{ color: 'red' }}>Error: {this.state.error?.toString()}</Text></View>;
        }
        return this.props.children;
    }
}

const createStyles = (theme) => StyleSheet.create({
    container: { flex: 1, padding: 20, backgroundColor: theme.colors.background },
    title: { fontSize: 24, marginBottom: 20, color: theme.colors.text },
    input: {
        height: 40,
        borderColor: theme.colors.border,
        borderWidth: 1,
        marginBottom: 12,
        paddingHorizontal: 8,
        borderRadius: 4,
        color: theme.colors.text,
        backgroundColor: theme.colors.inputBackground,
        fontSize: 16
    },
    switchContainer: {
        flexDirection: 'row',
        alignItems: 'center',
        marginBottom: 20,
        justifyContent: 'space-between'
    },
    text: {
        color: theme.colors.text
    }
});

export default CreateBikeScreen;
