import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedEducationUser, registerEducationApiMocks } from './support/education-mocks';

test('compliance cockpit is discoverable from the education dashboard and exposes compliance-focused quick links', async ({ page }) => {
  await bootstrapAuthenticatedEducationUser(page, {
    userId: 'registrator-1',
    name: 'Registrator Demo',
    email: 'registrator@example.test',
    roles: ['registrator'],
  });
  await registerEducationApiMocks(page, {
    me: {
      userId: 'registrator-1',
      name: 'Registrator Demo',
      email: 'registrator@example.test',
      roles: ['registrator'],
      institutionName: 'Scoala Gimnaziala Demonstrativa',
      permissions: [
        'education.read',
        'education.managerial.read',
        'education.regulations.read',
      ],
    },
  });

  await page.goto('/education/dashboard');

  await expect(page.getByText('Cockpit conformitate', { exact: true })).toBeVisible();
  await page.getByText('Cockpit conformitate', { exact: true })
    .locator('..')
    .locator('..')
    .getByRole('button', { name: 'Deschide cockpitul' })
    .click();

  await expect(page).toHaveURL('/education/dashboard/compliance');
  await expect(page.getByRole('heading', { name: 'Cockpit conformitate' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Actiuni prioritare' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Traseu operational' })).toBeVisible();
  await expect(page.getByText('Registrator', { exact: true }).first()).toBeVisible();
  await expect(page.getByRole('button', { name: 'Deschide lista' }).first()).toBeVisible();
});
