import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it } from 'vitest';
import { PrimeReactProvider } from '@primereact/core/config';
import { primeTheme, ThemeMenu } from './ThemeMenu';

describe('ThemeMenu', () => {
  beforeEach(() => {
    window.localStorage.clear();
    document.documentElement.classList.remove('app-dark');
  });

  it('opens from the PrimeReact trigger and applies a persisted dark scheme', () => {
    render(<PrimeReactProvider {...primeTheme}><ThemeMenu /></PrimeReactProvider>);

    fireEvent.click(screen.getByRole('button', { name: 'Tema aplicației' }));
    fireEvent.click(screen.getByRole('button', { name: 'Întunecat' }));

    expect(document.documentElement).toHaveClass('app-dark');
    expect(document.documentElement.style.colorScheme).toBe('dark');
    expect(window.localStorage.getItem('egueducation.theme')).toContain('"scheme":"dark"');
    expect(screen.getByText('Mod activ: întunecat')).toBeInTheDocument();
  });

  it('persists a selected preset, primary palette, and surface palette', () => {
    render(<PrimeReactProvider {...primeTheme}><ThemeMenu /></PrimeReactProvider>);
    fireEvent.click(screen.getByRole('button', { name: 'Tema aplicației' }));
    fireEvent.click(screen.getByRole('button', { name: 'Material' }));
    const rose = screen.getByRole('button', { name: 'Selectează culoare principală Rose' });
    const zinc = screen.getByRole('button', { name: 'Selectează suprafață Zinc' });
    fireEvent.click(rose);
    fireEvent.click(zinc);

    const stored = window.localStorage.getItem('egueducation.theme');
    expect(stored).toContain('"preset":"material"');
    expect(stored).toContain('"primary":"rose"');
    expect(stored).toContain('"surface":"zinc"');
    expect(rose).toHaveAttribute('aria-pressed', 'true');
    expect(zinc).toHaveAttribute('aria-pressed', 'true');
  });
});
