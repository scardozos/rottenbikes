import { useFocusEffect } from '@react-navigation/native';
import React, { useState, useCallback } from 'react';
import { View, Text, StyleSheet, FlatList, Button } from 'react-native';
import api from '../services/api';

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
    const { bike } = route.params;
    const [reviews, setReviews] = useState([]);
    const [aggregates, setAggregates] = useState([]);
    const [loading, setLoading] = useState(true);

    const fetchData = async () => {
        setLoading(true);
        try {
            const [reviewsRes, aggregatesRes] = await Promise.all([
                api.get(`/bikes/${bike.numerical_id}/reviews`),
                api.get(`/bikes/${bike.numerical_id}/ratings`)
            ]);
            setReviews(reviewsRes.data || []);
            setAggregates(aggregatesRes.data || []);
        } catch (e) {
            console.log("Failed to fetch bike details/reviews", e);
        } finally {
            setLoading(false);
        }
    };

    useFocusEffect(
        useCallback(() => {
            fetchData();
        }, [])
    );

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
                            <Text>{item.comment}</Text>
                            <Text style={styles.user}>- {item.poster_username || 'Anonymous'}</Text>
                        </View>
                    )}
                    ListEmptyComponent={<Text>No reviews yet.</Text>}
                />
            </View>
            <Button
                title="Write a Review"
                onPress={() => navigation.navigate('CreateReview', { bike })}
            />
        </View>
    );
};

const styles = StyleSheet.create({
    container: { flex: 1, padding: 20 },
    title: { fontSize: 24, fontWeight: 'bold', marginBottom: 10 },
    detail: { fontSize: 16, marginBottom: 5 },
    reviewsSection: { flex: 1, marginTop: 20 },
    aggregatesSection: { marginTop: 20, backgroundColor: '#f9f9f9', padding: 15, borderRadius: 10 },
    aggregatesGrid: { flexDirection: 'row', flexWrap: 'wrap', justifyContent: 'space-between' },
    aggItem: { width: '48%', backgroundColor: '#fff', padding: 10, borderRadius: 8, marginBottom: 10, alignItems: 'center', shadowColor: '#000', shadowOffset: { width: 0, height: 1 }, shadowOpacity: 0.1, shadowRadius: 2, elevation: 2 },
    aggLabel: { fontSize: 14, color: '#666', marginBottom: 2 },
    aggValue: { fontSize: 16, fontWeight: 'bold', color: '#f39c12' },
    subtitle: { fontSize: 20, marginBottom: 10, fontWeight: '600' },
    reviewItem: { padding: 10, borderBottomWidth: 1, borderBottomColor: '#eee', marginBottom: 10 },
    reviewHeader: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginBottom: 5 },
    timeText: { fontSize: 14, color: '#999' },
    rating: { fontSize: 18 },
    user: { fontStyle: 'italic', marginTop: 5, color: '#666' }
});

export default BikeDetailsScreen;
