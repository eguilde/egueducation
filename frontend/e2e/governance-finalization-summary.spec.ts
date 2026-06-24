import { expect, test } from '@playwright/test';

import { bootstrapAuthenticatedDirector, registerEducationApiMocks } from './support/education-mocks';

test('governance meeting detail shows finalization blockers and readiness summary', async ({ page }) => {
  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page);

  await page.goto('/education/governance?resource=meetings');

  const meetingRow = page.getByRole('row', { name: /Sedinta CA pentru aprobarea planului managerial/ }).first();
  await expect(meetingRow).toBeVisible();
  await meetingRow.locator('button').first().evaluate((button: HTMLButtonElement) => button.click());

  await expect(page.getByText('Stare de finalizare')).toBeVisible();
  await expect(page.getByText('Inchidere blocata')).toBeVisible();
  await expect(page.getByText('Publicare blocata')).toBeVisible();
  await expect(page.getByText('Cvorum incomplet')).toBeVisible();
  await expect(page.getByText('Cvorum necesar:')).toBeVisible();
  await expect(page.getByText('Lipsesc minutele')).toBeVisible();
  await expect(page.getByText('Lipsesc hotarari pentru voturile adoptate')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Verifica participantii' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Adauga minuta' })).toBeVisible();
});

test('governance meeting detail flags missing signatures as a blocker', async ({ page }) => {
  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page, {
    governanceMeetingDetail: {
      status: 'scheduled',
      quorum_required: 4,
      participants_count: 7,
    },
    governanceFinalizationSummary: {
      participants: {
        recorded_participants: 7,
        present_participants: 5,
        signed_participants: 3,
        voting_participants: 6,
      },
      votes: {
        total: 1,
        adopted: 1,
        requires_follow_up: 0,
        missing_resolutions: 0,
      },
      minutes: {
        total: 1,
        requires_publication: 0,
        open_follow_up_items: 0,
      },
      resolutions: {
        total: 1,
        ready_for_publication: 1,
        published: 0,
        pending_publication: 1,
        pending_anonymization: 0,
      },
      documents: {
        total: 1,
        process_verbal_documents: 1,
        published_process_verbals: 0,
      },
      readiness: {
        ready_to_close: false,
        ready_to_publish: false,
        blockers: ['signatures_missing'],
      },
    },
  });

  await page.goto('/education/governance?resource=meetings');

  const meetingRow = page.getByRole('row', { name: /Sedinta CA pentru aprobarea planului managerial/ }).first();
  await expect(meetingRow).toBeVisible();
  await meetingRow.locator('button').first().evaluate((button: HTMLButtonElement) => button.click());

  await expect(page.getByText('Semnaturi lipsa')).toBeVisible();
});

test('governance meeting detail can mark a ready meeting as held', async ({ page }) => {
  let patchedStatus = '';

  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page, {
    governanceMeetingDetail: {
      status: 'scheduled',
      quorum_required: 4,
      participants_count: 7,
    },
    governanceFinalizationSummary: {
      participants: {
        recorded_participants: 7,
        present_participants: 5,
        signed_participants: 5,
        voting_participants: 6,
      },
      votes: {
        total: 2,
        adopted: 1,
        requires_follow_up: 0,
        missing_resolutions: 0,
      },
      minutes: {
        total: 1,
        requires_publication: 0,
        open_follow_up_items: 0,
      },
      resolutions: {
        total: 1,
        ready_for_publication: 1,
        published: 0,
        pending_publication: 1,
        pending_anonymization: 0,
      },
      documents: {
        total: 2,
        process_verbal_documents: 1,
        published_process_verbals: 1,
      },
      readiness: {
        ready_to_close: true,
        ready_to_publish: false,
        blockers: [],
      },
    },
    onPatchMeeting: (payload) => {
      patchedStatus = String(payload.status ?? '');
    },
  });

  await page.goto('/education/governance?resource=meetings');

  const meetingRow = page.getByRole('row', { name: /Sedinta CA pentru aprobarea planului managerial/ }).first();
  await expect(meetingRow).toBeVisible();
  await meetingRow.locator('button').first().evaluate((button: HTMLButtonElement) => button.click());

  const markHeldButton = page.getByRole('button', { name: 'Marcheaza ca tinuta' });
  await expect(markHeldButton).toBeEnabled();
  await markHeldButton.click();

  await expect(page.getByText('Sedinta a fost marcata ca tinuta.')).toBeVisible();
  expect(patchedStatus).toBe('held');
});

test('governance meeting detail can mark a ready held meeting as published', async ({ page }) => {
  let patchedStatus = '';

  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page, {
    governanceMeetingDetail: {
      status: 'held',
      quorum_required: 4,
      participants_count: 7,
    },
    governanceFinalizationSummary: {
      participants: {
        recorded_participants: 7,
        present_participants: 5,
        signed_participants: 5,
        voting_participants: 6,
      },
      votes: {
        total: 2,
        adopted: 1,
        requires_follow_up: 0,
        missing_resolutions: 0,
      },
      minutes: {
        total: 1,
        requires_publication: 0,
        open_follow_up_items: 0,
      },
      resolutions: {
        total: 1,
        ready_for_publication: 1,
        published: 1,
        pending_publication: 0,
        pending_anonymization: 0,
      },
      documents: {
        total: 2,
        process_verbal_documents: 1,
        published_process_verbals: 1,
      },
      readiness: {
        ready_to_close: true,
        ready_to_publish: true,
        blockers: [],
      },
    },
    onPatchMeeting: (payload) => {
      patchedStatus = String(payload.status ?? '');
    },
  });

  await page.goto('/education/governance?resource=meetings');

  const meetingRow = page.getByRole('row', { name: /Sedinta CA pentru aprobarea planului managerial/ }).first();
  await expect(meetingRow).toBeVisible();
  await meetingRow.locator('button').first().evaluate((button: HTMLButtonElement) => button.click());

  const publishButton = page.getByRole('button', { name: 'Marcheaza ca publicata' });
  await expect(publishButton).toBeEnabled();
  await publishButton.click();

  await expect(page.getByText('Sedinta a fost marcata ca publicata.')).toBeVisible();
  expect(patchedStatus).toBe('published');
});

test('governance meeting detail exports the procedural summary as pdf', async ({ page }) => {
  let exportPayload: Record<string, unknown> | null = null;

  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page, {
    governanceMeetingDetail: {
      status: 'held',
      quorum_required: 4,
      participants_count: 7,
    },
    governanceFinalizationSummary: {
      participants: {
        recorded_participants: 7,
        present_participants: 5,
        signed_participants: 5,
        voting_participants: 6,
      },
      votes: {
        total: 2,
        adopted: 1,
        requires_follow_up: 0,
        missing_resolutions: 0,
      },
      minutes: {
        total: 1,
        requires_publication: 0,
        open_follow_up_items: 0,
      },
      resolutions: {
        total: 1,
        ready_for_publication: 1,
        published: 1,
        pending_publication: 0,
        pending_anonymization: 0,
      },
      documents: {
        total: 2,
        process_verbal_documents: 1,
        published_process_verbals: 1,
      },
      readiness: {
        ready_to_close: true,
        ready_to_publish: true,
        blockers: [],
      },
    },
  });

  await page.route('**/api/education/exports/pdf', async (route) => {
    exportPayload = route.request().postDataJSON() as Record<string, unknown>;
    await route.fulfill({
      status: 200,
      contentType: 'application/pdf',
      body: 'mock-governance-summary-pdf',
    });
  });

  await page.goto('/education/governance?resource=meetings');

  const meetingRow = page.getByRole('row', { name: /Sedinta CA pentru aprobarea planului managerial/ }).first();
  await expect(meetingRow).toBeVisible();
  await meetingRow.locator('button').first().evaluate((button: HTMLButtonElement) => button.click());

  await page.getByRole('button', { name: 'PDF sumar procedural' }).click();

  await expect.poll(() => exportPayload !== null).toBeTruthy();
  expect(exportPayload?.['title']).toBe('Sumar procedural sedinta - Sedinta CA pentru aprobarea planului managerial');
  expect(exportPayload?.['filename']).toBe('sumar-procedural-sedinta-ca-pentru-aprobarea-planului-managerial');
  expect(Array.isArray(exportPayload?.['rows'])).toBeTruthy();
});

test('governance meeting detail exposes pdf actions for minutes and resolutions', async ({ page }) => {
  await bootstrapAuthenticatedDirector(page);
  await registerEducationApiMocks(page, {
    governanceMinuteItems: [
      {
        id: 'minute-1',
        meeting_id: 'meeting-1',
        agenda_order: 1,
        topic_title: 'Aprobarea planului managerial',
        responsible_party: 'Secretar Demo',
        due_on: '2026-07-05',
        follow_up_status: 'in_urmarire',
        requires_publication: true,
      },
    ],
    governanceResolutionItems: [
      {
        id: 'resolution-1',
        meeting_id: 'meeting-1',
        resolution_code: 'HCA-2026-001',
        title: 'Aprobarea planului managerial',
        resolution_type: 'hotarare',
        publication_status: 'intern',
        anonymization_state: 'necesara',
        issued_on: '2026-06-30',
        signed_by: 'Prof. Director Demo',
      },
    ],
  });

  await page.goto('/education/governance?resource=meetings');

  const meetingRow = page.getByRole('row', { name: /Sedinta CA pentru aprobarea planului managerial/ }).first();
  await expect(meetingRow).toBeVisible();
  await meetingRow.locator('button').first().evaluate((button: HTMLButtonElement) => button.click());

  await page.getByRole('tab', { name: 'Proces-verbal structurat' }).click();
  await expect(page.getByRole('row', { name: /Aprobarea planului managerial.*Proces-verbal PDF/ }).first()).toBeVisible();

  await page.getByRole('tab', { name: 'Hotarari si avize' }).click();
  await expect(page.getByRole('row', { name: /HCA-2026-001.*Hotarare PDF/ }).first()).toBeVisible();
});
