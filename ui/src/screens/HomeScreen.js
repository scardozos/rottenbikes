import React, { useState, useRef, useContext, useEffect } from 'react';
import { Text, View, StyleSheet, Button, ActivityIndicator, Platform, Alert, TextInput, KeyboardAvoidingView, Pressable, Keyboard } from 'react-native';
// Only import CameraView/Permissions for Native. Web uses html5-qrcode dynamically.
import { CameraView, useCameraPermissions } from 'expo-camera';
import api from '../services/api';
import { useToast } from '../context/ToastContext';
import { ThemeContext } from '../context/ThemeContext';
import { AuthContext } from '../context/AuthContext';
import { useSession } from '../context/SessionContext';
import { LanguageContext } from '../context/LanguageContext';
import { Scanner } from '../components/Scanner';
import { isNumeric } from '../utils/validation';

let WebScanner;

if (Platform.OS === 'web') {
    try {
        const scannerLib = require('@yudiel/react-qr-scanner');
        WebScanner = scannerLib.Scanner;
    } catch (e) {
        console.warn("Failed to load @yudiel/react-qr-scanner", e);
    }
}

class ErrorBoundary extends React.Component {
    constructor(props) {
        super(props);
        this.state = { hasError: false, error: null };
    }
    static getDerivedStateFromError(error) {
        return { hasError: true, error };
    }
    componentDidCatch(error, errorInfo) {
        console.error("Scanner ErrorBoundary:", error, errorInfo);
    }
    render() {
        if (this.state.hasError) {
            return (
                <View style={{ flex: 1, justifyContent: 'center', alignItems: 'center', padding: 20 }}>
                    <Text style={{ color: 'red', fontSize: 16, marginBottom: 10, textAlign: 'center' }}>Scanner Error</Text>
                    <Text style={{ color: '#555', marginBottom: 20 }}>{this.state.error?.toString()}</Text>
                    <Button title="Retry" onPress={() => this.setState({ hasError: false })} />
                </View>
            );
        }
        return this.props.children;
    }
}

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

        // Basic numerical validation
        if (!isNumeric(manualId)) {
            showToast(t('invalid_numerical_id'), "error");
            return;
        }

        const bikeId = manualId;

        try {
            // Verify bike exists before validating session
            await api.get(`/bikes/${bikeId}/details`);
            console.log('[HomeScreen] Manual Submit & Verified:', bikeId);
            validateBike(bikeId);
            navigation.navigate('BikesList', { screen: 'BikeDetails', params: { bikeId } });
            setManualId('');
        } catch (e) {
            console.log('[HomeScreen] Bike lookup error:', e);

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
                        // Don't reset isScanning here so it doesn't immediately scan again if they navigate back
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
            console.error("error during scan lookup", e);
            const errMsg = e.response?.data?.error || t('scan_lookup_failed');
            showToast(errMsg, "error");
            isScanning.current = false;
        }
    };

    const stylesInternal = createStyles(theme);

    // Naive check for mobile browser agent
    const isMobileWeb = Platform.OS === 'web' && /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent);
    const showCamera = (Platform.OS !== 'web' || isMobileWeb);
    const shouldRenderCamera = showCamera && !isInputActive;

    const content = (
        <KeyboardAvoidingView
            style={stylesInternal.container}
            behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
            keyboardVerticalOffset={Platform.OS === 'ios' ? 100 : 0}
        >
            {/* Camera Area - Top 70% (Only on Mobile App or Mobile Web) */}
            {showCamera ? (
                <View style={stylesInternal.cameraContainer}>
                    {shouldRenderCamera ? (
                        <ErrorBoundary>
                            <Scanner onScan={handleScanSuccess} theme={theme} t={t} />
                        </ErrorBoundary>
                    ) : (
                        <View style={{ flex: 1, backgroundColor: 'black', justifyContent: 'center', alignItems: 'center' }}>
                            <Text style={{ color: 'gray' }}>{t('scanner_paused') || "Scanner Paused"}</Text>
                        </View>
                    )}
                </View>
            ) : (
                <View style={{ flex: 1, backgroundColor: theme.colors.background }} />
            )}

            {/* Manual Input Area - Bottom 30% */}
            <View style={stylesInternal.inputContainer}>
                <Text style={stylesInternal.inputLabel}>{t('enter_manual_id')}</Text>
                <View style={stylesInternal.inputRow}>
                    <TextInput
                        style={stylesInternal.input}
                        placeholder={t('bike_id_placeholder')}
                        placeholderTextColor={theme.colors.placeholder}
                        keyboardType="numeric"
                        value={manualId}
                        onChangeText={setManualId}
                        returnKeyType="done"
                        onSubmitEditing={handleManualSubmit}
                        onFocus={() => Platform.OS === 'web' && setIsInputActive(true)}
                        onBlur={() => Platform.OS === 'web' && setIsInputActive(false)}
                    />
                    <Button title={t('go')} onPress={handleManualSubmit} color={theme.colors.primary} />
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
        backgroundColor: theme.colors.background
    },
    cameraContainer: {
        flex: 2, // Takes up ~66% of screen
        backgroundColor: 'black',
        overflow: 'hidden'
    },
    inputContainer: {
        flex: 1, // Takes up ~33% of screen
        backgroundColor: theme.colors.card,
        padding: 20,
        justifyContent: 'center',
        borderTopLeftRadius: 20,
        borderTopRightRadius: 20,
        // On desktop web without camera, let it center or fill nicely?
        // Current layout: Camera (flex 2) + Input (flex 1).
        // If Camera is hidden (replaced by empty view), Input stays at bottom.
        // It's acceptable for now to keep consistent layout.
        ...Platform.select({
            ios: {
                shadowColor: '#000',
                shadowOffset: { width: 0, height: -2 },
                shadowOpacity: 0.1,
                shadowRadius: 5,
            },
            android: {
                elevation: 10,
            },
        }),
    },
    inputLabel: {
        fontSize: 16,
        fontWeight: 'bold',
        marginBottom: 10,
        color: theme.colors.text
    },
    inputRow: {
        flexDirection: 'row',
        alignItems: 'center',
        marginBottom: 10
    },
    input: {
        flex: 1,
        height: 50,
        borderColor: theme.colors.border,
        borderWidth: 1,
        borderRadius: 8,
        paddingHorizontal: 15,
        marginRight: 10,
        backgroundColor: theme.colors.inputBackground,
        fontSize: 18,
        color: theme.colors.text
    },
    scannerMessageContainer: {
        flex: 1,
        justifyContent: 'center',
        alignItems: 'center'
    },
    message: {
        textAlign: 'center',
        padding: 20,
        color: theme.colors.text,
        fontSize: 16
    },
    nativeCameraContainer: {
        flex: 1,
        width: '100%'
    }
});

export default HomeScreen;
