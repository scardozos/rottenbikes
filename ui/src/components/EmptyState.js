import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { useTheme } from '../context/ThemeContext';
import Icon from './Icon';

const EmptyState = ({
    icon = 'file-tray-outline',
    title,
    description,
    action,
    style,
}) => {
    const { theme } = useTheme();

    return (
        <View style={[styles.container, style]}>
            <View style={[styles.iconContainer, { backgroundColor: theme.colors.card }]}>
                {typeof icon === 'string' ? (
                    <Icon name={icon} size={48} color={theme.colors.subtext} />
                ) : (
                    icon
                )}
            </View>
            {title && (
                <Text style={[styles.title, { color: theme.colors.text }]}>
                    {title}
                </Text>
            )}
            {description && (
                <Text style={[styles.description, { color: theme.colors.subtext }]}>
                    {description}
                </Text>
            )}
            {action && <View style={styles.actionContainer}>{action}</View>}
        </View>
    );
};

const styles = StyleSheet.create({
    container: {
        alignItems: 'center',
        justifyContent: 'center',
        padding: 32,
        width: '100%',
    },
    iconContainer: {
        width: 80,
        height: 80,
        borderRadius: 40,
        alignItems: 'center',
        justifyContent: 'center',
        marginBottom: 16,
    },
    title: {
        fontSize: 18,
        fontWeight: 'bold',
        textAlign: 'center',
        marginBottom: 8,
    },
    description: {
        fontSize: 14,
        textAlign: 'center',
        lineHeight: 20,
        marginBottom: 16,
    },
    actionContainer: {
        marginTop: 8,
    },
});

export default EmptyState;
