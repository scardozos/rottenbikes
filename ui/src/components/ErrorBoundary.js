import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { ThemeContext } from '../context/ThemeContext';
import Button from './Button';

class ErrorBoundary extends React.Component {
    constructor(props) {
        super(props);
        this.state = { hasError: false, error: null };
    }

    static getDerivedStateFromError(error) {
        return { hasError: true, error };
    }

    componentDidCatch(error, errorInfo) {
        console.error("ErrorBoundary caught an error", error, errorInfo);
    }

    render() {
        const theme = this.context?.theme;
        const backgroundColor = theme?.colors?.background || '#FFFFFF';
        const textColor = theme?.colors?.text || '#000000';
        const errorColor = theme?.colors?.danger || theme?.colors?.error || '#EF4444';

        if (this.state.hasError) {
            return (
                <View style={[styles.container, { backgroundColor }]}>
                    <Text style={[styles.title, { color: textColor }]}>Oops! Something went wrong.</Text>
                    <Text style={[styles.subtitle, { color: errorColor }]}>{this.state.error?.toString()}</Text>
                    <Button title="Restart App" onPress={() => this.setState({ hasError: false })} />
                </View>
            );
        }

        return this.props.children;
    }
}

const styles = StyleSheet.create({
    container: {
        flex: 1,
        justifyContent: 'center',
        alignItems: 'center',
        padding: 20,
    },
    title: {
        fontSize: 24,
        fontWeight: 'bold',
        marginBottom: 10,
    },
    subtitle: {
        fontSize: 16,
        marginBottom: 20,
        textAlign: 'center',
    }
});

ErrorBoundary.contextType = ThemeContext;

export default ErrorBoundary;
