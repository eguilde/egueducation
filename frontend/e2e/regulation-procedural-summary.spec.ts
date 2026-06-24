import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

test('regulation detail shows ROF / ROI procedural summary derived from versions and workflow', async ({ page }) => {
  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page);

  await page.goto('/education/governance?resource=regulations');

  await expect(page.getByRole('heading', { name: 'Guvernanta si trasee procedurale' })).toBeVisible();
  await expect(page.getByRole('tab', { name: 'ROF / ROI' })).toBeVisible();
  const regulationRow = page.locator('tr', { hasText: 'Regulament de ordine interioara' });
  await expect(regulationRow).toBeVisible();
  await page.locator('[data-pc-section="mask"]').last().waitFor({ state: 'hidden' });
  await regulationRow.getByRole('button').first().dispatchEvent('click');

  await expect(page.getByRole('heading', { name: 'Circuit procedural ROF / ROI' })).toBeVisible();
  await expect(page.getByText('Total versiuni: 2')).toBeVisible();
  await expect(page.getByText('In consultare: 1')).toBeVisible();
  await expect(page.getByText('Feedback colectat: 14')).toBeVisible();
  await expect(page.getByText('Lipseste aprobarea in CA')).toBeVisible();
  await expect(page.getByText('v0.8').first()).toBeVisible();
  await expect(page.getByText('Consultare publica').first()).toBeVisible();

  await page.getByRole('button', { name: 'Deschide versiunile' }).click();
  await expect(page.getByRole('tab', { name: 'Versiuni' })).toHaveAttribute('aria-selected', 'true');
  await expect(page.locator('tr', { hasText: 'v0.8' })).toBeVisible();
  await expect(page.locator('tr', { hasText: 'v0.7' })).toBeVisible();
});
