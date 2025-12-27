import React, { useContext } from 'react';
import { View, Text, Switch, StyleSheet, Button, TouchableOpacity } from 'react-native';
import { ThemeContext } from '../context/ThemeContext';
import { AuthContext } from '../context/AuthContext';

const ConfigurationScreen = () => {
    const { theme, isDark, toggleTheme } = useContext(ThemeContext);
    const { logout } = useContext(AuthContext);

    const styles = createStyles(theme);

    return (
        <View style={styles.container}>
            <Text style={styles.header}>Settings</Text>

            <View style={styles.section}>
                <Text style={styles.sectionTitle}>Appearance</Text>
                <View style={styles.row}>
                    <Text style={styles.label}>Dark Mode</Text>
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
                <Text style={styles.sectionTitle}>Account</Text>
                <TouchableOpacity style={styles.logoutButton} onPress={logout}>
                    <Text style={styles.logoutText}>Logout</Text>
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
