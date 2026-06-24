import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

test('personnel record wizard submits a personnel payload', async ({ page }) => {
  let createdRecord: Record<string, unknown> | null = null;

  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page, {
    onCreatePersonnel: (payload) => {
      createdRecord = payload;
    },
  });

  await page.goto('/education/personnel/wizard');

  await expect(page.getByRole('heading', { name: 'Wizard cadru didactic' })).toBeVisible();
  await page.getByRole('textbox', { name: 'Nume complet' }).fill('Prof. Elena Popescu');
  await page.getByRole('textbox', { name: 'Functie' }).fill('Profesor matematica');
  await page.getByRole('button', { name: 'Pasul urmator' }).click();
  await page.getByText('Are portofoliu activ').click();
  await page.getByRole('button', { name: 'Pasul urmator' }).click();
  await page.getByRole('textbox', { name: 'Email' }).fill('elena.popescu@example.test');
  await page.getByRole('button', { name: 'Pasul urmator' }).click();
  await page.getByRole('button', { name: 'Creeaza fisa' }).click();

  await expect.poll(() => createdRecord).not.toBeNull();
  expect(createdRecord).toMatchObject({
    full_name: 'Prof. Elena Popescu',
    role_title: 'Profesor matematica',
    employment_type: 'titular',
    status: 'active',
    evaluation_status: 'draft',
    mobility_stage: 'none',
    school_year: '2025-2026',
    email: 'elena.popescu@example.test',
    has_portfolio: true,
  });
});
