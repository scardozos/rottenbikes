import { useFocusEffect } from '@react-navigation/native';
import React, { useEffect, useState, useContext, useCallback } from 'react';
import { View, Text, FlatList, TouchableOpacity, StyleSheet, Button, ActivityIndicator, TextInput } from 'react-native';
import api from '../services/api';
import { AuthContext } from '../context/AuthContext';
import { ThemeContext } from '../context/ThemeContext';
import { LanguageContext } from '../context/LanguageContext';

const BikesListScreen = ({ navigation }) => {
    const [bikes, setBikes] = useState([]);
    const [filteredBikes, setFilteredBikes] = useState([]);
    const [searchQuery, setSearchQuery] = useState('');
    const [loading, setLoading] = useState(true);
    const [loadingMore, setLoadingMore] = useState(false);
    const [hasMore, setHasMore] = useState(true);
    const { theme } = useContext(ThemeContext);
    const { t } = useContext(LanguageContext);

    const fetchBikes = async (isRefresh = false) => {
        if (loadingMore && !isRefresh) return;
        const nextOffset = isRefresh ? 0 : bikes.length;
        if (!isRefresh && !hasMore) return;

        if (isRefresh) {
            setLoading(true);
        } else {
            setLoadingMore(true);
        }

        try {
            const limit = 20;
            const bikesRes = await api.get(`/bikes?limit=${limit}&offset=${nextOffset}`);
            const data = bikesRes.data || [];

            if (data.length < limit) {
                setHasMore(false);
            } else {
                setHasMore(true);
            }

            if (isRefresh) {
                setBikes(data);
                setFilteredBikes(data);
            } else {
                const combined = [...bikes, ...data];
                setBikes(combined);
                setFilteredBikes(combined);
            }
        } catch (e) {
            console.error('Fetch bikes error:', e);
        } finally {
            setLoading(false);
            setLoadingMore(false);
        }
    };

    useFocusEffect(
        useCallback(() => {
            fetchBikes(true);
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
            onPress={() => navigation.navigate('BikeDetails', { bikeId: item.numerical_id })}
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

    // Kept for manual search creation if needed, though mostly handled in Home now.
    // Users can still search list here.
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
                <Text style={styles.title}>{t('all_bikes')}</Text>
            </View>

            <View style={styles.searchContainer}>
                <TextInput
                    style={styles.searchInput}
                    placeholder={t('search_list_placeholder')}
                    placeholderTextColor={theme.colors.placeholder}
                    value={searchQuery}
                    onChangeText={setSearchQuery}
                    clearButtonMode="while-editing"
                />
            </View>

            {loading && bikes.length === 0 ? <ActivityIndicator size="large" color={theme.colors.primary} style={{ marginTop: 20 }} /> : (
                <FlatList
                    data={filteredBikes}
                    keyExtractor={item => item.numerical_id.toString()}
                    renderItem={renderItem}
                    onEndReached={() => {
                        if (!searchQuery) {
                            fetchBikes(false);
                        }
                    }}
                    onEndReachedThreshold={0.5}
                    onRefresh={() => fetchBikes(true)}
                    refreshing={loading}
                    ListFooterComponent={() => {
                        if (loadingMore) {
                            return <ActivityIndicator size="small" color={theme.colors.primary} style={{ marginVertical: 15 }} />;
                        }
                        return null;
                    }}
                    ListEmptyComponent={
                        <View style={styles.emptyContainer}>
                            <Text style={styles.emptyText}>
                                {searchQuery ? t('no_bikes_found', { query: searchQuery }) : t('no_bikes_available')}
                            </Text>
                        </View>
                    }
                />
            )}
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
    item: { padding: 20, borderBottomWidth: 1, borderBottomColor: theme.colors.border },
    itemHeader: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' },
    itemText: { fontSize: 24, fontWeight: 'bold', color: theme.colors.text },
    ratingBadge: { fontSize: 20, fontWeight: 'bold', color: '#f39c12', backgroundColor: theme.colors.card, paddingHorizontal: 10, paddingVertical: 4, borderRadius: 12, overflow: 'hidden' },
    subText: { fontSize: 18, color: theme.colors.subtext, marginTop: 5 },
    emptyContainer: { alignItems: 'center', marginTop: 30 },
    emptyText: { fontSize: 18, color: theme.colors.subtext, marginBottom: 20 }
});

export default BikesListScreen;
