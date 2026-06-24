import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

test('portfolio detail shows transfer summary and can advance transfer flow', async ({ page }) => {
  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page);

  await page.goto('/education/portfolio');

  await expect(page.getByRole('heading', { name: 'Portofolii CD' })).toBeVisible();
  await page.locator('tr', { hasText: 'Prof. Elena Ionescu' }).getByRole('button').first().click();

  await expect(page.getByRole('heading', { name: 'Completitudine portofoliu' })).toBeVisible();
  await expect(page.getByText('Necesita completari')).toBeVisible();
  await expect(page.getByText('Opis: 4')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Regenerare opis' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Transfer digital si mobilitate' })).toBeVisible();
  await expect(page.getByText('Circuit clar')).toBeVisible();
  await expect(page.locator('div').filter({ hasText: /^Scoala Gimnaziala Demonstrativa -> Scoala Gimnaziala Partenera$/ }).first()).toBeVisible();
  await expect(page.getByText('Pregatit | predat 2026-06-24 | receptionat -')).toBeVisible();

  await page.getByRole('button', { name: 'Regenerare opis' }).click();
  await expect(page.getByText('Opisul a fost reconstruit din documentele existente ale portofoliului.')).toBeVisible();

  await page.getByRole('button', { name: 'Marcheaza trimis' }).click();
  await expect(page.getByText('Trimis | predat 2026-06-24 | receptionat -')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Confirma receptionarea' })).toBeVisible();

  await page.getByRole('button', { name: 'Confirma receptionarea' }).click();
  await expect(page.getByText('Receptionat | predat 2026-06-24 | receptionat 2026-06-25')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Inchide transferul' })).toBeVisible();
});
