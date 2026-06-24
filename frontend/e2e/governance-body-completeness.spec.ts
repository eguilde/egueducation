import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

test('governance body detail shows CA completeness and opens filtered members', async ({ page }) => {
  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page);

  await page.goto('/education/governance?resource=governance_bodies');

  await expect(page.getByRole('heading', { name: 'Guvernanta si trasee procedurale' })).toBeVisible();
  const caRow = page.getByRole('row', { name: /2025-2026.*ca.*5.*5.*1.*2026-06-30.*complet/i }).first();
  await expect(caRow).toBeVisible();
  await caRow.locator('button').first().evaluate((button: HTMLButtonElement) => button.click());

  await expect(page.getByRole('heading', { name: 'Constituire si operare organism' })).toBeVisible();
  await expect(page.getByText('Membri activi: 5')).toBeVisible();
  await expect(page.getByText('Membri cu vot: 5')).toBeVisible();
  await expect(page.getByText('Presedinte acoperit: Da')).toBeVisible();
  await expect(page.getByText('Ultima sedinta').locator('..').getByText('Sedinta CA pentru aprobarea planului managerial')).toBeVisible();
  await expect(page.getByText('Nu exista blocaje de constituire')).toBeVisible();

  await page.getByRole('button', { name: 'Deschide membrii' }).click();
  await expect(page.getByRole('tab', { name: 'Membri organism' })).toHaveAttribute('aria-selected', 'true');
  await expect(page.getByRole('row', { name: /Director Demo.*Presedinte/ }).first()).toBeVisible();
  await expect(page.getByRole('row', { name: /Secretariat Demo.*Secretar/ }).first()).toBeVisible();
});
