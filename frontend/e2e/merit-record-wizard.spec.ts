import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

test('merit record wizard submits a merit payload', async ({ page }) => {
  let createdMerit: Record<string, unknown> | null = null;

  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page, {
    onCreateMerit: (payload) => {
      createdMerit = payload;
    },
  });

  await page.goto('/education/personnel/merit-wizard');

  await expect(page.getByRole('heading', { name: 'Wizard dosar gradatie' })).toBeVisible();
  await page.getByRole('textbox', { name: 'Nume complet' }).fill('Prof. Sorin Pavel');
  await page.getByRole('textbox', { name: 'Functie' }).fill('Profesor informatica');
  await page.getByRole('button', { name: 'Pasul urmator' }).click();
  await page.getByRole('textbox', { name: 'Comisie' }).fill('Comisia judeteana');
  await page.getByRole('button', { name: 'Pasul urmator' }).click();
  await page.getByText('Include finantare').click();
  await page.getByRole('button', { name: 'Pasul urmator' }).click();
  await page.getByRole('button', { name: 'Creeaza dosarul' }).click();

  await expect.poll(() => createdMerit).not.toBeNull();
  expect(createdMerit).toMatchObject({
    full_name: 'Prof. Sorin Pavel',
    role_title: 'Profesor informatica',
    school_year: '2025-2026',
    category: 'predare',
    status: 'draft',
    score: 0,
    committee_name: 'Comisia judeteana',
    funded: true,
  });
});
