import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

vi.mock('./oidc-client', () => ({
    beginAuthorization: vi.fn(),
    beginLogout: vi.fn(),
    completeAuthorization: vi.fn(),
    completeLogout: vi.fn(),
    consumeReturnTo: vi.fn(() => '/'),
    refreshWithCookie: vi.fn(async () => ({ accessToken: 'access', expiresAt: 9999999999 }))
}));

import { AuthProvider, educationPermissionImplies, useAuth } from './AuthProvider';
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
    tenant_code: 'tenant-test',
    permissions: ['registratura.read'],
    platform_roles: [],
    authz_version: 1,
    modules: [{ code: 'registratura', active: true }],
    authentication: ['sms'],
    gdpr_capabilities: []
};
const activeGrants = {
    tenant_code: session.tenant_code,
    institution_id: session.institution_id,
    revision: '1',
    evaluated_at: '2026-09-10T00:00:00Z',
    grants: []
};
const jsonResponse = (body: unknown, status = 200) => new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' }
});

function SessionProbe() {
    const auth = useAuth();
    return <div>{`${auth.ready ? auth.user?.name : 'loading'}:${String(auth.has('registratura.read'))}:${String(auth.has('admin.users.manage'))}`}</div>;
}

function LogoutProbe() {
    const auth = useAuth();
    return <button type="button" disabled={!auth.user} onClick={() => void auth.logout()}>Logout test</button>;
}
function AuthorizationProbe() {
    const auth = useAuth();
    return <div>{`${auth.ready}:${auth.authorizationReady}:${auth.canEducation('education.governance.read')}:${auth.canEducation('education.portfolios.school.manage', { resourceType: 'portfolio', resourceId: 'portfolio-1' })}:${auth.canEducation('education.portfolios.school.manage', { resourceType: 'portfolio', resourceId: 'portfolio-2' })}`}</div>;
}

function ApiFetchProbe() {
    const auth = useAuth();
    return <button type="button" disabled={!auth.user} onClick={() => void auth.apiFetch(new Request('http://localhost:3000/api/earchiva/documents', {
        method: 'POST',
        headers: {
            Accept: 'application/json',
            'Content-Type': 'multipart/form-data; boundary=----contract-boundary'
        },
        body: new Blob(['multipart-body'])
    }))}>Upload test</button>;
}

function BackgroundAuthorizationRefreshProbe() {
    const auth = useAuth();
    return <div>
        <button type="button" disabled={!auth.authorizationReady} onClick={() => void auth.refreshEducationAuthorization()}>Refresh authorization</button>
        {auth.authorizationReady && <input aria-label="Persistent workspace state" defaultValue="kept" />}
        <output aria-label="Resource grants">{`${auth.canEducation('education.portfolios.school.manage', { resourceType: 'portfolio', resourceId: 'portfolio-1' })}:${auth.canEducation('education.portfolios.school.manage', { resourceType: 'portfolio', resourceId: 'portfolio-2' })}`}</output>
    </div>;
}

describe('AuthProvider', () => {
    afterEach(() => vi.restoreAllMocks());

    it('treats manage as read only inside the same School permission family', () => {
        expect(educationPermissionImplies('education.governance.manage', 'education.governance.read')).toBe(true);
        expect(educationPermissionImplies('education.governance.manage', 'education.decisions.read')).toBe(false);
        expect(educationPermissionImplies('education.governance.read', 'education.governance.manage')).toBe(false);
        expect(educationPermissionImplies('registratura.manage', 'registratura.read')).toBe(false);
    });

    it('accepts the backend nested SessionContext and exposes effective permissions', async () => {
        vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) =>
            (input instanceof Request ? input.url : String(input)).includes('/active-grants') ? jsonResponse(activeGrants) : jsonResponse(session));

        render(<AuthProvider><SessionProbe /></AuthProvider>);

        await waitFor(() => expect(screen.getByText('Administrator:true:false')).toBeInTheDocument());
    });

    it('revokes logout below the path-scoped OIDC refresh-cookie route', async () => {
        const fetchMock = vi.spyOn(globalThis, 'fetch')
            .mockResolvedValueOnce(jsonResponse(session))
            .mockResolvedValueOnce(jsonResponse(activeGrants))
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
            .mockResolvedValueOnce(jsonResponse(session))
            .mockResolvedValueOnce(jsonResponse(activeGrants))
            .mockResolvedValueOnce(new Response(JSON.stringify({ status: 'signed_out' }), { status: 200, headers: { 'Content-Type': 'application/json' } }));

        render(<AuthProvider><LogoutProbe /></AuthProvider>);
        await waitFor(() => expect(screen.getByRole('button', { name: 'Logout test' })).toBeEnabled());
        fireEvent.click(screen.getByRole('button', { name: 'Logout test' }));

        await waitFor(() => expect(beginLogout).toHaveBeenCalledWith(expect.anything(), 'id-token'));
        expect(fetchMock).toHaveBeenCalledWith(expect.stringMatching(/\/api\/oidc\/session\/logout$/), expect.objectContaining({ method: 'POST' }));
    });

    it('preserves OpenAPI request headers while adding authorization', async () => {
        const fetchMock = vi.spyOn(globalThis, 'fetch')
            .mockResolvedValueOnce(jsonResponse(session))
            .mockResolvedValueOnce(jsonResponse(activeGrants))
            .mockResolvedValueOnce(new Response(JSON.stringify({ id: 'archive-1' }), { status: 201, headers: { 'Content-Type': 'application/json' } }));

        render(<AuthProvider><ApiFetchProbe /></AuthProvider>);
        const button = await screen.findByRole('button', { name: 'Upload test' });
        await waitFor(() => expect(button).toBeEnabled());
        fireEvent.click(button);

        await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(3));
        const [, , init] = fetchMock.mock.calls;
        const uploadInit = init?.[1];
        const headers = new Headers(uploadInit?.headers);
        expect(headers.get('content-type')).toBe('multipart/form-data; boundary=----contract-boundary');
        expect(headers.get('accept')).toBe('application/json');
        expect(headers.get('authorization')).toBe('Bearer access');
    });

    it('uses only the evaluated, tenant-bound grant snapshot and keeps resource grants exact', async () => {
        const educationSession = {
            ...session,
            permissions: ['education.governance.read'],
            modules: [{ code: 'education', active: true }]
        };
        vi.spyOn(globalThis, 'fetch')
            .mockResolvedValueOnce(jsonResponse(educationSession))
            .mockResolvedValueOnce(jsonResponse({
                ...activeGrants,
                grants: [{ permission_code: 'education.governance.read', resource_type: 'institution', resource_id: educationSession.institution_id }, {
                    permission_code: 'education.portfolios.school.manage', resource_type: 'portfolio', resource_id: 'portfolio-1'
                }]
            }));

        render(<AuthProvider><AuthorizationProbe /></AuthProvider>);

        // The final state arrives after both /api/me and the evaluated grant
        // snapshot; no route/navigation consumer receives an optimistic grant.
        await waitFor(() => expect(screen.getByText('true:true:true:true:false')).toBeInTheDocument());
    });

    it('keeps the evaluated snapshot mounted while background authorization refreshes', async () => {
        let resolveRefresh!: (response: Response) => void;
        vi.spyOn(globalThis, 'fetch')
            .mockResolvedValueOnce(jsonResponse(session))
            .mockResolvedValueOnce(jsonResponse(activeGrants))
            .mockImplementationOnce(() => new Promise<Response>((resolve) => { resolveRefresh = resolve; }));

        render(<AuthProvider><BackgroundAuthorizationRefreshProbe /></AuthProvider>);
        const refresh = await screen.findByRole('button', { name: 'Refresh authorization' });
        await waitFor(() => expect(refresh).toBeEnabled());
        fireEvent.change(screen.getByLabelText('Persistent workspace state'), { target: { value: 'operator draft' } });
        fireEvent.click(refresh);

        // A pending background request must not trip route guards or remount
        // the workspace containing the operator's in-progress state.
        expect(screen.getByLabelText('Persistent workspace state')).toHaveValue('operator draft');
        resolveRefresh(jsonResponse(activeGrants));
        await waitFor(() => expect(screen.getByLabelText('Persistent workspace state')).toHaveValue('operator draft'));
    });

    it('never lets an older overlapping refresh restore stale grants', async () => {
        let resolveOlder!: (response: Response) => void;
        let resolveNewest!: (response: Response) => void;
        vi.spyOn(globalThis, 'fetch')
            .mockResolvedValueOnce(jsonResponse(session))
            .mockResolvedValueOnce(jsonResponse(activeGrants))
            .mockImplementationOnce(() => new Promise<Response>((resolve) => { resolveOlder = resolve; }))
            .mockImplementationOnce(() => new Promise<Response>((resolve) => { resolveNewest = resolve; }));

        render(<AuthProvider><BackgroundAuthorizationRefreshProbe /></AuthProvider>);
        const refresh = await screen.findByRole('button', { name: 'Refresh authorization' });
        await waitFor(() => expect(refresh).toBeEnabled());
        fireEvent.click(refresh);
        fireEvent.click(refresh);

        await act(async () => {
            resolveNewest(jsonResponse({
                ...activeGrants,
                revision: '3',
                grants: [{ permission_code: 'education.portfolios.school.manage', resource_type: 'portfolio', resource_id: 'portfolio-2' }]
            }));
        });
        await waitFor(() => expect(screen.getByLabelText('Resource grants')).toHaveTextContent('false:true'));

        await act(async () => {
            resolveOlder(jsonResponse({
                ...activeGrants,
                revision: '2',
                grants: [{ permission_code: 'education.portfolios.school.manage', resource_type: 'portfolio', resource_id: 'portfolio-1' }]
            }));
        });
        expect(screen.getByLabelText('Resource grants')).toHaveTextContent('false:true');
    });

    it('fails closed for malformed or cross-tenant grants without removing direct permissions', async () => {
        const educationSession = { ...session, permissions: ['education.governance.read'], modules: [{ code: 'education', active: true }] };
        vi.spyOn(globalThis, 'fetch')
            .mockResolvedValueOnce(jsonResponse(educationSession))
            .mockResolvedValueOnce(jsonResponse({ ...activeGrants, tenant_code: 'other-tenant', grants: [{ permission_code: 'education.portfolios.school.manage', resource_type: 'portfolio', resource_id: 'portfolio-1' }] }));

        render(<AuthProvider><AuthorizationProbe /></AuthProvider>);
        await waitFor(() => expect(screen.getByText('true:true:true:false:false')).toBeInTheDocument());
    });
});
