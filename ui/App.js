import { AuthProvider } from './src/context/AuthContext';
import { ToastProvider } from './src/context/ToastContext';
import { ThemeProvider } from './src/context/ThemeContext';
import AppNavigator from './src/navigation/AppNavigator';

export default function App() {
  return (
    <ThemeProvider>
      <ToastProvider>
        <AuthProvider>
          <AppNavigator />
        </AuthProvider>
      </ToastProvider>
    </ThemeProvider>
  );
}
