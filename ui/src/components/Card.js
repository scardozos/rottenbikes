import React from 'react';
import { View, StyleSheet } from 'react-native';
import { useTheme } from '../context/ThemeContext';

const Card = ({
    children,
    style,
    shadow = true,
    bordered = true,
    padding = 'md',
    overflowHidden = false,
    ...rest
}) => {
    const { theme } = useTheme();

    const paddingValue = theme?.metrics?.spacing?.[padding] !== undefined
        ? theme.metrics.spacing[padding]
        : (theme?.metrics?.spacing?.md || 16);

    const shadowStyle = shadow ? (theme?.shadows?.sm || {}) : {};

    return (
        <View
            style={[
                styles.card,
                {
                    backgroundColor: theme.colors.card,
                    borderRadius: theme.metrics?.radii?.lg || 16,
                    padding: paddingValue,
                },
                overflowHidden && { overflow: 'hidden' },
                bordered && {
                    borderWidth: 1,
                    borderColor: theme.colors.border,
                },
                shadowStyle,
                style,
            ]}
            {...rest}
        >
            {children}
        </View>
    );
};

const styles = StyleSheet.create({
    card: {
        width: '100%',
    },
});

export default Card;
