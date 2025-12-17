import React, { useState, useEffect } from 'react';
import { View, Text, TextInput, Button, StyleSheet, Switch, Alert } from 'react-native';
import api from '../services/api';
import { useToast } from '../context/ToastContext';

const CreateBikeScreen = ({ route, navigation }) => {
    const { initialNumericalId, initialHashId } = route.params || {};
    const { showToast } = useToast();

    const [numericalId, setNumericalId] = useState(initialNumericalId || '');
    const [hashId, setHashId] = useState(initialHashId || '');
    const [isElectric, setIsElectric] = useState(false);
    const [loading, setLoading] = useState(false);

    // Ensure we update state if params change while component is mounted (though usually it's a new mount)
    useEffect(() => {
        if (initialNumericalId) setNumericalId(initialNumericalId);
        if (initialHashId) setHashId(initialHashId);
    }, [initialNumericalId, initialHashId]);

    const handleSubmit = async () => {
        setLoading(true);
        try {
            const response = await api.post('/bikes', {
                numerical_id: parseInt(numericalId),
                hash_id: hashId,
                is_electric: isElectric
            });
            showToast("Bike created successfully!", "success");
            // Navigate to CreateReview (replacing CreateBike screen)
            navigation.replace('CreateReview', { bike: response.data });
        } catch (e) {
            console.error(e);
            showToast("Failed to create bike. " + (e.response?.status === 401 ? "Unauthorized" : ""), "error");
        } finally {
            setLoading(false);
        }
    };

    return (
        <View style={styles.container}>
            <Text style={styles.title}>Add a New Bike</Text>

            <TextInput
                placeholder="Numerical ID (e.g. 101)"
                style={styles.input}
                value={numericalId}
                onChangeText={setNumericalId}
                keyboardType="numeric"
            />

            <TextInput
                placeholder="Hash ID (e.g. frame-xyz)"
                style={styles.input}
                value={hashId}
                onChangeText={setHashId}
            />

            <View style={styles.switchContainer}>
                <Text>Electric Bike?</Text>
                <Switch value={isElectric} onValueChange={setIsElectric} />
            </View>

            <Button title="Create Bike" onPress={handleSubmit} disabled={loading} />
        </View>
    );
};

const styles = StyleSheet.create({
    container: { flex: 1, padding: 20 },
    title: { fontSize: 24, marginBottom: 20 },
    input: {
        height: 40,
        borderColor: 'gray',
        borderWidth: 1,
        marginBottom: 12,
        paddingHorizontal: 8,
        borderRadius: 4
    },
    switchContainer: {
        flexDirection: 'row',
        alignItems: 'center',
        marginBottom: 20,
        justifyContent: 'space-between'
    }
});

export default CreateBikeScreen;
