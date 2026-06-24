import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

test('portfolio detail shows valorification flows derived from the procedural portfolio scope', async ({ page }) => {
  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page);

  await page.goto('/education/portfolio');

  await expect(page.getByRole('heading', { name: 'Portofolii CD' })).toBeVisible();
  await page.locator('tr', { hasText: 'Prof. Elena Ionescu' }).getByRole('button').first().click();

  await expect(page.getByRole('heading', { name: 'Fluxuri de valorificare' })).toBeVisible();
  await expect(page.getByText('Evaluari corelate: 1')).toBeVisible();
  await expect(page.getByText('Mobilitati corelate: 1')).toBeVisible();
  await expect(page.getByText('Gradatii / distinctii: 1')).toBeVisible();
  await expect(page.getByText('Evaluare profesionala anuala: 1')).toBeVisible();
  await expect(page.getByText('Mobilitate: 1')).toBeVisible();
  await expect(page.getByText('Gradatie de merit: 1')).toBeVisible();

  await page.getByRole('button', { name: 'Deschide fluxurile' }).click();
  await expect(page.getByRole('tab', { name: 'Fluxuri de valorificare' })).toHaveAttribute('aria-selected', 'true');
  await expect(page.locator('tr', { hasText: 'VAL-2026-0001' })).toBeVisible();
  await expect(page.getByText('EVA-2026-0001').first()).toBeVisible();
  await expect(page.locator('tr', { hasText: 'VAL-2026-0002' })).toBeVisible();
});
