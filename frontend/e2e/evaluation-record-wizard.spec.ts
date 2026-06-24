import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

test('evaluation record wizard submits an evaluation payload', async ({ page }) => {
  let createdEvaluation: Record<string, unknown> | null = null;

  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page, {
    onCreateEvaluation: (payload) => {
      createdEvaluation = payload;
    },
  });

  await page.goto('/education/personnel/evaluations-wizard');

  await expect(page.getByRole('heading', { name: 'Wizard evaluare anuala' })).toBeVisible();
  await page.getByRole('textbox', { name: 'Cod angajat' }).fill('EMP-001');
  await page.getByRole('textbox', { name: 'Nume complet' }).fill('Prof. Elena Popescu');
  await page.getByRole('textbox', { name: 'Functie' }).fill('Profesor matematica');
  await page.getByRole('button', { name: 'Pasul urmator' }).click();
  await page.getByRole('button', { name: 'Pasul urmator' }).click();
  await page.getByRole('textbox', { name: 'Evaluator' }).fill('Director Demo');
  await page.getByRole('textbox', { name: 'Rezumat' }).fill('Evaluare initiala pentru anul scolar curent.');
  await page.getByRole('button', { name: 'Pasul urmator' }).click();
  await page.getByRole('button', { name: 'Creeaza evaluarea' }).click();

  await expect.poll(() => createdEvaluation).not.toBeNull();
  expect(createdEvaluation).toMatchObject({
    employee_code: 'EMP-001',
    full_name: 'Prof. Elena Popescu',
    role_title: 'Profesor matematica',
    school_year: '2025-2026',
    status: 'draft',
    score: 0,
    qualification: 'foarte_bine',
    evaluator_name: 'Director Demo',
    summary: 'Evaluare initiala pentru anul scolar curent.',
  });
});
