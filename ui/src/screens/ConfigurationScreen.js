import React, { useContext, useState } from 'react';
import { View, ScrollView, Text, Switch, StyleSheet, Button, TouchableOpacity, Modal, TextInput, Alert, Platform } from 'react-native';
import { ThemeContext } from '../context/ThemeContext';
import { AuthContext } from '../context/AuthContext';
import { LanguageContext } from '../context/LanguageContext';
import { useToast } from '../context/ToastContext';
import api from '../services/api';

const ConfigurationScreen = ({ navigation }) => {
    const { theme, isDark, toggleTheme } = useContext(ThemeContext);
    const { logout, username } = useContext(AuthContext);
    const { t, language, changeLanguage } = useContext(LanguageContext);
    const [deleteModalVisible, setDeleteModalVisible] = useState(false);
    const [confirmUsername, setConfirmUsername] = useState('');
    const [isDeleting, setIsDeleting] = useState(false);
    const [deleteContent, setDeleteContent] = useState(false);
    const { showToast } = useToast();

    const styles = createStyles(theme);

    const handleDeleteAccount = async () => {
        if (confirmUsername !== username) return;

        setIsDeleting(true);
        try {
            await api.delete('/auth/user', {
                data: { delete_poster_subresources: deleteContent }
            });
            setDeleteModalVisible(false);
            showToast(t('delete_account_success'), 'success');
            // On successful deletion, logout to clear state and redirect
            logout();
        } catch (error) {
            console.error('Failed to delete account:', error);
            setIsDeleting(false);
            showToast(t('delete_account_error'), 'error');
        }
    };

    return (
        <ScrollView style={styles.container} contentContainerStyle={styles.contentContainer}>
            <Text style={styles.header}>{t('settings')}</Text>

            <View style={styles.section}>
                <Text style={styles.sectionTitle}>{t('appearance')}</Text>
                <View style={styles.row}>
                    <Text style={styles.label}>{t('dark_mode')}</Text>
                    <Switch
                        trackColor={{ false: "#767577", true: theme.colors.primary }}
                        thumbColor={isDark ? "#f4f3f4" : "#f4f3f4"}
                        ios_backgroundColor="#3e3e3e"
                        onValueChange={toggleTheme}
                        value={isDark}
                    />
                </View>
            </View>

            <View style={styles.section}>
                <Text style={styles.sectionTitle}>Language</Text>
                <View style={styles.langRow}>
                    {['ca', 'es', 'en'].map((lang) => (
                        <TouchableOpacity
                            key={lang}
                            style={[
                                styles.langButton,
                                language === lang && styles.langButtonActive
                            ]}
                            onPress={() => changeLanguage(lang)}
                        >
                            <Text style={[
                                styles.langText,
                                language === lang && styles.langTextActive
                            ]}>
                                {lang.toUpperCase()}
                            </Text>
                        </TouchableOpacity>
                    ))}
                </View>
            </View>

            <View style={styles.section}>
                <Text style={styles.sectionTitle}>{t('account')}</Text>
                <TouchableOpacity style={styles.infoButton} onPress={() => navigation.navigate('Privacy')}>
                    <Text style={styles.infoButtonText}>{t('privacy_and_terms_title')}</Text>
                </TouchableOpacity>
                <TouchableOpacity style={styles.logoutButton} onPress={logout}>
                    <Text style={styles.logoutText}>{t('logout')}</Text>
                </TouchableOpacity>
            </View>

            <View style={[styles.section, styles.dangerZone]}>
                <Text style={[styles.sectionTitle, styles.dangerText]}>{t('danger_zone')}</Text>
                <TouchableOpacity
                    style={styles.deleteButton}
                    onPress={() => {
                        setConfirmUsername('');
                        setDeleteModalVisible(true);
                    }}
                >
                    <Text style={styles.deleteButtonText}>{t('delete_account')}</Text>
                </TouchableOpacity>
            </View>

            <Modal
                animationType="slide"
                transparent={true}
                visible={deleteModalVisible}
                onRequestClose={() => setDeleteModalVisible(false)}
            >
                <View style={styles.modalOverlay}>
                    <View style={styles.modalContent}>
                        <Text style={styles.modalTitle}>{t('delete_account_confirm_title')}</Text>

                        {/* 1. Warning Box inside Red Outline */}
                        <View style={styles.warningContainer}>
                            <Text style={styles.warningText}>
                                {t('delete_account_warning')}
                            </Text>
                        </View>

                        {/* 2. Toggle */}
                        <View style={styles.checkboxContainer}>
                            <Switch
                                trackColor={{ false: "#767577", true: theme.colors.error }}
                                thumbColor={deleteContent ? "#f4f3f4" : "#f4f3f4"}
                                ios_backgroundColor="#3e3e3e"
                                onValueChange={setDeleteContent}
                                value={deleteContent}
                            />
                            <Text style={styles.checkboxLabel}>{t('also_delete_content')}</Text>
                        </View>

                        {/* 3. Instruction */}
                        <Text style={styles.modalText}>
                            {t('delete_account_instruction', { username: username })}
                        </Text>

                        {/* 4. Input */}
                        <TextInput
                            style={styles.input}
                            placeholder={username}
                            placeholderTextColor={theme.colors.subtext}
                            value={confirmUsername}
                            onChangeText={setConfirmUsername}
                            autoCapitalize="none"
                        />

                        <View style={styles.modalButtons}>
                            <TouchableOpacity
                                style={[styles.modalButton, styles.cancelButton]}
                                onPress={() => setDeleteModalVisible(false)}
                                disabled={isDeleting}
                            >
                                <Text style={styles.cancelButtonText}>{t('cancel')}</Text>
                            </TouchableOpacity>
                            <TouchableOpacity
                                style={[
                                    styles.modalButton,
                                    styles.confirmDeleteButton,
                                    confirmUsername !== username && styles.disabledButton
                                ]}
                                onPress={handleDeleteAccount}
                                disabled={confirmUsername !== username || isDeleting}
                            >
                                <Text style={styles.confirmDeleteText}>
                                    {isDeleting ? t('deleting') : t('delete')}
                                </Text>
                            </TouchableOpacity>
                        </View>
                    </View>
                </View>
            </Modal>
        </ScrollView>
    );
};

const createStyles = (theme) => StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: theme.colors.background,
    },
    contentContainer: {
        padding: 20,
        paddingBottom: 50, // Add explicit bottom padding for scroll
    },
    header: {
        fontSize: 32,
        fontWeight: 'bold',
        marginBottom: 30,
        color: theme.colors.text,
    },
    section: {
        marginBottom: 30,
        backgroundColor: theme.colors.card,
        borderRadius: 12,
        padding: 15,
        shadowColor: "#000",
        shadowOffset: {
            width: 0,
            height: 2,
        },
        shadowOpacity: 0.1,
        shadowRadius: 3.84,
        elevation: 5,
    },
    sectionTitle: {
        fontSize: 18,
        fontWeight: '600',
        marginBottom: 15,
        color: theme.colors.subtext,
        textTransform: 'uppercase',
        letterSpacing: 1,
    },
    row: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        paddingVertical: 10,
    },
    label: {
        fontSize: 18,
        color: theme.colors.text,
    },
    langRow: {
        flexDirection: 'row',
        justifyContent: 'space-around',
        paddingVertical: 10,
    },
    langButton: {
        paddingVertical: 8,
        paddingHorizontal: 20,
        borderRadius: 20,
        borderWidth: 1,
        borderColor: theme.colors.border,
        backgroundColor: theme.colors.background,
    },
    langButtonActive: {
        backgroundColor: theme.colors.primary,
        borderColor: theme.colors.primary,
    },
    langText: {
        color: theme.colors.text,
        fontWeight: '600',
    },
    langTextActive: {
        color: 'white',
    },
    logoutButton: {
        backgroundColor: theme.colors.error,
        padding: 15,
        borderRadius: 8,
        alignItems: 'center',
    },
    infoButton: {
        backgroundColor: theme.colors.inputBackground,
        padding: 15,
        borderRadius: 8,
        alignItems: 'center',
        marginBottom: 10,
        borderWidth: 1,
        borderColor: theme.colors.border,
    },
    infoButtonText: {
        color: theme.colors.text,
        fontSize: 16,
        fontWeight: 'bold',
    },
    logoutText: {
        color: 'white',
        fontSize: 16,
        fontWeight: 'bold',
    },
    dangerZone: {
        borderColor: theme.colors.error,
        borderWidth: 1,
    },
    dangerText: {
        color: theme.colors.error,
    },
    deleteButton: {
        backgroundColor: 'transparent',
        padding: 15,
        borderRadius: 8,
        alignItems: 'center',
        borderWidth: 1,
        borderColor: theme.colors.error,
    },
    deleteButtonText: {
        color: theme.colors.error,
        fontSize: 16,
        fontWeight: 'bold',
    },
    modalOverlay: {
        flex: 1,
        backgroundColor: 'rgba(0, 0, 0, 0.5)',
        justifyContent: 'center',
        alignItems: 'center',
        padding: 20,
    },
    modalContent: {
        backgroundColor: theme.colors.card,
        borderRadius: 12,
        padding: 20,
        width: '100%',
        maxWidth: 400,
        alignItems: 'center',
    },
    modalTitle: {
        fontSize: 20,
        fontWeight: 'bold',
        color: theme.colors.text,
        marginBottom: 10,
    },
    modalText: {
        fontSize: 16,
        color: theme.colors.text,
        textAlign: 'center',
        marginBottom: 20,
    },
    input: {
        width: '100%',
        backgroundColor: theme.colors.inputBackground,
        color: theme.colors.text,
        padding: 15,
        borderRadius: 8,
        marginBottom: 20,
        borderWidth: 1,
        borderColor: theme.colors.border,
    },
    warningContainer: {
        borderWidth: 1,
        borderColor: theme.colors.error,
        borderRadius: 8,
        padding: 10,
        marginBottom: 20,
        width: '100%',
        backgroundColor: 'transparent',
    },
    warningText: {
        color: theme.colors.error,
        fontWeight: 'bold',
        textAlign: 'center',
        fontSize: 16,
    },
    checkboxContainer: {
        flexDirection: 'row',
        alignItems: 'center',
        marginBottom: 20,
        width: '100%',
        justifyContent: 'flex-start', // Ensure left align like text? Or center? Modal is alignItems: center.
    },
    checkboxLabel: {
        marginLeft: 10,
        color: theme.colors.text,
        fontSize: 16,
        flex: 1,
        textAlign: 'left', // Ensure text aligns left relative to switch
    },
    modalButtons: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        width: '100%',
        gap: 10,
    },
    modalButton: {
        flex: 1,
        padding: 15,
        borderRadius: 8,
        alignItems: 'center',
    },
    cancelButton: {
        backgroundColor: theme.colors.inputBackground,
    },
    cancelButtonText: {
        color: theme.colors.text,
        fontWeight: 'bold',
    },
    confirmDeleteButton: {
        backgroundColor: theme.colors.error,
    },
    confirmDeleteText: {
        color: 'white',
        fontWeight: 'bold',
    },
    disabledButton: {
        opacity: 0.5,
    },
});

export default ConfigurationScreen;
