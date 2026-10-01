import React, { useContext } from 'react';
import { View, ScrollView, Text, StyleSheet, Platform, TouchableOpacity, Linking } from 'react-native';
import { ThemeContext } from '../context/ThemeContext';
import { LanguageContext } from '../context/LanguageContext';

const CONTACT_EMAIL = 'privacy@rottenbik.es';

const PrivacyScreen = ({ navigation }) => {
    const { theme } = useContext(ThemeContext);
    const { t } = useContext(LanguageContext);
    const styles = React.useMemo(() => createStyles(theme), [theme]);

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
                <Text style={styles.updated}>{t('privacy_last_updated')}</Text>
                <Text style={styles.paragraph}>{t('privacy_intro')}</Text>

                <Text style={styles.subTitle}>{t('controller_title')}</Text>
                <Text style={styles.paragraph}>{t('controller_text')}</Text>
                <TouchableOpacity onPress={() => handleLinkPress(`mailto:${CONTACT_EMAIL}`)}>
                    <Text style={styles.link}>{CONTACT_EMAIL}</Text>
                </TouchableOpacity>

                <Text style={styles.subTitle}>{t('data_collection_title')}</Text>
                <View style={styles.bulletList}>
                    <Text style={styles.bulletItem}>• {t('data_collection_account')}</Text>
                    <Text style={styles.bulletItem}>• {t('data_collection_content')}</Text>
                    <Text style={styles.bulletItem}>• {t('data_collection_scans')}</Text>
                    <Text style={styles.bulletItem}>• {t('data_collection_session')}</Text>
                    <Text style={styles.bulletItem}>• {t('data_collection_moderation')}</Text>
                </View>
                <Text style={styles.paragraph}>{t('data_collection_not')}</Text>

                <Text style={styles.subTitle}>{t('legal_basis_title')}</Text>
                <Text style={styles.paragraph}>{t('legal_basis_text')}</Text>

                <Text style={styles.subTitle}>{t('storage_title')}</Text>
                <Text style={styles.paragraph}>{t('storage_text')}</Text>

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

                <Text style={styles.subTitle}>{t('transfers_title')}</Text>
                <Text style={styles.paragraph}>{t('transfers_text')}</Text>

                <Text style={styles.subTitle}>{t('retention_title')}</Text>
                <View style={styles.bulletList}>
                    <Text style={styles.bulletItem}>• {t('retention_account')}</Text>
                    <Text style={styles.bulletItem}>• {t('retention_login')}</Text>
                    <Text style={styles.bulletItem}>• {t('retention_deletion')}</Text>
                    <Text style={styles.bulletItem}>• {t('retention_moderation')}</Text>
                </View>

                <Text style={styles.subTitle}>{t('rights_title')}</Text>
                <Text style={styles.paragraph}>{t('rights_text')}</Text>
                <TouchableOpacity onPress={() => handleLinkPress('https://www.aepd.es')}>
                    <Text style={styles.link}>{t('aepd_link_text')}</Text>
                </TouchableOpacity>

                <Text style={styles.subTitle}>{t('age_title')}</Text>
                <Text style={styles.paragraph}>{t('age_text')}</Text>
            </View>

            {/* Legal Notice & Terms Section */}
            <View style={styles.section}>
                <Text style={styles.sectionTitle}>📜 {t('terms_conditions_title')}</Text>

                <Text style={styles.subTitle}>{t('legal_owner_title')}</Text>
                <Text style={styles.paragraph}>{t('legal_owner_text')}</Text>
                <TouchableOpacity onPress={() => handleLinkPress(`mailto:${CONTACT_EMAIL}`)}>
                    <Text style={styles.link}>{CONTACT_EMAIL}</Text>
                </TouchableOpacity>

                <Text style={styles.subTitle}>{t('affiliation_title')}</Text>
                <Text style={styles.paragraph}>
                    {t('affiliation_text')}
                </Text>

                <Text style={styles.subTitle}>{t('user_content_title')}</Text>
                <Text style={styles.paragraph}>
                    {t('user_content_text')}
                </Text>

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
    updated: {
        fontSize: 14,
        color: theme.colors.subtext,
        marginBottom: 10,
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
