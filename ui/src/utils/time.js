export const getRelativeTime = (dateString, t) => {
    if (!dateString) return '';
    const now = new Date();
    const then = new Date(dateString);
    const seconds = Math.floor((now - then) / 1000);

    if (seconds < 60) return t('just_now');
    const minutes = Math.floor(seconds / 60);
    if (minutes < 60) return t('m_ago', { minutes });
    const hours = Math.floor(minutes / 60);
    if (hours < 24) return t('h_ago', { hours });
    const days = Math.floor(hours / 24);
    if (days < 7) return t('d_ago', { days });
    const weeks = Math.floor(days / 7);
    if (weeks < 4) return t('w_ago', { weeks });
    const months = Math.floor(days / 30);
    if (months < 12) return t('mo_ago', { months });
    const years = Math.floor(days / 365);
    return t('y_ago', { years });
};
