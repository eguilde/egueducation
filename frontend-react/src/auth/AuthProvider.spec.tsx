import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

vi.mock('./oidc-client', () => ({
    beginAuthorization: vi.fn(),
    beginLogout: vi.fn(),
    completeAuthorization: vi.fn(),
    completeLogout: vi.fn(),
    consumeReturnTo: vi.fn(() => '/'),
    refreshWithCookie: vi.fn(async () => ({ accessToken: 'access', expiresAt: 9999999999 }))
}));

import { AuthProvider, useAuth } from './AuthProvider';
import { beginLogout, refreshWithCookie } from './oidc-client';

const session = {
    user: {
        id: '3f2b2335-a44d-4fa9-ac47-37667192a2d1',
        sub: 'subject-1',
        name: 'Administrator',
        email: 'admin@example.test',
        email_verified: true,
        phone_number: '',
        phone_number_verified: false,
        preferred_otp_channel: 'sms',
        locale: 'ro',
        roles: ['super_admin']
    },
    institution_id: 'cda3c78c-96f6-45ac-89ce-099428c7d448',
    institution_name: 'Școala de test',
    permissions: ['registratura.read'],
    modules: [{ code: 'registratura', active: true }],
    authentication: ['sms'],
    gdpr_capabilities: []
};

function SessionProbe() {
    const auth = useAuth();
    return <div>{`${auth.ready ? auth.user?.name : 'loading'}:${String(auth.has('registratura.read'))}:${String(auth.has('admin.users.manage'))}`}</div>;
}

function LogoutProbe() {
    const auth = useAuth();
    return <button type="button" disabled={!auth.user} onClick={() => void auth.logout()}>Logout test</button>;
}

describe('AuthProvider', () => {
    afterEach(() => vi.restoreAllMocks());

    it('accepts the backend nested SessionContext and exposes effective permissions', async () => {
        vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify(session), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
        }));

        render(<AuthProvider><SessionProbe /></AuthProvider>);

        await waitFor(() => expect(screen.getByText('Administrator:true:false')).toBeInTheDocument());
    });

    it('revokes logout below the path-scoped OIDC refresh-cookie route', async () => {
        const fetchMock = vi.spyOn(globalThis, 'fetch')
            .mockResolvedValueOnce(new Response(JSON.stringify(session), {
                status: 200,
                headers: { 'Content-Type': 'application/json' }
            }))
            .mockResolvedValueOnce(new Response(JSON.stringify({ status: 'signed_out' }), {
                status: 200,
                headers: { 'Content-Type': 'application/json' }
            }));

        render(<AuthProvider><LogoutProbe /></AuthProvider>);
        await waitFor(() => expect(screen.getByRole('button', { name: 'Logout test' })).toBeEnabled());
        fireEvent.click(screen.getByRole('button', { name: 'Logout test' }));

        await waitFor(() => expect(fetchMock).toHaveBeenCalledWith(
            expect.stringMatching(/\/api\/oidc\/session\/logout$/),
            expect.objectContaining({ method: 'POST', credentials: 'include' })
        ));
    });

    it('continues with RP-initiated logout when an in-memory ID token is available', async () => {
        vi.mocked(refreshWithCookie).mockResolvedValueOnce({ accessToken: 'access', idToken: 'id-token', expiresAt: 9999999999 });
        const fetchMock = vi.spyOn(globalThis, 'fetch')
            .mockResolvedValueOnce(new Response(JSON.stringify(session), { status: 200, headers: { 'Content-Type': 'application/json' } }))
            .mockResolvedValueOnce(new Response(JSON.stringify({ status: 'signed_out' }), { status: 200, headers: { 'Content-Type': 'application/json' } }));

        render(<AuthProvider><LogoutProbe /></AuthProvider>);
        await waitFor(() => expect(screen.getByRole('button', { name: 'Logout test' })).toBeEnabled());
        fireEvent.click(screen.getByRole('button', { name: 'Logout test' }));

        await waitFor(() => expect(beginLogout).toHaveBeenCalledWith(expect.anything(), 'id-token'));
        expect(fetchMock).toHaveBeenCalledWith(expect.stringMatching(/\/api\/oidc\/session\/logout$/), expect.objectContaining({ method: 'POST' }));
    });
});
