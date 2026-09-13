import { expect, test, type Page } from '@playwright/test';
import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { readFile } from 'node:fs/promises';
import { strFromU8, unzipSync } from 'fflate';
import { HeadObjectCommand, ListObjectVersionsCommand, PutObjectCommand, S3Client } from '@aws-sdk/client-s3';

async function inspectStoredVersion(bucket: string, key: string, versionID: string) {
  const endpoint = process.env.ARCHIVE_STORAGE_ENDPOINT;
  const accessKeyId = process.env.ARCHIVE_STORAGE_ACCESS_KEY;
  const secretAccessKey = process.env.ARCHIVE_STORAGE_SECRET_KEY;
  if (!endpoint || !accessKeyId || !secretAccessKey) throw new Error('Isolated archive storage configuration is required');
  const client = new S3Client({ endpoint, region: process.env.ARCHIVE_STORAGE_REGION ?? 'us-east-1', forcePathStyle: true, credentials: { accessKeyId, secretAccessKey } });
  try {
    return await client.send(new HeadObjectCommand({ Bucket: bucket, Key: key, VersionId: versionID }));
  } finally {
    client.destroy();
  }
}

async function countStoredVersions(bucket: string, key: string) {
  const client = new S3Client({ endpoint: process.env.ARCHIVE_STORAGE_ENDPOINT!, region: process.env.ARCHIVE_STORAGE_REGION ?? 'us-east-1', forcePathStyle: true, credentials: { accessKeyId: process.env.ARCHIVE_STORAGE_ACCESS_KEY!, secretAccessKey: process.env.ARCHIVE_STORAGE_SECRET_KEY! } });
  try {
    const result = await client.send(new ListObjectVersionsCommand({ Bucket: bucket, Prefix: key }));
    expect(result.IsTruncated).not.toBe(true);
    expect((result.DeleteMarkers ?? []).filter(version => version.Key === key)).toHaveLength(0);
    return (result.Versions ?? []).filter(version => version.Key === key).length;
  } finally { client.destroy(); }
}

async function putShortRetainedHeldVersion(key: string, body: Buffer) {
  const bucket = process.env.ARCHIVE_STORAGE_BUCKET;
  if (!bucket) throw new Error('ARCHIVE_STORAGE_BUCKET is required');
  const retainUntil = new Date(Date.now() + 5_000);
  const client = new S3Client({ endpoint: process.env.ARCHIVE_STORAGE_ENDPOINT!, region: process.env.ARCHIVE_STORAGE_REGION ?? 'us-east-1', forcePathStyle: true, credentials: { accessKeyId: process.env.ARCHIVE_STORAGE_ACCESS_KEY!, secretAccessKey: process.env.ARCHIVE_STORAGE_SECRET_KEY! } });
  try {
    const result = await client.send(new PutObjectCommand({ Bucket: bucket, Key: key, Body: body, ContentType: 'application/pdf', ObjectLockMode: 'COMPLIANCE', ObjectLockRetainUntilDate: retainUntil, ObjectLockLegalHoldStatus: 'ON' }));
    if (!result.VersionId || !result.ETag) throw new Error('Short-retention fixture did not receive immutable storage identity');
    return { bucket, key, versionID: result.VersionId, etag: result.ETag.replaceAll('"', ''), retainUntil };
  } finally { client.destroy(); }
}

const origin = `http://localhost:${process.env.E2E_TEACHER_FRONTEND_PORT ?? '4174'}`;
const adminOrigin = `http://localhost:${process.env.E2E_PRIMARY_FRONTEND_PORT ?? '4173'}`;
const identifier = 'oidc.approver.fixture@example.test';
const otp = '428615';
const tenant = 'tenant-egueducation';
const institution = 'inst-001';
const marker = `TEACHER-UPLOAD-${process.env.GITHUB_RUN_ID ?? 'local'}-${Date.now()}`;
function buildPDF(): Buffer {
  const stream = 'BT /F1 12 Tf 20 100 Td (Teacher upload) Tj ET\n';
  const objects = ['<</Type/Catalog/Pages 2 0 R>>', '<</Type/Pages/Count 1/Kids[3 0 R]>>', '<</Type/Page/Parent 2 0 R/MediaBox[0 0 200 200]/Contents 4 0 R/Resources<</Font<</F1 5 0 R>>>>>>', `<</Length ${Buffer.byteLength(stream)}>>stream\n${stream}endstream`, '<</Type/Font/Subtype/Type1/BaseFont/Helvetica>>'];
  let body = '%PDF-1.4\n'; const offsets: number[] = [];
  objects.forEach((object, index) => { offsets.push(Buffer.byteLength(body)); body += `${index + 1} 0 obj\n${object}\nendobj\n`; });
  const xrefOffset = Buffer.byteLength(body);
  body += `xref\n0 6\n0000000000 65535 f \n${offsets.map((offset) => `${String(offset).padStart(10, '0')} 00000 n \n`).join('')}trailer\n<</Size 6/Root 1 0 R>>\nstartxref\n${xrefOffset}\n%%EOF\n`;
  return Buffer.from(body);
}

function db(sql: string): string {
  const url = process.env.TEST_DATABASE_URL;
  if (!url) throw new Error('TEST_DATABASE_URL is required');
  const scoped = `set app.tenant_id='${tenant}'; set app.institution_id='${institution}'; set app.is_super_admin='true'; ${sql}`;
  return execFileSync('psql', ['--no-psqlrc', '--tuples-only', '--no-align', '--quiet', '-v', 'ON_ERROR_STOP=1', url], { input: Buffer.from(scoped, 'utf8'), encoding: 'utf8', env: { ...process.env, PGCLIENTENCODING: 'UTF8' } }).trim();
}

async function login(page: Page, loginIdentifier = identifier, loginOtp = otp, loginOrigin = origin): Promise<string> {
  let token: string | undefined;
  page.on('response', async (response) => {
    if (response.request().method() !== 'POST' || !response.url().includes('/api/oidc/token')) return;
    token ??= (await response.json()).access_token;
  });
  await page.goto(loginOrigin + '/', { waitUntil: 'domcontentloaded' });
  await page.getByRole('button', { name: 'Autentificare' }).last().click();
  await expect(page).toHaveURL(/\/api\/oidc\/authorize/);
  await page.getByRole('button', { name: /SMS/ }).click();
  await page.getByLabel('Utilizator, email sau numar de telefon').fill(loginIdentifier);
  await page.getByLabel('Canal OTP').selectOption('sms');
  await page.getByRole('button', { name: 'Trimite codul' }).click();
  const boxes = page.locator('.otp-box');
  await expect(boxes).toHaveCount(6);
  await boxes.first().click();
  await boxes.first().pressSequentially(loginOtp);
  await page.getByRole('button', { name: 'Verifica codul' }).click();
  const consent = page.getByRole('button', { name: 'Accepta si continua' });
  if (await consent.count()) await consent.click();
  await expect(page.getByRole('button', { name: 'Deconectare' })).toBeVisible();
  await expect.poll(() => token).toBeTruthy();
  return token!;
}

async function loadPortfolioReviewWorkspace(page: Page, destination?: string): Promise<void> {
  const loaded = page.waitForResponse((response) => response.request().method() === 'GET' && new URL(response.url()).pathname === '/api/education/portfolios/records');
  if (destination) await page.goto(destination);
  else await page.reload();
  expect((await loaded).status()).toBe(200);
}

async function openPortfolioReviewDetails(page: Page, ownerName: string, schoolYear: string): Promise<void> {
  const ownerFiltered = page.waitForResponse((response) => {
    const url = new URL(response.url());
    return response.request().method() === 'GET'
      && url.pathname === '/api/education/portfolios/records'
      && url.searchParams.get('filter.owner_name') === ownerName;
  });
  await page.getByLabel('Filtrează Profesor', { exact: true }).fill(ownerName);
  expect((await ownerFiltered).status()).toBe(200);
  const schoolYearFiltered = page.waitForResponse((response) => {
    const url = new URL(response.url());
    return response.request().method() === 'GET'
      && url.pathname === '/api/education/portfolios/records'
      && url.searchParams.get('filter.owner_name') === ownerName
      && url.searchParams.get('filter.school_year') === schoolYear;
  });
  await page.getByLabel('Filtrează An școlar', { exact: true }).fill(schoolYear);
  expect((await schoolYearFiltered).status()).toBe(200);
  const row = page.getByRole('row')
    .filter({ hasText: ownerName })
    .filter({ hasText: schoolYear });
  await expect(row).toHaveCount(1);
  await row.getByRole('button', { name: 'Detalii', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Returnează pentru completări', exact: true })).toBeVisible();
}

test('teacher portfolio upload, submission, director return, correction and validation through the real stack', async ({ page, browser }) => {
  const userID = db("select id::text from app_users where sub='oidc-browser-approver-subject'");
  const startYear = Number(db(`select greatest(2031, coalesce(max(split_part(school_year,'-',1)::int) + 1,2031)) from education_portfolios where owner_user_id='${userID}'`));
  const schoolYear = `${startYear}-${startYear + 1}`;
  const portfolioOwnerName = `${marker} Profesor`;
  const personnelCode = `${marker}-PERSONNEL`;
  db(`update app_memberships set position_code='profesor' where user_id='${userID}' and tenant_code='${tenant}'; delete from app_user_roles where user_id='${userID}' and tenant_code='${tenant}'; delete from app_user_platform_roles where user_id='${userID}';
      insert into education_personnel (employee_code,full_name,role_title,employment_type,status,evaluation_status,mobility_stage,school_year,assigned_unit,phone,email,has_portfolio,institution_id,notes,app_user_id)
      values ('${personnelCode}','${portfolioOwnerName}','Profesor','titular','active','draft','none','${schoolYear}','Învățământ gimnazial','+40100000888','${personnelCode.toLowerCase()}@example.test',true,'${institution}','teacher upload E2E','${userID}')
      on conflict (institution_id,app_user_id) where app_user_id is not null do update set employee_code=excluded.employee_code,full_name=excluded.full_name,school_year=excluded.school_year,has_portfolio=true`);
  const token = await login(page);
  const me = await page.evaluate(async (token) => { const response = await fetch('/api/me', { headers: { Authorization: `Bearer ${token}` } }); return { status: response.status, body: await response.json() }; }, token);
  expect(me.status).toBe(200);
  expect(me.body.permissions).toContain('education.portfolios.manage_own');
  expect(me.body.permissions).not.toContain('earchiva.manage');
  const adminPage = await (await browser.newContext({ baseURL: origin })).newPage();
  const adminID = db("select id::text from app_users where sub='oidc-browser-fixture-subject'");
  db(`update app_memberships set position_code='director' where user_id='${adminID}' and tenant_code='${tenant}'; delete from app_user_platform_roles where user_id='${adminID}'; insert into app_user_roles(tenant_code,user_id,role_code) values ('${tenant}','${adminID}','admin') on conflict do nothing; insert into app_user_permissions(user_id,tenant_code,permission_code) values ('${adminID}','${tenant}','education.portfolios.custody.manage'),('${adminID}','${tenant}','earchiva.manage') on conflict do nothing`);
  const adminToken = await login(adminPage, 'oidc.browser.fixture@example.test', '173829', adminOrigin);
  const call = async (path: string, init: RequestInit = {}) => adminPage.evaluate(async ({ path, init, token }) => { const r = await fetch(path, { ...init, headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' } }); return { status: r.status, body: await r.json().catch(() => ({})) }; }, { path, init, token: adminToken });
  const procedure = await call('/api/education/portfolios/procedures', { method: 'POST', body: JSON.stringify({ procedure_code: marker, title: `${marker} procedure`, source_ref: 'Ordinul nr. 3.858/2026', effective_from: '2026-09-01', calendar_rules: {}, access_rules: {}, accepted_formats: {}, retention_rules: {}, transfer_rules: {} }) });
  expect(procedure.status, JSON.stringify(procedure.body)).toBe(201);
  const rules = ['identificare_profesionala','predare_invatare_evaluare','activitati_complementare','managementul_clasei','evolutie_dezvoltare_profesionala'];
const pr = await call(`/api/education/portfolios/procedures/${procedure.body.id}/section-rules`, { method: 'PUT', body: JSON.stringify({ expected_updated_at: procedure.body.updated_at, rules: rules.map((section_code, index) => ({ id: crypto.randomUUID(), procedure_id: procedure.body.id, section_code, label_ro: section_code, label_en: '', source_catalog_version: 'ome-3858-2026-annexa-1-v1', required: true, sort_order: (index + 1) * 10, active: true })) }) });
  expect(pr.status, JSON.stringify(pr.body)).toBe(200);
  const afterRules = await call(`/api/education/portfolios/procedures/${procedure.body.id}`);
  expect(afterRules.status, JSON.stringify(afterRules.body)).toBe(200);
  const approved = await call(`/api/education/portfolios/procedures/${procedure.body.id}/approve`, { method: 'POST', body: JSON.stringify({ expected_updated_at: afterRules.body.updated_at, evidence: { decision_reference: marker } }) });
  expect(approved.status, JSON.stringify(approved.body)).toBe(200);
  const published = await call(`/api/education/portfolios/procedures/${procedure.body.id}/publish`, { method: 'POST', body: JSON.stringify({ expected_updated_at: approved.body.updated_at, evidence: { publication_reference: marker } }) });
  expect(published.status, JSON.stringify(published.body)).toBe(200);
  await page.goto(origin + '/scoala/portfolio/me');
  await page.getByRole('button', { name: 'Portofoliu nou' }).click();
  const portfolio = page.getByRole('dialog', { name: 'Portofoliu profesional' });
  await portfolio.getByLabel('An școlar *').fill(schoolYear);
  const created = page.waitForResponse(r => new URL(r.url()).pathname === '/api/education/portfolios/me' && r.request().method() === 'POST');
  await portfolio.getByRole('button', { name: 'Salvează ciorna' }).click();
  expect((await created).status()).toBe(201);
  const createdPortfolio = await (await created).json();
  const portfolioID = createdPortfolio.id as string;
  const portfolioReviewOwnerName = createdPortfolio.owner_name as string;
  const portfolioReviewSchoolYear = createdPortfolio.school_year as string;
  expect(portfolioReviewOwnerName).toBeTruthy();
  expect(portfolioReviewSchoolYear).toBe(schoolYear);
  expect(db(`select (activity_ceased_on is null and retention_until is null)::text from education_portfolios where id='${portfolioID}' and owner_user_id='${userID}'`), 'Active custody must not invent a cessation date or retention deadline').toBe('true');
  await expect(page.getByText(createdPortfolio.portfolio_code, { exact: true })).toBeVisible();
  const appliedProcedureLoaded = page.waitForResponse(r => r.request().method() === 'GET' && new URL(r.url()).pathname === `/api/education/portfolios/me/${portfolioID}/procedure`);
  await page.getByRole('button', { name: 'Procedura aplicată', exact: true }).click();
  const appliedProcedureResponse = await appliedProcedureLoaded;
  expect(appliedProcedureResponse.status()).toBe(200);
  const appliedProcedure = await appliedProcedureResponse.json();
  expect(appliedProcedure.procedure.id).toBe(procedure.body.id);
  expect(appliedProcedure.rules.map((rule: { section_code: string }) => rule.section_code)).toEqual(rules);
  const procedureDialog = page.getByRole('dialog', { name: 'Procedura aplicată portofoliului' });
  await expect(procedureDialog.getByText(`${marker} procedure`, { exact: true })).toBeVisible();
  await procedureDialog.getByRole('button', { name: 'Închide procedura aplicată' }).click();
  for (const [index, section] of rules.entries()) {
    const evidenceMarker = `${marker}-${index}`;
  await page.getByRole('button', { name: 'Adaugă document' }).click();
  const dialog = page.getByRole('dialog', { name: 'Adaugă document în portofoliu' });
  const pdf = buildPDF();
  await dialog.locator('input[type=file]').setInputFiles({ name: `${evidenceMarker}.pdf`, mimeType: 'application/pdf', buffer: pdf });
  await dialog.getByLabel('Titlu PDF încărcat').fill(`${evidenceMarker} Planificare`);
  let firstUploadID: string | undefined;
  let firstUploadKey: string | undefined;
  if (index === 0) {
    // The real server commits the upload; only delivery of its response is lost.
    // Retry must retain identity and must not create a second stored version.
    await page.route(`**/api/education/portfolios/me/${portfolioID}/archive-documents`, async route => {
      firstUploadKey = route.request().headers()['idempotency-key'];
      const response = await route.fetch();
      try {
        expect(response.status(), await response.text()).toBe(201);
        firstUploadID = (await response.json()).id;
      } finally { await route.abort('failed'); }
    }, { times: 1 });
    await dialog.getByRole('button', { name: 'Încarcă PDF propriu' }).click();
    await expect(dialog.getByText('Încărcarea nu a fost confirmată. Reîncercați cu același fișier.')).toBeVisible();
  }
  const upload = page.waitForResponse(r => new URL(r.url()).pathname.endsWith('/archive-documents') && r.request().method() === 'POST');
  await dialog.getByRole('button', { name: index === 0 ? 'Reîncearcă încărcarea PDF' : 'Încarcă PDF propriu' }).click();
  const uploadedResponse = await upload;
  expect(uploadedResponse.status(), await uploadedResponse.text()).toBe(index === 0 ? 200 : 201);
  if (index === 0) {
    expect(firstUploadKey).toBeTruthy();
    expect(uploadedResponse.request().headers()['idempotency-key']).toBe(firstUploadKey);
    expect((await uploadedResponse.json()).id).toBe(firstUploadID);
  }
  await expect(dialog.getByText('PDF-ul este disponibil în lista de documente autorizate.')).toBeVisible({ timeout: 120_000 });
  await dialog.getByRole('combobox', { name: 'Componentă din catalog' }).click();
  await page.locator('[role=listbox]:visible').last().getByRole('option', { name: new RegExp(section) }).click();
  for (const [label, value] of [['Titlu *', `${evidenceMarker} dovadă`], ['Tip dovadă *', 'document'], ['Descriere pedagogică *', 'Dovadă încărcată de profesor'], ['An școlar *', schoolYear], ['Disciplina *', 'Matematică'], ['Clasa aplicabilă *', 'clasa a V-a'], ['Competențe *', 'competență']]) await dialog.getByLabel(label, { exact: true }).fill(value);
  await dialog.getByRole('combobox', { name: 'Document eArhivă autorizat' }).click();
  await page.locator('[role=listbox]:visible').last().getByRole('option', { name: `${evidenceMarker} Planificare · v1`, exact: true }).click();
  const attached = page.waitForResponse(r => new URL(r.url()).pathname.endsWith('/documents') && r.request().method() === 'POST');
  await dialog.getByRole('button', { name: 'Adaugă document' }).click();
  expect((await attached).status()).toBe(201);
  const sha = createHash('sha256').update(pdf).digest('hex');
  expect(db(`select count(*)::text from archive_documents where title='${evidenceMarker} Planificare' and institution_id='${institution}' and created_by=(select sub from app_users where id='${userID}')`)).toBe('1');
  expect(db(`select count(*)::text from archive_document_versions v join archive_documents d on d.id=v.document_id where d.title='${evidenceMarker} Planificare' and v.source_sha256='${sha}'`)).toBe('1');
  expect(db(`select count(*)::text from education_portfolio_archive_attachment_grants g join archive_documents d on d.id=g.archive_document_id where d.title='${evidenceMarker} Planificare' and g.grantee_user_id='${userID}'`)).toBe('1');
  expect(db(`select count(*)::text from archive_document_versions v join archive_documents d on d.id=v.document_id where d.title='${evidenceMarker} Planificare' and v.status='active' and length(trim(v.source_object_version_id))>0 and v.source_object_version_id<>'null'`), 'Real upload must persist the immutable storage version, not merely a non-NULL field').toBe('1');
  const stored = JSON.parse(db(`select json_build_object('bucket',v.source_bucket,'key',v.source_object_key,'versionID',v.source_object_version_id,'custodyHold',v.custody_hold_active,'retentionUntil',v.retention_until) from archive_document_versions v join archive_documents d on d.id=v.document_id where d.title='${evidenceMarker} Planificare' and v.status='active'`));
  expect(stored.custodyHold).toBe(true);
  expect(stored.retentionUntil).toBeNull();
  const protectedObject = await inspectStoredVersion(stored.bucket, stored.key, stored.versionID);
  expect(protectedObject.VersionId).toBe(stored.versionID);
  expect(protectedObject.ObjectLockLegalHoldStatus).toBe('ON');
  expect(protectedObject.ObjectLockRetainUntilDate).toBeUndefined();
  expect(protectedObject.ContentLength).toBe(buildPDF().byteLength);
  if (index === 0) expect(await countStoredVersions(stored.bucket, stored.key)).toBe(1);
  expect(db(`select count(*)::text from education_portfolio_documents d join education_portfolios p on p.id=d.portfolio_id where p.owner_user_id='${userID}' and d.document_title='${evidenceMarker} dovadă'`)).toBe('1');
  }

  for (let index = 0; index < 2; index++) {
    await page.getByRole('button', { name: 'Citește și confirmă' }).first().click();
    const declaration = page.getByRole('dialog');
    await declaration.getByLabel('Confirm declarația afișată').click();
    const acknowledged = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname.includes(`/portfolios/me/${portfolioID}/declarations/`));
    await declaration.getByRole('button', { name: 'Confirmă declarația' }).click();
    expect((await acknowledged).status()).toBe(200);
  }
  const opis = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname.endsWith('/opis/regenerate'));
  await page.getByRole('button', { name: 'Regenerare opis' }).click();
  expect((await opis).status()).toBe(200);
  const submit = async () => {
    const submitted = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname === `/api/education/portfolios/me/${portfolioID}/submit`);
    await page.getByRole('button', { name: 'Trimite spre verificare' }).click();
    await page.getByRole('button', { name: 'Confirmă trimiterea' }).click();
    const response = await submitted;
    expect(response.status(), await response.text()).toBe(200);
  };
  await submit();
  await loadPortfolioReviewWorkspace(adminPage, `${adminOrigin}/scoala/portfolio/workflow`);
  await openPortfolioReviewDetails(adminPage, portfolioReviewOwnerName, portfolioReviewSchoolYear);
  await adminPage.getByRole('button', { name: 'Returnează pentru completări', exact: true }).click();
  const correction = adminPage.getByRole('dialog', { name: 'Returnează pentru completări' });
  const correctionText = `${marker} Completați observațiile privind planificarea.`;
  await correction.getByLabel('Observații').fill(correctionText);
  await correction.getByLabel('Scor conformitate').fill('80');
  const returned = adminPage.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname === `/api/education/portfolios/records/${portfolioID}/return`);
  await correction.getByRole('button', { name: 'Returnează', exact: true }).click();
  const returnResponse = await returned;
  expect(returnResponse.status(), await returnResponse.text()).toBe(200);
  await page.reload();
  await expect(page.getByText(correctionText).first()).toBeVisible();
  const documentTitle = `${marker}-0 dovadă`;
  const documentID = db(`select id::text from education_portfolio_documents where portfolio_id='${portfolioID}' and document_title='${documentTitle}'`);
  const originalArchiveHash = db(`select archive_sha256 from education_portfolio_documents where id='${documentID}'`);
  await page.getByRole('button', { name: `Editează ${documentTitle}`, exact: true }).click();
  const documentEdit = page.getByRole('dialog', { name: 'Editează document în portofoliu' });
  await expect(documentEdit.getByLabel('Titlu *', { exact: true })).toHaveValue(documentTitle);
  const revisedDescription = `${marker} Planificare completată după observațiile directorului.`;
  await documentEdit.getByLabel('Descriere pedagogică *', { exact: true }).fill(revisedDescription);
  await documentEdit.getByLabel('Observații', { exact: true }).fill(correctionText);
  const documentSaved = page.waitForResponse(r => r.request().method() === 'PATCH' && new URL(r.url()).pathname === `/api/education/portfolios/me/${portfolioID}/documents/${documentID}`);
  await documentEdit.getByRole('button', { name: 'Salvează modificările', exact: true }).click();
  const documentSaveResponse = await documentSaved;
  expect(documentSaveResponse.status(), await documentSaveResponse.text()).toBe(200);
  const documentUpdatePayload = documentSaveResponse.request().postDataJSON();
  expect(db(`select description from education_portfolio_documents where id='${documentID}'`)).toBe(revisedDescription);
  expect(db(`select archive_sha256 from education_portfolio_documents where id='${documentID}'`)).toBe(originalArchiveHash);
  const historyLoaded = page.waitForResponse(r => r.request().method() === 'GET' && new URL(r.url()).pathname === `/api/education/portfolios/me/${portfolioID}/documents/${documentID}/versions`);
  await page.getByRole('button', { name: `Istoric versiuni ${documentTitle}`, exact: true }).click();
  const historyResponse = await historyLoaded;
  expect(historyResponse.status()).toBe(200);
  const history = await historyResponse.json();
  expect(history.items.some((version: { snapshot: { description: string } }) => version.snapshot.description === revisedDescription)).toBe(true);
  expect(history.items.some((version: { snapshot: { description: string } }) => version.snapshot.description === 'Dovadă încărcată de profesor')).toBe(true);
  await expect(page.getByRole('dialog').getByText('Document și metadate actualizate', { exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Închide istoricul versiunilor' }).click();
  await page.getByRole('button', { name: 'Editează ciorna' }).click();
  const edit = page.getByRole('dialog', { name: 'Portofoliu profesional' });
  await edit.getByLabel('Observații').fill(`${marker} Observațiile au fost completate.`);
  const saved = page.waitForResponse(r => r.request().method() === 'PATCH' && new URL(r.url()).pathname === `/api/education/portfolios/me/${portfolioID}`);
  await edit.getByRole('button', { name: 'Salvează ciorna' }).click();
  expect((await saved).status()).toBe(200);
  await submit();
  await loadPortfolioReviewWorkspace(adminPage);
  await openPortfolioReviewDetails(adminPage, portfolioReviewOwnerName, portfolioReviewSchoolYear);
  await adminPage.getByRole('button', { name: 'Decizie managerială', exact: true }).click();
  const decision = adminPage.getByRole('dialog', { name: 'Decizie managerială' });
  await decision.getByLabel('Observații').fill(`${marker} Portofoliu verificat după completări.`);
  await decision.getByLabel('Scor conformitate').fill('100');
  const validated = adminPage.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname === `/api/education/portfolios/records/${portfolioID}/managerial-decision`);
  await decision.getByRole('button', { name: 'Înregistrează decizia' }).click();
  const validationResponse = await validated;
  expect(validationResponse.status(), await validationResponse.text()).toBe(200);
  expect(db(`select status from education_portfolios where id='${portfolioID}' and owner_user_id='${userID}'`)).toBe('validated');
  expect(db(`select count(*) from education_portfolio_documents where portfolio_id='${portfolioID}'`)).toBe('5');
  expect(db(`select count(*) from education_portfolio_opis where portfolio_id='${portfolioID}'`)).toBe('5');
  expect(db(`select count(*) from education_portfolio_reviews where portfolio_id='${portfolioID}'`)).toBe('2');
  await page.reload();
  await expect(page.getByText('Portofoliul este validat; îl puteți consulta.', { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Adaugă document' })).toHaveCount(0);
  await expect(page.getByRole('button', { name: 'Editează ciorna' })).toHaveCount(0);
  await expect(page.getByRole('button', { name: 'Trimite spre verificare' })).toHaveCount(0);
  await expect(page.getByRole('button', { name: `Editează ${documentTitle}`, exact: true })).toHaveCount(0);
  const deniedEdit = await page.evaluate(async ({ token, portfolioID, documentID, payload }) => {
    const response = await fetch(`/api/education/portfolios/me/${portfolioID}/documents/${documentID}`, {
      method: 'PATCH', headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ ...payload, description: 'Must not overwrite validated evidence' }),
    });
    return { status: response.status, body: await response.json() };
  }, { token, portfolioID, documentID, payload: documentUpdatePayload });
  expect(deniedEdit).toEqual({ status: 409, body: { code: 'education_own_portfolio_not_editable' } });
  expect(db(`select description from education_portfolio_documents where id='${documentID}'`)).toBe(revisedDescription);
  const invalidExport = await page.evaluate(async ({ token, portfolioID }) => {
    const response = await fetch(`/api/education/portfolios/me/${portfolioID}/export`, {
      method: 'POST', headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ documents: [], institution_id: 'client-selected-tenant' }),
    });
    return { status: response.status, body: await response.json() };
  }, { token, portfolioID });
  expect(invalidExport).toEqual({ status: 400, body: { code: 'education_portfolio_export_body_not_allowed' } });
  expect(db(`select count(*) from education_portfolio_export_manifests where portfolio_id='${portfolioID}'`)).toBe('0');
  const exportResponsePromise = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname === `/api/education/portfolios/me/${portfolioID}/export`);
  const downloadPromise = page.waitForEvent('download');
  await page.getByRole('button', { name: 'Descarcă portofoliul', exact: true }).click();
  const exportResponse = await exportResponsePromise;
  expect(exportResponse.status(), exportResponse.status() !== 200 ? await exportResponse.text() : '').toBe(200);
  expect(exportResponse.headers()['content-type']).toBe('application/zip');
  const download = await downloadPromise;
  expect(await download.failure()).toBeNull();
  const downloadPath = await download.path();
  expect(downloadPath).not.toBeNull();
  const entries = unzipSync(await readFile(downloadPath!));
  for (const name of ['portfolio.json', 'opis.json', 'opis.txt', 'manifest.json', 'manifest.canonical.json', 'manifest.sha256']) expect(entries[name], name).toBeDefined();
  expect(Object.keys(entries).every(name => !name.startsWith('/') && !name.includes('..') && !name.includes('\\'))).toBe(true);
  const exportedPortfolio = JSON.parse(strFromU8(entries['portfolio.json']));
  expect(exportedPortfolio.id).toBe(portfolioID);
  expect(exportedPortfolio.status).toBe('validated');
  const manifest = JSON.parse(strFromU8(entries['manifest.json']));
  expect(manifest.manifest_version).toBe('egueducation.portfolio-export-manifest/v2');
  expect(manifest.institution_id).toBe(institution);
  expect(manifest.tenant_code).toBe(tenant);
  expect(manifest.portfolio.id).toBe(portfolioID);
  expect(manifest.documents).toHaveLength(5);
  expect(manifest.generated_files.map((file: { zip_path: string }) => file.zip_path)).toEqual(['portfolio.json', 'opis.json', 'opis.txt']);
  for (const file of manifest.generated_files) {
    const bytes = entries[file.zip_path];
    expect(bytes, file.zip_path).toBeDefined();
    expect(bytes.byteLength).toBe(file.size_bytes);
    expect(createHash('sha256').update(bytes).digest('hex')).toBe(file.sha256);
  }
  const canonicalHash = createHash('sha256').update(entries['manifest.canonical.json']).digest('hex');
  expect(manifest.manifest_sha256).toBe(canonicalHash);
  const canonicalManifest = JSON.parse(strFromU8(entries['manifest.canonical.json']));
  expect(canonicalManifest).toEqual({ ...manifest, manifest_sha256: '' });
  expect(strFromU8(entries['manifest.sha256']).trim()).toBe(`${canonicalHash}  manifest.canonical.json`);
  const exportedOpis = JSON.parse(strFromU8(entries['opis.json']));
  expect(exportedOpis).toHaveLength(5);
  expect(exportedOpis.every((entry: { checked_on: string }) => /^\d{4}-\d{2}-\d{2}$/.test(entry.checked_on))).toBe(true);
  expect(Object.keys(entries).filter(name => name.startsWith('evidence/'))).toHaveLength(5);
  for (const document of manifest.documents) {
    const bytes = entries[document.zip_path];
    expect(bytes, document.zip_path).toBeDefined();
    expect(Buffer.from(bytes)).toEqual(buildPDF());
    expect(bytes.length).toBe(document.source_size_bytes);
    expect(createHash('sha256').update(bytes).digest('hex')).toBe(document.source_sha256);
    expect(document.source_object_version_id).toBeTruthy();
    expect(db(`select archive_sha256 from education_portfolio_documents where id='${document.evidence_record_id}' and portfolio_id='${portfolioID}'`)).toBe(document.source_sha256);
  }
  expect(db(`select count(*) from education_portfolio_export_manifests where portfolio_id='${portfolioID}' and manifest_sha256='${canonicalHash}'`)).toBe('1');

  // All WORM changes below affect only this disposable fixture's exact versions.
  const lifecycle = async (label: string, command: string, date?: string) => {
    await loadPortfolioReviewWorkspace(adminPage);
    await openPortfolioReviewDetails(adminPage, portfolioReviewOwnerName, portfolioReviewSchoolYear);
    await adminPage.getByRole('button', { name: label, exact: true }).click();
    const dialog = adminPage.getByRole('dialog', { name: label, exact: true });
    if (date) await dialog.getByLabel('Data încetării *').fill(date);
    await dialog.getByLabel('Motiv *').fill(`${marker} verificare ciclu de viață izolat`);
    const responsePromise = adminPage.waitForResponse(response => response.request().method() === 'POST' && new URL(response.url()).pathname === `/api/education/portfolios/records/${portfolioID}/${command}`);
    await dialog.getByRole('button', { name: 'Confirmă', exact: true }).click();
    const response = await responsePromise;
    expect(response.status(), await response.text()).toBe(202);
    const accepted = await response.json();
    expect(accepted.operation.id).toBeTruthy();
    await expect(adminPage.getByRole('region', { name: 'Starea protecției portofoliului' }).getByText(/Finalizată. Versiuni verificate: 5 \/ 5/)).toBeVisible({ timeout: 90000 });
    expect(db(`select status from education_portfolio_lifecycle_operations where id='${accepted.operation.id}' and portfolio_id='${portfolioID}'`)).toBe('completed');
  };
  await lifecycle('Aplică blocare juridică', 'legal-hold');
  await lifecycle('Înregistrează încetarea activității', 'activity-cessation', db('select current_date::text'));
  const sources = JSON.parse(db(`select json_agg(json_build_object('bucket',v.source_bucket,'key',v.source_object_key,'versionID',v.source_object_version_id)) from archive_document_versions v join education_portfolio_documents d on d.archive_version_id=v.id where d.portfolio_id='${portfolioID}'`)) as Array<{ bucket: string; key: string; versionID: string }>;
  expect(sources).toHaveLength(5);
  for (const source of sources) {
    const object = await inspectStoredVersion(source.bucket, source.key, source.versionID);
    expect(object.ObjectLockMode).toBe('COMPLIANCE');
    expect(object.ObjectLockRetainUntilDate!.getTime()).toBeGreaterThan(Date.now());
    expect(object.ObjectLockLegalHoldStatus, 'Cessation must not clear a real legal hold').toBe('ON');
  }
  await lifecycle('Ridică blocarea juridică', 'legal-hold');
  for (const source of sources) {
    const object = await inspectStoredVersion(source.bucket, source.key, source.versionID);
    expect(object.ObjectLockMode).toBe('COMPLIANCE');
    expect(object.ObjectLockRetainUntilDate!.getTime()).toBeGreaterThan(Date.now());
    // Releasing the explicit legal hold must not release the independent
    // disposition hold while COMPLIANCE retention is still active. Only the
    // four-eyes expiry workflow below may turn this exact-version hold OFF.
    expect(object.ObjectLockLegalHoldStatus, 'Pre-expiry disposition hold must remain fail-closed').toBe('ON');
  }
  // Rediscover persisted operations after losing all component-local state.
  await loadPortfolioReviewWorkspace(adminPage);
  await openPortfolioReviewDetails(adminPage, portfolioReviewOwnerName, portfolioReviewSchoolYear);
  const lifecycleHistoryResponse = adminPage.waitForResponse(response => response.request().method() === 'GET' && new URL(response.url()).pathname === `/api/education/portfolios/records/${portfolioID}/lifecycle-operations`);
  await adminPage.getByRole('button', { name: 'Istoric operații de protecție', exact: true }).click();
  const lifecycleHistory = adminPage.getByRole('dialog', { name: 'Istoric operații de protecție', exact: true });
  const historyPayload = await (await lifecycleHistoryResponse).json();
  expect(historyPayload.total).toBe(3);
  expect(historyPayload.items.every((item: { status: string; completed_versions: number }) => item.status === 'completed' && item.completed_versions === 5)).toBe(true);
  await expect(lifecycleHistory.getByText('3 operații', { exact: true })).toBeVisible();
  const requestedDate = db("select (current_timestamp at time zone 'UTC')::date::text");
  const filteredResponse = adminPage.waitForResponse(response => response.request().method() === 'GET' && new URL(response.url()).pathname === `/api/education/portfolios/records/${portfolioID}/lifecycle-operations` && new URL(response.url()).searchParams.get('filter.requested_at') === requestedDate);
  await lifecycleHistory.getByLabel('Filtru data solicitării UTC').fill(requestedDate);
  expect((await filteredResponse).status()).toBe(200);
  await lifecycleHistory.getByRole('button', { name: /^Stare operație / }).first().click();
  await expect(lifecycleHistory.getByText(/Finalizată. Versiuni verificate: 5 \/ 5/)).toBeVisible();
  await expect(lifecycleHistory.getByRole('button', { name: 'Reia verificarea', exact: true })).toHaveCount(0);

  // Browser proof for the irreversible expiry path uses a separate disposable
  // version with a genuinely short COMPLIANCE deadline. Existing portfolio
  // objects retain their statutory deadline and are never shortened for tests.
  const retentionPDF = buildPDF();
  const retentionSHA = createHash('sha256').update(retentionPDF).digest('hex');
  const retentionKey = `portfolio-retention-e2e/${crypto.randomUUID()}/original.pdf`;
  const retained = await putShortRetainedHeldVersion(retentionKey, retentionPDF);
  const retentionPortfolioID = crypto.randomUUID();
  const retentionDocumentID = crypto.randomUUID();
  const retentionVersionID = crypto.randomUUID();
  const retentionIntentID = crypto.randomUUID();
  const retentionOperationID = crypto.randomUUID();
  const retentionTransitionID = crypto.randomUUID();
  const retentionSchoolYear = `${startYear + 2}-${startYear + 3}`;
  const retentionPortfolioCode = `RETENTION-E2E-${retentionPortfolioID}`;
  const personnelID = db(`select id::text from education_personnel where institution_id='${institution}' and app_user_id='${userID}'`);
  db(`set app.actor_subject='oidc-browser-approver-subject';
    insert into education_portfolios(id,portfolio_code,owner_name,owner_role,school_year,status,section_count,last_updated_on,transfer_status,institution_id,owner_user_id,owner_personnel_id)
      values('${retentionPortfolioID}','${retentionPortfolioCode}','${marker} Profesor','Profesor','${retentionSchoolYear}','draft',1,current_date,'none','${institution}','${userID}','${personnelID}');
    insert into archive_documents(id,institution_id,title,original_file_name,mime_type,source_kind,source_system,status,current_version_no,created_by)
      values('${retentionDocumentID}','${institution}','${marker} retenție expirată','retention-evidence.pdf','application/pdf','upload','portfolio_retention_e2e','ready',1,'oidc-browser-approver-subject');
    insert into portfolio_custody_upload_intents(id,tenant_code,institution_id,portfolio_id,actor_subject,idempotency_key,expected_sha256,expected_size_bytes,expected_mime_type,expected_request_fingerprint,expected_metadata,bucket_name,object_key,reserved_document_id,reserved_version_id)
      values('${retentionIntentID}','${tenant}','${institution}','${retentionPortfolioID}','oidc-browser-approver-subject','retention-e2e-${retentionIntentID}','${retentionSHA}',${retentionPDF.byteLength},'application/pdf',repeat('a',64),'{}'::jsonb,'${retained.bucket}','${retained.key}','${retentionDocumentID}','${retentionVersionID}');
    update portfolio_custody_upload_intents set status='stored',stored_version_id='${retained.versionID}',stored_etag='${retained.etag}',stored_size_bytes=${retentionPDF.byteLength} where id='${retentionIntentID}';
    insert into archive_document_versions(id,institution_id,document_id,version_no,mime_type,title,bucket_name,object_key,hash_sha256,size_bytes,metadata,ocr_text,status,source_bucket,source_object_key,artifact_bucket,artifact_object_key,source_sha256,source_size_bytes,page_count,text_status,extracted_text,extracted_metadata,created_by,source_object_version_id,source_object_etag,legal_hold_active,custody_hold_active,retention_disposition_hold_active,retention_until,portfolio_custody_intent_id)
      values('${retentionVersionID}','${institution}','${retentionDocumentID}',1,'application/pdf','${marker} retenție expirată','${retained.bucket}','${retained.key}','${retentionSHA}',${retentionPDF.byteLength},'{}'::jsonb,'','active','${retained.bucket}','${retained.key}','${retained.bucket}','${retained.key}','${retentionSHA}',${retentionPDF.byteLength},1,'processed','','{}'::jsonb,'oidc-browser-approver-subject','${retained.versionID}','${retained.etag}',false,true,true,'${retained.retainUntil.toISOString()}', '${retentionIntentID}');
    update portfolio_custody_upload_intents set status='committed',final_disposition='teacher_access',recovery_committed_at=now() where id='${retentionIntentID}';
    insert into education_portfolio_documents(portfolio_id,section_code,component_code,document_title,description,school_year,subject_discipline,applicable_class,competencies,source_scope,evidence_type,issued_on,added_on,chronological_index,sensitive_data,authenticity_status,file_reference,institution_id,archive_document_id,archive_version_id,archive_version_no,archive_source_bucket,archive_source_object_key,archive_sha256,last_change_reason)
      values('${retentionPortfolioID}','identificare_profesionala','structura_cadru','${marker} retenție expirată','Dovadă E2E pentru analizarea expirării','${retentionSchoolYear}','Matematică','clasa a V-a',array['retenție'],'portofoliu','document',current_date,current_date,1,false,'declarat','archive://${retentionDocumentID}','${institution}','${retentionDocumentID}','${retentionVersionID}',1,'${retained.bucket}','${retained.key}','${retentionSHA}','Fixture E2E retenție');
    update education_portfolios set status='validated',activity_ceased_on='2019-01-01',activity_cessation_reason='Fixture E2E cu retenție deja expirată' where id='${retentionPortfolioID}';
    insert into education_portfolio_lifecycle_operations(id,tenant_code,institution_id,portfolio_id,operation_type,status,activity_ceased_on,retention_through,reason,requested_by_subject,completed_at)
      select '${retentionOperationID}','${tenant}','${institution}',id,'cessation_retention','completed',activity_ceased_on,retention_until,'Fixture E2E retenție','oidc-browser-approver-subject',now() from education_portfolios where id='${retentionPortfolioID}';
    insert into education_portfolio_storage_transitions(id,operation_id,tenant_code,institution_id,portfolio_id,archive_document_id,archive_version_id,source_bucket,source_object_key,source_object_version_id,source_object_etag,source_sha256,source_size_bytes,required_retention_until,status,attempts,last_error,observed_retention_until,observed_hold_active,observed_custody_required,observed_legal_hold_required,observed_retention_disposition_hold_required,storage_verified_at)
      values('${retentionTransitionID}','${retentionOperationID}','${tenant}','${institution}','${retentionPortfolioID}','${retentionDocumentID}','${retentionVersionID}','${retained.bucket}','${retained.key}','${retained.versionID}','${retained.etag}','${retentionSHA}',${retentionPDF.byteLength},'${retained.retainUntil.toISOString()}','blocked',1,'portfolio_retention_expired_review_required','${retained.retainUntil.toISOString()}',true,true,false,true,now());
    set app.actor_subject='portfolio-storage-lifecycle-worker';
    insert into education_portfolio_retention_expiry_events(transition_id,tenant_code,institution_id,required_retention_until,promoted_by_subject)
      values('${retentionTransitionID}','${tenant}','${institution}','${retained.retainUntil.toISOString()}','portfolio-storage-lifecycle-worker');`);
  await expect.poll(() => Date.now(), { timeout: 15_000 }).toBeGreaterThan(retained.retainUntil.getTime() + 1_000);

  await page.goto(origin + '/scoala/portfolio/me');
  await page.getByRole('button', { name: `${retentionSchoolYear} Validat`, exact: true }).click();
  await page.getByRole('button', { name: 'Istoric operații de protecție', exact: true }).click();
  const teacherRetentionHistory = page.getByRole('dialog', { name: 'Istoric operații de protecție', exact: true });
  await teacherRetentionHistory.getByRole('button', { name: /^Stare operație / }).click();
  await teacherRetentionHistory.getByRole('button', { name: 'Solicită analizarea expirării', exact: true }).click();
  const requestDialog = page.getByRole('dialog', { name: 'Solicitare de analizare a expirării', exact: true });
  await requestDialog.getByLabel('Justificare și dovezi *').fill(`${marker} confirmă expirarea termenului și lipsa blocajelor juridice`);
  await requestDialog.getByLabel('Referință document').fill(`${marker}-retention-review`);
  const submittedDisposition = page.waitForResponse(response => response.request().method() === 'POST' && new URL(response.url()).pathname.endsWith(`/${retentionTransitionID}/retention-dispositions`));
  await requestDialog.getByRole('button', { name: 'Trimite pentru aprobare', exact: true }).click();
  expect((await submittedDisposition).status()).toBe(202);
  const requestID = db(`select id::text from portfolio_retention_disposition_requests where transition_id='${retentionTransitionID}' and status='submitted'`);

  await loadPortfolioReviewWorkspace(adminPage, `${adminOrigin}/scoala/portfolio/workflow`);
  await openPortfolioReviewDetails(adminPage, portfolioOwnerName, retentionSchoolYear);
  await adminPage.getByRole('button', { name: 'Istoric operații de protecție', exact: true }).click();
  const adminRetentionHistory = adminPage.getByRole('dialog', { name: 'Istoric operații de protecție', exact: true });
  await adminRetentionHistory.getByRole('button', { name: /^Stare operație / }).click();
  await adminRetentionHistory.getByRole('button', { name: 'Înregistrează decizia', exact: true }).click();
  const decisionDialog = adminPage.getByRole('dialog', { name: 'Decizie independentă de retenție', exact: true });
  const submittedEvidence = decisionDialog.getByRole('group', { name: 'Dovezile solicitării', exact: true });
  await expect(submittedEvidence).toContainText('oidc-browser-approver-subject');
  await expect(submittedEvidence).toContainText(`${marker} confirmă expirarea termenului și lipsa blocajelor juridice`);
  await expect(submittedEvidence).toContainText(`${marker}-retention-review`);
  await decisionDialog.getByLabel('Motivul deciziei *').fill(`${marker} aprobare independentă după verificarea expirării`);
  const decidedDisposition = adminPage.waitForResponse(response => response.request().method() === 'POST' && new URL(response.url()).pathname.endsWith(`/retention-dispositions/${requestID}/decision`));
  await decisionDialog.getByRole('button', { name: 'Înregistrează decizia', exact: true }).click();
  expect((await decidedDisposition).status()).toBe(200);
  await expect.poll(() => db(`select status from portfolio_retention_disposition_requests where id='${requestID}'`), { timeout: 90_000 }).toBe('closed');
  expect(db(`select (not custody_hold_active and not legal_hold_active and not retention_disposition_hold_active)::text from archive_document_versions where id='${retentionVersionID}'`)).toBe('true');
  expect(db(`select count(*)::text from portfolio_retention_disposition_receipts where request_id='${requestID}' and outcome='released' and observed_hold_active=false`)).toBe('1');
  const releasedObject = await inspectStoredVersion(retained.bucket, retained.key, retained.versionID);
  expect(releasedObject.ObjectLockMode).toBe('COMPLIANCE');
  expect(releasedObject.ObjectLockLegalHoldStatus).toBe('OFF');
  await adminPage.context().close();
});
