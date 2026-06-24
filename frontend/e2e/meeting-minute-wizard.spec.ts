import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

test('meeting minute wizard submits a minute item for the selected meeting', async ({ page }) => {
  let createdMinute: { meetingId: string; payload: Record<string, unknown> } | null = null;

  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page, {
    onCreateMinute: (meetingId, payload) => {
      createdMinute = { meetingId, payload };
    },
  });

  await page.goto('/education/governance/minutes-wizard');

  await expect(page.getByRole('heading', { name: 'Wizard punct de minuta' })).toBeVisible();

  await page.getByRole('combobox', { name: 'Sedinta' }).click();
  await page.getByText('CA • 2026-06-30 • Sedinta CA pentru aprobarea planului managerial').click();
  await page.getByRole('spinbutton', { name: 'Ordine pe agenda' }).fill('2');
  await page.getByRole('textbox', { name: 'Titlu subiect' }).fill('Aprobarea calendarului de monitorizare');
  await page.getByRole('button', { name: 'Pasul urmator' }).click();

  await page.getByRole('textbox', { name: 'Rezumat discutii' }).fill(
    'Membrii CA au analizat calendarul propus si au convenit corelarea lui cu activitatile CEAC.',
  );
  await page.getByRole('textbox', { name: 'Rezumat decizie' }).fill(
    'Calendarul a fost aprobat cu ajustari minore si va fi comunicat cadrelor didactice.',
  );
  await page.getByRole('button', { name: 'Pasul urmator' }).click();

  await page.getByRole('textbox', { name: 'Responsabil' }).fill('Director adjunct');
  await page.getByRole('button', { name: 'Pasul urmator' }).click();

  await page.getByRole('button', { name: 'Creeaza punctul de minuta' }).click();

  await expect.poll(() => createdMinute).not.toBeNull();
  expect(createdMinute).toMatchObject({
    meetingId: 'meeting-1',
    payload: {
      agenda_order: 2,
      topic_title: 'Aprobarea calendarului de monitorizare',
      responsible_party: 'Director adjunct',
      follow_up_status: 'de_stabilit',
      requires_publication: false,
    },
  });

  await page.waitForURL('**/education/governance');
  await expect(page.getByRole('heading', { name: 'Guvernanta si trasee procedurale' })).toBeVisible();
});
