import { createContext, useContext, useEffect, useMemo, useState, type CSSProperties, type PropsWithChildren } from 'react';
import { PrimeReactProvider } from '@primereact/core/config';
import { Button } from '@primereact/ui/button';
import { Popover } from '@primereact/ui/popover';
import Aura from '@primeuix/themes/aura';
import Lara from '@primeuix/themes/lara';
import Nora from '@primeuix/themes/nora';
import Material from '@primeuix/themes/material';
import { definePreset, palette, type Preset } from '@primeuix/themes';
import { Check, Moon, Sun } from '@primeicons/react';
import {
  applyThemePreferences,
  defaultThemePreferences,
  readThemePreferences,
  resolveDarkScheme,
  type ThemePalette,
  type ThemePreferences,
  type ThemeScheme,
  type ThemeSurface,
} from '../theme/preferences';

const schemes: Array<{ label: string; value: ThemeScheme }> = [
  { label: 'Sistem', value: 'system' },
  { label: 'Luminos', value: 'light' },
  { label: 'Întunecat', value: 'dark' },
];

const presets = { aura: Aura, lara: Lara, nora: Nora, material: Material } satisfies Record<ThemePreferences['preset'], Preset>;
const primaryPalettes: ThemePalette[] = ['noir', 'emerald', 'green', 'lime', 'orange', 'amber', 'yellow', 'teal', 'cyan', 'sky', 'blue', 'indigo', 'violet', 'purple', 'fuchsia', 'pink', 'rose'];
// PrimeUIX 3 exposes exactly these neutral primitive palettes in every bundled preset.
// Apollo-only names (soho, viva, ocean, taupe, mauve, mist, olive) are not fabricated.
const surfacePalettes: ThemeSurface[] = ['slate', 'gray', 'zinc', 'neutral', 'stone'];
const primaryTokens = (primary: ThemePalette, surface: ThemeSurface) =>
  palette(`{${primary === 'noir' ? surface : primary}}`);

type PrimitivePalette = Record<number, string>;
type PresetWithPrimitives = Preset & { primitive: Record<string, PrimitivePalette> };

function primitiveSwatch(preset: ThemePreferences['preset'], color: string, surface: ThemeSurface): string {
  const paletteName = color === 'noir' ? surface : color;
  return (presets[preset] as PresetWithPrimitives).primitive[paletteName][500];
}

function configurePreset(preferences: ThemePreferences): Preset {
  return definePreset(presets[preferences.preset], {
    semantic: {
      primary: primaryTokens(preferences.primary, preferences.surface),
      surface: palette(`{${preferences.surface}}`),
    },
  });
}

interface ThemeContextValue {
  preferences: ThemePreferences;
  setPreferences: (update: Partial<ThemePreferences>) => void;
}

const ThemeContext = createContext<ThemeContextValue | undefined>(undefined);

export function AppThemeProvider({ children }: PropsWithChildren) {
  const [preferences, setPreferencesState] = useState<ThemePreferences>(readThemePreferences);
  const darkModeSelector = '.app-dark';
  const configuredPreset = useMemo(() => configurePreset(preferences), [preferences]);

  useEffect(() => {
    applyThemePreferences(preferences);
    if (preferences.scheme !== 'system' || typeof window === 'undefined') return;
    const media = window.matchMedia('(prefers-color-scheme: dark)');
    const change = () => applyThemePreferences(preferences);
    media.addEventListener('change', change);
    return () => media.removeEventListener('change', change);
  }, [preferences]);

  const value = useMemo<ThemeContextValue>(() => ({
    preferences,
    setPreferences: (update) => setPreferencesState((current) => ({ ...current, ...update })),
  }), [preferences]);

  return (
    <ThemeContext.Provider value={value}>
      <PrimeReactProvider theme={{ preset: configuredPreset, options: { darkModeSelector } }} license={import.meta.env.VITE_PRIMEUI_LICENSE}>
        {children}
      </PrimeReactProvider>
    </ThemeContext.Provider>
  );
}

export function useThemePreferences() {
  const value = useContext(ThemeContext);
  const [fallbackPreferences, setFallbackPreferences] = useState<ThemePreferences>(readThemePreferences);
  return value ?? {
    preferences: fallbackPreferences,
    setPreferences: (update: Partial<ThemePreferences>) => setFallbackPreferences((current) => ({ ...current, ...update })),
  };
}

export function ThemeMenu() {
  const { preferences, setPreferences } = useThemePreferences();
  const dark = resolveDarkScheme(preferences.scheme);

  // Keep the menu usable in isolated component hosts as well as the app provider.
  useEffect(() => { applyThemePreferences(preferences); }, [preferences]);

  return (
    <Popover.Root>
      <Popover.Trigger
        as={Button}
        aria-label="Tema aplicației"
        title="Tema aplicației"
        variant="text"
        rounded
        iconOnly
      >
        {dark ? <Moon aria-hidden="true" /> : <Sun aria-hidden="true" />}
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Positioner side="bottom" align="end" sideOffset={8}>
          <Popover.Popup className="app-theme-popover">
            <Popover.Content>
              <div className="flex min-w-64 flex-col gap-3">
                <div>
                  <strong>Temă</strong>
                  <div className="text-sm">Preferințele se păstrează pentru acest dispozitiv.</div>
                </div>
                <ThemeChoice label="Preset PrimeReact" value={preferences.preset} options={Object.keys(presets)} onChange={(preset) => setPreferences({ preset: preset as ThemePreferences['preset'] })} />
                <SwatchChoice label="Culoare principală" preset={preferences.preset} surface={preferences.surface} value={preferences.primary} options={primaryPalettes} onChange={(primary) => setPreferences({ primary: primary as ThemePalette })} />
                <SwatchChoice label="Suprafață" preset={preferences.preset} surface={preferences.surface} value={preferences.surface} options={surfacePalettes} onChange={(surface) => setPreferences({ surface: surface as ThemeSurface })} />
                <div className="flex flex-wrap gap-2" role="group" aria-label="Mod de culoare">
                  {schemes.map((option) => (
                    <Button
                      key={option.value}
                      size="small"
                      variant={preferences.scheme === option.value ? undefined : 'outlined'}
                      aria-pressed={preferences.scheme === option.value}
                      onClick={() => setPreferences({ scheme: option.value })}
                    >
                      {option.label}
                    </Button>
                  ))}
                </div>
                <small aria-live="polite">Mod activ: {dark ? 'întunecat' : 'luminos'}</small>
              </div>
            </Popover.Content>
          </Popover.Popup>
        </Popover.Positioner>
      </Popover.Portal>
    </Popover.Root>
  );
}

function ThemeChoice({ label, value, options, onChange }: { label: string; value: string; options: readonly string[]; onChange: (value: string) => void }) {
  return <div className="flex flex-col gap-1"><span className="text-sm">{label}</span><div className="flex flex-wrap gap-1" role="group" aria-label={label}>{options.map((option) => <Button key={option} size="small" variant={value === option ? undefined : 'outlined'} aria-pressed={value === option} onClick={() => onChange(option)}>{option[0].toUpperCase() + option.slice(1)}</Button>)}</div></div>;
}

function SwatchChoice({ label, preset, surface, value, options, onChange }: { label: string; preset: ThemePreferences['preset']; surface: ThemeSurface; value: string; options: readonly string[]; onChange: (value: string) => void }) {
  return <div className="flex flex-col gap-1"><span className="text-sm">{label}</span><div className="flex flex-wrap gap-2" role="group" aria-label={label}>{options.map((option) => {
    const display = option[0].toUpperCase() + option.slice(1);
    const swatch = primitiveSwatch(preset, option, surface);
    return <Button key={option} className="app-theme-swatch" style={{ '--theme-swatch': swatch } as CSSProperties} aria-label={`Selectează ${label.toLowerCase()} ${display}`} title={display} aria-pressed={value === option} onClick={() => onChange(option)}>{value === option ? <Check aria-hidden="true" /> : null}</Button>;
  })}</div></div>;
}

// Test and isolated component consumers can use the default PrimeReact configuration.
export const primeTheme = { theme: { preset: configurePreset(defaultThemePreferences), options: { darkModeSelector: '.app-dark' } } };
