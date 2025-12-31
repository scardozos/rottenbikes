import React, { useState, useEffect, useContext } from 'react';
import { View, Text, TextInput, Button, StyleSheet, Switch, ScrollView, Modal, ActivityIndicator } from 'react-native';
import api from '../services/api';
import { useToast } from '../context/ToastContext';
import { ThemeContext } from '../context/ThemeContext';
import { LanguageContext } from '../context/LanguageContext';
import { useSession } from '../context/SessionContext';
import { Scanner } from '../components/Scanner';



const UpdateBikeScreen = ({ route, navigation }) => {
    const { bikeId } = route.params || {};
    const { showToast } = useToast();
    const { theme } = useContext(ThemeContext);
    const { t } = useContext(LanguageContext);
    const { validatedBikeId } = useSession();

    // Initialize styles first
    const styles = createStyles(theme);


    const [bike, setBike] = useState(null);
    const [hashId, setHashId] = useState('');
    const [isElectric, setIsElectric] = useState(false);
    const [loading, setLoading] = useState(true);
    const [submitting, setSubmitting] = useState(false);
    const [showScanner, setShowScanner] = useState(false);
    const [confirmed, setConfirmed] = useState(false);

    // Initial Fetch & Security Check
    useEffect(() => {
        const checkAndFetch = async () => {
            // Security Check
            if (!bikeId || Number(validatedBikeId) !== Number(bikeId)) {
                showToast(t('unauthorized_update'), "error");
                navigation.goBack();
                return;
            }

            try {
                const res = await api.get(`/bikes/${bikeId}/details`);
                const details = res.data;
                setBike(details);
                setHashId(details.hash_id || '');
                setIsElectric(details.is_electric || false);
            } catch (e) {
                console.error("Failed to fetch bike for update", e);
                showToast(t('error_fetching_bike'), "error");
                navigation.goBack();
            } finally {
                setLoading(false);
            }
        };

        checkAndFetch();
    }, [bikeId, validatedBikeId]);

    const handleSubmit = async () => {
        setSubmitting(true);

        const hId = hashId.trim();
        if (hId !== '' && !/^[a-zA-Z0-9]+$/.test(hId)) {
            showToast(t('invalid_hash_id'), "error");
            setSubmitting(false);
            return;
        }

        try {
            await api.put(`/bikes/${bikeId}`, {
                hash_id: hId === '' ? null : hId,
                is_electric: isElectric
            });
            showToast(t('update_success'), "success");
            navigation.goBack(); // Return to details
        } catch (e) {
            console.error(e);
            const errMsg = e.response?.data?.error || t('error');
            showToast(errMsg, "error");
        } finally {
            setSubmitting(false);
        }
    };

    if (loading) {
        return (
            <View style={[styles.container, styles.center]}>
                <ActivityIndicator size="large" color={theme.colors.primary} />
            </View>
        );
    }

    if (!bike) return null;

    if (!bike) return null;

    return (
        <View style={styles.container}>

            <ScrollView contentContainerStyle={styles.scrollContent}>
                <Text style={styles.title}>{t('update_bike_title')}</Text>

                <Text style={styles.label}>{t('numerical_id')}: {bike.numerical_id}</Text>

                <View style={styles.labelRow}>
                    <Text style={styles.label}>{t('hash_id_input_label')}</Text>
                    <Button title={t('scan')} onPress={() => setShowScanner(true)} color={theme.colors.primary} />
                </View>

                <TextInput
                    placeholder={t('hash_id_placeholder')}
                    placeholderTextColor={theme.colors.placeholder}
                    style={styles.input}
                    value={hashId || ''}
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

                <View style={styles.checkboxContainer}>
                    <Switch
                        value={confirmed}
                        onValueChange={setConfirmed}
                        trackColor={{ false: "#767577", true: theme.colors.primary }}
                    />
                    <Text style={[styles.text, styles.checkboxLabel]}>{t('update_confirm_checkbox')}</Text>
                </View>

                <Button
                    title={t('update_bike_btn')}
                    onPress={handleSubmit}
                    disabled={submitting || !confirmed}
                    color={theme.colors.primary}
                />
            </ScrollView>

            <Modal visible={showScanner} animationType="slide">
                <Scanner
                    onScan={(data) => {
                        setHashId(data);
                        setShowScanner(false);
                    }}
                    onClose={() => setShowScanner(false)}
                    theme={theme}
                    t={t}
                />
            </Modal>


        </View>
    );
};

const createStyles = (theme) => StyleSheet.create({
    container: { flex: 1, backgroundColor: theme.colors.background },
    center: { justifyContent: 'center', alignItems: 'center' },
    scrollContent: { padding: 20 },
    title: { fontSize: 24, marginBottom: 20, color: theme.colors.text },
    label: { fontSize: 16, marginBottom: 5, color: theme.colors.text, fontWeight: 'bold' },
    labelRow: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginBottom: 5 },
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
    checkboxContainer: {
        flexDirection: 'row',
        alignItems: 'center',
        marginBottom: 20,
        padding: 10,
        backgroundColor: theme.colors.inputBackground,
        borderRadius: 8
    },
    checkboxLabel: {
        marginLeft: 10,
        flex: 1,
        fontSize: 14
    },
    text: {
        color: theme.colors.text
    }
});

export default UpdateBikeScreen;


