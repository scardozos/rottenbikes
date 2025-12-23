import React, { useState, useRef } from 'react';
import { Text, View, StyleSheet, Button, Alert } from 'react-native';
import { CameraView, useCameraPermissions } from 'expo-camera';
import api from '../services/api';
import { useToast } from '../context/ToastContext';

const ScannerScreen = ({ navigation }) => {
    const [permission, requestPermission] = useCameraPermissions();
    const [scanned, setScanned] = useState(false);
    const { showToast } = useToast();
    const isScanning = useRef(false);

    if (!permission) {
        // Camera permissions are still loading.
        return <View />;
    }

    if (!permission.granted) {
        // Camera permissions are not granted yet.
        return (
            <View style={styles.container}>
                <Text style={styles.message}>We need your permission to show the camera</Text>
                <Button onPress={requestPermission} title="Grant Permission" />
            </View>
        );
    }

    const handleBarCodeScanned = async ({ data }) => {
        if (isScanning.current) return;
        isScanning.current = true;
        setScanned(true);

        // Data should be the hash_id
        try {
            const response = await api.get('/bikes');
            const bikes = response.data || [];
            const bike = bikes.find(b => b.hash_id === data);

            if (bike) {
                showToast(`Found Bike #${bike.numerical_id}`, "success");
                isScanning.current = false;
                navigation.navigate('BikeDetails', { bike });
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
                                // isScanning.current remains true as we navigate away
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

    return (
        <View style={styles.container}>
            <CameraView
                style={StyleSheet.absoluteFillObject}
                onBarcodeScanned={scanned ? undefined : handleBarCodeScanned}
                barcodeScannerSettings={{
                    barcodeTypes: ["qr", "aztec", "ean13", "code128", "pdf417", "upc_e", "datamatrix"],
                }}
            />
            {scanned && (
                <View style={styles.scanAgainContainer}>
                    <Button title={'Tap to Scan Again'} onPress={() => setScanned(false)} />
                </View>
            )}
        </View>
    );
};

const styles = StyleSheet.create({
    container: {
        flex: 1,
        justifyContent: 'center',
        backgroundColor: '#000',
    },
    message: {
        textAlign: 'center',
        paddingBottom: 10,
        color: '#fff'
    },
    scanAgainContainer: {
        position: 'absolute',
        bottom: 50,
        left: 20,
        right: 20,
        backgroundColor: 'rgba(255,255,255,0.8)',
        borderRadius: 10,
        padding: 10
    }
});

export default ScannerScreen;
