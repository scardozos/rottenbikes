import React, { useEffect, useMemo } from 'react';
import { Text, TouchableOpacity, StyleSheet, Animated, Platform } from 'react-native';
import { useTheme } from '../context/ThemeContext';
import Icon from './Icon';

const Toast = ({ message, type = 'success', onClose, duration = 4000 }) => {
    const { theme } = useTheme();
    const opacity = useMemo(() => new Animated.Value(0), []);

    useEffect(() => {
        const anim = Animated.sequence([
            Animated.timing(opacity, {
                toValue: 1,
                duration: 300,
                // Native driver is not supported on web for some properties or requires config
                useNativeDriver: Platform.OS !== 'web',
            }),
            Animated.delay(duration),
            Animated.timing(opacity, {
                toValue: 0,
                duration: 300,
                useNativeDriver: Platform.OS !== 'web',
            }),
        ]);

        anim.start(({ finished }) => {
            if (finished && onClose) onClose();
        });

        return () => anim.stop();
    }, [opacity, duration, onClose]);

    let backgroundColor = theme.colors.success;
    if (type === 'error') backgroundColor = theme.colors.danger || theme.colors.error;
    if (type === 'info') backgroundColor = theme.colors.primary;

    const textColor = theme.colors.buttonText || '#FFFFFF';

    return (
        <Animated.View 
            style={[
                styles.container,
                {
                    opacity,
                    backgroundColor,
                    borderRadius: theme.metrics?.radii?.md || 8,
                },
                theme.shadows?.md,
            ]}
            accessibilityLiveRegion="assertive"
            accessibilityRole="alert"
        >
            <Text style={[styles.message, { color: textColor }]}>{message}</Text>
            <TouchableOpacity 
                onPress={onClose} 
                style={styles.closeButton}
                accessibilityRole="button"
                accessibilityLabel="Close notification"
            >
                <Icon name="close" size={20} color={textColor} />
            </TouchableOpacity>
        </Animated.View>
    );
};

const styles = StyleSheet.create({
    container: {
        position: 'absolute',
        bottom: Platform.OS === 'web' ? 20 : 50,
        left: 20,
        right: 20,
        padding: 15,
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'space-between',
        zIndex: 9999,
        elevation: 5,
    },
    message: {
        fontSize: 16,
        fontWeight: '600',
        flex: 1,
    },
    closeButton: {
        marginLeft: 10,
        padding: 4,
    },
});

export default Toast;
