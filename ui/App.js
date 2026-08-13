import { AuthProvider } from './src/context/AuthContext';
import { ToastProvider } from './src/context/ToastContext';
import { ThemeProvider } from './src/context/ThemeContext';
import { SessionProvider } from './src/context/SessionContext';
import { LanguageProvider } from './src/context/LanguageContext';
import AppNavigator from './src/navigation/AppNavigator';
import ErrorBoundary from './src/components/ErrorBoundary';

export default function App() {
  return (
    <ErrorBoundary>
      <ThemeProvider>
        <ToastProvider>
          <LanguageProvider>
            <AuthProvider>
              <SessionProvider>
                <AppNavigator />
              </SessionProvider>
            </AuthProvider>
          </LanguageProvider>
        </ToastProvider>
      </ThemeProvider>
    </ErrorBoundary>
  );
}
