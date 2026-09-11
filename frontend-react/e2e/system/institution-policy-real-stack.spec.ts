import { expect, test, type Page } from '@playwright/test';
import { execFileSync } from 'node:child_process';

/* No request interception: both cases cross React, the production OIDC
 * provider, Go policy middleware and PostgreSQL. */
const hasStack = Boolean(process.env.TEST_DATABASE_URL && process.env.DATABASE_URL);
test.skip(!hasStack, 'Requires the real PostgreSQL/OIDC system topology.');

type TenantFixture = {
  origin: string;
  tenant: string;
  institution: string;
  identifier: string;
  otp: string;
  subject: string;
};

const publicSchool: TenantFixture = { origin: 'http://localhost:4173', tenant: 'tenant-egueducation', institution: 'inst-001', identifier: 'oidc.browser.fixture@example.test', otp: '173829', subject: 'oidc-browser-fixture-subject' };
const privateSchool: TenantFixture = { origin: 'http://localhost:4175', tenant: 'tenant-balotesti', institution: 'inst-balotesti', identifier: 'oidc.balotesti.fixture@example.test', otp: '739204', subject: 'oidc-browser-balotesti-subject' };

function sql(fixture: TenantFixture, statement: string): string {
  const url = process.env.TEST_DATABASE_URL;
  if (!url) throw new Error('TEST_DATABASE_URL is required.');
  return execFileSync('psql', ['--no-psqlrc', '--tuples-only', '--no-align', '--quiet', url, '-c', `set app.tenant_id='${fixture.tenant}'; set app.institution_id='${fixture.institution}'; set app.is_super_admin='true'; ${statement}`], { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] }).trim();
}

async function login(page: Page, fixture: TenantFixture): Promise<string> {
  let accessToken = '';
  page.on('response', async (response) => {
    if (!response.url().includes('/api/oidc/token') || response.request().method() !== 'POST') return;
    const body = await response.json().catch(() => undefined) as { access_token?: string } | undefined;
    accessToken ||= body?.access_token ?? '';
  });
  await page.goto(fixture.origin);
  await page.getByRole('button', { name: 'Autentificare' }).last().click();
  await page.getByRole('button', { name: /SMS/ }).click();
  await page.getByLabel('Utilizator, email sau numar de telefon').fill(fixture.identifier);
  await page.getByLabel('Canal OTP').selectOption('sms');
  await page.getByRole('button', { name: 'Trimite codul' }).click();
  const boxes = page.locator('.otp-box');
  await expect(boxes).toHaveCount(6);
  await boxes.first().pressSequentially(fixture.otp);
  await page.getByRole('button', { name: 'Verifica codul' }).click();
  const consent = page.getByRole('button', { name: 'Accepta si continua' });
  if (await consent.count()) await consent.click();
  await expect(page).toHaveURL(`${fixture.origin}/`);
  await expect.poll(() => accessToken, { message: 'Browser did not observe the OIDC token response.' }).not.toBe('');
  return accessToken;
}

async function select(page: Page, label: string, option: string): Promise<void> {
  await page.getByRole('combobox', { name: label, exact: true }).click();
  await page.locator('[role="listbox"]:visible').last().getByRole('option', { name: option, exact: true }).click();
}

async function classify(page: Page, fixture: TenantFixture, legalForm: 'Școală publică' | 'Școală privată', publicFunding: boolean): Promise<string> {
  const userID = sql(fixture, `select id::text from app_users where sub='${fixture.subject}'`);
  sql(fixture, `update app_memberships set position_code='director', active=true where user_id='${userID}' and tenant_code='${fixture.tenant}'; delete from app_user_roles where user_id='${userID}' and tenant_code='${fixture.tenant}'; insert into app_user_roles(tenant_code,user_id,role_code) values ('${fixture.tenant}','${userID}','admin') on conflict do nothing`);
  const accessToken = await login(page, fixture);
  await page.goto(`${fixture.origin}/administrare`);
  await page.getByRole('tab', { name: 'Profil instituțional' }).click();
  await page.getByRole('button', { name: /Versiune nouă/ }).click();
  await select(page, 'Formă juridică', legalForm);
  await select(page, 'Stare profil', 'Activ');
  await page.getByLabel('Sursa aprobării').fill(`E2E-${fixture.tenant}`);
  if (publicFunding) {
    const fundingField = page.getByText('Finanțare publică', { exact: true }).locator('..');
    await fundingField.getByRole('combobox').click();
    await page.locator('[role="listbox"]:visible').last().getByRole('option', { name: 'Da', exact: true }).click();
  }
  const save = page.waitForResponse((response) => response.request().method() === 'PUT' && new URL(response.url()).pathname === '/api/institution/regulatory-profile');
  await page.getByRole('button', { name: 'Salvează versiunea' }).click();
  expect((await save).status()).toBe(200);
  await expect(page.getByText('education.publication.manage', { exact: true })).toBeVisible();
  return accessToken;
}

async function createPublication(page: Page, fixture: TenantFixture): Promise<{ id: string; policy_evaluation_id: string }> {
  await page.goto(`${fixture.origin}/scoala/compliance`);
  await page.getByLabel('Adaugă înregistrare').click();
  const dialog = page.getByRole('dialog');
  for (const [label, option] of [['Domeniu', 'conformitate'], ['Tip entitate', 'anunt'], ['Canal', 'site_public'], ['Stare', 'pregatit'], ['Anonimizare', 'nu_este_necesara']] as const) {
    await dialog.getByRole('combobox', { name: label, exact: true }).click();
    await page.locator('[role="listbox"]:visible').last().getByRole('option', { name: option, exact: true }).click();
  }
  const marker = `POLICY-${fixture.tenant}-${Date.now()}`;
  await dialog.getByLabel('Entitate', { exact: true }).fill(marker);
  const response = page.waitForResponse((candidate) => candidate.request().method() === 'POST' && new URL(candidate.url()).pathname === '/api/education/compliance/publications');
  await dialog.getByRole('button', { name: 'Salvează' }).click();
  const created = await response;
  expect(created.status()).toBe(201);
  const body = await created.json() as { id: string; policy_evaluation_id: string };
  expect(body.policy_evaluation_id).toMatch(/^[0-9a-f-]{36}$/);
  expect(sql(fixture, `select count(*)::text from education_publications publication join school_policy_evaluations evaluation on evaluation.id=publication.policy_evaluation_id and evaluation.tenant_code=publication.tenant_code and evaluation.institution_id=publication.institution_id where publication.id='${body.id}' and publication.tenant_code='${fixture.tenant}' and publication.institution_id='${fixture.institution}' and not evaluation.blocked`)).toBe('1');
  return body;
}

async function assertServerDerivedPolicyScope(page: Page, accessToken: string, fixture: TenantFixture, forged: TenantFixture): Promise<void> {
  const result = await page.evaluate(async ({ accessToken, tenant, institution }) => {
    const response = await fetch('/api/education/compliance/publications', {
      method: 'POST',
      credentials: 'include',
      headers: { authorization: `Bearer ${accessToken}`, 'content-type': 'application/json' },
      body: JSON.stringify({
        domain: 'conformitate',
        entity_type: 'anunt',
        entity_label: 'FORGED-POLICY-SCOPE',
        publication_channel: 'site_public',
        publication_status: 'pregatit',
        anonymization_status: 'nu_este_necesara',
        mandatory: false,
        reviewed_by: '',
        notes: '',
        tenant_code: tenant,
        institution_id: institution,
        policy_evaluation_id: '00000000-0000-0000-0000-000000000000',
        legal_form: 'public',
      }),
    });
    return { status: response.status, body: await response.json() as { code?: string } };
  }, { accessToken, tenant: forged.tenant, institution: forged.institution });
  expect(result).toEqual({ status: 400, body: { code: 'invalid_publication_payload' } });
  expect(sql(fixture, `select count(*)::text from education_publications where entity_label='FORGED-POLICY-SCOPE'`)).toBe('0');
}

test('public and private-with-public-funding institutions resolve and enforce distinct policy overlays', async ({ browser }) => {
  for (const [fixture, legalForm, publicFunding] of [[publicSchool, 'Școală publică', false], [privateSchool, 'Școală privată', true]] as const) {
    const context = await browser.newContext();
    const page = await context.newPage();
    const accessToken = await classify(page, fixture, legalForm, publicFunding);
    await createPublication(page, fixture);
    await assertServerDerivedPolicyScope(page, accessToken, fixture, fixture === publicSchool ? privateSchool : publicSchool);
    const legalPack = legalForm === 'Școală publică' ? 'legal-form.ro.public' : 'legal-form.ro.private';
    expect(sql(fixture, `select count(*)::text from school_policy_assignments where tenant_code='${fixture.tenant}' and institution_id='${fixture.institution}' and pack_code='${legalPack}' and status='active'`)).toBe('1');
    expect(sql(fixture, `select count(*)::text from school_policy_assignments where tenant_code='${fixture.tenant}' and institution_id='${fixture.institution}' and pack_code='funding.public' and status='active'`)).toBe(publicFunding ? '1' : '0');
    await context.close();
  }
});
