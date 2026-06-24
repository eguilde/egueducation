import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

const scenarios = [
  {
    resource: 'director_portfolio',
    label: 'Portofoliu director',
    wizardTitle: 'Wizard portofoliu director',
    wizardRoute: /\/education\/governance\/managerial-wizard\?dossierType=director_portfolio$/,
  },
  {
    resource: 'adjunct_director_portfolio',
    label: 'Portofoliu director adjunct',
    wizardTitle: 'Wizard portofoliu director adjunct',
    wizardRoute: /\/education\/governance\/managerial-wizard\?dossierType=adjunct_director_portfolio$/,
  },
] as const;

for (const scenario of scenarios) {
  test(`managerial vertical ${scenario.resource} opens the contextual wizard`, async ({ page }) => {
    await bootstrapAuthenticatedDirector(page);
    await registerEducationApiMocks(page);

    await page.goto(`/education/governance?resource=${scenario.resource}`);

    await expect(page.getByRole('heading', { name: scenario.label })).toBeVisible();
    await page.locator(`[aria-label="Adauga ${scenario.label}"] button`).evaluate((button: HTMLButtonElement) => button.click());

    await expect(page).toHaveURL(scenario.wizardRoute);
    await expect(page.getByRole('heading', { name: scenario.wizardTitle })).toBeVisible();
  });
}
