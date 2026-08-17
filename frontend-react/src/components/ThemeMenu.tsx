import { useEffect, useState } from 'react';
import { Button } from '@primereact/ui/button';
import { Popover } from '@primereact/ui/popover';
import Aura from '@primeuix/themes/aura';
import { Palette } from '@primeicons/react';
import {
  applyThemeScheme,
  readThemeScheme,
  resolveDarkScheme,
  type ThemeScheme,
} from '../theme/preferences';

const schemes: Array<{ label: string; value: ThemeScheme }> = [
  { label: 'Sistem', value: 'system' },
  { label: 'Luminos', value: 'light' },
  { label: 'Întunecat', value: 'dark' },
];

export function ThemeMenu() {
  const [scheme, setScheme] = useState<ThemeScheme>(readThemeScheme);
  const [dark, setDark] = useState(() => resolveDarkScheme(scheme));

  useEffect(() => {
    setDark(applyThemeScheme(scheme));
    if (scheme !== 'system') return;

    const media = window.matchMedia('(prefers-color-scheme: dark)');
    const change = () => setDark(applyThemeScheme('system'));
    media.addEventListener('change', change);
    return () => media.removeEventListener('change', change);
  }, [scheme]);

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
        <Palette />
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Positioner side="bottom" align="end" sideOffset={8}>
          <Popover.Popup>
            <Popover.Content>
              <div className="flex min-w-64 flex-col gap-3">
                <div>
                  <strong>Aspect</strong>
                  <div className="text-sm">Tema PrimeReact Aura</div>
                </div>
                <div className="flex flex-wrap gap-2" role="group" aria-label="Mod de culoare">
                  {schemes.map((option) => (
                    <Button
                      key={option.value}
                      size="small"
                      variant={scheme === option.value ? undefined : 'outlined'}
                      aria-pressed={scheme === option.value}
                      onClick={() => setScheme(option.value)}
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
export const primeTheme = { theme: { preset: Aura, options: { darkModeSelector: '.app-dark' } } };
