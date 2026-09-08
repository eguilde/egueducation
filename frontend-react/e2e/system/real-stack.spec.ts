import { expect, test, type Page } from '@playwright/test';
import { execFileSync } from 'node:child_process';
import type { components } from '../../src/api/generated';

const fixtureIdentifier = 'oidc.browser.fixture@example.test';
const fixtureOTP = '173829';
const approverIdentifier = 'oidc.approver.fixture@example.test';
const approverOTP = '428615';
const balotestiIdentifier = 'oidc.balotesti.fixture@example.test';
const balotestiOTP = '739204';
const marker = `SYSTEM-E2E-${process.env.GITHUB_RUN_ID ?? 'local'}-${Date.now()}`;

type TokenResponse = { access_token: string };
type CreatedDocument = components['schemas']['RegistraturaDocument'];
type AdminUser = { id: string; name: string; phone: string; phone_verified: boolean };

type TenantScope = { code: string; institutionID: string };

const egueducationScope: TenantScope = { code: 'tenant-egueducation', institutionID: 'inst-001' };
const balotestiScope: TenantScope = { code: 'tenant-balotesti', institutionID: 'inst-balotesti' };

function databaseScalar(sql: string, scope: TenantScope = egueducationScope): string {
  const databaseURL = process.env.TEST_DATABASE_URL;
  if (!databaseURL) throw new Error('TEST_DATABASE_URL is required for the non-mocked system E2E suite.');
  // Application tables use FORCE RLS. The verification connection therefore
  // declares exactly the tenant it is asserting; it never relies on table
  // ownership to read across tenants.
  const scopedSQL = `set app.tenant_id = '${scope.code}'; set app.institution_id = '${scope.institutionID}'; set app.is_super_admin = 'true'; ${sql}`;
  return execFileSync('psql', ['--no-psqlrc', '--tuples-only', '--no-align', '--quiet', databaseURL, '-c', scopedSQL], {
    encoding: 'utf8',
    stdio: ['ignore', 'pipe', 'pipe'],
  }).trim();
}

function databaseExec(sql: string, scope: TenantScope = egueducationScope): void {
  databaseScalar(`${sql}; select 'ok'`, scope);
}

async function authenticated(page: Page, identifier = fixtureIdentifier, otp = fixtureOTP, expectedOrigin = 'http://localhost:4173'): Promise<string> {
  let token: string | undefined;
  page.on('response', async (response) => {
    if (!response.url().includes('/api/oidc/token') || response.request().method() !== 'POST') return;
    const body = await response.json().catch(() => undefined) as TokenResponse | undefined;
    token ??= body?.access_token;
  });

  await page.goto('/');
  await page.getByRole('button', { name: 'Autentificare' }).last().click();
  await expect(page).toHaveURL(/\/api\/oidc\/authorize/);
  await page.getByRole('button', { name: /SMS/ }).click();
  await page.getByLabel('Utilizator, email sau numar de telefon').fill(identifier);
  await page.getByLabel('Canal OTP').selectOption('sms');
  await page.getByRole('button', { name: 'Trimite codul' }).click();

  const boxes = page.locator('.otp-box');
  await expect(boxes).toHaveCount(6);
  // This must exercise the interaction users have: each browser keypress
  // triggers the provider page's input handler, advances focus, and finally
  // moves it to Verify. Filling individual controls would bypass that code.
  await boxes.first().click();
  await boxes.first().pressSequentially(otp);
  const verify = page.getByRole('button', { name: 'Verifica codul' });
  await expect(verify).toBeEnabled();
  await expect(verify).toBeFocused();
  await verify.click();
  const consent = page.getByRole('button', { name: 'Accepta si continua' });
  if (await consent.count()) await consent.click();
  await expect(page).toHaveURL(expectedOrigin + '/');
  await expect(page.getByText('Utilizator Test')).toBeVisible();
  await expect.poll(() => token, { message: 'OIDC token response was not observed by the real browser' }).toBeTruthy();
  return token!;
}

async function authenticatedWithPasskey(page: Page, expectedOrigin = 'http://localhost:4173'): Promise<string> {
  let token: string | undefined;
  page.on('response', async (response) => {
    if (!response.url().includes('/api/oidc/token') || response.request().method() !== 'POST') return;
    const body = await response.json().catch(() => undefined) as TokenResponse | undefined;
    token ??= body?.access_token;
  });
  await page.goto('/');
  await page.getByRole('button', { name: 'Autentificare' }).last().click();
  await expect(page).toHaveURL(/\/api\/oidc\/authorize/);
  await page.getByRole('button', { name: /Passkey/ }).click();
  const consent = page.getByRole('button', { name: 'Accepta si continua' });
  // The native WebAuthn ceremony completes asynchronously before the provider
  // renders consent. Waiting for the actual interaction prevents the test from
  // racing directly into the callback assertion.
  await expect(consent).toBeVisible();
  await consent.click();
  await expect(page).toHaveURL(expectedOrigin + '/');
  await expect(page.getByText('Utilizator Test')).toBeVisible();
  await expect.poll(() => token, { message: 'OIDC passkey token response was not observed by the real browser' }).toBeTruthy();
  return token!;
}

function jwtPayload(token: string): Record<string, unknown> {
  const encoded = token.split('.')[1];
  if (!encoded) throw new Error('Access token is not a JWT.');
  return JSON.parse(Buffer.from(encoded, 'base64url').toString('utf8')) as Record<string, unknown>;
}

async function api<T>(page: Page, token: string, path: string, init: RequestInit = {}): Promise<{ status: number; body: T }> {
  return page.evaluate(async ({ token, path, init }) => {
    const headers = new Headers(init.headers);
    headers.set('Authorization', `Bearer ${token}`);
    headers.set('Content-Type', 'application/json');
    const response = await fetch(path, { ...init, headers, credentials: 'include' });
    const text = await response.text();
    return { status: response.status, body: text ? JSON.parse(text) : null };
  }, { token, path, init });
}

test('real React, two OIDC users, RBAC, Flux, tenant isolation and PostgreSQL contract proof', async ({ page, browser }) => {
  // The bootstrap fixture is deliberately unverified. The browser must prove
  // possession through the ordinary OTP transaction before the profile and
  // global identity projections are promoted.
  expect(databaseScalar("select phone_number_verified::text from app_users where sub='oidc-browser-fixture-subject'"))
    .toBe('false');
  // The primary automated actor deliberately has every authority required by
  // the production E2E suite. Platform authority remains an explicit global
  // assignment and is loaded into the OIDC token; it is never inferred from a
  // tenant administrator role.
  const platformAdminID = databaseScalar("select id::text from app_users where sub='oidc-browser-fixture-subject'");
  databaseExec(`
    insert into app_user_platform_roles(user_id, role_code)
    values ('${platformAdminID}', 'platform_super_admin')
    on conflict (user_id, role_code) do nothing
  `);
  const token = await authenticated(page);

  // The second server instance has provisioned an independent fixture in the
  // same tenant. Demote it from the all-powerful fixture position before it
  // authenticates, so its access token proves tenant RBAC rather than a
  // platform bypass.
  const approverID = databaseScalar("select id::text from app_users where sub='oidc-browser-approver-subject'");
  databaseExec(`
    update app_memberships set position_code='profesor'
    where user_id='${approverID}' and tenant_code='tenant-egueducation';
    delete from app_user_roles where user_id='${approverID}' and tenant_code='tenant-egueducation'
  `);
  const approverContext = await browser.newContext({ baseURL: 'http://localhost:4174' });
  const approverPage = await approverContext.newPage();
  const approverToken = await authenticated(approverPage, approverIdentifier, approverOTP, 'http://localhost:4174');
  const approverMe = await api<{ tenant_code: string; user: { roles: string[] }; platform_roles: string[]; permissions: string[] }>(approverPage, approverToken, '/api/me');
  expect(approverMe.status).toBe(200);
  expect(approverMe.body.tenant_code).toBe('tenant-egueducation');
  expect(approverMe.body.user.roles).toContain('profesor');
  expect(approverMe.body.platform_roles).not.toContain('platform_super_admin');
  expect(approverMe.body.permissions).not.toContain('workflow.manage');

  const me = await api<{ tenant_code: string; platform_roles: string[]; permissions: string[]; authz_version: number; user: { phone_number_verified: boolean } }>(page, token, '/api/me');
  expect(me.status).toBe(200);
  expect(me.body.tenant_code).toBe('tenant-egueducation');
  expect(me.body.platform_roles).toContain('platform_super_admin');
  expect(jwtPayload(token).platform_roles).toContain('platform_super_admin');
  expect(me.body.permissions).toContain('registratura.manage');
  expect(me.body.permissions).toContain('admin.users.manage');
  expect(me.body.authz_version).toBeGreaterThan(0);
  expect(me.body.user.phone_number_verified).toBe(true);
  expect(databaseScalar(`
    select count(*)::text
    from app_audit_log l
    join app_users u on u.id::text = l.details->>'user_id'
    where u.sub='oidc-browser-fixture-subject'
      and l.action='identity.phone.enrollment_verified'
      and l.status='success'
  `)).toBe('1');

  await page.goto('/administrare');
  await expect(page.getByRole('heading', { name: /Administrare/ })).toBeVisible();
  const dashboard = await api<{ stats: { users: number } }>(page, token, '/api/admin/dashboard');
  expect(dashboard.status).toBe(200);

  // Admin provisioning is exercised through the React administration UI. The
  // server deliberately ignores a tenant administrator's attempted
  // phone_verified assertion: only the normal OTP interaction may promote it.
  await page.getByRole('tab', { name: 'Utilizatori' }).click();
  await page.getByRole('button', { name: 'Utilizator' }).click();
  const userDialog = page.getByRole('dialog', { name: 'Utilizator nou' });
  const managedPhone = `+40710${String(Date.now()).slice(-6)}`;
  await userDialog.getByLabel('Nume').fill(marker);
  await userDialog.getByLabel('Telefon (SMS)').fill(managedPhone);
  await userDialog.getByLabel('Verificare telefon').click();
  await page.getByRole('option', { name: 'Telefon verificat' }).click();
  const userCreated = page.waitForResponse((response) => new URL(response.url()).pathname === '/api/admin/users' && response.request().method() === 'POST');
  await userDialog.getByRole('button', { name: 'Salvează' }).click();
  expect((await userCreated).status()).toBe(201);
  const managedUserID = databaseScalar(`select id::text from app_users where phone_number='${managedPhone}'`);
  expect(databaseScalar(`select phone_number_verified::text from app_users where id='${managedUserID}'`)).toBe('false');

  // The generic AdministrationResources UI is the actual admin surface for
  // memberships, role grants, and role permissions. It uses the same
  // generated React transport as production users; these waits bind the UI
  // controls to their real, tenant-scoped HTTP writes.
  await page.getByRole('button', { name: 'Apartenențe' }).click();
  await page.getByRole('button', { name: 'Adaugă sau actualizează' }).click();
  const membershipDialog = page.getByRole('dialog', { name: 'Configurare instituție' });
  await membershipDialog.getByLabel('ID utilizator').fill(managedUserID);
  await membershipDialog.getByLabel('Cod funcție').fill('registrator');
  await membershipDialog.getByLabel('Cod unitate').fill('unit-root');
  await membershipDialog.getByLabel('Organizație').fill('EguEducation');
  await membershipDialog.getByLabel('Dată început').fill('2026-01-01');
  await membershipDialog.getByText('Principală: nu').click();
  await page.getByRole('option', { name: 'Principală: da' }).click();
  await membershipDialog.getByText('Activă: nu').click();
  await page.getByRole('option', { name: 'Activă: da' }).click();
  const membershipCreated = page.waitForResponse((response) => new URL(response.url()).pathname === '/api/admin/memberships' && response.request().method() === 'POST');
  await membershipDialog.getByRole('button', { name: 'Salvează' }).click();
  expect((await membershipCreated).status()).toBe(201);

  await page.getByRole('button', { name: 'Permisiuni roluri' }).click();
  await page.getByRole('button', { name: 'Adaugă sau actualizează' }).click();
  const permissionDialog = page.getByRole('dialog', { name: 'Configurare instituție' });
  await permissionDialog.getByLabel('Cod rol').fill('registrator');
  await permissionDialog.getByLabel('Cod permisiune').fill('registratura.read');
  await permissionDialog.getByText('Atribuit: nu').click();
  await page.getByRole('option', { name: 'Atribuit: da' }).click();
  const permissionAssigned = page.waitForResponse((response) => new URL(response.url()).pathname === '/api/admin/role-permissions' && response.request().method() === 'POST');
  await permissionDialog.getByRole('button', { name: 'Salvează' }).click();
  expect((await permissionAssigned).status()).toBe(201);

  // Keep approval authority independent from Registratură read access so the
  // later revocation proves least privilege instead of removing every useful
  // permission from the second actor.
  await page.getByRole('button', { name: 'Adaugă sau actualizează' }).click();
  const workflowPermissionDialog = page.getByRole('dialog', { name: 'Configurare instituție' });
  await workflowPermissionDialog.getByLabel('Cod rol').fill('workflow_admin');
  await workflowPermissionDialog.getByLabel('Cod permisiune').fill('workflow.manage');
  await workflowPermissionDialog.getByText('Atribuit: nu').click();
  await page.getByRole('option', { name: 'Atribuit: da' }).click();
  const workflowPermissionAssigned = page.waitForResponse((response) => new URL(response.url()).pathname === '/api/admin/role-permissions' && response.request().method() === 'POST');
  await workflowPermissionDialog.getByRole('button', { name: 'Salvează' }).click();
  expect((await workflowPermissionAssigned).status()).toBe(201);

  await page.getByRole('button', { name: 'Atribuiri roluri' }).click();
  await page.getByRole('button', { name: 'Adaugă sau actualizează' }).click();
  const workflowRoleDialog = page.getByRole('dialog', { name: 'Configurare instituție' });
  await workflowRoleDialog.getByLabel('ID utilizator').fill(approverID);
  await workflowRoleDialog.getByLabel('Cod rol').fill('workflow_admin');
  await workflowRoleDialog.getByText('Atribuit: nu').click();
  await page.getByRole('option', { name: 'Atribuit: da' }).click();
  const workflowRoleAssigned = page.waitForResponse((response) => new URL(response.url()).pathname === '/api/admin/role-assignments' && response.request().method() === 'POST');
  await workflowRoleDialog.getByRole('button', { name: 'Salvează' }).click();
  expect((await workflowRoleAssigned).status()).toBe(201);

  await page.getByRole('button', { name: 'Adaugă sau actualizează' }).click();
  const roleDialog = page.getByRole('dialog', { name: 'Configurare instituție' });
  await roleDialog.getByLabel('ID utilizator').fill(approverID);
  await roleDialog.getByLabel('Cod rol').fill('registrator');
  await roleDialog.getByText('Atribuit: nu').click();
  await page.getByRole('option', { name: 'Atribuit: da' }).click();
  const roleAssigned = page.waitForResponse((response) => new URL(response.url()).pathname === '/api/admin/role-assignments' && response.request().method() === 'POST');
  await roleDialog.getByRole('button', { name: 'Salvează' }).click();
  expect((await roleAssigned).status()).toBe(201);
  const grantedAuthorizationVersion = databaseScalar(`select version::text from app_tenant_authorization_versions where tenant_code='tenant-egueducation' and user_id='${approverID}'`);
  expect(grantedAuthorizationVersion).toMatch(/^[1-9][0-9]*$/);

  // A new OIDC transaction is required after an authorization change. The
  // refreshed user B token must expose its newly granted navigation and API.
  await approverPage.getByRole('button', { name: 'Deconectare' }).click();
  await expect(approverPage.getByRole('button', { name: 'Autentificare' }).last()).toBeVisible();
  const grantedApproverToken = await authenticated(approverPage, approverIdentifier, approverOTP, 'http://localhost:4174');
  const grantedApproverMe = await api<{ authz_version: number; permissions: string[] }>(approverPage, grantedApproverToken, '/api/me');
  expect(grantedApproverMe.status).toBe(200);
  expect(String(grantedApproverMe.body.authz_version)).toBe(grantedAuthorizationVersion);
  expect(grantedApproverMe.body.permissions).toContain('registratura.read');
  await expect(approverPage.getByRole('link', { name: 'Registratură' })).toBeVisible();
  await approverPage.getByRole('link', { name: 'Registratură' }).click();
  await expect(approverPage.getByRole('region', { name: 'Registratură', exact: true })).toBeVisible();
  expect((await api(approverPage, grantedApproverToken, '/api/registratura/documents?page=1&pageSize=1')).status).toBe(200);

  // Revocation uses the same React editor. It deliberately writes an explicit
  // assigned=false event rather than deleting identity data; the fresh token
  // must carry a higher authorization version and receive a backend 403.
  await page.getByRole('button', { name: 'Adaugă sau actualizează' }).click();
  const revokeDialog = page.getByRole('dialog', { name: 'Configurare instituție' });
  await revokeDialog.getByLabel('ID utilizator').fill(approverID);
  await revokeDialog.getByLabel('Cod rol').fill('registrator');
  const revokeResponse = page.waitForResponse((response) => new URL(response.url()).pathname === '/api/admin/role-assignments' && response.request().method() === 'POST');
  await revokeDialog.getByRole('button', { name: 'Salvează' }).click();
  expect((await revokeResponse).status()).toBe(201);
  const revokedAuthorizationVersion = databaseScalar(`select version::text from app_tenant_authorization_versions where tenant_code='tenant-egueducation' and user_id='${approverID}'`);
  expect(Number(revokedAuthorizationVersion)).toBeGreaterThan(Number(grantedAuthorizationVersion));

  // Revocation is effective immediately at the API boundary; it does not wait
  // for token expiry or for the user to sign in again.
  const staleGrantedToken = await api<{ code: string }>(approverPage, grantedApproverToken, '/api/registratura/documents?page=1&pageSize=1');
  expect(staleGrantedToken.status).toBe(401);
  expect(staleGrantedToken.body).toMatchObject({ code: 'token_authorization_stale' });

  await approverPage.getByRole('button', { name: 'Deconectare' }).click();
  await expect(approverPage.getByRole('button', { name: 'Autentificare' }).last()).toBeVisible();
  const revokedApproverToken = await authenticated(approverPage, approverIdentifier, approverOTP, 'http://localhost:4174');
  const revokedApproverMe = await api<{ authz_version: number; permissions: string[] }>(approverPage, revokedApproverToken, '/api/me');
  expect(revokedApproverMe.status).toBe(200);
  expect(String(revokedApproverMe.body.authz_version)).toBe(revokedAuthorizationVersion);
  expect(revokedApproverMe.body.permissions).toContain('workflow.manage');
  expect(revokedApproverMe.body.permissions).not.toContain('registratura.read');
  await expect(approverPage.getByRole('link', { name: 'Registratură' })).toHaveCount(0);
  expect((await api(approverPage, revokedApproverToken, '/api/registratura/documents?page=1&pageSize=1')).status).toBe(403);

  // Both module routes are rendered by the real React application and backed
  // by authenticated, tenant-scoped HTTP handlers. Storage/OCR ingestion is
  // intentionally not faked here; it has its own service integration job.
  await page.goto('/flux-documente');
  await expect(page.getByRole('region', { name: 'Flux documente' })).toBeVisible();
  expect((await api(page, token, '/api/workflow/dashboard')).status).toBe(200);
  expect((await api(page, token, '/api/workflow/tasks?page=1&pageSize=20')).status).toBe(200);
  await page.goto('/earchiva');
  await expect(page.getByRole('region', { name: 'eArhivă' })).toBeVisible();
  expect((await api(page, token, '/api/earchiva/dashboard')).status).toBe(200);
  expect((await api(page, token, '/api/earchiva/documents?page=1&pageSize=20')).status).toBe(200);

  await page.goto('/registratura');
  await expect(page.getByRole('region', { name: 'Registratură', exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Intrare', exact: true })).toBeEnabled();
  await expect(page.getByRole('button', { name: 'Ieșire', exact: true })).toBeEnabled();

  const actorID = databaseScalar("select id::text from app_users where sub='oidc-browser-fixture-subject'");
  const departmentID = databaseScalar(`
    insert into registratura_departments(tenant_code,institution_id,name,description,role_tag,active)
    values ('tenant-egueducation','inst-001','System E2E ${marker}','System E2E workflow department','registrator',true)
    returning id::text
  `);
  databaseExec(`
    insert into registratura_user_departments(tenant_code,institution_id,user_id,department_id,is_primary)
    values ('tenant-egueducation','inst-001','${actorID}','${departmentID}',true)
    on conflict (tenant_code,user_id,department_id) do update set is_primary=true
  `);

  // Create through the real PrimeReact dialog, not an API shortcut. The
  // response and the later SQL assertion prove the complete UI -> generated
  // client -> handler -> database path.
  await page.getByRole('button', { name: 'Intrare', exact: true }).click();
  const createDialog = page.getByRole('dialog', { name: 'Înregistrare intrare' });
  await createDialog.getByRole('textbox', { name: 'Conținut', exact: true }).fill(marker);
  await createDialog.getByRole('textbox', { name: 'Emitent', exact: true }).fill('System E2E Sender');
  await createDialog.getByRole('textbox', { name: 'Destinatar', exact: true }).fill('System E2E Recipient');
  await createDialog.getByRole('button', { name: `System E2E ${marker}` }).click();
  await createDialog.getByLabel('Număr extern').fill(marker);
  await createDialog.getByLabel('Activitate').fill('system-e2e');
  const createResponse = page.waitForResponse((response) => new URL(response.url()).pathname === '/api/registratura/documents' && response.request().method() === 'POST');
  await createDialog.getByRole('button', { name: 'Salvează' }).click();
  const persistedResponse = await createResponse;
  const persistedBody = await persistedResponse.text();
  expect(persistedResponse.status(), `create document response: ${persistedBody}`).toBe(201);
  const created = { status: persistedResponse.status(), body: JSON.parse(persistedBody) as CreatedDocument };
  expect(created.body).toMatchObject({ subject: marker, institution_id: 'inst-001', status: 'INCOMING', workflow_version: 1 });
  const createdDetailDialog = page.getByRole('dialog', { name: new RegExp(`Detalii document ${created.body.registry_number}`) });
  await expect(createdDetailDialog).toBeVisible();
  const attachmentUpload = page.waitForResponse((response) => new URL(response.url()).pathname === `/api/registratura/documents/${created.body.id}/attachments/upload` && response.request().method() === 'POST');
  await createdDetailDialog.locator('input[type="file"]').setInputFiles({
    name: 'document-scanat-system-e2e.pdf',
    mimeType: 'application/pdf',
    buffer: Buffer.from('%PDF-1.7\n1 0 obj<</Type/Catalog>>endobj\n%%EOF\n'),
  });
  await createdDetailDialog.getByRole('button', { name: 'Încarcă', exact: true }).click();
  expect((await attachmentUpload).status()).toBe(201);
  expect(databaseScalar(`select status || '|' || scan_status || '|' || storage_state from registratura_document_attachments where document_id='${created.body.id}'`)).toBe('ready|clean|ready');
  await page.getByRole('button', { name: 'Închide' }).last().click();

  const outgoingSubject = `SYSTEM-OUTGOING-${Date.now()}`;
  await page.getByRole('button', { name: 'Ieșire', exact: true }).click();
  const outgoingDialog = page.getByRole('dialog', { name: 'Înregistrare ieșire' });
  await outgoingDialog.getByRole('textbox', { name: 'Conținut', exact: true }).fill(outgoingSubject);
  await outgoingDialog.getByRole('textbox', { name: 'Emitent', exact: true }).fill('Scoala E2E');
  await outgoingDialog.getByRole('textbox', { name: 'Destinatar', exact: true }).fill('Inspectorat E2E');
  await outgoingDialog.getByRole('button', { name: `System E2E ${marker}` }).click();
  const outgoingResponse = page.waitForResponse((response) => new URL(response.url()).pathname === '/api/registratura/documents' && response.request().method() === 'POST');
  await outgoingDialog.getByRole('button', { name: 'Salvează' }).click();
  const outgoingHTTP = await outgoingResponse;
  expect(outgoingHTTP.status()).toBe(201);
  const outgoingDocument = await outgoingHTTP.json() as CreatedDocument & { direction: string };
  expect(outgoingDocument).toMatchObject({ subject: outgoingSubject, direction: 'iesire', institution_id: 'inst-001' });
  await page.getByRole('button', { name: 'Închide' }).last().click();

  const batchSubject = `SYSTEM-MULTI-${Date.now()}`;
  await page.getByRole('button', { name: 'MULTIPLU', exact: true }).click();
  const batchDialog = page.getByRole('dialog', { name: 'Înregistrare MULTIPLU' });
  await batchDialog.getByLabel('Număr documente').fill('20');
  await batchDialog.getByLabel('Conținut opțional').fill(batchSubject);
  const batchResponse = page.waitForResponse((response) => new URL(response.url()).pathname === '/api/registratura/documents/batch' && response.request().method() === 'POST');
  await batchDialog.getByRole('button', { name: 'Salvează' }).click();
  const batchHTTP = await batchResponse;
  expect(batchHTTP.status()).toBe(201);
  const batchDocuments = await batchHTTP.json() as CreatedDocument[];
  expect(batchDocuments).toHaveLength(20);
  expect(batchDocuments).toEqual(expect.arrayContaining([expect.objectContaining({ subject: batchSubject, document_type: 'MULTIPLU' })]));
  await page.getByRole('button', { name: 'Închide' }).last().click();
  expect(databaseScalar(`select count(*)::text from registratura_documents where subject='${outgoingSubject}' and direction='iesire'`)).toBe('1');
  expect(databaseScalar(`select count(*)::text from registratura_documents where subject='${batchSubject}' and document_type='MULTIPLU'`)).toBe('20');

  // The 20 newly-created batch rows legitimately move the earlier outgoing
  // document off page one. Locate it through the actual server-side header
  // filter before exercising its action column.
  const registryFilter = page.getByLabel('Filtru coloană Nr. Doc');
  const outgoingFilterResponse = page.waitForResponse((response) => {
    const url = new URL(response.url());
    return url.pathname === '/api/registratura/documents'
      && url.searchParams.get('filter.registry_number') === outgoingDocument.registry_number;
  });
  await registryFilter.fill(outgoingDocument.registry_number);
  await registryFilter.press('Enter');
  expect((await outgoingFilterResponse).status()).toBe(200);
  await page.getByRole('button', { name: `Editează ${outgoingDocument.registry_number}` }).click();
  const editDialog = page.getByRole('dialog', { name: new RegExp(`Editare document ${outgoingDocument.registry_number}`) });
  const editedSubject = `${outgoingSubject}-EDITAT`;
  await editDialog.getByLabel('Subiect document').fill(editedSubject);
  await editDialog.getByLabel('Notă modificare').fill('Corecție verificată prin testul complet.');
  const updateResponse = page.waitForResponse((response) => new URL(response.url()).pathname === `/api/registratura/documents/${outgoingDocument.id}` && response.request().method() === 'PATCH');
  await editDialog.getByRole('button', { name: 'Salvează documentul' }).click();
  expect((await updateResponse).status()).toBe(200);
  expect(databaseScalar(`select subject from registratura_documents where id='${outgoingDocument.id}'`)).toBe(editedSubject);
  await page.getByRole('button', { name: 'Închide' }).last().click();

  await page.getByRole('button', { name: `Anulează ${outgoingDocument.registry_number}` }).click();
  const cancelDialog = page.getByRole('dialog', { name: new RegExp(`Anulare document ${outgoingDocument.registry_number}`) });
  await cancelDialog.getByLabel('Motiv anulare').fill('Document anulat în verificarea E2E completă.');
  const cancelResponse = page.waitForResponse((response) => new URL(response.url()).pathname === `/api/registratura/documents/${outgoingDocument.id}/cancel` && response.request().method() === 'POST');
  await cancelDialog.getByRole('button', { name: 'Anulează documentul' }).click();
  expect((await cancelResponse).status()).toBe(200);
  expect(databaseScalar(`select status || '|' || cancellation_reason from registratura_documents where id='${outgoingDocument.id}'`)).toBe('ANULAT|Document anulat în verificarea E2E completă.');
  await page.getByRole('button', { name: 'Închide' }).last().click();

  const printResponse = page.waitForResponse((response) => new URL(response.url()).pathname === `/api/registratura/documents/${outgoingDocument.id}/print-pdf`);
  await page.getByRole('button', { name: `PDF ${outgoingDocument.registry_number}` }).click();
  const printed = await printResponse;
  expect(printed.status()).toBe(200);
  expect(printed.headers()['content-type']).toContain('application/pdf');

  const resetOutgoingFilter = page.waitForResponse((response) => {
    const url = new URL(response.url());
    return url.pathname === '/api/registratura/documents'
      && !url.searchParams.has('filter.registry_number');
  });
  await registryFilter.fill('');
  await registryFilter.press('Enter');
  expect((await resetOutgoingFilter).status()).toBe(200);

  await page.getByRole('button', { name: 'Exportă registrul în PDF' }).click();
  const exportDialog = page.getByRole('dialog', { name: 'Selectați intervalul de date pentru export PDF' });
  const exportResponse = page.waitForResponse((response) => new URL(response.url()).pathname === '/api/registratura/documents/export-pdf' && response.request().method() === 'POST');
  await exportDialog.getByRole('button', { name: 'Generează PDF' }).click();
  const exported = await exportResponse;
  expect(exported.status()).toBe(200);
  expect(exported.headers()['content-type']).toContain('application/pdf');

  const sortedResponse = page.waitForResponse((response) => {
    const url = new URL(response.url());
    return url.pathname === '/api/registratura/documents' && url.searchParams.get('sort') === 'registry_number';
  });
  await page.getByRole('button', { name: 'Sortează după Nr. Doc' }).click();
  expect(new URL((await sortedResponse).url()).searchParams.get('direction')).toBe('asc');
  const pageTwoResponse = page.waitForResponse((response) => {
    const url = new URL(response.url());
    return url.pathname === '/api/registratura/documents' && url.searchParams.get('page') === '2';
  });
  await page.getByRole('button', { name: 'Pagina 2' }).click();
  expect(new URL((await pageTwoResponse).url()).searchParams.get('pageSize')).toBe('20');
  await page.getByRole('button', { name: 'Pagina 1' }).click();

  // The React route renders the document returned by the real server, then
  // its server-side filtering/sorting/pagination contract is exercised.
  await page.reload();
  await page.getByRole('button', { name: 'Deschide căutarea' }).click();
  await page.getByLabel('Conținut', { exact: true }).fill(marker);
  await page.getByRole('button', { name: 'Caută documente' }).click();
  await expect(page.getByText(marker)).toBeVisible();
  const filtered = await api<{ items: CreatedDocument[]; total: number }>(page, token, `/api/registratura/documents?filter.subject=${encodeURIComponent(marker)}&sort=registry_number&direction=asc&page=1&pageSize=20`);
  expect(filtered.status).toBe(200);
  expect(filtered.body.total).toBe(1);
  expect(filtered.body.items[0].id).toBe(created.body.id);

  // This independent SQL assertion proves persistence and tenant binding at
  // the database boundary, not only a successful HTTP response.
  const persisted = databaseScalar(`select tenant_code || '|' || institution_id || '|' || subject from registratura_documents where id='${created.body.id.replace(/'/g, "''")}'`);
  expect(persisted).toBe(`tenant-egueducation|inst-001|${marker}`);

  // The following mutations still pass through the authenticated router,
  // RBAC, RLS, optimistic-write predicate and immutable event table.
  const action = (body: Record<string, unknown>) => api<CreatedDocument | { code: string; current_version?: number }>(page, token, `/api/registratura/documents/${created.body.id}/workflow-actions`, {
    method: 'POST', body: JSON.stringify(body),
  });
  const departmentAssigned = await action({ action: 'assign_department', department_id: departmentID, note: '', expected_version: 1 });
  expect(departmentAssigned.status).toBe(200);
  expect(departmentAssigned.body).toMatchObject({ status: 'ALOCAT_COMPARTIMENT', workflow_version: 2 });

  // A repeated write with the pre-transition version must fail at the real
  // aggregate's optimistic predicate, not merely in the React client.
  const stale = await action({ action: 'assign_department', department_id: departmentID, note: '', expected_version: 1 });
  expect(stale.status).toBe(409);
  expect(stale.body).toMatchObject({ code: 'stale_workflow_version' });

  const userAssigned = await action({ action: 'assign_user', user_id: actorID, note: '', expected_version: 2 });
  expect(userAssigned.status).toBe(200);
  expect(userAssigned.body).toMatchObject({ status: 'IN_LUCRU', workflow_version: 3, workflow_assignment: { user_id: actorID } });

  // The server denies a self-approval attempt even for the tenant's
  // super-admin: workflow authorization is an action-level RBAC invariant.
  const selfApproval = await action({ action: 'send_for_approval', user_id: actorID, note: '', expected_version: 3 });
  expect(selfApproval.status).toBe(422);
  expect(selfApproval.body).toMatchObject({ code: 'workflow_self_approval_forbidden' });

  const approvalSent = await action({ action: 'send_for_approval', user_id: approverID, note: 'Independent approver required.', expected_version: 3 });
  expect(approvalSent.status).toBe(200);
  expect(approvalSent.body).toMatchObject({ status: 'FLUX_APROBARE', workflow_version: 4, workflow_assignment: { target_approver_id: approverID } });
  const wrongApprover = await action({ action: 'approve', note: '', expected_version: 4 });
  expect(wrongApprover.status).toBe(403);
  expect(wrongApprover.body).toMatchObject({ code: 'workflow_target_approver_required' });
  const approved = await api<CreatedDocument>(approverPage, revokedApproverToken, `/api/registratura/documents/${created.body.id}/workflow-actions`, {
    method: 'POST', body: JSON.stringify({ action: 'approve', note: 'Aprobat de utilizatorul independent.', expected_version: 4 }),
  });
  expect(approved.status).toBe(200);
  expect(approved.body).toMatchObject({ status: 'FINALIZAT', workflow_version: 5 });
  expect(databaseScalar(`
    select status || '|' || workflow_version::text || '|' || count(*)::text
    from registratura_documents d
    join registratura_document_workflow_events e on e.document_id=d.id
    where d.id='${created.body.id}'
    group by status, workflow_version
  `)).toBe('FINALIZAT|5|4');
  await expect.poll(() => databaseScalar(`select status || '|' || (payload->>'workflow_version') from registratura_archive_outbox where document_id='${created.body.id}'`), { timeout: 30_000 }).toBe('delivered|5');
  expect(databaseScalar(`select status || '|' || source_system from archive_documents where external_reference='${created.body.id}'`)).toBe('queued|registratura');
  expect(databaseScalar(`select status from archive_ingestion_jobs where document_id=(select id from archive_documents where external_reference='${created.body.id}')`)).toBe('pending');

  await page.goto('/flux-documente');
  await expect(page.getByRole('region', { name: 'Flux documente' })).toBeVisible();
  const fluxPipeline = await api<{ items: CreatedDocument[]; total: number }>(page, token, `/api/registratura/flux/pipeline?filter.continut=${encodeURIComponent(marker)}&sort=subject&direction=asc&page=1&pageSize=20`);
  expect(fluxPipeline.status).toBe(200);
  expect(fluxPipeline.body.total).toBe(1);
  expect(fluxPipeline.body.items[0]).toMatchObject({ id: created.body.id, status: 'FINALIZAT', workflow_version: 5 });

  // A third independent OIDC browser authenticates against the Balotesti
  // tenant server and creates its own persisted document. Tenant A cannot
  // read that identifier or replay its token against tenant B; the tenant-B
  // token is symmetrically rejected by tenant A.
  const balotestiContext = await browser.newContext({ baseURL: 'http://localhost:4175' });
  const balotestiPage = await balotestiContext.newPage();
  const balotestiToken = await authenticated(balotestiPage, balotestiIdentifier, balotestiOTP, 'http://localhost:4175');
  const balotestiMe = await api<{ tenant_code: string; institution_id: string; permissions: string[] }>(balotestiPage, balotestiToken, '/api/me');
  expect(balotestiMe.status).toBe(200);
  expect(balotestiMe.body).toMatchObject({ tenant_code: 'tenant-balotesti', institution_id: 'inst-balotesti' });
  expect(balotestiMe.body.permissions).toContain('registratura.manage');
  const balotestiRegistries = await api<Array<{ id: number }>>(balotestiPage, balotestiToken, '/api/registratura/registre');
  expect(balotestiRegistries.status).toBe(200);
  const balotestiDocument = await api<CreatedDocument>(balotestiPage, balotestiToken, '/api/registratura/documents', {
    method: 'POST',
    body: JSON.stringify({
      registru_id: balotestiRegistries.body[0].id,
      subject: `${marker} tenant B`,
      document_type: 'DOCUMENT',
      direction: 'intrare',
      status: 'INCOMING',
      correspondent: 'System E2E Balotesti',
      assigned_to: 'Tenant B',
      confidentiality: 'normal',
      summary: 'Created by the independently authenticated tenant-B browser.',
      department_ids: [],
    }),
  });
  expect(balotestiDocument.status).toBe(201);
  expect(balotestiDocument.body.institution_id).toBe('inst-balotesti');
  const hiddenTenantBRecord = await api<unknown>(page, token, `/api/registratura/documents/${balotestiDocument.body.id}`);
  expect(hiddenTenantBRecord.status).toBe(404);
  const crossTenantCreate = await page.request.post('http://127.0.0.1:8082/api/registratura/documents', {
    headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
    data: { registru_id: balotestiRegistries.body[0].id, subject: `${marker} forbidden`, document_type: 'document', direction: 'intrare', status: 'INCOMING' },
  });
  expect(crossTenantCreate.status()).toBe(401);
  expect(databaseScalar(`select count(*)::text from registratura_documents where id='${balotestiDocument.body.id}'`, balotestiScope)).toBe('1');

  // A token minted for tenant A cannot be replayed against tenant B's
  // hostname. This calls the real backend directly because Vite correctly
  // preserves the local browser host for normal application traffic.
  const denied = await page.request.get('http://127.0.0.1:8082/api/registratura/documents?page=1&pageSize=20', {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(denied.status()).toBe(401);
  const reverseDenied = await page.request.get('http://127.0.0.1:8080/api/registratura/documents?page=1&pageSize=20', {
    headers: { Authorization: `Bearer ${balotestiToken}` },
  });
  expect(reverseDenied.status()).toBe(401);
  expect(databaseScalar(`select count(*) from registratura_documents where id='${created.body.id.replace(/'/g, "''")}' and tenant_code='tenant-balotesti'`)).toBe('0');

  const passkeys = await api<unknown[]>(page, token, '/api/passkeys');
  expect(passkeys.status).toBe(200);
  expect(passkeys.body).toEqual([]);

  // Chromium's virtual CTAP2 authenticator performs a real WebAuthn ceremony:
  // the React profile registers the credential, the backend verifies and
  // persists its public key, and a fresh OIDC flow signs an assertion with it.
  const cdp = await page.context().newCDPSession(page);
  await cdp.send('WebAuthn.enable');
  const { authenticatorId } = await cdp.send('WebAuthn.addVirtualAuthenticator', {
    options: {
      protocol: 'ctap2',
      ctap2Version: 'ctap2_1',
      transport: 'internal',
      hasResidentKey: true,
      hasUserVerification: true,
      isUserVerified: true,
      automaticPresenceSimulation: true,
    },
  });
  await page.goto('/profil');
  const addPasskey = page.getByRole('button', { name: 'Adaugă cheie' });
  await expect(addPasskey).toBeEnabled();
  await addPasskey.click();
  await expect(page.getByText('Cheia de acces a fost înregistrată.')).toBeVisible();
  await expect(page.getByText('Cheie de acces browser')).toBeVisible();
  expect(databaseScalar(`select count(*)::text from app_passkeys where user_id='${actorID}'`)).toBe('1');

  await page.getByRole('button', { name: 'Deconectare' }).click();
  await expect(page.getByRole('button', { name: 'Autentificare' }).last()).toBeVisible();
  const passkeyToken = await authenticatedWithPasskey(page);
  const passkeyClaims = jwtPayload(passkeyToken);
  expect(passkeyClaims.amr).toEqual(expect.arrayContaining(['passkey']));
  expect(passkeyClaims.tenant_code).toBe('tenant-egueducation');
  expect((await api(page, passkeyToken, '/api/me')).status).toBe(200);
  expect(databaseScalar(`select (last_used_at is not null)::text from app_passkeys where user_id='${actorID}'`)).toBe('true');
  await cdp.send('WebAuthn.removeVirtualAuthenticator', { authenticatorId });

  await page.getByRole('button', { name: 'Deconectare' }).click();
  await expect(page.getByRole('button', { name: 'Autentificare' }).last()).toBeVisible();
  await approverPage.getByRole('button', { name: 'Deconectare' }).click();
  await expect(approverPage.getByRole('button', { name: 'Autentificare' }).last()).toBeVisible();
  await approverContext.close();
  await balotestiPage.getByRole('button', { name: 'Deconectare' }).click();
  await expect(balotestiPage.getByRole('button', { name: 'Autentificare' }).last()).toBeVisible();
  await balotestiContext.close();
  const afterLogout = await page.request.get('http://127.0.0.1:8080/api/oidc/token');
  expect(afterLogout.status()).toBe(405);
});
