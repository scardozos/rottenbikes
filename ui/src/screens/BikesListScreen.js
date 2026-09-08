import { useFocusEffect } from '@react-navigation/native';
import React, { useEffect, useState, useContext, useCallback } from 'react';
import { View, Text, FlatList, TouchableOpacity, StyleSheet, ActivityIndicator, Platform, RefreshControl } from 'react-native';
import api from '../services/api';
import { ThemeContext } from '../context/ThemeContext';
import { LanguageContext } from '../context/LanguageContext';
import SortDropdown from '../components/SortDropdown';
import { isNumeric } from '../utils/validation';
import { uniqueBy } from '../utils/reviews';
import Button from '../components/Button';
import Icon from '../components/Icon';
import Badge from '../components/Badge';
import Input from '../components/Input';
import EmptyState from '../components/EmptyState';

const BikesListScreen = ({ navigation }) => {
    const [bikes, setBikes] = useState([]);
    const [searchQuery, setSearchQuery] = useState('');
    const [serverSearchQuery, setServerSearchQuery] = useState('');
    const [sortOption, setSortOption] = useState('recent');
    const [loading, setLoading] = useState(true);
    const [refreshing, setRefreshing] = useState(false);
    const [loadingMore, setLoadingMore] = useState(false);
    const [hasMore, setHasMore] = useState(true);
    const [error, setError] = useState(false);
    const { theme } = useContext(ThemeContext);
    const { t } = useContext(LanguageContext);

    const fetchBikes = async (isRefresh = false) => {
        if (loadingMore && !isRefresh) return;
        const nextOffset = isRefresh ? 0 : bikes.length;
        if (!isRefresh && !hasMore) return;

        if (isRefresh) {
            if (bikes.length === 0) {
                setLoading(true);
            }
        } else {
            setLoadingMore(true);
        }

        try {
            const limit = 20;
            const bikesRes = await api.get(`/bikes?limit=${limit}&offset=${nextOffset}&q=${encodeURIComponent(serverSearchQuery)}&sort=${sortOption}`);
            const data = bikesRes.data || [];

            if (data.length < limit) {
                setHasMore(false);
            } else {
                setHasMore(true);
            }

            if (isRefresh) {
                setBikes(data);
            } else {
                setBikes(prev => {
                    const combined = [...prev, ...data];
                    return uniqueBy(combined, (v) => v.numerical_id);
                });
            }
            setError(false);
        } catch (e) {
            console.error('Fetch bikes error:', e);
            if (isRefresh || bikes.length === 0) {
                setError(true);
            }
        } finally {
            setLoading(false);
            setRefreshing(false);
            setLoadingMore(false);
        }
    };

    const onRefresh = () => {
        setRefreshing(true);
        fetchBikes(true);
    };

    useFocusEffect(
        useCallback(() => {
            fetchBikes(true);
        }, [serverSearchQuery, sortOption])
    );

    // Debounce search query before sending to server
    useEffect(() => {
        const timeoutId = setTimeout(() => {
            setServerSearchQuery(searchQuery);
        }, 500);
        return () => clearTimeout(timeoutId);
    }, [searchQuery]);

    const styles = React.useMemo(() => createStyles(theme), [theme]);

    const renderItem = useCallback(({ item }) => (
        <TouchableOpacity
            style={styles.card}
            onPress={() => navigation.navigate('BikeDetails', { bikeId: item.numerical_id })}
            activeOpacity={0.7}
            accessibilityRole="button"
            accessibilityLabel={`Bike ${item.numerical_id}`}
        >
            <View style={styles.cardContent}>
                <View style={styles.cardHeaderRow}>
                    <Text style={styles.bikeNumber}>#{item.numerical_id}</Text>
                    <View style={[styles.typeBadge, item.is_electric ? styles.electricBadge : styles.mechanicalBadge]}>
                        <Icon
                            name={item.is_electric ? 'flash' : 'bicycle'}
                            size={14}
                            color={item.is_electric ? theme.colors.warning : theme.colors.primary}
                        />
                        <Text style={[styles.typeText, { color: item.is_electric ? theme.colors.warning : theme.colors.primary }]}>
                            {item.is_electric ? t('electric') : t('mechanical')}
                        </Text>
                    </View>
                </View>

                <Text style={styles.hashText} numberOfLines={1} ellipsizeMode="middle">
                    {item.hash_id || '—'}
                </Text>
            </View>

            <View style={styles.cardRight}>
                {item.average_rating != null ? (
                    <Badge
                        variant="warning"
                        size="md"
                        icon={<Icon name="star" size={13} color={theme.colors.warning} />}
                        label={item.average_rating.toFixed(1)}
                    />
                ) : (
                    <Badge
                        variant="default"
                        size="md"
                        label={t('no_reviews')}
                    />
                )}
                <Icon name="chevron-forward" size={18} color={theme.colors.subtext} style={styles.chevron} />
            </View>
        </TouchableOpacity>
    ), [styles, navigation, theme, t]);

    const handleCreateSearchBike = () => {
        const numeric = isNumeric(searchQuery);
        const params = {};
        if (numeric) {
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

            <Input
                placeholder={t('search_list_placeholder')}
                value={searchQuery}
                onChangeText={setSearchQuery}
                clearButtonMode="while-editing"
                leftIcon={<Icon name="search-outline" size={20} color={theme.colors.subtext} />}
                containerStyle={styles.searchContainer}
                style={styles.searchInputWrapper}
            />

            <SortDropdown selectedSort={sortOption} onSortChange={setSortOption} />

            {loading && bikes.length === 0 ? (
                <View style={styles.centerLoading}>
                    <ActivityIndicator size="large" color={theme.colors.primary} />
                </View>
            ) : (
                <FlatList
                    data={bikes}
                    keyExtractor={item => item.numerical_id.toString()}
                    renderItem={renderItem}
                    contentContainerStyle={styles.listContent}
                    contentInsetAdjustmentBehavior="never"
                    automaticallyAdjustContentInsets={false}
                    automaticallyAdjustsScrollIndicatorInsets={false}
                    refreshControl={
                        <RefreshControl refreshing={refreshing} onRefresh={onRefresh} colors={[theme.colors.primary]} />
                    }
                    contentOffset={{ x: 0, y: 0 }}
                    onEndReached={() => {
                        if (!loadingMore && hasMore && !error) {
                            fetchBikes(false);
                        }
                    }}
                    onEndReachedThreshold={0.5}
                    ListFooterComponent={() => {
                        if (error && bikes.length > 0) {
                            return (
                                <View style={styles.footerAction}>
                                    <Text style={styles.errorText}>{t('error')}</Text>
                                    <Button title={t('retry') || 'Retry'} onPress={() => fetchBikes(false)} variant="primary" size="sm" />
                                </View>
                            );
                        }
                        if (loadingMore) {
                            return <ActivityIndicator size="small" color={theme.colors.primary} style={styles.footerLoading} />;
                        }
                        if (!hasMore && bikes.length > 0) {
                            return <Text style={styles.endText}>{t('no_more_bikes')}</Text>;
                        }
                        return null;
                    }}
                    ListEmptyComponent={() => {
                        if (error) {
                            return (
                                <EmptyState
                                    icon="alert-circle-outline"
                                    title={t('error')}
                                    description={t('scan_lookup_failed')}
                                    action={<Button title={t('retry') || 'Retry'} onPress={() => fetchBikes(true)} variant="primary" />}
                                />
                            );
                        }
                        if (!loading) {
                            return (
                                <EmptyState
                                    icon="bicycle-outline"
                                    title={searchQuery ? t('no_bikes_found', { query: searchQuery }) : t('no_bikes_available')}
                                    action={
                                        isNumeric(searchQuery) ? (
                                            <Button
                                                title={t('create_new_bike', { numerical_id: searchQuery })}
                                                onPress={handleCreateSearchBike}
                                                variant="primary"
                                                size="sm"
                                            />
                                        ) : null
                                    }
                                />
                            );
                        }
                        return null;
                    }}
                />
            )}
        </View>
    );
};

const createStyles = (theme) => StyleSheet.create({
    container: {
        flex: 1,
        paddingHorizontal: theme?.metrics?.spacing?.lg || 16,
        paddingTop: theme?.metrics?.spacing?.md || 12,
        backgroundColor: theme.colors.background,
    },
    header: {
        marginBottom: theme?.metrics?.spacing?.md || 12,
    },
    title: {
        fontSize: theme?.typography?.h2?.fontSize || 24,
        fontWeight: theme?.typography?.h2?.fontWeight || 'bold',
        color: theme.colors.text,
    },
    searchContainer: {
        marginBottom: theme?.metrics?.spacing?.sm || 8,
    },
    searchInputWrapper: {
        borderRadius: theme?.metrics?.radii?.pill || 9999,
        paddingHorizontal: theme?.metrics?.spacing?.md || 12,
    },
    listContent: {
        paddingBottom: theme?.metrics?.spacing?.xxl || 32,
    },
    card: {
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'space-between',
        backgroundColor: theme.colors.card,
        borderRadius: theme?.metrics?.radii?.lg || 16,
        padding: theme?.metrics?.spacing?.md || 14,
        marginBottom: theme?.metrics?.spacing?.sm || 10,
        borderWidth: 1,
        borderColor: theme.colors.border,
        ...(theme?.shadows?.sm || {}),
    },
    cardContent: {
        flex: 1,
        marginRight: theme?.metrics?.spacing?.sm || 8,
    },
    cardHeaderRow: {
        flexDirection: 'row',
        alignItems: 'center',
        marginBottom: 4,
    },
    bikeNumber: {
        fontSize: 18,
        fontWeight: '700',
        color: theme.colors.text,
        marginRight: 8,
    },
    typeBadge: {
        flexDirection: 'row',
        alignItems: 'center',
        paddingHorizontal: 8,
        paddingVertical: 2,
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
        fontSize: 12,
        color: theme.colors.subtext,
        fontFamily: Platform.OS === 'ios' ? 'Menlo' : 'monospace',
    },
    cardRight: {
        flexDirection: 'row',
        alignItems: 'center',
        gap: 8,
    },
    chevron: {
        marginLeft: 2,
    },
    centerLoading: {
        marginTop: 40,
        alignItems: 'center',
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
    endText: {
        textAlign: 'center',
        color: theme.colors.subtext,
        marginVertical: 16,
        fontSize: 14,
    },
});

export default BikesListScreen;
