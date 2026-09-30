// Login codes are the 6 digits emailed next to the magic link. People type
// or paste them with spaces or dashes, so only the digits are kept.
export const LOGIN_CODE_LENGTH = 6;

export const normalizeLoginCode = (text) =>
    String(text || '').replace(/[^0-9]/g, '').slice(0, LOGIN_CODE_LENGTH);

export const isCompleteLoginCode = (code) =>
    typeof code === 'string' && /^[0-9]{6}$/.test(code);
