import React, { useState, useEffect, useContext } from 'react';
import api from '../services/api';
import { useToast } from '../context/ToastContext';
import { ThemeContext } from '../context/ThemeContext';
import { LanguageContext } from '../context/LanguageContext';
import { SessionContext } from '../context/SessionContext';
import { ReviewForm } from '../components/ReviewForm';

const CreateReviewScreen = ({ route, navigation }) => {
    const { bikeId } = route.params;
    const { showToast } = useToast();
    const { theme } = useContext(ThemeContext);
    const { t } = useContext(LanguageContext);
    const { validatedBikeId } = useContext(SessionContext);

    // Security Check: Ensure the user has actually scanned this bike
    useEffect(() => {
        if (!validatedBikeId || String(validatedBikeId) !== String(bikeId)) {
            showToast(t('unauthorized'), 'error');
            navigation.reset({
                index: 0,
                routes: [{ name: 'Main', state: { routes: [{ name: 'Home' }] } }],
            });
        }
    }, [bikeId, validatedBikeId, navigation, showToast, t]);

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

            await api.post(`/bikes/${bikeId}/reviews`, payload);

            showToast(t('review_submitted'), "success");

            // Reset navigation stack and go to BikesList (Browse) tab
            navigation.reset({
                index: 0,
                routes: [{
                    name: 'Main',
                    state: {
                        routes: [{ name: 'BikesList' }]
                    }
                }],
            });
        } catch (e) {
            console.error(e);
            const errMsg = e.response?.data?.error || t('failed_submit_review');
            showToast(errMsg, "error");
        } finally {
            setLoading(false);
        }
    };

    return (
        <ReviewForm
            title={t('review_bike_title', { numerical_id: bikeId })}
            breaks={breaks} setBreaks={setBreaks}
            seat={seat} setSeat={setSeat}
            sturdiness={sturdiness} setSturdiness={setSturdiness}
            power={power} setPower={setPower}
            pedals={pedals} setPedals={setPedals}
            overall={overall}
            comment={comment} setComment={setComment}
            onSubmit={handleSubmit}
            loading={loading}
            submitButtonText={t('submit_review')}
            t={t} theme={theme}
        />
    );
};

export default CreateReviewScreen;
