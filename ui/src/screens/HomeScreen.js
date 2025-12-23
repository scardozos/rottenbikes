import { useFocusEffect } from '@react-navigation/native';
import React, { useEffect, useState, useContext, useCallback } from 'react';
import { View, Text, FlatList, TouchableOpacity, StyleSheet, Button, ActivityIndicator, TextInput, Platform } from 'react-native';
import api from '../services/api';
import { AuthContext } from '../context/AuthContext';
import { ThemeContext } from '../context/ThemeContext';

const HomeScreen = ({ navigation }) => {
    const [bikes, setBikes] = useState([]);
    const [filteredBikes, setFilteredBikes] = useState([]);
    const [searchQuery, setSearchQuery] = useState('');
    const [loading, setLoading] = useState(true);
    const { logout } = useContext(AuthContext);
    const { theme } = useContext(ThemeContext);

    const fetchBikes = async () => {
        try {
            const bikesRes = await api.get('/bikes');
            setBikes(bikesRes.data);
            setFilteredBikes(bikesRes.data);
        } catch (e) {
            console.error('Fetch bikes error:', e);
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

    const styles = createStyles(theme);

    const renderItem = ({ item }) => (
        <TouchableOpacity
            style={styles.item}
            onPress={() => navigation.navigate('BikeDetails', { bike: item })}
        >
            <View style={styles.itemHeader}>
                <Text style={styles.itemText}>
                    #{item.numerical_id} {item.is_electric ? '⚡' : '🚲'}
                </Text>
                {item.average_rating != null && (
                    <Text style={styles.ratingBadge}>{item.average_rating.toFixed(1)} ⭐</Text>
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
                <Button title="Logout" onPress={logout} color={theme.colors.error} />
            </View>

            <View style={styles.searchContainer}>
                <TextInput
                    style={styles.searchInput}
                    placeholder="Search by ID or Hash..."
                    placeholderTextColor={theme.colors.placeholder}
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

            {loading ? <ActivityIndicator size="large" color={theme.colors.primary} /> : (
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
                                    color={theme.colors.primary}
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

const createStyles = (theme) => StyleSheet.create({
    container: { flex: 1, padding: 20, backgroundColor: theme.colors.background },
    header: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 },
    title: { fontSize: 24, fontWeight: 'bold', color: theme.colors.text },
    searchInput: {
        flex: 1,
        height: 50,
        borderColor: theme.colors.border,
        borderWidth: 1,
        borderRadius: 8,
        paddingHorizontal: 15,
        backgroundColor: theme.colors.inputBackground,
        fontSize: 16,
        color: theme.colors.text
    },
    searchContainer: {
        flexDirection: 'row',
        alignItems: 'center',
        marginBottom: 20,
    },
    qrButton: {
        marginLeft: 10,
        backgroundColor: theme.colors.inputBackground,
        height: 50,
        width: 50,
        borderRadius: 8,
        justifyContent: 'center',
        alignItems: 'center',
        borderWidth: 1,
        borderColor: theme.colors.border
    },
    qrButtonText: {
        fontSize: 24
    },
    item: { padding: 20, borderBottomWidth: 1, borderBottomColor: theme.colors.border },
    itemHeader: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' },
    itemText: { fontSize: 24, fontWeight: 'bold', color: theme.colors.text },
    ratingBadge: { fontSize: 20, fontWeight: 'bold', color: '#f39c12', backgroundColor: theme.colors.card, paddingHorizontal: 10, paddingVertical: 4, borderRadius: 12, overflow: 'hidden' }, // Helper color for badge
    subText: { fontSize: 18, color: theme.colors.subtext, marginTop: 5 },
    emptyContainer: { alignItems: 'center', marginTop: 30 },
    emptyText: { fontSize: 18, color: theme.colors.subtext, marginBottom: 20 }
});

export default HomeScreen;
