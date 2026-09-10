import { expect, test, type Browser, type Page } from '@playwright/test';
import { execFileSync } from 'node:child_process';

/*
 * This is intentionally a real-stack proof.  It does not register routes or
 * forge a session: each actor completes the public OIDC SMS ceremony and every
 * application assertion traverses React -> Go -> PostgreSQL with RLS enabled.
 */
const origin = 'http://localhost:4173';
const director = { identifier: 'oidc.browser.fixture@example.test', otp: '173829', subject: 'oidc-browser-fixture-subject' };
const adjunct = { identifier: 'oidc.approver.fixture@example.test', otp: '428615', subject: 'oidc-browser-approver-subject' };
const otherTenant = { identifier: 'oidc.balotesti.fixture@example.test', otp: '739204' };
const marker = `SCHOOL-GOVERNANCE-E2E-${process.env.GITHUB_RUN_ID ?? 'local'}-${Date.now()}`;

type Scope = { tenant: string; institution: string };
const school: Scope = { tenant: 'tenant-egueducation', institution: 'inst-001' };
const balotesti: Scope = { tenant: 'tenant-balotesti', institution: 'inst-balotesti' };

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

/** Exercise the wizard's server-side selector before opening its Select. */
async function searchAndSelectWizardUser(page: Page, label: string, name: string): Promise<void> {
  const response = page.waitForResponse((candidate) => {
    const url = new URL(candidate.url());
    return candidate.request().method() === 'GET'
      && url.pathname === '/api/education/governance/eligible-users'
      && url.searchParams.get('filter.name') === name;
  });
  await page.getByLabel(`Caută ${label}`, { exact: true }).fill(name);
  expect((await response).status()).toBe(200);
  const trigger = page.getByRole('combobox', { name: label, exact: true });
  await expect(trigger).toBeEnabled();
  await trigger.click();
  await selectOpenOption(page, name);
}

async function clickOpenPopoverAction(page: Page, name: string): Promise<void> {
  const menu = page.locator('[role="menu"]:visible').last();
  await expect(menu).toBeVisible();
  const action = menu.getByRole('button', { name, exact: true });
  await expect(action).toBeVisible();
  await action.focus();
  await action.press('Enter');
}

const hasRealStack = Boolean(process.env.TEST_DATABASE_URL && process.env.DATABASE_URL);
test.skip(!hasRealStack, 'Real-stack school governance requires TEST_DATABASE_URL and DATABASE_URL; it never substitutes mocks.');

function sql(value: string, scope: Scope = school, privileged = true): string {
  const connection = privileged ? process.env.TEST_DATABASE_URL : process.env.DATABASE_URL;
  if (!connection) throw new Error('Missing PostgreSQL URL for real-stack school governance proof.');
  const scoped = `set app.tenant_id = '${scope.tenant}'; set app.institution_id = '${scope.institution}'; set app.is_super_admin = '${privileged}'; ${value}`;
  return execFileSync('psql', ['--no-psqlrc', '--tuples-only', '--no-align', '--quiet', connection, '-c', scoped], {
    encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'],
  }).trim();
}

async function login(page: Page, fixture: { identifier: string; otp: string }, expectedOrigin = origin): Promise<string> {
  let token = '';
  page.on('response', async response => {
    if (response.request().method() !== 'POST' || !response.url().includes('/api/oidc/token')) return;
    const body = await response.json().catch(() => undefined) as { access_token?: string } | undefined;
    token ||= body?.access_token ?? '';
  });
  await page.goto(`${expectedOrigin}/`);
  await page.getByRole('button', { name: 'Autentificare' }).last().click();
  await expect(page).toHaveURL(/\/api\/oidc\/authorize/);
  await page.getByRole('button', { name: /SMS/ }).click();
  await page.getByLabel('Utilizator, email sau numar de telefon').fill(fixture.identifier);
  await page.getByLabel('Canal OTP').selectOption('sms');
  await page.getByRole('button', { name: 'Trimite codul' }).click();
  const boxes = page.locator('.otp-box');
  await expect(boxes).toHaveCount(6);
  await boxes.first().click();
  await boxes.first().pressSequentially(fixture.otp);
  await page.getByRole('button', { name: 'Verifica codul' }).click();
  const consent = page.getByRole('button', { name: 'Accepta si continua' });
  if (await consent.count()) await consent.click();
  await expect(page).toHaveURL(`${expectedOrigin}/`);
  await expect.poll(() => token, { message: 'Browser did not observe the real OIDC token response.' }).not.toBe('');
  const claims = JSON.parse(Buffer.from(token.split('.')[1], 'base64url').toString('utf8')) as { platform_roles?: string[] };
  expect(claims.platform_roles ?? []).toEqual([]);
  const me = await page.evaluate(async accessToken => {
    const response = await fetch('/api/me', { headers: { Authorization: `Bearer ${accessToken}` }, credentials: 'include' });
    return { status: response.status, body: await response.json() as { platform_roles?: string[] } };
  }, token);
  expect(me.status).toBe(200);
  expect(me.body.platform_roles ?? []).toEqual([]);
  return token;
}

async function request<T>(page: Page, token: string, path: string, init: RequestInit = {}): Promise<{ status: number; body: T }> {
  return page.evaluate(async ({ token, path, init }) => {
    const headers = new Headers(init.headers);
    headers.set('Authorization', `Bearer ${token}`);
    headers.set('Content-Type', 'application/json');
    const response = await fetch(path, { ...init, headers, credentials: 'include' });
    const text = await response.text();
    return { status: response.status, body: text ? JSON.parse(text) : null };
  }, { token, path, init });
}

type RecordWithID = { id: string };

async function createRootThroughReact(
  page: Page,
  route: string,
  endpoint: string,
  values: Record<string, string>,
): Promise<RecordWithID> {
  await page.goto(route);
  await page.getByLabel('Adaugă înregistrare').click();
  const dialog = page.getByRole('dialog');
  await expect(dialog).toBeVisible();
  for (const [label, value] of Object.entries(values)) await dialog.getByLabel(label).fill(value);
  const created = page.waitForResponse(response => new URL(response.url()).pathname === endpoint && response.request().method() === 'POST');
  await dialog.getByRole('button', { name: 'Salvează' }).click();
  const response = await created;
  expect(response.status()).toBe(201);
  return response.json() as Promise<RecordWithID>;
}

async function openReactRootDetails(page: Page, exactTitle: string): Promise<void> {
  const row = page.getByText(exactTitle, { exact: true }).locator('xpath=ancestor::tr[1]');
  await expect(row).toBeVisible();
  await row.getByLabel('Acțiuni înregistrare').click();
  await clickOpenPopoverAction(page, 'Detalii');
  const details = page.locator('[role="dialog"]:visible').last();
  await expect(details).toBeVisible();
  await details.getByRole('button', { name: 'Închide' }).click();
  await expect(details).toBeHidden();
}

async function createRelatedThroughReact(
  page: Page,
  addLabel: string,
  endpoint: string,
  values: Record<string, string>,
): Promise<RecordWithID> {
  await page.getByLabel(addLabel).click();
  const dialog = page.getByRole('dialog');
  await expect(dialog).toBeVisible();
  for (const [label, value] of Object.entries(values)) await dialog.getByLabel(label).fill(value);
  const created = page.waitForResponse(response => new URL(response.url()).pathname === endpoint && response.request().method() === 'POST');
  await dialog.getByRole('button', { name: 'Salvează' }).click();
  const response = await created;
  expect(response.status()).toBe(201);
  return response.json() as Promise<RecordWithID>;
}

async function makeDirectorAndAdjunct(): Promise<{ directorID: string; adjunctID: string; directorName: string; adjunctName: string }> {
  const directorID = sql(`select id::text from app_users where sub='${director.subject}'`);
  const adjunctID = sql(`select id::text from app_users where sub='${adjunct.subject}'`);
  const directorName = `${marker} Director`;
  const adjunctName = `${marker} Adjunct`;
  // Only the role projection is seeded.  The test still proves permissions
  // from tokens and the request-time delegation lookup, not DB-only access.
  sql(`
    update app_users set name='${directorName}' where id='${directorID}';
    update app_users set name='${adjunctName}' where id='${adjunctID}';
    update app_memberships set position_code='director', active=true, start_date=current_date, end_date=null where user_id='${directorID}' and tenant_code='${school.tenant}';
    update app_memberships set position_code='director_adjunct', active=true, start_date=current_date, end_date=null where user_id='${adjunctID}' and tenant_code='${school.tenant}';
    delete from app_user_roles where user_id in ('${directorID}','${adjunctID}') and tenant_code='${school.tenant}';
    delete from app_user_platform_roles where user_id in ('${directorID}','${adjunctID}');
    insert into app_user_roles(tenant_code,user_id,role_code) values ('${school.tenant}','${directorID}','director') on conflict do nothing;
  `);
  return { directorID, adjunctID, directorName, adjunctName };
}

test('director delegation is accepted, immediately usable in React/API, and revocation removes it at request time', async ({ page, browser }) => {
  const actors = await makeDirectorAndAdjunct();
  const directorToken = await login(page, director);
  const adjunctContext = await browser.newContext({ baseURL: 'http://localhost:4174' });
  const adjunctPage = await adjunctContext.newPage();
  let adjunctToken = await login(adjunctPage, adjunct, 'http://localhost:4174');

  const before = await request<{ permissions: string[] }>(adjunctPage, adjunctToken, '/api/me');
  expect(before.status).toBe(200);
  expect(before.body.permissions).not.toContain('education.governance.manage');
  expect((await request<unknown>(adjunctPage, adjunctToken, '/api/education/governance/meetings', {
    method: 'POST', body: JSON.stringify({ school_year: '2026-2027', organism: 'ca', title: `${marker} forbidden`, meeting_type: 'ordinary', status: 'draft', quorum_required: 1, meeting_date: '2026-09-10', location: 'Sala', chairperson_user_id: actors.directorID, secretary_user_id: actors.adjunctID }),
  })).status).toBe(403);

  // Offer through the same PrimeReact component a director uses.  The user
  // chooses labelled principals and permissions; no UUID is typed into UI.
  await page.goto('/scoala');
  await expect(page.getByText('Delegări de autoritate')).toBeVisible();
  await page.getByRole('button', { name: 'Oferă delegare' }).click();
  const dialog = page.getByRole('dialog', { name: 'Oferă delegare educațională' });
  await dialog.getByRole('combobox', { name: 'Director adjunct' }).click();
  await selectOpenOption(page, new RegExp(actors.adjunctName));
  await dialog.getByRole('combobox', { name: 'Drept delegat' }).click();
  await selectOpenOption(page, 'education.governance.manage');
  const offeredResponse = page.waitForResponse(response => new URL(response.url()).pathname === '/api/education/delegations' && response.request().method() === 'POST');
  await dialog.getByRole('button', { name: 'Oferă delegarea' }).click();
  const offeredHTTP = await offeredResponse;
  expect(offeredHTTP.status()).toBe(201);
  const delegation = await offeredHTTP.json() as { id: string; status: string };
  expect(delegation.status).toBe('offered');

  await adjunctPage.goto('/scoala');
  await expect(adjunctPage.getByRole('button', { name: 'Acceptă education.governance.manage' })).toBeVisible();
  const acceptedResponse = adjunctPage.waitForResponse(response => new URL(response.url()).pathname.endsWith(`/delegations/${delegation.id}/accept`) && response.request().method() === 'POST');
  await adjunctPage.getByRole('button', { name: 'Acceptă education.governance.manage' }).click();
  expect((await acceptedResponse).status()).toBe(200);
  expect(sql(`select status || '|' || permission_code || '|' || delegate_user_id::text from education_role_delegations where id='${delegation.id}'`)).toBe(`accepted|education.governance.manage|${actors.adjunctID}`);
  expect(sql(`select count(*)::text from app_audit_log where action='education.delegations.accept' and target_id='${delegation.id}'`)).toBe('1');

  // Reload obtains a fresh request-time authorization snapshot. Assert the
  // accepted, institution-scoped grant before proving its route/API effect.
  const activeGrantsResponse = adjunctPage.waitForResponse((response) =>
    response.request().method() === 'GET'
    && new URL(response.url()).pathname === '/api/education/delegations/active-grants');
  await adjunctPage.reload();
  const activeGrantsHTTP = await activeGrantsResponse;
  expect(activeGrantsHTTP.status()).toBe(200);
  const activeGrants = await activeGrantsHTTP.json() as {
    tenant_code: string;
    institution_id: string;
    grants: Array<{ permission_code: string; resource_type: string; resource_id: string }>;
  };
  expect(activeGrants).toMatchObject({ tenant_code: school.tenant, institution_id: school.institution });
  expect(activeGrants.grants).toContainEqual({
    permission_code: 'education.governance.manage',
    resource_type: 'institution',
    // Institution scope is represented by the response envelope; an empty
    // resource_id prevents accidentally treating it as an exact resource.
    resource_id: '',
  });
  await adjunctPage.goto('/scoala/governance');
  await expect(adjunctPage.getByText('Ședințe de guvernanță')).toBeVisible();
  const delegatedCommittee = await request<{ id: string }>(adjunctPage, adjunctToken, '/api/education/committees/records', {
    method: 'POST', body: JSON.stringify({ school_year: '2026-2027', committee_type: 'curriculum', title: `${marker} comisie delegată`, status: 'active', starts_on: '2026-09-01', notes: 'created through a request-time delegated authority' }),
  });
  expect(delegatedCommittee.status).toBe(201);
  expect(sql(`select count(*)::text from education_committees where id='${delegatedCommittee.body.id}' and institution_id='${school.institution}'`)).toBe('1');

  await page.goto('/scoala');
  // Multiple CI retries can leave historical revoked rows.  Narrow the
  // server-backed table to this labelled adjunct before selecting its action.
  await page.getByLabel('Filtru Delegat').fill(actors.adjunctName);
  await expect(page.getByText(actors.adjunctName, { exact: true })).toBeVisible();
  const delegatedRow = page.getByText('education.governance.manage', { exact: true }).locator('xpath=ancestor::tr[1]');
  await delegatedRow.getByRole('button', { name: 'Revocă education.governance.manage' }).click();
  await expect.poll(() => sql(`select status from education_role_delegations where id='${delegation.id}'`)).toBe('revoked');
  await adjunctPage.reload();
  // UI authorization is recomputed from active grants; the protected route is
  // no longer rendered after revocation, and the next write is forbidden.
  await adjunctPage.goto('/scoala/governance');
  await expect(adjunctPage.getByText('Ședințe de guvernanță')).toHaveCount(0);
  expect((await request<unknown>(adjunctPage, adjunctToken, '/api/education/committees/records', {
    method: 'POST', body: JSON.stringify({ school_year: '2026-2027', committee_type: 'curriculum', title: `${marker} must fail`, status: 'active', starts_on: '2026-09-01' }),
  })).status).toBe(403);
  expect(sql(`select count(*)::text from app_audit_log where action='education.delegations.revoke' and target_id='${delegation.id}'`)).toBe('1');
  await adjunctContext.close();
});

test('governance lifecycle, committee completeness, evaluations and declarations preserve UI/RBAC/RLS contracts', async ({ page, browser }) => {
  const actors = await makeDirectorAndAdjunct();
  const token = await login(page, director);
  const title = `${marker} ședință`;

  // Create and select the meeting using its labelled row.  The child wizards
  // receive the selected context; no human-facing UUID field is exposed.
  await page.goto('/scoala/governance/ca-wizard');
  await page.getByLabel('Titlu').fill(title);
  await page.getByRole('button', { name: 'Continuă' }).click();
  await page.getByLabel('Cvorum').fill('1');
  await page.getByRole('button', { name: 'Continuă' }).click();
  await page.getByLabel('Data').fill('2026-09-10');
  await page.getByRole('button', { name: 'Continuă' }).click();
  await page.getByLabel('Locație').fill('Sala profesorală');
  await searchAndSelectWizardUser(page, 'Președinte *', actors.directorName);
  await searchAndSelectWizardUser(page, 'Secretar *', actors.adjunctName);
  await page.getByRole('button', { name: 'Continuă', exact: true }).click();
  await expect(page.getByText('4. Creare', { exact: true })).toHaveClass(/font-semibold/);
  await expect(page.getByRole('button', { name: 'Salvează', exact: true })).toBeVisible();
  const createdResponse = page.waitForResponse(response => new URL(response.url()).pathname === '/api/education/governance/meetings' && response.request().method() === 'POST');
  await page.getByRole('button', { name: 'Salvează' }).click();
  const meetingHTTP = await createdResponse; expect(meetingHTTP.status()).toBe(201);
  const meeting = await meetingHTTP.json() as { id: string };
  expect(sql(`select count(*)::text from education_meetings where id='${meeting.id}' and institution_id='${school.institution}'`)).toBe('1');

  // Membership is a first-class School screen: choose the canonical user from
  // the PrimeReact selector rather than posting a hidden ID through fetch.
  await page.goto('/scoala/governance');
  await page.getByLabel('Adaugă membri').click();
  const membershipDialog = page.getByRole('dialog');
  await membershipDialog.getByLabel('An școlar').fill('2026-2027');
  await membershipDialog.getByLabel('Organism').fill('ca');
  const eligibleMemberResponse = page.waitForResponse((candidate) => {
    const url = new URL(candidate.url());
    return candidate.request().method() === 'GET'
      && url.pathname === '/api/education/governance/eligible-users'
      && url.searchParams.get('filter.name') === actors.directorName;
  });
  await membershipDialog.getByLabel('Caută Utilizator').fill(actors.directorName);
  expect((await eligibleMemberResponse).status()).toBe(200);
  const eligibleMemberSelect = membershipDialog.getByRole('combobox', { name: 'Utilizator', exact: true });
  await expect(eligibleMemberSelect).toBeEnabled();
  await eligibleMemberSelect.click();
  await selectOpenOption(page, actors.directorName);
  await membershipDialog.getByLabel('Rol').fill('președinte');
  await membershipDialog.getByLabel('Mandat de la').fill('2026-09-01');
  await membershipDialog.getByLabel('Mandat până la').fill('2027-08-31');
  await membershipDialog.getByLabel('Stare').fill('activ');
  const membershipCreated = page.waitForResponse(response => new URL(response.url()).pathname === '/api/education/governance/memberships' && response.request().method() === 'POST');
  await membershipDialog.getByRole('button', { name: 'Salvează' }).click();
  expect((await membershipCreated).status()).toBe(201);
  await page.goto('/scoala/governance');
  const row = page.getByText(title, { exact: true }).locator('xpath=ancestor::tr[1]');
  await row.getByRole('button', { name: 'Acțiuni înregistrare' }).click();
  await clickOpenPopoverAction(page, 'Detalii');
  const meetingDetails = page.getByRole('dialog', { name: /^Ședință de guvernanță(?:\s|$)/ });
  await expect(meetingDetails).toBeVisible();
  await page.keyboard.press('Escape');
  await expect(meetingDetails).toBeHidden();
  await page.getByRole('button', { name: 'Adaugă participanți' }).click();
  const participant = page.getByRole('dialog', { name: 'Adaugă participanți' });
  await participant.getByLabel('Nume').fill(actors.directorName);
  await participant.getByLabel('Rol').fill('Director');
  await participant.getByLabel('Tip membru').fill('presedinte');
  await participant.getByLabel('Prezență').fill('prezent');
  await participant.getByLabel('Drept vot').click(); await selectOpenOption(page, 'Da');
  const participantCreated = page.waitForResponse(response => new URL(response.url()).pathname.endsWith(`/meetings/${meeting.id}/participants`) && response.request().method() === 'POST');
  await participant.getByRole('button', { name: 'Salvează' }).click(); expect((await participantCreated).status()).toBe(201);

  // Choosing the meeting and vote is an accessible Select option, never an ID
  // text box.  The API responses supply IDs only for DB/audit assertions.
  await page.getByRole('button', { name: 'Ghid vot' }).click();
  await expect(page.getByRole('heading', { name: 'Vot al ședinței' })).toBeVisible();
  await page.getByLabel('Subiect').fill(`${marker} vot`); await page.getByRole('button', { name: 'Continuă' }).click();
  await page.getByLabel('Pentru').fill('1'); await page.getByRole('button', { name: 'Continuă' }).click();
  await page.getByLabel('Temei legal').fill('ROFUIP'); await page.getByRole('button', { name: 'Continuă' }).click();
  const voteCreated = page.waitForResponse(response => new URL(response.url()).pathname.endsWith(`/meetings/${meeting.id}/votes`) && response.request().method() === 'POST');
  await page.getByRole('button', { name: 'Salvează' }).click(); const voteHTTP = await voteCreated; expect(voteHTTP.status()).toBe(201);
  const vote = await voteHTTP.json() as { id: string };
  await page.goto('/scoala/governance'); await row.getByRole('button', { name: 'Acțiuni înregistrare' }).click(); await clickOpenPopoverAction(page, 'Detalii');
  await expect(meetingDetails).toBeVisible();
  await page.keyboard.press('Escape');
  await expect(meetingDetails).toBeHidden();
  await page.getByRole('button', { name: 'Ghid minută' }).click();
  await page.getByLabel('Subiect').fill(`${marker} minută`);
  await page.getByLabel('Rezumat discuții').fill('Discuție consemnată.');
  await page.getByRole('button', { name: 'Continuă' }).click();
  await page.getByLabel('Decizie').fill('Aprobat.');
  await page.getByLabel('Responsabil').fill(actors.directorName);
  await page.getByRole('button', { name: 'Continuă' }).click();
  await page.getByRole('button', { name: 'Continuă' }).click();
  const minuteCreated = page.waitForResponse(response => new URL(response.url()).pathname.endsWith(`/meetings/${meeting.id}/minutes`) && response.request().method() === 'POST');
  await page.getByRole('button', { name: 'Salvează' }).click(); expect((await minuteCreated).status()).toBe(201);
  await page.goto('/scoala/governance'); await row.getByRole('button', { name: 'Acțiuni înregistrare' }).click(); await clickOpenPopoverAction(page, 'Detalii');
  await expect(meetingDetails).toBeVisible();
  await page.keyboard.press('Escape');
  await expect(meetingDetails).toBeHidden();
  await page.getByRole('button', { name: 'Ghid hotărâre' }).click();
  await page.getByRole('combobox', { name: 'Vot' }).click(); await selectOpenOption(page, new RegExp(`${marker} vot`));
  await page.getByLabel('Titlu').fill(`${marker} hotărâre`); await page.getByRole('button', { name: 'Continuă' }).click(); await page.getByLabel('Data emiterii').fill('2026-09-10'); await page.getByRole('button', { name: 'Continuă' }).click(); await page.getByLabel('Semnat de').fill(actors.directorName); await page.getByRole('button', { name: 'Continuă' }).click();
  const resolutionCreated = page.waitForResponse(response => new URL(response.url()).pathname.endsWith(`/meetings/${meeting.id}/resolutions`) && response.request().method() === 'POST');
  await page.getByRole('button', { name: 'Salvează' }).click(); expect((await resolutionCreated).status()).toBe(201);
  expect(sql(`select count(*)::text from education_meeting_votes where id='${vote.id}' and meeting_id='${meeting.id}'`)).toBe('1');
  expect(sql(`select count(*)::text from app_audit_log where target_id='${meeting.id}' and action like 'education.governance.%'`)).not.toBe('0');

  const committeeTitle = `${marker} CE`;
  const committee = await createRootThroughReact(page, '/scoala/committees', '/api/education/committees/records', { 'An școlar': '2026-2027', Tip: 'evaluare_personal_didactic', 'Denumire comisie': committeeTitle, Stare: 'active', 'Începe la': '2026-09-01' });
  await openReactRootDetails(page, committeeTitle);
  const member = await createRelatedThroughReact(page, 'Adaugă membri comisie', `/api/education/committees/records/${committee.id}/members`, { 'Nume complet': actors.directorName, Rol: 'Președinte', Tip: 'cadru_didactic', Stare: 'active', 'Numit la': '2026-09-01' });
  expect(member.id).toEqual(expect.any(String));
  const completeness = await request<{ complete: boolean; active_members?: string[] }>(page, token, `/api/education/committees/records/${committee.id}/completeness-summary`);
  expect(completeness.status).toBe(200); expect(completeness.body.active_members).toContain(actors.directorName);

  const employeeCode = `E2E-${Date.now()}`;
  const evaluation = await createRootThroughReact(page, '/scoala/evaluations', '/api/education/evaluations/records', { 'Cod angajat': employeeCode, 'Nume complet': actors.adjunctName, Funcție: 'Profesor', 'An școlar': '2026-2027', Stare: 'draft', Punctaj: '75', Rezumat: marker });
  const declaration = await createRootThroughReact(page, '/scoala/declarations', '/api/education/declarations/records', { 'Cod angajat': employeeCode, 'Nume complet': actors.adjunctName, Tip: 'interese', 'An școlar': '2026-2027', Stare: 'draft', 'Depus la': '2026-09-10', Rezumat: marker });
  expect(sql(`select count(*)::text from education_evaluations where id='${evaluation.id}'`)).toBe('1');
  expect(sql(`select count(*)::text from education_declarations where id='${declaration.id}'`)).toBe('1');

  // An ordinary adjunct receives neither evaluation nor declaration management.
  const restricted = await browser.newContext({ baseURL: 'http://localhost:4174' }); const restrictedPage = await restricted.newPage(); const restrictedToken = await login(restrictedPage, adjunct, 'http://localhost:4174');
  const evaluationPayload = { employee_code: employeeCode, full_name: actors.adjunctName, role_title: 'Profesor', school_year: '2026-2027', status: 'draft' };
  expect((await request<unknown>(restrictedPage, restrictedToken, '/api/education/evaluations/records', { method: 'POST', body: JSON.stringify(evaluationPayload) })).status).toBe(403);
  expect((await request<unknown>(restrictedPage, restrictedToken, '/api/education/declarations/records', { method: 'POST', body: JSON.stringify({ employee_code: employeeCode, full_name: actors.adjunctName, declaration_type: 'interese', school_year: '2026-2027', status: 'draft', submitted_on: '2026-09-10' }) })).status).toBe(403);
  const isolated = await browser.newContext({ baseURL: 'http://localhost:4175' }); const isolatedPage = await isolated.newPage(); const isolatedToken = await login(isolatedPage, otherTenant, 'http://localhost:4175');
  expect((await request<unknown>(isolatedPage, isolatedToken, `/api/education/committees/records/${committee.id}`)).status).toBe(404);
  expect(sql(`select count(*)::text from education_committees where id='${committee.id}'`, balotesti, false)).toBe('0');
  await restricted.close(); await isolated.close();
});
