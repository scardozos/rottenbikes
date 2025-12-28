import React, { useState, useEffect, useContext } from 'react';
import { View, Text, TextInput, Button, StyleSheet, Switch, Alert, ScrollView, TouchableOpacity, KeyboardAvoidingView, Platform, Modal } from 'react-native';
import api from '../services/api';
import { useToast } from '../context/ToastContext';
import { ThemeContext } from '../context/ThemeContext';
import { LanguageContext } from '../context/LanguageContext';

const StarRating = ({ label, value, onValueChange, theme, styles, onInfoPress }) => (
    <View style={styles.ratingRow}>
        <View style={styles.labelRow}>
            <Text style={styles.ratingLabel}>{label}</Text>
            <TouchableOpacity onPress={onInfoPress} style={styles.infoButton} hitSlop={{ top: 10, bottom: 10, left: 10, right: 10 }}>
                <Text style={[styles.infoIcon, { color: theme.colors.primary }]}>ⓘ</Text>
            </TouchableOpacity>
        </View>
        <View style={styles.starsContainer}>
            {[1, 2, 3, 4, 5].map((star) => (
                <TouchableOpacity key={star} onPress={() => onValueChange(star)}>
                    <Text style={[styles.star, { color: star <= value ? '#f39c12' : theme.colors.border }]}>
                        ★
                    </Text>
                </TouchableOpacity>
            ))}
        </View>
    </View>
);

const UpdateReviewScreen = ({ route, navigation }) => {
    const { review, bikeId } = route.params;
    const { showToast } = useToast();
    const { theme } = useContext(ThemeContext);
    const { t } = useContext(LanguageContext);

    // Subcategories
    const [breaks, setBreaks] = useState(review.ratings?.breaks || 0);
    const [seat, setSeat] = useState(review.ratings?.seat || 0);
    const [sturdiness, setSturdiness] = useState(review.ratings?.sturdiness || 0);
    const [power, setPower] = useState(review.ratings?.power || 0);
    const [pedals, setPedals] = useState(review.ratings?.pedals || 0);

    const [overall, setOverall] = useState(review.ratings?.overall || null);
    const [comment, setComment] = useState(review.comment || '');
    const [loading, setLoading] = useState(false);

    // Modal State
    const [modalVisible, setModalVisible] = useState(false);
    const [activeCategory, setActiveCategory] = useState(null);

    const handleInfoPress = (category) => {
        setActiveCategory(category);
        setModalVisible(true);
    };

    // Auto-calculate overall
    useEffect(() => {
        const ratings = [breaks, seat, sturdiness, power, pedals].filter(r => r > 0);
        if (ratings.length === 0) {
            // Keep existing overall if no sub-ratings changed? 
            // Better to re-calculate if user is actively changing things.
            // If user initially had just overall rating (legacy?), we might lose it if we recalculate based on empty subratings.
            // But new reviews enforce subratings. Assuming existing reviews might have subratings.
            // If user clears all stars, overall becomes null.
            return;
        }
        const sum = ratings.reduce((a, b) => a + b, 0);
        const avg = sum / ratings.length;
        setOverall(avg);
    }, [breaks, seat, sturdiness, power, pedals]);

    const handleSubmit = async () => {
        const ratings = [breaks, seat, sturdiness, power, pedals];
        const hasRating = ratings.some(r => r > 0);

        if (!hasRating) {
            showToast(t('please_rate'), "error");
            return;
        }

        setLoading(true);
        try {
            const payload = {
                comment: comment,
                overall: overall ? Math.round(overall) : undefined
            };

            // Only include non-null ratings in payload
            if (breaks > 0) payload.breaks = breaks;
            if (seat > 0) payload.seat = seat;
            if (sturdiness > 0) payload.sturdiness = sturdiness;
            if (power > 0) payload.power = power;
            if (pedals > 0) payload.pedals = pedals;

            await api.put(`/reviews/${review.review_id}`, payload);

            showToast(t('review_updated_success'), "success"); // Add translation later or use generic success

            // Go back
            navigation.goBack();
        } catch (e) {
            console.error(e);
            const errMsg = e.response?.data?.error || t('failed_update_review');
            showToast(errMsg, "error");
        } finally {
            setLoading(false);
        }
    };

    const styles = createStyles(theme);

    return (
        <KeyboardAvoidingView
            style={{ flex: 1, backgroundColor: theme.colors.background }}
            behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
            keyboardVerticalOffset={Platform.OS === 'ios' ? 100 : 0}
        >
            <ScrollView
                contentContainerStyle={styles.container}
                keyboardShouldPersistTaps="handled"
                automaticallyAdjustKeyboardInsets={true}
            >
                <Text style={styles.title}>{t('update_review_title')}</Text>

                <View style={styles.ratingsContainer}>
                    <Text style={styles.subtitle}>{t('ratings')}</Text>
                    <View style={styles.divider} />
                    <StarRating
                        label={t('breaks')}
                        value={breaks || 0}
                        onValueChange={setBreaks}
                        theme={theme}
                        styles={styles}
                        onInfoPress={() => handleInfoPress('breaks')}
                    />
                    <StarRating
                        label={t('seat')}
                        value={seat || 0}
                        onValueChange={setSeat}
                        theme={theme}
                        styles={styles}
                        onInfoPress={() => handleInfoPress('seat')}
                    />
                    <StarRating
                        label={t('sturdiness')}
                        value={sturdiness || 0}
                        onValueChange={setSturdiness}
                        theme={theme}
                        styles={styles}
                        onInfoPress={() => handleInfoPress('sturdiness')}
                    />
                    <StarRating
                        label={t('power')}
                        value={power || 0}
                        onValueChange={setPower}
                        theme={theme}
                        styles={styles}
                        onInfoPress={() => handleInfoPress('power')}
                    />
                    <StarRating
                        label={t('pedals')}
                        value={pedals || 0}
                        onValueChange={setPedals}
                        theme={theme}
                        styles={styles}
                        onInfoPress={() => handleInfoPress('pedals')}
                    />

                    <View style={styles.overallRow}>
                        <Text style={styles.overallLabel}>{t('overall_rating')}</Text>
                        <Text style={styles.overallValue}>{overall !== null ? overall.toFixed(1) : '-'} ⭐</Text>
                    </View>
                </View>

                <Text style={styles.label}>{t('comment')}</Text>
                <TextInput
                    placeholder={t('write_review_placeholder')}
                    placeholderTextColor={theme.colors.placeholder}
                    style={[styles.input, { height: 120 }]}
                    value={comment}
                    onChangeText={setComment}
                    multiline
                    textAlignVertical="top"
                />

                <Button title={t('update_review_button')} onPress={handleSubmit} disabled={loading} color={theme.colors.primary} />

                {/* Info Modal */}
                <Modal
                    animationType="fade"
                    transparent={true}
                    visible={modalVisible}
                    onRequestClose={() => setModalVisible(false)}
                >
                    <View style={styles.centeredView}>
                        <View style={styles.modalView}>
                            {activeCategory && (
                                <>
                                    <Text style={styles.modalTitle}>{t(activeCategory) + ': ' + t('info_title')}</Text>
                                    <Text style={styles.modalDescription}>{t(activeCategory + '_desc')}</Text>

                                    <View style={styles.exampleContainer}>
                                        <Text style={styles.exampleHeader}>5 ⭐</Text>
                                        <Text style={styles.exampleText}>{t(activeCategory + '_5star')}</Text>
                                    </View>

                                    <View style={styles.exampleContainer}>
                                        <Text style={styles.exampleHeader}>1 ⭐</Text>
                                        <Text style={styles.exampleText}>{t(activeCategory + '_1star')}</Text>
                                    </View>
                                </>
                            )}
                            <TouchableOpacity
                                style={[styles.button, styles.buttonClose]}
                                onPress={() => setModalVisible(false)}
                            >
                                <Text style={styles.textStyle}>{t('info_close')}</Text>
                            </TouchableOpacity>
                        </View>
                    </View>
                </Modal>
            </ScrollView>
        </KeyboardAvoidingView>
    );
};

const createStyles = (theme) => StyleSheet.create({
    container: { padding: 20, paddingBottom: 60, backgroundColor: theme.colors.background },
    title: { fontSize: 24, marginBottom: 20, textAlign: 'center', color: theme.colors.text },
    subtitle: { fontSize: 18, fontWeight: 'bold', marginBottom: 5, color: theme.colors.text },
    divider: { height: 1, backgroundColor: theme.colors.border, marginBottom: 15 },
    ratingsContainer: { marginBottom: 20, padding: 15, backgroundColor: theme.colors.card, borderRadius: 8 },
    ratingRow: { marginBottom: 15 },
    labelRow: { flexDirection: 'row', alignItems: 'center', marginBottom: 5 },
    ratingLabel: { fontSize: 16, fontWeight: '600', color: theme.colors.text, marginRight: 8 },
    infoButton: { padding: 2 },
    infoIcon: { fontSize: 16 },
    starsContainer: { flexDirection: 'row' },
    star: { fontSize: 40, paddingHorizontal: 2 },
    overallRow: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginTop: 15, borderTopWidth: 1, borderTopColor: theme.colors.border, paddingTop: 10 },
    overallLabel: { fontSize: 18, fontWeight: 'bold', color: theme.colors.text },
    overallValue: { fontSize: 28, fontWeight: 'bold', color: '#f39c12' },
    label: { fontSize: 16, marginBottom: 5, fontWeight: 'bold', color: theme.colors.text },
    input: {
        borderColor: theme.colors.border,
        borderWidth: 1,
        marginBottom: 20,
        paddingHorizontal: 12,
        paddingVertical: 12,
        borderRadius: 8,
        backgroundColor: theme.colors.inputBackground,
        fontSize: 16,
        color: theme.colors.text
    },
    // Modal Styles
    centeredView: {
        flex: 1,
        justifyContent: "center",
        alignItems: "center",
        backgroundColor: 'rgba(0,0,0,0.5)'
    },
    modalView: {
        width: '85%',
        margin: 20,
        backgroundColor: theme.colors.card,
        borderRadius: 15,
        padding: 25,
        alignItems: "flex-start",
        shadowColor: "#000",
        shadowOffset: {
            width: 0,
            height: 2
        },
        shadowOpacity: 0.25,
        shadowRadius: 4,
        elevation: 5
    },
    modalTitle: {
        fontSize: 20,
        fontWeight: 'bold',
        marginBottom: 10,
        color: theme.colors.text,
        alignSelf: 'center'
    },
    modalDescription: {
        fontSize: 16,
        marginBottom: 20,
        color: theme.colors.text,
        lineHeight: 22
    },
    exampleContainer: {
        width: '100%',
        marginBottom: 10,
        padding: 10,
        backgroundColor: theme.colors.background,
        borderRadius: 8,
        borderLeftWidth: 4,
        borderLeftColor: theme.colors.primary
    },
    exampleHeader: {
        fontSize: 14,
        fontWeight: 'bold',
        color: theme.colors.text,
        marginBottom: 4
    },
    exampleText: {
        fontSize: 14,
        color: theme.colors.text,
        fontStyle: 'italic'
    },
    button: {
        borderRadius: 10,
        padding: 12,
        elevation: 2,
        marginTop: 15,
        alignSelf: 'center',
        width: '100%'
    },
    buttonClose: {
        backgroundColor: theme.colors.primary,
    },
    textStyle: {
        color: "white",
        fontWeight: "bold",
        textAlign: "center",
        fontSize: 16
    }
});

export default UpdateReviewScreen;
