import React, { useContext, useState } from 'react';
import { View, Text, TouchableOpacity, StyleSheet, Modal, FlatList } from 'react-native';
import { ThemeContext } from '../context/ThemeContext';
import { LanguageContext } from '../context/LanguageContext';

const SORT_OPTIONS = [
    { value: 'recent', labelKey: 'sort_recent' },
    { value: 'rating', labelKey: 'sort_rating' },
    { value: 'most_reviewed', labelKey: 'sort_most_reviewed' },
];

const SortDropdown = ({ selectedSort, onSortChange }) => {
    const { theme } = useContext(ThemeContext);
    const { t } = useContext(LanguageContext);
    const [modalVisible, setModalVisible] = useState(false);

    const styles = createStyles(theme);

    const selectedOption = SORT_OPTIONS.find(o => o.value === selectedSort) || SORT_OPTIONS[0];

    return (
        <View style={styles.container}>
            <TouchableOpacity style={styles.button} onPress={() => setModalVisible(true)}>
                <Text style={styles.buttonText}>{t('sort_by_label')}: {t(selectedOption.labelKey) || selectedOption.labelKey}</Text>
                <Text style={styles.arrow}>▼</Text>
            </TouchableOpacity>

            <Modal visible={modalVisible} transparent={true} animationType="fade">
                <TouchableOpacity style={styles.modalOverlay} activeOpacity={1} onPress={() => setModalVisible(false)}>
                    <View style={styles.modalContent}>
                        <Text style={styles.modalTitle}>{t('sort_options_title') || 'Sort Options'}</Text>
                        <FlatList
                            data={SORT_OPTIONS}
                            keyExtractor={item => item.value}
                            renderItem={({ item }) => (
                                <TouchableOpacity
                                    style={[styles.optionItem, selectedSort === item.value && styles.selectedOptionItem]}
                                    onPress={() => {
                                        onSortChange(item.value);
                                        setModalVisible(false);
                                    }}
                                >
                                    <Text style={[styles.optionText, selectedSort === item.value && styles.selectedOptionText]}>
                                        {t(item.labelKey) || item.labelKey}
                                    </Text>
                                    {selectedSort === item.value && <Text style={styles.selectedCheck}>✓</Text>}
                                </TouchableOpacity>
                            )}
                        />
                    </View>
                </TouchableOpacity>
            </Modal>
        </View>
    );
};

const createStyles = (theme) => StyleSheet.create({
    container: {
        marginBottom: 10,
        zIndex: 10,
    },
    button: {
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'space-between',
        backgroundColor: theme.colors.card,
        paddingHorizontal: 15,
        paddingVertical: 10,
        borderRadius: 8,
        borderWidth: 1,
        borderColor: theme.colors.border,
    },
    buttonText: {
        fontSize: 16,
        color: theme.colors.text,
    },
    arrow: {
        fontSize: 14,
        color: theme.colors.subtext,
    },
    modalOverlay: {
        flex: 1,
        backgroundColor: 'rgba(0,0,0,0.5)',
        justifyContent: 'center',
        alignItems: 'center',
    },
    modalContent: {
        width: '80%',
        backgroundColor: theme.colors.card,
        borderRadius: 12,
        padding: 20,
        elevation: 5,
        shadowColor: '#000',
        shadowOffset: { width: 0, height: 2 },
        shadowOpacity: 0.25,
        shadowRadius: 3.84,
    },
    modalTitle: {
        fontSize: 18,
        fontWeight: 'bold',
        color: theme.colors.text,
        marginBottom: 15,
    },
    optionItem: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        paddingVertical: 12,
        borderBottomWidth: 1,
        borderBottomColor: theme.colors.border,
    },
    selectedOptionItem: {
        backgroundColor: theme.colors.inputBackground,
    },
    optionText: {
        fontSize: 16,
        color: theme.colors.text,
    },
    selectedOptionText: {
        color: theme.colors.primary,
        fontWeight: 'bold',
    },
    selectedCheck: {
        fontSize: 16,
        color: theme.colors.primary,
        fontWeight: 'bold',
    }
});

export default SortDropdown;
