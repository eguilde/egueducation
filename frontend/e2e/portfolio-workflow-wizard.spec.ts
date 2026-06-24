import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedEducationUser, registerEducationApiMocks } from './support/education-mocks';

test('portfolio workflow wizard updates and persists the current teacher portfolio', async ({ page }) => {
  await bootstrapAuthenticatedEducationUser(page, {
    userId: 'teacher-1',
    name: 'Profesor Demo',
    email: 'teacher@example.test',
    roles: ['profesor'],
  });
  await registerEducationApiMocks(page, {
    me: {
      userId: 'teacher-1',
      name: 'Profesor Demo',
      email: 'teacher@example.test',
      roles: ['profesor'],
      institutionName: 'Scoala Gimnaziala Demonstrativa',
      permissions: ['education.portfolios.read'],
    },
  });

  await page.goto('/education/portfolio/workflow?owner_name=Profesor%20Demo');

  await expect(page.getByRole('heading', { name: 'Wizard de completare si validare' })).toBeVisible();
  await expect(page.getByText('PORT-2026-001')).toBeVisible();

  await page.getByRole('button', { name: 'Pasul urmator' }).click();
  await page.getByRole('button', { name: 'Pasul urmator' }).click();
  await page.getByRole('button', { name: 'Trimite la verificare' }).click();
  await page.getByRole('button', { name: 'Pasul urmator' }).click();
  await page.getByRole('button', { name: 'Salveaza si intoarce' }).click();

  await expect(page).toHaveURL(/\/education\/portfolio\/me/);
  await expect(page.getByText('In verificare')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Revino la completare' })).toBeVisible();
});
