import React from 'react';
import { StyleSheet, ScrollView } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useTheme } from '../context/ThemeContext';

const Screen = ({
    children,
    style,
    contentContainerStyle,
    scrollable = false,
    edges = ['top', 'bottom', 'left', 'right'],
    ...rest
}) => {
    const { theme } = useTheme();

    if (scrollable) {
        return (
            <SafeAreaView
                edges={edges}
                style={[styles.container, { backgroundColor: theme.colors.background }, style]}
                {...rest}
            >
                <ScrollView
                    contentContainerStyle={[styles.scrollContent, contentContainerStyle]}
                    keyboardShouldPersistTaps="handled"
                >
                    {children}
                </ScrollView>
            </SafeAreaView>
        );
    }

    return (
        <SafeAreaView
            edges={edges}
            style={[styles.container, { backgroundColor: theme.colors.background }, style]}
            {...rest}
        >
            {children}
        </SafeAreaView>
    );
};

const styles = StyleSheet.create({
    container: {
        flex: 1,
    },
    scrollContent: {
        flexGrow: 1,
    },
});

export default Screen;
