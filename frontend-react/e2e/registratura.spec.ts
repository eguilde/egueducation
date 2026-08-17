import { expect, test, type Page } from '@playwright/test';

const document = { id: 'doc-1', registru_id: 1, registry_number: 'REG-1', subject: 'Cerere înscriere', document_type: 'DOCUMENT', direction: 'intrare', status: 'INCOMING', correspondent: 'Ana Pop', assigned_to: '', registered_at: '2026-08-16', confidentiality: 'normal', summary: '' };

async function authenticatedRegistratura(page: Page) {
  await page.route('**/api/oidc/.well-known/openid-configuration', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ issuer: 'http://127.0.0.1:4173/api/oidc', authorization_endpoint: 'http://127.0.0.1:4173/api/oidc/authorize', token_endpoint: 'http://127.0.0.1:4173/api/oidc/token', jwks_uri: 'http://127.0.0.1:4173/api/oidc/jwks', response_types_supported: ['code'], subject_types_supported: ['public'], id_token_signing_alg_values_supported: ['RS256'], grant_types_supported: ['authorization_code', 'refresh_token'], code_challenge_methods_supported: ['S256'] }) }));
  await page.route('**/api/oidc/token', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ access_token: 'test-access-token', token_type: 'Bearer', expires_in: 3600 }) }));
  await page.route('**/api/me', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ user: { id: 'user-1', sub: 'subject-1', name: 'Ana', email: 'ana@example.test', email_verified: true, phone_number: '', phone_number_verified: false, preferred_otp_channel: 'sms', locale: 'ro', roles: [] }, institution_id: 'inst-1', institution_name: 'Școala Test', permissions: ['registratura.read', 'registratura.manage', 'workflow.manage', 'registratura.links.read', 'registratura.links.manage'], modules: [{ code: 'registratura', active: true }], authentication: ['sms'], gdpr_capabilities: [] }) }));
  await page.route('**/api/registratura/registre', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify([{ id: 1, nume: 'Registru general', prefix_nr: 'REG', tip_registru: 'public', isDefault: true }]) }));
  await page.route('**/api/registratura/documents/filters', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ directions: ['intrare', 'iesire'], statuses: ['INCOMING'], document_types: ['DOCUMENT'], confidentialities: ['normal'] }) }));
  await page.route('**/api/registratura/documents?**', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ items: [document], total: 45, page: Number(new URL(route.request().url()).searchParams.get('page') ?? 1), pageSize: 20 }) }));
  await page.route('**/api/registratura/parties?**', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ items: [{ id: 'party-1', party_type: 'physical', display_name: 'Ana Pop' }], total: 1, page: 1, pageSize: 50 }) }));
  await page.route('**/api/registratura/admin/departments?**', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ items: [{ id: 'dept-1', name: 'Secretariat', description: '', active: true }], total: 1, page: 1, pageSize: 50 }) }));
  await page.route('**/api/registratura/documents/doc-1', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify(document) }));
  await page.route('**/api/registratura/documents/doc-1/versions', (route) => route.fulfill({ contentType: 'application/json', body: '[]' }));
  await page.route('**/api/registratura/documents/doc-1/attachments', (route) => route.fulfill({ contentType: 'application/json', body: '[]' }));
  await page.route('**/api/registratura/documents/doc-1/workflow-history', (route) => route.fulfill({ contentType: 'application/json', body: '[]' }));
  await page.route('**/api/registratura/workflow-assignees', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ departments: [{ id: 'dept-1', name: 'Secretariat' }], users: [] }) }));
  await page.route('**/api/registratura/document-links?**', (route) => route.fulfill({ contentType: 'application/json', body: '[]' }));
  await page.route('**/api/registratura/document-links', (route) => route.fulfill({ status: 201, contentType: 'application/json', body: JSON.stringify({ link_id: 'link-1', document_id: 'doc-1', registry_number: 'REG-1', subject: 'Cerere înscriere', document_type: 'DOCUMENT', status: 'INCOMING', relation_type: 'supporting', registered_at: '2026-08-16', confidentiality: 'normal' }) }));
  await page.goto('/');
  if ((page.viewportSize()?.width ?? 1280) < 768) {
    await page.getByRole('button', { name: 'Deschide navigația' }).click();
  }
  await expect(page.getByText('Ana')).toBeVisible();
  await page.goto('/registratura');
  await expect(page.getByLabel('Registratură')).toBeVisible();
}

test('registratura creates an incoming document with operational fields', async ({ page }) => {
  await authenticatedRegistratura(page);
  await page.route('**/api/registratura/documents', (route) => route.fulfill({ status: 201, contentType: 'application/json', body: JSON.stringify(document) }));
  await page.getByRole('button', { name: 'Intrare' }).click();
  const createDialog = page.getByRole('dialog', { name: 'Înregistrare intrare' });
  await createDialog.getByRole('textbox', { name: 'Conținut', exact: true }).fill('Cerere înscriere');
  await createDialog.getByRole('textbox', { name: 'Emitent', exact: true }).fill('Ana Pop');
  await createDialog.getByRole('textbox', { name: 'Destinatar', exact: true }).fill('Școala Test');
  await createDialog.getByRole('button', { name: 'Secretariat' }).click();
  await createDialog.getByLabel('Număr extern').fill('EXT-11');
  await expect(createDialog.getByLabel('Data numărului extern')).toBeVisible();
  await expect(createDialog.getByText('Compartimente responsabile')).toBeVisible();
  await expect(createDialog.getByText('Atașamente scanate (se încarcă după crearea documentului)')).toBeVisible();
  await createDialog.getByRole('button', { name: 'Salvează' }).click();
  await expect(page.getByRole('dialog', { name: /Detalii document REG-1/ })).toBeVisible();
  await page.getByLabel('Modul sursă legătură').fill('education');
  await page.getByLabel('ID înregistrare sursă legătură').fill('11111111-1111-1111-1111-111111111111');
  const linkRequest = page.waitForRequest((request) => request.url().endsWith('/api/registratura/document-links') && request.method() === 'POST');
  await page.getByRole('button', { name: 'Adaugă legătură' }).click();
  const request = await linkRequest;
  expect(request.postDataJSON()).toEqual({ document_id: 'doc-1', source_module: 'education', source_record_id: '11111111-1111-1111-1111-111111111111', relation_type: 'supporting' });
});

test('registratura mirrors the Costești table controls with server-side requests', async ({ page }) => {
  await authenticatedRegistratura(page);
  await expect(page.getByRole('button', { name: 'Istoric REG-1' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Editează REG-1' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Anulează REG-1' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'PDF REG-1' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Flux REG-1' })).toBeVisible();
  await page.getByRole('button', { name: 'Extinde REG-1' }).click();
  await expect(page.getByText(/Compartimente:/)).toBeVisible();

  const sorted = page.waitForRequest((request) => new URL(request.url()).searchParams.get('sort') === 'registry_number');
  await page.getByRole('button', { name: 'Sortează după Nr. Doc' }).click();
  expect(new URL((await sorted).url()).searchParams.get('direction')).toBe('asc');

  const pageTwo = page.waitForRequest((request) => new URL(request.url()).searchParams.get('page') === '2');
  await page.getByRole('button', { name: 'Pagina 2' }).click();
  expect(new URL((await pageTwo).url()).searchParams.get('pageSize')).toBe('20');

  await expect(page.getByLabel('Căutare documente')).toHaveCount(0);
  await page.getByRole('button', { name: 'Deschide căutarea' }).click();
  await expect(page.getByLabel('Nr. Extern')).toBeVisible();
  await expect(page.getByLabel('Data intrare de la')).toBeVisible();
  await expect(page.getByLabel('Data ieșire până la')).toBeVisible();
});

test('registratura stays usable in the narrow responsive layout and exposes administration', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await authenticatedRegistratura(page);
  await expect(page.getByRole('button', { name: 'Intrare' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Administrare' })).toBeVisible();
  await page.getByRole('button', { name: 'Administrare' }).click();
  await expect(page.getByRole('dialog', { name: 'Administrare Registratură' })).toBeVisible();
  await expect(page.getByLabel('Nume element')).toBeVisible();
});
