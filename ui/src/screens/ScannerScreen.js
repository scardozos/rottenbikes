import React, { useState, useRef, useContext, useEffect } from 'react';
import { Text, View, StyleSheet, Button, ActivityIndicator, Platform, Alert } from 'react-native';
// Only import CameraView/Permissions for Native. Web uses html5-qrcode dynamically.
import { CameraView, useCameraPermissions } from 'expo-camera';
import api from '../services/api';
import { useToast } from '../context/ToastContext';
import { ThemeContext } from '../context/ThemeContext';

let Html5Qrcode;
let Html5QrcodeSupportedFormats;

if (Platform.OS === 'web') {
    try {
        const html5QrCodeLib = require('html5-qrcode');
        Html5Qrcode = html5QrCodeLib.Html5Qrcode;
        Html5QrcodeSupportedFormats = html5QrCodeLib.Html5QrcodeSupportedFormats;
    } catch (e) {
        console.warn("Failed to load html5-qrcode", e);
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
                <View style={{ flex: 1, justifyContent: 'center', alignItems: 'center', padding: 20, backgroundColor: 'white' }}>
                    <Text style={{ color: 'red', fontSize: 16, marginBottom: 10, textAlign: 'center' }}>Scanner Error</Text>
                    <Text style={{ color: '#555', marginBottom: 20 }}>{this.state.error?.toString()}</Text>
                    <Button title="Retry" onPress={() => this.setState({ hasError: false })} />
                </View>
            );
        }
        return this.props.children;
    }
}

const ScannerScreen = (props) => {
    if (Platform.OS === 'web') {
        return (
            <ErrorBoundary>
                <WebScannerLocal {...props} />
            </ErrorBoundary>
        );
    }
    return <NativeScannerLocal {...props} />;
};

const WebScannerLocal = ({ navigation }) => {
    // cameraActive = false -> Show Start Overlay
    // cameraActive = true -> Hide Overlay (Camera is scanning underneath)
    const [cameraActive, setCameraActive] = useState(false);
    const { showToast } = useToast();
    const { theme } = useContext(ThemeContext);
    const scannerRef = useRef(null);
    const isScanning = useRef(false);

    // Check for Secure Context
    const isSecure = typeof window !== 'undefined' && window.isSecureContext;

    useEffect(() => {
        return () => {
            // Cleanup on unmount
            if (scannerRef.current) {
                try {
                    // Try to stop if running
                    scannerRef.current.stop().catch(e => console.warn(e));
                    scannerRef.current.clear();
                } catch (e) { }
            }
        };
    }, []);

    const startScanning = async () => {
        if (!Html5Qrcode) {
            Alert.alert("Error", "Scanner library not loaded.");
            return;
        }

        try {
            const element = document.getElementById("reader");
            if (!element) {
                Alert.alert("Error", "Reader element missing. Please reload.");
                return;
            }

            // Ensure formats are available
            const formatsToSupport = [];
            if (Html5QrcodeSupportedFormats) {
                if (Html5QrcodeSupportedFormats.QR_CODE) formatsToSupport.push(Html5QrcodeSupportedFormats.QR_CODE);
                if (Html5QrcodeSupportedFormats.DATA_MATRIX) formatsToSupport.push(Html5QrcodeSupportedFormats.DATA_MATRIX);
            }

            const config = formatsToSupport.length > 0 ? { formatsToSupport } : undefined;

            // Initialize on the ALREADY VISIBLE (but covered) element
            const html5QrCode = new Html5Qrcode("reader", config);
            scannerRef.current = html5QrCode;

            // Start immediately - User Gesture is preserved here
            await html5QrCode.start(
                { facingMode: "environment" },
                { fps: 10, qrbox: { width: 250, height: 250 } },
                (decodedText) => handleScanSuccess(decodedText),
                () => { }
            );

            // Once started successfully, hide the overlay to reveal the camera
            setCameraActive(true);

        } catch (err) {
            console.error("Error starting scanner", err);
            Alert.alert("Start Failed", err?.message || "Unknown error");
        }
    };

    const stopScanning = async () => {
        if (scannerRef.current) {
            try {
                await scannerRef.current.stop();
                scannerRef.current.clear();
            } catch (ignore) { }
        }
        setCameraActive(false); // Show overlay again
    };

    const handleScanSuccess = async (data) => {
        if (isScanning.current) return;
        isScanning.current = true;

        // Stop scanning
        if (scannerRef.current) {
            try {
                await scannerRef.current.stop();
                scannerRef.current.clear();
            } catch (e) { }
        }

        // Show overlay (or loading state) while we process
        setCameraActive(false);

        // Delay for navigation safety
        setTimeout(async () => {
            try {
                const response = await api.get('/bikes');
                const bikes = response.data || [];
                const bike = bikes.find(b => b.hash_id === data);

                if (bike) {
                    showToast(`Found Bike #${bike.numerical_id}`, "success");
                    isScanning.current = false;
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
                const errMsg = e.response?.data?.error || "Failed to lookup bike after scan.";
                showToast(errMsg, "error");
                isScanning.current = false;
            }
        }, 300);
    };

    const stylesInternal = createStyles(theme);

    if (Platform.OS === 'web' && !isSecure) {
        return (
            <View style={[stylesInternal.container, { backgroundColor: theme.colors.background }]}>
                <Text style={[stylesInternal.message, { color: 'red' }]}>
                    Camera requires a Secure Context (HTTPS or Localhost).
                </Text>
            </View>
        );
    }

    return (
        <View style={{ flex: 1, backgroundColor: 'black', position: 'relative', width: '100%' }}>

            {/* 1. READER LAYER (Bottom) */}
            {/* Always rendered, always layout-ready. */}
            <View
                nativeID="reader"
                style={{
                    width: '100%',
                    height: '100%',
                    backgroundColor: 'black'
                }}
            />

            {/* 2. OVERLAY LAYER (Top) */}
            {/* Covers the reader when camera is NOT active */}
            {!cameraActive && (
                <View style={{
                    ...StyleSheet.absoluteFillObject,
                    backgroundColor: theme.colors.background,
                    justifyContent: 'center',
                    alignItems: 'center',
                    zIndex: 10
                }}>
                    <Text style={stylesInternal.message}>Ready to Scan (Web)</Text>
                    <Button title="Start Camera" onPress={startScanning} color={theme.colors.primary} />
                </View>
            )}

            {/* 3. STOP BUTTON (Floating on top when active) */}
            {cameraActive && (
                <View style={{ position: 'absolute', bottom: 30, width: '100%', alignItems: 'center', zIndex: 20 }}>
                    <Button title="Stop Camera" onPress={stopScanning} color="red" />
                </View>
            )}
        </View>
    );
};

const NativeScannerLocal = ({ navigation }) => {
    const [permission, requestPermission] = useCameraPermissions();
    const [scanned, setScanned] = useState(false);
    const { showToast } = useToast();
    const { theme } = useContext(ThemeContext);
    const isScanning = useRef(false);

    if (!permission) {
        return (
            <View style={[styles.container, { backgroundColor: theme.colors.background }]}>
                <ActivityIndicator size="large" color={theme.colors.primary} />
                <Text style={{ marginTop: 10, color: theme.colors.text }}>Initializing Camera...</Text>
            </View>
        );
    }

    if (!permission.granted) {
        return (
            <View style={[styles.container, { backgroundColor: theme.colors.background }]}>
                <Text style={styles.message}>We need your permission to show the camera</Text>
                <Button onPress={requestPermission} title="Grant Permission" color={theme.colors.primary} />
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
                showToast(`Found Bike #${bike.numerical_id}`, "success");
                isScanning.current = false;
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
            const errMsg = e.response?.data?.error || "Failed to lookup bike after scan.";
            showToast(errMsg, "error");
            isScanning.current = false;
            setScanned(false);
        }
    };

    const stylesInternal = createStyles(theme);

    return (
        <View style={stylesInternal.container}>
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
                    bottom: 50,
                    left: 20,
                    right: 20,
                    backgroundColor: theme.colors.card,
                    borderRadius: 10,
                    padding: 10,
                    opacity: 0.9
                }}>
                    <Button title={'Tap to Scan Again'} onPress={() => setScanned(false)} color={theme.colors.primary} />
                </View>
            )}
        </View>
    );
};

const styles = { container: { flex: 1 } };

const createStyles = (theme) => StyleSheet.create({
    container: {
        flex: 1,
        justifyContent: 'center',
        alignItems: 'center',
        width: '100%'
    },
    message: {
        textAlign: 'center',
        padding: 20,
        color: theme.colors.text,
        fontSize: 16
    }
});

export default ScannerScreen;
