import { describe, expect, it, vi } from 'vitest';
import { createWorkflowApi } from './api';

const response = (body: unknown) => new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } });

describe('Flux documente contract adapter', () => {
  it('uses Costești queue/mapa/pipeline server query contracts rather than client filtering', async () => {
    const fetcher = vi.fn().mockImplementation(() => response({ items: [], total: 0, page: 2, pageSize: 20 }));
    const api = createWorkflowApi(fetcher);
    await api.queue({ page: 2, pageSize: 20, nr_doc: 'REG-9', sort: 'registry_number', direction: 'asc' });
    await api.mapa({ page: 1, pageSize: 20, mapa_filter: 'peers', compartiment: 'Secretariat' });
    await api.pipeline({ page: 1, pageSize: 20, status: 'FLUX_APROBARE', continut: 'avizare' });
    expect((fetcher.mock.calls[0][0] as Request).url).toContain('/api/registratura/flux/queue?');
    expect((fetcher.mock.calls[0][0] as Request).url).toContain('nr_doc=REG-9');
    expect((fetcher.mock.calls[1][0] as Request).url).toContain('mapa_filter=peers');
    expect((fetcher.mock.calls[2][0] as Request).url).toContain('status=FLUX_APROBARE');
  });

  it('uses the generated workflow-action path and authenticated transport', async () => {
    const fetcher = vi.fn().mockImplementation(() => response({}));
    const api = createWorkflowApi(fetcher);
    await api.transition('doc/a', { action: 'reject', expected_version: 7, note: 'Motiv justificat' });
    const request = fetcher.mock.calls[0][0] as Request;
    expect(request.url).toContain('/api/registratura/documents/doc%2Fa/workflow-actions');
    expect(request.method).toBe('POST');
    expect(await request.json()).toEqual({ action: 'reject', expected_version: 7, note: 'Motiv justificat' });
  });
});
