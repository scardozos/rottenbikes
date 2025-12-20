import React, { useContext } from 'react';
import { NavigationContainer } from '@react-navigation/native';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { ActivityIndicator, View } from 'react-native';

import { AuthContext } from '../context/AuthContext';
import LoginScreen from '../screens/LoginScreen';
import RegisterScreen from '../screens/RegisterScreen';
import ConfirmLoginScreen from '../screens/ConfirmLoginScreen';
import HomeScreen from '../screens/HomeScreen';
import BikeDetailsScreen from '../screens/BikeDetailsScreen';
import CreateBikeScreen from '../screens/CreateBikeScreen';
import CreateReviewScreen from '../screens/CreateReviewScreen';
import ScannerScreen from '../screens/ScannerScreen';

const Stack = createNativeStackNavigator();

const linking = {
    prefixes: ['http://localhost:8081', 'rottenbikes://'],
    config: {
        screens: {
            Home: 'home',
            ConfirmLogin: 'confirm/:token',
        },
    },
};

const AppNavigator = () => {
    const { isLoading, userToken } = useContext(AuthContext);

    if (isLoading) {
        return (
            <View style={{ flex: 1, justifyContent: 'center', alignItems: 'center' }}>
                <ActivityIndicator size="large" />
            </View>
        );
    }

    return (
        <NavigationContainer linking={linking}>
            <Stack.Navigator initialRouteName={userToken == null ? "Login" : "Home"}>
                {userToken == null ? (
                    // Auth Stack
                    <>
                        <Stack.Screen name="Login" component={LoginScreen} options={{ headerShown: false }} />
                        <Stack.Screen name="Register" component={RegisterScreen} />
                    </>
                ) : (
                    // App Stack
                    <>
                        <Stack.Screen name="Home" component={HomeScreen} />
                        <Stack.Screen name="BikeDetails" component={BikeDetailsScreen} options={{ title: 'Bike Details' }} />
                        <Stack.Screen name="CreateBike" component={CreateBikeScreen} options={{ title: 'Add Bike' }} />
                        <Stack.Screen name="CreateReview" component={CreateReviewScreen} options={{ title: 'Write Review' }} />
                        <Stack.Screen name="Scanner" component={ScannerScreen} options={{ title: 'Scan QR Code' }} />
                    </>
                )}
                <Stack.Screen name="ConfirmLogin" component={ConfirmLoginScreen} options={{ title: 'Confirming Login' }} />
            </Stack.Navigator>
        </NavigationContainer>
    );
};

export default AppNavigator;
