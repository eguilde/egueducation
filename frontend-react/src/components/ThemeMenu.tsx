import { useEffect, useState } from 'react';
import { Button } from '@primereact/ui/button';
import { Popover } from '@primereact/ui/popover';
import Aura from '@primeuix/themes/aura';
import { Palette } from '@primeicons/react';

type Scheme = 'light' | 'dark' | 'system';
const schemes = [{ label: 'Sistem', value: 'system' }, { label: 'Luminos', value: 'light' }, { label: 'Întunecat', value: 'dark' }];
export function ThemeMenu() {
  const [scheme, setScheme] = useState<Scheme>(() => window.localStorage?.getItem('egueducation.scheme') as Scheme || 'system');
  const dark = scheme === 'dark' || (scheme === 'system' && window.matchMedia?.('(prefers-color-scheme: dark)').matches);
  useEffect(() => { document.documentElement.classList.toggle('app-dark', dark); window.localStorage?.setItem('egueducation.scheme', scheme); }, [dark, scheme]);
  return <Popover.Root><Popover.Trigger asChild><Button aria-label="Preferințe de temă" variant="text" rounded iconOnly><Palette /></Button></Popover.Trigger><Popover.Portal><Popover.Positioner><Popover.Popup><Popover.Content><div className="flex flex-col gap-3 min-w-64"><span>Temă PrimeReact: Aura</span><div className="flex gap-2">{schemes.map((option) => <Button key={option.value} size="small" variant={scheme === option.value ? undefined : 'outlined'} onClick={() => setScheme(option.value as Scheme)}>{option.label}</Button>)}</div><small>Aspect: {dark ? 'întunecat' : 'luminos'}</small></div></Popover.Content></Popover.Popup></Popover.Positioner></Popover.Portal></Popover.Root>;
}
export const primeTheme = { theme: { preset: Aura, options: { darkModeSelector: '.app-dark' } } };
