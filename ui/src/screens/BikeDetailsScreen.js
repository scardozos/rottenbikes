import { useFocusEffect } from '@react-navigation/native';
import React, { useState, useCallback, useContext } from 'react';
import { View, Text, StyleSheet, FlatList, Button } from 'react-native';
import api from '../services/api';
import { ThemeContext } from '../context/ThemeContext';

const getRelativeTime = (dateString) => {
    if (!dateString) return '';
    const now = new Date();
    const then = new Date(dateString);
    const seconds = Math.floor((now - then) / 1000);

    if (seconds < 60) return 'just now';
    const minutes = Math.floor(seconds / 60);
    if (minutes < 60) return `${minutes}m ago`;
    const hours = Math.floor(minutes / 60);
    if (hours < 24) return `${hours}h ago`;
    const days = Math.floor(hours / 24);
    if (days < 7) return `${days}d ago`;
    const weeks = Math.floor(days / 7);
    if (weeks < 4) return `${weeks}w ago`;
    const months = Math.floor(days / 30);
    if (months < 12) return `${months}mo ago`;
    const years = Math.floor(days / 365);
    return `${years}y ago`;
};

const BikeDetailsScreen = ({ route, navigation }) => {
    const params = route.params || {};
    // Handle both object navigation and deep linking ID
    const initialBike = params.bike || { numerical_id: params.bikeId };
    const [bike, setBike] = useState(initialBike);
    const [reviews, setReviews] = useState([]);
    const [aggregates, setAggregates] = useState([]);
    const [loading, setLoading] = useState(true);
    const { theme } = useContext(ThemeContext);

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
            <Text style={styles.title}>Bike #{bike.numerical_id}</Text>
            <Text style={styles.detail}>Hash ID: {bike.hash_id}</Text>
            <Text style={styles.detail}>Type: {bike.is_electric ? 'Electric ⚡' : 'Mechanical 🚲'}</Text>

            {aggregates.length > 0 && (
                <View style={styles.aggregatesSection}>
                    <Text style={styles.subtitle}>Average Ratings</Text>
                    <View style={styles.aggregatesGrid}>
                        {aggregates
                            .filter(agg => agg.subcategory !== 'overall')
                            .map(agg => (
                                <View key={agg.subcategory} style={styles.aggItem}>
                                    <Text style={styles.aggLabel}>{agg.subcategory.charAt(0).toUpperCase() + agg.subcategory.slice(1)}</Text>
                                    <Text style={styles.aggValue}>{agg.average_rating.toFixed(1)} ⭐</Text>
                                </View>
                            ))}
                    </View>
                </View>
            )}

            <View style={styles.reviewsSection}>
                <Text style={styles.subtitle}>Reviews</Text>
                <FlatList
                    data={reviews}
                    keyExtractor={item => item.review_id ? item.review_id.toString() : Math.random().toString()}
                    renderItem={({ item }) => (
                        <View style={styles.reviewItem}>
                            <View style={styles.reviewHeader}>
                                <Text style={styles.rating}>{'⭐'.repeat(item.ratings?.overall || 0)}</Text>
                                <Text style={styles.timeText}>{getRelativeTime(item.created_at)}</Text>
                            </View>
                            <Text style={styles.commentText}>{item.comment}</Text>
                            <Text style={styles.user}>- {item.poster_username || 'Anonymous'}</Text>
                        </View>
                    )}
                    ListEmptyComponent={<Text style={styles.emptyText}>No reviews yet.</Text>}
                />
            </View>
            <Button
                title="Write a Review"
                onPress={() => navigation.navigate('CreateReview', { bike })}
                color={theme.colors.primary}
            />
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
    emptyText: { color: theme.colors.subtext }
});

export default BikeDetailsScreen;
