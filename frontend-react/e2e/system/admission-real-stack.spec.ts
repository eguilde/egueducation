import { expect, test, type Page } from '@playwright/test';
import { createHash } from 'node:crypto';

/*
 * No route interception or database writes live in this file. The test creates
 * its isolated prerequisites through the same published HTTP commands used by
 * the UI, after a real OIDC login.
 */
const origin = 'http://localhost:4173';
const director = { identifier: 'oidc.browser.fixture@example.test', otp: '173829' };
const otherTenant = { identifier: 'oidc.balotesti.fixture@example.test', otp: '739204' };
const marker = `ADMISSION-E2E-${process.env.GITHUB_RUN_ID ?? 'local'}-${Date.now()}`.replace(/[^A-Za-z0-9-]/g, '');

type Fixture = {
  classSearch: string; classLabel: string; classIDPrefix: string;
  authorizationSearch: string; authorizationLabel: string; sourceSearch: string; sourceLabel: string;
	retentionSourceID: string;
  schoolYear: string; effectiveFrom: string;
  candidateSearch: string; candidateLabel: string; secondCandidatePartyID: string;
};
type TokenResponse = { access_token: string };
type Command = { id: string; status: string; expected_version?: number };
type Detail = { application: { id: string; expected_version: number; status: string }; documents: Array<{ id: string; expected_version: number }>; assessments: Array<{ criterion_id: string; expected_version: number }>; decisions: Array<{ id: string; outcome: string }> };
type PageResult<T> = { items: T[] };
type AdminUser = { id: string; email: string };
type Membership = {
  id: string; user_id: string; position_code: string; org_unit_code: string;
  organization_name: string; is_primary: boolean; active: boolean;
  start_date: string; end_date: string;
};
type RetentionRule = { id: string; status: string; expected_version: number };
type LegalPreparation = { id: string; artifact_id: string; canonical_payload_sha256: string; canonical_payload_base64: string; resulting_decision_id?: string; resulting_decision_payload_base64?: string };
const systemTestCertificateSHA256 = createHash('sha256').update('egueducation-system-test-leaf-certificate-v1').digest('hex');

async function login(page: Page, actor: typeof director, expectedOrigin = origin): Promise<string> {
  let token: string | undefined;
  page.on('response', async response => {
    if (response.request().method() !== 'POST' || !response.url().includes('/api/oidc/token')) return;
    token ??= (await response.json().catch(() => undefined) as TokenResponse | undefined)?.access_token;
  });
  await page.goto(`${expectedOrigin}/`, { waitUntil: 'domcontentloaded' });
  await page.getByRole('button', { name: 'Autentificare' }).last().click();
  await expect(page).toHaveURL(/\/api\/oidc\/authorize/);
  await page.getByRole('button', { name: /SMS/ }).click();
  await page.getByLabel('Utilizator, email sau numar de telefon').fill(actor.identifier);
  await page.getByLabel('Canal OTP').selectOption('sms');
  await page.getByRole('button', { name: 'Trimite codul' }).click();
  const boxes = page.locator('.otp-box');
  await expect(boxes).toHaveCount(6);
  await boxes.first().click();
  await boxes.first().pressSequentially(actor.otp);
  await expect(page.getByRole('button', { name: 'Verifica codul' })).toBeFocused();
  await page.getByRole('button', { name: 'Verifica codul' }).click();
  const consent = page.getByRole('button', { name: 'Accepta si continua' });
  if (await consent.count()) await consent.click();
  await expect(page).toHaveURL(`${expectedOrigin}/`);
  await expect(page.getByRole('button', { name: 'Deconectare' })).toBeVisible();
  await expect.poll(() => token, { message: 'Real OIDC token exchange was not observed.' }).toBeTruthy();
  return token!;
}

async function api<T>(page: Page, token: string, path: string, init: RequestInit = {}): Promise<{ status: number; body: T }> {
  return page.evaluate(async ({ bearer, url, input }) => {
    const headers = new Headers(input.headers);
    headers.set('Authorization', `Bearer ${bearer}`);
    headers.set('Content-Type', 'application/json');
    if (input.method && input.method !== 'GET') headers.set('Idempotency-Key', crypto.randomUUID());
    const response = await fetch(url, { ...input, headers, credentials: 'include' });
    const text = await response.text();
    return { status: response.status, body: text ? JSON.parse(text) : null };
  }, { bearer: token, url: path, input: init });
}

function today(): string { return new Date().toISOString().slice(0, 10); }
function currentSchoolYear(date = new Date()): string {
  const year = date.getUTCFullYear() - (date.getUTCMonth() < 8 ? 1 : 0);
  return `${year}-${year + 1}`;
}

function singlePagePDF(): string {
  const objects = [
    '<</Type/Catalog/Pages 2 0 R>>',
    '<</Type/Pages/Count 1/Kids[3 0 R]>>',
    '<</Type/Page/Parent 2 0 R/MediaBox[0 0 200 200]/Contents 4 0 R/Resources<</Font<</F1 5 0 R>>>>>>',
    '<</Length 51>>stream\nBT /F1 12 Tf 20 100 Td (Admission evidence) Tj ET\nendstream',
    '<</Type/Font/Subtype/Type1/BaseFont/Helvetica>>',
  ];
  let body = '%PDF-1.4\n'; const offsets: number[] = [];
  for (const [index, object] of objects.entries()) { offsets.push(new TextEncoder().encode(body).length); body += `${index + 1} 0 obj\n${object}\nendobj\n`; }
  const xref = new TextEncoder().encode(body).length;
  body += `xref\n0 ${objects.length + 1}\n0000000000 65535 f \n${offsets.map(offset => `${String(offset).padStart(10, '0')} 00000 n \n`).join('')}trailer\n<</Size ${objects.length + 1}/Root 1 0 R>>\nstartxref\n${xref}\n%%EOF\n`;
  return body;
}

type PreparationArtifact = { preparation_id: string; artifact_slot: 'primary' | 'resulting_decision'; document: { id: string; status: string }; version: { id: string; document_id: string } };

async function uploadPreparationArtifact(page: Page, token: string, preparationID: string, artifactSlot: PreparationArtifact['artifact_slot']): Promise<PreparationArtifact> {
  const path = `/api/admissions/legal-preparations/${preparationID}/artifacts/${artifactSlot}`;
  const uploaded = await page.evaluate(async ({ bearer, endpoint, pdf }) => {
    const body = new FormData();
    body.set('file', new File([pdf], 'synthetic-dss-signed.pdf', { type: 'application/pdf' }));
    const response = await fetch(endpoint, { method: 'POST', headers: { Authorization: `Bearer ${bearer}`, 'Idempotency-Key': crypto.randomUUID() }, credentials: 'include', body });
    const text = await response.text();
    return { status: response.status, body: text ? JSON.parse(text) : null };
  }, { bearer: token, endpoint: path, pdf: singlePagePDF() }) as { status: number; body: PreparationArtifact };
  expect(uploaded.status).toBe(201);
  expect(uploaded.body.preparation_id).toBe(preparationID);
  expect(uploaded.body.artifact_slot).toBe(artifactSlot);
  await expect.poll(async () => {
    const recovered = await api<PreparationArtifact>(page, token, path);
    return recovered.status === 200 ? recovered.body : undefined;
  }, { timeout: 60_000, intervals: [500, 1_000, 2_000] }).toMatchObject({ preparation_id: preparationID, artifact_slot: artifactSlot, document: { status: 'ready' } });
  const recovered = await api<PreparationArtifact>(page, token, path);
  expect(recovered.status).toBe(200);
  expect(recovered.body.document.id).toBe(uploaded.body.document.id);
  expect(recovered.body.version.id).toBe(uploaded.body.version.id);
  expect(recovered.body.version.document_id).toBe(recovered.body.document.id);
  return recovered.body;
}

/**
 * Retention authority needs a distinct effective director. The ordinary
 * approver fixture already completes the same OIDC flow as a real user; this
 * helper grants its second, non-primary director membership through the
 * tenant's published admin command, then obtains a new token carrying the
 * resulting RBAC permissions. No database write is used.
 */
async function retentionApprover(browser: import('@playwright/test').Browser, page: Page, directorToken: string): Promise<{ context: import('@playwright/test').BrowserContext; page: Page; token: string; userID: string }> {
  const fixture = { identifier: 'oidc.approver.fixture@example.test', otp: '428615' };
  const initialContext = await browser.newContext({ baseURL: 'http://localhost:4174' });
  const initialPage = await initialContext.newPage();
  await login(initialPage, fixture, 'http://localhost:4174');
  await initialContext.close();

  const users = await api<PageResult<AdminUser>>(page, directorToken, `/api/admin/users?filter.email=${encodeURIComponent(fixture.identifier)}&page=1&pageSize=20&sort=email&direction=asc`);
  expect(users.status).toBe(200);
  const user = users.body.items.find(item => item.email.toLowerCase() === fixture.identifier);
  expect(user, 'OIDC approver fixture must exist in the same tenant user directory.').toBeTruthy();

  const memberships = await api<PageResult<Membership>>(page, directorToken, '/api/admin/memberships?page=1&pageSize=100&sort=user_name&direction=asc');
  expect(memberships.status).toBe(200);
  const userMemberships = memberships.body.items.filter(item => item.user_id === user!.id);
  const todayDate = today();
  const activeDirector = userMemberships.find(item => item.position_code === 'director'
    && item.active && item.start_date <= todayDate && (!item.end_date || item.end_date >= todayDate));
  if (!activeDirector) {
    const basis = userMemberships.find(item => item.active) ?? userMemberships[0];
    expect(basis, 'OIDC approver fixture must have an existing tenant membership to establish its institution scope.').toBeTruthy();
    const assigned = await api<Membership>(page, directorToken, '/api/admin/memberships', {
      method: 'POST',
      body: JSON.stringify({
        id: '', user_id: user!.id, position_code: 'director',
        org_unit_code: basis!.org_unit_code, organization_name: basis!.organization_name,
        is_primary: false, active: true, start_date: todayDate, end_date: '',
      }),
    });
    expect(assigned.status, 'Admin command must grant the independent effective director membership.').toBe(201);
    expect(assigned.body.position_code).toBe('director');
  }

  const context = await browser.newContext({ baseURL: 'http://localhost:4174' });
  const approverPage = await context.newPage();
  const approverToken = await login(approverPage, fixture, 'http://localhost:4174');
  const me = await api<{ permissions: string[]; user: { id: string } }>(approverPage, approverToken, '/api/me');
  expect(me.status).toBe(200);
  expect(me.body.permissions).toEqual(expect.arrayContaining(['education.admissions.retention.approve', 'education.admissions.signer.approve']));
  return { context, page: approverPage, token: approverToken, userID: me.body.user.id };
}

async function bootstrapFixture(page: Page, token: string): Promise<Fixture> {
  const key = `ADM${Date.now().toString(36).toUpperCase()}`;
  const label = `Admission ${key}`;
  const effectiveFrom = today();
  const schoolYear = currentSchoolYear();
  const create = async <T>(path: string, body: unknown): Promise<T> => {
    const result = await api<T>(page, token, path, { method: 'POST', body: JSON.stringify(body) });
    expect(result.status, `${path} fixture create: ${JSON.stringify(result.body)}`).toBe(201);
    return result.body;
  };
  const location = await create<{ id: string }>('/api/institution/locations', { code: `${key}LOC`, name: `${label} location`, address: 'CI', active: true, effective_from: effectiveFrom, effective_to: null, idempotency_key: crypto.randomUUID() });
  const offering = await create<{ id: string }>('/api/institution/education-offerings', { code: `${key}OFF`, education_level: 'PRIMARY', specialization_code: '', language_code: 'ro', title: `${label} offering`, active: true, effective_from: effectiveFrom, effective_to: null, idempotency_key: crypto.randomUUID() });
  await create<{ id: string }>('/api/institution/offering-authorizations', { offering_id: offering.id, location_id: location.id, status: 'accredited', authority_name: 'ARACIP', decision_reference: `${key}-AUTH`, capacity: 2, capacity_unit: 'students', shift: 'day', effective_from: effectiveFrom, effective_to: null, source: { source_kind: 'accreditation', citation: `${label} authorization`, article_reference: '1', issuer: 'ARACIP', source_url: `https://example.test/${key}`, published_on: effectiveFrom, consolidated_on: null, checksum_sha256: 'a'.repeat(64) }, replaces_authorization_id: null, expected_version: null, idempotency_key: crypto.randomUUID() });
	const sources = await api<{ items: Array<{ id: string }> }>(page, token, `/api/admissions/regulatory-sources?q=${encodeURIComponent(label)}&page=1&pageSize=10&sort=citation&direction=asc`);
	expect(sources.status).toBe(200); expect(sources.body.items[0]).toBeTruthy();
  const schoolClass = await create<{ id: string }>('/api/education/classes', { class_code: `${key}CLS`, class_name: `${label} class`, school_year: schoolYear, grade_level: 'I', study_shift: 'day', active: true });
  await create<{ id: string }>('/api/registratura/parties', { code: `${key}C1`, party_type: 'physical', display_name: `${label} candidate one`, first_name: 'Candidate', last_name: key, active: true });
  const secondCandidate = await create<{ id: string }>('/api/registratura/parties', { code: `${key}C2`, party_type: 'physical', display_name: `${label} candidate two`, first_name: 'Candidate', last_name: `${key}B`, active: true });
  return { classSearch: key, classLabel: label, classIDPrefix: schoolClass.id.slice(0, 8), authorizationSearch: `${key}OFF`, authorizationLabel: label, sourceSearch: label, sourceLabel: label, retentionSourceID: sources.body.items[0]!.id, schoolYear, effectiveFrom, candidateSearch: label, candidateLabel: `${label} candidate one`, secondCandidatePartyID: secondCandidate.id };
}

async function choose(page: Page, label: string, option: string | RegExp): Promise<void> {
  await page.getByRole('combobox', { name: label, exact: true }).click();
  const list = page.locator('[role="listbox"]:visible').last();
  await expect(list).toBeVisible();
  await list.getByRole('option', { name: option, exact: typeof option === 'string' }).click();
}

test('admission browser workflow proves OIDC, React/OpenAPI, RBAC, WORM, capacity, appeal and enrolment', async ({ page, browser }) => {
  const token = await login(page, director);
  const me = await api<{ permissions: string[]; user: { id: string } }>(page, token, '/api/me');
  expect(me.status).toBe(200);
  expect(me.body.permissions).toEqual(expect.arrayContaining(['education.admissions.read', 'education.admissions.manage', 'education.admissions.decide', 'education.admissions.appeals.manage', 'education.admissions.retention.manage', 'education.admissions.signer.manage', 'education.admissions.signer.approve']));
  const f = await bootstrapFixture(page, token);
  const approver = await retentionApprover(browser, page, token);
  const proposed = await api<RetentionRule>(page, token, '/api/admissions/retention-rule-versions', {
    method: 'POST',
    body: JSON.stringify({ artifact_kind: 'admission_dss', minimum_retention_days: 1, effective_from: f.effectiveFrom, effective_to: null, source_id: f.retentionSourceID }),
  });
  expect(proposed.status).toBe(201);
  expect(proposed.body.status).toBe('proposed');
  const approved = await api<RetentionRule>(approver.page, approver.token, '/api/admissions/retention-rule-versions/approve', {
    method: 'POST', body: JSON.stringify({ rule_version_id: proposed.body.id }),
  });
  expect(approved.status).toBe(200);
  expect(approved.body.status).toBe('active');
  const retention = await api<Command>(page, token, '/api/admissions/dss-retention-policies', {
    method: 'POST', body: JSON.stringify({ rule_version_id: proposed.body.id, effective_from: f.effectiveFrom }),
  });
  expect(retention.status).toBe(201);

  // The verifier fixture is intentionally synthetic: this validates the
  // browser/API/DB/DSS contract, not a cryptographic signature implementation.
  // Production remains fail-closed against the real DSS verifier.
  const signerProposal = await api<Command>(page, token, '/api/admissions/signer-authorizations', {
    method: 'POST', body: JSON.stringify({ certificate_sha256: systemTestCertificateSHA256, user_id: me.body.user.id, permission_code: 'education.admissions.decide', valid_until: '2027-09-12T00:00:00Z' }),
  });
  expect(signerProposal.status).toBe(201);
  const signerApproval = await api<Command>(approver.page, approver.token, '/api/admissions/signer-authorizations/approve', { method: 'POST', body: JSON.stringify({ proposal_id: signerProposal.body.id }) });
  expect(signerApproval.status).toBe(200);
  const appealSignerProposal = await api<Command>(page, token, '/api/admissions/signer-authorizations', {
    method: 'POST', body: JSON.stringify({ certificate_sha256: systemTestCertificateSHA256, user_id: approver.userID, permission_code: 'education.admissions.appeals.manage', valid_until: '2027-09-12T00:00:00Z' }),
  });
  expect(appealSignerProposal.status).toBe(201);
  expect((await api<Command>(approver.page, approver.token, '/api/admissions/signer-authorizations/approve', { method: 'POST', body: JSON.stringify({ proposal_id: appealSignerProposal.body.id }) })).status).toBe(200);

  // The React table must call the server with its filter/pagination contract.
  await page.goto('/scoala/admitere');
  await expect(page.getByRole('heading', { name: 'Admitere' })).toBeVisible();
  const pageResponse = page.waitForResponse(response => new URL(response.url()).pathname === '/api/admissions/campaigns' && new URL(response.url()).searchParams.get('page') === '1');
  await page.getByLabel('Filtru Campanie').fill(marker);
  expect((await pageResponse).status()).toBe(200);

  // Create the authorized class context and campaign entirely through UI. The
  // supplied fixture is an eligible offering/location/authorization/source;
  // context, campaign and lifecycle mutations are published commands.
  await page.getByLabel('Adaugă context autorizat').click();
  const contextDialog = page.getByRole('dialog', { name: 'Context autorizat de admitere' });
  await contextDialog.getByLabel('Caută Clasă').fill(f.classSearch);
  await choose(page, 'Clasă', new RegExp(f.classLabel));
  await contextDialog.getByLabel('Caută autorizare').fill(f.authorizationSearch);
  await choose(page, 'Autorizare ofertă', new RegExp(f.authorizationLabel));
  await contextDialog.getByLabel('An școlar context').fill(f.schoolYear);
  await contextDialog.getByLabel('Valabil de la').fill(f.effectiveFrom);
  const contextCreated = page.waitForResponse(response => new URL(response.url()).pathname === '/api/admissions/class-offering-contexts' && response.request().method() === 'POST');
  await contextDialog.getByRole('button', { name: 'Creează contextul' }).click();
  const context = await (await contextCreated).json() as Command;
  expect(context.status).toBe('active');

  await page.getByLabel('Adaugă campanie').click();
  const campaignDialog = page.getByRole('dialog', { name: 'Campanie de admitere' });
  await campaignDialog.getByLabel('Caută context autorizat').fill(f.classIDPrefix);
  await choose(page, 'Context autorizat', new RegExp(`${f.schoolYear}.*${f.classIDPrefix}`, 'i'));
  await campaignDialog.getByLabel('Caută Referință legală').fill(f.sourceSearch);
  await choose(page, 'Referință legală', new RegExp(f.sourceLabel));
  const campaignCode = `${marker}-CAMPAIGN`;
  await campaignDialog.getByLabel('Cod').fill(campaignCode);
  await campaignDialog.getByLabel('Titlu').fill(marker);
  await campaignDialog.getByLabel('An școlar').fill(f.schoolYear);
  await campaignDialog.getByLabel('Deschide la').fill(f.effectiveFrom);
  await campaignDialog.getByLabel('Închide la').fill(f.effectiveFrom);
  await campaignDialog.getByLabel('Capacitate').fill('1');
  await campaignDialog.getByLabel('Locuri elevi').fill('1');
  const campaignCreated = page.waitForResponse(response => new URL(response.url()).pathname === '/api/admissions/campaigns' && response.request().method() === 'POST');
  await campaignDialog.getByRole('button', { name: 'Creează campania' }).click();
  const campaign = await (await campaignCreated).json() as Command;
  expect(campaign.status).toBe('draft');

  const campaignRow = page.getByText(campaignCode, { exact: true }).locator('xpath=ancestor::tr[1]');
  await expect(campaignRow).toBeVisible();
  await campaignRow.getByRole('button', { name: 'Configurare' }).click();
  const configure = page.getByRole('dialog', { name: `Configurare · ${campaignCode}` });
  await configure.getByLabel('Cod configurare').fill('ELIG');
  await configure.getByLabel('Titlu configurare').fill('Eligibilitate');
  const criterionCreated = page.waitForResponse(response => new URL(response.url()).pathname.endsWith('/criteria') && response.request().method() === 'POST');
  await configure.getByRole('button', { name: 'Adaugă' }).click();
  expect((await criterionCreated).status()).toBe(201);
  await configure.getByRole('button', { name: 'Document' }).click();
  await configure.getByLabel('Cod configurare').fill('IDENTITY');
  await configure.getByLabel('Titlu configurare').fill('Identitate');
  const requirementCreated = page.waitForResponse(response => new URL(response.url()).pathname.endsWith('/document-requirements') && response.request().method() === 'POST');
  await configure.getByRole('button', { name: 'Adaugă' }).click();
  expect((await requirementCreated).status()).toBe(201);
  await configure.getByLabel('Închide dialogul').click();
  for (const next of ['published', 'open']) {
    const transitioned = page.waitForResponse(response => new URL(response.url()).pathname === `/api/admissions/campaigns/${campaign.id}/transitions` && response.request().method() === 'POST');
    await campaignRow.getByRole('button', { name: `${next} ${campaignCode}` }).click();
    expect((await transitioned).status()).toBe(200);
  }

  // Create through the PrimeReact dialog: selectable campaign/candidate labels
  // prove there is no hidden client-side UUID injection.
  await page.getByRole('button', { name: 'Aplicații' }).click();
  await page.getByLabel('Adaugă aplicație').click();
  const create = page.getByRole('dialog', { name: 'Aplicație nouă' });
  await create.getByLabel('Caută Campanie').fill(campaignCode);
  await choose(page, 'Campanie', new RegExp(campaignCode));
  await create.getByLabel('Caută Candidat').fill(f.candidateSearch);
  await choose(page, 'Candidat', new RegExp(f.candidateLabel));
  await create.getByLabel('Număr aplicație').fill(`${marker}-A1`);
  const createdResponse = page.waitForResponse(response => new URL(response.url()).pathname === '/api/admissions/applications' && response.request().method() === 'POST');
  await create.getByRole('button', { name: 'Creează aplicația' }).click();
  const application = await (await createdResponse).json() as Command;
  expect(application.status).toBe('draft');

  // Submission/review/document and criterion transitions are rendered by the
  // app. The first application has a documented dispensation; its signed
  // decision is then the first governed WORM artifact available to selectors.
  await page.getByLabel(`Deschide ${marker}-A1`).click();
  await page.getByRole('button', { name: 'Depune' }).click();
  await page.getByRole('button', { name: 'Începe verificarea' }).click();
  await page.getByRole('button', { name: 'Verifică' }).click();
  const document = page.getByRole('dialog', { name: 'Verificare document' });
  await document.getByLabel('Notă document').fill('dispensă documentată pentru fluxul de test');
  await document.getByRole('button', { name: 'Dispensă' }).click();
  await page.getByRole('button', { name: 'Evaluează' }).click();
  const assessment = page.getByRole('dialog', { name: 'Evaluare criteriu' });
  await assessment.getByLabel('Motivare criteriu').fill('criteriu îndeplinit');
  await assessment.getByRole('button', { name: 'Salvează evaluarea' }).click();
  await page.getByRole('button', { name: 'Emite decizie' }).click();
  const decision = page.getByRole('dialog', { name: 'Emitere decizie' });
  await decision.getByLabel('Număr decizie').fill(`${marker}-D1`);
  await decision.getByLabel('Motivare decizie').fill('dosar complet');
  const preparedDecisionResponse = page.waitForResponse(response => new URL(response.url()).pathname.endsWith('/decision-preparations') && response.request().method() === 'POST');
  await decision.getByRole('button', { name: 'Pregătește pentru semnare' }).click();
  const preparedDecision = await (await preparedDecisionResponse).json() as LegalPreparation;
  expect(preparedDecision.canonical_payload_sha256).toMatch(/^[a-f0-9]{64}$/);
  expect(preparedDecision.canonical_payload_base64).toBeTruthy();
  const payloadDownload = page.waitForEvent('download');
  await decision.getByRole('button', { name: 'Descarcă payload' }).click();
  const downloadedPayload = await payloadDownload;
  expect(downloadedPayload.suggestedFilename()).toContain(preparedDecision.id);
  const payloadStream = await downloadedPayload.createReadStream();
  expect(payloadStream).toBeTruthy();
  const payloadChunks: Buffer[] = [];
  for await (const chunk of payloadStream!) payloadChunks.push(Buffer.from(chunk));
  const canonicalPayloadBytes = Buffer.concat(payloadChunks);
  expect(canonicalPayloadBytes).toEqual(Buffer.from(preparedDecision.canonical_payload_base64, 'base64'));
  expect(createHash('sha256').update(canonicalPayloadBytes).digest('hex')).toBe(preparedDecision.canonical_payload_sha256);
  const decisionArtifactPath = `/api/admissions/legal-preparations/${preparedDecision.id}/artifacts/primary`;
  const decisionArtifactResponse = page.waitForResponse(response => new URL(response.url()).pathname === decisionArtifactPath && response.request().method() === 'POST');
  await decision.locator('input[type="file"]').setInputFiles({ name: 'synthetic-dss-signed.pdf', mimeType: 'application/pdf', buffer: Buffer.from(singlePagePDF()) });
  await decision.getByRole('button', { name: 'Încarcă documentul semnat' }).click();
  const decisionArtifact = await (await decisionArtifactResponse).json() as PreparationArtifact;
  expect(decisionArtifact.preparation_id).toBe(preparedDecision.id);
  expect(decisionArtifact.artifact_slot).toBe('primary');
  await expect(decision.getByText('Versiunea WORM este pregătită pentru finalizare.')).toBeVisible({ timeout: 60_000 });
  const decisionResponse = page.waitForResponse(response => new URL(response.url()).pathname === '/api/admissions/legal-preparations/finalize' && response.request().method() === 'POST');
  await decision.getByRole('button', { name: 'Finalizează decizia semnată' }).click();
  expect((await decisionResponse).status()).toBe(201);

  // The only archive evidence used below is created by the governed
  // preparation-bound WORM endpoint. Re-read it through the public selector;
  // this proves the immutable object version and retention invariants rather
  // than manufacturing an archive database row in the fixture.
  const archiveLabel = 'Admission legal artifact primary';
  const archive = { documentID: decisionArtifact.document.id, versionID: decisionArtifact.version.id, search: archiveLabel, label: archiveLabel };
  await expect.poll(async () => {
    const result = await api<{ items: Array<{ document_id: string; version_id: string }> }>(page, token, `/api/admissions/eligible-archive-versions?purpose=application_document&q=${encodeURIComponent(archive.search)}&page=1&pageSize=10&sort=title&direction=asc`);
    return result.status === 200 ? result.body.items.find(item => item.document_id === archive.documentID && item.version_id === archive.versionID) : undefined;
  }, { timeout: 60_000, intervals: [500, 1_000, 2_000] }).toBeTruthy();

  // Use public commands for a second complete application: admission capacity
  // must reject its decision, rather than letting a client counter decide.
  const second = await api<Command>(page, token, '/api/admissions/applications', { method: 'POST', body: JSON.stringify({ campaign_id: campaign.id, application_no: `${marker}-A2`, candidate_party_id: f.secondCandidatePartyID, consent_snapshot: { confirmed: true } }) });
  expect(second.status).toBe(201);
  let secondDetail = await api<Detail>(page, token, `/api/admissions/applications/${second.body.id}`);
  for (const status of ['submitted', 'under_review']) {
    expect((await api<Command>(page, token, `/api/admissions/applications/${second.body.id}/transitions`, { method: 'POST', body: JSON.stringify({ status, expected_version: secondDetail.body.application.expected_version }) })).status).toBe(200);
    secondDetail = await api<Detail>(page, token, `/api/admissions/applications/${second.body.id}`);
  }
  expect((await api<Command>(page, token, `/api/admissions/applications/${second.body.id}/documents/${secondDetail.body.documents[0]!.id}`, { method: 'POST', body: JSON.stringify({ status: 'accepted', review_note: 'WORM', expected_version: secondDetail.body.documents[0]!.expected_version, archive: { document_id: archive.documentID, version_id: archive.versionID } }) })).status).toBe(200);
  secondDetail = await api<Detail>(page, token, `/api/admissions/applications/${second.body.id}`);
  expect((await api<Command>(page, token, `/api/admissions/applications/${second.body.id}/assessments`, { method: 'POST', body: JSON.stringify({ criterion_id: secondDetail.body.assessments[0]!.criterion_id, outcome: 'met', rationale: 'met', evidence_snapshot: {}, expected_version: secondDetail.body.assessments[0]!.expected_version }) })).status).toBe(200);
  secondDetail = await api<Detail>(page, token, `/api/admissions/applications/${second.body.id}`);
  expect((await api<unknown>(page, token, `/api/admissions/applications/${second.body.id}/decisions`, { method: 'POST', body: JSON.stringify({ decision_no: `${marker}-D2`, outcome: 'admitted', rationale: 'legacy must fail closed', expected_version: secondDetail.body.application.expected_version, archive: { document_id: archive.documentID, version_id: archive.versionID } }) })).status).toBe(409);
  expect((await api<unknown>(page, token, `/api/admissions/applications/${second.body.id}/decision-preparations`, { method: 'POST', body: JSON.stringify({ decision_no: `${marker}-D2`, outcome: 'admitted', rationale: 'capacity invariant', expected_version: secondDetail.body.application.expected_version }) })).status).toBe(422);
  const cancellable = await api<LegalPreparation>(page, token, `/api/admissions/applications/${second.body.id}/decision-preparations`, { method: 'POST', body: JSON.stringify({ decision_no: `${marker}-D2W`, outcome: 'waitlisted', rationale: 'cancellation proof', expected_version: secondDetail.body.application.expected_version }) });
  expect(cancellable.status).toBe(201);
  expect((await api<Command>(page, token, `/api/admissions/legal-preparations/${cancellable.body.id}/cancel`, { method: 'POST', body: JSON.stringify({}) })).status).toBe(201);

  // A second OIDC identity performs the resolution; the server enforces that
  // it is not the original decision actor. The browser never receives DB data.
  await page.getByRole('button', { name: 'Contestație' }).click();
  const appeal = page.getByRole('dialog', { name: 'Contestație' });
  await appeal.getByLabel('Număr contestație').fill(`${marker}-C1`);
  await appeal.getByLabel('Motivare contestație').fill('reverificare');
  await appeal.getByLabel('Caută versiune arhivă').fill(archive.search);
  await choose(page, 'Versiune arhivă WORM', new RegExp(archive.label));
  await appeal.getByRole('button', { name: 'Înregistrează contestația' }).click();
  const appeals = await api<{ items: Array<{ id: string; expected_version: number }> }>(approver.page, approver.token, `/api/admissions/appeals?page=1&pageSize=20&sort=submitted_at&direction=desc&filter.appeal_no=${marker}`);
  expect(appeals.status).toBe(200); expect(appeals.body.items).toHaveLength(1);
  const finalDetail = await api<Detail>(approver.page, approver.token, `/api/admissions/applications/${application.id}`);
  // Legacy one-step resolution stays fail-closed. A favourable resolution has
  // two independently signed artifacts: the appeal resolution and resulting
  // replacement decision, each stored as a distinct WORM version.
  expect((await api<unknown>(approver.page, approver.token, `/api/admissions/appeals/${appeals.body.items[0]!.id}/resolution`, { method: 'POST', body: JSON.stringify({ outcome: 'upheld', rationale: 'legacy must fail closed', expected_version: appeals.body.items[0]!.expected_version, application_expected_version: finalDetail.body.application.expected_version, resulting_decision_no: `${marker}-D1R`, resulting_outcome: 'admitted', archive: { document_id: archive.documentID, version_id: archive.versionID } }) })).status).toBe(409);
  const preparedAppeal = await api<LegalPreparation>(approver.page, approver.token, `/api/admissions/appeals/${appeals.body.items[0]!.id}/resolution-preparations`, { method: 'POST', body: JSON.stringify({ outcome: 'upheld', rationale: 'analiză independentă favorabilă', expected_version: appeals.body.items[0]!.expected_version, application_expected_version: finalDetail.body.application.expected_version, resulting_decision_no: `${marker}-D1R`, resulting_outcome: 'admitted' }) });
  expect(preparedAppeal.status).toBe(201);
  expect(preparedAppeal.body.canonical_payload_base64).toBeTruthy();
  expect(preparedAppeal.body.resulting_decision_id).toBeTruthy();
  expect(preparedAppeal.body.resulting_decision_payload_base64).toBeTruthy();
  const appealResolutionArtifact = await uploadPreparationArtifact(approver.page, approver.token, preparedAppeal.body.id, 'primary');
  const replacementDecisionArtifact = await uploadPreparationArtifact(approver.page, approver.token, preparedAppeal.body.id, 'resulting_decision');
  expect(appealResolutionArtifact.document.id).not.toBe(replacementDecisionArtifact.document.id);
  expect(appealResolutionArtifact.version.id).not.toBe(replacementDecisionArtifact.version.id);
  const finalizedAppeal = await api<Command>(approver.page, approver.token, '/api/admissions/legal-preparations/finalize', { method: 'POST', body: JSON.stringify({ preparation_id: preparedAppeal.body.id, archive: { document_id: appealResolutionArtifact.document.id, version_id: appealResolutionArtifact.version.id }, resulting_decision_archive: { document_id: replacementDecisionArtifact.document.id, version_id: replacementDecisionArtifact.version.id } }) });
  expect(finalizedAppeal.status).toBe(201);
  await approver.context.close();

  const admitted = await api<Detail>(page, token, `/api/admissions/applications/${application.id}`);
  expect((await api<Command>(page, token, `/api/admissions/applications/${application.id}/enrolment`, { method: 'POST', body: JSON.stringify({ student_code: `${marker}-STUDENT`, enrolled_from: new Date().toISOString().slice(0, 10), expected_version: admitted.body.application.expected_version }) })).status).toBe(201);

  const foreign = await browser.newContext({ baseURL: 'http://localhost:4175' });
  const foreignPage = await foreign.newPage();
  const foreignToken = await login(foreignPage, otherTenant, 'http://localhost:4175');
  expect((await api<unknown>(foreignPage, foreignToken, `/api/admissions/applications/${application.id}`)).status).toBe(404);
  await foreign.close();
});
