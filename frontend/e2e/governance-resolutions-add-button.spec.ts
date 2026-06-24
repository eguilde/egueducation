import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

test('resolution child add button opens the resolution wizard from the meeting detail', async ({ page }) => {
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
  await page.getByRole('tab', { name: /Hotarari si avize/ }).click();
  await page
    .locator('[aria-label="Adauga Hotarari si avize"]')
    .locator('button')
    .click();

  await expect(page).toHaveURL(/\/education\/governance\/resolutions-wizard\?meetingId=meeting-1$/);
  await expect(page.getByRole('heading', { name: 'Wizard hotarare si aviz' })).toBeVisible();
});
