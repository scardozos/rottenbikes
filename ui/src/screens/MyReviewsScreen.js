import React, { useState, useEffect, useContext, useCallback } from 'react';
import { View, Text, StyleSheet, FlatList, ActivityIndicator, TouchableOpacity, RefreshControl } from 'react-native';
import api from '../services/api';
import { ThemeContext } from '../context/ThemeContext';
import { LanguageContext } from '../context/LanguageContext';
import { AuthContext } from '../context/AuthContext';
import ReviewItem from '../components/ReviewItem';
import { useFocusEffect } from '@react-navigation/native';

const REVIEWS_LIMIT = 20;

const MyReviewsScreen = ({ navigation }) => {
    const [reviews, setReviews] = useState([]);
    const [loading, setLoading] = useState(true);
    const [refreshing, setRefreshing] = useState(false);
    const [offset, setOffset] = useState(0);
    const [hasMore, setHasMore] = useState(true);
    const [loadingMore, setLoadingMore] = useState(false);
    const [expandedReviews, setExpandedReviews] = useState(new Set());
    const [error, setError] = useState(false);

    const { theme } = useContext(ThemeContext);
    const { t } = useContext(LanguageContext);
    const { userToken } = useContext(AuthContext);

    const toggleReview = (reviewId) => {
        setExpandedReviews(prev => {
            const newSet = new Set(prev);
            if (newSet.has(reviewId)) {
                newSet.delete(reviewId);
            } else {
                newSet.add(reviewId);
            }
            return newSet;
        });
    };

    const fetchReviews = async (isRefresh = false) => {
        if (!userToken) return;

        try {
            const currentOffset = isRefresh ? 0 : offset;
            const res = await api.get(`/users/me/reviews?limit=${REVIEWS_LIMIT}&offset=${currentOffset}`);
            const fetchedReviews = res.data || [];

            if (isRefresh) {
                setReviews(fetchedReviews);
                setOffset(fetchedReviews.length);
            } else {
                setReviews(prev => [...prev, ...fetchedReviews]);
                setOffset(prev => prev + fetchedReviews.length);
            }
            
            setHasMore(fetchedReviews.length === REVIEWS_LIMIT);
            setError(false);
        } catch (e) {
            console.log('Failed to fetch my reviews:', e);
            if (isRefresh || reviews.length === 0) {
                setError(true);
            }
        } finally {
            setLoading(false);
            setRefreshing(false);
            setLoadingMore(false);
        }
    };

    useFocusEffect(
        useCallback(() => {
            setLoading(true);
            fetchReviews(true);
        }, [userToken])
    );

    const onRefresh = () => {
        setRefreshing(true);
        fetchReviews(true);
    };

    const loadMore = () => {
        if (!loadingMore && hasMore) {
            setLoadingMore(true);
            fetchReviews(false);
        }
    };

    const styles = React.useMemo(() => createStyles(theme), [theme]);

    const renderItem = useCallback(({ item }) => (
        <ReviewItem
            item={item}
            isExpanded={expandedReviews.has(item.review_id)}
            onToggle={() => toggleReview(item.review_id)}
            onEdit={() => navigation.navigate('BikesList', { screen: 'UpdateReview', params: { reviewId: item.review_id } })}
            showBikeId={true}
        />
    ), [expandedReviews, toggleReview, navigation]);

    if (loading && !refreshing && reviews.length === 0) {
        return (
            <View style={[styles.container, styles.centered]}>
                <ActivityIndicator size="large" color={theme.colors.primary} />
            </View>
        );
    }

    if (!userToken) {
        return (
            <View style={[styles.container, styles.centered]}>
                <Text style={styles.emptyText}>{t('must_be_logged_in_to_view_reviews') || 'You must be logged in to view your reviews.'}</Text>
            </View>
        );
    }

    return (
        <View style={styles.container}>
            <FlatList
                data={reviews}
                keyExtractor={(item) => item.review_id.toString()}
                renderItem={renderItem}
                contentContainerStyle={styles.listContent}
                refreshControl={
                    <RefreshControl refreshing={refreshing} onRefresh={onRefresh} colors={[theme.colors.primary]} />
                }
                onEndReached={loadMore}
                onEndReachedThreshold={0.5}
                ListEmptyComponent={
                    error ? (
                        <View style={[styles.container, styles.centered]}>
                            <Text style={{ color: theme.colors.error, marginBottom: 20 }}>{t('error')}</Text>
                            <Button title={t('retry') || 'Retry'} onPress={() => fetchReviews(true)} color={theme.colors.primary} />
                        </View>
                    ) : (
                        <View style={[styles.container, styles.centered]}>
                            <Text style={styles.emptyText}>{t('no_reviews') || 'No reviews found.'}</Text>
                        </View>
                    )
                }
                ListFooterComponent={() => {
                    if (error && reviews.length > 0) {
                        return (
                            <View style={{ marginVertical: 15, alignItems: 'center' }}>
                                <Text style={{ color: theme.colors.error, marginBottom: 10 }}>{t('error')}</Text>
                                <Button title={t('retry') || 'Retry'} onPress={() => fetchReviews(false)} color={theme.colors.primary} />
                            </View>
                        );
                    }
                    if (loadingMore) {
                        return <ActivityIndicator size="small" color={theme.colors.primary} style={{ marginVertical: 15 }} />;
                    }
                    return null;
                }}
            />
        </View>
    );
};

const createStyles = (theme) => StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: theme.colors.background,
    },
    centered: {
        justifyContent: 'center',
        alignItems: 'center',
    },
    listContent: {
        padding: 20,
        paddingBottom: 40,
    },
    emptyText: {
        color: theme.colors.subtext,
        textAlign: 'center',
        marginTop: 20,
    },
});

export default MyReviewsScreen;
