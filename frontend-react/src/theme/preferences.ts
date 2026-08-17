export type ThemeScheme = 'light' | 'dark' | 'system';

export const THEME_STORAGE_KEY = 'egueducation.scheme';

export function readThemeScheme(): ThemeScheme {
  const stored = window.localStorage?.getItem(THEME_STORAGE_KEY);
  return stored === 'light' || stored === 'dark' || stored === 'system' ? stored : 'system';
}

export function systemPrefersDark(): boolean {
  return Boolean(window.matchMedia?.('(prefers-color-scheme: dark)').matches);
}

export function resolveDarkScheme(scheme: ThemeScheme): boolean {
  return scheme === 'dark' || (scheme === 'system' && systemPrefersDark());
}

export function persistThemeScheme(scheme: ThemeScheme): void {
  window.localStorage?.setItem(THEME_STORAGE_KEY, scheme);
  const secure = window.location.protocol === 'https:' ? '; Secure' : '';
  document.cookie = `${THEME_STORAGE_KEY}=${scheme}; Path=/; Max-Age=31536000; SameSite=Lax${secure}`;
}

export function applyThemeScheme(scheme: ThemeScheme): boolean {
  const dark = resolveDarkScheme(scheme);
  document.documentElement.classList.toggle('app-dark', dark);
  document.documentElement.style.colorScheme = dark ? 'dark' : 'light';
  persistThemeScheme(scheme);
  return dark;
}
