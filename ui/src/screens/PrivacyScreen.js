import React, { useContext } from 'react';
import { View, ScrollView, Text, StyleSheet, Platform, TouchableOpacity, Linking } from 'react-native';
import { ThemeContext } from '../context/ThemeContext';
import { LanguageContext } from '../context/LanguageContext';

const PrivacyScreen = ({ navigation }) => {
    const { theme } = useContext(ThemeContext);
    const { t } = useContext(LanguageContext);
    const styles = createStyles(theme);

    const handleLinkPress = (url) => {
        if (Platform.OS === 'web') {
            window.open(url, '_blank');
        } else {
            Linking.openURL(url);
        }
    };

    return (
        <ScrollView style={styles.container} contentContainerStyle={styles.contentContainer}>
            <Text style={styles.header}>{t('privacy_and_terms_title')}</Text>

            {/* Privacy Policy Section */}
            <View style={styles.section}>
                <Text style={styles.sectionTitle}>🔒 {t('privacy_policy_title')}</Text>
                <Text style={styles.paragraph}>{t('privacy_intro')}</Text>

                <Text style={styles.subTitle}>{t('data_collection_title')}</Text>
                <Text style={styles.paragraph}>
                    {t('data_collection_intro')}
                </Text>
                <View style={styles.bulletList}>
                    <Text style={styles.bulletItem}>• {t('data_collection_user')}</Text>
                    <Text style={styles.bulletItem}>• {t('data_collection_email')}</Text>
                </View>

                <Text style={styles.subTitle}>{t('third_party_title')}</Text>

                <Text style={styles.subTitle}>{t('mailtrap_title')}</Text>
                <Text style={styles.paragraph}>
                    {t('mailtrap_text')}
                </Text>
                <TouchableOpacity onPress={() => handleLinkPress('https://docs.mailtrap.io/account-and-organization/privacy-and-security/gdpr-compliance')}>
                    <Text style={styles.link}>
                        {t('mailtrap_link_text')}
                    </Text>
                </TouchableOpacity>

                <Text style={styles.subTitle}>{t('hcaptcha_title')}</Text>
                <Text style={styles.paragraph}>
                    {t('hcaptcha_text')}
                </Text>
                <TouchableOpacity onPress={() => handleLinkPress('https://www.hcaptcha.com/privacy')}>
                    <Text style={styles.link}>
                        {t('hcaptcha_link_text')}
                    </Text>
                </TouchableOpacity>
            </View>

            {/* Terms & Conditions Section */}
            <View style={styles.section}>
                <Text style={styles.sectionTitle}>📜 {t('terms_conditions_title')}</Text>
                <Text style={styles.subTitle}>{t('abuse_policy_title')}</Text>
                <Text style={styles.paragraph}>
                    {t('abuse_policy_text')}
                </Text>

                <Text style={styles.subTitle}>{t('rate_limits_title')}</Text>
                <Text style={styles.paragraph}>
                    {t('rate_limits_text')}
                </Text>

                <Text style={styles.subTitle}>{t('accuracy_title')}</Text>
                <Text style={styles.paragraph}>
                    {t('accuracy_text')}
                </Text>
            </View>
        </ScrollView>
    );
};

const createStyles = (theme) => StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: theme.colors.background,
    },
    contentContainer: {
        padding: 20,
        paddingBottom: 50,
        maxWidth: 800,
        alignSelf: 'center',
        width: '100%',
    },
    header: {
        fontSize: 32,
        fontWeight: 'bold',
        marginBottom: 30,
        color: theme.colors.text,
        textAlign: 'center',
    },
    section: {
        marginBottom: 30,
        backgroundColor: theme.colors.card,
        borderRadius: 12,
        padding: 20,
        shadowColor: "#000",
        shadowOffset: {
            width: 0,
            height: 2,
        },
        shadowOpacity: 0.1,
        shadowRadius: 3.84,
        elevation: 5,
    },
    sectionTitle: {
        fontSize: 24,
        fontWeight: 'bold',
        marginBottom: 20,
        color: theme.colors.text,
        borderBottomWidth: 1,
        borderBottomColor: theme.colors.border,
        paddingBottom: 10,
    },
    subTitle: {
        fontSize: 18,
        fontWeight: '600',
        marginTop: 15,
        marginBottom: 10,
        color: theme.colors.text,
    },
    paragraph: {
        fontSize: 16,
        lineHeight: 24,
        color: theme.colors.text,
        marginBottom: 10,
    },
    bulletList: {
        marginBottom: 10,
        paddingLeft: 10,
    },
    bulletItem: {
        fontSize: 16,
        lineHeight: 24,
        color: theme.colors.text,
        marginBottom: 5,
    },
    link: {
        fontSize: 16,
        color: theme.colors.primary,
        textDecorationLine: 'underline',
        marginTop: 5,
    }
});

export default PrivacyScreen;
