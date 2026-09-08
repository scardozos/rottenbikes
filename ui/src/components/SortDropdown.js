import React, { useContext } from 'react';
import { View, Text, TouchableOpacity, StyleSheet, ScrollView } from 'react-native';
import { ThemeContext } from '../context/ThemeContext';
import { LanguageContext } from '../context/LanguageContext';
import Icon from './Icon';

const SORT_OPTIONS = [
    { value: 'recent', labelKey: 'sort_recent', icon: 'time-outline' },
    { value: 'rating', labelKey: 'sort_rating', icon: 'star-outline' },
    { value: 'most_reviewed', labelKey: 'sort_most_reviewed', icon: 'chatbubbles-outline' },
];

const SortDropdown = ({ selectedSort, onSortChange }) => {
    const { theme } = useContext(ThemeContext);
    const { t } = useContext(LanguageContext);

    const styles = React.useMemo(() => createStyles(theme), [theme]);

    return (
        <View style={styles.container}>
            <ScrollView
                horizontal
                showsHorizontalScrollIndicator={false}
                contentContainerStyle={styles.scrollContent}
                contentInsetAdjustmentBehavior="never"
                automaticallyAdjustContentInsets={false}
            >
                {SORT_OPTIONS.map((option) => {
                    const isSelected = selectedSort === option.value;
                    return (
                        <TouchableOpacity
                            key={option.value}
                            onPress={() => onSortChange(option.value)}
                            style={[
                                styles.chip,
                                isSelected ? styles.activeChip : styles.inactiveChip,
                            ]}
                            activeOpacity={0.7}
                            accessibilityRole="button"
                            accessibilityState={{ selected: isSelected }}
                        >
                            <Icon
                                name={option.icon}
                                size={15}
                                color={isSelected ? theme.colors.buttonText : theme.colors.subtext}
                                style={styles.chipIcon}
                            />
                            <Text
                                style={[
                                    styles.chipText,
                                    isSelected ? styles.activeChipText : styles.inactiveChipText,
                                ]}
                            >
                                {t(option.labelKey) || option.labelKey}
                            </Text>
                        </TouchableOpacity>
                    );
                })}
            </ScrollView>
        </View>
    );
};

const createStyles = (theme) => StyleSheet.create({
    container: {
        marginBottom: theme?.metrics?.spacing?.md || 12,
    },
    scrollContent: {
        flexDirection: 'row',
        gap: theme?.metrics?.spacing?.sm || 8,
        paddingVertical: 2,
    },
    chip: {
        flexDirection: 'row',
        alignItems: 'center',
        paddingVertical: 8,
        paddingHorizontal: 14,
        borderRadius: theme?.metrics?.radii?.pill || 9999,
        borderWidth: 1,
    },
    activeChip: {
        backgroundColor: theme.colors.primary,
        borderColor: theme.colors.primary,
    },
    inactiveChip: {
        backgroundColor: theme.colors.card,
        borderColor: theme.colors.border,
    },
    chipIcon: {
        marginRight: 6,
    },
    chipText: {
        fontSize: 14,
        fontWeight: '600',
    },
    activeChipText: {
        color: theme.colors.buttonText,
    },
    inactiveChipText: {
        color: theme.colors.subtext,
    },
});

export default SortDropdown;
