import { expect, test } from '@playwright/test';

test('workflow renders a backend-provided task and opens its dossier detail', async ({ page }) => {
  await page.route('**/api/oidc/.well-known/openid-configuration', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ issuer: 'http://127.0.0.1:4173/api/oidc', authorization_endpoint: 'http://127.0.0.1:4173/api/oidc/authorize', token_endpoint: 'http://127.0.0.1:4173/api/oidc/token', jwks_uri: 'http://127.0.0.1:4173/api/oidc/jwks', response_types_supported: ['code'], subject_types_supported: ['public'], id_token_signing_alg_values_supported: ['RS256'], grant_types_supported: ['authorization_code', 'refresh_token'], code_challenge_methods_supported: ['S256'] }) }));
  await page.route('**/api/oidc/token', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ access_token: 'test-access-token', token_type: 'Bearer', expires_in: 3600 }) }));
  await page.route('**/api/me', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ user: { id: 'user-1', sub: 'subject-1', name: 'Ana', email: 'ana@example.test', email_verified: true, phone_number: '', phone_number_verified: false, preferred_otp_channel: 'sms', locale: 'ro', roles: ['super_admin'] }, institution_id: 'inst-1', institution_name: 'Școala Test', permissions: ['workflow.read', 'workflow.transition'], modules: [{ code: 'workflow', active: true }], authentication: ['sms'], gdpr_capabilities: [] }) }));
  await page.route('**/api/workflow/dashboard', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ stats: { active_tasks: 1, overdue_tasks: 0, waiting_approval: 0, active_definitions: 1, ready_dossiers: 1, blocked_dossiers: 0 } }) }));
  await page.route('**/api/workflow/tasks/filters', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ statuses: ['new'], priorities: ['medium'], assignees: [] }) }));
  await page.route('**/api/workflow/definitions', (route) => route.fulfill({ contentType: 'application/json', body: '[]' }));
  await page.route('**/api/workflow/tasks?**', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ items: [{ id: 'task-1', title: 'Avizare plan', definition_name: 'Avizare', definition_code: 'approval', document_number: '1', source_module: 'education', status: 'new', priority: 'medium', assigned_to: 'Ana', current_step: 'initial', due_at: null, started_at: '2026-01-01', updated_at: '2026-01-01', summary: 'Test', linked_documents_count: 1, dossier_ready: true, missing_relations: [], available_actions: [] }], total: 1, page: 1, pageSize: 25 }) }));
  // Establish the refresh-cookie session first; a direct protected deep link is
  // intentionally redirected while the bootstrap session is still unresolved.
  await page.goto('/');
  await expect(page.getByText('Ana')).toBeVisible();
  await page.goto('/flux-documente');
  await expect(page.getByText('Avizare plan')).toBeVisible();
  await page.getByRole('button', { name: 'Detalii' }).click();
  await expect(page.getByText(/Dosar pregătit/)).toBeVisible();
});

test('earchiva displays an archive document and its versions from backend routes', async ({ page }) => {
  await page.route('**/api/oidc/.well-known/openid-configuration', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ issuer: 'http://127.0.0.1:4173/api/oidc', authorization_endpoint: 'http://127.0.0.1:4173/api/oidc/authorize', token_endpoint: 'http://127.0.0.1:4173/api/oidc/token', jwks_uri: 'http://127.0.0.1:4173/api/oidc/jwks', response_types_supported: ['code'], subject_types_supported: ['public'], id_token_signing_alg_values_supported: ['RS256'], grant_types_supported: ['authorization_code', 'refresh_token'], code_challenge_methods_supported: ['S256'] }) }));
  await page.route('**/api/oidc/token', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ access_token: 'test-access-token', token_type: 'Bearer', expires_in: 3600 }) }));
  await page.route('**/api/me', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ user: { id: 'user-1', sub: 'subject-1', name: 'Ana', email: 'ana@example.test', email_verified: true, phone_number: '', phone_number_verified: false, preferred_otp_channel: 'sms', locale: 'ro', roles: ['archivist'] }, institution_id: 'inst-1', institution_name: 'Școala Test', permissions: ['earchiva.read', 'earchiva.manage'], modules: [], authentication: ['sms'], gdpr_capabilities: [] }) }));
  await page.route('**/api/earchiva/dashboard', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ stats: { total_records: 0, validated_records: 0, draft_records: 0, unique_fonds: 0 } }) }));
  await page.route('**/api/earchiva/records/filters', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ fonds: [], series: [], statuses: [], source_modules: [], archivists: [] }) }));
  await page.route('**/api/earchiva/records?**', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ items: [], total: 0 }) }));
  await page.route('**/api/earchiva/taxonomy?**', (route) => route.fulfill({ contentType: 'application/json', body: '[]' }));
  await page.route('**/api/earchiva/admin/health', (route) => route.fulfill({ contentType: 'application/json', body: '{}' }));
  await page.route('**/api/earchiva/admin/stats', (route) => route.fulfill({ contentType: 'application/json', body: '{}' }));
  await page.route('**/api/earchiva/admin/jobs?**', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ items: [] }) }));
  await page.route('**/api/earchiva/documents/doc-1/versions', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify([{ id: 'v1', document_id: 'doc-1', version_no: 1, source_sha256: '', source_size_bytes: 12, page_count: 1, text_status: 'ready', created_at: '2026-01-01', chunk_count: 2 }]) }));
  await page.route('**/api/earchiva/documents/doc-1', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ id: 'doc-1', title: 'Catalog 2026', original_file_name: 'catalog.pdf', mime_type: 'application/pdf', source_kind: 'scan', source_system: '', external_reference: '', status: 'ready', current_version_no: 1, received_at: '2026-01-01', created_at: '2026-01-01', updated_at: '2026-01-01' }) }));
  await page.route('**/api/earchiva/documents?**', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ items: [{ id: 'doc-1', title: 'Catalog 2026', original_file_name: 'catalog.pdf', mime_type: 'application/pdf', source_kind: 'scan', status: 'ready', current_version_no: 1, received_at: '2026-01-01', created_at: '2026-01-01', updated_at: '2026-01-01' }], total: 1 }) }));
  await page.goto('/');
  await expect(page.getByText('Ana')).toBeVisible();
  await page.goto('/earchiva');
  await expect(page.getByText('Catalog 2026')).toBeVisible();
  await page.getByRole('button', { name: 'Detalii' }).click();
  await expect(page.getByText(/v1/)).toBeVisible();
  await expect(page.getByText('Administrare eArhivă')).toBeVisible();
});
