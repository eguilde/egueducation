import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

test('managerial detail shows director portfolio summary and opens managerial documents', async ({ page }) => {
  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page);

  await page.goto('/education/governance?resource=managerial');

  const dossierRow = page.getByRole('row', { name: /MGR-DIR-2026-001.*Portofoliul directorului unitatii de invatamant/i }).first();
  await expect(dossierRow).toBeVisible();
  await dossierRow.locator('button').first().evaluate((button: HTMLButtonElement) => button.click());

  await expect(page.getByRole('heading', { name: 'Portofoliu managerial' })).toBeVisible();
  await expect(page.getByText('Documente totale: 2')).toBeVisible();
  await expect(page.getByText('Obligatorii: 2')).toBeVisible();
  await expect(page.getByText('Persoane potrivite: 1')).toBeVisible();
  await expect(page.getByText('Pentru avizare: Da')).toBeVisible();
  await expect(page.getByText('Nu exista blocaje manageriale')).not.toBeVisible();
  await expect(page.getByText('Documentele de baza sunt prezente')).toBeVisible();

  await page.getByRole('button', { name: 'Deschide documentele' }).click();
  await expect(page.getByRole('tab', { name: 'Documente dosar' })).toHaveAttribute('aria-selected', 'true');
  await expect(page.getByRole('row', { name: /MDOC-DIR-2026-0001.*Opis documente portofoliu director/i }).first()).toBeVisible();
});
