import { useFocusEffect } from '@react-navigation/native';
import React, { useState, useCallback, useContext, useRef } from 'react';
import { View, Text, StyleSheet, FlatList, TouchableOpacity, Pressable, Animated, Dimensions, Easing, ActivityIndicator } from 'react-native';
import Icon from '../components/Icon';
import Button from '../components/Button';

import api from '../services/api';
import { AuthContext } from '../context/AuthContext';
import { ThemeContext } from '../context/ThemeContext';
import { useSession } from '../context/SessionContext';
import { LanguageContext } from '../context/LanguageContext';
import { useToast } from '../context/ToastContext';

import ReviewItem from '../components/ReviewItem';
import HealthTrendBadge from '../components/HealthTrendBadge';
import { sortReviews, getPreviewReviews } from '../utils/reviews';

const BikeDetailsScreen = ({ route, navigation }) => {
    const params = route.params || {};
    // Handle both object navigation and deep linking ID
    // We treat IDs as strings now to preserve leading zeros
    const initialBike = params.bike || { numerical_id: params.bikeId ? String(params.bikeId) : null };
    if (initialBike.numerical_id) {
        initialBike.numerical_id = String(initialBike.numerical_id);
    }
    const [bike, setBike] = useState(initialBike);
    const [reviews, setReviews] = useState([]);
    const [aggregates, setAggregates] = useState([]);
    const [loading, setLoading] = useState(true);
    const [sortBy, setSortBy] = useState('date'); // 'date' | 'rating'
    const [sortOrder, setSortOrder] = useState('desc'); // 'asc' | 'desc'
    const [timeWindow, setTimeWindow] = useState('2w'); // '1w', '2w', 'overall'

    const REVIEWS_LIMIT = 5;
    const [reviewsOffset, setReviewsOffset] = useState(0);
    const [hasMoreReviews, setHasMoreReviews] = useState(true);
    const [loadingMoreReviews, setLoadingMoreReviews] = useState(false);
    const [totalReviews, setTotalReviews] = useState(0);
    const [isModalRendered, setIsModalRendered] = useState(false);
    const slideAnim = useRef(new Animated.Value(Dimensions.get('window').height)).current;
    const fadeAnim = useRef(new Animated.Value(0)).current;

    const openModal = useCallback(() => {
        setIsModalRendered(true);
    }, []);

    React.useEffect(() => {
        if (isModalRendered) {
            // Reset values to start state
            slideAnim.setValue(Dimensions.get('window').height);
            fadeAnim.setValue(0);

            Animated.parallel([
                Animated.timing(slideAnim, {
                    toValue: 0,
                    duration: 300,
                    easing: Easing.out(Easing.cubic),
                    useNativeDriver: true,
                }),
                Animated.timing(fadeAnim, {
                    toValue: 1,
                    duration: 300,
                    useNativeDriver: true,
                })
            ]).start();
        }
    }, [isModalRendered, slideAnim, fadeAnim]);

    const closeModal = useCallback(() => {
        Animated.parallel([
            Animated.timing(slideAnim, {
                toValue: Dimensions.get('window').height,
                duration: 250,
                easing: Easing.in(Easing.cubic),
                useNativeDriver: true,
            }),
            Animated.timing(fadeAnim, {
                toValue: 0,
                duration: 250,
                useNativeDriver: true,
            })
        ]).start(() => {
            setIsModalRendered(false);
        });
    }, [slideAnim, fadeAnim]);
    const [expandedReviews, setExpandedReviews] = useState(new Set());

    // Ref to track if we need to set the default window (first load or bike switch)
    const isFirstLoad = useRef(true);

    const { theme } = useContext(ThemeContext);

    const toggleReview = useCallback((reviewId, context) => {
        const key = `${reviewId}-${context}`;
        setExpandedReviews(prev => {
            const newSet = new Set(prev);
            if (newSet.has(key)) {
                newSet.delete(key);
            } else {
                newSet.add(key);
            }
            return newSet;
        });
    }, []);

    const { userToken } = useContext(AuthContext);
    const { validatedBikeId } = useSession();
    const { t } = useContext(LanguageContext);
    const { showToast } = useToast();

    // Determine if review is allowed based on session context
    // Using string comparison to handle leading zeros
    const isReviewAllowed = validatedBikeId != null && String(validatedBikeId) === String(bike.numerical_id);

    const fetchData = useCallback(async (currentId) => {
        setLoading(true);
        setReviewsOffset(0);
        setHasMoreReviews(true);
        try {
            const targetId = currentId || bike.numerical_id;
            const res = await api.get(`/bikes/${targetId}/details?limit=${REVIEWS_LIMIT}&offset=0`);
            const details = res.data;

            setBike(prev => ({ ...prev, ...details }));
            const initialReviews = details.reviews || [];
            setReviews(initialReviews);
            setReviewsOffset(initialReviews.length);
            setHasMoreReviews(initialReviews.length === REVIEWS_LIMIT);
            setTotalReviews(details.total_reviews || 0);

            const ratings = details.ratings || [];
            setAggregates(ratings);

            // Smart Default Logic
            if (isFirstLoad.current) {
                const has2w = ratings.some(r => r.window === '2w');
                setTimeWindow(has2w ? '2w' : 'overall');
                isFirstLoad.current = false;
            }
        } catch (e) {
            console.error("Failed to fetch bike details", e);
        } finally {
            setLoading(false);
        }
    }, [bike.numerical_id]);

    const fetchMoreReviews = async () => {
        if (loadingMoreReviews || !hasMoreReviews) return;
        setLoadingMoreReviews(true);
        try {
            const targetId = bike.numerical_id;
            const res = await api.get(`/bikes/${targetId}/reviews?limit=${REVIEWS_LIMIT}&offset=${reviewsOffset}`);
            const newReviews = res.data?.reviews || [];

            if (newReviews.length < REVIEWS_LIMIT) {
                setHasMoreReviews(false);
            }
            setReviews(prev => [...prev, ...newReviews]);
            setReviewsOffset(prev => prev + newReviews.length);
        } catch (e) {
            console.error("Failed to fetch more reviews", e);
        } finally {
            setLoadingMoreReviews(false);
        }
    };

    useFocusEffect(
        useCallback(() => {
            const currentRouteParams = route.params || {};
            const newId = currentRouteParams.bikeId || (currentRouteParams.bike ? currentRouteParams.bike.numerical_id : null);

            if (newId && String(newId) !== String(bike.numerical_id)) {
                isFirstLoad.current = true;
                setBike(prev => ({ ...prev, numerical_id: String(newId) }));
                fetchData(String(newId));
            } else {
                fetchData();
            }
        }, [route.params, bike.numerical_id, fetchData])
    );

    const styles = React.useMemo(() => createStyles(theme), [theme]);

    const activeAgg = aggregates.find(a => a.subcategory === 'overall' && a.window === timeWindow);
    const overallRating = activeAgg ? activeAgg.average_rating : null;
    const nonOverallAggs = aggregates.filter(agg => agg.subcategory !== 'overall' && agg.window === timeWindow);

    const renderHeader = () => (
        <View style={styles.headerContainer}>
            {/* Bike Identity Card */}
            <View style={styles.bikeCard}>
                <View style={styles.bikeCardTop}>
                    <Text style={styles.title}>{t('bike_title', { numerical_id: bike.numerical_id })}</Text>
                    <View style={[styles.typeBadge, bike.is_electric ? styles.electricBadge : styles.mechanicalBadge]}>
                        <Icon
                            name={bike.is_electric ? 'flash' : 'bicycle'}
                            size={14}
                            color={bike.is_electric ? theme.colors.warning : theme.colors.primary}
                        />
                        <Text style={[styles.typeText, { color: bike.is_electric ? theme.colors.warning : theme.colors.primary }]}>
                            {bike.is_electric ? t('electric') : t('mechanical')}
                        </Text>
                    </View>
                </View>

                <Text style={styles.hashText} numberOfLines={1} ellipsizeMode="middle">
                    {t('hash_id_label', { hash_id: bike.hash_id || '—' })}
                </Text>

                {isReviewAllowed && (
                    <TouchableOpacity
                        onPress={() => navigation.navigate('UpdateBike', { bikeId: bike.numerical_id })}
                        style={styles.editLinkBtn}
                        activeOpacity={0.7}
                    >
                        <Icon name="create-outline" size={14} color={theme.colors.primary} />
                        <Text style={styles.updateLink}>{t('incorrect_info_link')}</Text>
                    </TouchableOpacity>
                )}
            </View>

            {/* Hero Overall Rating Card */}
            <View style={styles.heroCard}>
                <Text style={styles.heroLabel}>{t('overall_rating')}</Text>

                <View style={styles.heroContent}>
                    {overallRating != null ? (
                        <View style={styles.heroScoreRow}>
                            <Text style={styles.heroScore}>{overallRating.toFixed(1)}</Text>
                            <Icon name="star" size={36} color={theme.colors.warning} style={styles.heroStar} />
                            {timeWindow === 'overall' && (
                                <View style={styles.heroTrendContainer}>
                                    <HealthTrendBadge aggregates={aggregates} subcategory="overall" />
                                </View>
                            )}
                        </View>
                    ) : (
                        <Text style={styles.noRatingsText}>{t('no_reviews')}</Text>
                    )}
                </View>

                {/* Time Window Tabs Under Rating Score */}
                <View style={styles.tabContainer}>
                    {['overall', '2w', '1w'].map((window) => (
                        <TouchableOpacity
                            key={window}
                            style={[
                                styles.tabButton,
                                timeWindow === window && styles.activeTabButton
                            ]}
                            onPress={() => setTimeWindow(window)}
                            activeOpacity={0.7}
                            accessibilityRole="button"
                            accessibilityState={{ selected: timeWindow === window }}
                        >
                            <Text style={[
                                styles.tabText,
                                timeWindow === window && styles.activeTabText
                            ]}>
                                {t(`window_${window}`)}
                            </Text>
                        </TouchableOpacity>
                    ))}
                </View>
            </View>

            {/* Subcategory Uniform Grid */}
            {nonOverallAggs.length > 0 && (
                <View style={styles.aggregatesSection}>
                    <Text style={styles.sectionSubtitle}>{t('average_ratings')}</Text>
                    <View style={styles.aggregatesGrid}>
                        {nonOverallAggs.map(agg => (
                            <View key={agg.subcategory} style={styles.aggItem}>
                                <Text style={styles.aggLabel} numberOfLines={1}>
                                    {t(agg.subcategory) || agg.subcategory.charAt(0).toUpperCase() + agg.subcategory.slice(1)}
                                </Text>
                                <View style={styles.aggValueRow}>
                                    <Text style={styles.aggValue}>{agg.average_rating.toFixed(1)}</Text>
                                    <Icon name="star" size={14} color={theme.colors.warning} />
                                    {timeWindow === 'overall' && (
                                        <View style={styles.tileTrend}>
                                            <HealthTrendBadge aggregates={aggregates} subcategory={agg.subcategory} />
                                        </View>
                                    )}
                                </View>
                            </View>
                        ))}
                    </View>
                </View>
            )}

            <View style={styles.reviewsTitleRow}>
                <Text style={styles.sectionSubtitle}>{t('reviews')}</Text>
            </View>
        </View>
    );

    const renderReviewItem = useCallback(({ item }, context) => {
        const key = `${item.review_id}-${context}`;
        const isExpanded = expandedReviews.has(key);

        return (
            <ReviewItem
                item={item}
                isExpanded={isExpanded}
                onToggle={() => toggleReview(item.review_id, context)}
                onEdit={() => {
                    closeModal();
                    navigation.navigate('UpdateReview', { reviewId: item.review_id });
                }}
            />
        );
    }, [expandedReviews, toggleReview, closeModal, navigation]);

    const sortedReviews = React.useMemo(() => sortReviews(reviews, sortBy, sortOrder), [reviews, sortBy, sortOrder]);
    const previewReviews = React.useMemo(() => getPreviewReviews(reviews, 3), [reviews]);

    return (
        <View style={styles.container}>
            {loading && !bike.created_at ? (
                <View style={styles.centerLoading}>
                    <ActivityIndicator size="large" color={theme.colors.primary} />
                </View>
            ) : (
                <View style={styles.mainContent}>
                    <FlatList
                        ListHeaderComponent={renderHeader}
                        data={previewReviews}
                        keyExtractor={item => item.review_id ? item.review_id.toString() : Math.random().toString()}
                        renderItem={(props) => renderReviewItem(props, 'preview')}
                        contentContainerStyle={styles.scrollList}
                        ListFooterComponent={
                            reviews.length === 0 ? (
                                <Text style={styles.emptyText}>{t('no_reviews')}</Text>
                            ) : (reviews.length > 3 || hasMoreReviews) ? (
                                <Button
                                    title={t('see_all_reviews', { count: totalReviews })}
                                    onPress={openModal}
                                    variant="ghost"
                                    style={styles.seeAllButton}
                                />
                            ) : null
                        }
                    />

                    {/* Bottom CTA Button */}
                    <View style={styles.footerContainer}>
                        <Button
                            title={t('write_review')}
                            onPress={() => {
                                if (!userToken) {
                                    showToast(t('login_to_review_toast'), 'info');
                                    navigation.navigate('Home');
                                    return;
                                }
                                if (!isReviewAllowed) {
                                    showToast(t('scan_to_review_toast'), 'info');
                                    navigation.navigate('Home');
                                    return;
                                }
                                navigation.navigate('CreateReview', { bikeId: bike.numerical_id });
                            }}
                            variant={isReviewAllowed ? 'primary' : 'ghost'}
                            size="lg"
                            disabled={!isReviewAllowed && !userToken}
                        />
                    </View>
                </View>
            )}

            {/* Custom Bottom Sheet Modal for All Reviews */}
            {isModalRendered && (
                <View style={styles.customModalOverlay}>
                    <Pressable onPress={closeModal} style={StyleSheet.absoluteFill}>
                        <Animated.View style={[styles.customModalBackdrop, { opacity: fadeAnim }]} />
                    </Pressable>

                    <Animated.View style={[
                        styles.modalContent,
                        { transform: [{ translateY: slideAnim }] }
                    ]}>
                        {/* Drag Handle Indicator */}
                        <View style={styles.dragHandleContainer}>
                            <View style={styles.dragHandle} />
                        </View>

                        <View style={styles.modalHeader}>
                            <Text style={styles.modalTitle}>{t('all_reviews')}</Text>
                            <TouchableOpacity onPress={closeModal} style={styles.closeButton}>
                                <Icon name="close" size={24} color={theme.colors.text} />
                            </TouchableOpacity>
                        </View>

                        {/* Sorting Chips Row */}
                        <View style={styles.modalSortRow}>
                            <TouchableOpacity
                                style={[styles.modalSortChip, sortBy === 'date' && styles.modalSortChipActive]}
                                onPress={() => setSortBy('date')}
                            >
                                <Icon
                                    name="time-outline"
                                    size={14}
                                    color={sortBy === 'date' ? (theme.colors.buttonText || '#FFFFFF') : theme.colors.subtext}
                                />
                                <Text style={[styles.modalSortText, sortBy === 'date' && styles.modalSortTextActive]}>
                                    {t('sort_date')}
                                </Text>
                            </TouchableOpacity>

                            <TouchableOpacity
                                style={[styles.modalSortChip, sortBy === 'rating' && styles.modalSortChipActive]}
                                onPress={() => setSortBy('rating')}
                            >
                                <Icon
                                    name="star-outline"
                                    size={14}
                                    color={sortBy === 'rating' ? (theme.colors.buttonText || '#FFFFFF') : theme.colors.subtext}
                                />
                                <Text style={[styles.modalSortText, sortBy === 'rating' && styles.modalSortTextActive]}>
                                    {t('sort_rating')}
                                </Text>
                            </TouchableOpacity>

                            <TouchableOpacity
                                style={styles.modalOrderToggle}
                                onPress={() => setSortOrder(prev => prev === 'asc' ? 'desc' : 'asc')}
                            >
                                <Icon name={sortOrder === 'asc' ? 'arrow-up' : 'arrow-down'} size={16} color={theme.colors.text} />
                            </TouchableOpacity>
                        </View>

                        <FlatList
                            data={sortedReviews}
                            keyExtractor={item => item.review_id ? 'modal-' + item.review_id.toString() : Math.random().toString()}
                            renderItem={(props) => renderReviewItem(props, 'modal')}
                            contentContainerStyle={styles.modalListContent}
                            onEndReached={fetchMoreReviews}
                            onEndReachedThreshold={0.5}
                            ListFooterComponent={() => {
                                if (loadingMoreReviews) {
                                    return <ActivityIndicator size="small" color={theme.colors.primary} style={{ marginVertical: 15 }} />;
                                }
                                return null;
                            }}
                        />
                    </Animated.View>
                </View>
            )}
        </View>
    );
};

const createStyles = (theme) => StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: theme.colors.background,
    },
    mainContent: {
        flex: 1,
    },
    scrollList: {
        paddingHorizontal: theme?.metrics?.spacing?.lg || 16,
        paddingTop: theme?.metrics?.spacing?.md || 12,
        paddingBottom: theme?.metrics?.spacing?.xl || 24,
    },
    headerContainer: {
        marginBottom: theme?.metrics?.spacing?.md || 12,
    },
    centerLoading: {
        flex: 1,
        justifyContent: 'center',
        alignItems: 'center',
    },
    bikeCard: {
        backgroundColor: theme.colors.card,
        borderRadius: theme?.metrics?.radii?.lg || 16,
        padding: theme?.metrics?.spacing?.lg || 16,
        marginBottom: theme?.metrics?.spacing?.md || 12,
        borderWidth: 1,
        borderColor: theme.colors.border,
        ...(theme?.shadows?.sm || {}),
    },
    bikeCardTop: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: 8,
    },
    title: {
        fontSize: theme?.typography?.h2?.fontSize || 24,
        fontWeight: 'bold',
        color: theme.colors.text,
    },
    typeBadge: {
        flexDirection: 'row',
        alignItems: 'center',
        paddingHorizontal: 8,
        paddingVertical: 3,
        borderRadius: theme?.metrics?.radii?.pill || 9999,
    },
    electricBadge: {
        backgroundColor: theme.colors.warning + '18',
    },
    mechanicalBadge: {
        backgroundColor: theme.colors.primary + '18',
    },
    typeText: {
        fontSize: 12,
        fontWeight: '600',
        marginLeft: 4,
    },
    hashText: {
        fontSize: 13,
        color: theme.colors.subtext,
    },
    editLinkBtn: {
        flexDirection: 'row',
        alignItems: 'center',
        gap: 6,
        marginTop: 10,
        paddingTop: 8,
        borderTopWidth: 1,
        borderTopColor: theme.colors.border,
        alignSelf: 'stretch',
    },
    updateLink: {
        fontSize: 13,
        color: theme.colors.primary,
        fontWeight: '500',
        flexShrink: 1,
    },
    heroCard: {
        backgroundColor: theme.colors.card,
        borderRadius: theme?.metrics?.radii?.lg || 16,
        padding: theme?.metrics?.spacing?.lg || 16,
        marginBottom: theme?.metrics?.spacing?.md || 12,
        borderWidth: 1,
        borderColor: theme.colors.border,
        ...(theme?.shadows?.sm || {}),
    },
    heroLabel: {
        fontSize: 14,
        fontWeight: '600',
        color: theme.colors.subtext,
        textTransform: 'uppercase',
        letterSpacing: 0.5,
        marginBottom: 6,
    },
    heroContent: {
        alignItems: 'flex-start',
        marginBottom: 14,
    },
    heroScoreRow: {
        flexDirection: 'row',
        alignItems: 'center',
    },
    heroScore: {
        fontSize: 42,
        fontWeight: 'bold',
        color: theme.colors.text,
        letterSpacing: -1,
    },
    heroStar: {
        marginLeft: 8,
    },
    heroTrendContainer: {
        marginLeft: 12,
    },
    noRatingsText: {
        color: theme.colors.subtext,
        fontSize: 15,
        marginVertical: 6,
    },
    tabContainer: {
        flexDirection: 'row',
        width: '100%',
        backgroundColor: theme.colors.inputBackground,
        borderRadius: theme?.metrics?.radii?.pill || 9999,
        padding: 3,
    },
    tabButton: {
        flex: 1,
        paddingVertical: 7,
        alignItems: 'center',
        justifyContent: 'center',
        borderRadius: theme?.metrics?.radii?.pill || 9999,
    },
    activeTabButton: {
        backgroundColor: theme.colors.card,
        ...(theme?.shadows?.sm || {}),
    },
    tabText: {
        fontSize: 13,
        color: theme.colors.subtext,
        fontWeight: '500',
    },
    activeTabText: {
        color: theme.colors.primary,
        fontWeight: 'bold',
    },
    aggregatesSection: {
        marginBottom: theme?.metrics?.spacing?.md || 12,
    },
    sectionSubtitle: {
        fontSize: 18,
        fontWeight: 'bold',
        color: theme.colors.text,
        marginBottom: theme?.metrics?.spacing?.sm || 8,
    },
    aggregatesGrid: {
        flexDirection: 'row',
        flexWrap: 'wrap',
        justifyContent: 'space-between',
        rowGap: 10,
    },
    aggItem: {
        width: '48.5%',
        backgroundColor: theme.colors.card,
        padding: 12,
        borderRadius: theme?.metrics?.radii?.md || 12,
        borderWidth: 1,
        borderColor: theme.colors.border,
    },
    aggLabel: {
        fontSize: 13,
        color: theme.colors.subtext,
        fontWeight: '500',
        marginBottom: 6,
    },
    aggValueRow: {
        flexDirection: 'row',
        alignItems: 'center',
    },
    aggValue: {
        fontSize: 17,
        fontWeight: 'bold',
        color: theme.colors.text,
        marginRight: 4,
    },
    tileTrend: {
        marginLeft: 'auto',
    },
    reviewsTitleRow: {
        marginTop: 6,
    },
    emptyText: {
        color: theme.colors.subtext,
        textAlign: 'center',
        marginVertical: 20,
        fontSize: 14,
    },
    seeAllButton: {
        marginTop: 10,
        marginBottom: 20,
    },
    footerContainer: {
        paddingHorizontal: theme?.metrics?.spacing?.lg || 16,
        paddingVertical: theme?.metrics?.spacing?.md || 12,
        backgroundColor: theme.colors.card,
        borderTopWidth: 1,
        borderTopColor: theme.colors.border,
    },
    customModalOverlay: {
        position: 'absolute',
        top: 0,
        bottom: 0,
        left: 0,
        right: 0,
        zIndex: 1000,
        justifyContent: 'flex-end',
    },
    customModalBackdrop: {
        position: 'absolute',
        top: 0,
        bottom: 0,
        left: 0,
        right: 0,
        backgroundColor: 'rgba(0,0,0,0.5)',
    },
    modalContent: {
        backgroundColor: theme.colors.background,
        borderTopLeftRadius: theme?.metrics?.radii?.xl || 24,
        borderTopRightRadius: theme?.metrics?.radii?.xl || 24,
        height: '80%',
        paddingHorizontal: theme?.metrics?.spacing?.lg || 16,
        paddingBottom: theme?.metrics?.spacing?.lg || 16,
        ...(theme?.shadows?.lg || {}),
    },
    dragHandleContainer: {
        alignItems: 'center',
        paddingVertical: 10,
    },
    dragHandle: {
        width: 40,
        height: 4,
        borderRadius: 2,
        backgroundColor: theme.colors.border,
    },
    modalHeader: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        paddingBottom: 12,
        borderBottomWidth: 1,
        borderBottomColor: theme.colors.border,
        marginBottom: 12,
    },
    modalTitle: {
        fontSize: 18,
        fontWeight: 'bold',
        color: theme.colors.text,
    },
    closeButton: {
        padding: 4,
    },
    modalSortRow: {
        flexDirection: 'row',
        alignItems: 'center',
        gap: 8,
        marginBottom: 14,
    },
    modalSortChip: {
        flexDirection: 'row',
        alignItems: 'center',
        paddingVertical: 6,
        paddingHorizontal: 12,
        borderRadius: theme?.metrics?.radii?.pill || 9999,
        backgroundColor: theme.colors.card,
        borderWidth: 1,
        borderColor: theme.colors.border,
        gap: 6,
    },
    modalSortChipActive: {
        backgroundColor: theme.colors.primary,
        borderColor: theme.colors.primary,
    },
    modalSortText: {
        fontSize: 13,
        color: theme.colors.subtext,
        fontWeight: '500',
    },
    modalSortTextActive: {
        color: theme.colors.buttonText || '#FFFFFF',
        fontWeight: 'bold',
    },
    modalOrderToggle: {
        padding: 8,
        borderRadius: theme?.metrics?.radii?.pill || 9999,
        backgroundColor: theme.colors.card,
        borderWidth: 1,
        borderColor: theme.colors.border,
    },
    modalListContent: {
        paddingBottom: 40,
    },
});

export default BikeDetailsScreen;
