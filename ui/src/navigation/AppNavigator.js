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
import BikesListScreen from '../screens/BikesListScreen';
import BikeDetailsScreen from '../screens/BikeDetailsScreen';
import CreateBikeScreen from '../screens/CreateBikeScreen';
import CreateReviewScreen from '../screens/CreateReviewScreen';
import ConfigurationScreen from '../screens/ConfigurationScreen';

import { createBottomTabNavigator } from '@react-navigation/bottom-tabs';
import { LanguageContext } from '../context/LanguageContext';

const Stack = createNativeStackNavigator();
const Tab = createBottomTabNavigator();

const linking = {
    prefixes: ['http://localhost:8081', 'rottenbikes://'],
    config: {
        screens: {
            Main: {
                screens: {
                    Home: 'home',
                    BikesList: 'bikes',
                    Configuration: 'config',
                }
            },
            Login: 'login',
            Register: 'register',
            CreateBike: 'create-bike',
            BikeDetails: 'bike/:bikeId',
            ConfirmLogin: 'confirm/:token',
        },
    },
};

const MainTabs = () => {
    const { theme } = useContext(ThemeContext);
    const { t } = useContext(LanguageContext);

    return (
        <Tab.Navigator
            screenOptions={({ route }) => ({
                tabBarIcon: ({ focused, color, size }) => {
                    let iconName;

                    if (route.name === 'Home') {
                        iconName = '📷';
                    } else if (route.name === 'BikesList') {
                        iconName = '🚲';
                    } else if (route.name === 'Configuration') {
                        iconName = '⚙️';
                    }

                    return <Text style={{ fontSize: size, color: color }}>{iconName}</Text>;
                },
                tabBarActiveTintColor: theme.colors.primary,
                tabBarInactiveTintColor: 'gray',
                headerShown: true,
                headerTitleStyle: { color: theme.colors.text },
                headerTintColor: theme.colors.primary,
                headerStyle: {
                    backgroundColor: theme.colors.card,
                    borderBottomWidth: 1,
                    borderBottomColor: theme.colors.border
                },
                tabBarStyle: {
                    backgroundColor: theme.colors.card,
                    borderTopColor: theme.colors.border,
                }
            })}
        >
            <Tab.Screen name="Home" component={HomeScreen} options={{ title: t('home') }} />
            <Tab.Screen name="BikesList" component={BikesListScreen} options={{ title: t('browse_bikes') }} />
            <Tab.Screen name="Configuration" component={ConfigurationScreen} options={{ title: t('settings') }} />
        </Tab.Navigator>
    );
};

const AppNavigator = () => {
    const { isLoading, userToken } = useContext(AuthContext);
    const { theme, isDark } = useContext(ThemeContext);
    const { t } = useContext(LanguageContext);

    const baseTheme = isDark ? DarkTheme : DefaultTheme;
    const navTheme = {
        ...baseTheme,
        colors: {
            ...baseTheme.colors,
            ...theme.colors,
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

    return (
        <NavigationContainer linking={linking} theme={navTheme}>
            <Stack.Navigator
                initialRouteName={userToken == null ? "Login" : "Main"}
                screenOptions={{
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
                        <Stack.Screen name="Register" component={RegisterScreen} options={{ title: t('register') }} />
                    </>
                ) : (
                    // App Stack
                    <>
                        <Stack.Screen name="Main" component={MainTabs} options={{ headerShown: false }} />
                        <Stack.Screen name="BikeDetails" component={BikeDetailsScreen} options={{ title: t('bike_details') }} />
                        <Stack.Screen name="CreateBike" component={CreateBikeScreen} options={{ title: t('add_bike_title') }} />
                        <Stack.Screen name="CreateReview" component={CreateReviewScreen} options={{ title: t('write_review') }} />
                    </>
                )}
                <Stack.Screen name="ConfirmLogin" component={ConfirmLoginScreen} options={{ title: t('confirm_login_title') }} />
            </Stack.Navigator>
        </NavigationContainer>
    );
};

export default AppNavigator;
