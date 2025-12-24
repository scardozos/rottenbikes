import React, { useContext } from 'react';
import { NavigationContainer, DefaultTheme, DarkTheme } from '@react-navigation/native';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { ActivityIndicator, View, Button, TouchableOpacity, Text } from 'react-native';

import { AuthContext } from '../context/AuthContext';
import { ThemeContext } from '../context/ThemeContext';
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
            Login: 'login',
            Register: 'register',
            Scanner: 'scanner',
            CreateBike: 'create-bike',
            BikeDetails: 'bike/:bikeId',
            ConfirmLogin: 'confirm/:token',
        },
    },
};

const AppNavigator = () => {
    const { isLoading, userToken } = useContext(AuthContext);
    const { theme, isDark, toggleTheme } = useContext(ThemeContext);

    const baseTheme = isDark ? DarkTheme : DefaultTheme;
    const navTheme = {
        ...baseTheme,
        colors: {
            ...baseTheme.colors,
            ...theme.colors,
            // Ensure compatibility with React Navigation Theme object
            // Required keys: primary, background, card, text, border, notification
            primary: theme.colors.primary,
            background: theme.colors.background,
            card: theme.colors.card,
            text: theme.colors.text,
            border: theme.colors.border,
            notification: theme.colors.notification,
        },
    };

    if (isLoading) {
        return (
            <View style={{ flex: 1, justifyContent: 'center', alignItems: 'center', backgroundColor: theme.colors.background }}>
                <ActivityIndicator size="large" color={theme.colors.primary} />
            </View>
        );
    }

    const ThemeToggleButton = () => (
        <TouchableOpacity onPress={toggleTheme} style={{ marginRight: 15 }}>
            <Text style={{ fontSize: 24 }}>{isDark ? '☀️' : '🌙'}</Text>
        </TouchableOpacity>
    );

    return (
        <NavigationContainer linking={linking} theme={navTheme}>
            <Stack.Navigator
                initialRouteName={userToken == null ? "Login" : "Home"}
                screenOptions={{
                    headerRight: () => <ThemeToggleButton />,
                    headerTitleStyle: { color: theme.colors.text },
                    headerTintColor: theme.colors.primary,
                    headerStyle: {
                        backgroundColor: theme.colors.card,
                    }
                }}
            >
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
