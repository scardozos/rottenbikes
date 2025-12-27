import React, { useState, useRef, useContext } from 'react';
import { Text, View, StyleSheet, Button, ActivityIndicator, Platform, Alert, TextInput, KeyboardAvoidingView, TouchableWithoutFeedback, Keyboard } from 'react-native';
// Only import CameraView/Permissions for Native. Web uses html5-qrcode dynamically.
import { CameraView, useCameraPermissions } from 'expo-camera';
import api from '../services/api';
import { useToast } from '../context/ToastContext';
import { ThemeContext } from '../context/ThemeContext';
import { AuthContext } from '../context/AuthContext';
import { useSession } from '../context/SessionContext';
import { LanguageContext } from '../context/LanguageContext';

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
    const { showToast } = useToast();
    const { t } = useContext(LanguageContext); // Use language context

    const handleManualSubmit = async () => {
        if (!manualId.trim()) return;

        // Basic numerical validation
        if (!/^\d+$/.test(manualId)) {
            showToast(t('invalid_numerical_id'), "error");
            return;
        }

        const bikeId = parseInt(manualId, 10);

        try {
            // Verify bike exists before validating session
            await api.get(`/bikes/${bikeId}/details`);
            console.log('[HomeScreen] Manual Submit & Verified:', bikeId);
            validateBike(bikeId);
            navigation.navigate('BikeDetails', { bikeId });
            setManualId('');
        } catch (e) {
            console.log('[HomeScreen] Bike not found:', bikeId);

            if (Platform.OS === 'web') {
                const create = window.confirm(`Bike #${bikeId} not found. Would you like to create it?`);
                if (create) {
                    navigation.navigate('CreateBike', { initialNumericalId: bikeId });
                }
            } else {
                Alert.alert(
                    "Bike Not Found",
                    `Bike #${bikeId} not found. Would you like to create it?`,
                    [
                        { text: "Cancel", style: "cancel" },
                        {
                            text: "Create",
                            onPress: () => navigation.navigate('CreateBike', { initialNumericalId: bikeId })
                        }
                    ]
                );
            }
        }
    };

    const stylesInternal = createStyles(theme);

    const content = (
        <KeyboardAvoidingView
            style={stylesInternal.container}
            behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
            keyboardVerticalOffset={Platform.OS === 'ios' ? 100 : 0}
        >
            {/* Camera Area - Top 70% */}
            <View style={stylesInternal.cameraContainer}>
                {Platform.OS === 'web' ? (
                    <ErrorBoundary>
                        <WebScannerLocal navigation={navigation} theme={theme} validateBike={validateBike} t={t} />
                    </ErrorBoundary>
                ) : (
                    <NativeScannerLocal navigation={navigation} theme={theme} validateBike={validateBike} t={t} />
                )}
            </View>

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
        <TouchableWithoutFeedback onPress={Keyboard.dismiss}>
            {content}
        </TouchableWithoutFeedback>
    );
};

const WebScannerLocal = ({ navigation, theme, validateBike, t }) => {
    const { showToast } = useToast();
    const isScanning = useRef(false);

    // Check for Secure Context
    const isSecure = typeof window !== 'undefined' && window.isSecureContext;

    const handleScanSuccess = async (data) => {
        if (isScanning.current) return;
        isScanning.current = true;

        // Delay for navigation safety
        setTimeout(async () => {
            try {
                const response = await api.get('/bikes');
                const bikes = response.data || [];
                const bike = bikes.find(b => b.hash_id === data);

                if (bike) {
                    showToast(t('found_bike', { id: bike.numerical_id }), "success");
                    isScanning.current = false;
                    validateBike(bike.numerical_id);
                    navigation.navigate('BikeDetails', { bikeId: bike.numerical_id });
                } else {
                    const create = window.confirm(`No bike found with Hash ID: ${data}. Create it?`);
                    if (create) {
                        navigation.navigate('CreateBike', { initialHashId: data });
                    } else {
                        isScanning.current = false;
                    }
                }
            } catch (e) {
                const errMsg = e.response?.data?.error || t('scan_lookup_failed');
                showToast(errMsg, "error");
                isScanning.current = false;
            }
        }, 300);
    };

    const stylesInternal = createStyles(theme);

    if (Platform.OS === 'web' && !isSecure) {
        return (
            <View style={[stylesInternal.scannerMessageContainer, { backgroundColor: theme.colors.background }]}>
                <Text style={[stylesInternal.message, { color: 'red' }]}>
                    Camera requires a Secure Context (HTTPS or Localhost).
                </Text>
            </View>
        );
    }

    if (!WebScanner) {
        return (
            <View style={[stylesInternal.scannerMessageContainer, { backgroundColor: theme.colors.background }]}>
                <Text style={[stylesInternal.message, { color: 'red' }]}>
                    Scanner library not loaded.
                </Text>
            </View>
        );
    }

    return (
        <View style={{ flex: 1, backgroundColor: 'black', width: '100%', height: '100%', justifyContent: 'center', alignItems: 'center' }}>
            <View style={{ width: '100%', height: '100%', maxWidth: 500, maxHeight: 500 }}>
                <WebScanner
                    onScan={(result) => {
                        if (result && result.length > 0) {
                            handleScanSuccess(result[0].rawValue);
                        }
                    }}
                    components={{
                        audio: false,
                        finder: false
                    }}
                    styles={{
                        container: {
                            width: "100%",
                            height: "100%"
                        }
                    }}
                />
            </View>
            <Text style={{ color: 'white', marginTop: 20 }}>{t('scan_qr')}</Text>
        </View>
    );
};

const NativeScannerLocal = ({ navigation, theme, validateBike, t }) => {
    const [permission, requestPermission] = useCameraPermissions();
    const [scanned, setScanned] = useState(false);
    const { showToast } = useToast();
    const isScanning = useRef(false);

    if (!permission) {
        return (
            <View style={[styles.scannerMessageContainer, { backgroundColor: theme.colors.background }]}>
                <ActivityIndicator size="large" color={theme.colors.primary} />
                <Text style={{ marginTop: 10, color: theme.colors.text }}>{t('loading')}</Text>
            </View>
        );
    }

    if (!permission.granted) {
        return (
            <View style={[styles.scannerMessageContainer, { backgroundColor: theme.colors.background }]}>
                <Text style={styles.message}>{t('camera_permission')}</Text>
                <Button onPress={requestPermission} title={t('grant_permission')} color={theme.colors.primary} />
            </View>
        );
    }

    const handleBarCodeScanned = async ({ data }) => {
        if (isScanning.current) return;
        isScanning.current = true;
        setScanned(true);

        try {
            const response = await api.get('/bikes');
            const bikes = response.data || [];
            const bike = bikes.find(b => b.hash_id === data);

            if (bike) {
                showToast(t('found_bike', { id: bike.numerical_id }), "success");
                isScanning.current = false;
                validateBike(bike.numerical_id);
                navigation.navigate('BikeDetails', { bikeId: bike.numerical_id });
            } else {
                Alert.alert(
                    "Not Found",
                    `No bike found with Hash ID: ${data}. Would you like to create it?`,
                    [
                        {
                            text: "Cancel",
                            onPress: () => {
                                isScanning.current = false;
                                setScanned(false);
                            },
                            style: "cancel"
                        },
                        {
                            text: "Create",
                            onPress: () => {
                                navigation.replace('CreateBike', { initialHashId: data });
                            }
                        }
                    ]
                );
            }
        } catch (e) {
            console.error("error during scan lookup", e);
            const errMsg = e.response?.data?.error || t('scan_lookup_failed');
            showToast(errMsg, "error");
            isScanning.current = false;
            setScanned(false);
        }
    };

    const stylesInternal = createStyles(theme);

    return (
        <View style={stylesInternal.nativeCameraContainer}>
            <CameraView
                style={StyleSheet.absoluteFillObject}
                onBarcodeScanned={scanned ? undefined : handleBarCodeScanned}
                barcodeScannerSettings={{
                    barcodeTypes: ["qr", "aztec", "ean13", "code128", "pdf417", "upc_e", "datamatrix"],
                }}
            />
            {scanned && (
                <View style={{
                    position: 'absolute',
                    bottom: 20,
                    left: 20,
                    right: 20,
                    backgroundColor: theme.colors.card,
                    borderRadius: 10,
                    padding: 10,
                    opacity: 0.9,
                    alignItems: 'center'
                }}>
                    <Button title={'Tap to Scan Again'} onPress={() => setScanned(false)} color={theme.colors.primary} />
                </View>
            )}
        </View>
    );
};

const styles = { container: { flex: 1 }, scannerMessageContainer: { flex: 1, justifyContent: 'center', alignItems: 'center' } };

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
