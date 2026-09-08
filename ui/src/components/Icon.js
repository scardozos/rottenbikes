import React from 'react';
import { Ionicons } from '@expo/vector-icons';
import { useTheme } from '../context/ThemeContext';

const Icon = ({ name, size = 20, color, style }) => {
    const { theme } = useTheme();
    const resolvedColor = color || theme?.colors?.text || '#000000';

    return (
        <Ionicons
            name={name}
            size={size}
            color={resolvedColor}
            style={style}
        />
    );
};

export default Icon;
