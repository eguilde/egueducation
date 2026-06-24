import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

test('portfolio add button opens the portfolio wizard from the portfolio tab', async ({ page }) => {
  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page);

  await page.goto('/education/portfolio');

  await page.locator('[aria-label="Adauga Portofolii CD"] button').click();

  await expect(page).toHaveURL(/\/education\/portfolio\/wizard$/);
  await expect(page.getByRole('heading', { name: 'Wizard portofoliu cadru didactic' })).toBeVisible();
});
