import { useFocusEffect } from '@react-navigation/native';
import React, { useState, useCallback, useContext, useRef } from 'react';

import { View, Text, StyleSheet, FlatList, Button, TouchableOpacity, Pressable, Animated, Dimensions, Easing } from 'react-native';

import api from '../services/api';
import { AuthContext } from '../context/AuthContext';
import { ThemeContext } from '../context/ThemeContext';
import { useSession } from '../context/SessionContext';
import { LanguageContext } from '../context/LanguageContext';
import { useToast } from '../context/ToastContext';

const getRelativeTime = (dateString, t) => {
    if (!dateString) return '';
    const now = new Date();
    const then = new Date(dateString);
    const seconds = Math.floor((now - then) / 1000);

    if (seconds < 60) return t('just_now');
    const minutes = Math.floor(seconds / 60);
    if (minutes < 60) return t('m_ago', { minutes });
    const hours = Math.floor(minutes / 60);
    if (hours < 24) return t('h_ago', { hours });
    const days = Math.floor(hours / 24);
    if (days < 7) return t('d_ago', { days });
    const weeks = Math.floor(days / 7);
    if (weeks < 4) return t('w_ago', { weeks });
    const months = Math.floor(days / 30);
    if (months < 12) return t('mo_ago', { months });
    const years = Math.floor(days / 365);
    return t('y_ago', { years });
};

const getBorderColor = (rating) => {
    if (rating == null) return 'transparent';
    if (rating >= 4) return '#2ecc71'; // Green
    if (rating >= 3) return '#f1c40f'; // Yellow
    return '#e74c3c'; // Red
};

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
    }, []);
    const [expandedReviews, setExpandedReviews] = useState(new Set());

    // Ref to track if we need to set the default window (first load or bike switch)
    const isFirstLoad = useRef(true);

    const { theme } = useContext(ThemeContext);

    const toggleReview = (reviewId, context) => {
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
    };



    const { userId, userToken } = useContext(AuthContext);
    const { validatedBikeId } = useSession();
    const { t } = useContext(LanguageContext);
    const { showToast } = useToast();



    // Determine if review is allowed based on session context
    // Using string comparison to handle leading zeros
    console.log('[BikeDetails] ValidatedID:', validatedBikeId, 'CurrentBikeID:', bike.numerical_id);
    const isReviewAllowed = validatedBikeId != null && String(validatedBikeId) === String(bike.numerical_id);

    const fetchData = useCallback(async (currentId) => {
        setLoading(true);
        try {
            // Ensure we use the ID from params if available, otherwise fallback to state
            const targetId = currentId || bike.numerical_id;
            console.log('[BikeDetails] Fetching details for:', targetId);

            const res = await api.get(`/bikes/${targetId}/details`);
            const details = res.data;

            setBike(prev => ({ ...prev, ...details }));
            setReviews(details.reviews || []);
            const ratings = details.ratings || [];
            setAggregates(ratings);

            // Smart Default Logic
            if (isFirstLoad.current) {
                const has2w = ratings.some(r => r.window === '2w');
                setTimeWindow(has2w ? '2w' : 'overall');
                isFirstLoad.current = false;
            }
        } catch (e) {
            console.log("Failed to fetch bike details", e);
        } finally {
            setLoading(false);
        }
    }, [bike.numerical_id]);

    useFocusEffect(
        useCallback(() => {
            // When screen focuses, check if params have changed
            const params = route.params || {};
            const newId = params.bikeId || (params.bike ? params.bike.numerical_id : null);

            // Use String() for comparison
            if (newId && String(newId) !== String(bike.numerical_id)) {
                console.log('[BikeDetails] Params changed, updating bike ID:', newId);
                isFirstLoad.current = true; // Reset for new bike
                setBike(prev => ({ ...prev, numerical_id: String(newId) }));
                fetchData(String(newId));
            } else {
                fetchData();
            }
        }, [route.params, bike.numerical_id, fetchData])
    );

    const styles = createStyles(theme);

    const renderHeader = () => (
        <View>
            <Text style={styles.title}>{t('bike_title', { numerical_id: bike.numerical_id })}</Text>
            {isReviewAllowed && (
                <Text
                    style={styles.updateLink}
                    onPress={() => navigation.navigate('UpdateBike', { bikeId: bike.numerical_id })}
                >
                    {t('incorrect_info_link')}
                </Text>
            )}

            <Text style={styles.detail}>{t('hash_id_label', { hash_id: bike.hash_id || '-' })}</Text>

            <Text style={styles.detail}>{t('type_label', { type: bike.is_electric ? t('electric') : t('mechanical') })}</Text>

            {aggregates.length > 0 && (
                <View style={styles.aggregatesSection}>
                    <View style={styles.aggregatesHeader}>
                        <Text style={styles.subtitle}>{t('average_ratings')}</Text>
                    </View>

                    {/* Tabular Selector for Time Window */}
                    <View style={styles.tabContainer}>
                        {['overall', '2w', '1w'].map((window) => (
                            <TouchableOpacity
                                key={window}
                                style={[
                                    styles.tabButton,
                                    timeWindow === window && styles.activeTabButton
                                ]}
                                onPress={() => setTimeWindow(window)}
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

                    <View style={styles.aggregatesGrid}>
                        {aggregates
                            .filter(agg => agg.subcategory === 'overall' && agg.window === timeWindow)
                            .map(agg => (
                                <View key="overall" style={[styles.aggItem, styles.overallItem]}>
                                    <Text style={[styles.aggLabel, styles.overallLabel]}>{t('overall_rating')}</Text>
                                    <Text style={[styles.aggValue, styles.overallValue]}>{agg.average_rating.toFixed(1)} ⭐</Text>
                                </View>
                            ))
                        }
                        {aggregates
                            .filter(agg => agg.subcategory !== 'overall' && agg.window === timeWindow)
                            .map(agg => (
                                <View key={agg.subcategory} style={styles.aggItem}>
                                    <Text style={styles.aggLabel}>{t(agg.subcategory) || agg.subcategory.charAt(0).toUpperCase() + agg.subcategory.slice(1)}</Text>
                                    <Text style={styles.aggValue}>{agg.average_rating.toFixed(1)} ⭐</Text>
                                </View>
                            ))}
                        {aggregates.filter(agg => agg.window === timeWindow).length === 0 && (
                            <Text style={styles.noRatingsText}>{t('no_reviews')}</Text>
                        )}
                    </View>
                </View>
            )}

            <Text style={styles.subtitle}>{t('reviews')}</Text>
        </View>
    );

    const renderReviewItem = ({ item }, context) => {
        const key = `${item.review_id}-${context}`;
        const isExpanded = expandedReviews.has(key);
        const subRatings = item.ratings ? Object.entries(item.ratings).filter(([key]) => key !== 'overall') : [];

        return (
            <TouchableOpacity
                style={styles.reviewItem}
                onPress={() => toggleReview(item.review_id, context)}
                activeOpacity={0.7}
            >
                <View style={styles.reviewHeader}>
                    <Text style={styles.rating}>{'⭐'.repeat(item.ratings?.overall || 0)}</Text>
                    <View style={{ flexDirection: 'row', alignItems: 'center' }}>
                        {item.poster_id === userId && (
                            <TouchableOpacity onPress={() => {
                                closeModal(); // Close modal if navigating
                                navigation.navigate('UpdateReview', { reviewId: item.review_id });
                            }}>
                                <Text style={{ color: theme.colors.primary, marginRight: 10, fontWeight: 'bold' }}>{t('edit')}</Text>
                            </TouchableOpacity>
                        )}
                        <Text style={styles.timeText}>{getRelativeTime(item.created_at, t)}</Text>
                        <View style={styles.dropdownButton}>
                            <Text style={styles.dropdownArrow}>{isExpanded ? '▲' : '▼'}</Text>
                        </View>
                    </View>
                </View>

                {isExpanded && subRatings.length > 0 && (
                    <View style={styles.subRatingsContainer}>
                        {subRatings.map(([key, score]) => (
                            <View key={key} style={styles.subRatingItem}>
                                <Text style={styles.subRatingText}>
                                    {t(key)}: <Text style={{ fontWeight: 'bold' }}>{score}⭐</Text>
                                </Text>
                            </View>
                        ))}
                    </View>
                )}

                <Text style={styles.commentText}>{item.comment}</Text>
                <Text style={styles.user}>- {item.poster_username || t('anonymous')}</Text>
            </TouchableOpacity>
        );
    };

    const getSortedReviews = () => {
        return [...reviews].sort((a, b) => {
            let diff = 0;
            if (sortBy === 'date') {
                diff = new Date(a.created_at) - new Date(b.created_at);
            } else {
                diff = (a.ratings?.overall || 0) - (b.ratings?.overall || 0);
            }
            return sortOrder === 'asc' ? diff : -diff;
        });
    }

    // Default view shows only top 3, always sorted by date (newest)
    const previewReviews = [...reviews]
        .sort((a, b) => new Date(b.created_at) - new Date(a.created_at))
        .slice(0, 3);

    const activeAgg = aggregates.find(a => a.subcategory === 'overall' && a.window === timeWindow);
    const activeRating = activeAgg ? activeAgg.average_rating : null;
    const dynamicBorderColor = getBorderColor(activeRating);

    return (
        <View style={styles.container}>
            <View style={[
                styles.contentWrapper,
                activeRating != null && { borderWidth: 2, borderColor: dynamicBorderColor, borderRadius: 12, margin: 12 }
            ]}>
                <FlatList
                    ListHeaderComponent={renderHeader}
                    data={previewReviews}
                    keyExtractor={item => item.review_id ? item.review_id.toString() : Math.random().toString()}
                    renderItem={(props) => renderReviewItem(props, 'preview')}
                    ListFooterComponent={
                        reviews.length === 0 ? (
                            <Text style={styles.emptyText}>{t('no_reviews')}</Text>
                        ) : reviews.length > 3 ? (
                            <TouchableOpacity style={styles.seeAllButton} onPress={openModal}>
                                <Text style={styles.seeAllText}>
                                    {t('see_all_reviews', { count: reviews.length })}
                                </Text>
                            </TouchableOpacity>
                        ) : null
                    }
                    contentContainerStyle={{ paddingBottom: 20 }}
                />

                {/* Only show "Write a Review" if validated in current session */}
                {/* Show "Write a Review" for everyone, but redirect if not allowed */}
                <View style={styles.footerButton}>
                    <TouchableOpacity
                        style={[
                            styles.actionButton,
                            (!isReviewAllowed) && styles.disabledActionButton
                        ]}
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
                        activeOpacity={0.8}
                    >
                        <Text style={[styles.actionButtonText, (!isReviewAllowed) && styles.disabledActionButtonText]}>
                            {t('write_review')}
                        </Text>
                    </TouchableOpacity>
                </View>
            </View>

            {/* Custom Modal for All Reviews (Absolute Positioned View) */}
            {isModalRendered && (
                <View style={styles.customModalOverlay}>
                    <Pressable onPress={closeModal} style={StyleSheet.absoluteFill}>
                        <Animated.View style={[styles.customModalBackdrop, { opacity: fadeAnim }]} />
                    </Pressable>

                    <Animated.View style={[
                        styles.modalContent,
                        { transform: [{ translateY: slideAnim }] }
                    ]}>
                        <View style={styles.modalHeader}>
                            <Text style={styles.subtitle}>{t('all_reviews')}</Text>
                            <TouchableOpacity onPress={closeModal} style={styles.closeButton}>
                                <Text style={styles.closeButtonText}>✕</Text>
                            </TouchableOpacity>
                        </View>

                        {/* Sorting Controls Inside Modal */}
                        <View style={{ flexDirection: 'row', marginBottom: 15, justifyContent: 'flex-start', gap: 10 }}>
                            <TouchableOpacity
                                style={styles.sortButton}
                                onPress={() => setSortBy(prev => prev === 'date' ? 'rating' : 'date')}
                            >
                                <Text style={styles.sortButtonText}>
                                    {t('sort_by_label')}: {sortBy === 'date' ? t('sort_date') : t('sort_rating')}
                                </Text>
                            </TouchableOpacity>

                            <TouchableOpacity
                                style={styles.sortButton}
                                onPress={() => setSortOrder(prev => prev === 'asc' ? 'desc' : 'asc')}
                            >
                                <Text style={styles.sortButtonText}>
                                    {sortOrder === 'asc' ? '↑' : '↓'}
                                </Text>
                            </TouchableOpacity>
                        </View>

                        <FlatList
                            data={getSortedReviews()}
                            keyExtractor={item => item.review_id ? 'modal-' + item.review_id.toString() : Math.random().toString()}
                            renderItem={(props) => renderReviewItem(props, 'modal')}
                            contentContainerStyle={{ paddingBottom: 40 }}
                        />
                    </Animated.View>
                </View>
            )}
        </View>
    );
};

const createStyles = (theme) => StyleSheet.create({
    container: { flex: 1, backgroundColor: theme.colors.background }, // Removed padding
    contentWrapper: { flex: 1, padding: 20 }, // Added content wrapper with padding
    title: { fontSize: 24, fontWeight: 'bold', marginBottom: 10, color: theme.colors.text },
    detail: { fontSize: 16, marginBottom: 5, color: theme.colors.text },
    // reviewsSection: { flex: 1, marginTop: 20 }, // Removed as now part of FlatList custom header
    aggregatesSection: { marginTop: 20, backgroundColor: theme.colors.card, padding: 15, borderRadius: 10, marginBottom: 20 },
    aggregatesGrid: { flexDirection: 'row', flexWrap: 'wrap', justifyContent: 'space-between' },
    aggItem: { width: '48%', backgroundColor: theme.colors.inputBackground, padding: 10, borderRadius: 8, marginBottom: 10, alignItems: 'center', shadowColor: '#000', shadowOffset: { width: 0, height: 1 }, shadowOpacity: 0.1, shadowRadius: 2, elevation: 2 },
    aggLabel: { fontSize: 14, color: theme.colors.subtext, marginBottom: 2 },
    aggValue: { fontSize: 16, fontWeight: 'bold', color: '#f39c12' },
    subtitle: { fontSize: 20, marginBottom: 10, fontWeight: '600', color: theme.colors.text },
    reviewItem: { padding: 10, borderBottomWidth: 1, borderBottomColor: theme.colors.border, marginBottom: 10 },
    reviewHeader: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginBottom: 5 },
    timeText: { fontSize: 14, color: theme.colors.subtext },
    rating: { fontSize: 18, color: theme.colors.text },
    commentText: { color: theme.colors.text },
    user: { fontStyle: 'italic', marginTop: 5, color: theme.colors.subtext },
    emptyText: { color: theme.colors.subtext },
    updateLink: {
        fontSize: 14,
        color: theme.colors.primary,
        marginBottom: 10,
        textDecorationLine: 'underline'
    },
    // reviewsHeader removed as unused
    sortContainer: {
        flexDirection: 'row',
        gap: 8
    },
    sortButton: {
        paddingVertical: 6,
        paddingHorizontal: 12,
        borderRadius: 20,
        backgroundColor: theme.colors.card,
        borderWidth: 1,
        borderColor: theme.colors.border
    },
    sortButtonText: {
        fontSize: 14,
        color: theme.colors.text,
        fontWeight: '500'
    },
    footerButton: {
        marginTop: 10,
        paddingTop: 10,
        borderTopWidth: 1,
        borderTopColor: theme.colors.border
    },
    seeAllButton: {
        padding: 15,
        backgroundColor: theme.colors.card,
        borderRadius: 8,
        alignItems: 'center',
        marginTop: 10,
        marginBottom: 20,
        borderWidth: 1,
        borderColor: theme.colors.border
    },
    seeAllText: {
        color: theme.colors.primary,
        fontWeight: 'bold',
        fontSize: 16
    },
    // New / Updated styles for Custom Modal
    customModalOverlay: {
        position: 'absolute',
        top: 0,
        bottom: 0,
        left: 0,
        right: 0,
        zIndex: 1000,
        justifyContent: 'flex-end',
        elevation: 20,
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
        borderTopLeftRadius: 20,
        borderTopRightRadius: 20,
        height: '80%', // Bottom sheet taking 80% screen height
        padding: 20,
        shadowColor: "#000",
        shadowOffset: { width: 0, height: -2 },
        shadowOpacity: 0.25,
        shadowRadius: 3.84,
        elevation: 5
    },
    modalHeader: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: 20,
        borderBottomWidth: 1,
        borderBottomColor: theme.colors.border,
        paddingBottom: 15
    },
    closeButton: {
        padding: 5
    },
    closeButtonText: {
        fontSize: 24,
        color: theme.colors.text,
        fontWeight: 'bold'
    },
    dropdownButton: {
        padding: 5,
        marginLeft: 10,
    },
    dropdownArrow: {
        fontSize: 14,
        color: theme.colors.subtext,
    },
    subRatingsContainer: {
        flexDirection: 'row',
        flexWrap: 'wrap',
        marginTop: 5,
        marginBottom: 10,
        backgroundColor: theme.colors.inputBackground,
        padding: 8,
        borderRadius: 5
    },
    subRatingItem: {
        width: '50%',
        paddingVertical: 2
    },
    subRatingText: {
        fontSize: 12,
        color: theme.colors.subtext
    },
    actionButton: {
        backgroundColor: theme.colors.primary,
        paddingVertical: 12,
        borderRadius: 8,
        alignItems: 'center',
        justifyContent: 'center'
    },
    disabledActionButton: {
        backgroundColor: theme.colors.primary,
        opacity: 0.5
    },
    actionButtonText: {
        color: '#FFFFFF',
        fontSize: 16,
        fontWeight: 'bold'
    },
    disabledActionButtonText: {
        color: '#FFFFFF',
        opacity: 0.8
    },
    aggregatesHeader: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: 10
    },
    // New tab styles
    tabContainer: {
        flexDirection: 'row',
        backgroundColor: theme.colors.inputBackground,
        borderRadius: 8,
        padding: 2,
        marginBottom: 15
    },
    tabButton: {
        flex: 1,
        paddingVertical: 8,
        alignItems: 'center',
        borderRadius: 6
    },
    activeTabButton: {
        backgroundColor: theme.colors.card,
        shadowColor: "#000",
        shadowOffset: { width: 0, height: 1 },
        shadowOpacity: 0.1,
        shadowRadius: 1,
        elevation: 1
    },
    tabText: {
        fontSize: 14,
        color: theme.colors.subtext,
        fontWeight: '500'
    },
    activeTabText: {
        color: theme.colors.primary,
        fontWeight: 'bold'
    },
    noRatingsText: {
        width: '100%',
        textAlign: 'center',
        color: theme.colors.subtext,
        fontStyle: 'italic',
        marginTop: 10
    },
    overallItem: {
        width: '100%',
        marginBottom: 10,
        backgroundColor: theme.colors.card,
        borderWidth: 2,
        borderColor: theme.colors.primary,
        shadowColor: theme.colors.primary,
        shadowOpacity: 0.2,
        elevation: 4
    },
    overallLabel: {
        fontSize: 16,
        fontWeight: 'bold',
        color: theme.colors.text
    },
    overallValue: {
        fontSize: 24,
        marginTop: 5
    }
});

export default BikeDetailsScreen;
