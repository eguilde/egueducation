import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedEducationUser, registerEducationApiMocks } from './support/education-mocks';

test('portfolio self-service page opens a teacher-specific portfolio entry point', async ({ page }) => {
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

  await page.goto('/education/portfolio/me');

  await expect(page.getByRole('heading', { name: 'Portofoliul meu' })).toBeVisible();
  await expect(page.getByRole('main').getByText('Profesor Demo')).toBeVisible();
  await expect(page.getByText('Portofoliu disponibil')).toBeVisible();
  await expect(page.getByText('Ultimul portofoliu gasit')).toBeVisible();
  await expect(page.getByText('Datare automata')).toBeVisible();
  await expect(page.getByText('2026-06-24')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Deschide portofoliul meu' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Trimite la verificare' })).toBeVisible();

  await page.getByRole('button', { name: 'Deschide portofoliul meu' }).click();
  await expect(page).toHaveURL(/\/education\/portfolio\?.*resource=portfolios/);
  await expect(page).toHaveURL(/filter_owner_name=Profesor%20Demo|filter_owner_name=Profesor\+Demo/);
  await expect(page.getByRole('heading', { name: 'Portofolii CD' })).toBeVisible();

  await page.goBack();
  await page.getByRole('button', { name: 'Trimite la verificare' }).click();
  await expect(page.getByText('In verificare')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Revino la completare' })).toBeVisible();
});
