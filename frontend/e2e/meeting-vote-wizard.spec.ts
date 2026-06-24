import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

test('meeting vote wizard submits a vote item for the selected meeting', async ({ page }) => {
  let createdVote: { meetingId: string; payload: Record<string, unknown> } | null = null;

  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page, {
    onCreateVote: (meetingId, payload) => {
      createdVote = { meetingId, payload };
    },
  });

  await page.goto('/education/governance/votes-wizard?meetingId=meeting-1');

  await expect(page.getByRole('heading', { name: 'Wizard vot si rezultat' })).toBeVisible();
  await page.getByRole('textbox', { name: 'Subiect' }).fill('Aprobarea procedurii de monitorizare');
  await page.getByRole('button', { name: 'Pasul urmator' }).click();

  await page.getByRole('spinbutton', { name: 'Voturi pentru' }).fill('6');
  await page.getByRole('spinbutton', { name: 'Voturi impotriva' }).fill('1');
  await page.getByRole('spinbutton', { name: 'Abtineri' }).fill('0');
  await page.getByRole('button', { name: 'Pasul urmator' }).click();

  await page.getByRole('textbox', { name: 'Temei legal' }).fill('Legea 198/2023 si regulamentul intern aplicabil.');
  await page.getByRole('button', { name: 'Pasul urmator' }).click();
  await page.getByRole('button', { name: 'Creeaza votul' }).click();

  await expect.poll(() => createdVote).not.toBeNull();
  expect(createdVote).toMatchObject({
    meetingId: 'meeting-1',
    payload: {
      agenda_order: 1,
      subject_title: 'Aprobarea procedurii de monitorizare',
      decision_type: 'hotarare',
      votes_for: 6,
      votes_against: 1,
      abstentions: 0,
      outcome: 'adoptat',
    },
  });
});
