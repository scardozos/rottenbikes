import React, { useState, useRef, useEffect } from 'react';
import { View, Text, StyleSheet, ActivityIndicator, Platform, Linking } from 'react-native';
// Only import CameraView/Permissions for Native. Web uses html5-qrcode dynamically.
import { CameraView, useCameraPermissions } from 'expo-camera';
import { useIsFocused } from '@react-navigation/native';
import Button from './Button';

let WebScanner;
if (Platform.OS === 'web') {
    try {
        const scannerLib = require('@yudiel/react-qr-scanner');
        WebScanner = scannerLib.Scanner;
    } catch (e) {
        console.warn("Failed to load @yudiel/react-qr-scanner", e);
    }
}

export const Scanner = ({ onScan, onClose, theme, t }) => {
    if (Platform.OS === 'web') {
        return <WebScannerWrapper onScan={onScan} onClose={onClose} theme={theme} t={t} />;
    }
    return <NativeScannerWrapper onScan={onScan} onClose={onClose} theme={theme} t={t} />;
};

const WebScannerWrapper = ({ onScan, onClose, theme, t }) => {
    // Check for Secure Context
    const isSecure = typeof window !== 'undefined' && window.isSecureContext;
    const isScanning = useRef(false);

    if (!isSecure) {
        return (
            <View style={styles.center}>
                <Text style={{ color: theme.colors.error, marginBottom: 12 }}>Camera requires Secure Context.</Text>
                {onClose && <Button title={t('cancel')} onPress={onClose} variant="ghost" />}
            </View>
        );
    }

    if (!WebScanner) {
        return (
            <View style={styles.center}>
                <Text style={{ color: theme.colors.error, marginBottom: 12 }}>Scanner library not loaded.</Text>
                {onClose && <Button title={t('cancel')} onPress={onClose} variant="ghost" />}
            </View>
        );
    }

    return (
        <View style={styles.fullScreen}>
            <View style={styles.webScannerContainer}>
                <WebScanner
                    onScan={(result) => {
                        if (result && result.length > 0 && !isScanning.current) {
                            isScanning.current = true;
                            onScan(result[0].rawValue);
                        }
                    }}
                    components={{ audio: false, finder: false }}
                    styles={{ container: { width: "100%", height: "100%" } }}
                />
            </View>
            {onClose && (
                <View style={styles.overlay}>
                    <Button title={t('cancel')} onPress={onClose} variant="danger" />
                </View>
            )}
        </View>
    );
};

const NativeScannerWrapper = ({ onScan, onClose, theme, t }) => {
    const [permission, requestPermission] = useCameraPermissions();
    const [scanned, setScanned] = useState(false);
    const isFocused = useIsFocused();

    useEffect(() => {
        if (isFocused) {
            setScanned(false);
        }
    }, [isFocused]);

    if (!permission) {
        return (
            <View style={styles.center}>
                <ActivityIndicator color={theme.colors.primary} />
            </View>
        );
    }

    if (!permission.granted) {
        return (
            <View style={styles.center}>
                <Text style={{ color: theme.colors.text, marginBottom: 16, textAlign: 'center', paddingHorizontal: 20 }}>
                    {t('camera_permission')}
                </Text>
                <Button
                    onPress={requestPermission}
                    title={t('grant_permission')}
                    variant="primary"
                    style={{ marginBottom: 10 }}
                />
                <Button
                    onPress={() => Linking.openSettings()}
                    title={t('open_settings') || "Open Settings"}
                    variant="secondary"
                    style={{ marginBottom: 10 }}
                />
                {onClose && <Button onPress={onClose} title={t('cancel')} variant="danger" />}
            </View>
        );
    }

    return (
        <View style={styles.fullScreen}>
            {isFocused && (
                <CameraView
                    key={isFocused ? 'camera-active' : 'camera-inactive'}
                    style={styles.camera}
                    facing="back"
                    active={isFocused}
                    onCameraReady={() => console.log('[CameraView] camera ready')}
                    onMountError={(error) => console.error('[CameraView] mount error:', error)}
                    onBarcodeScanned={scanned ? undefined : ({ data }) => {
                        console.log('[CameraView] barcode scanned:', data);
                        setScanned(true);
                        onScan(data);
                    }}
                    barcodeScannerSettings={{
                        barcodeTypes: ["qr", "aztec", "ean13", "code128", "pdf417", "upc_e", "datamatrix"],
                    }}
                />
            )}
            {onClose && (
                <View style={styles.overlay}>
                    <Button title={t('cancel')} onPress={onClose} variant="danger" />
                </View>
            )}
        </View>
    );
};

const styles = StyleSheet.create({
    center: { flex: 1, justifyContent: 'center', alignItems: 'center', padding: 20 },
    fullScreen: { flex: 1, width: '100%', height: '100%', position: 'relative', overflow: 'hidden' },
    camera: { flex: 1, width: '100%', height: '100%' },
    webScannerContainer: { width: '100%', height: '100%', maxWidth: 500, maxHeight: 500, alignSelf: 'center' },
    overlay: { position: 'absolute', bottom: 40, alignSelf: 'center', width: 150 }
});

