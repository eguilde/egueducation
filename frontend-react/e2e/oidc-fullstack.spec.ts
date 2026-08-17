import { expect, test } from '@playwright/test';

const fixtureIdentifier = 'oidc.browser.fixture@example.test';
const fixtureOTP = '173829';

test('completes real PKCE, OTP, callback, refresh-cookie and /api/me flow', async ({ page }) => {
  await page.goto('/');

  await page.getByRole('button', { name: 'Tema aplicației' }).click();
  await page.getByRole('button', { name: 'Întunecat' }).click();
  await page.keyboard.press('Escape');
  await page.getByRole('button', { name: 'Autentificare' }).last().click();

  await expect(page).toHaveURL(/\/api\/oidc\/authorize/);
  await expect(page.locator('html')).toHaveCSS('color-scheme', 'dark');
  await expect(page.locator('body')).toHaveCSS('background-color', 'rgb(15, 23, 42)');
  await expect(page.getByText('Acces securizat la servicii educationale digitale.')).toHaveCount(0);

  await page.getByRole('button', { name: /SMS/ }).click();
  await page.getByLabel('Utilizator, email sau numar de telefon').fill(fixtureIdentifier);
  await page.getByRole('button', { name: 'Trimite codul prin SMS' }).click();

  const otpBoxes = page.locator('.otp-box');
  await expect(otpBoxes).toHaveCount(6);
  for (const [index, digit] of [...fixtureOTP].entries()) {
    await otpBoxes.nth(index).fill(digit);
  }
  await page.getByRole('button', { name: 'Verifica codul' }).click();

  const consent = page.getByRole('button', { name: 'Accepta si continua' });
  if (await consent.count()) await consent.click();

  await expect(page).toHaveURL('http://127.0.0.1:4173/');
  await expect(page.getByText('Utilizator Test')).toBeVisible();

  const me = await page.request.get('/api/me');
  expect(me.status()).toBe(401);

  await page.reload();
  await expect(page.getByText('Utilizator Test')).toBeVisible();
  await page.goto('/administrare');
  await expect(page).toHaveURL(/\/administrare$/);
});
