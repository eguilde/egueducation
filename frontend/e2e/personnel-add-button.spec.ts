import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

test('personnel add button opens the personnel wizard from the personnel tab', async ({ page }) => {
  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page);

  await page.goto('/education/personnel');

  await page.locator('[aria-label="Adauga Cadre didactice"] button').click();

  await expect(page).toHaveURL(/\/education\/personnel\/wizard$/);
  await expect(page.getByRole('heading', { name: 'Wizard cadru didactic' })).toBeVisible();
});
