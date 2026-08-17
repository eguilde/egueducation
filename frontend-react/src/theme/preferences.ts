export type ThemeScheme = 'light' | 'dark' | 'system';
export type ThemePresetName = 'aura' | 'lara' | 'nora' | 'material';
export type ThemePalette = 'noir' | 'emerald' | 'green' | 'lime' | 'orange' | 'amber' | 'yellow' | 'teal' | 'cyan' | 'sky' | 'blue' | 'indigo' | 'violet' | 'purple' | 'fuchsia' | 'pink' | 'rose';
export type ThemeSurface = 'slate' | 'gray' | 'zinc' | 'neutral' | 'stone';

export interface ThemePreferences {
  preset: ThemePresetName;
  primary: ThemePalette;
  surface: ThemeSurface;
  scheme: ThemeScheme;
}

export const THEME_STORAGE_KEY = 'egueducation.theme';
const LEGACY_SCHEME_STORAGE_KEY = 'egueducation.scheme';
export const defaultThemePreferences: ThemePreferences = {
  preset: 'aura', primary: 'rose', surface: 'slate', scheme: 'system',
};

const isBrowser = () => typeof window !== 'undefined' && typeof document !== 'undefined';
const isPreset = (value: unknown): value is ThemePresetName =>
  value === 'aura' || value === 'lara' || value === 'nora' || value === 'material';
const isPalette = (value: unknown): value is ThemePalette =>
  ['noir', 'emerald', 'green', 'lime', 'orange', 'amber', 'yellow', 'teal', 'cyan', 'sky', 'blue', 'indigo', 'violet', 'purple', 'fuchsia', 'pink', 'rose'].includes(value as string);
const isSurface = (value: unknown): value is ThemeSurface =>
  value === 'slate' || value === 'gray' || value === 'zinc' || value === 'neutral' || value === 'stone';
const isScheme = (value: unknown): value is ThemeScheme => value === 'light' || value === 'dark' || value === 'system';

export function readThemePreferences(): ThemePreferences {
  if (!isBrowser()) return defaultThemePreferences;
  try {
    const parsed: unknown = JSON.parse(window.localStorage.getItem(THEME_STORAGE_KEY) ?? 'null');
    if (parsed && typeof parsed === 'object') {
      const value = parsed as Partial<ThemePreferences>;
      return {
        preset: isPreset(value.preset) ? value.preset : defaultThemePreferences.preset,
        primary: isPalette(value.primary) ? value.primary : defaultThemePreferences.primary,
        surface: isSurface(value.surface) ? value.surface : defaultThemePreferences.surface,
        scheme: isScheme(value.scheme) ? value.scheme : defaultThemePreferences.scheme,
      };
    }
    const legacy = window.localStorage.getItem(LEGACY_SCHEME_STORAGE_KEY);
    return isScheme(legacy) ? { ...defaultThemePreferences, scheme: legacy } : defaultThemePreferences;
  } catch {
    return defaultThemePreferences;
  }
}

export function systemPrefersDark(): boolean {
  return isBrowser() && window.matchMedia('(prefers-color-scheme: dark)').matches;
}

export function resolveDarkScheme(scheme: ThemeScheme): boolean {
  return scheme === 'dark' || (scheme === 'system' && systemPrefersDark());
}

export function persistThemePreferences(preferences: ThemePreferences): void {
  if (!isBrowser()) return;
  try { window.localStorage.setItem(THEME_STORAGE_KEY, JSON.stringify(preferences)); } catch { /* storage may be unavailable */ }
  const secure = window.location.protocol === 'https:' ? '; Secure' : '';
  document.cookie = `${THEME_STORAGE_KEY}=${encodeURIComponent(JSON.stringify(preferences))}; Path=/; Max-Age=31536000; SameSite=Lax${secure}`;
}

export function applyThemePreferences(preferences: ThemePreferences): boolean {
  const dark = resolveDarkScheme(preferences.scheme);
  if (!isBrowser()) return dark;
  document.documentElement.classList.toggle('app-dark', dark);
  document.documentElement.style.colorScheme = dark ? 'dark' : 'light';
  persistThemePreferences(preferences);
  return dark;
}

// Compatibility exports for existing consumers while migration completes.
export const readThemeScheme = () => readThemePreferences().scheme;
export const persistThemeScheme = (scheme: ThemeScheme) => persistThemePreferences({ ...readThemePreferences(), scheme });
export const applyThemeScheme = (scheme: ThemeScheme) => applyThemePreferences({ ...readThemePreferences(), scheme });
