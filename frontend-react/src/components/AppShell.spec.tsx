import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import { PrimeReactProvider } from '@primereact/core/config';
import { primeTheme } from './ThemeMenu';

vi.mock('../auth/AuthProvider', () => ({
    useAuth: () => ({ user: null, session: null, login: vi.fn(), logout: vi.fn(), has: () => false })
}));

import { AppShell } from './AppShell';

describe('AppShell', () => {
    beforeEach(() => {
        vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({
            institutionName: 'Școala Balotești',
        }), { status: 200, headers: { 'Content-Type': 'application/json' } }));
    });

    it('renders a left-side off-canvas navigation controlled by the bars button on mobile', async () => {
        render(
            <PrimeReactProvider {...primeTheme}>
                <MemoryRouter><AppShell /></MemoryRouter>
            </PrimeReactProvider>
        );

        fireEvent.click(screen.getByRole('button', { name: 'Deschide navigația' }));
        expect(screen.getByRole('button', { name: 'Închide navigația' })).toBeInTheDocument();
        expect(document.getElementById('main-navigation')).toHaveAttribute('role', 'complementary');
        fireEvent.click(screen.getByRole('button', { name: 'Închide navigația' }));
        await waitFor(() => expect(screen.queryByRole('button', { name: 'Închide navigația' })).not.toBeInTheDocument());
        await waitFor(() => expect(document.querySelector('.app-title')).toHaveTextContent('Școala Balotești'));
    });

    it('keeps the left navigation open and omits the bars button on desktop', () => {
        Object.defineProperty(window, 'matchMedia', {
            configurable: true,
            value: (query: string) => ({
                matches: query === '(min-width: 768px)', media: query, onchange: null,
                addEventListener: () => undefined, removeEventListener: () => undefined,
                addListener: () => undefined, removeListener: () => undefined, dispatchEvent: () => false,
            }),
        });
        render(
            <PrimeReactProvider {...primeTheme}>
                <MemoryRouter><AppShell /></MemoryRouter>
            </PrimeReactProvider>
        );

        expect(document.getElementById('main-navigation')).toHaveAttribute('aria-label', 'Navigație principală');
        expect(screen.queryByRole('button', { name: /navigația/i })).not.toBeInTheDocument();
    });
});
