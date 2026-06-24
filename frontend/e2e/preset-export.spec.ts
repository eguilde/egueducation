import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

test('csv export includes preset context in the export payload', async ({ page }) => {
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
        pageSize: 20,
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

  await page.goto('/education/personnel?resource=evaluations&filter_status=contested&presetLabel=Evaluari%20contestate');

  await page.getByRole('button', { name: 'Excel (CSV)' }).click();

  await expect.poll(() => exportPayload).not.toBeNull();
  expect(exportPayload).toMatchObject({
    title: 'Evaluari anuale - Evaluari contestate',
    filename: 'evaluations-evaluari-contestate',
  });
});
