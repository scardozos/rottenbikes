import React, { useState, useRef, useEffect } from 'react';
import { View, Text, Button, StyleSheet, ActivityIndicator, Platform, Alert } from 'react-native';
// Only import CameraView/Permissions for Native. Web uses html5-qrcode dynamically.
import { CameraView, useCameraPermissions } from 'expo-camera';

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
                <Text style={{ color: 'red' }}>Camera requires Secure Context.</Text>
                <Button title={t('cancel')} onPress={onClose} color={theme.colors.primary} />
            </View>
        );
    }

    if (!WebScanner) {
        return (
            <View style={styles.center}>
                <Text style={{ color: 'red' }}>Scanner library not loaded.</Text>
                <Button title={t('cancel')} onPress={onClose} color={theme.colors.primary} />
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
            <View style={styles.overlay}>
                <Button title={t('cancel')} onPress={onClose} color="red" />
            </View>
        </View>
    );
};

const NativeScannerWrapper = ({ onScan, onClose, theme, t }) => {
    const [permission, requestPermission] = useCameraPermissions();
    const [scanned, setScanned] = useState(false);

    if (!permission) return <View style={styles.center}><ActivityIndicator color={theme.colors.primary} /></View>;
    if (!permission.granted) {
        return (
            <View style={styles.center}>
                <Text style={{ color: theme.colors.text, marginBottom: 10 }}>{t('camera_permission')}</Text>
                <Button onPress={requestPermission} title={t('grant_permission')} color={theme.colors.primary} />
                <Button onPress={onClose} title={t('cancel')} color="red" />
            </View>
        );
    }

    return (
        <View style={styles.fullScreen}>
            <CameraView
                style={StyleSheet.absoluteFillObject}
                onBarcodeScanned={scanned ? undefined : ({ data }) => {
                    setScanned(true);
                    onScan(data);
                }}
                barcodeScannerSettings={{
                    barcodeTypes: ["qr", "aztec", "ean13", "code128", "pdf417", "upc_e", "datamatrix"],
                }}
            />
            <View style={styles.overlay}>
                <Button title={t('cancel')} onPress={onClose} color="red" />
            </View>
        </View>
    );
};

const styles = StyleSheet.create({
    center: { flex: 1, justifyContent: 'center', alignItems: 'center', padding: 20 },
    fullScreen: { flex: 1, backgroundColor: 'black' },
    webScannerContainer: { width: '100%', height: '100%', maxWidth: 500, maxHeight: 500, alignSelf: 'center' },
    overlay: { position: 'absolute', bottom: 40, alignSelf: 'center', width: 150 }
});
