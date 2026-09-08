import React, { useState, useEffect, useContext, useCallback } from 'react';
import { View, Text, StyleSheet, FlatList, ActivityIndicator, RefreshControl } from 'react-native';
import api from '../services/api';
import { ThemeContext } from '../context/ThemeContext';
import { LanguageContext } from '../context/LanguageContext';
import { AuthContext } from '../context/AuthContext';
import ReviewItem from '../components/ReviewItem';
import Button from '../components/Button';
import EmptyState from '../components/EmptyState';
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
            onEdit={() => navigation.navigate('UpdateReview', { reviewId: item.review_id })}
            showBikeId={true}
            onPressBike={() => navigation.navigate('BikeDetails', { bikeId: item.bike_numerical_id })}
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
            <View style={styles.container}>
                <EmptyState
                    icon="lock-closed-outline"
                    title={t('must_be_logged_in_to_view_reviews')}
                    action={
                        <Button
                            title={t('login')}
                            onPress={() => navigation.navigate('PublicHome', { screen: 'Login' })}
                            variant="primary"
                        />
                    }
                />
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
                contentInsetAdjustmentBehavior="never"
                automaticallyAdjustContentInsets={false}
                automaticallyAdjustsScrollIndicatorInsets={false}
                refreshControl={
                    <RefreshControl refreshing={refreshing} onRefresh={onRefresh} colors={[theme.colors.primary]} />
                }
                onEndReached={loadMore}
                onEndReachedThreshold={0.5}
                ListEmptyComponent={
                    error ? (
                        <EmptyState
                            icon="alert-circle-outline"
                            title={t('error')}
                            action={
                                <Button
                                    title={t('retry') || 'Retry'}
                                    onPress={() => fetchReviews(true)}
                                    variant="primary"
                                    size="sm"
                                />
                            }
                        />
                    ) : (
                        <EmptyState
                            icon="star-outline"
                            title={t('no_reviews')}
                            action={
                                <Button
                                    title={t('browse_bikes')}
                                    onPress={() => navigation.navigate('BikesList', { screen: 'BikesCatalog' })}
                                    variant="primary"
                                    size="sm"
                                />
                            }
                        />
                    )
                }
                ListFooterComponent={() => {
                    if (error && reviews.length > 0) {
                        return (
                            <View style={styles.footerAction}>
                                <Text style={styles.errorText}>{t('error')}</Text>
                                <Button title={t('retry') || 'Retry'} onPress={() => fetchReviews(false)} variant="primary" size="sm" />
                            </View>
                        );
                    }
                    if (loadingMore) {
                        return <ActivityIndicator size="small" color={theme.colors.primary} style={styles.footerLoading} />;
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
        flex: 1,
        justifyContent: 'center',
        alignItems: 'center',
    },
    listContent: {
        paddingHorizontal: theme?.metrics?.spacing?.lg || 16,
        paddingTop: theme?.metrics?.spacing?.md || 12,
        paddingBottom: theme?.metrics?.spacing?.xxl || 32,
    },
    footerAction: {
        marginVertical: 15,
        alignItems: 'center',
    },
    footerLoading: {
        marginVertical: 15,
    },
    errorText: {
        color: theme.colors.error,
        marginBottom: 10,
    },
});

export default MyReviewsScreen;
