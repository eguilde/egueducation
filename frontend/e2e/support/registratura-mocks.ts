import { Page, Route } from '@playwright/test';

const tenantId = 'tenant-registratura-e2e';

const json = (route: Route, body: unknown, status = 200) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) });

export async function bootstrapRegistraturaManager(page: Page): Promise<void> {
  await page.addInitScript(({ institutionId }) => {
    // Production sessions never place bearer material in web storage. This
    // localhost-only fixture is read explicitly by AuthService for E2E tests.
    localStorage.setItem('egueducation_e2e_session', JSON.stringify({ profile: { sub: 'manager-1', name: 'Manager Registratură', email: 'manager@example.test', roles: ['director'] }, accessToken: 'registratura-e2e-token', expiresAt: Math.floor(Date.now() / 1000) + 3600, session: { user: { id: 'manager-1', sub: 'manager-1', name: 'Manager Registratură', email: 'manager@example.test', locale: 'ro', roles: ['director'] }, institution_id: institutionId, institution_name: 'Tenant Registratură E2E', permissions: ['registratura.read', 'registratura.manage', 'workflow.manage', 'admin.read'], modules: [{ code: 'registratura', active: true }, { code: 'workflow', active: true }, { code: 'earchiva', active: true }, { code: 'admin', active: true }], authentication: ['oidc'], gdpr_capabilities: [] } }));
  }, { institutionId: tenantId });
}

export async function registerRegistraturaApiMocks(page: Page, captured: { workflowBodies: unknown[]; uploadBodies: string[] }): Promise<void> {
  const registryA = { id: 11, nume: 'Registru E2E A', prefix_nr: 'E2EA', nr_inceput: 1, nr_curent: '1', nr_urmator: '2', data_resetare: null, tip_registru: 'public', isDefault: true, created_at: '', updated_at: '' };
  const registryB = { ...registryA, id: 12, nume: 'Registru E2E B', prefix_nr: 'E2EB', isDefault: false };
  const document = { id: 'doc-1', registru_id: 11, registry_number: 'E2EA-2026-0001', subject: 'Cerere E2E', document_type: 'cerere', direction: 'intrare', status: 'INCOMING', correspondent: 'Emitent E2E', assigned_to: 'Instituție E2E', correspondent_party_id: 'party-sender', assigned_party_id: 'party-org', institution_id: tenantId, confidentiality: 'normal', summary: '', registered_at: '2026-08-16', due_date: null, department_ids: [], department_names: [] };
  const party = { id: 'party-org', code: 'ORG', party_type: 'institution', display_name: 'Instituție E2E', short_name: 'E2E', first_name: '', last_name: '', legal_name: 'Instituție E2E', identifier_code: '', tax_id: '', phone_number: '', email: '', address_line1: '', address_line2: '', locality: '', county: '', country: 'RO', notes: '', is_default_organization: true, active: true, created_at: '', updated_at: '' };

  await page.route('**/api/**', async (route) => {
    const url = new URL(route.request().url());
    const path = url.pathname;
    const method = route.request().method();
    if (path === '/api/config') return json(route, { institutionId: tenantId, institutionName: 'Tenant Registratură E2E', customer: { name: 'E2E' }, service: { title: 'EguEducation' } });
    if (path === '/api/me') return json(route, { user: { id: 'manager-1', sub: 'manager-1', name: 'Manager Registratură', email: 'manager@example.test', locale: 'ro', roles: ['director'] }, institution_id: tenantId, institution_name: 'Tenant Registratură E2E', permissions: ['registratura.read', 'registratura.manage', 'workflow.manage', 'admin.read'], modules: [{ code: 'registratura', active: true }, { code: 'workflow', active: true }, { code: 'earchiva', active: true }, { code: 'admin', active: true }], authentication: ['oidc'], gdpr_capabilities: [] });
    if (path === '/api/auth/role-catalog') return json(route, { roles: [] });
    if (path === '/api/registratura/registre') return json(route, [registryA, registryB]);
    if (path === '/api/registratura/registre/default') return json(route, registryA);
    if (path === '/api/registratura/documents/filters') return json(route, { document_types: ['cerere'], directions: ['intrare', 'iesire'], statuses: ['INCOMING', 'ALOCAT_COMPARTIMENT', 'IN_LUCRU', 'FLUX_APROBARE', 'FINALIZAT', 'ANULAT'], confidentialities: ['normal'] });
    if (path === '/api/registratura/parties/lookup') return json(route, [party]);
    if (path === '/api/registratura/documents' && method === 'GET') return json(route, { items: [{ ...document, registru_id: Number(url.searchParams.get('filter.registru_id') ?? 11) }], total: 1, page: 1, pageSize: 20 });
    if (path === '/api/registratura/documents' && method === 'POST') return json(route, document, 201);
    if (path === '/api/registratura/documents/doc-1/workflow-history') return json(route, []);
    if (path === '/api/registratura/workflow-assignees') return json(route, { departments: [{ id: 'dept-1', name: 'Secretariat' }], users: [{ id: 'user-1', name: 'Responsabil E2E', department_ids: ['dept-1'] }] });
    if (path === '/api/registratura/documents/doc-1/workflow-actions') { captured.workflowBodies.push(route.request().postDataJSON()); return json(route, { ...document, status: 'ALOCAT_COMPARTIMENT', workflow_version: 2, workflow_assignment: { department_id: 'dept-1', department_name: 'Secretariat' } }); }
    if (path === '/api/registratura/documents/doc-1/attachments/upload') { captured.uploadBodies.push((await route.request().headerValue('content-type')) ?? ''); return json(route, { id: 'attachment-1', document_id: 'doc-1', title: 'e2e.txt', file_name: 'e2e.txt', mime_type: 'text/plain', storage_key: `${tenantId}/e2e.txt`, size_bytes: 3, category: 'primary', status: 'ready', uploaded_by: 'manager-1', uploaded_at: '2026-08-16T00:00:00Z' }, 201); }
    if (path === '/api/registratura/documents/doc-1/attachments/attachment-1/download') return route.fulfill({ status: 200, contentType: 'text/plain', body: 'e2e' });
    if (path === '/api/registratura/documents/doc-1') return json(route, document);
    if (path === '/api/registratura/documents/doc-1/versions' || path === '/api/registratura/documents/doc-1/attachments') return json(route, []);
    if (path.startsWith('/api/registratura/admin/')) return json(route, path.endsWith('organization-chart') ? [] : { items: [], total: 0, page: 1, pageSize: 25 });
    if (path === '/api/registratura/parties') return json(route, { items: [], total: 0, page: 1, pageSize: 25 });
    return json(route, {});
  });
}
