import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

test('governance add button opens the meeting wizard from the meetings tab', async ({ page }) => {
  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page);

  await page.goto('/education/governance');

  await expect(page.getByRole('heading', { name: 'Guvernanta si trasee procedurale' })).toBeVisible();
  await page.getByRole('tab', { name: /Sedinte CA\/CP\/CEAC/ }).click();
  await page
    .locator('[aria-label="Adauga Sedinte CA/CP/CEAC"]')
    .locator('button')
    .click();

  await expect(page).toHaveURL(/\/education\/governance\/ca-wizard$/);
  await expect(page.getByRole('heading', { name: 'Wizard sedinta Consiliu de Administratie' })).toBeVisible();
});
