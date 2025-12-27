import React, { useState, useEffect, useContext } from 'react';
import { View, Text, TextInput, Button, StyleSheet, Switch, Alert, ScrollView, TouchableOpacity, KeyboardAvoidingView, Platform } from 'react-native';
import api from '../services/api';
import { useToast } from '../context/ToastContext';
import { ThemeContext } from '../context/ThemeContext';

const StarRating = ({ label, value, onValueChange, theme, styles }) => (
    <View style={styles.ratingRow}>
        <Text style={styles.ratingLabel}>{label}</Text>
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

const CreateReviewScreen = ({ route, navigation }) => {
    const { bike } = route.params;
    const { showToast } = useToast();
    const { theme } = useContext(ThemeContext);

    // Subcategories
    const [breaks, setBreaks] = useState(null);
    const [seat, setSeat] = useState(null);
    const [sturdiness, setSturdiness] = useState(null);
    const [power, setPower] = useState(null);
    const [pedals, setPedals] = useState(null);

    const [overall, setOverall] = useState(null);
    const [comment, setComment] = useState('');
    const [loading, setLoading] = useState(false);

    // Auto-calculate overall
    useEffect(() => {
        const ratings = [breaks, seat, sturdiness, power, pedals].filter(r => r > 0);
        if (ratings.length === 0) {
            setOverall(null);
            return;
        }
        const sum = ratings.reduce((a, b) => a + b, 0);
        const avg = sum / ratings.length;
        setOverall(avg);
    }, [breaks, seat, sturdiness, power, pedals]);

    const handleSubmit = async () => {
        // Validate that at least one rating is provided
        const ratings = [breaks, seat, sturdiness, power, pedals];
        const hasRating = ratings.some(r => r > 0);

        if (!hasRating) {
            showToast("Please rate at least one category", "error");
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

            await api.post(`/bikes/${bike.numerical_id}/reviews`, payload);

            showToast("Review submitted!", "success");
            navigation.navigate('Home');
        } catch (e) {
            console.error(e);
            const errMsg = e.response?.data?.error || "Failed to submit review.";
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
                <Text style={styles.title}>Review Bike #{bike.numerical_id}</Text>

                <View style={styles.ratingsContainer}>
                    <Text style={styles.subtitle}>Ratings</Text>
                    <View style={styles.divider} />
                    <StarRating label="Breaks" value={breaks || 0} onValueChange={setBreaks} theme={theme} styles={styles} />
                    <StarRating label="Seat" value={seat || 0} onValueChange={setSeat} theme={theme} styles={styles} />
                    <StarRating label="Sturdiness" value={sturdiness || 0} onValueChange={setSturdiness} theme={theme} styles={styles} />
                    <StarRating label="Power" value={power || 0} onValueChange={setPower} theme={theme} styles={styles} />
                    <StarRating label="Pedals" value={pedals || 0} onValueChange={setPedals} theme={theme} styles={styles} />

                    <View style={styles.overallRow}>
                        <Text style={styles.overallLabel}>Overall Rating:</Text>
                        {/* Display with 1 decimal place */}
                        <Text style={styles.overallValue}>{overall !== null ? overall.toFixed(1) : '-'} ⭐</Text>
                    </View>
                </View>

                <Text style={styles.label}>Comment</Text>
                <TextInput
                    placeholder="Write your review..."
                    placeholderTextColor={theme.colors.placeholder}
                    style={[styles.input, { height: 120 }]}
                    value={comment}
                    onChangeText={setComment}
                    multiline
                    textAlignVertical="top"
                />

                {/* Image Upload Placeholder */}
                <View style={styles.placeholderContainer}>
                    <Text style={{ color: theme.colors.subtext }}>Image Upload Placeholder</Text>
                    <Button title="Select Image (Mock)" onPress={() => { }} disabled />
                </View>


                <Button title="Submit Review" onPress={handleSubmit} disabled={loading} color={theme.colors.primary} />
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
    ratingLabel: { fontSize: 16, fontWeight: '600', marginBottom: 5, color: theme.colors.text },
    starsContainer: { flexDirection: 'row' },
    star: { fontSize: 40, paddingHorizontal: 2 },
    ratingInput: { borderWidth: 1, borderColor: theme.colors.border, width: 50, textAlign: 'center', padding: 5, borderRadius: 4, backgroundColor: theme.colors.inputBackground },
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
    placeholderContainer: {
        borderWidth: 1,
        borderColor: theme.colors.border,
        borderStyle: 'dashed',
        padding: 20,
        alignItems: 'center',
        marginBottom: 20,
        borderRadius: 8
    },
});

export default CreateReviewScreen;

