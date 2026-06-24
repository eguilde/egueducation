import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

test('director cockpit metric card opens a filtered workspace route', async ({ page }) => {
  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page);

  await page.goto('/education/dashboard/director');

  await page.locator('p-card').filter({ hasText: 'Evaluari contestate' }).getByRole('button', { name: 'Deschide' }).click();

  await expect(page).toHaveURL(/\/education\/personnel\?resource=evaluations&filter_status=contested/);
  await expect(page.getByRole('tab', { name: /Evaluari anuale/ })).toHaveAttribute('aria-selected', 'true');
});

test('director cockpit alert opens a filtered workspace route', async ({ page }) => {
  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page);

  await page.goto('/education/dashboard/director');

  await page.locator('article').filter({ hasText: 'Evaluari contestate' }).getByRole('button', { name: 'Vezi lista' }).click();

  await expect(page).toHaveURL(/\/education\/personnel\?resource=evaluations&filter_status=contested/);
  await expect(page.getByRole('tab', { name: /Evaluari anuale/ })).toHaveAttribute('aria-selected', 'true');
});
