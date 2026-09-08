import React, { useContext } from 'react';
import { View, Text, TouchableOpacity, StyleSheet } from 'react-native';
import { ThemeContext } from '../context/ThemeContext';
import { LanguageContext } from '../context/LanguageContext';
import { AuthContext } from '../context/AuthContext';
import { getRelativeTime } from '../utils/time';
import Icon from './Icon';
import Badge from './Badge';

const ReviewItem = React.memo(({ item, isExpanded, onToggle, onEdit, showBikeId, onPressBike }) => {
    const { theme } = useContext(ThemeContext);
    const { t } = useContext(LanguageContext);
    const { userId } = useContext(AuthContext);

    const subRatings = item.ratings ? Object.entries(item.ratings).filter(([key]) => key !== 'overall') : [];
    const styles = React.useMemo(() => createStyles(theme), [theme]);
    const overallScore = Math.max(0, Math.min(5, Math.round(item.ratings?.overall || 0)));

    if (showBikeId) {
        return (
            <View style={styles.card}>
                {/* Dedicated Bike Header Row */}
                <View style={styles.bikeCardHeader}>
                    <TouchableOpacity
                        style={styles.bikeHeaderLeft}
                        onPress={onPressBike}
                        disabled={!onPressBike}
                        activeOpacity={0.7}
                        accessibilityRole="button"
                        accessibilityLabel={`Bike ${item.bike_numerical_id}`}
                    >
                        <View style={styles.bikeIconBadge}>
                            <Icon name="bicycle" size={16} color={theme.colors.primary} />
                        </View>
                        <Text style={styles.bikeNumber}>#{item.bike_numerical_id}</Text>
                        {onPressBike && (
                            <Icon name="chevron-forward" size={15} color={theme.colors.subtext} style={styles.bikeChevron} />
                        )}
                    </TouchableOpacity>

                    <View style={styles.headerRight}>
                        {onEdit && (
                            <TouchableOpacity
                                onPress={onEdit}
                                style={styles.editBtn}
                                accessibilityRole="button"
                                accessibilityLabel={t('edit_review')}
                                hitSlop={{ top: 8, bottom: 8, left: 8, right: 8 }}
                            >
                                <Icon name="create-outline" size={14} color={theme.colors.primary} />
                                <Text style={styles.editText}>{t('edit')}</Text>
                            </TouchableOpacity>
                        )}
                        <Text style={styles.timeText}>{getRelativeTime(item.created_at, t)}</Text>
                    </View>
                </View>

                {/* Rating & Expand Section */}
                <TouchableOpacity
                    style={styles.bodyTouchable}
                    onPress={onToggle}
                    activeOpacity={0.7}
                    accessibilityRole="button"
                    accessibilityState={{ expanded: isExpanded }}
                >
                    <View style={styles.ratingRowWithBike}>
                        <View style={styles.starsRow}>
                            {[1, 2, 3, 4, 5].map((i) => (
                                <Icon
                                    key={i}
                                    name={i <= overallScore ? 'star' : 'star-outline'}
                                    size={16}
                                    color={i <= overallScore ? theme.colors.warning : theme.colors.border}
                                    style={styles.starIcon}
                                />
                            ))}
                            {item.ratings?.overall != null && (
                                <Text style={styles.scoreText}>{item.ratings.overall.toFixed(1)}</Text>
                            )}
                        </View>

                        {subRatings.length > 0 && (
                            <View style={styles.subRatingsToggle}>
                                <Text style={styles.subRatingsToggleText}>{t('ratings')}</Text>
                                <Icon
                                    name={isExpanded ? 'chevron-up' : 'chevron-down'}
                                    size={14}
                                    color={theme.colors.subtext}
                                />
                            </View>
                        )}
                    </View>

                    {/* Comment Body */}
                    {item.comment ? (
                        <Text style={styles.commentText} numberOfLines={isExpanded ? undefined : 3}>
                            {item.comment}
                        </Text>
                    ) : null}

                    {/* Expanded Sub-Ratings */}
                    {isExpanded && subRatings.length > 0 && (
                        <View style={styles.subRatingsContainer}>
                            <Text style={styles.subRatingsHeader}>{t('ratings')}</Text>
                            <View style={styles.subRatingsGrid}>
                                {subRatings.map(([key, score]) => (
                                    <View key={key} style={styles.subRatingItem}>
                                        <Text style={styles.subRatingLabel} numberOfLines={1}>{t(key) || key}</Text>
                                        <View style={styles.subRatingScoreRow}>
                                            <Text style={styles.subRatingScore}>{score} </Text>
                                            <Icon name="star" size={12} color={theme.colors.warning} />
                                        </View>
                                    </View>
                                ))}
                            </View>
                        </View>
                    )}
                </TouchableOpacity>
            </View>
        );
    }

    return (
        <TouchableOpacity
            style={styles.card}
            onPress={onToggle}
            activeOpacity={0.7}
            accessibilityRole="button"
            accessibilityState={{ expanded: isExpanded }}
            accessibilityLabel={`${t('review_by')} ${item.poster_username || t('anonymous')}. ${item.ratings?.overall || 0} ${t('stars')}.`}
        >
            {/* Header Row */}
            <View style={styles.headerRow}>
                <View style={styles.ratingGroup}>
                    <View style={styles.starsRow}>
                        {[1, 2, 3, 4, 5].map((i) => (
                            <Icon
                                key={i}
                                name={i <= overallScore ? 'star' : 'star-outline'}
                                size={16}
                                color={i <= overallScore ? theme.colors.warning : theme.colors.border}
                                style={styles.starIcon}
                            />
                        ))}
                    </View>
                    {item.ratings?.overall != null && (
                        <Text style={styles.scoreText}>{item.ratings.overall.toFixed(1)}</Text>
                    )}
                </View>

                <View style={styles.headerRight}>
                    {item.poster_id === userId && onEdit && (
                        <TouchableOpacity
                            onPress={onEdit}
                            style={styles.editBtn}
                            accessibilityRole="button"
                            accessibilityLabel={t('edit_review')}
                            hitSlop={{ top: 8, bottom: 8, left: 8, right: 8 }}
                        >
                            <Icon name="create-outline" size={14} color={theme.colors.primary} />
                            <Text style={styles.editText}>{t('edit')}</Text>
                        </TouchableOpacity>
                    )}
                    <Text style={styles.timeText}>{getRelativeTime(item.created_at, t)}</Text>
                    <Icon
                        name={isExpanded ? 'chevron-up' : 'chevron-down'}
                        size={15}
                        color={theme.colors.subtext}
                        style={styles.chevron}
                    />
                </View>
            </View>

            {/* Comment Body */}
            {item.comment ? (
                <Text style={styles.commentText} numberOfLines={isExpanded ? undefined : 3}>
                    {item.comment}
                </Text>
            ) : null}

            {/* Expanded Sub-Ratings */}
            {isExpanded && subRatings.length > 0 && (
                <View style={styles.subRatingsContainer}>
                    <Text style={styles.subRatingsHeader}>{t('ratings')}</Text>
                    <View style={styles.subRatingsGrid}>
                        {subRatings.map(([key, score]) => (
                            <View key={key} style={styles.subRatingItem}>
                                <Text style={styles.subRatingLabel} numberOfLines={1}>{t(key) || key}</Text>
                                <View style={styles.subRatingScoreRow}>
                                    <Text style={styles.subRatingScore}>{score} </Text>
                                    <Icon name="star" size={12} color={theme.colors.warning} />
                                </View>
                            </View>
                        ))}
                    </View>
                </View>
            )}

            {/* Footer Row */}
            <View style={styles.footerRow}>
                <View style={styles.userGroup}>
                    <Icon name="person-circle-outline" size={16} color={theme.colors.subtext} />
                    <Text style={styles.userText}>{item.poster_username || t('anonymous')}</Text>
                </View>
            </View>
        </TouchableOpacity>
    );
});

const createStyles = (theme) => StyleSheet.create({
    card: {
        backgroundColor: theme.colors.card,
        borderRadius: theme?.metrics?.radii?.lg || 16,
        padding: theme?.metrics?.spacing?.lg || 16,
        marginBottom: theme?.metrics?.spacing?.md || 12,
        borderWidth: 1,
        borderColor: theme.colors.border,
        ...(theme?.shadows?.sm || {}),
    },
    bikeCardHeader: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        paddingBottom: 10,
        marginBottom: 10,
        borderBottomWidth: 1,
        borderBottomColor: theme.colors.border + '60',
    },
    bikeHeaderLeft: {
        flexDirection: 'row',
        alignItems: 'center',
    },
    bikeIconBadge: {
        width: 30,
        height: 30,
        borderRadius: theme?.metrics?.radii?.sm || 8,
        backgroundColor: theme.colors.primary + '18',
        justifyContent: 'center',
        alignItems: 'center',
        marginRight: 8,
    },
    bikeNumber: {
        fontSize: 16,
        fontWeight: '700',
        color: theme.colors.text,
    },
    bikeChevron: {
        marginLeft: 4,
    },
    bodyTouchable: {
        width: '100%',
    },
    ratingRowWithBike: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: 6,
    },
    subRatingsToggle: {
        flexDirection: 'row',
        alignItems: 'center',
        gap: 4,
        paddingHorizontal: 8,
        paddingVertical: 2,
        borderRadius: theme?.metrics?.radii?.sm || 8,
        backgroundColor: theme.colors.inputBackground,
    },
    subRatingsToggleText: {
        fontSize: 12,
        fontWeight: '600',
        color: theme.colors.subtext,
    },
    headerRow: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: 8,
    },
    ratingGroup: {
        flexDirection: 'row',
        alignItems: 'center',
        flexWrap: 'wrap',
        gap: 6,
    },
    starsRow: {
        flexDirection: 'row',
        alignItems: 'center',
    },
    starIcon: {
        marginRight: 2,
    },
    scoreText: {
        fontSize: 14,
        fontWeight: '700',
        color: theme.colors.text,
        marginLeft: 2,
    },
    headerRight: {
        flexDirection: 'row',
        alignItems: 'center',
        gap: 8,
    },
    editBtn: {
        flexDirection: 'row',
        alignItems: 'center',
        gap: 3,
        paddingVertical: 3,
        paddingHorizontal: 8,
        borderRadius: theme?.metrics?.radii?.sm || 8,
        backgroundColor: theme.colors.primary + '18',
    },
    editText: {
        color: theme.colors.primary,
        fontSize: 12,
        fontWeight: 'bold',
    },
    timeText: {
        fontSize: 12,
        color: theme.colors.subtext,
    },
    chevron: {
        marginLeft: 2,
    },
    commentText: {
        fontSize: 14,
        lineHeight: 20,
        color: theme.colors.text,
        marginVertical: 6,
    },
    subRatingsContainer: {
        marginTop: 8,
        marginBottom: 8,
        backgroundColor: theme.colors.inputBackground,
        padding: 10,
        borderRadius: theme?.metrics?.radii?.md || 12,
        borderWidth: 1,
        borderColor: theme.colors.border,
    },
    subRatingsHeader: {
        fontSize: 12,
        fontWeight: '600',
        color: theme.colors.subtext,
        marginBottom: 6,
        textTransform: 'uppercase',
        letterSpacing: 0.5,
    },
    subRatingsGrid: {
        flexDirection: 'row',
        flexWrap: 'wrap',
        rowGap: 6,
    },
    subRatingItem: {
        width: '50%',
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'space-between',
        paddingRight: 8,
    },
    subRatingLabel: {
        fontSize: 12,
        color: theme.colors.subtext,
    },
    subRatingScoreRow: {
        flexDirection: 'row',
        alignItems: 'center',
    },
    subRatingScore: {
        fontSize: 12,
        fontWeight: 'bold',
        color: theme.colors.text,
    },
    footerRow: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginTop: 6,
        paddingTop: 8,
        borderTopWidth: 1,
        borderTopColor: theme.colors.border + '50',
    },
    userGroup: {
        flexDirection: 'row',
        alignItems: 'center',
        gap: 6,
    },
    userText: {
        fontSize: 12,
        color: theme.colors.subtext,
        fontStyle: 'italic',
    },
});

ReviewItem.displayName = 'ReviewItem';

export default ReviewItem;
