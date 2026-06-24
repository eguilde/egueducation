import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

test('evaluation add button opens the evaluation wizard from the evaluations tab', async ({ page }) => {
  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page);

  await page.goto('/education/personnel');

  await page.getByRole('tab', { name: /Evaluari anuale/ }).click();
  await page.locator('[aria-label="Adauga Evaluari anuale"] button').click();

  await expect(page).toHaveURL(/\/education\/personnel\/evaluations-wizard$/);
  await expect(page.getByRole('heading', { name: 'Wizard evaluare anuala' })).toBeVisible();
});
