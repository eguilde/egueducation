import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

test('committee detail shows evaluation committee completeness and opens members', async ({ page }) => {
  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page);

  await page.goto('/education/governance?resource=committees');

  const committeeRow = page.getByRole('row', { name: /COM-EVAL-2026-001.*Comisia temporara de evaluare a personalului didactic/i }).first();
  await expect(committeeRow).toBeVisible();
  await committeeRow.locator('button').first().evaluate((button: HTMLButtonElement) => button.click());

  await expect(page.getByRole('heading', { name: 'Comisie institutionala' })).toBeVisible();
  await expect(page.getByText('Membri activi: 2')).toBeVisible();
  await expect(page.getByText('Membri cu vot: 2')).toBeVisible();
  await expect(page.getByText('Evaluare: Da')).toBeVisible();
  await expect(page.getByText('DEC-2026-CP-014')).toBeVisible();
  await expect(page.getByText('Nu exista blocaje de constituire')).toBeVisible();

  await page.getByRole('button', { name: 'Deschide membrii' }).click();
  await expect(page.getByRole('tab', { name: 'Membri comisie' })).toHaveAttribute('aria-selected', 'true');
  await expect(page.getByRole('row', { name: /Director Demo.*presedinte/i }).first()).toBeVisible();
  await expect(page.getByRole('row', { name: /Secretariat Demo.*secretar/i }).first()).toBeVisible();
});
