import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

test('ca meeting wizard submits a governance meeting payload', async ({ page }) => {
  let createdPayload: Record<string, unknown> | null = null;

  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page, {
    onCreateMeeting: (payload) => {
      createdPayload = payload;
    },
  });

  await page.goto('/education/governance/ca-wizard');

  await expect(page.getByRole('heading', { name: 'Wizard sedinta Consiliu de Administratie' })).toBeVisible();

  await page.getByRole('textbox', { name: 'Titlu sedinta' }).fill('Sedinta CA pentru aprobarea planului managerial');
  await page.getByRole('textbox', { name: 'Locatie' }).fill('Sala profesorală');
  await page.getByRole('button', { name: 'Pasul urmator' }).click();

  await page.getByRole('textbox', { name: 'Presedinte sedinta' }).fill('Prof. Director Demo');
  await page.getByRole('textbox', { name: 'Secretar' }).fill('Secretar Demo');
  await page.getByRole('textbox', { name: 'Rezumat / scop' }).fill(
    'Sedinta dedicata aprobarii planului managerial si a calendarului de monitorizare.',
  );

  await page.getByRole('button', { name: 'Pasul urmator' }).click();
  await expect(page.getByText('Rezumat procedural')).toBeVisible();

  await page.getByRole('button', { name: 'Pasul urmator' }).click();
  await page.getByRole('button', { name: 'Creeaza sedinta' }).click();

  await expect.poll(() => createdPayload).not.toBeNull();
  expect(createdPayload).toMatchObject({
    school_year: '2025-2026',
    organism: 'ca',
    meeting_type: 'ordinary',
    status: 'draft',
    location: 'Sala profesorală',
    chairperson: 'Prof. Director Demo',
    secretary_name: 'Secretar Demo',
  });

  await page.waitForURL('**/education/governance');
  await expect(page.getByRole('heading', { name: 'Guvernanta si trasee procedurale' })).toBeVisible();
});
