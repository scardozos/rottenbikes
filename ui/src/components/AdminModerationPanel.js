import React, { useState, useContext } from 'react';
import { View, Text, TextInput, TouchableOpacity, StyleSheet, Alert, Platform, ActivityIndicator } from 'react-native';
import api from '../services/api';
import { ThemeContext } from '../context/ThemeContext';
import { LanguageContext } from '../context/LanguageContext';
import { useToast } from '../context/ToastContext';
import Icon from './Icon';
import Badge from './Badge';

// Admin-only moderation panel: search users and purge malicious ones.
// A purge deletes the user, all their reviews, and the bikes they created
// (reviews by others on those bikes are deleted too). Rendered only when
// the current user has the admin role (managed via cmd/adminctl).
const AdminModerationPanel = () => {
    const { theme } = useContext(ThemeContext);
    const { t } = useContext(LanguageContext);
    const { showToast } = useToast();
    const [query, setQuery] = useState('');
    const [results, setResults] = useState(null);
    const [searching, setSearching] = useState(false);
    const [purgingId, setPurgingId] = useState(null);

    const styles = React.useMemo(() => createStyles(theme), [theme]);

    const handleSearch = async () => {
        if (!query.trim()) return;
        setSearching(true);
        try {
            const res = await api.get('/admin/users', {
                params: { q: query.trim(), limit: 20 }
            });
            setResults(res.data || []);
        } catch (e) {
            const errMsg = e.response?.data?.error || t('admin_search_error');
            showToast(errMsg, 'error');
        } finally {
            setSearching(false);
        }
    };

    const confirmPurge = (user) => {
        // The backend rejects purging admins (demote first); don't offer it.
        if (user.role === 'admin') return;

        const message = t('admin_purge_confirm_desc', {
            username: user.username,
            reviews: user.review_count,
            bikes: user.bike_count
        });

        const doPurge = async () => {
            setPurgingId(user.poster_id);
            try {
                await api.delete(`/admin/users/${user.poster_id}`);
                showToast(t('admin_purge_success'), 'success');
                setResults((prev) => (prev || []).filter((u) => u.poster_id !== user.poster_id));
            } catch (e) {
                const errMsg = e.response?.data?.error || t('admin_purge_error');
                showToast(errMsg, 'error');
            } finally {
                setPurgingId(null);
            }
        };

        if (Platform.OS === 'web') {
            if (window.confirm(message)) {
                doPurge();
            }
        } else {
            Alert.alert(t('admin_purge_confirm_title'), message, [
                { text: t('cancel'), style: 'cancel' },
                { text: t('admin_purge'), style: 'destructive', onPress: doPurge }
            ]);
        }
    };

    return (
        <View style={styles.panel}>
            <View style={styles.searchRow}>
                <TextInput
                    style={styles.searchInput}
                    placeholder={t('admin_search_placeholder')}
                    placeholderTextColor={theme.colors.subtext}
                    value={query}
                    onChangeText={setQuery}
                    autoCapitalize="none"
                    autoCorrect={false}
                    onSubmitEditing={handleSearch}
                    returnKeyType="search"
                />
                <TouchableOpacity
                    style={styles.searchButton}
                    onPress={handleSearch}
                    disabled={searching || !query.trim()}
                >
                    {searching ? (
                        <ActivityIndicator size="small" color={theme.colors.buttonText} />
                    ) : (
                        <Icon name="search" size={18} color={theme.colors.buttonText} />
                    )}
                </TouchableOpacity>
            </View>

            {results !== null && results.length === 0 && (
                <Text style={styles.noResults}>{t('admin_no_results')}</Text>
            )}

            {results !== null && results.length > 0 && (
                <View style={styles.resultsList}>
                    {results.map((user) => {
                        const isAdminUser = user.role === 'admin';
                        return (
                            <View key={user.poster_id} style={styles.userRow}>
                                <View style={styles.userInfo}>
                                    <View style={styles.usernameRow}>
                                        <Text style={styles.username}>{user.username}</Text>
                                        {isAdminUser && (
                                            <Badge label={t('admin_badge')} variant="warning" size="sm" />
                                        )}
                                    </View>
                                    <Text style={styles.userMeta}>
                                        {user.email} · {t('admin_user_stats', { reviews: user.review_count, bikes: user.bike_count })}
                                    </Text>
                                </View>
                                {isAdminUser ? (
                                    <Text style={styles.protectedText}>{t('admin_protected')}</Text>
                                ) : (
                                    <TouchableOpacity
                                        style={[styles.purgeButton, purgingId === user.poster_id && styles.purgeButtonDisabled]}
                                        onPress={() => confirmPurge(user)}
                                        disabled={purgingId === user.poster_id}
                                    >
                                        {purgingId === user.poster_id ? (
                                            <ActivityIndicator size="small" color={theme.colors.buttonText} />
                                        ) : (
                                            <Text style={styles.purgeButtonText}>{t('admin_purge')}</Text>
                                        )}
                                    </TouchableOpacity>
                                )}
                            </View>
                        );
                    })}
                </View>
            )}
        </View>
    );
};

const createStyles = (theme) => StyleSheet.create({
    panel: {
        gap: 12,
    },
    searchRow: {
        flexDirection: 'row',
        gap: 8,
    },
    searchInput: {
        flex: 1,
        backgroundColor: theme.colors.inputBackground,
        color: theme.colors.text,
        padding: 12,
        borderRadius: theme?.metrics?.radii?.md || 12,
        borderWidth: 1,
        borderColor: theme.colors.border,
    },
    searchButton: {
        backgroundColor: theme.colors.primary,
        borderRadius: theme?.metrics?.radii?.md || 12,
        paddingHorizontal: 16,
        justifyContent: 'center',
        alignItems: 'center',
    },
    noResults: {
        color: theme.colors.subtext,
        fontSize: 14,
        textAlign: 'center',
        paddingVertical: 8,
    },
    resultsList: {
        gap: 8,
    },
    userRow: {
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'space-between',
        gap: 10,
        padding: 12,
        borderRadius: theme?.metrics?.radii?.md || 12,
        backgroundColor: theme.colors.inputBackground,
        borderWidth: 1,
        borderColor: theme.colors.border,
    },
    userInfo: {
        flex: 1,
    },
    usernameRow: {
        flexDirection: 'row',
        alignItems: 'center',
        gap: 8,
    },
    username: {
        color: theme.colors.text,
        fontWeight: '700',
        fontSize: 15,
    },
    protectedText: {
        color: theme.colors.subtext,
        fontSize: 12,
        fontStyle: 'italic',
    },
    userMeta: {
        color: theme.colors.subtext,
        fontSize: 12,
        marginTop: 2,
    },
    purgeButton: {
        backgroundColor: theme.colors.error,
        borderRadius: theme?.metrics?.radii?.sm || 8,
        paddingVertical: 8,
        paddingHorizontal: 14,
    },
    purgeButtonDisabled: {
        opacity: 0.5,
    },
    purgeButtonText: {
        color: theme.colors.buttonText,
        fontWeight: 'bold',
        fontSize: 13,
    },
});

export default AdminModerationPanel;
