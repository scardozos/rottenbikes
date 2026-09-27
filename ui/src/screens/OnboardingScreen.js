import React, { useState, useContext, useEffect } from 'react';
import { View, Text, StyleSheet, ScrollView, TouchableOpacity, Animated, Easing } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { Image } from 'expo-image';
import Button from '../components/Button';
import Icon from '../components/Icon';
import { ThemeContext } from '../context/ThemeContext';
import { LanguageContext } from '../context/LanguageContext';

const TOTAL_STEPS = 4;

// Step 3: the five rating criteria, each with its Ionicon
const RATING_CRITERIA = [
    { labelKey: 'breaks', descKey: 'onboarding_breaks_desc', icon: 'stop-circle-outline' },
    { labelKey: 'seat', descKey: 'onboarding_seat_desc', icon: 'resize-outline' },
    { labelKey: 'sturdiness', descKey: 'onboarding_sturdiness_desc', icon: 'construct-outline' },
    { labelKey: 'power', descKey: 'onboarding_power_desc', icon: 'flash-outline' },
    { labelKey: 'pedals', descKey: 'onboarding_pedals_desc', icon: 'bicycle-outline' },
];

const OnboardingScreen = ({ onFinish }) => {
    const { theme } = useContext(ThemeContext);
    const { t } = useContext(LanguageContext);
    const [step, setStep] = useState(1);

    const styles = React.useMemo(() => createStyles(theme), [theme]);

    // Top progress bar fill (step / TOTAL_STEPS)
    const [progress] = useState(() => new Animated.Value(1 / TOTAL_STEPS));
    // Fade-in for each step's content
    const [stepFade] = useState(() => new Animated.Value(1));

    useEffect(() => {
        Animated.timing(progress, {
            toValue: step / TOTAL_STEPS,
            duration: 250,
            easing: Easing.out(Easing.ease),
            useNativeDriver: false, // width animations require the JS driver
        }).start();
    }, [step, progress]);

    useEffect(() => {
        if (step === 1) return; // no fade on the initial step
        stepFade.setValue(0);
        Animated.timing(stepFade, {
            toValue: 1,
            duration: 200,
            useNativeDriver: true,
        }).start();
    }, [step, stepFade]);

    const handleNext = () => {
        if (step < TOTAL_STEPS) {
            setStep(step + 1);
        } else if (onFinish) {
            onFinish();
        }
    };

    return (
        <SafeAreaView style={styles.safeArea} edges={['top', 'bottom', 'left', 'right']}>
            {/* Progress bar */}
            <View style={styles.progressTrack}>
                <Animated.View
                    style={[
                        styles.progressFill,
                        {
                            width: progress.interpolate({
                                inputRange: [0, 1],
                                outputRange: ['0%', '100%'],
                            }),
                        },
                    ]}
                />
            </View>

            {/* Skip (the last step uses "Get Started" instead) */}
            {step < TOTAL_STEPS && (
                <TouchableOpacity
                    onPress={onFinish}
                    style={styles.skipButton}
                    hitSlop={{ top: 12, bottom: 12, left: 12, right: 12 }}
                    accessibilityRole="button"
                    accessibilityLabel={t('onboarding_skip')}
                >
                    <Text style={styles.skipText}>{t('onboarding_skip')}</Text>
                </TouchableOpacity>
            )}

            {/* Step content */}
            <ScrollView
                style={styles.scrollView}
                contentContainerStyle={styles.scrollContent}
                keyboardShouldPersistTaps="handled"
            >
                <Animated.View style={[styles.stepContainer, { opacity: stepFade }]}>
                    {step === 1 && (
                        <>
                            {/* Bike QR SVG */}
                            <View style={styles.visualColumn}>
                                <Image 
                                    source={require('../../assets/bicing_bike_qr.svg')} 
                                    style={{ width: 240, height: 320 }} 
                                    contentFit="contain" 
                                />
                            </View>
                            <Text style={styles.title}>{t('onboarding_title_1')}</Text>
                            <Text style={styles.body}>{t('onboarding_body_1')}</Text>
                        </>
                    )}

                    {step === 2 && (
                        <>
                            {/* Bike ID SVG */}
                            <View style={styles.visualColumn}>
                                <Image 
                                    source={require('../../assets/bicing_bike_id.svg')} 
                                    style={{ width: 240, height: 320 }} 
                                    contentFit="contain" 
                                />
                            </View>
                            <Text style={styles.title}>{t('onboarding_title_id')}</Text>
                            <Text style={styles.body}>{t('onboarding_body_id')}</Text>
                            <View style={styles.proTipCard}>
                                <Icon name="bulb-outline" size={20} color={theme.colors.warning} style={styles.proTipIcon} />
                                <Text style={styles.proTipText}>{t('onboarding_pro_tip')}</Text>
                            </View>
                        </>
                    )}

                    {step === 3 && (
                        <>
                            {/* Lifecycle flow: scan -> add -> review */}
                            <View style={styles.visualColumn}>
                                <View style={styles.flowRow}>
                                    <View style={[styles.flowChip, { backgroundColor: theme.colors.secondary + '22' }]}>
                                        <Icon name="scan-outline" size={30} color={theme.colors.secondary} />
                                    </View>
                                    <Icon name="chevron-forward" size={18} color={theme.colors.subtext} />
                                    <View style={[styles.flowChip, { backgroundColor: theme.colors.primary + '22' }]}>
                                        <Icon name="add-circle-outline" size={30} color={theme.colors.primary} />
                                    </View>
                                    <Icon name="chevron-forward" size={18} color={theme.colors.subtext} />
                                    <View style={[styles.flowChip, { backgroundColor: theme.colors.warning + '22' }]}>
                                        <Icon name="star" size={30} color={theme.colors.warning} />
                                    </View>
                                </View>
                            </View>
                            <Text style={styles.title}>{t('onboarding_title_2')}</Text>
                            <Text style={styles.body}>{t('onboarding_body_2a')}</Text>
                            <Text style={styles.bodyLast}>{t('onboarding_body_2b')}</Text>
                        </>
                    )}

                    {step === 4 && (
                        <>
                            <Text style={styles.title}>{t('onboarding_title_3')}</Text>
                            <View style={styles.criteriaList}>
                                {RATING_CRITERIA.map((item) => (
                                    <View key={item.labelKey} style={styles.criteriaRow}>
                                        <View style={styles.criteriaChip}>
                                            <Icon name={item.icon} size={22} color={theme.colors.primary} />
                                        </View>
                                        <View style={styles.criteriaTexts}>
                                            <Text style={styles.criteriaLabel}>{t(item.labelKey)}</Text>
                                            <Text style={styles.criteriaDesc}>{t(item.descKey)}</Text>
                                        </View>
                                    </View>
                                ))}
                            </View>
                        </>
                    )}
                </Animated.View>
            </ScrollView>

            {/* Bottom navigation */}
            <View style={styles.footer}>
                <Button
                    title={step < TOTAL_STEPS ? t('onboarding_next') : t('onboarding_get_started')}
                    onPress={handleNext}
                    variant="primary"
                    size="lg"
                    style={styles.footerButton}
                />
            </View>
        </SafeAreaView>
    );
};

const createStyles = (theme) => StyleSheet.create({
    safeArea: {
        flex: 1,
        backgroundColor: theme.colors.background,
    },
    progressTrack: {
        height: 4,
        backgroundColor: theme.colors.border,
    },
    progressFill: {
        height: 4,
        backgroundColor: theme.colors.primary,
    },
    skipButton: {
        alignSelf: 'flex-end',
        marginTop: 8,
        marginRight: 8,
        paddingHorizontal: theme?.metrics?.spacing?.lg || 16,
        paddingVertical: theme?.metrics?.spacing?.sm || 8,
    },
    skipText: {
        color: theme.colors.subtext,
        fontSize: 15,
        fontWeight: '600',
    },
    scrollView: {
        flex: 1,
    },
    scrollContent: {
        flexGrow: 1,
    },
    stepContainer: {
        flex: 1,
        justifyContent: 'center',
        alignItems: 'center',
        alignSelf: 'center',
        width: '100%',
        maxWidth: 500,
        paddingHorizontal: theme?.metrics?.spacing?.xl || 24,
        paddingVertical: theme?.metrics?.spacing?.xl || 24,
    },
    visualColumn: {
        alignItems: 'center',
        gap: 12,
        marginBottom: 24,
    },
    bikeCircle: {
        width: 120,
        height: 120,
        borderRadius: 60,
        backgroundColor: theme.colors.primary + '18',
        alignItems: 'center',
        justifyContent: 'center',
    },
    idTag: {
        backgroundColor: theme.colors.card,
        borderRadius: theme?.metrics?.radii?.sm || 8,
        borderWidth: 1,
        borderColor: theme.colors.border,
        paddingHorizontal: 14,
        paddingVertical: 6,
    },
    idTagText: {
        color: theme.colors.text,
        fontSize: 18,
        fontWeight: '700',
        letterSpacing: 1,
    },
    flowRow: {
        flexDirection: 'row',
        alignItems: 'center',
        gap: 10,
    },
    flowChip: {
        width: 64,
        height: 64,
        borderRadius: 32,
        alignItems: 'center',
        justifyContent: 'center',
    },
    title: {
        fontSize: 24,
        fontWeight: 'bold',
        color: theme.colors.text,
        textAlign: 'center',
        marginBottom: 12,
    },
    body: {
        fontSize: 16,
        lineHeight: 24,
        color: theme.colors.subtext,
        textAlign: 'center',
        marginBottom: 12,
    },
    bodyLast: {
        fontSize: 16,
        lineHeight: 24,
        color: theme.colors.subtext,
        textAlign: 'center',
    },
    proTipCard: {
        flexDirection: 'row',
        alignItems: 'flex-start',
        gap: 10,
        backgroundColor: theme.colors.card,
        borderRadius: theme?.metrics?.radii?.lg || 16,
        borderWidth: 1,
        borderColor: theme.colors.border,
        padding: theme?.metrics?.spacing?.lg || 16,
        marginTop: 4,
        ...(theme?.shadows?.sm || {}),
    },
    proTipIcon: {
        marginTop: 2,
    },
    proTipText: {
        flex: 1,
        fontSize: 14,
        lineHeight: 21,
        color: theme.colors.subtext,
    },
    criteriaList: {
        alignSelf: 'center',
        width: '100%',
        maxWidth: 400,
        marginTop: 20,
        gap: 18,
    },
    criteriaRow: {
        flexDirection: 'row',
        alignItems: 'center',
        gap: 12,
    },
    criteriaChip: {
        width: 44,
        height: 44,
        borderRadius: 22,
        backgroundColor: theme.colors.primary + '18',
        alignItems: 'center',
        justifyContent: 'center',
    },
    criteriaTexts: {
        flex: 1,
    },
    criteriaLabel: {
        fontSize: 16,
        fontWeight: '600',
        color: theme.colors.text,
        marginBottom: 2,
    },
    criteriaDesc: {
        fontSize: 14,
        lineHeight: 20,
        color: theme.colors.subtext,
    },
    footer: {
        padding: theme?.metrics?.spacing?.lg || 16,
        borderTopWidth: 1,
        borderTopColor: theme.colors.border,
        alignItems: 'center',
    },
    footerButton: {
        width: '100%',
        maxWidth: 500,
    },
});

export default OnboardingScreen;
