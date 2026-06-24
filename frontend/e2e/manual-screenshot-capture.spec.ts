import path from 'node:path';
import { expect, test } from '@playwright/test';

import {
  bootstrapAuthenticatedDirector,
  bootstrapAuthenticatedEducationUser,
  bootstrapAuthenticatedSecretariat,
  registerEducationApiMocks,
} from './support/education-mocks';

const screenshotDir = path.resolve(__dirname, '../../.scratch-help-manual/screenshots');

async function registerPlatformWorkspaceMocks(page: Parameters<typeof test>[0]['page']): Promise<void> {
  await page.route('**/api/**', async (route) => {
    const url = route.request().url();
    const method = route.request().method();

    if (url.endsWith('/api/registratura/registre') && method === 'GET') {
      return fulfill(route, [
        {
          id: 1,
          nume: 'Registru General',
          prefix_nr: 'RG',
          nr_inceput: 1,
          nr_curent: '154',
          nr_urmator: '155',
          data_resetare: null,
          tip_registru: 'general',
          isDefault: true,
          created_at: '2026-01-10T08:00:00Z',
          updated_at: '2026-06-24T08:00:00Z',
        },
        {
          id: 2,
          nume: 'Registru decizii',
          prefix_nr: 'DEC',
          nr_inceput: 1,
          nr_curent: '48',
          nr_urmator: '49',
          data_resetare: null,
          tip_registru: 'decizii',
          isDefault: false,
          created_at: '2026-01-10T08:00:00Z',
          updated_at: '2026-06-24T08:00:00Z',
        },
      ]);
    }

    if (url.endsWith('/api/registratura/registre/default') && method === 'GET') {
      return fulfill(route, {
        id: 1,
        nume: 'Registru General',
        prefix_nr: 'RG',
        nr_inceput: 1,
        nr_curent: '154',
        nr_urmator: '155',
        data_resetare: null,
        tip_registru: 'general',
        isDefault: true,
        created_at: '2026-01-10T08:00:00Z',
        updated_at: '2026-06-24T08:00:00Z',
      });
    }

    if (url.endsWith('/api/registratura/documents/filters') && method === 'GET') {
      return fulfill(route, {
        document_types: ['Adresa', 'Cerere', 'Decizie', 'Proces-verbal', 'Adeverinta'],
        directions: ['intrare', 'iesire', 'intern'],
        statuses: ['draft', 'registered', 'in_workflow', 'resolved', 'archived'],
        confidentialities: ['normal', 'confidential', 'strict_confidential'],
      });
    }

    if (url.includes('/api/registratura/parties/lookup') && method === 'GET') {
      return fulfill(route, [
        {
          id: 'party-1',
          code: 'SJ-001',
          display_name: 'Inspectoratul Scolar Judetean',
          party_type: 'legal',
          email: 'contact@isj.example.test',
        },
        {
          id: 'party-2',
          code: 'SC-001',
          display_name: 'Scoala Gimnaziala Demonstrativa',
          party_type: 'legal',
          email: 'secretariat@example.test',
        },
      ]);
    }

    if (/\/api\/registratura\/parties(?:\?.*)?$/.test(url) && method === 'GET') {
      return fulfill(route, {
        items: [
          {
            id: 'party-1',
            code: 'SJ-001',
            display_name: 'Inspectoratul Scolar Judetean',
            party_type: 'legal',
            email: 'contact@isj.example.test',
          },
          {
            id: 'party-2',
            code: 'SC-001',
            display_name: 'Scoala Gimnaziala Demonstrativa',
            party_type: 'legal',
            email: 'secretariat@example.test',
          },
        ],
        total: 2,
        page: 1,
        pageSize: 20,
      });
    }

    if (url.includes('/api/registratura/documents/lookup') && method === 'GET') {
      return fulfill(route, [
        {
          id: 'doc-2',
          registry_number: 'RG-2026-00098',
          subject: 'Solicitare situatie statistica',
          document_type: 'Adresa',
          status: 'registered',
        },
      ]);
    }

    if (url.includes('/api/registratura/document-links') && method === 'GET') {
      return fulfill(route, [
        {
          link_id: 'link-1',
          registry_number: 'RG-2026-00098',
          subject: 'Solicitare situatie statistica',
          document_type: 'Adresa',
          relation_type: 'referinta',
          status: 'registered',
        },
      ]);
    }

    if (/\/api\/registratura\/documents\/doc-1\/versions(?:\?.*)?$/.test(url) && method === 'GET') {
      return fulfill(route, [
        {
          id: 'ver-1',
          version_no: 1,
          change_notes: 'Inregistrare initiala',
          created_at: '2026-06-20T09:00:00Z',
          created_by: 'Secretariat Demo',
        },
        {
          id: 'ver-2',
          version_no: 2,
          change_notes: 'Actualizare rezumat si termen',
          created_at: '2026-06-22T11:30:00Z',
          created_by: 'Director Demo',
        },
      ]);
    }

    if (/\/api\/registratura\/documents\/doc-1\/attachments(?:\?.*)?$/.test(url) && method === 'GET') {
      return fulfill(route, [
        {
          id: 'att-1',
          title: 'Adresa scanata',
          file_name: 'adresa-scanata.pdf',
          mime_type: 'application/pdf',
        },
        {
          id: 'att-2',
          title: 'Anexa tabel centralizator',
          file_name: 'centralizator.xlsx',
          mime_type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
        },
      ]);
    }

    if (/\/api\/registratura\/documents\/doc-1(?:\?.*)?$/.test(url) && method === 'GET') {
      return fulfill(route, {
        id: 'doc-1',
        registry_number: 'RG-2026-00154',
        registru_id: 1,
        subject: 'Solicitare actualizare plan operational',
        document_type: 'Adresa',
        direction: 'intrare',
        status: 'registered',
        confidentiality: 'normal',
        correspondent: 'Inspectoratul Scolar Judetean',
        assigned_to: 'Director',
        summary: 'Document de intrare pentru actualizarea planului operational si transmiterea raspunsului in termen.',
        due_date: '2026-06-30',
        registered_at: '2026-06-24T08:45:00Z',
      });
    }

    if (url.includes('/api/registratura/documents') && method === 'GET' && !url.includes('/filters') && !url.includes('/lookup') && !url.includes('/versions') && !url.includes('/attachments') && !/\/api\/registratura\/documents\/[^/?]+$/.test(url)) {
      return fulfill(route, {
        items: [
          {
            id: 'doc-1',
            registry_number: 'RG-2026-00154',
            registru_id: 1,
            subject: 'Solicitare actualizare plan operational',
            document_type: 'Adresa',
            direction: 'intrare',
            status: 'registered',
            confidentiality: 'normal',
            correspondent: 'Inspectoratul Scolar Judetean',
            assigned_to: 'Director',
            summary: 'Document de intrare pentru actualizarea planului operational.',
            due_date: '2026-06-30',
            registered_at: '2026-06-24T08:45:00Z',
          },
          {
            id: 'doc-2',
            registry_number: 'RG-2026-00155',
            registru_id: 1,
            subject: 'Raspuns catre ISJ privind portofoliile cadrelor didactice',
            document_type: 'Adresa',
            direction: 'iesire',
            status: 'in_workflow',
            confidentiality: 'normal',
            correspondent: 'Scoala Gimnaziala Demonstrativa',
            assigned_to: 'ISJ',
            summary: 'Raspuns pregatit pentru avizare si semnare.',
            due_date: '2026-06-28',
            registered_at: '2026-06-24T10:10:00Z',
          },
        ],
        total: 2,
        page: 1,
        pageSize: 20,
      });
    }

    if (url.endsWith('/api/workflow/dashboard') && method === 'GET') {
      return fulfill(route, {
        stats: {
          active_tasks: 12,
          overdue_tasks: 2,
          waiting_approval: 4,
          active_definitions: 5,
        },
      });
    }

    if (url.endsWith('/api/workflow/definitions') && method === 'GET') {
      return fulfill(route, [
        {
          code: 'doc-approval',
          name: 'Aprobare document',
          description: 'Circuit de verificare, avizare si aprobare documente institutionale.',
        },
        {
          code: 'archive-transfer',
          name: 'Transfer spre arhiva',
          description: 'Pregatirea documentelor finalizate pentru arhivare electronica.',
        },
      ]);
    }

    if (url.endsWith('/api/workflow/tasks/filters') && method === 'GET') {
      return fulfill(route, {
        statuses: ['open', 'in_progress', 'waiting_approval', 'done'],
        priorities: ['low', 'medium', 'high', 'urgent'],
        modules: ['registratura', 'education', 'earchiva'],
      });
    }

    if (url.includes('/api/workflow/tasks') && method === 'GET' && !url.includes('/filters')) {
      return fulfill(route, {
        items: [
          {
            id: 'task-1',
            title: 'Avizare raspuns catre ISJ',
            definition_code: 'doc-approval',
            status: 'waiting_approval',
            priority: 'high',
            assigned_to: 'Director',
            due_at: '2026-06-26',
            source_module: 'registratura',
          },
          {
            id: 'task-2',
            title: 'Pregatire arhivare lot iunie',
            definition_code: 'archive-transfer',
            status: 'in_progress',
            priority: 'medium',
            assigned_to: 'Arhivar',
            due_at: '2026-06-30',
            source_module: 'earchiva',
          },
        ],
        total: 2,
        page: 1,
        pageSize: 20,
      });
    }

    if (url.endsWith('/api/earchiva/dashboard') && method === 'GET') {
      return fulfill(route, {
        stats: {
          total_records: 248,
          validated_records: 231,
          draft_records: 17,
          unique_fonds: 6,
        },
      });
    }

    if (url.endsWith('/api/earchiva/records/filters') && method === 'GET') {
      return fulfill(route, {
        fonds: ['registratura', 'management', 'resurse-umane'],
        statuses: ['draft', 'validated', 'archived'],
        archivists: ['Arhivar Demo', 'Secretariat Demo'],
      });
    }

    if (url.includes('/api/earchiva/records') && method === 'GET' && !url.includes('/filters')) {
      return fulfill(route, {
        items: [
          {
            id: 'arch-1',
            title: 'Dosar documente intrare iunie 2026',
            fond: 'registratura',
            series: 'documente intrare',
            source_reference: 'RG-2026-00154',
            status: 'validated',
            assigned_archivist: 'Arhivar Demo',
            archived_at: '2026-06-24',
            box_number: 'BX-12',
            location_code: 'R1-S2',
          },
          {
            id: 'arch-2',
            title: 'Hotarari CA semestrul II',
            fond: 'management',
            series: 'hotarari',
            source_reference: 'HCA-2026-014',
            status: 'draft',
            assigned_archivist: 'Secretariat Demo',
            archived_at: '2026-06-23',
            box_number: 'BX-09',
            location_code: 'R2-S1',
          },
        ],
        total: 2,
        page: 1,
        pageSize: 20,
      });
    }

    if (url.includes('/api/registratura/') && ['POST', 'PATCH', 'DELETE'].includes(method)) {
      return fulfill(route, { status: 'ok' });
    }
    if (url.includes('/api/workflow/') && ['POST', 'PATCH', 'DELETE'].includes(method)) {
      return fulfill(route, { status: 'ok' });
    }
    if (url.includes('/api/earchiva/') && ['POST', 'PATCH', 'DELETE'].includes(method)) {
      return fulfill(route, { status: 'ok' });
    }

    return route.continue();
  });
}

function fulfill(route: Parameters<Parameters<typeof test>[0]['page']['route']>[1], body: unknown) {
  return route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify(body),
  });
}

async function capture(page: Parameters<typeof test>[0]['page'], route: string, fileName: string): Promise<void> {
  await page.goto(route, { waitUntil: 'networkidle' });
  const expectedPath = new URL(route, 'http://127.0.0.1:4200').pathname;
  await expect(page).toHaveURL(new RegExp(`${expectedPath.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}`));
  await expect(page.locator('body')).toBeVisible();
  await page.screenshot({
    path: path.join(screenshotDir, fileName),
    fullPage: true,
  });
}

test.describe('manual screenshots', () => {
  test('director screenshots', async ({ page }) => {
    await bootstrapAuthenticatedDirector(page);
    await registerEducationApiMocks(page);
    await registerPlatformWorkspaceMocks(page);

    await capture(page, '/documente', '01-registratura-documente.png');
    await capture(page, '/documente/new', '02-formular-document-nou.png');
    await capture(page, '/documente/doc-1', '03-detaliu-document.png');
    await capture(page, '/registre', '04-registre.png');
    await capture(page, '/workflow', '05-workflow-documente.png');
    await capture(page, '/earchiva', '06-earchiva.png');
    await capture(page, '/education/dashboard', '07-dashboard-general.png');
    await capture(page, '/education/dashboard/director', '08-cockpit-director.png');
    await capture(page, '/education/dashboard/director/reports', '09-rapoarte-standard.png');
    await capture(page, '/education/governance', '10-guvernanta.png');
    await capture(page, '/education/governance/ca-wizard', '11-wizard-sedinta-ca.png');
    await capture(page, '/education/governance/minutes-wizard', '12-wizard-minuta.png');
    await capture(page, '/education/governance/votes-wizard', '13-wizard-vot.png');
    await capture(page, '/education/governance/resolutions-wizard', '14-wizard-hotarare.png');
    await capture(page, '/education/governance/managerial-wizard', '15-wizard-dosar-managerial.png');
    await capture(page, '/education/personnel', '16-personal.png');
    await capture(page, '/education/personnel/wizard', '17-wizard-cadru-didactic.png');
    await capture(page, '/education/personnel/evaluations-wizard', '18-wizard-evaluare.png');
    await capture(page, '/education/personnel/declarations-wizard', '19-wizard-declaratie.png');
    await capture(page, '/education/personnel/mobility-wizard', '20-wizard-mobilitate.png');
    await capture(page, '/education/personnel/merit-wizard', '21-wizard-gradatie.png');
    await capture(page, '/education/portfolio', '22-portofolii.png');
    await capture(page, '/education/portfolio/wizard', '23-wizard-portofoliu.png');
    await capture(page, '/education/compliance', '24-conformitate.png');
    await capture(page, '/help', '25-help-center.png');
  });

  test('teacher screenshots', async ({ page }) => {
    await bootstrapAuthenticatedEducationUser(page, {
      userId: 'teacher-1',
      name: 'Profesor Demo',
      email: 'teacher@example.test',
      roles: ['profesor'],
      permissions: [
        'education.portfolios.read',
        'education.evaluations.read',
        'education.declarations.read',
        'education.mobility.read',
      ],
      modules: [{ code: 'education', active: true }],
    });
    await registerEducationApiMocks(page, {
      me: {
        userId: 'teacher-1',
        name: 'Profesor Demo',
        email: 'teacher@example.test',
        roles: ['profesor'],
        institutionName: 'Scoala Gimnaziala Demonstrativa',
        permissions: [
          'education.portfolios.read',
          'education.evaluations.read',
          'education.declarations.read',
          'education.mobility.read',
        ],
      },
    });

    await capture(page, '/education/dashboard/teacher', '26-cockpit-profesor.png');
    await capture(page, '/education/portfolio/me', '27-portofoliul-meu.png');
    await capture(page, '/education/portfolio/workflow?owner_name=Profesor%20Demo', '28-workflow-portofoliu.png');
  });

  test('secretariat screenshots', async ({ page }) => {
    await bootstrapAuthenticatedSecretariat(page);
    await registerEducationApiMocks(page, {
      me: {
        userId: 'secretar-1',
        name: 'Secretariat Demo',
        email: 'secretariat@example.test',
        roles: ['secretar'],
        institutionName: 'Scoala Gimnaziala Demonstrativa',
        permissions: [
          'education.read',
          'education.governance.read',
          'education.personnel.read',
          'education.portfolios.read',
          'education.managerial.read',
        ],
      },
    });

    await capture(page, '/education/dashboard/secretariat', '29-cockpit-secretariat.png');
    await capture(page, '/education/dashboard/compliance', '30-cockpit-conformitate.png');
  });
});
