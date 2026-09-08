import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { useTheme } from '../context/ThemeContext';

const Badge = ({
    label,
    children,
    icon,
    variant = 'default',
    size = 'md',
    style,
    textStyle,
    ...rest
}) => {
    const { theme } = useTheme();

    const getVariantStyles = () => {
        switch (variant) {
            case 'primary':
                return {
                    bg: theme.colors.primary + '20', // 20% opacity
                    text: theme.colors.primary,
                };
            case 'secondary':
                return {
                    bg: theme.colors.secondary + '20',
                    text: theme.colors.secondary,
                };
            case 'success':
                return {
                    bg: theme.colors.success + '20',
                    text: theme.colors.success,
                };
            case 'warning':
                return {
                    bg: theme.colors.warning + '25',
                    text: theme.colors.warning,
                };
            case 'danger':
                return {
                    bg: (theme.colors.danger || theme.colors.error) + '20',
                    text: theme.colors.danger || theme.colors.error,
                };
            case 'default':
            default:
                return {
                    bg: theme.colors.border,
                    text: theme.colors.subtext,
                };
        }
    };

    const variantStyles = getVariantStyles();
    const isSm = size === 'sm';

    return (
        <View
            style={[
                styles.badge,
                {
                    backgroundColor: variantStyles.bg,
                    borderRadius: theme.metrics?.radii?.pill || 9999,
                    paddingHorizontal: isSm ? 6 : 10,
                    paddingVertical: isSm ? 2 : 4,
                },
                style,
            ]}
            {...rest}
        >
            {icon && <View style={styles.icon}>{icon}</View>}
            {label ? (
                <Text
                    style={[
                        styles.text,
                        {
                            color: variantStyles.text,
                            fontSize: isSm ? 11 : 13,
                        },
                        textStyle,
                    ]}
                >
                    {label}
                </Text>
            ) : children}
        </View>
    );
};

const styles = StyleSheet.create({
    badge: {
        flexDirection: 'row',
        alignItems: 'center',
        alignSelf: 'flex-start',
    },
    icon: {
        marginRight: 4,
    },
    text: {
        fontWeight: '600',
    },
});

export default Badge;
