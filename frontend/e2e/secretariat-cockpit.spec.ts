import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedSecretariat, registerEducationApiMocks } from './support/education-mocks';

test('secretariat cockpit is discoverable from the education dashboard and opens role-aware quick links', async ({ page }) => {
  await bootstrapAuthenticatedSecretariat(page);
  await registerEducationApiMocks(page, {
    me: {
      userId: 'secretar-1',
      name: 'Secretariat Demo',
      email: 'secretariat@example.test',
      roles: ['secretar'],
      institutionName: 'Scoala Gimnaziala Demonstrativa',
      permissions: [
        'education.read',
        'education.governance.read',
        'education.governance.manage',
        'education.managerial.read',
        'education.managerial.manage',
        'education.personnel.manage',
        'education.evaluations.manage',
        'education.declarations.read',
        'education.declarations.manage',
        'education.mobility.read',
        'education.mobility.manage',
        'education.gradatii.read',
        'education.gradatii.manage',
        'education.portfolios.manage',
        'education.personnel.read',
        'education.evaluations.read',
        'education.portfolios.read',
      ],
    },
  });

  await page.goto('/education/dashboard');

  await expect(page.getByText('Cockpit secretariat', { exact: true })).toBeVisible();
  await page.getByText('Cockpit secretariat', { exact: true })
    .locator('..')
    .locator('..')
    .getByRole('button', { name: 'Deschide cockpitul' })
    .click();

  await expect(page).toHaveURL('/education/dashboard/secretariat');
  await expect(page.getByRole('heading', { name: 'Cockpit secretariat' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Acces rapid' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Traseu de lucru' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Roluri active' })).toBeVisible();
  await expect(page.getByText('Secretariat', { exact: true }).first()).toBeVisible();
});
