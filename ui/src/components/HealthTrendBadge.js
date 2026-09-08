import React, { useContext } from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { ThemeContext } from '../context/ThemeContext';
import { LanguageContext } from '../context/LanguageContext';
import { computeTrend } from '../utils/ratings';

import Icon from './Icon';

const HealthTrendBadge = ({ aggregates, subcategory = 'overall' }) => {
    const { theme } = useContext(ThemeContext);
    const { t } = useContext(LanguageContext);
    const styles = React.useMemo(() => createStyles(theme), [theme]);

    if (!aggregates || aggregates.length === 0) return null;

    const w2 = aggregates.find(a => a.subcategory === subcategory && a.window === '2w')?.average_rating;
    const w1 = aggregates.find(a => a.subcategory === subcategory && a.window === '1w')?.average_rating;

    // We need at least w1 and w2 to compare trends
    if (w1 == null || w2 == null) return null;

    const trend = computeTrend(w1, w2);
    let iconName = 'arrow-forward';
    let color = theme.colors.subtext;
    let label = t('trend_stable') || 'Stable';

    if (trend === 'improving') {
        iconName = 'trending-up';
        color = theme.colors.success;
        label = t('trend_improving') || 'Improving';
    } else if (trend === 'degrading') {
        iconName = 'trending-down';
        color = theme.colors.danger || theme.colors.error;
        label = t('trend_degrading') || 'Degrading';
    }

    return (
        <View style={[styles.badge, { borderColor: color, backgroundColor: color + '1A' }]}>
            <Icon name={iconName} size={14} color={color} style={styles.icon} />
            <Text style={[styles.label, { color }]}>{label}</Text>
        </View>
    );
};

const createStyles = (theme) => StyleSheet.create({
    badge: {
        flexDirection: 'row',
        alignItems: 'center',
        paddingHorizontal: 8,
        paddingVertical: 4,
        borderRadius: 12,
        borderWidth: 1,
        alignSelf: 'flex-start',
        marginTop: 5,
    },
    icon: {
        fontSize: 12,
        marginRight: 4,
    },
    label: {
        fontSize: 12,
        fontWeight: 'bold',
    }
});

export default HealthTrendBadge;
