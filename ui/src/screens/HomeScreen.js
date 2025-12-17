import { useFocusEffect } from '@react-navigation/native';
import React, { useEffect, useState, useContext, useCallback } from 'react';
import { View, Text, FlatList, TouchableOpacity, StyleSheet, Button, ActivityIndicator, TextInput, Platform } from 'react-native';
import api from '../services/api';
import { AuthContext } from '../context/AuthContext';

const HomeScreen = ({ navigation }) => {
    const [bikes, setBikes] = useState([]);
    const [filteredBikes, setFilteredBikes] = useState([]);
    const [searchQuery, setSearchQuery] = useState('');
    const [loading, setLoading] = useState(true);
    const { logout } = useContext(AuthContext);

    const fetchBikes = async () => {
        try {
            const [bikesRes, ratingsRes] = await Promise.all([
                api.get('/bikes'),
                api.get('/bikes/ratings')
            ]);

            const bikesData = bikesRes.data;
            const ratingsData = ratingsRes.data || [];

            // Map ratings for efficient lookup (filter for 'overall' subcategory)
            const overallRatings = ratingsData
                .filter(r => r.subcategory === 'overall')
                .reduce((acc, curr) => {
                    acc[curr.bike_numerical_id] = curr.average_rating;
                    return acc;
                }, {});

            const merged = bikesData.map(bike => ({
                ...bike,
                overallRating: overallRatings[bike.numerical_id] || null
            }));

            setBikes(merged);
            setFilteredBikes(merged);
        } catch (e) {
            console.error('Fetch bikes/ratings error:', e);
        } finally {
            setLoading(false);
        }
    };

    useFocusEffect(
        useCallback(() => {
            fetchBikes();
        }, [])
    );

    useEffect(() => {
        if (!searchQuery) {
            setFilteredBikes(bikes);
        } else {
            const query = searchQuery.toLowerCase();
            const filtered = bikes.filter(bike =>
                bike.numerical_id.toString().includes(query) ||
                (bike.hash_id && bike.hash_id.toLowerCase().includes(query))
            );
            setFilteredBikes(filtered);
        }
    }, [searchQuery, bikes]);

    const renderItem = ({ item }) => (
        <TouchableOpacity
            style={styles.item}
            onPress={() => navigation.navigate('BikeDetails', { bike: item })}
        >
            <View style={styles.itemHeader}>
                <Text style={styles.itemText}>
                    #{item.numerical_id} {item.is_electric ? '⚡' : '🚲'}
                </Text>
                {item.overallRating !== null && (
                    <Text style={styles.ratingBadge}>{item.overallRating.toFixed(1)} ⭐</Text>
                )}
            </View>
            <Text style={styles.subText}>{item.hash_id}</Text>
        </TouchableOpacity>
    );

    const handleCreateSearchBike = () => {
        const isNumeric = /^\d+$/.test(searchQuery);
        const params = {};
        if (isNumeric) {
            params.initialNumericalId = searchQuery;
        } else {
            params.initialHashId = searchQuery;
        }
        navigation.navigate('CreateBike', params);
    };

    return (
        <View style={styles.container}>
            <View style={styles.header}>
                <Text style={styles.title}>Bikes</Text>
                <Button title="Logout" onPress={logout} />
            </View>

            <View style={styles.searchContainer}>
                <TextInput
                    style={styles.searchInput}
                    placeholder="Search by ID or Hash..."
                    value={searchQuery}
                    onChangeText={setSearchQuery}
                    clearButtonMode="while-editing"
                />
                {Platform.OS !== 'web' && (
                    <TouchableOpacity
                        style={styles.qrButton}
                        onPress={() => navigation.navigate('Scanner')}
                    >
                        <Text style={styles.qrButtonText}>📷</Text>
                    </TouchableOpacity>
                )}
            </View>

            {loading ? <ActivityIndicator size="large" /> : (
                <FlatList
                    data={filteredBikes}
                    keyExtractor={item => item.numerical_id.toString()}
                    renderItem={renderItem}
                    ListEmptyComponent={
                        <View style={styles.emptyContainer}>
                            <Text style={styles.emptyText}>
                                {searchQuery ? `No bikes found for "${searchQuery}"` : "No bikes available"}
                            </Text>
                            {searchQuery && (
                                <Button
                                    title={`Create Bike "${searchQuery}"`}
                                    onPress={handleCreateSearchBike}
                                />
                            )}
                        </View>
                    }
                />
            )}
            {/* Legacy Add Bike button removed */}
        </View>
    );
};

const styles = StyleSheet.create({
    container: { flex: 1, padding: 20 },
    header: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 },
    title: { fontSize: 24, fontWeight: 'bold' },
    searchInput: {
        flex: 1,
        height: 50,
        borderColor: '#ccc',
        borderWidth: 1,
        borderRadius: 8,
        paddingHorizontal: 15,
        backgroundColor: '#fff',
        fontSize: 16
    },
    searchContainer: {
        flexDirection: 'row',
        alignItems: 'center',
        marginBottom: 20,
    },
    qrButton: {
        marginLeft: 10,
        backgroundColor: '#eee',
        height: 50,
        width: 50,
        borderRadius: 8,
        justifyContent: 'center',
        alignItems: 'center',
        borderWidth: 1,
        borderColor: '#ccc'
    },
    qrButtonText: {
        fontSize: 24
    },
    item: { padding: 20, borderBottomWidth: 1, borderBottomColor: '#ccc' },
    itemHeader: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' },
    itemText: { fontSize: 24, fontWeight: 'bold' },
    ratingBadge: { fontSize: 20, fontWeight: 'bold', color: '#f39c12', backgroundColor: '#fff3e0', paddingHorizontal: 10, paddingVertical: 4, borderRadius: 12 },
    subText: { fontSize: 18, color: '#666', marginTop: 5 },
    emptyContainer: { alignItems: 'center', marginTop: 30 },
    emptyText: { fontSize: 18, color: '#666', marginBottom: 20 }
});

export default HomeScreen;
