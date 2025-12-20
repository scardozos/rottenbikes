import React, { useState, useEffect } from 'react';
import { View, Text, TextInput, Button, StyleSheet, Switch, Alert, ScrollView, TouchableOpacity, KeyboardAvoidingView, Platform } from 'react-native';
import api from '../services/api';
import { useToast } from '../context/ToastContext';

const StarRating = ({ label, value, onValueChange }) => (
    <View style={styles.ratingRow}>
        <Text style={styles.ratingLabel}>{label}</Text>
        <View style={styles.starsContainer}>
            {[1, 2, 3, 4, 5].map((star) => (
                <TouchableOpacity key={star} onPress={() => onValueChange(star)}>
                    <Text style={[styles.star, { color: star <= value ? '#f39c12' : '#ccc' }]}>
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

    // Subcategories
    const [breaks, setBreaks] = useState(3);
    const [seat, setSeat] = useState(3);
    const [sturdiness, setSturdiness] = useState(3);
    const [power, setPower] = useState(3);
    const [pedals, setPedals] = useState(3);

    const [overall, setOverall] = useState(3.0);
    const [comment, setComment] = useState('');
    const [captchaVerified, setCaptchaVerified] = useState(false);
    const [loading, setLoading] = useState(false);

    // Auto-calculate overall
    useEffect(() => {
        const sum = breaks + seat + sturdiness + power + pedals;
        const avg = sum / 5;
        setOverall(avg);
    }, [breaks, seat, sturdiness, power, pedals]);

    const handleSubmit = async () => {
        if (!captchaVerified) {
            showToast("Please verify you are not a robot.", "error");
            return;
        }

        setLoading(true);
        try {
            await api.post(`/bikes/${bike.numerical_id}/reviews`, {
                comment: comment,
                // Send subcategories as integers
                breaks: breaks,
                seat: seat,
                sturdiness: sturdiness,
                power: power,
                pedals: pedals,
                // Send rounded overall to backend (as it expects int16)
                overall: Math.round(overall)
            });

            showToast("Review submitted!", "success");
            navigation.navigate('Home');
        } catch (e) {
            console.error(e);
            showToast("Failed to submit review. " + (e.response?.status === 401 ? "Unauthorized (login required)" : ""), "error");
        } finally {
            setLoading(false);
        }
    };

    return (
        <KeyboardAvoidingView
            style={{ flex: 1 }}
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
                    <StarRating label="Breaks" value={breaks} onValueChange={setBreaks} />
                    <StarRating label="Seat" value={seat} onValueChange={setSeat} />
                    <StarRating label="Sturdiness" value={sturdiness} onValueChange={setSturdiness} />
                    <StarRating label="Power" value={power} onValueChange={setPower} />
                    <StarRating label="Pedals" value={pedals} onValueChange={setPedals} />

                    <View style={styles.overallRow}>
                        <Text style={styles.overallLabel}>Overall Rating:</Text>
                        {/* Display with 1 decimal place */}
                        <Text style={styles.overallValue}>{overall.toFixed(1)} ⭐</Text>
                    </View>
                </View>

                <Text style={styles.label}>Comment</Text>
                <TextInput
                    placeholder="Write your review..."
                    style={[styles.input, { height: 120 }]}
                    value={comment}
                    onChangeText={setComment}
                    multiline
                    textAlignVertical="top"
                />

                {/* Image Upload Placeholder */}
                <View style={styles.placeholderContainer}>
                    <Text>Image Upload Placeholder</Text>
                    <Button title="Select Image (Mock)" onPress={() => { }} disabled />
                </View>

                {/* Captcha Placeholder */}
                <View style={styles.captchaContainer}>
                    <Switch value={captchaVerified} onValueChange={setCaptchaVerified} />
                    <Text style={styles.captchaText}>I am not a robot (Captcha Placeholder)</Text>
                </View>

                <Button title="Submit Review" onPress={handleSubmit} disabled={loading} />
            </ScrollView>
        </KeyboardAvoidingView>
    );
};

const styles = StyleSheet.create({
    container: { padding: 20, paddingBottom: 60 },
    title: { fontSize: 24, marginBottom: 20, textAlign: 'center' },
    subtitle: { fontSize: 18, fontWeight: 'bold', marginBottom: 5 },
    divider: { height: 1, backgroundColor: '#ddd', marginBottom: 15 },
    ratingsContainer: { marginBottom: 20, padding: 15, backgroundColor: '#f9f9f9', borderRadius: 8 },
    ratingRow: { marginBottom: 15 },
    ratingLabel: { fontSize: 16, fontWeight: '600', marginBottom: 5 },
    starsContainer: { flexDirection: 'row' },
    star: { fontSize: 40, paddingHorizontal: 2 },
    ratingInput: { borderWidth: 1, borderColor: '#ccc', width: 50, textAlign: 'center', padding: 5, borderRadius: 4, backgroundColor: '#fff' },
    overallRow: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginTop: 15, borderTopWidth: 1, borderTopColor: '#ddd', paddingTop: 10 },
    overallLabel: { fontSize: 18, fontWeight: 'bold' },
    overallValue: { fontSize: 28, fontWeight: 'bold', color: '#f39c12' },
    label: { fontSize: 16, marginBottom: 5, fontWeight: 'bold' },
    input: {
        borderColor: '#ccc',
        borderWidth: 1,
        marginBottom: 20,
        paddingHorizontal: 12,
        paddingVertical: 12,
        borderRadius: 8,
        backgroundColor: '#fff',
        fontSize: 16,
    },
    placeholderContainer: {
        borderWidth: 1,
        borderColor: '#ccc',
        borderStyle: 'dashed',
        padding: 20,
        alignItems: 'center',
        marginBottom: 20,
        borderRadius: 8
    },
    captchaContainer: {
        flexDirection: 'row',
        alignItems: 'center',
        marginBottom: 20
    },
    captchaText: {
        marginLeft: 10
    }
});

export default CreateReviewScreen;

