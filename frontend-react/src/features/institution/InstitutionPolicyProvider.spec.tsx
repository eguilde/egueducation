import { render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { InstitutionPolicyProvider, useInstitutionPolicy } from './InstitutionPolicyProvider';

let authState: { session: { tenant_code: string; institution_id: string } | null; apiFetch: (request: Request) => Promise<Response> };
vi.mock('../../auth/AuthProvider', () => ({ useAuth: () => authState }));

const payload = (tenant: string, institution: string) => ({
  tenant_code: tenant, institution_id: institution, evaluated_at: '2026-09-11T00:00:00Z', evaluation_id: null,
  profile_id: 'profile-1', profile_version: 2, profile_status: 'active', school_legal_form: 'public', blocked: false,
  block_reason: '', warnings: [], effective_policies: [], capabilities: [{ code: 'education.publication.manage', enabled: true,
    reason: 'allowed', required_permission: 'education.compliance.manage', required_documents: [], required_approvals: [], wizard_steps: [] }],
});
const json = (value: unknown) => new Response(JSON.stringify(value), { status: 200, headers: { 'content-type': 'application/json' } });

function Probe() {
  const policy = useInstitutionPolicy();
  return <div data-testid="policy">{`${policy.ready}:${policy.blocked}:${policy.canPolicy('education.publication.manage')}:${policy.capabilities?.tenant_code ?? 'none'}`}</div>;
}

describe('InstitutionPolicyProvider', () => {
  it('publishes only a runtime-valid response for the current session scope', async () => {
    authState = { session: { tenant_code: 'tenant-a', institution_id: 'inst-a' }, apiFetch: vi.fn(async () => json(payload('tenant-a', 'inst-a'))) };
    render(<InstitutionPolicyProvider><Probe /></InstitutionPolicyProvider>);
    await waitFor(() => expect(screen.getByTestId('policy')).toHaveTextContent('true:false:true:tenant-a'));
  });

  it('fails closed for malformed or cross-tenant responses', async () => {
    authState = { session: { tenant_code: 'tenant-a', institution_id: 'inst-a' }, apiFetch: vi.fn(async () => json(payload('tenant-b', 'inst-b'))) };
    const view = render(<InstitutionPolicyProvider><Probe /></InstitutionPolicyProvider>);
    await waitFor(() => expect(screen.getByTestId('policy')).toHaveTextContent('true:true:false:none'));
    authState = { ...authState, apiFetch: vi.fn(async () => json({ tenant_code: 'tenant-a', institution_id: 'inst-a', capabilities: 'invalid' })) };
    view.rerender(<InstitutionPolicyProvider><Probe /></InstitutionPolicyProvider>);
    await waitFor(() => expect(screen.getByTestId('policy')).toHaveTextContent('true:true:false:none'));
  });
});
