import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

test('cp complet resource opens a cp-prefilled wizard and lists cp meetings', async ({ page }) => {
  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page);

  await page.goto('/education/governance?resource=cp_meetings');

  await expect(page.getByRole('heading', { name: 'CP complet' })).toBeVisible();
  await expect(page.getByRole('row', { name: /Sedinta CP pentru analiza rezultatelor/i }).first()).toBeVisible();

  await page.locator('[aria-label="Adauga CP complet"] button').evaluate((button: HTMLButtonElement) => button.click());

  await expect(page).toHaveURL(/organism=cp/);
  await expect(page.getByRole('heading', { name: 'Wizard sedinta Consiliu Profesoral' })).toBeVisible();
});
