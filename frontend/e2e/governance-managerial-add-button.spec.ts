import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

test('managerial add button opens the managerial wizard from the managerial tab', async ({ page }) => {
  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page);

  await page.goto('/education/governance');

  await page.getByRole('tab', { name: /Dosare manageriale/ }).click();
  await page.locator('[aria-label="Adauga Dosare manageriale"] button').evaluate((button: HTMLButtonElement) => button.click());

  await expect(page).toHaveURL(/\/education\/governance\/managerial-wizard$/);
  await expect(page.getByRole('heading', { name: 'Wizard dosar managerial' })).toBeVisible();
});
