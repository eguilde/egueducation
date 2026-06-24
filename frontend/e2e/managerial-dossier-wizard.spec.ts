import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

test('managerial dossier wizard submits a managerial record payload', async ({ page }) => {
  let createdDossier: Record<string, unknown> | null = null;

  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page, {
    onCreateManagerial: (payload) => {
      createdDossier = payload;
    },
  });

  await page.goto('/education/governance/managerial-wizard');

  await expect(page.getByRole('heading', { name: 'Wizard dosar managerial' })).toBeVisible();
  await page.getByRole('textbox', { name: 'Titlu dosar' }).fill('Plan managerial operational 2026-2027');
  await page.getByRole('button', { name: 'Pasul urmator' }).click();

  await page.getByRole('textbox', { name: 'Responsabil' }).fill('Prof. Director Demo');
  await page.getByRole('textbox', { name: 'Rezumat' }).fill('Dosar initial pentru obiective, termene si documente manageriale.');
  await page.getByRole('button', { name: 'Pasul urmator' }).click();

  await page.getByText('Documentul necesita publicare institutionala').click();
  await page.getByRole('button', { name: 'Pasul urmator' }).click();
  await page.getByRole('button', { name: 'Creeaza dosarul' }).click();

  await expect.poll(() => createdDossier).not.toBeNull();
  expect(createdDossier).toMatchObject({
    school_year: '2025-2026',
    dossier_type: 'annual_plan',
    title: 'Plan managerial operational 2026-2027',
    status: 'draft',
    owner_name: 'Prof. Director Demo',
    publication_required: true,
    summary: 'Dosar initial pentru obiective, termene si documente manageriale.',
  });
});
