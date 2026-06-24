import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

test('mobility add button opens the mobility wizard from the mobility tab', async ({ page }) => {
  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page);

  await page.goto('/education/personnel');

  await page.getByRole('tab', { name: /Mobilitate/ }).click();
  await page.locator('[aria-label="Adauga Mobilitate"] button').click();

  await expect(page).toHaveURL(/\/education\/personnel\/mobility-wizard$/);
  await expect(page.getByRole('heading', { name: 'Wizard caz de mobilitate' })).toBeVisible();
});
