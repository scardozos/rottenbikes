import React, { useState, useEffect, useContext } from 'react';
import { View, Text, TextInput, Button, StyleSheet, Switch, ScrollView, Modal, TouchableOpacity } from 'react-native';
import api from '../services/api';
import { useToast } from '../context/ToastContext';
import { ThemeContext } from '../context/ThemeContext';
import { LanguageContext } from '../context/LanguageContext';
import { useSession } from '../context/SessionContext';
import { Scanner } from '../components/Scanner';

const UpdateBikeScreen = ({ route, navigation }) => {
    const { bike } = route.params || {};
    const { showToast } = useToast();
    const { theme } = useContext(ThemeContext);
    const { t } = useContext(LanguageContext);
    const { validatedBikeId } = useSession();

    const [hashId, setHashId] = useState(bike?.hash_id || '');
    const [isElectric, setIsElectric] = useState(bike?.is_electric || false);
    const [loading, setLoading] = useState(false);
    const [showScanner, setShowScanner] = useState(false);
    const [confirmed, setConfirmed] = useState(false);

    // Security Check
    useEffect(() => {
        if (!bike || Number(validatedBikeId) !== Number(bike.numerical_id)) {
            showToast(t('unauthorized_update'), "error");
            navigation.goBack();
        }
    }, [bike, validatedBikeId]);

    const handleSubmit = async () => {
        setLoading(true);
        try {
            await api.put(`/bikes/${bike.numerical_id}`, {
                hash_id: hashId.trim() === '' ? null : hashId,
                is_electric: isElectric
            });
            showToast(t('update_success'), "success");
            navigation.goBack(); // Return to details
        } catch (e) {
            console.error(e);
            const errMsg = e.response?.data?.error || t('error');
            showToast(errMsg, "error");
        } finally {
            setLoading(false);
        }
    };

    if (!bike) return null;

    const styles = createStyles(theme);

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
                    disabled={loading || !confirmed}
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

