import { expect, test, type Page } from '@playwright/test';

const canaryIdentifier = requiredSecret('PRODUCTION_E2E_CANARY_IDENTIFIER');
const canaryOTP = requiredSecret('PRODUCTION_E2E_CANARY_OTP');
const productionOrigin = requiredOrigin();

test.use({ trace: 'off', screenshot: 'off', video: 'off' });
test.describe.configure({ mode: 'serial' });

function requiredSecret(name: string): string {
  const value = process.env[name]?.trim();
  if (!value) throw new Error(name + ' is required for the production OIDC canary.');
  return value;
}

function requiredOrigin(): string {
  const value = process.env.PRODUCTION_ORIGIN?.trim();
  if (!value) throw new Error('PRODUCTION_ORIGIN is required for the production OIDC canary.');
  const parsed = new URL(value);
  if (parsed.protocol !== 'https:' || parsed.pathname !== '/' || parsed.search || parsed.hash) {
    throw new Error('PRODUCTION_ORIGIN must be an HTTPS origin without a path, query, or fragment.');
  }
  return parsed.origin;
}

async function startOTPLogin(page: Page): Promise<void> {
  await page.goto('/');
  await page.getByRole('button', { name: 'Autentificare' }).last().click();
  await expect(page).toHaveURL(/\/api\/oidc\/authorize/);

  await page.getByRole('button', { name: /SMS/ }).click();
  await page.getByLabel('Utilizator, email sau numar de telefon').fill(canaryIdentifier);
  await page.getByRole('button', { name: 'Trimite codul prin SMS' }).click();
  await expect(page.locator('.otp-box')).toHaveCount(6);
  await expect(page.locator('html[data-oidc-ui-ready="true"]')).toHaveCount(1);
}

async function submitOTP(page: Page): Promise<void> {
  const otpBoxes = page.locator('.otp-box');
  await otpBoxes.first().click();
  await page.keyboard.type(canaryOTP, { delay: 40 });

  await expect(otpBoxes).toHaveCount(6);
  for (const [index, digit] of [...canaryOTP].entries()) {
    await expect(otpBoxes.nth(index)).toHaveValue(digit);
  }

  const verifyButton = page.getByRole('button', { name: 'Verifica codul' });
  await expect(verifyButton).toBeEnabled();
  await verifyButton.click();
}

test('logs in the dedicated RBAC test user through the normal production OTP flow', async ({ browser }) => {
  const context = await browser.newContext({ baseURL: productionOrigin });
  try {
    const page = await context.newPage();
    await startOTPLogin(page);

    const meResponse = page.waitForResponse((response) => {
      const url = new URL(response.url());
      return url.origin === productionOrigin && url.pathname === '/api/me' && response.request().method() === 'GET';
    });
    await submitOTP(page);

    const consent = page.getByRole('button', { name: 'Accepta si continua' });
    if (await consent.count()) await consent.click();

    await expect(page).toHaveURL(productionOrigin + '/');
    const authenticatedSessionResponse = await meResponse;
    expect(authenticatedSessionResponse.status()).toBe(200);
    const authenticatedSession = await authenticatedSessionResponse.json() as {
      institution_id?: string;
      permissions?: string[];
      modules?: Array<{ code?: string; active?: boolean }>;
      user?: { email?: string; roles?: string[] };
    };
    expect(authenticatedSession.institution_id).toBe('inst-balotesti');
    expect(authenticatedSession.user?.email?.toLowerCase()).toBe(canaryIdentifier.toLowerCase());
    expect(authenticatedSession.user?.roles).toEqual(['e2e_canary']);
    expect([...(authenticatedSession.permissions ?? [])].sort()).toEqual(['admin.read', 'dashboard.read']);
    expect((authenticatedSession.permissions ?? []).some((permission) => permission.endsWith('.manage'))).toBe(false);
    expect((authenticatedSession.modules ?? []).filter((module) => module.active).map((module) => module.code).sort()).toEqual(['admin', 'dashboard']);
    await expect(page.getByRole('button', { name: 'Deconectare' })).toBeVisible();

    const logoutResponse = page.waitForResponse((response) => {
      const url = new URL(response.url());
      return url.origin === productionOrigin && url.pathname === '/api/oidc/session/logout' && response.request().method() === 'POST';
    });
    await page.getByRole('button', { name: 'Deconectare' }).click();
    expect((await logoutResponse).status()).toBe(200);
    await expect(page).toHaveURL(productionOrigin + '/');
    await expect(page.getByRole('button', { name: 'Autentificare' }).last()).toBeVisible();
  } finally {
    await context.close().catch(() => undefined);
  }
});
