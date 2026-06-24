import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

test('portfolio record wizard submits a portfolio payload', async ({ page }) => {
  let createdPortfolio: Record<string, unknown> | null = null;

  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page, {
    onCreatePortfolio: (payload) => {
      createdPortfolio = payload;
    },
  });

  await page.goto('/education/portfolio/wizard');

  await expect(page.getByRole('heading', { name: 'Wizard portofoliu cadru didactic' })).toBeVisible();
  await page.getByRole('textbox', { name: 'Titular' }).fill('Prof. Ana Ionescu');
  await page.getByRole('textbox', { name: 'Functie' }).fill('Profesor limba romana');
  await page.getByRole('button', { name: 'Pasul urmator' }).click();

  await page.getByRole('textbox', { name: 'Custode' }).fill('Secretariat');
  await page.getByRole('button', { name: 'Pasul urmator' }).click();

  await page.getByText('Declaratie de autenticitate disponibila').click();
  await page.getByText('Consimtamant / informare GDPR capturat(a)').click();
  await page.getByRole('textbox', { name: 'Note initiale' }).fill('Portofoliu initial creat pentru noul an scolar.');
  await page.getByRole('button', { name: 'Pasul urmator' }).click();
  await page.getByRole('button', { name: 'Creeaza portofoliul' }).click();

  await expect.poll(() => createdPortfolio).not.toBeNull();
  expect(createdPortfolio).toMatchObject({
    owner_name: 'Prof. Ana Ionescu',
    owner_role: 'Profesor limba romana',
    school_year: '2025-2026',
    status: 'draft',
    transfer_status: 'none',
    authenticity_declared: true,
    consent_captured: true,
    custodian: 'Secretariat',
    notes: 'Portofoliu initial creat pentru noul an scolar.',
  });
});
