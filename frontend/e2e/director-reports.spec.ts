import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

test('standard reports page renders report cards from the director cockpit route family', async ({ page }) => {
  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page);

  await page.goto('/education/dashboard/director/reports');

  await expect(page.getByRole('heading', { name: 'Rapoarte standard' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'CA complet' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'CP complet' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'CA - documente si minute' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'CA - participanti si semnaturi' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'CP - documente si minute' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Portofoliu director', exact: true })).toBeVisible();
  await expect(page.getByText('Situatie portofolii', { exact: true })).toBeVisible();
  await expect(page.getByText('Mobilitate si cazuri deschise', { exact: true })).toBeVisible();
  await expect(page.getByText('status: contestat', { exact: true })).toBeVisible();
});

test('personnel workspace honors preset resource and filters from standard report deep links', async ({ page }) => {
  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page);

  await page.goto('/education/personnel?resource=evaluations&filter_status=contested&presetLabel=Evaluari%20contestate');

  await expect(page).toHaveURL(/\/education\/personnel\?resource=evaluations&filter_status=contested&presetLabel=Evaluari%20contestate/);
  await expect(page.getByRole('tab', { name: /Evaluari anuale/ })).toHaveAttribute('aria-selected', 'true');
  await expect(page.getByText('Evaluari contestate', { exact: true })).toBeVisible();
});

test('workspace can clear an active preset and keep the selected resource', async ({ page }) => {
  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page);

  await page.goto('/education/personnel?resource=evaluations&filter_status=contested&presetLabel=Evaluari%20contestate');

  await page.getByRole('button', { name: 'Reseteaza presetarea' }).click();

  await expect(page).toHaveURL(/\/education\/personnel\?resource=evaluations$/);
  await expect(page.getByRole('tab', { name: /Evaluari anuale/ })).toHaveAttribute('aria-selected', 'true');
  await expect(page.getByText('Evaluari contestate', { exact: true })).toHaveCount(0);
});

test('standard reports page exports csv with preset context', async ({ page }) => {
  let exportPayload: Record<string, unknown> | null = null;

  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page);

  await page.route('**/api/education/evaluations/records**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        items: [
          {
            id: 'evaluation-1',
            evaluation_code: 'EVAL-1',
            full_name: 'Prof. Elena Popescu',
            school_year: '2025-2026',
            status: 'contested',
            qualification: 'foarte_bine',
            score: 95,
            finalized_on: '2026-06-20',
          },
        ],
        total: 1,
        page: 1,
        pageSize: 200,
      }),
    });
  });

  await page.route('**/api/education/exports/csv', async (route) => {
    exportPayload = route.request().postDataJSON() as Record<string, unknown>;
    await route.fulfill({
      status: 200,
      contentType: 'text/csv',
      body: 'col1,col2',
    });
  });

  await page.goto('/education/dashboard/director/reports');
  await page.locator('p-card').filter({ hasText: 'Stadiu evaluari si contestatii' }).getByRole('button', { name: 'CSV' }).click();

  await expect.poll(() => exportPayload).not.toBeNull();
  expect(exportPayload).toMatchObject({
    title: 'Stadiu evaluari si contestatii - Evaluari contestate',
    filename: 'evaluations-evaluari-contestate',
  });
});

test('director portfolio report opens the explicit managerial vertical', async ({ page }) => {
  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page);

  await page.goto('/education/dashboard/director/reports');
  await page.locator('p-card').filter({ has: page.getByRole('heading', { name: 'Portofoliu director', exact: true }) }).getByRole('button', { name: 'Deschide lista' }).first().click();

  await expect(page).toHaveURL(/\/education\/governance\?.*resource=director_portfolio/);
  await expect(page.getByRole('heading', { name: 'Portofoliu director', exact: true })).toBeVisible();
});

test('cp document report opens the cp meetings resource', async ({ page }) => {
  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page);

  await page.goto('/education/dashboard/director/reports');
  await page.locator('p-card').filter({ has: page.getByRole('heading', { name: 'CP - documente si minute', exact: true }) }).getByRole('button', { name: 'Deschide lista' }).first().click();

  await expect(page).toHaveURL(/\/education\/governance\?.*resource=cp_meetings/);
  await expect(page.getByRole('heading', { name: 'CP complet', exact: true })).toBeVisible();
});

test('ca attendance report opens the ca meetings resource', async ({ page }) => {
  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page);

  await page.goto('/education/dashboard/director/reports');
  await page.locator('p-card').filter({ has: page.getByRole('heading', { name: 'CA - participanti si semnaturi', exact: true }) }).getByRole('button', { name: 'Deschide lista' }).first().click();

  await expect(page).toHaveURL(/\/education\/governance\?.*resource=meetings/);
  await expect(page.getByRole('heading', { name: 'Sedinte CA/CP/CEAC', exact: true })).toBeVisible();
});

test('standard reports page exports pdf with preset context', async ({ page }) => {
  let exportPayload: Record<string, unknown> | null = null;

  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page);

  await page.route('**/api/education/evaluations/records**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        items: [
          {
            id: 'evaluation-1',
            evaluation_code: 'EVAL-1',
            full_name: 'Prof. Elena Popescu',
            school_year: '2025-2026',
            status: 'contested',
            qualification: 'foarte_bine',
            score: 95,
            finalized_on: '2026-06-20',
          },
        ],
        total: 1,
        page: 1,
        pageSize: 200,
      }),
    });
  });

  await page.route('**/api/education/exports/pdf', async (route) => {
    exportPayload = route.request().postDataJSON() as Record<string, unknown>;
    await route.fulfill({
      status: 200,
      contentType: 'application/pdf',
      body: 'pdf-binary',
    });
  });

  await page.goto('/education/dashboard/director/reports');
  await page.locator('p-card').filter({ hasText: 'Stadiu evaluari si contestatii' }).getByRole('button', { name: 'PDF' }).click();

  await expect.poll(() => exportPayload).not.toBeNull();
  expect(exportPayload).toMatchObject({
    title: 'Stadiu evaluari si contestatii - Evaluari contestate',
    filename: 'evaluations-evaluari-contestate',
  });
});
