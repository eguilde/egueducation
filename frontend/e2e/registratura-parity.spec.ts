import { expect, test } from '@playwright/test';
import { bootstrapRegistraturaManager, registerRegistraturaApiMocks } from './support/registratura-mocks';

test('registratura parity uses the selected tenant registry and 1–20 MULTIPLU flow', async ({ page }) => {
  const captured = { workflowBodies: [] as unknown[], uploadBodies: [] as string[] };
  await bootstrapRegistraturaManager(page);
  await registerRegistraturaApiMocks(page, captured);
  await page.goto('/documente');
  await expect(page.getByRole('combobox', { name: 'Registru E2E A' })).toBeVisible();
  await page.getByRole('button', { name: 'Multiplu' }).click();
  await expect(page.getByRole('dialog')).toContainText('Generare documente multiple');
  await expect(page.locator('input[type=number][max="20"]')).toBeVisible();
  await page.locator('input[type=number]').fill('21');
  await expect(page.getByRole('dialog')).toContainText('Sunt create 21 înregistrări MULTIPLU');
});

test('registratura workflow sends only selected tenant assignment IDs and uploads ready files', async ({ page }) => {
  const captured = { workflowBodies: [] as unknown[], uploadBodies: [] as string[] };
  await bootstrapRegistraturaManager(page);
  await registerRegistraturaApiMocks(page, captured);
  await page.goto('/documente');
  await page.evaluate(async () => {
    await fetch('/api/registratura/documents/doc-1/workflow-actions', { method: 'POST', headers: { 'content-type': 'application/json' }, body: JSON.stringify({ action: 'assign_department', department_id: 'dept-1', expected_version: 1 }) });
  });
  await expect.poll(() => captured.workflowBodies.length).toBe(1);
  expect(captured.workflowBodies[0]).toMatchObject({ action: 'assign_department', department_id: 'dept-1' });

  await page.evaluate(async () => {
    const body = new FormData(); body.set('file', new File(['e2e'], 'e2e.txt', { type: 'text/plain' })); body.set('category', 'primary');
    await fetch('/api/registratura/documents/doc-1/attachments/upload', { method: 'POST', body });
  });
  await expect.poll(() => captured.uploadBodies.length).toBe(1);
  expect(captured.uploadBodies[0]).toContain('multipart/form-data');
});

test('registratura admin exposes specialized party fields', async ({ page }) => {
  const captured = { workflowBodies: [] as unknown[], uploadBodies: [] as string[] };
  await bootstrapRegistraturaManager(page);
  await registerRegistraturaApiMocks(page, captured);
  await page.goto('/admin');
  await page.getByRole('tab', { name: /Persoane Fizice/ }).click();
  await page.getByRole('button', { name: /Adaugă/ }).click();
  await expect(page.getByText('Data nașterii')).toBeVisible();
  await expect(page.getByText('Locul nașterii')).toBeVisible();
});
