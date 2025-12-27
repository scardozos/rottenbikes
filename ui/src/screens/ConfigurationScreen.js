import React, { useContext } from 'react';
import { View, Text, Switch, StyleSheet, Button, TouchableOpacity } from 'react-native';
import { ThemeContext } from '../context/ThemeContext';
import { AuthContext } from '../context/AuthContext';
import { LanguageContext } from '../context/LanguageContext';

const ConfigurationScreen = () => {
    const { theme, isDark, toggleTheme } = useContext(ThemeContext);
    const { logout } = useContext(AuthContext);
    const { t, language, changeLanguage } = useContext(LanguageContext);

    const styles = createStyles(theme);

    return (
        <View style={styles.container}>
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
                <TouchableOpacity style={styles.logoutButton} onPress={logout}>
                    <Text style={styles.logoutText}>{t('logout')}</Text>
                </TouchableOpacity>
            </View>
        </View>
    );
};

const createStyles = (theme) => StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: theme.colors.background,
        padding: 20,
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
        color: 'white', // Assuming primary is dark or contrasting
    },
    logoutButton: {
        backgroundColor: theme.colors.error,
        padding: 15,
        borderRadius: 8,
        alignItems: 'center',
    },
    logoutText: {
        color: 'white',
        fontSize: 16,
        fontWeight: 'bold',
    },
});

export default ConfigurationScreen;
