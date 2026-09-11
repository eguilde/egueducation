import { expect, test, type Page } from '@playwright/test';
import { execFileSync } from 'node:child_process';

// This suite intentionally has no route interception.  It drives the deployed
// React/OIDC surface, then performs authenticated browser fetches so the same
// access token reaches the real Go handlers and PostgreSQL/RLS boundary.
// It is selected only by playwright.system.config.ts in CI.
const administratorIdentifier = 'oidc.browser.fixture@example.test';
const administratorOTP = '173829';
const teacherIdentifier = 'oidc.approver.fixture@example.test';
const teacherOTP = '428615';
const marker = `SCHOOL-OPERATIONS-E2E-${process.env.GITHUB_RUN_ID ?? 'local'}-${Date.now()}`;

type Scope = { code: string; institutionID: string };
type ApiResult<T> = { status: number; body: T };
type RecordWithID = { id: string; [key: string]: unknown };

const egueducation: Scope = { code: 'tenant-egueducation', institutionID: 'inst-001' };
const balotesti: Scope = { code: 'tenant-balotesti', institutionID: 'inst-balotesti' };

/** Scope a Select option to the currently open PrimeReact portal. */
async function selectOpenOption(page: Page, name: string | RegExp): Promise<void> {
  const listbox = page.locator('[role="listbox"]:visible').last();
  await expect(listbox).toBeVisible();
  const option = listbox.getByRole('option', { name, exact: typeof name === 'string' });
  await expect(option).toBeAttached();
  const box = await option.boundingBox();
  const listboxBox = await listbox.boundingBox();
  const viewport = page.viewportSize();
  const isInViewport = Boolean(box && listboxBox && viewport
    && box.x >= listboxBox.x && box.x + box.width <= listboxBox.x + listboxBox.width
    && box.y >= listboxBox.y && box.y + box.height <= listboxBox.y + listboxBox.height
    && box.y >= 0 && box.y + box.height <= viewport.height);
  if (isInViewport) {
    await option.click();
  } else {
    const position = Number(await option.getAttribute('aria-posinset'));
    const listboxID = await listbox.getAttribute('id');
    if (!Number.isInteger(position) || position < 1 || !listboxID) throw new Error(`Cannot keyboard-select option ${String(name)}`);
    const trigger = page.locator(`[role="combobox"][aria-controls="${listboxID}"]`);
    await expect(trigger).toBeVisible();
    await trigger.focus();
    await trigger.press('Home');
    for (let index = 1; index < position; index += 1) await trigger.press('ArrowDown');
    await trigger.press('Enter');
  }
  await expect(listbox).toBeHidden();
}

/** Query the paginated owner selector before selecting the server-returned option. */
async function searchAndSelectPortfolioOwner(page: Page, query: string, optionName: string): Promise<void> {
  const response = page.waitForResponse((candidate) => {
    const url = new URL(candidate.url());
    return candidate.request().method() === 'GET'
      && url.pathname === '/api/education/portfolios/eligible-owners'
      && url.searchParams.get('filter.display_name') === query;
  });
  await page.getByLabel('Caută Titular *', { exact: true }).fill(query);
  expect((await response).status()).toBe(200);
  const trigger = page.getByRole('combobox', { name: 'Titular *', exact: true });
  await expect(trigger).toBeEnabled();
  await trigger.click();
  const listbox = page.locator('[role="listbox"]:visible').last();
  const option = listbox.getByRole('option', { name: optionName, exact: true });
  await option.scrollIntoViewIfNeeded();
  await option.click();
  await expect(page.getByLabel('Titular selectat')).toHaveValue(optionName.replace(/ · .*$/, ''));
}

async function clickOpenPopoverAction(page: Page, name: string): Promise<void> {
  const menu = page.locator('[role="menu"]:visible').last();
  await expect(menu).toBeVisible();
  const action = menu.getByRole('button', { name, exact: true });
  await expect(action).toBeVisible();
  await action.focus();
  await action.press('Enter');
}

async function fillVisibleWizardFields(page: Page, values: Record<string, string>): Promise<void> {
  for (const [label, value] of Object.entries(values)) {
    const control = page.getByLabel(label, { exact: true });
    if (!(await control.count()) || !(await control.first().isVisible())) continue;
    if ((await control.first().getAttribute('role')) === 'combobox') {
      await control.first().click();
      await selectOpenOption(page, value);
    } else {
      await control.first().fill(value);
    }
  }
}

function requireSystemEnvironment(): void {
  for (const name of ['TEST_DATABASE_URL', 'DATABASE_URL']) {
    if (!process.env[name]) throw new Error(`${name} is required: this is a non-mocked browser → OIDC → Go → PostgreSQL system suite.`);
  }
}

function sql(value: string, scope: Scope = egueducation, applicationRole = false): string {
  const url = process.env[applicationRole ? 'DATABASE_URL' : 'TEST_DATABASE_URL'];
  if (!url) throw new Error(`${applicationRole ? 'DATABASE_URL' : 'TEST_DATABASE_URL'} is required.`);
  const statement = `set app.tenant_id = '${scope.code}'; set app.institution_id = '${scope.institutionID}'; set app.is_super_admin = '${applicationRole ? 'false' : 'true'}'; ${value}`;
  return execFileSync('psql', ['--no-psqlrc', '--tuples-only', '--no-align', '--quiet', url, '-c', statement], {
    encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'],
  }).trim();
}

function adminSQL(value: string): void { sql(`${value}; select 'ok'`); }

function prepareAdministrator(): string {
  const adminID = sql("select id::text from app_users where sub='oidc-browser-fixture-subject'");
  // The full-system actor deliberately uses the tenant-scoped canary role. A
  // normal tenant administrator must not regain implicit access to pedagogical
  // content merely so this broad functional proof can exercise every module.
  adminSQL(`update app_memberships set position_code='e2e_canary' where user_id='${adminID}' and tenant_code='tenant-egueducation'; delete from app_user_platform_roles where user_id='${adminID}'; delete from app_user_roles where user_id='${adminID}' and tenant_code='tenant-egueducation'; insert into app_user_roles(tenant_code,user_id,role_code) values ('tenant-egueducation','${adminID}','e2e_canary') on conflict do nothing`);
  return adminID;
}

async function authenticate(page: Page, identifier: string, otp: string, origin: string): Promise<string> {
  let token: string | undefined;
  page.on('response', async (response) => {
    if (response.request().method() !== 'POST' || !response.url().includes('/api/oidc/token')) return;
    const body = await response.json().catch(() => undefined) as { access_token?: string } | undefined;
    token ??= body?.access_token;
  });
  await page.goto(`${origin}/`);
  await page.getByRole('button', { name: 'Autentificare' }).last().click();
  await expect(page).toHaveURL(/\/api\/oidc\/authorize/);
  await page.getByRole('button', { name: /SMS/ }).click();
  await page.getByLabel('Utilizator, email sau numar de telefon').fill(identifier);
  await page.getByLabel('Canal OTP').selectOption('sms');
  await page.getByRole('button', { name: 'Trimite codul' }).click();
  const boxes = page.locator('.otp-box');
  await expect(boxes).toHaveCount(6);
  await boxes.first().click();
  await boxes.first().pressSequentially(otp);
  const verify = page.getByRole('button', { name: 'Verifica codul' });
  await expect(verify).toBeEnabled();
  await expect(verify).toBeFocused();
  await verify.click();
  const consent = page.getByRole('button', { name: 'Accepta si continua' });
  if (await consent.count()) await consent.click();
  await expect(page).toHaveURL(`${origin}/`);
  await expect(page.getByRole('button', { name: 'Deconectare' })).toBeVisible();
  await expect.poll(() => token, { message: 'Browser did not observe the OIDC access token.' }).toBeTruthy();
  const claims = JSON.parse(Buffer.from(token!.split('.')[1], 'base64url').toString('utf8')) as { platform_roles?: string[] };
  expect(claims.platform_roles ?? []).toEqual([]);
  const me = await page.evaluate(async (accessToken) => {
    const response = await fetch('/api/me', { headers: { Authorization: `Bearer ${accessToken}` }, credentials: 'include' });
    return { status: response.status, body: await response.json() as { platform_roles?: string[] } };
  }, token!);
  expect(me.status).toBe(200);
  expect(me.body.platform_roles ?? []).toEqual([]);
  return token!;
}

async function api<T>(page: Page, token: string, path: string, init: RequestInit = {}): Promise<ApiResult<T>> {
  return page.evaluate(async ({ token, path, init }) => {
    const headers = new Headers(init.headers);
    headers.set('Authorization', `Bearer ${token}`);
    headers.set('Content-Type', 'application/json');
    const response = await fetch(path, { ...init, headers, credentials: 'include' });
    const text = await response.text();
    return { status: response.status, body: text ? JSON.parse(text) : null };
  }, { token, path, init });
}

async function pdf(page: Page, token: string, path: string): Promise<{ status: number; type: string; size: number }> {
  return page.evaluate(async ({ token, path }) => {
    const response = await fetch(path, { headers: { Authorization: `Bearer ${token}` }, credentials: 'include' });
    return { status: response.status, type: response.headers.get('content-type') ?? '', size: (await response.arrayBuffer()).byteLength };
  }, { token, path });
}

/** Use the actual React root-record dialog or configured wizard; never bypass the UI with fetch. */
async function createRootThroughReact(
  page: Page,
  route: string,
  endpoint: string,
  values: Record<string, string>,
): Promise<RecordWithID> {
  await page.goto(route);
  await expect(page.getByLabel('Adaugă înregistrare')).toBeVisible();
  const created = page.waitForResponse((response) => new URL(response.url()).pathname === endpoint && response.request().method() === 'POST');
  await page.getByLabel('Adaugă înregistrare').click();
  const dialog = page.getByRole('dialog');
  if (await dialog.isVisible({ timeout: 750 }).catch(() => false)) {
    for (const [label, value] of Object.entries(values)) await dialog.getByLabel(label, { exact: true }).fill(value);
    await dialog.getByRole('button', { name: 'Salvează' }).click();
  } else {
    await expect(page).toHaveURL(/(?:\/|-)wizard(?:\?.*)?$/);
    for (;;) {
      await fillVisibleWizardFields(page, values);
      const save = page.getByRole('button', { name: 'Salvează', exact: true });
      if (await save.isVisible()) {
        await save.click();
        break;
      }
      await page.getByRole('button', { name: 'Continuă', exact: true }).click();
    }
  }
  const response = await created;
  expect(response.status()).toBe(201);
  return await response.json() as RecordWithID;
}

async function openReactRootDetails(page: Page, exactTitle: string): Promise<void> {
  const row = page.getByText(exactTitle, { exact: true }).locator('xpath=ancestor::tr[1]');
  await expect(row).toBeVisible();
  await row.getByLabel('Acțiuni înregistrare').click();
  await clickOpenPopoverAction(page, 'Detalii');
  const details = page.locator('[role="dialog"]:visible').last();
  await expect(details).toBeVisible();
  await page.keyboard.press('Escape');
  await expect(details).toBeHidden();
}

async function createManagerialChildThroughReact(
  page: Page,
  addLabel: string,
  endpoint: string,
  values: Record<string, string>,
): Promise<RecordWithID> {
  await page.getByLabel(addLabel).click();
  const dialog = page.getByRole('dialog');
  await expect(dialog).toBeVisible();
  for (const [label, value] of Object.entries(values)) {
    const control = dialog.getByLabel(label, { exact: true });
    if ((await control.getAttribute('role')) === 'combobox') {
      await control.click();
      await selectOpenOption(page, value);
    } else {
      await control.fill(value);
    }
  }
  const created = page.waitForResponse((response) => new URL(response.url()).pathname === endpoint && response.request().method() === 'POST');
  await dialog.getByRole('button', { name: 'Salvează' }).click();
  const response = await created;
  expect(response.status()).toBe(201);
  return await response.json() as RecordWithID;
}

/** Create, configure, approve and publish through the PrimeReact procedure UI. */
async function publishPortfolioProcedureThroughReact(page: Page): Promise<void> {
  const title = `${marker} procedură`;
  const code = `PORT-${marker.slice(-10)}`;
  const sections = ['identificare_profesionala', 'predare_invatare_evaluare', 'activitati_complementare', 'managementul_clasei', 'evolutie_dezvoltare_profesionala'];
  await page.goto('/scoala/portfolios');
  await expect(page.getByRole('heading', { name: 'Proceduri instituționale pentru portofolii' })).toBeVisible();
  await page.getByLabel('Adaugă procedură').click();
  const editor = page.getByRole('dialog', { name: 'Adaugă procedură' });
  await editor.getByLabel('Cod *').fill(code);
  await editor.getByLabel('Titlu *').fill(title);
  await editor.getByLabel('Descriere').fill('Ordinul nr. 3.858/2026');
  const created = page.waitForResponse((response) => new URL(response.url()).pathname === '/api/education/portfolios/procedures' && response.request().method() === 'POST');
  await editor.getByRole('button', { name: 'Salvează' }).click();
  expect((await created).status()).toBe(201);

  const procedureRow = page.getByText(title, { exact: true }).locator('xpath=ancestor::tr[1]');
  await expect(procedureRow).toBeVisible();
  await procedureRow.getByRole('button', { name: `Detalii și reguli pentru ${title}` }).click();
  const details = page.getByRole('dialog', { name: title });
  await details.getByRole('button', { name: 'Editează reguli' }).click();
  const rules = page.getByRole('dialog', { name: new RegExp(`Reguli procedură: ${title}`) });
  for (const [index, section] of sections.entries()) {
    await rules.getByLabel('Adaugă regulă').click();
    const ruleEditor = page.getByRole('dialog', { name: 'Adaugă regulă' });
    await ruleEditor.getByLabel('Cod secțiune').fill(section);
    await ruleEditor.getByLabel('Ordine').fill(String((index + 1) * 10));
    await ruleEditor.getByLabel('Denumire română').fill(section);
    await ruleEditor.getByLabel('Versiunea catalogului').fill('ome-3858-2026-annexa-1-v1');
    await ruleEditor.getByRole('button', { name: 'Aplică' }).click();
  }
  const rulesSaved = page.waitForResponse((response) => new URL(response.url()).pathname.endsWith('/section-rules') && response.request().method() === 'PUT');
  await rules.getByRole('button', { name: 'Salvează reguli' }).click();
  expect((await rulesSaved).status()).toBe(200);
  await expect(rules).toBeHidden();

  const approved = page.waitForResponse((response) => new URL(response.url()).pathname.endsWith('/approve') && response.request().method() === 'POST');
  await details.getByRole('button', { name: 'Aprobă' }).click();
  await details.getByRole('textbox').last().fill(`${marker}-decizie`);
  await details.getByRole('button', { name: 'Aprobă' }).last().click();
  expect((await approved).status()).toBe(200);
  const published = page.waitForResponse((response) => new URL(response.url()).pathname.endsWith('/publish') && response.request().method() === 'POST');
  await details.getByRole('button', { name: 'Publică' }).click();
  await details.getByRole('textbox').last().fill(`${marker}-publicare`);
  await details.getByRole('button', { name: 'Publică' }).last().click();
  expect((await published).status()).toBe(200);
  await expect(details.getByLabel('Stare procedură')).toHaveText('published');
  await details.getByRole('button', { name: 'Închide', exact: true }).click();
}

function expectCreated<T extends RecordWithID>(result: ApiResult<T>): T {
  expect(result.status, JSON.stringify(result.body)).toBe(201);
  expect(result.body.id).toEqual(expect.any(String));
  return result.body;
}

test.beforeAll(requireSystemEnvironment);

test('React creates the managerial/regulation roots; API contract persists child evidence, PDFs, RBAC and RLS', async ({ page, browser }) => {
  const adminID = prepareAdministrator();
  const teacherID = sql("select id::text from app_users where sub='oidc-browser-approver-subject'");
  // Establish the two roles explicitly, rather than assuming bootstrap roles.
  adminSQL(`update app_memberships set position_code='profesor' where user_id='${teacherID}' and tenant_code='tenant-egueducation'; delete from app_user_roles where user_id='${teacherID}' and tenant_code='tenant-egueducation'; delete from app_user_platform_roles where user_id='${teacherID}'; delete from app_user_permissions where user_id='${teacherID}' and tenant_code='tenant-egueducation'`);
  const adminToken = await authenticate(page, administratorIdentifier, administratorOTP, 'http://localhost:4173');

  const dossierTitle = `${marker} dosar managerial`;
  const dossier = await createRootThroughReact(page, '/scoala/managerial', '/api/education/managerial/records', {
    'An școlar': '2026-2027', 'Tip dosar': 'director_portfolio', Titlu: dossierTitle, Stare: 'draft', Responsabil: 'Director E2E', Termen: '2026-12-20', Rezumat: marker,
  });
  await openReactRootDetails(page, dossierTitle);
  const document = await createManagerialChildThroughReact(page, 'Adaugă documente dosar', `/api/education/managerial/records/${dossier.id}/documents`, {
    Categorie: 'hotarare', Titlu: `${marker} document managerial`, Stare: 'approved', Versiune: 'v1', 'Înregistrat la': '2026-09-10', 'Aprobat la': '2026-09-10', Responsabil: 'Director E2E', Note: marker,
  });
  await page.getByRole('button', { name: 'Pași flux', exact: true }).click();
  const workflow = await createManagerialChildThroughReact(page, 'Adaugă pași flux', `/api/education/managerial/records/${dossier.id}/workflow`, {
    Ordine: '1', Etapă: 'avizare_cp', Stare: 'completed', Alocat: 'Director E2E', Termen: '2026-09-11', 'Finalizat la': '2026-09-11', Rezultat: marker,
  });
  for (const endpoint of [`/api/education/managerial/records/${dossier.id}/pdf`, `/api/education/managerial/records/${dossier.id}/documents/${document.id}/pdf`]) {
    const result = await pdf(page, adminToken, endpoint); expect(result.status).toBe(200); expect(result.type).toContain('application/pdf'); expect(result.size).toBeGreaterThan(100);
  }
  expect(sql(`select count(*)::text from education_managerial_workflow_steps where id='${workflow.id}' and dossier_id='${dossier.id}'`)).toBe('1');
  expect(sql(`select count(*)::text from app_audit_log where action='education.managerial.documents.create' and target_id='${document.id}'`)).toBe('1');

  const regulationTitle = `${marker} regulament`;
  const regulation = await createRootThroughReact(page, '/scoala/regulations', '/api/education/regulations/records', {
    'An școlar': '2026-2027', 'Tip regulament': 'roi', Titlu: regulationTitle, Stare: 'draft', Aprobare: 'working_group', Responsabil: 'Director E2E', 'Revizuire până la': '2027-06-01', Rezumat: marker,
  });
  await openReactRootDetails(page, regulationTitle);
  const version = await createManagerialChildThroughReact(page, 'Adaugă versiuni', `/api/education/regulations/records/${regulation.id}/versions`, {
    Versiune: '1.0', Stare: 'draft', 'Pregătit de': 'Director E2E', 'Aprobat la': '2026-09-10', 'Aplicabil de la': '2026-09-15', 'Sinteza modificărilor': marker,
  });
  await page.getByRole('button', { name: 'Pași flux', exact: true }).click();
  const regulationWorkflow = await createManagerialChildThroughReact(page, 'Adaugă pași flux', `/api/education/regulations/records/${regulation.id}/workflow`, {
    Ordine: '1', Etapă: 'consultare_publica', Destinatari: 'personal', 'Început la': '2026-09-10', Stare: 'completed', Termen: '2026-09-12', 'Finalizat la': '2026-09-12', 'Referință decizie': 'CA-E2E', Observații: '0', Note: marker,
  });
  expect(sql(`select count(*)::text from education_regulation_versions where id='${version.id}' and regulation_id='${regulation.id}'`)).toBe('1');
  expect(sql(`select count(*)::text from education_regulation_workflow_steps where id='${regulationWorkflow.id}' and regulation_id='${regulation.id}'`)).toBe('1');
  expect(sql(`select count(*)::text from education_regulations where id='${regulation.id}'`, egueducation, true)).toBe('1');
  expect(sql(`select count(*)::text from education_regulations where id='${regulation.id}'`, balotesti, true)).toBe('0');

  const teacherContext = await browser.newContext({ baseURL: 'http://localhost:4174' });
  const teacher = await teacherContext.newPage();
  const teacherToken = await authenticate(teacher, teacherIdentifier, teacherOTP, 'http://localhost:4174');
  expect((await api<unknown>(teacher, teacherToken, '/api/education/managerial/records', { method: 'POST', body: JSON.stringify({}) })).status).toBe(403);
  expect((await api<unknown>(teacher, teacherToken, `/api/education/regulations/records/${regulation.id}`)).status).toBe(403);
  await teacherContext.close();
});

test('React creates mobility/merit roots and every operational child; DB verifies outcomes and PDFs', async ({ page }) => {
  prepareAdministrator();
  const adminToken = await authenticate(page, administratorIdentifier, administratorOTP, 'http://localhost:4173');
  const mobilityName = `${marker} Profesor mobilitate`;
  const mobility = await createRootThroughReact(page, '/scoala/mobility', '/api/education/mobility/records', {
    'Cod angajat': `MOB-${marker.slice(-8)}`, Nume: mobilityName, 'An școlar': '2026-2027', 'Tip solicitare': 'transfer', Etapă: 'review', Status: 'pending', 'Unitate sursă': 'Școala sursă', Destinație: 'Școala destinație', 'Depus la': '2026-09-10', 'Analizat de': 'Director E2E', Note: marker,
  });
  await openReactRootDetails(page, mobilityName);
  const mobilityDocument = await createManagerialChildThroughReact(page, 'Adaugă document', `/api/education/mobility/records/${mobility.id}/documents`, {
    Tip: 'cerere', Titlu: `${marker} cerere`, 'Înregistrat la': '2026-09-10', Validare: 'validated', 'Etapă document': 'verificare', 'Depus de': 'Profesor mobilitate', 'Verificat de': 'Director E2E', Note: marker,
  });
  await page.getByRole('button', { name: 'Punctaje', exact: true }).click();
  const mobilityScore = await createManagerialChildThroughReact(page, 'Adaugă punctaje', `/api/education/mobility/records/${mobility.id}/scores`, {
    'Categorie criteriu': 'performanta', 'Cod criteriu': 'M-01', Criteriu: 'Experiență', Maxim: '100', Acordat: '90', 'Referință dovezi': marker, 'Validat de': 'Director E2E', Note: marker,
  });
  await page.getByRole('button', { name: 'Contestații', exact: true }).click();
  const mobilityAppeal = await createManagerialChildThroughReact(page, 'Adaugă contestații', `/api/education/mobility/records/${mobility.id}/appeals`, {
    'Depus de': 'Profesor mobilitate', 'Depus la': '2026-09-11', Stare: 'resolved', Motive: marker, 'Audiere la': '2026-09-12', 'Soluționat la': '2026-09-12', Decizie: 'respins', Note: marker,
  });
  await page.getByRole('button', { name: 'Decizii finale', exact: true }).click();
  const mobilityDecision = await createManagerialChildThroughReact(page, 'Adaugă decizii finale', `/api/education/mobility/records/${mobility.id}/final-decisions`, {
    'Tip decizie': 'transfer', Rezultat: 'admis', 'Aprobat la': '2026-09-13', 'Aplicabil de la': '2026-09-15', Comisie: 'CA', 'Unitate destinație': 'Școala destinație', 'Temei legal': 'ROFUIP', Note: marker,
  });
  await page.getByRole('button', { name: 'Comunicări rezultat', exact: true }).click();
  const mobilityIssue = await createManagerialChildThroughReact(page, 'Adaugă comunicări rezultat', `/api/education/mobility/records/${mobility.id}/result-issues`, {
    'Tip document': 'comunicare', Destinatar: 'Profesor mobilitate', Funcție: 'profesor', Canal: 'email', 'Stare livrare': 'confirmat', 'Emis la': '2026-09-14', 'Livrat la': '2026-09-14', 'Referință registratură': 'REG-E2E', Note: marker,
  });
  for (const endpoint of [`/api/education/mobility/records/${mobility.id}/pdf`, `/api/education/mobility/records/${mobility.id}/appeals/${mobilityAppeal.id}/pdf`, `/api/education/mobility/records/${mobility.id}/final-decisions/${mobilityDecision.id}/pdf`, `/api/education/mobility/records/${mobility.id}/result-issues/${mobilityIssue.id}/pdf`]) {
    const result = await pdf(page, adminToken, endpoint); expect(result.status).toBe(200); expect(result.type).toContain('application/pdf');
  }
  expect(sql(`select count(*)::text from education_mobility_documents where id='${mobilityDocument.id}' and mobility_case_id='${mobility.id}'`)).toBe('1');
  expect(sql(`select count(*)::text from education_mobility_scores where id='${mobilityScore.id}' and mobility_case_id='${mobility.id}'`)).toBe('1');

  const meritName = `${marker} Profesor merit`;
  const merit = await createRootThroughReact(page, '/scoala/merit', '/api/education/gradatii/records', {
    Nume: meritName, Funcție: 'Profesor', 'An școlar': '2026-2027', Categorie: 'predare', Status: 'approved', Punctaj: '95', Comisie: 'Comisie E2E', 'Data deciziei': '2026-09-10', Note: marker,
  });
  await openReactRootDetails(page, meritName);
  const meritDocument = await createManagerialChildThroughReact(page, 'Adaugă document', `/api/education/gradatii/records/${merit.id}/documents`, {
    Tip: 'portofoliu', Titlu: `${marker} dosar merit`, 'Înregistrat la': '2026-09-10', Validare: 'validated', 'Depus de': 'Profesor merit', Note: marker,
  });
  await page.getByRole('button', { name: 'Punctaje', exact: true }).click();
  const meritScore = await createManagerialChildThroughReact(page, 'Adaugă punctaje', `/api/education/gradatii/records/${merit.id}/scores`, {
    'Cod criteriu': 'G-01', Criteriu: 'Rezultate', Categorie: 'Performanță', Maxim: '100', Acordat: '95', Evaluator: 'Director E2E', 'Etapă comisie': 'Evaluare comisie', 'Referință dovezi': marker, Note: marker,
  });
  await page.getByRole('button', { name: 'Contestații', exact: true }).click();
  const meritAppeal = await createManagerialChildThroughReact(page, 'Adaugă contestații', `/api/education/gradatii/records/${merit.id}/appeals`, {
    'Depus de': 'Profesor merit', 'Depus la': '2026-09-11', Stare: 'resolved', Motive: marker, 'Soluționat la': '2026-09-12', Decizie: 'admis', Note: marker,
  });
  await page.getByRole('button', { name: 'Decizii finale', exact: true }).click();
  const meritDecision = await createManagerialChildThroughReact(page, 'Adaugă decizii finale', `/api/education/gradatii/records/${merit.id}/final-decisions`, {
    Etapă: 'validare_finala', Rezultat: 'admis', 'Aprobat la': '2026-09-13', 'Aplicabil de la': '2026-09-15', Comisie: 'CA', 'Temei legal': 'metodologie', Note: marker,
  });
  await page.getByRole('button', { name: 'Comunicări rezultat', exact: true }).click();
  const meritIssue = await createManagerialChildThroughReact(page, 'Adaugă comunicări rezultat', `/api/education/gradatii/records/${merit.id}/result-issues`, {
    'Tip document': 'comunicare', Destinatar: 'Profesor merit', Funcție: 'profesor', Canal: 'email', 'Stare livrare': 'confirmat', 'Emis la': '2026-09-14', 'Livrat la': '2026-09-14', 'Referință registratură': 'MERIT-E2E', Note: marker,
  });
  for (const endpoint of [`/api/education/gradatii/records/${merit.id}/pdf`, `/api/education/gradatii/records/${merit.id}/appeals/${meritAppeal.id}/pdf`, `/api/education/gradatii/records/${merit.id}/final-decisions/${meritDecision.id}/pdf`, `/api/education/gradatii/records/${merit.id}/result-issues/${meritIssue.id}/pdf`]) {
    const result = await pdf(page, adminToken, endpoint); expect(result.status).toBe(200); expect(result.type).toContain('application/pdf');
  }
  expect(sql(`select count(*)::text from education_merit_documents where id='${meritDocument.id}' and grant_id='${merit.id}'`)).toBe('1');
  expect(sql(`select count(*)::text from education_merit_scores where id='${meritScore.id}' and grant_id='${merit.id}'`)).toBe('1');
  expect(sql(`select count(*)::text from app_audit_log where action='education.gradatii.result_issue.create' and target_id='${meritIssue.id}'`)).toBe('1');
});

test('React creates portfolio relations and drives transfer/valorification lifecycle; DB verifies contracts', async ({ page }) => {
  const adminID = prepareAdministrator();
  const adminName = sql(`select name from app_users where id='${adminID}'`);
  const suffix = marker.slice(-12).replace(/[^A-Za-z0-9]/g, 'X');
  adminSQL(`insert into education_personnel (employee_code,full_name,role_title,employment_type,status,evaluation_status,mobility_stage,school_year,assigned_unit,phone,email,has_portfolio,institution_id,notes,app_user_id) values ('PORT-${suffix}','${marker} Proprietar','Profesor','titular','active','draft','none','2026-2027','Învățământ gimnazial','+40100000111','portfolio-${suffix}@example.test',true,'inst-001','${marker}','${adminID}') on conflict (institution_id,app_user_id) where app_user_id is not null do update set employee_code=excluded.employee_code,full_name=excluded.full_name,school_year=excluded.school_year`);
  const personnelID = sql(`select id::text from education_personnel where institution_id='inst-001' and app_user_id='${adminID}'`);
  const adminToken = await authenticate(page, administratorIdentifier, administratorOTP, 'http://localhost:4173');
  await publishPortfolioProcedureThroughReact(page);
  await page.goto('/scoala/portfolios');
  await page.getByRole('button', { name: 'Adaugă înregistrare' }).click();
  await expect(page).toHaveURL(/\/scoala\/portfolio\/wizard$/);
  await searchAndSelectPortfolioOwner(page, marker, `${marker} Proprietar · Profesor`);
  await page.getByRole('button', { name: 'Continuă' }).click();
  await page.getByLabel('Actualizat la').fill('2026-09-10');
  await page.getByLabel('Custode').fill('Școala E2E');
  await page.getByRole('button', { name: 'Continuă' }).click();
  await page.getByLabel('Note').fill(marker);
  await page.getByRole('button', { name: 'Continuă' }).click();
  const portfolioCreated = page.waitForResponse((response) => new URL(response.url()).pathname === '/api/education/portfolios/records' && response.request().method() === 'POST');
  await page.getByRole('button', { name: 'Salvează' }).click();
  const portfolioResponse = await portfolioCreated;
  expect(portfolioResponse.status()).toBe(201);
  const portfolio = await portfolioResponse.json() as RecordWithID;
  expect(portfolio.owner_user_id).toBe(adminID);
  expect(portfolio.owner_personnel_id).toBe(personnelID);
  await page.goto('/scoala/portfolios');
  await openReactRootDetails(page, `${marker} Proprietar`);
  await expect(page.getByText('Portofoliu — operațiuni dosar')).toBeVisible();
  // A portfolio entry is a reference to immutable evidence already stored in
  // eArhivă. Seed only that external storage/OCR precondition; creation and
  // subsequent removal of the portfolio reference remain browser operations.
  const institutionalArchiveTitle = `${marker} document instituțional`;
  const institutionalArchiveDocumentID = sql(`insert into archive_documents (institution_id,title,original_file_name,mime_type,source_kind,source_system,external_reference,status,original_bucket,original_object_key,artifact_bucket,artifact_object_key,current_version_no,created_by) values ('inst-001','${institutionalArchiveTitle}','${marker}-institutional.pdf','application/pdf','upload','e2e','${marker}-institutional','ready','earhive','e2e/${marker}-institutional.pdf','earhive','e2e/${marker}-institutional.pdf',1,'e2e') returning id::text`);
  sql(`insert into archive_document_versions (document_id,institution_id,version_no,mime_type,title,bucket_name,object_key,hash_sha256,size_bytes,status,source_bucket,source_object_key,source_sha256,source_size_bytes,text_status) values ('${institutionalArchiveDocumentID}','inst-001',1,'application/pdf','${institutionalArchiveTitle}','earhive','e2e/${marker}-institutional.pdf','${'c'.repeat(64)}',128,'active','earhive','e2e/${marker}-institutional.pdf','${'c'.repeat(64)}',128,'processed')`);
  const document = await createManagerialChildThroughReact(page, 'Adaugă document', `/api/education/portfolios/records/${portfolio.id}/documents`, {
    Titlu: `${marker} document portofoliu`, 'Descriere pedagogică': 'Dovadă instituțională verificată', 'An școlar': '2026-2027', Disciplina: 'Management educațional', 'Clasa aplicabilă': 'Instituție', 'Competențe (separate prin virgulă)': 'management, conformitate', 'Tip dovadă': 'decizie', Secțiune: 'identificare_profesionala', Componentă: 'structura_cadru', 'Domeniu sursă': 'portofoliu', Autenticitate: 'verificat', 'Data emiterii': '2026-09-10', 'Data adăugării': '2026-09-10', 'Ordine cronologică': '1', 'Referință arhivă': `archive://${institutionalArchiveDocumentID}`, Observații: marker,
  });
  await page.getByRole('button', { name: 'Checklist', exact: true }).click();
  const checklist = await createManagerialChildThroughReact(page, 'Adaugă cerință', `/api/education/portfolios/records/${portfolio.id}/checklist`, {
    'Cod cerință *': 'P-01', 'Cerință *': 'Document identitate profesională', 'Secțiune *': 'identificare_profesionala', 'Domeniu sursă *': 'portofoliu', 'Stare *': 'complet', 'Ultima verificare *': '2026-09-10', 'Verificat de': 'Director E2E', 'Număr documente': '1', Observații: marker,
  });
  await page.getByRole('button', { name: 'Opis', exact: true }).click();
  const opis = await createManagerialChildThroughReact(page, 'Adaugă poziție opis', `/api/education/portfolios/records/${portfolio.id}/opis`, {
    'Secțiune *': 'identificare_profesionala', 'Componentă *': 'structura_cadru', 'Titlu *': `${marker} opis`, 'Referință document *': document.id, 'Domeniu sursă *': 'portofoliu', 'Data verificării *': '2026-09-10', 'Verificat de': 'Director E2E', 'Ordine cronologică': '1', Observații: marker,
  });
  await page.getByRole('button', { name: 'Custodie', exact: true }).click();
  const custody = await createManagerialChildThroughReact(page, 'Adaugă eveniment de custodie', `/api/education/portfolios/records/${portfolio.id}/custody`, {
    'Tip eveniment *': 'consultare', 'Custode *': 'Director E2E', 'Rol custode *': 'director', 'Locație *': 'arhiva', 'Mod acces *': 'digital', 'Motiv acces *': 'verificare', 'Început *': '2026-09-10', Observații: marker,
  });
  await page.getByRole('button', { name: 'Revizuiri', exact: true }).click();
  const review = await createManagerialChildThroughReact(page, 'Adaugă revizuire', `/api/education/portfolios/records/${portfolio.id}/reviews`, {
    'Etapă *': 'verificare_secretariat', 'Rezultat *': 'acceptat', 'Evaluator *': 'Director E2E', 'Data revizuirii *': '2026-09-10', 'Scor conformitate': '100', 'Documente lipsă': '0', Observații: marker,
  });
  expect(sql(`select count(*)::text from education_portfolio_documents where id='${document.id}' and portfolio_id='${portfolio.id}'`)).toBe('1');
  expect(sql(`select count(*)::text from education_portfolio_opis where id='${opis.id}' and portfolio_id='${portfolio.id}'`)).toBe('1');
  expect(sql(`select count(*)::text from education_portfolio_custody where id='${custody.id}' and portfolio_id='${portfolio.id}'`)).toBe('1');
  expect(sql(`select count(*)::text from education_portfolio_reviews where id='${review.id}' and portfolio_id='${portfolio.id}'`)).toBe('1');
  expect(sql(`select count(*)::text from education_portfolio_checklist where id='${checklist.id}' and portfolio_id='${portfolio.id}'`)).toBe('1');

  // The valorification source is an ordinary School mobility case, so create
  // it through the same PrimeReact root dialog used by the operator.
  const mobilitySource = await createRootThroughReact(page, '/scoala/mobility', '/api/education/mobility/records', {
    'Cod angajat': `PORT-${suffix}`, Nume: `${marker} Proprietar`, 'An școlar': '2026-2027', 'Tip solicitare': 'transfer', Etapă: 'review', Status: 'pending', 'Unitate sursă': 'Școala E2E', Destinație: 'Școala Balotești', 'Depus la': '2026-09-10', 'Analizat de': 'Director E2E', Note: marker,
  });
  // These ready archive documents model the external eArhivă storage/OCR
  // precondition only.  Every School-side authorization and portfolio action
  // below is performed through the actual React interface.
  const portfolioSections = ['identificare_profesionala', 'predare_invatare_evaluare', 'activitati_complementare', 'managementul_clasei', 'evolutie_dezvoltare_profesionala'];
  const archiveEvidence = portfolioSections.map((section, index) => {
    const title = `${marker} dovadă ${index + 1}`;
    const documentID = sql(`insert into archive_documents (institution_id,title,original_file_name,mime_type,source_kind,source_system,external_reference,status,original_bucket,original_object_key,artifact_bucket,artifact_object_key,created_by) values ('inst-001','${title}','${marker}-${index + 1}.pdf','application/pdf','upload','e2e','${marker}-${index + 1}','ready','earhive','e2e/${marker}-${index + 1}.pdf','earhive','e2e/${marker}-${index + 1}.pdf','e2e') returning id::text`);
    sql(`insert into archive_document_versions (document_id,institution_id,version_no,mime_type,title,bucket_name,object_key,hash_sha256,size_bytes,status,source_bucket,source_object_key,source_sha256,source_size_bytes,text_status) values ('${documentID}','inst-001',1,'application/pdf','${title}','earhive','e2e/${marker}-${index + 1}.pdf','${'b'.repeat(64)}',128,'active','earhive','e2e/${marker}-${index + 1}.pdf','${'b'.repeat(64)}',128,'processed')`);
    return { section, title, documentID };
  });
  await page.goto('/scoala/portfolios');
  await expect(page.getByText('Acces documente eArhivă pentru portofolii')).toBeVisible();
  for (const archive of archiveEvidence) {
    await page.getByLabel('Adaugă drept de atașare').click();
    const grantDialog = page.getByRole('dialog', { name: 'Acordă acces la document eArhivă' });
    const documentRow = grantDialog.getByText(archive.title, { exact: true }).locator('xpath=ancestor::tr[1]');
    await expect(documentRow).toBeVisible({ timeout: 10_000 });
    await documentRow.getByRole('button', { name: 'Selectează' }).click();
    const userRow = grantDialog.getByText(adminName, { exact: true }).locator('xpath=ancestor::tr[1]');
    await expect(userRow).toBeVisible({ timeout: 10_000 });
    await userRow.getByRole('button', { name: 'Selectează' }).click();
    const granted = page.waitForResponse((response) => new URL(response.url()).pathname === '/api/education/portfolios/archive-attachment-grants' && response.request().method() === 'POST');
    await grantDialog.getByRole('button', { name: 'Acordă acces' }).click();
    expect((await granted).status()).toBe(201);
    await expect(grantDialog).toBeHidden();
  }

  await page.goto('/scoala/portfolio/me');
  await expect(page.getByRole('region', { name: 'Portofoliul meu profesional' })).toBeVisible();
  for (const [index, archive] of archiveEvidence.entries()) {
    await page.getByRole('button', { name: 'Adaugă document' }).click();
    const documentDialog = page.getByRole('dialog', { name: 'Adaugă document în portofoliu' });
    await documentDialog.getByRole('combobox', { name: 'Componentă din catalog' }).click();
    await selectOpenOption(page, new RegExp(`^${archive.section} ·`));
    await documentDialog.getByLabel('Titlu *').fill(archive.title);
    await documentDialog.getByLabel('Tip dovadă *').fill('adeverinta');
    await documentDialog.getByLabel('Descriere pedagogică *').fill(`Dovadă pedagogică pentru ${archive.title}`);
    await documentDialog.getByLabel('An școlar *').fill('2026-2027');
    await documentDialog.getByLabel('Disciplina *').fill('Management educațional');
    await documentDialog.getByLabel('Clasa aplicabilă *').fill('Instituție');
    await documentDialog.getByLabel('Competențe *').fill('management, conformitate');
    await documentDialog.getByLabel('Data emiterii *').fill('2026-09-10');
    await documentDialog.getByLabel('Data adăugării *').fill('2026-09-10');
    await documentDialog.getByLabel('Index cronologic').fill(String(index + 1));
    await documentDialog.getByRole('combobox', { name: 'Document eArhivă autorizat' }).click();
    await selectOpenOption(page, `${archive.title} · v1`);
    const documentAdded = page.waitForResponse((response) => new URL(response.url()).pathname === `/api/education/portfolios/me/${portfolio.id}/documents` && response.request().method() === 'POST');
    await documentDialog.getByRole('button', { name: 'Adaugă document' }).click();
    expect((await documentAdded).status()).toBe(201);
  }
  // The institutional CRUD proof above intentionally created an extra valid
  // archive-backed reference. Remove it through the owner's UI so submission
  // covers precisely the five statutory catalog sections.
  const deletedDraftDocument = page.waitForResponse((response) =>
    response.request().method() === 'DELETE'
    && new URL(response.url()).pathname === `/api/education/portfolios/me/${portfolio.id}/documents/${document.id}`);
  await page.getByRole('button', { name: `Șterge ${marker} document portofoliu` }).click();
  await page.getByRole('dialog', { name: 'Elimină documentul?' }).getByRole('button', { name: 'Confirmă eliminarea' }).click();
  expect((await deletedDraftDocument).status()).toBe(204);
  for (let declarationIndex = 0; declarationIndex < 2; declarationIndex += 1) {
    await page.getByRole('button', { name: 'Citește și confirmă' }).first().click();
    const declarationDialog = page.getByRole('dialog');
    await declarationDialog.getByLabel('Confirm declarația afișată').click();
    const acknowledged = page.waitForResponse((response) => response.request().method() === 'POST' && new URL(response.url()).pathname.includes(`/api/education/portfolios/me/${portfolio.id}/declarations/`));
    await declarationDialog.getByRole('button', { name: 'Confirmă declarația' }).click();
    expect((await acknowledged).status()).toBe(200);
  }
  const opisRegenerated = page.waitForResponse((response) => new URL(response.url()).pathname === `/api/education/portfolios/me/${portfolio.id}/opis/regenerate` && response.request().method() === 'POST');
  await page.getByRole('button', { name: 'Regenerare opis' }).click();
  expect((await opisRegenerated).status()).toBe(200);
  const submitted = page.waitForResponse((response) => new URL(response.url()).pathname === `/api/education/portfolios/me/${portfolio.id}/submit` && response.request().method() === 'POST');
  await expect(page.getByRole('button', { name: 'Trimite spre verificare' })).toBeEnabled();
  await page.getByRole('button', { name: 'Trimite spre verificare' }).click();
  await page.getByRole('button', { name: 'Confirmă trimiterea' }).click();
  expect((await submitted).status()).toBe(200);
  await expect(page.getByText('Portofoliul a fost trimis spre verificare.')).toBeVisible();
  expect(sql(`select status || '|' || authenticity_declared::text || '|' || consent_captured::text from education_portfolios where id='${portfolio.id}'`)).toBe('submitted|true|true');
  await page.goto('/scoala/portfolios');
  await openReactRootDetails(page, `${marker} Proprietar`);
  // The transfer is tenant-addressed (not a browser-supplied institution ID).
  // The submitted portfolio already has complete immutable eArhivă provenance,
  // so sending must seal a server-generated export manifest and succeed.
  await expect(page.getByText('Expediere inter-tenant')).toBeVisible();
  await page.getByLabel('Inițiază transfer').click();
  const transferDialog = page.getByRole('dialog');
  await transferDialog.getByLabel('Tenant destinație').click();
  const destinationLabel = sql("select coalesce(nullif(short_name,''), display_name) from app_tenants where code='tenant-balotesti'");
  await selectOpenOption(page, destinationLabel);
  await transferDialog.locator('input[type="date"]').fill('2026-09-20');
  await transferDialog.locator('input[type="text"]').fill(marker);
  const transferCreated = page.waitForResponse((response) => new URL(response.url()).pathname === `/api/education/portfolios/records/${portfolio.id}/transfers` && response.request().method() === 'POST');
  await transferDialog.getByRole('button', { name: 'Inițiază transfer' }).click();
  const transferResponse = await transferCreated;
  expect(transferResponse.status()).toBe(201);
  const transfer = await transferResponse.json() as RecordWithID;
  expect(transfer.status).toBe('pregatit');
  const sentResponse = page.waitForResponse((response) => new URL(response.url()).pathname === `/api/education/portfolios/records/${portfolio.id}/transfers/${transfer.id}/advance` && response.request().method() === 'POST');
  await page.getByLabel(`Marchează trimis ${String(transfer.transfer_code)}`).click();
  const sentHTTP = await sentResponse;
  expect(sentHTTP.status()).toBe(200);
  expect(await sentHTTP.json()).toMatchObject({ id: transfer.id, status: 'trimis' });
  expect(sql(`select status || '|' || (export_manifest_id is not null)::text from education_portfolio_transfers where id='${transfer.id}'`)).toBe('trimis|true');
  expect(sql(`select count(*)::text from app_audit_log where action='education.portfolios.transfer.advance' and target_id='${transfer.id}'`)).toBe('1');

  // A ready eArhiva version is storage/OCR fixture state, not a user-facing
  // School operation; its setup remains intentionally outside the UI proof.
  const archiveTitle = `${marker} dovadă arhivă`;
  const archiveDocumentID = sql(`insert into archive_documents (institution_id,title,original_file_name,mime_type,source_kind,source_system,external_reference,status,original_bucket,original_object_key,artifact_bucket,artifact_object_key,created_by) values ('inst-001','${archiveTitle}','${marker}.pdf','application/pdf','upload','e2e','${marker}','ready','earhive','e2e/${marker}.pdf','earhive','e2e/${marker}.pdf','e2e') returning id::text`);
  const archiveVersionID = sql(`insert into archive_document_versions (document_id,institution_id,version_no,mime_type,title,bucket_name,object_key,hash_sha256,size_bytes,status,source_bucket,source_object_key,source_sha256,source_size_bytes,text_status) values ('${archiveDocumentID}','inst-001',1,'application/pdf','${archiveTitle}','earhive','e2e/${marker}.pdf','${'a'.repeat(64)}',128,'active','earhive','e2e/${marker}.pdf','${'a'.repeat(64)}',128,'processed') returning id::text`);
  await expect(page.getByText('Pachete de valorificare')).toBeVisible();
  await page.getByLabel('Adaugă pachet').click();
  const packageDialog = page.getByRole('dialog');
  await packageDialog.getByLabel('Domeniu valorificare').click();
  await selectOpenOption(page, 'Mobilitate');
  await packageDialog.getByLabel('Sursă eligibilă').click();
  await selectOpenOption(page, new RegExp(marker));
  const packageCreated = page.waitForResponse((response) => new URL(response.url()).pathname === `/api/education/portfolios/records/${portfolio.id}/valorification-packages` && response.request().method() === 'POST');
  await packageDialog.getByRole('button', { name: 'Continuă' }).click();
  const packageResponse = await packageCreated;
  expect(packageResponse.status()).toBe(201);
  const packageItem = await packageResponse.json() as RecordWithID;
  await expect(packageDialog.getByLabel('Versiune eArhivă')).toBeVisible();
  await packageDialog.getByLabel('Versiune eArhivă').click();
  await selectOpenOption(page, `${archiveTitle} · versiunea 1`);
  const packageEvidence = page.waitForResponse((response) => new URL(response.url()).pathname === `/api/education/portfolios/records/${portfolio.id}/valorification-packages/${packageItem.id}/documents` && response.request().method() === 'POST');
  await packageDialog.getByRole('button', { name: 'Atașează versiunea' }).click();
  expect((await packageEvidence).status()).toBe(201);
  expect(archiveVersionID).toMatch(/^[0-9a-f-]{36}$/i);
  expect(mobilitySource.id).toEqual(expect.any(String));
  for (const action of ['submit', 'validate', 'complete']) {
    const advanced = page.waitForResponse((response) => new URL(response.url()).pathname === `/api/education/portfolios/records/${portfolio.id}/valorification-packages/${packageItem.id}/advance` && response.request().method() === 'POST');
    await page.getByLabel(`${action} Mobilitate`).click();
    expect((await advanced).status()).toBe(200);
  }
  expect(sql(`select status from education_portfolio_valorification_packages where id='${packageItem.id}'`)).toBe('completed');
  expect(sql(`select count(*)::text from app_audit_log where action='education.portfolios.valorification_package.advance' and target_id='${packageItem.id}'`)).toBe('3');
});
