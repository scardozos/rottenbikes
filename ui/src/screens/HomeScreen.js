import React, { useState, useRef, useContext, useEffect } from 'react';
import { Text, View, StyleSheet, Platform, Alert, KeyboardAvoidingView, Pressable, Keyboard } from 'react-native';
import api from '../services/api';
import { useToast } from '../context/ToastContext';
import { ThemeContext } from '../context/ThemeContext';
import { useSession } from '../context/SessionContext';
import { LanguageContext } from '../context/LanguageContext';
import { Scanner } from '../components/Scanner';
import { isNumeric } from '../utils/validation';
import Button from '../components/Button';
import Input from '../components/Input';
import Icon from '../components/Icon';
import ErrorBoundary from '../components/ErrorBoundary';

const HomeScreen = ({ navigation }) => {
    const { theme } = useContext(ThemeContext);
    const { validateBike } = useSession();
    const [manualId, setManualId] = useState('');
    const [isInputActive, setIsInputActive] = useState(false);
    const { showToast } = useToast();
    const { t } = useContext(LanguageContext);
    const isScanning = useRef(false);

    // Monitor keyboard visibility on Native
    useEffect(() => {
        if (Platform.OS !== 'web') {
            const showSubscription = Keyboard.addListener('keyboardDidShow', () => setIsInputActive(true));
            const hideSubscription = Keyboard.addListener('keyboardDidHide', () => setIsInputActive(false));
            return () => {
                showSubscription.remove();
                hideSubscription.remove();
            };
        }
    }, []);

    const handleManualSubmit = async () => {
        if (!manualId.trim()) return;

        if (!isNumeric(manualId.trim())) {
            showToast(t('invalid_numerical_id'), "error");
            return;
        }

        const bikeId = manualId.trim();

        try {
            await api.get(`/bikes/${bikeId}/details`);
            validateBike(bikeId);
            navigation.navigate('BikesList', { screen: 'BikeDetails', params: { bikeId } });
            setManualId('');
        } catch (e) {
            if (e.response && e.response.status === 404) {
                if (Platform.OS === 'web') {
                    const create = window.confirm(`Bike #${bikeId} not found. Would you like to create it?`);
                    if (create) {
                        navigation.navigate('BikesList', { screen: 'CreateBike', params: { initialNumericalId: bikeId } });
                    }
                } else {
                    Alert.alert(
                        "Bike Not Found",
                        `Bike #${bikeId} not found. Would you like to create it?`,
                        [
                            { text: "Cancel", style: "cancel" },
                            {
                                text: "Create",
                                onPress: () => navigation.navigate('BikesList', { screen: 'CreateBike', params: { initialNumericalId: bikeId } })
                            }
                        ]
                    );
                }
            } else {
                const errMsg = e.response?.data?.error || t('scan_lookup_failed');
                showToast(errMsg, "error");
            }
        }
    };

    const handleScanSuccess = async (data) => {
        if (isScanning.current) return;
        isScanning.current = true;

        try {
            const response = await api.get('/bikes');
            const bikes = response.data || [];
            const bike = bikes.find(b => b.hash_id === data);

            if (bike) {
                showToast(t('found_bike', { id: bike.numerical_id }), "success");
                isScanning.current = false;
                validateBike(bike.numerical_id);
                navigation.navigate('BikesList', { screen: 'BikeDetails', params: { bikeId: bike.numerical_id } });
            } else {
                if (Platform.OS === 'web') {
                    const create = window.confirm(`No bike found with Hash ID: ${data}. Create it?`);
                    if (create) {
                        navigation.navigate('BikesList', { screen: 'CreateBike', params: { initialHashId: data } });
                    } else {
                        isScanning.current = false;
                    }
                } else {
                    Alert.alert(
                        "Not Found",
                        `No bike found with Hash ID: ${data}. Would you like to create it?`,
                        [
                            { text: "Cancel", onPress: () => { isScanning.current = false; }, style: "cancel" },
                            { text: "Create", onPress: () => { navigation.navigate('BikesList', { screen: 'CreateBike', params: { initialHashId: data } }); } }
                        ]
                    );
                }
            }
        } catch (e) {
            const errMsg = e.response?.data?.error || t('scan_lookup_failed');
            showToast(errMsg, "error");
            isScanning.current = false;
        }
    };

    const stylesInternal = React.useMemo(() => createStyles(theme), [theme]);

    const isMobileWeb = Platform.OS === 'web' && /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent);
    const showCamera = (Platform.OS !== 'web' || isMobileWeb);
    const shouldRenderCamera = showCamera && !isInputActive;

    const content = (
        <KeyboardAvoidingView
            style={stylesInternal.container}
            behavior={Platform.OS === 'ios' ? 'padding' : undefined}
            keyboardVerticalOffset={Platform.OS === 'ios' ? 100 : 0}
        >
            {/* Camera Area - Top Portion */}
            {showCamera ? (
                <View style={stylesInternal.cameraContainer}>
                    {shouldRenderCamera ? (
                        <ErrorBoundary>
                            <Scanner onScan={handleScanSuccess} theme={theme} t={t} />
                        </ErrorBoundary>
                    ) : (
                        <View style={stylesInternal.scannerPaused}>
                            <Icon name="camera-reverse-outline" size={40} color={theme.colors.subtext} />
                            <Text style={stylesInternal.scannerPausedText}>{t('scanner_paused') || "Scanner Paused"}</Text>
                        </View>
                    )}
                </View>
            ) : (
                <View style={stylesInternal.desktopPlaceholder}>
                    <View style={stylesInternal.desktopIconCircle}>
                        <Icon name="qr-code-outline" size={56} color={theme.colors.primary} />
                    </View>
                    <Text style={stylesInternal.desktopTitle}>{t('scan_qr') || "Scan Bike QR Code"}</Text>
                    <Text style={stylesInternal.desktopSubtitle}>
                        Use a mobile device camera to scan, or look up by bike number below.
                    </Text>
                </View>
            )}

            {/* Manual Input Section - Styled Clean Card */}
            <View style={stylesInternal.cardWrapper}>
                <View style={stylesInternal.card}>
                    <View style={stylesInternal.cardHeader}>
                        <Icon name="keypad-outline" size={20} color={theme.colors.primary} />
                        <Text style={stylesInternal.cardTitle}>{t('enter_manual_id')}</Text>
                    </View>

                    <View style={stylesInternal.inputRow}>
                        <View style={stylesInternal.inputFlex}>
                            <Input
                                placeholder={t('bike_id_placeholder')}
                                keyboardType="numeric"
                                value={manualId}
                                onChangeText={setManualId}
                                returnKeyType="done"
                                onSubmitEditing={handleManualSubmit}
                                onFocus={() => Platform.OS === 'web' && setIsInputActive(true)}
                                onBlur={() => Platform.OS === 'web' && setIsInputActive(false)}
                                containerStyle={{ marginBottom: 0 }}
                                leftIcon={<Icon name="bicycle-outline" size={18} color={theme.colors.subtext} />}
                            />
                        </View>
                        <Button
                            title={t('go')}
                            onPress={handleManualSubmit}
                            variant="primary"
                            size="md"
                            style={stylesInternal.goButton}
                        />
                    </View>
                </View>
            </View>
        </KeyboardAvoidingView>
    );

    if (Platform.OS === 'web') {
        return content;
    }

    return (
        <Pressable onPress={Keyboard.dismiss} style={{ flex: 1 }}>
            {content}
        </Pressable>
    );
};

const createStyles = (theme) => StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: theme.colors.background,
    },
    cameraContainer: {
        flex: 1,
        backgroundColor: '#000000',
        overflow: 'hidden',
    },
    scannerPaused: {
        flex: 1,
        backgroundColor: '#000000',
        justifyContent: 'center',
        alignItems: 'center',
        gap: 10,
    },
    scannerPausedText: {
        color: theme.colors.subtext,
        fontSize: 15,
    },
    desktopPlaceholder: {
        flex: 1,
        justifyContent: 'center',
        alignItems: 'center',
        padding: 30,
        backgroundColor: theme.colors.background,
    },
    desktopIconCircle: {
        width: 100,
        height: 100,
        borderRadius: 50,
        backgroundColor: theme.colors.primary + '18',
        alignItems: 'center',
        justifyContent: 'center',
        marginBottom: 20,
    },
    desktopTitle: {
        fontSize: 22,
        fontWeight: 'bold',
        color: theme.colors.text,
        marginBottom: 8,
        textAlign: 'center',
    },
    desktopSubtitle: {
        fontSize: 15,
        color: theme.colors.subtext,
        textAlign: 'center',
        maxWidth: 380,
        lineHeight: 22,
    },
    cardWrapper: {
        paddingHorizontal: theme?.metrics?.spacing?.lg || 16,
        paddingVertical: theme?.metrics?.spacing?.md || 14,
        backgroundColor: theme.colors.background,
        borderTopWidth: 1,
        borderTopColor: theme.colors.border,
    },
    card: {
        backgroundColor: theme.colors.card,
        borderRadius: theme?.metrics?.radii?.lg || 16,
        padding: theme?.metrics?.spacing?.lg || 16,
        borderWidth: 1,
        borderColor: theme.colors.border,
        ...(theme?.shadows?.sm || {}),
    },
    cardHeader: {
        flexDirection: 'row',
        alignItems: 'center',
        gap: 8,
        marginBottom: 12,
    },
    cardTitle: {
        fontSize: 15,
        fontWeight: '600',
        color: theme.colors.text,
    },
    inputRow: {
        flexDirection: 'row',
        alignItems: 'center',
        gap: 10,
    },
    inputFlex: {
        flex: 1,
    },
    goButton: {
        minWidth: 70,
    },
});

export default HomeScreen;
