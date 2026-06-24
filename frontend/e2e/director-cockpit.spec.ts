import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

test('director cockpit renders operational metrics and alerts', async ({ page }) => {
  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page);

  await page.goto('/education/dashboard/director');

  await expect(page.getByRole('heading', { name: 'Cockpit director' })).toBeVisible();
  await expect(page.getByText('Scoala Gimnaziala Demonstrativa')).toBeVisible();
  await expect(page.getByText('Sedinte guvernanta', { exact: true })).toBeVisible();
  await expect(page.getByText('12')).toBeVisible();
  await expect(page.getByText('Portofolii validate', { exact: true })).toBeVisible();
  await expect(page.getByText('27')).toBeVisible();

  await expect(page.getByRole('heading', { name: 'Alerte operationale' })).toBeVisible();
  await expect(page.locator('article').filter({ hasText: 'Evaluari contestate' }).first()).toBeVisible();
  await expect(page.locator('article').filter({ hasText: 'Publicari restante' }).first()).toBeVisible();

  await expect(page.getByRole('heading', { name: 'Rapoarte standard' })).toBeVisible();
  await expect(page.getByText('Registru sedinte si hotarari')).toBeVisible();
  await expect(page.getByText('Situatie portofolii')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Rapoarte standard' })).toBeVisible();
});
