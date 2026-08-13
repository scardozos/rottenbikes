import React, { useContext } from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { ThemeContext } from '../context/ThemeContext';
import { LanguageContext } from '../context/LanguageContext';

const HealthTrendBadge = ({ aggregates, subcategory = 'overall' }) => {
    const { theme } = useContext(ThemeContext);
    const { t } = useContext(LanguageContext);

    if (!aggregates || aggregates.length === 0) return null;

    const overall = aggregates.find(a => a.subcategory === subcategory && a.window === 'overall')?.average_rating;
    const w2 = aggregates.find(a => a.subcategory === subcategory && a.window === '2w')?.average_rating;
    const w1 = aggregates.find(a => a.subcategory === subcategory && a.window === '1w')?.average_rating;

    // We need at least w1 and w2 to compare trends
    if (w1 == null || w2 == null) return null;

    let trend = 'stable';
    let icon = '➡️';
    let color = theme.colors.subtext;
    let label = t('trend_stable') || 'Stable';

    // Simple trend logic
    if (w1 > w2 + 0.2) {
        trend = 'improving';
        icon = '↗️';
        color = '#2ecc71'; // Green
        label = t('trend_improving') || 'Improving';
    } else if (w1 < w2 - 0.2) {
        trend = 'degrading';
        icon = '↘️';
        color = '#e74c3c'; // Red
        label = t('trend_degrading') || 'Degrading';
    }

    const styles = createStyles(theme);

    return (
        <View style={[styles.badge, { borderColor: color, backgroundColor: color + '1A' }]}>
            <Text style={styles.icon}>{icon}</Text>
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
