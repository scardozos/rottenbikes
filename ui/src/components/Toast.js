import React, { useEffect, useRef } from 'react';
import { View, Text, TouchableOpacity, StyleSheet, Animated, Platform } from 'react-native';

const Toast = ({ message, type = 'success', onClose, duration = 4000 }) => {
    const opacity = useRef(new Animated.Value(0)).current;

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

    let backgroundColor = '#2ecc71';
    if (type === 'error') backgroundColor = '#e74c3c';
    if (type === 'info') backgroundColor = '#3498db';

    return (
        <Animated.View style={[styles.container, { opacity, backgroundColor }]}>
            <Text style={styles.message}>{message}</Text>
            <TouchableOpacity onPress={onClose} style={styles.closeButton}>
                <Text style={styles.closeText}>✕</Text>
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
        borderRadius: 8,
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'space-between',
        zIndex: 9999,
        // Elevation for Android
        elevation: 5,
        // Shadow for iOS/Web
        // shadowColor: '#000',
        // shadowOffset: { width: 0, height: 2 },
        // shadowOpacity: 0.25,
        // shadowRadius: 3.84,
        boxShadow: '0px 2px 3.84px rgba(0, 0, 0, 0.25)',
    },
    message: {
        color: '#fff',
        fontSize: 16,
        fontWeight: '600',
        flex: 1,
    },
    closeButton: {
        marginLeft: 10,
        padding: 5,
    },
    closeText: {
        color: '#fff',
        fontSize: 18,
        fontWeight: 'bold',
    },
});

export default Toast;
