import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

test('declaration record wizard submits a declaration payload', async ({ page }) => {
  let createdDeclaration: Record<string, unknown> | null = null;

  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page, {
    onCreateDeclaration: (payload) => {
      createdDeclaration = payload;
    },
  });

  await page.goto('/education/personnel/declarations-wizard');

  await expect(page.getByRole('heading', { name: 'Wizard declaratie' })).toBeVisible();
  await page.getByRole('textbox', { name: 'Cod angajat' }).fill('EMP-002');
  await page.getByRole('textbox', { name: 'Nume complet' }).fill('Prof. Ioana Marin');
  await page.getByRole('button', { name: 'Pasul urmator' }).click();
  await page.getByRole('button', { name: 'Pasul urmator' }).click();
  await page.getByRole('textbox', { name: 'Rezumat' }).fill('Declaratie initiala inregistrata pentru audit anual.');
  await page.getByRole('button', { name: 'Pasul urmator' }).click();
  await page.getByRole('button', { name: 'Creeaza declaratia' }).click();

  await expect.poll(() => createdDeclaration).not.toBeNull();
  expect(createdDeclaration).toMatchObject({
    employee_code: 'EMP-002',
    full_name: 'Prof. Ioana Marin',
    declaration_type: 'authenticity',
    status: 'draft',
    school_year: '2025-2026',
    summary: 'Declaratie initiala inregistrata pentru audit anual.',
  });
});
