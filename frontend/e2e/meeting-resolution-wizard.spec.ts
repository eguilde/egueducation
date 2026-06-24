import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

test('meeting resolution wizard submits a resolution for the selected vote', async ({ page }) => {
  let createdResolution: { meetingId: string; payload: Record<string, unknown> } | null = null;

  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page, {
    onCreateResolution: (meetingId, payload) => {
      createdResolution = { meetingId, payload };
    },
  });

  await page.goto('/education/governance/resolutions-wizard?meetingId=meeting-1&voteId=vote-1');

  await expect(page.getByRole('heading', { name: 'Wizard hotarare si aviz' })).toBeVisible();
  await page.getByRole('button', { name: 'Pasul urmator' }).click();

  await page.getByRole('textbox', { name: 'Titlu hotarare / aviz' }).fill('Hotarare privind procedura de monitorizare');
  await page.getByRole('textbox', { name: 'Semnat de' }).fill('Prof. Director Demo');
  await page.getByRole('button', { name: 'Pasul urmator' }).click();

  await page.getByRole('button', { name: 'Pasul urmator' }).click();
  await page.getByRole('button', { name: 'Creeaza hotararea' }).click();

  await expect.poll(() => createdResolution).not.toBeNull();
  expect(createdResolution).toMatchObject({
    meetingId: 'meeting-1',
    payload: {
      vote_id: 'vote-1',
      title: 'Hotarare privind procedura de monitorizare',
      resolution_type: 'hotarare',
      publication_status: 'intern',
      anonymization_state: 'necesara',
      signed_by: 'Prof. Director Demo',
    },
  });
});
