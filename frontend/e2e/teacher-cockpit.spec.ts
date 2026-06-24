import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedEducationUser, registerEducationApiMocks } from './support/education-mocks';

test('teacher cockpit is discoverable from the education dashboard and exposes teacher-focused quick links', async ({ page }) => {
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
      permissions: [
        'education.portfolios.read',
        'education.evaluations.read',
        'education.declarations.read',
        'education.mobility.read',
      ],
    },
  });

  await page.goto('/education/dashboard');

  await expect(page.getByText('Cockpit profesor', { exact: true })).toBeVisible();
  await page.getByText('Cockpit profesor', { exact: true })
    .locator('..')
    .locator('..')
    .getByRole('button', { name: 'Deschide cockpitul' })
    .click();

  await expect(page).toHaveURL('/education/dashboard/teacher');
  await expect(page.getByRole('heading', { name: 'Cockpit profesor' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Fluxuri personale' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Traseu de lucru' })).toBeVisible();
  await expect(page.getByText('Profesor', { exact: true }).first()).toBeVisible();
  await expect(page.getByRole('button', { name: 'Deschide lista' }).first()).toBeVisible();
});
