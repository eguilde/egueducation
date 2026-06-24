import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedEducationUser, registerEducationApiMocks } from './support/education-mocks';

test('help center exposes the operational manual and supports filtering', async ({ page }) => {
  await bootstrapAuthenticatedEducationUser(page, {
    userId: 'director-1',
    name: 'Director Demo',
    email: 'director@example.test',
    roles: ['director'],
  });

  await registerEducationApiMocks(page, {
    me: {
      userId: 'director-1',
      name: 'Director Demo',
      email: 'director@example.test',
      roles: ['director'],
      institutionName: 'Scoala Gimnaziala Demonstrativa',
      permissions: ['education.read', 'education.portfolios.read', 'education.governance.read'],
    },
  });

  await page.goto('/help');

  await expect(page.getByText('Ajutor si manual de lucru')).toBeVisible();
  await expect(page.getByText('Portofoliul profesional al cadrului didactic')).toBeVisible();
  await expect(page.getByText('ROF si ROI')).toBeVisible();

  await page.getByPlaceholder('Caută după titlu, public sau rută').fill('rapoarte');

  await expect(page.getByText('Rapoarte manageriale strict necesare')).toBeVisible();
  await expect(page.getByText('CA complet')).toHaveCount(0);
});
