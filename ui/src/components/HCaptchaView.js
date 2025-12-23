import React, { useEffect, useRef } from 'react';
import { View, Platform, StyleSheet } from 'react-native';
import { WebView } from 'react-native-webview';

const HCaptchaView = ({ siteKey, onVerify, onExpired, onError, onCancel }) => {
    const webviewRef = useRef(null);

    if (Platform.OS === 'web') {
        return <WebHCaptcha siteKey={siteKey} onVerify={onVerify} onExpired={onExpired} onError={onError} />;
    }

    // Native implementation using WebView
    const html = `
    <!DOCTYPE html>
    <html>
    <head>
      <meta name="viewport" content="width=device-width, initial-scale=1.0">
      <script src="https://hcaptcha.com/1/api.js" async defer></script>
    </head>
    <body style="display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0;">
      <div id="h-captcha" class="h-captcha" 
           data-sitekey="${siteKey}" 
           data-callback="onSuccess" 
           data-expired-callback="onExpired" 
           data-error-callback="onError"></div>
      <script>
        function onSuccess(token) {
          console.log("hCaptcha Native Success, token received");
          window.ReactNativeWebView.postMessage(JSON.stringify({type: 'success', token: token}));
        }
        function onExpired() {
          console.log("hCaptcha Native Expired");
          window.ReactNativeWebView.postMessage(JSON.stringify({type: 'expired'}));
        }
        function onError() {
          console.log("hCaptcha Native Error");
          window.ReactNativeWebView.postMessage(JSON.stringify({type: 'error'}));
        }
      </script>
    </body>
    </html>
  `;

    return (
        <WebView
            ref={webviewRef}
            originWhitelist={['*']}
            source={{ html: html, baseUrl: 'https://dev.rottenbik.es' }}
            onMessage={(event) => {
                const data = JSON.parse(event.nativeEvent.data);
                if (data.type === 'success') onVerify(data.token);
                else if (data.type === 'expired') onExpired();
                else if (data.type === 'error') onError();
            }}
            style={{ flex: 1 }}
        />
    );
};

// Internal Web implementation
const WebHCaptcha = ({ siteKey, onVerify, onExpired, onError }) => {
    const containerRef = useRef(null);
    const widgetId = useRef(null);

    // Use refs for callbacks to avoid stale closures
    const callbacks = useRef({ onVerify, onExpired, onError });
    useEffect(() => {
        callbacks.current = { onVerify, onExpired, onError };
    }, [onVerify, onExpired, onError]);

    useEffect(() => {
        // 1. Load the script if not already present
        if (!window.hcaptcha) {
            console.log("Loading hCaptcha JS API...");
            const script = document.createElement('script');
            script.src = 'https://hcaptcha.com/1/api.js?render=explicit';
            script.async = true;
            script.defer = true;
            script.onload = renderCaptcha;
            document.head.appendChild(script);
        } else {
            renderCaptcha();
        }

        function renderCaptcha() {
            if (containerRef.current && window.hcaptcha && widgetId.current === null) {
                console.log("Rendering hCaptcha widget (Web environment)");
                widgetId.current = window.hcaptcha.render(containerRef.current, {
                    sitekey: siteKey,
                    callback: (token) => {
                        console.log("hCaptcha verified successfully on Web");
                        callbacks.current.onVerify(token);
                    },
                    'expired-callback': () => {
                        console.log("hCaptcha expired on Web");
                        callbacks.current.onExpired();
                    },
                    'error-callback': () => {
                        console.log("hCaptcha error on Web");
                        callbacks.current.onError();
                    },
                });
            }
        }

        return () => {
            // Cleanup widget if it exists would go here, but usually not needed for a modal
        };
    }, [siteKey]);

    return <div ref={containerRef} style={styles.webContainer} />;
};

const styles = StyleSheet.create({
    webContainer: {
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        minHeight: 100, // Reduced height for inline rendering
        padding: 20,
    }
});

export default HCaptchaView;
