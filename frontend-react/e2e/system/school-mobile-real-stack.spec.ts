import { expect, test, type Page } from '@playwright/test';
import { execFileSync } from 'node:child_process';

const fixture = { identifier: 'oidc.browser.fixture@example.test', otp: '173829', subject: 'oidc-browser-fixture-subject' };
const enabled = Boolean(process.env.TEST_DATABASE_URL && process.env.DATABASE_URL);
test.skip(!enabled, 'Requires the real PostgreSQL/OIDC system topology.');
test.use({ viewport: { width: 390, height: 844 }, isMobile: true });
function sql(statement: string) { const url = process.env.TEST_DATABASE_URL; if (!url) throw new Error('TEST_DATABASE_URL is required.'); return execFileSync('psql', ['--no-psqlrc', '--quiet', url, '-c', `set app.tenant_id='tenant-egueducation'; set app.institution_id='inst-001'; set app.is_super_admin='true'; ${statement}`], { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] }); }
async function login(page: Page) { await page.goto('http://localhost:4173/'); await page.getByRole('button', { name: 'Autentificare' }).last().click(); await page.getByRole('button', { name: /SMS/ }).click(); await page.getByLabel('Utilizator, email sau numar de telefon').fill(fixture.identifier); await page.getByLabel('Canal OTP').selectOption('sms'); await page.getByRole('button', { name: 'Trimite codul' }).click(); const boxes = page.locator('.otp-box'); await expect(boxes).toHaveCount(6); await boxes.first().click(); await boxes.first().pressSequentially(fixture.otp); await page.getByRole('button', { name: 'Verifica codul' }).click(); const consent = page.getByRole('button', { name: 'Accepta si continua' }); if (await consent.count()) await consent.click(); await expect(page).toHaveURL('http://localhost:4173/'); }
test('mobile School governance keeps the drawer, sticky table and action column usable through real OIDC', async ({ page }) => {
  const userID = execFileSync('psql', ['--no-psqlrc', '--tuples-only', '--no-align', process.env.TEST_DATABASE_URL!, '-c', `select id::text from app_users where sub='${fixture.subject}'`], { encoding: 'utf8' }).trim();
  sql(`update app_memberships set position_code='director',active=true where user_id='${userID}' and tenant_code='tenant-egueducation'; delete from app_user_roles where user_id='${userID}' and tenant_code='tenant-egueducation'; insert into app_user_roles(tenant_code,user_id,role_code) values ('tenant-egueducation','${userID}','director') on conflict do nothing`);
  await login(page);
  await page.goto('/scoala/governance');
  const meetingsTitle = page.getByText('Ședințe de guvernanță', { exact: true });
  await expect(meetingsTitle).toBeVisible();
  const meetingsTable = meetingsTitle.locator('xpath=following::table[1]');
  await expect(meetingsTable.getByRole('button', { name: 'Acțiuni înregistrare' }).first()).toBeVisible();
  await expect(meetingsTable.locator('thead.sticky')).toBeVisible();
});
