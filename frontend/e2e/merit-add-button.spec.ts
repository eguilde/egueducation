import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

test('merit add button opens the merit wizard from the merit tab', async ({ page }) => {
  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page);

  await page.goto('/education/personnel');

  await page.getByRole('tab', { name: /Gradatii/ }).click();
  await page.locator('[aria-label="Adauga Gradatii"] button').click();

  await expect(page).toHaveURL(/\/education\/personnel\/merit-wizard$/);
  await expect(page.getByRole('heading', { name: 'Wizard dosar gradatie' })).toBeVisible();
});
