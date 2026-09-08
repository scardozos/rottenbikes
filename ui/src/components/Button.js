import React from 'react';
import { TouchableOpacity, Text, StyleSheet, ActivityIndicator, View } from 'react-native';
import { useTheme } from '../context/ThemeContext';

const Button = ({
    title,
    children,
    onPress,
    variant = 'primary',
    size = 'md',
    disabled = false,
    loading = false,
    icon,
    color,
    style,
    textStyle,
    accessibilityLabel,
    ...rest
}) => {
    const { theme } = useTheme();

    // Map color prop for backward-compatibility with native Button
    let effectiveVariant = variant;
    let customBgColor = null;

    if (color && variant === 'primary') {
        if (color === 'red' || color === theme?.colors?.error || color === theme?.colors?.danger) {
            effectiveVariant = 'danger';
        } else if (color === theme?.colors?.subtext || color === 'gray') {
            effectiveVariant = 'ghost';
        } else {
            customBgColor = color;
        }
    }

    const getVariantStyles = () => {
        switch (effectiveVariant) {
            case 'secondary':
                return {
                    container: { backgroundColor: theme.colors.secondary },
                    text: { color: theme.colors.buttonText },
                };
            case 'danger':
                return {
                    container: { backgroundColor: theme.colors.danger || theme.colors.error },
                    text: { color: theme.colors.buttonText },
                };
            case 'ghost':
                return {
                    container: {
                        backgroundColor: 'transparent',
                        borderWidth: 1,
                        borderColor: theme.colors.border,
                    },
                    text: { color: theme.colors.text },
                };
            case 'primary':
            default:
                return {
                    container: { backgroundColor: customBgColor || theme.colors.primary },
                    text: { color: theme.colors.buttonText || '#FFFFFF' },
                };
        }
    };

    const getSizeStyles = () => {
        const spacing = theme?.metrics?.spacing || { xs: 4, sm: 8, md: 12, lg: 16, xl: 24 };
        switch (size) {
            case 'sm':
                return {
                    container: {
                        paddingVertical: spacing.xs + 2, // ~6px
                        paddingHorizontal: spacing.md, // 12px
                        minHeight: 34,
                    },
                    text: { fontSize: 13 },
                };
            case 'lg':
                return {
                    container: {
                        paddingVertical: spacing.md + 2, // ~14px
                        paddingHorizontal: spacing.xl, // 24px
                        minHeight: 52,
                    },
                    text: { fontSize: 18 },
                };
            case 'md':
            default:
                return {
                    container: {
                        paddingVertical: spacing.sm + 2, // ~10px
                        paddingHorizontal: spacing.lg, // 16px
                        minHeight: 44,
                    },
                    text: { fontSize: 15 },
                };
        }
    };

    const variantStyles = getVariantStyles();
    const sizeStyles = getSizeStyles();
    const borderRadius = theme?.metrics?.radii?.md || 12;

    const content = children || (
        <Text
            style={[
                styles.text,
                variantStyles.text,
                sizeStyles.text,
                textStyle,
            ]}
        >
            {title}
        </Text>
    );

    return (
        <TouchableOpacity
            accessibilityRole="button"
            accessibilityLabel={accessibilityLabel || title}
            disabled={disabled || loading}
            onPress={onPress}
            activeOpacity={0.7}
            style={[
                styles.base,
                { borderRadius },
                variantStyles.container,
                sizeStyles.container,
                disabled && styles.disabled,
                style,
            ]}
            {...rest}
        >
            {loading ? (
                <ActivityIndicator
                    size="small"
                    color={variantStyles.text.color}
                />
            ) : (
                <View style={styles.contentRow}>
                    {icon && <View style={styles.iconContainer}>{icon}</View>}
                    {content}
                </View>
            )}
        </TouchableOpacity>
    );
};

const styles = StyleSheet.create({
    base: {
        alignItems: 'center',
        justifyContent: 'center',
        flexDirection: 'row',
    },
    contentRow: {
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'center',
    },
    iconContainer: {
        marginRight: 8,
    },
    text: {
        fontWeight: '600',
        textAlign: 'center',
    },
    disabled: {
        opacity: 0.5,
    },
});

export default Button;
