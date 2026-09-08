import React, { useState } from 'react';
import { View, Text, TextInput, StyleSheet, TouchableOpacity, ScrollView, Modal, KeyboardAvoidingView, Platform } from 'react-native';
import * as Haptics from 'expo-haptics';
import Button from './Button';
import Icon from './Icon';

export const StarRating = ({ label, value, onValueChange, theme, styles, onInfoPress }) => {
    const handleStarPress = (star) => {
        if (Platform.OS !== 'web') {
            try {
                Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
            } catch {
                // Fallback silently if haptics unavailable
            }
        }
        onValueChange(star);
    };

    return (
        <View style={styles.ratingRow}>
            <View style={styles.labelRow}>
                <Text style={styles.ratingLabel}>{label}</Text>
                <TouchableOpacity
                    onPress={onInfoPress}
                    style={styles.infoButton}
                    hitSlop={{ top: 12, bottom: 12, left: 12, right: 12 }}
                    accessibilityRole="button"
                    accessibilityLabel={`${label} info`}
                >
                    <Icon name="information-circle-outline" size={18} color={theme.colors.primary} />
                </TouchableOpacity>
            </View>
            <View style={styles.starsContainer}>
                {[1, 2, 3, 4, 5].map((star) => (
                    <TouchableOpacity
                        key={star}
                        onPress={() => handleStarPress(star)}
                        style={styles.starTouchTarget}
                        hitSlop={{ top: 8, bottom: 8, left: 6, right: 6 }}
                        accessibilityRole="button"
                        accessibilityLabel={`${star} star`}
                    >
                        <Icon
                            name={star <= value ? 'star' : 'star-outline'}
                            size={34}
                            color={star <= value ? theme.colors.warning : theme.colors.border}
                        />
                    </TouchableOpacity>
                ))}
            </View>
        </View>
    );
};

export const ReviewForm = ({
    title,
    breaks, setBreaks,
    seat, setSeat,
    sturdiness, setSturdiness,
    power, setPower,
    pedals, setPedals,
    overall,
    comment, setComment,
    onSubmit,
    onDelete,
    loading,
    submitButtonText,
    t, theme
}) => {
    const [modalVisible, setModalVisible] = useState(false);
    const [activeCategory, setActiveCategory] = useState(null);

    const handleInfoPress = (category) => {
        setActiveCategory(category);
        setModalVisible(true);
    };

    const styles = React.useMemo(() => createStyles(theme), [theme]);

    return (
        <KeyboardAvoidingView
            style={styles.keyboardAvoid}
            behavior={Platform.OS === 'ios' ? 'padding' : undefined}
            keyboardVerticalOffset={Platform.OS === 'ios' ? 100 : 0}
        >
            <ScrollView
                contentContainerStyle={styles.container}
                keyboardShouldPersistTaps="handled"
                automaticallyAdjustKeyboardInsets={true}
            >
                {title && <Text style={styles.title}>{title}</Text>}

                <View style={styles.ratingsCard}>
                    <Text style={styles.cardHeader}>{t('ratings')}</Text>
                    <View style={styles.divider} />
                    <StarRating label={t('breaks')} value={breaks || 0} onValueChange={setBreaks} theme={theme} styles={styles} onInfoPress={() => handleInfoPress('breaks')} />
                    <StarRating label={t('seat')} value={seat || 0} onValueChange={setSeat} theme={theme} styles={styles} onInfoPress={() => handleInfoPress('seat')} />
                    <StarRating label={t('sturdiness')} value={sturdiness || 0} onValueChange={setSturdiness} theme={theme} styles={styles} onInfoPress={() => handleInfoPress('sturdiness')} />
                    <StarRating label={t('power')} value={power || 0} onValueChange={setPower} theme={theme} styles={styles} onInfoPress={() => handleInfoPress('power')} />
                    <StarRating label={t('pedals')} value={pedals || 0} onValueChange={setPedals} theme={theme} styles={styles} onInfoPress={() => handleInfoPress('pedals')} />

                    <View style={styles.overallRow}>
                        <Text style={styles.overallLabel}>{t('overall_rating')}</Text>
                        <View style={styles.overallValueGroup}>
                            <Text style={styles.overallValue}>{overall !== null ? overall.toFixed(1) : '-'} </Text>
                            <Icon name="star" size={26} color={theme.colors.warning} />
                        </View>
                    </View>
                </View>

                <View style={styles.commentSection}>
                    <Text style={styles.inputLabel}>{t('comment')}</Text>
                    <TextInput
                        placeholder={t('write_review_placeholder')}
                        placeholderTextColor={theme.colors.placeholder}
                        style={styles.textArea}
                        value={comment}
                        onChangeText={setComment}
                        multiline
                        textAlignVertical="top"
                    />
                </View>

                <View style={styles.actionsContainer}>
                    <Button
                        title={submitButtonText}
                        onPress={onSubmit}
                        disabled={loading}
                        loading={loading}
                        variant="primary"
                        size="lg"
                        style={styles.fullWidth}
                    />
                    {onDelete && (
                        <Button
                            title={t('delete')}
                            onPress={onDelete}
                            disabled={loading}
                            variant="danger"
                            size="md"
                            style={[styles.fullWidth, styles.deleteBtn]}
                        />
                    )}
                </View>

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
                                        <View style={styles.exampleHeaderRow}>
                                            <Text style={styles.exampleHeader}>5 </Text>
                                            <Icon name="star" size={14} color={theme.colors.warning} />
                                        </View>
                                        <Text style={styles.exampleText}>{t(activeCategory + '_5star')}</Text>
                                    </View>

                                    <View style={styles.exampleContainer}>
                                        <View style={styles.exampleHeaderRow}>
                                            <Text style={styles.exampleHeader}>1 </Text>
                                            <Icon name="star" size={14} color={theme.colors.warning} />
                                        </View>
                                        <Text style={styles.exampleText}>{t(activeCategory + '_1star')}</Text>
                                    </View>
                                </>
                            )}
                            <Button
                                title={t('info_close')}
                                onPress={() => setModalVisible(false)}
                                variant="primary"
                                size="md"
                                style={styles.modalCloseBtn}
                            />
                        </View>
                    </View>
                </Modal>
            </ScrollView>
        </KeyboardAvoidingView>
    );
};

export const createStyles = (theme) => StyleSheet.create({
    keyboardAvoid: {
        flex: 1,
        backgroundColor: theme.colors.background,
    },
    container: {
        padding: theme?.metrics?.spacing?.lg || 16,
        paddingBottom: theme?.metrics?.spacing?.xxl || 40,
    },
    title: {
        fontSize: theme?.typography?.h2?.fontSize || 24,
        fontWeight: 'bold',
        marginBottom: theme?.metrics?.spacing?.lg || 16,
        textAlign: 'center',
        color: theme.colors.text,
    },
    cardHeader: {
        fontSize: 18,
        fontWeight: 'bold',
        marginBottom: 8,
        color: theme.colors.text,
    },
    divider: {
        height: 1,
        backgroundColor: theme.colors.border,
        marginBottom: 16,
    },
    ratingsCard: {
        marginBottom: theme?.metrics?.spacing?.lg || 16,
        padding: theme?.metrics?.spacing?.lg || 16,
        backgroundColor: theme.colors.card,
        borderRadius: theme?.metrics?.radii?.lg || 16,
        borderWidth: 1,
        borderColor: theme.colors.border,
        ...(theme?.shadows?.sm || {}),
    },
    ratingRow: {
        marginBottom: 16,
    },
    labelRow: {
        flexDirection: 'row',
        alignItems: 'center',
        marginBottom: 8,
    },
    ratingLabel: {
        fontSize: 15,
        fontWeight: '600',
        color: theme.colors.text,
        marginRight: 6,
    },
    infoButton: {
        padding: 4,
    },
    starsContainer: {
        flexDirection: 'row',
        alignItems: 'center',
        gap: 6,
    },
    starTouchTarget: {
        minWidth: 44,
        minHeight: 44,
        justifyContent: 'center',
        alignItems: 'center',
    },
    overallRow: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginTop: 12,
        borderTopWidth: 1,
        borderTopColor: theme.colors.border,
        paddingTop: 14,
    },
    overallLabel: {
        fontSize: 17,
        fontWeight: 'bold',
        color: theme.colors.text,
    },
    overallValueGroup: {
        flexDirection: 'row',
        alignItems: 'center',
    },
    overallValue: {
        fontSize: 26,
        fontWeight: 'bold',
        color: theme.colors.warning,
    },
    commentSection: {
        marginBottom: theme?.metrics?.spacing?.lg || 16,
    },
    inputLabel: {
        fontSize: 15,
        marginBottom: 8,
        fontWeight: '600',
        color: theme.colors.text,
    },
    textArea: {
        borderColor: theme.colors.border,
        borderWidth: 1,
        borderRadius: theme?.metrics?.radii?.md || 12,
        backgroundColor: theme.colors.inputBackground,
        paddingHorizontal: 14,
        paddingVertical: 12,
        height: 120,
        fontSize: 15,
        color: theme.colors.text,
    },
    actionsContainer: {
        marginTop: 8,
        gap: 12,
    },
    fullWidth: {
        width: '100%',
    },
    deleteBtn: {
        marginTop: 4,
    },
    centeredView: {
        flex: 1,
        justifyContent: "center",
        alignItems: "center",
        backgroundColor: 'rgba(0,0,0,0.5)',
        padding: 20,
    },
    modalView: {
        width: '100%',
        maxWidth: 400,
        backgroundColor: theme.colors.card,
        borderRadius: theme?.metrics?.radii?.lg || 16,
        padding: theme?.metrics?.spacing?.xl || 24,
        borderWidth: 1,
        borderColor: theme.colors.border,
        ...(theme?.shadows?.lg || {}),
    },
    modalTitle: {
        fontSize: 18,
        fontWeight: 'bold',
        marginBottom: 10,
        color: theme.colors.text,
        textAlign: 'center',
    },
    modalDescription: {
        fontSize: 15,
        marginBottom: 16,
        color: theme.colors.text,
        lineHeight: 22,
    },
    exampleContainer: {
        width: '100%',
        marginBottom: 10,
        padding: 12,
        backgroundColor: theme.colors.background,
        borderRadius: theme?.metrics?.radii?.sm || 8,
        borderLeftWidth: 4,
        borderLeftColor: theme.colors.primary,
    },
    exampleHeaderRow: {
        flexDirection: 'row',
        alignItems: 'center',
        marginBottom: 4,
    },
    exampleHeader: {
        fontSize: 13,
        fontWeight: 'bold',
        color: theme.colors.text,
    },
    exampleText: {
        fontSize: 13,
        color: theme.colors.subtext,
        fontStyle: 'italic',
    },
    modalCloseBtn: {
        width: '100%',
        marginTop: 14,
    },
});

export default ReviewForm;
