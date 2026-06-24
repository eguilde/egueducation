import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

test('vote child add button opens the vote wizard from the meeting detail', async ({ page }) => {
  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page);

  await page.goto('/education/governance');

  await page.getByRole('tab', { name: /Sedinte CA\/CP\/CEAC/ }).click();
  const openButton = page
    .getByRole('row', { name: /Sedinta CA pentru aprobarea planului managerial/ })
    .getByRole('button')
    .first();
  await openButton.evaluate((element: HTMLElement) => element.click());

  await expect(page.getByRole('dialog', { name: 'Detalii inregistrare' })).toBeVisible();
  await page.getByRole('tab', { name: /Voturi si rezultate/ }).click();

  await page
    .locator('[aria-label="Adauga Voturi si rezultate"]')
    .locator('button')
    .click();

  await expect(page).toHaveURL(/\/education\/governance\/votes-wizard\?meetingId=meeting-1$/);
  await expect(page.getByRole('heading', { name: 'Wizard vot si rezultat' })).toBeVisible();
});
