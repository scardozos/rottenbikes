import React, { useState, useEffect, useContext } from 'react';
import { Alert } from 'react-native';
import api from '../services/api';
import { useToast } from '../context/ToastContext';
import { ThemeContext } from '../context/ThemeContext';
import { LanguageContext } from '../context/LanguageContext';
import { ReviewForm } from '../components/ReviewForm';

const UpdateReviewScreen = ({ route, navigation }) => {
    const { reviewId } = route.params;
    const { showToast } = useToast();
    const { theme } = useContext(ThemeContext);
    const { t } = useContext(LanguageContext);

    // Subcategories
    const [breaks, setBreaks] = useState(0);
    const [seat, setSeat] = useState(0);
    const [sturdiness, setSturdiness] = useState(0);
    const [power, setPower] = useState(0);
    const [pedals, setPedals] = useState(0);

    const [overall, setOverall] = useState(null);
    const [comment, setComment] = useState('');
    const [loading, setLoading] = useState(true);
    const [review, setReview] = useState(null);

    useEffect(() => {
        const fetchReview = async () => {
            try {
                const res = await api.get(`/reviews/${reviewId}`);
                const data = res.data;
                setReview(data);
                setBreaks(data.ratings?.breaks || 0);
                setSeat(data.ratings?.seat || 0);
                setSturdiness(data.ratings?.sturdiness || 0);
                setPower(data.ratings?.power || 0);
                setPedals(data.ratings?.pedals || 0);
                setOverall(data.ratings?.overall || null);
                setComment(data.comment || '');
            } catch (e) {
                console.error(e);
                showToast(t('error_fetching_bike'), "error"); // Reuse error msg or add new one
                navigation.goBack();
            } finally {
                setLoading(false);
            }
        };
        fetchReview();
    }, [reviewId]);

    // Auto-calculate overall
    useEffect(() => {
        const ratings = [breaks, seat, sturdiness, power, pedals].filter(r => r > 0);
        if (ratings.length === 0) {
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

            await api.put(`/reviews/${reviewId}`, payload);

            showToast(t('review_updated_success'), "success");

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

    const handleDelete = () => {
        Alert.alert(
            t('delete_review_title') || 'Delete Review',
            t('delete_review_confirm') || 'Are you sure you want to delete this review?',
            [
                { text: t('cancel') || 'Cancel', style: 'cancel' },
                {
                    text: t('delete') || 'Delete',
                    style: 'destructive',
                    onPress: async () => {
                        setLoading(true);
                        try {
                            await api.delete(`/reviews/${reviewId}`);
                            showToast(t('review_deleted_success') || 'Review deleted successfully', "success");
                            navigation.goBack();
                        } catch (e) {
                            console.error(e);
                            const errMsg = e.response?.data?.error || t('failed_delete_review') || 'Failed to delete review';
                            showToast(errMsg, "error");
                        } finally {
                            setLoading(false);
                        }
                    }
                }
            ]
        );
    };

    return (
        <ReviewForm
            title={t('update_review_title')}
            breaks={breaks} setBreaks={setBreaks}
            seat={seat} setSeat={setSeat}
            sturdiness={sturdiness} setSturdiness={setSturdiness}
            power={power} setPower={setPower}
            pedals={pedals} setPedals={setPedals}
            overall={overall}
            comment={comment} setComment={setComment}
            onSubmit={handleSubmit}
            onDelete={handleDelete}
            loading={loading}
            submitButtonText={t('update_review_button')}
            t={t} theme={theme}
        />
    );
};

export default UpdateReviewScreen;
