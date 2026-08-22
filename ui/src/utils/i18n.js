// i18n interpolation helpers.

// Replaces {param} placeholders in text with values from params.
export const interpolate = (text, params = {}) => {
    let result = text;
    Object.keys(params).forEach((param) => {
        result = result.replace(`{${param}}`, params[param]);
    });
    return result;
};

// Resolves a translation key against a dictionary, interpolating params.
// Falls back to the key itself when no translation is found.
export const translate = (dictionary, key, params = {}) => {
    const text = dictionary[key] || key;
    return interpolate(text, params);
};
