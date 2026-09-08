import React, { forwardRef } from 'react';
import { View, Text, TextInput, StyleSheet } from 'react-native';
import { useTheme } from '../context/ThemeContext';

const Input = forwardRef(({
    label,
    error,
    leftIcon,
    rightIcon,
    containerStyle,
    inputStyle,
    labelStyle,
    errorStyle,
    style,
    ...props
}, ref) => {
    const { theme } = useTheme();

    return (
        <View style={[styles.container, containerStyle]}>
            {label && (
                <Text style={[
                    styles.label,
                    { color: theme.colors.text },
                    labelStyle,
                ]}>
                    {label}
                </Text>
            )}
            <View style={[
                styles.inputWrapper,
                {
                    backgroundColor: theme.colors.inputBackground,
                    borderColor: error ? theme.colors.error : theme.colors.border,
                    borderRadius: theme.metrics.radii.md,
                },
                style,
            ]}>
                {leftIcon && <View style={styles.leftIcon}>{leftIcon}</View>}
                <TextInput
                    ref={ref}
                    placeholderTextColor={theme.colors.placeholder}
                    style={[
                        styles.input,
                        { color: theme.colors.text },
                        inputStyle,
                    ]}
                    {...props}
                />
                {rightIcon && <View style={styles.rightIcon}>{rightIcon}</View>}
            </View>
            {error && (
                <Text style={[
                    styles.errorText,
                    { color: theme.colors.error },
                    errorStyle,
                ]}>
                    {error}
                </Text>
            )}
        </View>
    );
});

Input.displayName = 'Input';

const styles = StyleSheet.create({
    container: {
        marginBottom: 12,
        width: '100%',
    },
    label: {
        fontSize: 14,
        fontWeight: '600',
        marginBottom: 6,
    },
    inputWrapper: {
        flexDirection: 'row',
        alignItems: 'center',
        borderWidth: 1,
        minHeight: 48,
        paddingHorizontal: 12,
    },
    input: {
        flex: 1,
        height: '100%',
        paddingVertical: 10,
        fontSize: 16,
    },
    leftIcon: {
        marginRight: 8,
    },
    rightIcon: {
        marginLeft: 8,
    },
    errorText: {
        fontSize: 12,
        marginTop: 4,
    },
});

export default Input;
