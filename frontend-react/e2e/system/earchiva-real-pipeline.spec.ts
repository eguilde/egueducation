import { expect, test, type Page } from '@playwright/test';
import { execFileSync } from 'node:child_process';

const marker = `E2E-ARCHIVE-SEARCH-UNIQUE-${process.env.GITHUB_RUN_ID ?? 'local'}-${Date.now()}`;
const primary = { identifier: 'archive.pipeline.fixture@example.test', otp: '864210', origin: 'http://127.0.0.1:4176', tenant: 'tenant-egueducation', institution: 'inst-001' };
const isolated = { identifier: 'archive.pipeline.tenant-b@example.test', otp: '975310', origin: 'http://127.0.0.1:4177', tenant: 'tenant-balotesti', institution: 'inst-balotesti' };

function db(sql: string, scope = primary): string {
  const url = process.env.TEST_DATABASE_URL;
  if (!url) throw new Error('TEST_DATABASE_URL is required by the real archive system proof.');
  const scoped = `set app.tenant_id = '${scope.tenant}'; set app.institution_id = '${scope.institution}'; set app.is_super_admin = 'true'; ${sql}`;
  return execFileSync('psql', ['--no-psqlrc', '--tuples-only', '--no-align', '--quiet', url, '-c', scoped], { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] }).trim();
}

function configureMinioClient(): void {
  execFileSync('docker', ['exec', 'egueducation-earchiva-system-minio', 'mc', 'alias', 'set', 'e2e', 'http://127.0.0.1:9000', 'minioadmin', 'minioadmin123'], {
    encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'],
  });
}

function minioObjectExists(key: string): boolean {
  try {
    configureMinioClient();
    execFileSync('docker', ['exec', 'egueducation-earchiva-system-minio', 'mc', 'stat', '--json', `e2e/earchiva-system-e2e/${key}`], {
      encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'],
    });
    return true;
  } catch { return false; }
}

function readArchiveArtifact(key: string): string {
  configureMinioClient();
  return execFileSync('docker', ['exec', 'egueducation-earchiva-system-minio', 'mc', 'cat', `e2e/earchiva-system-e2e/${key}`], {
    encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'],
  });
}

async function login(page: Page, fixture: typeof primary): Promise<string> {
  let token = '';
  page.on('response', async (response) => {
    if (response.request().method() !== 'POST' || !response.url().includes('/api/oidc/token')) return;
    const body = await response.json().catch(() => undefined) as { access_token?: string } | undefined;
    token ||= body?.access_token ?? '';
  });
  await page.goto('/');
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
  await expect(page).toHaveURL(fixture.origin + '/');
  await expect.poll(() => token, { timeout: 15_000 }).not.toBe('');
  return token;
}

async function apiStatus(page: Page, token: string, path: string): Promise<number> {
  return page.evaluate(async ({ path, token }) => (await fetch(path, { headers: { Authorization: `Bearer ${token}` }, credentials: 'include' })).status, { path, token });
}

// Build a standards-shaped, single-page PDF with exact xref offsets. The
// backend's isolated parser (not a browser mock) validates this binary before
// it reaches MinIO.
function buildPDF(): Buffer {
  const objects = [
    '<</Type/Catalog/Pages 2 0 R>>',
    '<</Type/Pages/Count 1/Kids[3 0 R]>>',
    '<</Type/Page/Parent 2 0 R/MediaBox[0 0 200 200]/Contents 4 0 R/Resources<</Font<</F1 5 0 R>>>>>>',
    '<</Length 51>>stream\nBT /F1 12 Tf 20 100 Td (Dosar Maria Popescu) Tj ET\nendstream',
    '<</Type/Font/Subtype/Type1/BaseFont/Helvetica>>',
  ];
  let body = '%PDF-1.4\n';
  const offsets: number[] = [];
  objects.forEach((object, index) => {
    offsets.push(Buffer.byteLength(body));
    body += `${index + 1} 0 obj\n${object}\nendobj\n`;
  });
  const xrefOffset = Buffer.byteLength(body);
  body += `xref\n0 ${objects.length + 1}\n0000000000 65535 f \n`;
  body += offsets.map((offset) => `${String(offset).padStart(10, '0')} 00000 n \n`).join('');
  body += `trailer\n<</Size ${objects.length + 1}/Root 1 0 R>>\nstartxref\n${xrefOffset}\n%%EOF\n`;
  return Buffer.from(body);
}

const pdf = buildPDF();

test('PrimeReact upload reaches PostgreSQL, MinIO, Azure OCR, classification, FTS and remains tenant-isolated', async ({ page, browser }) => {
  const token = await login(page, primary);
  await page.goto('/earchiva');
  await expect(page.getByRole('region', { name: 'eArhivă', exact: true })).toBeVisible();
  await expect(page.getByText('Administrare eArhivă')).toBeVisible();

  await page.getByRole('button', { name: 'Încarcă PDF-uri' }).click();
  const dialog = page.getByRole('dialog', { name: 'Import PDF în eArhivă' });
  await expect(dialog.getByLabel('Tip sursă')).toBeVisible();
  await expect(dialog.getByRole('textbox', { name: 'Tip sursă' })).toHaveCount(0);
  await dialog.locator('input[type="file"]').setInputFiles({ name: 'dosar-elev.pdf', mimeType: 'application/pdf', buffer: pdf });
  await dialog.getByLabel('Titlu pentru dosar-elev.pdf').fill(marker);
  const upload = page.waitForResponse((response) => new URL(response.url()).pathname === '/api/earchiva/documents' && response.request().method() === 'POST');
  await dialog.getByRole('button', { name: 'Transmite lotul' }).click();
  const uploadResponse = await upload;
  const uploadBody = await uploadResponse.text();
  expect(uploadResponse.status(), `archive upload response: ${uploadBody}`).toBe(201);
  const created = JSON.parse(uploadBody) as { id: string };
  expect(created.id).toMatch(/^[0-9a-f-]{36}$/i);

  // This uses the actual worker's committed records, not a fabricated API
  // response.  It proves the queue was consumed and OCR-derived index data
  // were persisted.
  await expect.poll(() => db(`select status from archive_documents where id='${created.id}'`), { timeout: 45_000, intervals: [500, 1000, 2000] }).toBe('ready');
  const documentRow = db(`select original_object_key || '|' || artifact_object_key from archive_documents where id='${created.id}'`);
  const [originalKey, artifactKey] = documentRow.split('|');
  expect(originalKey).toContain(`archive/${primary.institution}/`);
  expect(artifactKey).toContain('/artifact.json');
  expect(minioObjectExists(originalKey)).toBe(true);
  expect(minioObjectExists(artifactKey)).toBe(true);
  expect(readArchiveArtifact(artifactKey)).toContain('Maria Popescu');
  expect(db(`select text_status || '|' || page_count::text || '|' || cardinality(search_embedding)::text from archive_document_versions where document_id='${created.id}'`)).toBe('processed|1|256');
  expect(db(`select count(*)::text from archive_document_chunks where version_id=(select id from archive_document_versions where document_id='${created.id}')`)).toBe('1');
  expect(db(`select count(*)::text from archive_document_classification_reviews where document_id='${created.id}' and state in ('pending_review', 'needs_review')`)).toBe('1');
  expect(db(`select search_tsv is not null and search_tsv @@ plainto_tsquery('simple', 'E2E-ARCHIVE-SEARCH-UNIQUE') from archive_documents where id='${created.id}'`)).toBe('t');
  expect(db(`select content_tsv @@ websearch_to_tsquery('simple', 'Maria Popescu') from archive_document_chunks where version_id=(select id from archive_document_versions where document_id='${created.id}')`)).toBe('t');

  // Exercise the user-facing search, including its generated OpenAPI adapter.
  await page.getByLabel('Caută în arhivă').fill('Maria Popescu');
  await expect(page.getByText(marker)).toBeVisible({ timeout: 15_000 });

  // A distinct OIDC user in another host-derived tenant can neither read nor
  // discover the document.  This is a real bearer-token call through the
  // application handler, after all DB/OCR storage writes have completed.
  const isolatedContext = await browser.newContext({ baseURL: isolated.origin });
  const isolatedPage = await isolatedContext.newPage();
  const isolatedToken = await login(isolatedPage, isolated);
  expect(await apiStatus(isolatedPage, isolatedToken, `/api/earchiva/documents/${created.id}`)).toBe(404);
  await isolatedPage.goto('/earchiva');
  await isolatedPage.getByLabel('Caută în arhivă').fill('Maria Popescu');
  await expect(isolatedPage.getByText(marker)).toHaveCount(0);
  await isolatedContext.close();
});
