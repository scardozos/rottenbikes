import React, { useContext } from 'react';
import { View, Text, TouchableOpacity, StyleSheet } from 'react-native';
import { ThemeContext } from '../context/ThemeContext';
import { LanguageContext } from '../context/LanguageContext';
import { AuthContext } from '../context/AuthContext';
import { getRelativeTime } from '../utils/time';
import Icon from './Icon';

const ReviewItem = React.memo(({ item, isExpanded, onToggle, onEdit, showBikeId }) => {
    const { theme } = useContext(ThemeContext);
    const { t } = useContext(LanguageContext);
    const { userId } = useContext(AuthContext);

    const subRatings = item.ratings ? Object.entries(item.ratings).filter(([key]) => key !== 'overall') : [];
    const styles = React.useMemo(() => createStyles(theme), [theme]);
    const overallScore = Math.max(0, Math.min(5, Math.round(item.ratings?.overall || 0)));

    return (
        <TouchableOpacity
            style={styles.reviewItem}
            onPress={onToggle}
            activeOpacity={0.7}
            accessibilityRole="button"
            accessibilityState={{ expanded: isExpanded }}
            accessibilityLabel={`${t('review_by')} ${item.poster_username || t('anonymous')}. ${item.ratings?.overall || 0} ${t('stars')}.`}
        >
            <View style={styles.reviewHeader}>
                <View style={styles.starsRow}>
                    {[1, 2, 3, 4, 5].map((i) => (
                        <Icon
                            key={i}
                            name={i <= overallScore ? 'star' : 'star-outline'}
                            size={16}
                            color={i <= overallScore ? theme.colors.warning : theme.colors.border}
                            style={{ marginRight: 2 }}
                        />
                    ))}
                </View>
                <View style={{ flexDirection: 'row', alignItems: 'center' }}>
                    {item.poster_id === userId && onEdit && (
                        <TouchableOpacity 
                            onPress={onEdit}
                            accessibilityRole="button"
                            accessibilityLabel={t('edit_review')}
                        >
                            <Text style={{ color: theme.colors.primary, marginRight: 10, fontWeight: 'bold' }}>{t('edit')}</Text>
                        </TouchableOpacity>
                    )}
                    <Text style={styles.timeText}>{getRelativeTime(item.created_at, t)}</Text>
                    <View style={styles.dropdownButton}>
                        <Icon name={isExpanded ? 'chevron-up' : 'chevron-down'} size={14} color={theme.colors.subtext} />
                    </View>
                </View>
            </View>

            {isExpanded && subRatings.length > 0 && (
                <View style={styles.subRatingsContainer}>
                    {subRatings.map(([key, score]) => (
                        <View key={key} style={styles.subRatingItem}>
                            <Text style={styles.subRatingText}>
                                {t(key)}: <Text style={{ fontWeight: 'bold' }}>{score} </Text>
                                <Icon name="star" size={12} color={theme.colors.warning} />
                            </Text>
                        </View>
                    ))}
                </View>
            )}

            <Text style={styles.commentText}>{item.comment}</Text>
            
            <View style={styles.footerRow}>
                <Text style={styles.user}>- {item.poster_username || t('anonymous')}</Text>
                {showBikeId && (
                    <Text style={styles.bikeIdBadge}>Bike #{item.bike_numerical_id}</Text>
                )}
            </View>
        </TouchableOpacity>
    );
});

const createStyles = (theme) => StyleSheet.create({
    reviewItem: { padding: 10, borderBottomWidth: 1, borderBottomColor: theme.colors.border, marginBottom: 10 },
    reviewHeader: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginBottom: 5 },
    starsRow: { flexDirection: 'row', alignItems: 'center' },
    timeText: { fontSize: 14, color: theme.colors.subtext },
    rating: { fontSize: 18, color: theme.colors.text },
    commentText: { color: theme.colors.text, marginTop: 5 },
    footerRow: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginTop: 5 },
    user: { fontStyle: 'italic', color: theme.colors.subtext },
    bikeIdBadge: { fontSize: 12, backgroundColor: theme.colors.card, paddingHorizontal: 6, paddingVertical: 2, borderRadius: 4, color: theme.colors.text, overflow: 'hidden' },
    dropdownButton: { padding: 5, marginLeft: 10 },
    dropdownArrow: { fontSize: 14, color: theme.colors.subtext },
    subRatingsContainer: {
        flexDirection: 'row', flexWrap: 'wrap', marginTop: 5, marginBottom: 10,
        backgroundColor: theme.colors.inputBackground, padding: 8, borderRadius: 5
    },
    subRatingItem: { width: '50%', paddingVertical: 2 },
    subRatingText: { fontSize: 12, color: theme.colors.subtext }
});

ReviewItem.displayName = 'ReviewItem';

export default ReviewItem;
