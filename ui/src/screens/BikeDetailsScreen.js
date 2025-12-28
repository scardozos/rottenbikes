import { useFocusEffect } from '@react-navigation/native';
import React, { useState, useCallback, useContext } from 'react';

import { View, Text, StyleSheet, FlatList, Button, TouchableOpacity } from 'react-native';

import api from '../services/api';
import { AuthContext } from '../context/AuthContext';
import { ThemeContext } from '../context/ThemeContext';
import { useSession } from '../context/SessionContext';
import { LanguageContext } from '../context/LanguageContext';

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

const BikeDetailsScreen = ({ route, navigation }) => {
    const params = route.params || {};
    // Handle both object navigation and deep linking ID
    const initialBike = params.bike || { numerical_id: params.bikeId };
    const [bike, setBike] = useState(initialBike);
    const [reviews, setReviews] = useState([]);
    const [aggregates, setAggregates] = useState([]);
    const [loading, setLoading] = useState(true);
    const [sortBy, setSortBy] = useState('date'); // 'date' | 'rating'
    const [sortOrder, setSortOrder] = useState('desc'); // 'asc' | 'desc'
    const { theme } = useContext(ThemeContext);
    const { userId } = useContext(AuthContext);
    const { validatedBikeId } = useSession();
    const { t } = useContext(LanguageContext);



    // Determine if review is allowed based on session context
    // Determine if review is allowed based on session context
    // Using loose equality or number conversion to handle potential string/number mismatches
    console.log('[BikeDetails] ValidatedID:', validatedBikeId, 'CurrentBikeID:', bike.numerical_id);
    const isReviewAllowed = validatedBikeId != null && Number(validatedBikeId) === Number(bike.numerical_id);

    const fetchData = async () => {
        setLoading(true);
        try {
            const res = await api.get(`/bikes/${bike.numerical_id}/details`);
            const details = res.data;
            setBike(prev => ({ ...prev, ...details }));
            setReviews(details.reviews || []);
            setAggregates(details.ratings || []);
        } catch (e) {
            console.log("Failed to fetch bike details", e);
        } finally {
            setLoading(false);
        }
    };

    useFocusEffect(
        useCallback(() => {
            fetchData();
        }, [])
    );

    const styles = createStyles(theme);

    return (
        <View style={styles.container}>
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
                    <Text style={styles.subtitle}>{t('average_ratings')}</Text>
                    <View style={styles.aggregatesGrid}>
                        {aggregates
                            .filter(agg => agg.subcategory !== 'overall')
                            .map(agg => (
                                <View key={agg.subcategory} style={styles.aggItem}>
                                    <Text style={styles.aggLabel}>{t(agg.subcategory) || agg.subcategory.charAt(0).toUpperCase() + agg.subcategory.slice(1)}</Text>
                                    <Text style={styles.aggValue}>{agg.average_rating.toFixed(1)} ⭐</Text>
                                </View>
                            ))}
                    </View>
                </View>
            )}

            <View style={styles.reviewsSection}>
                <View style={styles.reviewsHeader}>
                    <Text style={styles.subtitle}>{t('reviews')}</Text>

                    {/* Compact Sorting Controls */}
                    <View style={styles.sortContainer}>
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
                </View>


                <FlatList
                    data={[...reviews].sort((a, b) => {
                        let diff = 0;
                        if (sortBy === 'date') {
                            diff = new Date(a.created_at) - new Date(b.created_at);
                        } else {
                            diff = (a.ratings?.overall || 0) - (b.ratings?.overall || 0);
                        }
                        return sortOrder === 'asc' ? diff : -diff;
                    })}
                    keyExtractor={item => item.review_id ? item.review_id.toString() : Math.random().toString()}

                    renderItem={({ item }) => (
                        <View style={styles.reviewItem}>
                            <View style={styles.reviewHeader}>
                                <Text style={styles.rating}>{'⭐'.repeat(item.ratings?.overall || 0)}</Text>
                                <View style={{ flexDirection: 'row', alignItems: 'center' }}>
                                    {item.poster_id === userId && (
                                        <TouchableOpacity onPress={() => navigation.navigate('UpdateReview', { review: item, bikeId: bike.numerical_id })}>
                                            <Text style={{ color: theme.colors.primary, marginRight: 10, fontWeight: 'bold' }}>{t('edit')}</Text>
                                        </TouchableOpacity>
                                    )}
                                    <Text style={styles.timeText}>{getRelativeTime(item.created_at, t)}</Text>
                                </View>
                            </View>
                            <Text style={styles.commentText}>{item.comment}</Text>
                            <Text style={styles.user}>- {item.poster_username || t('anonymous')}</Text>
                        </View>
                    )}
                    ListEmptyComponent={<Text style={styles.emptyText}>{t('no_reviews')}</Text>}
                />
            </View>
            {/* Only show "Write a Review" if validated in current session */}
            {isReviewAllowed && (
                <Button
                    title={t('write_review')}
                    onPress={() => navigation.navigate('CreateReview', { bikeId: bike.numerical_id })}
                    color={theme.colors.primary}
                />
            )}


        </View>
    );
};

const createStyles = (theme) => StyleSheet.create({
    container: { flex: 1, padding: 20, backgroundColor: theme.colors.background },
    title: { fontSize: 24, fontWeight: 'bold', marginBottom: 10, color: theme.colors.text },
    detail: { fontSize: 16, marginBottom: 5, color: theme.colors.text },
    reviewsSection: { flex: 1, marginTop: 20 },
    aggregatesSection: { marginTop: 20, backgroundColor: theme.colors.card, padding: 15, borderRadius: 10 },
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
    reviewsHeader: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: 10
    },
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
    }
});





export default BikeDetailsScreen;
