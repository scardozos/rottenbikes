// Validation helpers for forms and user input.

export const USERNAME_REGEX = /^[a-zA-Z0-9.]+$/;
export const EMAIL_REGEX = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
export const NUMERICAL_ID_REGEX = /^\d{4,5}$/;
export const HASH_ID_REGEX = /^[a-zA-Z0-9]+$/;
export const DIGITS_REGEX = /^\d+$/;

export const isValidUsername = (value) => USERNAME_REGEX.test(value);

export const isValidEmail = (value) => EMAIL_REGEX.test(value);

export const isValidNumericalId = (value) =>
    NUMERICAL_ID_REGEX.test(String(value).trim());

// A hash id is valid when empty (optional) or alphanumeric.
export const isValidHashId = (value) => {
    const trimmed = String(value).trim();
    return trimmed === '' || HASH_ID_REGEX.test(trimmed);
};

export const isNumeric = (value) => DIGITS_REGEX.test(value);
