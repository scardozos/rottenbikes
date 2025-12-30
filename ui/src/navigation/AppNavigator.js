import React, { useContext } from 'react';
import { NavigationContainer, DefaultTheme, DarkTheme, CommonActions } from '@react-navigation/native';
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
import UpdateReviewScreen from '../screens/UpdateReviewScreen';
import UpdateBikeScreen from '../screens/UpdateBikeScreen';
import ConfigurationScreen from '../screens/ConfigurationScreen';
import PrivacyScreen from '../screens/PrivacyScreen';


import { createBottomTabNavigator } from '@react-navigation/bottom-tabs';
import { LanguageContext } from '../context/LanguageContext';

const Stack = createNativeStackNavigator();
const HomeStack = createNativeStackNavigator();
const BikesListStack = createNativeStackNavigator();
const PublicHomeStack = createNativeStackNavigator(); // New stack for public home
const Tab = createBottomTabNavigator();

const linkingPrefixes = ['http://localhost:8081', 'rottenbikes://'];

const mainLinking = {
    prefixes: linkingPrefixes,
    config: {
        screens: {
            Main: {
                screens: {
                    Home: {
                        screens: {
                            ScanBike: 'home',
                        }
                    },
                    BikesList: {
                        screens: {
                            BikesCatalog: 'bikes',
                            BikeDetails: 'bikes/:bikeId',
                            CreateBike: 'bikes/create',
                            UpdateBike: 'bikes/:bikeId/update',
                            CreateReview: 'bikes/:bikeId/createReview',
                            UpdateReview: 'reviews/:reviewId/edit',
                        }
                    },
                    Configuration: 'config',
                }
            },
            // Shared screens accessible when logged in
            ConfirmLogin: 'confirm/:token',
            Privacy: 'privacy',
        }
    }
};

const publicLinking = {
    prefixes: linkingPrefixes,
    config: {
        screens: {
            Public: {
                screens: {
                    Home: {
                        screens: {
                            Login: 'login',
                            Register: 'register',
                        }
                    },
                    BikesList: {
                        screens: {
                            BikesCatalog: 'bikes',
                            BikeDetails: 'bikes/:bikeId',
                        }
                    }
                }
            },
            // Shared screens accessible when public
            ConfirmLogin: 'confirm/:token',
            Privacy: 'privacy',
        }
    }
};

const HomeStackNavigator = () => {
    const { theme } = useContext(ThemeContext);
    const { t } = useContext(LanguageContext);
    return (
        <HomeStack.Navigator
            screenOptions={{
                headerTitleStyle: { color: theme.colors.text },
                headerTintColor: theme.colors.primary,
                headerStyle: { backgroundColor: theme.colors.card },
            }}
        >
            <HomeStack.Screen name="ScanBike" component={HomeScreen} options={{ title: t('home') }} />
        </HomeStack.Navigator>
    );
};

const BikesListStackNavigator = () => {
    const { theme } = useContext(ThemeContext);
    const { t } = useContext(LanguageContext);
    return (
        <BikesListStack.Navigator
            screenOptions={{
                headerTitleStyle: { color: theme.colors.text },
                headerTintColor: theme.colors.primary,
                headerStyle: { backgroundColor: theme.colors.card },
            }}
        >
            <BikesListStack.Screen name="BikesCatalog" component={BikesListScreen} options={{ title: t('browse_bikes') }} />
            <BikesListStack.Screen name="BikeDetails" component={BikeDetailsScreen} options={{ title: t('bike_details') }} />
            <BikesListStack.Screen name="CreateBike" component={CreateBikeScreen} options={{ title: t('add_bike_title') }} />
            <BikesListStack.Screen name="UpdateBike" component={UpdateBikeScreen} options={{ title: t('update_bike_title') }} />
            <BikesListStack.Screen name="CreateReview" component={CreateReviewScreen} options={{ title: t('write_review') }} />
            <BikesListStack.Screen name="UpdateReview" component={UpdateReviewScreen} options={{ title: t('update_review_title') }} />
        </BikesListStack.Navigator>
    );
};

// Public Home Stack: Login -> Register
const PublicHomeStackNavigator = () => {
    const { theme } = useContext(ThemeContext);
    const { t } = useContext(LanguageContext);
    return (
        <PublicHomeStack.Navigator
            screenOptions={{
                headerTitleStyle: { color: theme.colors.text },
                headerTintColor: theme.colors.primary,
                headerStyle: { backgroundColor: theme.colors.card },
            }}
        >
            <PublicHomeStack.Screen name="Login" component={LoginScreen} options={{ headerShown: false }} />
            <PublicHomeStack.Screen name="Register" component={RegisterScreen} options={{ title: t('register') }} />
        </PublicHomeStack.Navigator>
    );
};

// Reusable tab bar icon logic
const getTabBarIcon = (route, focused, color, size) => {
    let iconName;
    if (route.name === 'Home') {
        iconName = '🏠';
    } else if (route.name === 'BikesList') {
        iconName = '🚲';
    } else if (route.name === 'Configuration') {
        iconName = '⚙️';
    }
    return <Text style={{ fontSize: size, color: color }}>{iconName}</Text>;
};

// Reusable tab bar options
const getTabScreenOptions = (theme) => ({ route }) => ({
    tabBarIcon: ({ focused, color, size }) => getTabBarIcon(route, focused, color, size),
    tabBarActiveTintColor: theme.colors.primary,
    tabBarInactiveTintColor: 'gray',
    headerShown: false,
    tabBarStyle: {
        backgroundColor: theme.colors.card,
        borderTopColor: theme.colors.border,
    }
});

// Listener to reset BikesList stack
const getBikesListListeners = () => ({ navigation }) => ({
    tabPress: (e) => {
        const state = navigation.getState();
        if (state) {
            const currentTabRoute = state.routes[state.index];
            if (currentTabRoute.name === 'BikesList') {
                // Reset if we are NOT on the BikesCatalog screen
                const nestedState = currentTabRoute.state;
                if (nestedState && nestedState.routes && nestedState.routes.length > 0) {
                    // Safety check: index might be missing in some deep link partial states
                    const routeIndex = nestedState.index ?? (nestedState.routes.length - 1);
                    const currentRoute = nestedState.routes[routeIndex];

                    // If we are not on BikesCatalog (e.g., BikeDetails), reset.
                    if (currentRoute && currentRoute.name !== 'BikesCatalog') {
                        e.preventDefault(); // Prevent default behavior (popToTop which might toggle)

                        // Hard reset the stack by replacing the Tab route state
                        navigation.dispatch({
                            ...CommonActions.reset({
                                index: state.index,
                                routes: state.routes.map(r =>
                                    r.name === 'BikesList'
                                        ? { name: 'BikesList', state: { index: 0, routes: [{ name: 'BikesCatalog' }] } }
                                        : r
                                ),
                            }),
                        });
                    }
                }
            }
        }
    },
});

const PublicTabs = () => {
    const { theme } = useContext(ThemeContext);
    const { t } = useContext(LanguageContext);

    return (
        <Tab.Navigator screenOptions={getTabScreenOptions(theme)}>
            <Tab.Screen
                name="Home"
                component={PublicHomeStackNavigator}
                options={{ title: t('home') }}
            />
            <Tab.Screen
                name="BikesList"
                component={BikesListStackNavigator}
                options={{ title: t('browse_bikes') }}
                listeners={getBikesListListeners()}
            />
        </Tab.Navigator>
    );
};

const MainTabs = () => {
    const { theme } = useContext(ThemeContext);
    const { t } = useContext(LanguageContext);

    return (
        <Tab.Navigator screenOptions={getTabScreenOptions(theme)}>
            <Tab.Screen name="Home" component={HomeStackNavigator} options={{ title: t('home'), tabBarIcon: ({ color, size }) => <Text style={{ fontSize: size, color: color }}>📷</Text> }} />
            <Tab.Screen
                name="BikesList"
                component={BikesListStackNavigator}
                options={{ title: t('browse_bikes') }}
                listeners={getBikesListListeners()}
            />
            <Tab.Screen name="Configuration" component={ConfigurationScreen} options={{
                title: t('settings'),
                headerShown: true,
                headerTitleStyle: { color: theme.colors.text },
                headerTintColor: theme.colors.primary,
                headerStyle: {
                    backgroundColor: theme.colors.card,
                    borderBottomWidth: 1,
                    borderBottomColor: theme.colors.border
                }
            }} />
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

    const currentLinking = userToken == null ? publicLinking : mainLinking;

    return (
        <NavigationContainer
            linking={{
                ...currentLinking,
                fallback: <ActivityIndicator color={theme.colors.primary} size="large" />,
            }}
            documentTitle={{
                formatter: (options, route) =>
                    `RottenBikes: Rate City Bikes${options?.title ? ` - ${options.title}` : ''}`,
            }}
            theme={navTheme}
        >
            <Stack.Navigator
                initialRouteName={userToken == null ? "Public" : "Main"}
                screenOptions={{
                    headerTitleStyle: { color: theme.colors.text },
                    headerTintColor: theme.colors.primary,
                    headerStyle: {
                        backgroundColor: theme.colors.card,
                    }
                }}
            >
                {userToken == null ? (
                    // Public Stack
                    <>
                        <Stack.Screen name="Public" component={PublicTabs} options={{ headerShown: false }} />
                    </>
                ) : (
                    // App Stack
                    <>
                        <Stack.Screen name="Main" component={MainTabs} options={{ headerShown: false }} />
                    </>
                )}
                <Stack.Screen name="ConfirmLogin" component={ConfirmLoginScreen} options={{ title: t('confirm_login_title') }} />
                <Stack.Screen name="Privacy" component={PrivacyScreen} options={{ title: t('privacy_and_terms_title') }} />
            </Stack.Navigator>
        </NavigationContainer>
    );
};

export default AppNavigator;
