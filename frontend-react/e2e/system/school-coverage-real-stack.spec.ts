import { expect, test, type Page } from '@playwright/test';
import { execFileSync } from 'node:child_process';

/* No page.route in this file: every browser mutation below crosses React, the
 * production OIDC handler, Go authorization and PostgreSQL. SQL is limited to
 * deterministic role setup and postcondition assertions. */
const origin = 'http://localhost:4173';
const fixture = { identifier: 'oidc.browser.fixture@example.test', otp: '173829', subject: 'oidc-browser-fixture-subject' };
const marker = `SCHOOL-COVERAGE-E2E-${process.env.GITHUB_RUN_ID ?? 'local'}-${Date.now()}`;
const hasStack = Boolean(process.env.TEST_DATABASE_URL && process.env.DATABASE_URL);
test.skip(!hasStack, 'Requires the real PostgreSQL/OIDC system topology.');

function sql(statement: string): string {
  const url = process.env.TEST_DATABASE_URL;
  if (!url) throw new Error('TEST_DATABASE_URL is required.');
  return execFileSync('psql', ['--no-psqlrc', '--tuples-only', '--no-align', '--quiet', url, '-c', `set app.tenant_id='tenant-egueducation'; set app.institution_id='inst-001'; set app.is_super_admin='true'; ${statement}`], { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] }).trim();
}

async function login(page: Page): Promise<void> {
  await page.goto(origin);
  await page.getByRole('button', { name: 'Autentificare' }).last().click();
  await expect(page).toHaveURL(/\/api\/oidc\/authorize/);
  await page.getByRole('button', { name: /SMS/ }).click();
  await page.getByLabel('Utilizator, email sau numar de telefon').fill(fixture.identifier);
  await page.getByLabel('Canal OTP').selectOption('sms');
  await page.getByRole('button', { name: 'Trimite codul' }).click();
  const boxes = page.locator('.otp-box'); await expect(boxes).toHaveCount(6);
  await boxes.first().click(); await boxes.first().pressSequentially(fixture.otp);
  await page.getByRole('button', { name: 'Verifica codul' }).click();
  const consent = page.getByRole('button', { name: 'Accepta si continua' }); if (await consent.count()) await consent.click();
  await expect(page).toHaveURL(`${origin}/`); await expect(page.getByRole('button', { name: 'Deconectare' })).toBeVisible();
}

async function choose(page: Page, label: string, value: string): Promise<void> {
  const control = page.getByLabel(label, { exact: true });
  if (await control.getAttribute('role') === 'combobox') {
    await control.click(); const list = page.locator('[role="listbox"]:visible').last(); await expect(list).toBeVisible(); await list.getByRole('option', { name: value, exact: true }).click();
  } else await control.fill(value);
}

async function createRoot(page: Page, route: string, endpoint: string, values: Record<string, string>): Promise<{ id: string }> {
  await page.goto(route); await page.getByLabel('Adaugă înregistrare').click();
  const dialog = page.getByRole('dialog'); await expect(dialog).toBeVisible();
  for (const [label, value] of Object.entries(values)) await choose(dialog, label, value);
  const response = page.waitForResponse((item) => item.request().method() === 'POST' && new URL(item.url()).pathname === endpoint);
  await dialog.getByRole('button', { name: 'Salvează' }).click(); expect((await response).status()).toBe(201);
  return response.then((item) => item.json()) as Promise<{ id: string }>;
}

async function openDetails(page: Page, title: string): Promise<void> {
  const row = page.getByText(title, { exact: true }).locator('xpath=ancestor::tr[1]'); await expect(row).toBeVisible();
  await row.getByLabel('Acțiuni înregistrare').click(); const menu = page.locator('[role="menu"]:visible').last(); await menu.getByRole('button', { name: 'Detalii', exact: true }).click();
  await expect(page.getByRole('dialog').last()).toBeVisible(); await page.keyboard.press('Escape');
}

async function createRelated(page: Page, add: string, endpoint: string, values: Record<string, string>): Promise<void> {
  await page.getByLabel(add, { exact: true }).click(); const dialog = page.getByRole('dialog'); await expect(dialog).toBeVisible();
  for (const [label, value] of Object.entries(values)) await choose(dialog, label, value);
  const response = page.waitForResponse((item) => item.request().method() === 'POST' && new URL(item.url()).pathname === endpoint);
  await dialog.getByRole('button', { name: 'Salvează' }).click(); expect((await response).status()).toBe(201);
}

async function createPersonnel(page: Page, name: string): Promise<{ id: string }> {
  await page.goto('/scoala/personnel'); await page.getByLabel('Adaugă înregistrare').click(); await expect(page).toHaveURL(/\/scoala\/personnel\/wizard$/);
  await page.getByLabel('Nume complet').fill(name); await page.getByLabel('Funcție').fill('Profesor'); await page.getByRole('button', { name: 'Continuă' }).click();
  await page.getByRole('button', { name: 'Continuă' }).click(); await page.getByRole('button', { name: 'Continuă' }).click();
  const response = page.waitForResponse((item) => item.request().method() === 'POST' && new URL(item.url()).pathname === '/api/education/personnel/records');
  await page.getByRole('button', { name: 'Salvează' }).click(); expect((await response).status()).toBe(201); return response.then((item) => item.json()) as Promise<{ id: string }>;
}

async function createEvaluation(page: Page, employeeCode: string, name: string): Promise<{ id: string }> {
  await page.goto('/scoala/evaluations'); await page.getByLabel('Adaugă înregistrare').click(); await expect(page).toHaveURL(/\/scoala\/personnel\/evaluations-wizard$/);
  await page.getByLabel('Cod angajat').fill(employeeCode); await page.getByLabel('Nume').fill(name); await page.getByLabel('Funcție').fill('Profesor'); await page.getByRole('button', { name: 'Continuă' }).click();
  await page.getByRole('combobox', { name: 'Status' }).click(); const statuses = page.locator('[role="listbox"]:visible').last(); await statuses.getByRole('option', { name: 'draft', exact: true }).click(); await page.getByRole('button', { name: 'Continuă' }).click();
  await page.getByLabel('Evaluator').fill('Director'); await page.getByRole('button', { name: 'Continuă' }).click(); const response = page.waitForResponse((item) => item.request().method() === 'POST' && new URL(item.url()).pathname === '/api/education/evaluations/records'); await page.getByRole('button', { name: 'Salvează' }).click(); expect((await response).status()).toBe(201); return response.then((item) => item.json()) as Promise<{ id: string }>;
}

test('personnel and evaluation children, decisions and compliance are persisted through React', async ({ page }) => {
  const userID = sql(`select id::text from app_users where sub='${fixture.subject}'`);
  sql(`update app_memberships set position_code='director', active=true where user_id='${userID}' and tenant_code='tenant-egueducation'; delete from app_user_platform_roles where user_id='${userID}'; delete from app_user_roles where user_id='${userID}' and tenant_code='tenant-egueducation'; insert into app_user_roles(tenant_code,user_id,role_code) values ('tenant-egueducation','${userID}','director') on conflict do nothing`);
  await login(page);
  const personnelName = `${marker} Profesor`; const personnel = await createPersonnel(page, personnelName); expect(sql(`select count(*)::text from education_personnel where id='${personnel.id}' and institution_id='inst-001'`)).toBe('1');
  await page.goto('/scoala/personnel'); await openDetails(page, personnelName);
  await createRelated(page, 'Adaugă încadrări', `/api/education/personnel/records/${personnel.id}/assignments`, { 'Titlu încadrare': 'Profesor', Tip: 'diriginte', Stare: 'activ', 'Atribuit la': '2026-09-01', 'Ore săptămânale': '18' });
  await page.getByRole('button', { name: 'Documente dosar' }).click(); await createRelated(page, 'Adaugă documente dosar', `/api/education/personnel/records/${personnel.id}/file-documents`, { 'Categorie document': 'cariera', Titlu: 'Contract', 'Nivel confidențialitate': 'confidential', 'Domeniu fișier': 'dosar_personal', 'Emis la': '2026-09-01' });
  await page.getByRole('button', { name: 'Cazuri disciplinare' }).click(); await createRelated(page, 'Adaugă cazuri disciplinare', `/api/education/personnel/records/${personnel.id}/disciplinary-cases`, { 'Tip caz': 'sesizare', Stare: 'deschis', 'Raportat la': '2026-09-01' });
  await page.getByRole('button', { name: 'Evenimente acces' }).click(); await createRelated(page, 'Adaugă evenimente acces', `/api/education/personnel/records/${personnel.id}/access-events`, { Tip: 'consultare', 'Accesat la': '2026-09-01', Operator: 'Director', 'Rol operator': 'director', 'Canal acces': 'digital', Scop: 'verificare' });
  expect(sql(`select count(*)::text from education_personnel_assignments where personnel_id='${personnel.id}'`)).toBe('1'); expect(sql(`select count(*)::text from education_personnel_file_documents where personnel_id='${personnel.id}'`)).toBe('1');

  const evaluation = await createEvaluation(page, `EVAL-${Date.now()}`, personnelName);
  await page.goto('/scoala/evaluations'); await openDetails(page, personnelName);
  await createRelated(page, 'Adaugă autoevaluări', `/api/education/evaluations/records/${evaluation.id}/self-reviews`, { 'Finalizat la': '2026-09-01', 'Tip relatare': 'autoevaluare', Secțiune: 'Predare', Stare: 'draft' });
  await page.getByRole('button', { name: 'Criterii' }).click(); await createRelated(page, 'Adaugă criterii', `/api/education/evaluations/records/${evaluation.id}/criteria`, { Categorie: 'predare', Criteriu: 'Calitate', Maxim: '100', Stare: 'draft' });
  await page.getByRole('button', { name: 'Contestații' }).click(); await createRelated(page, 'Adaugă contestații', `/api/education/evaluations/records/${evaluation.id}/appeals`, { 'Depus de': personnelName, 'Depus la': '2026-09-01', Stare: 'submitted', Motive: 'clarificare' });
  await page.getByRole('button', { name: 'Comunicări rezultat' }).click(); await createRelated(page, 'Adaugă comunicări rezultat', `/api/education/evaluations/records/${evaluation.id}/result-issues`, { 'Tip document': 'fisa_evaluare', Destinatar: personnelName, Canal: 'email', 'Stare livrare': 'pregatit', 'Emis la': '2026-09-01' });
  expect(sql(`select count(*)::text from education_evaluation_criteria where evaluation_id='${evaluation.id}'`)).toBe('1');

  const decision = await createRoot(page, '/scoala/decisions', '/api/education/decisions/records', { 'An școlar': '2026-2027', Organism: 'ca', Titlu: `${marker} Decizie`, Stare: 'draft', Publicare: 'internal', 'Data deciziei': '2026-09-01' });
  const publication = await createRoot(page, '/scoala/compliance', '/api/education/compliance/records', { Domeniu: 'education', 'Tip entitate': 'decision', Entitate: `${marker} conformitate`, Canal: 'website', Stare: 'draft', Anonimizare: 'not_required' });
  expect(sql(`select count(*)::text from education_decisions where id='${decision.id}'`)).toBe('1'); expect(sql(`select count(*)::text from education_publications where id='${publication.id}'`)).toBe('1');
});

test('cockpit routes perform a real RBAC gate before requesting role metrics', async ({ page }) => {
  const userID = sql(`select id::text from app_users where sub='${fixture.subject}'`); sql(`update app_memberships set position_code='profesor', active=true where user_id='${userID}' and tenant_code='tenant-egueducation'; delete from app_user_roles where user_id='${userID}' and tenant_code='tenant-egueducation'; delete from app_user_platform_roles where user_id='${userID}'`);
  await login(page); let requested = false; page.on('request', (request) => { if (new URL(request.url()).pathname === '/api/education/hr/cockpit') requested = true; });
  await page.goto('/scoala/hr'); await expect(page).toHaveURL(`${origin}/`); expect(requested).toBe(false);
});
