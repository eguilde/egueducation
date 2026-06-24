import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

test('declaration add button opens the declaration wizard from the declarations tab', async ({ page }) => {
  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page);

  await page.goto('/education/personnel');

  await page.getByRole('tab', { name: /Declaratii/ }).click();
  await page.locator('[aria-label="Adauga Declaratii"] button').click();

  await expect(page).toHaveURL(/\/education\/personnel\/declarations-wizard$/);
  await expect(page.getByRole('heading', { name: 'Wizard declaratie' })).toBeVisible();
});
