import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { PrimeReactProvider } from '@primereact/core/config';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { primeTheme } from '../components/ThemeMenu';

const { loginMock } = vi.hoisted(() => ({ loginMock: vi.fn() }));

vi.mock('../auth/AuthProvider', () => ({
    useAuth: () => ({ login: loginMock }),
}));

import { CanaryActivationPage } from './Pages';

const renderPage = () => render(
    <PrimeReactProvider {...primeTheme}>
        <MemoryRouter><CanaryActivationPage /></MemoryRouter>
    </PrimeReactProvider>
);

describe('CanaryActivationPage', () => {
    beforeEach(() => {
        loginMock.mockReset();
        vi.restoreAllMocks();
    });

    it('activates the HttpOnly canary session before starting OIDC login', async () => {
        const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(null, { status: 204 }));
        renderPage();

        fireEvent.change(screen.getByLabelText('Cheie temporară de activare'), { target: { value: 'temporary-secret' } });
        fireEvent.click(screen.getByRole('button', { name: 'Activează și autentifică' }));

        await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/api/oidc/e2e-canary/session', expect.objectContaining({
            method: 'POST',
            credentials: 'include',
            cache: 'no-store',
            headers: { Authorization: 'Bearer temporary-secret' },
        })));
        await waitFor(() => expect(loginMock).toHaveBeenCalledOnce());
        expect(screen.getByLabelText('Cheie temporară de activare')).toHaveValue('');
    });

    it('clears a rejected key and does not start OIDC login', async () => {
        vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(null, { status: 401 }));
        renderPage();

        fireEvent.change(screen.getByLabelText('Cheie temporară de activare'), { target: { value: 'wrong-secret' } });
        fireEvent.click(screen.getByRole('button', { name: 'Activează și autentifică' }));

        await screen.findByText('Activarea sesiunii de test a fost refuzată.');
        expect(loginMock).not.toHaveBeenCalled();
        expect(screen.getByLabelText('Cheie temporară de activare')).toHaveValue('');
    });
});
