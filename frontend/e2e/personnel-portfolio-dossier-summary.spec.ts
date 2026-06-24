import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

test('personnel detail shows the portfolio vs personal dossier summary', async ({ page }) => {
  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page);

  await page.goto('/education/personnel');

  await expect(page.getByRole('heading', { name: 'Cadre didactice' })).toBeVisible();
  await page.locator('tr', { hasText: 'Prof. Elena Ionescu' }).getByRole('button').first().click();

  await expect(page.getByRole('heading', { name: 'Relatia portofoliu - dosar personal' })).toBeVisible();
  await expect(page.getByText('Delimitare clara')).toBeVisible();
  await expect(page.getByText('Referibile in portofoliu:')).toBeVisible();
  await expect(page.getByText('Documentele administrative rezultate din evaluare se arhiveaza in dosarul personal.')).toBeVisible();

  await page.getByRole('button', { name: 'Deschide dosarul personal' }).click();
  await expect(page.getByRole('heading', { name: 'Dosar personal' })).toBeVisible();
});
