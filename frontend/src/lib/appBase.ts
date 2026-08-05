const rawBase = import.meta.env.BASE_URL || '/';

export const APP_BASE = rawBase;
export const APP_BASENAME = rawBase === '/' ? '' : rawBase.replace(/\/+$/, '');
export const APP_LOGIN_PATH = `${APP_BASE}login`;
