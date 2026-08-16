import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import { PrimeReactProvider } from '@primereact/core/config';
import { primeTheme } from './ThemeMenu';

vi.mock('../auth/AuthProvider', () => ({
    useAuth: () => ({ user: null, login: vi.fn(), logout: vi.fn(), has: () => false })
}));

import { AppShell } from './AppShell';

describe('AppShell', () => {
    it('renders a right-side off-canvas navigation on mobile', () => {
        render(
            <PrimeReactProvider {...primeTheme}>
                <MemoryRouter><AppShell /></MemoryRouter>
            </PrimeReactProvider>
        );

        expect(screen.getByRole('button', { name: /navigația/i })).toBeInTheDocument();
        expect(screen.getAllByText('eGuEducation')).toHaveLength(2);
        expect(document.getElementById('main-navigation')).toHaveAttribute('data-side', 'right');
        expect(document.getElementById('main-navigation')).toHaveAttribute('data-collapsible-mode', 'offcanvas');
    });
});
