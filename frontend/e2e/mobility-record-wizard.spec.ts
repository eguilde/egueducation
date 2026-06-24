import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

test('mobility record wizard submits a mobility payload', async ({ page }) => {
  let createdMobility: Record<string, unknown> | null = null;

  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page, {
    onCreateMobility: (payload) => {
      createdMobility = payload;
    },
  });

  await page.goto('/education/personnel/mobility-wizard');

  await expect(page.getByRole('heading', { name: 'Wizard caz de mobilitate' })).toBeVisible();
  await page.getByRole('textbox', { name: 'Cod angajat' }).fill('EMP-003');
  await page.getByRole('textbox', { name: 'Nume complet' }).fill('Prof. Mihai Dumitrescu');
  await page.getByRole('button', { name: 'Pasul urmator' }).click();
  await page.getByRole('textbox', { name: 'Analizat de' }).fill('Inspector Demo');
  await page.getByRole('button', { name: 'Pasul urmator' }).click();
  await page.getByRole('textbox', { name: 'Unitate sursa' }).fill('Scoala Gimnaziala Demonstrativa');
  await page.getByRole('textbox', { name: 'Unitate destinatie' }).fill('Liceul Teoretic Exemplu');
  await page.getByRole('button', { name: 'Pasul urmator' }).click();
  await page.getByRole('button', { name: 'Creeaza cazul' }).click();

  await expect.poll(() => createdMobility).not.toBeNull();
  expect(createdMobility).toMatchObject({
    employee_code: 'EMP-003',
    full_name: 'Prof. Mihai Dumitrescu',
    school_year: '2025-2026',
    request_type: 'transfer',
    stage: 'draft',
    status: 'open',
    source_school: 'Scoala Gimnaziala Demonstrativa',
    destination_school: 'Liceul Teoretic Exemplu',
    reviewed_by: 'Inspector Demo',
  });
});
