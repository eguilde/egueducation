import { createRegistraturaApi } from './api';
import type { CreateDocumentInput } from './types';

const document = {
  id: '9dc429da-7549-460d-9376-979769c35a03',
  registry_number: 'I-42',
  subject: 'Cerere contract',
  document_type: 'CERERE',
  direction: 'intrare',
  status: 'INCOMING',
  correspondent: 'Solicitant',
  assigned_to: 'Secretariat',
  registered_at: '2026-09-08T09:00:00Z',
};

describe('Registratura generated contract transport', () => {
  it('serializes server-side paging, sorting and global search through OpenAPI', async () => {
    let observed: Request | undefined;
    const api = createRegistraturaApi(async (input) => {
      observed = input as Request;
      return new Response(JSON.stringify({ items: [document], total: 1, page: 2, pageSize: 20 }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      });
    });

    const result = await api.documents({
      registryId: 7,
      page: 2,
      pageSize: 20,
      filters: { q: 'contract', status: 'INCOMING' },
      sort: 'subject',
      direction: 'desc',
    });

    expect(result.total).toBe(1);
    const url = new URL(observed!.url);
    expect(url.pathname).toBe('/api/registratura/documents');
    expect(url.searchParams.get('page')).toBe('2');
    expect(url.searchParams.get('pageSize')).toBe('20');
    expect(url.searchParams.get('filter.registru_id')).toBe('7');
    expect(url.searchParams.get('q')).toBe('contract');
    expect(url.searchParams.get('filter.status')).toBe('INCOMING');
    expect(url.searchParams.get('sort')).toBe('subject');
    expect(url.searchParams.get('direction')).toBe('desc');
  });

  it('sends the handler snake_case create contract and accepts 201', async () => {
    let observed: Request | undefined;
    const api = createRegistraturaApi(async (input) => {
      observed = input as Request;
      return new Response(JSON.stringify(document), {
        status: 201,
        headers: { 'Content-Type': 'application/json' },
      });
    });
    const input: CreateDocumentInput = {
      registru_id: 7,
      subject: 'Cerere contract',
      document_type: 'CERERE',
      direction: 'intrare',
      status: 'INCOMING',
      correspondent: 'Solicitant',
      assigned_to: 'Secretariat',
      confidentiality: 'NORMAL',
      summary: '',
      activity: null,
      external_number: null,
    };

    await api.create(input);

    expect(observed!.method).toBe('POST');
    const body = JSON.parse(await observed!.text());
    expect(body.registru_id).toBe(7);
    expect(body.registryId).toBeUndefined();
    expect(body.activity).toBeUndefined();
    expect(body.external_number).toBeUndefined();
  });

  it('rejects a list response that violates the generated runtime schema', async () => {
    const api = createRegistraturaApi(async () => new Response(JSON.stringify({ items: [] }), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    }));

    await expect(api.documents({ registryId: 7, page: 1, pageSize: 20, filters: {} }))
      .rejects.toThrow('invalid conform OpenAPI');
  });
});
