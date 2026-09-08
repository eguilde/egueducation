import { describe, expect, it, vi } from 'vitest';
import { createRegistraturaApi } from './api';

const requestAt = (fetcher: ReturnType<typeof vi.fn>, index = 0) => fetcher.mock.calls[index][0] as Request;

describe('Registratura API adapter', () => {
  it('scopes document requests to the selected registry and sends typed filters', async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ items: [], total: 0, page: 1, pageSize: 50 }), { headers: { 'content-type': 'application/json' } }));
    await createRegistraturaApi(fetcher, '/api').documents({ registryId: 7, page: 1, pageSize: 50, filters: { q: 'cerere', direction: 'intrare' } });
    const request = fetcher.mock.calls[0][0] as Request;
    expect(request.url).toContain('filter.registru_id=7'); expect(request.url).toContain('filter.direction=intrare'); expect(request.url).toContain('q=cerere');
  });
  it('uploads the scanned file as multipart without a client-controlled tenant header', async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ id: 'a1' }), { status: 201, headers: { 'content-type': 'application/json' } }));
    await createRegistraturaApi(fetcher, '/api').upload('doc 1', new File(['safe'], 'scan.pdf', { type: 'application/pdf' }), 'primary');
    const request = requestAt(fetcher);
    expect(request.url).toContain('/documents/doc%201/attachments/upload');
    expect(request.credentials).toBe('include');
    expect(request.headers.get('X-Institution-ID')).toBeNull();
    // Browser multipart serialization is verified by the Playwright system
    // suite; this adapter unit test verifies its protected OpenAPI transport.
  });
  it('sends expected workflow version and assignment only to the action endpoint', async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ id: 'doc-1' }), { headers: { 'content-type': 'application/json' } }));
    await createRegistraturaApi(fetcher, '/api').workflow('doc-1', { action: 'assign_department', department_id: 'department-1', expected_version: 4, note: null });
    const request = requestAt(fetcher);
    expect(request.url).toContain('/workflow-actions');
    expect(await request.json()).toEqual({ action: 'assign_department', department_id: 'department-1', expected_version: 4, note: null });
  });
  it('uses server pagination, safe sort and advanced date filters rather than filtering client-side', async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ items: [], total: 92, page: 2, pageSize: 50 }), { headers: { 'content-type': 'application/json' } }));
    await createRegistraturaApi(fetcher, '/api').documents({ registryId: 7, page: 2, pageSize: 50, sort: 'registered_at', direction: 'desc', filters: { registered_at_from: '2026-01-01', confidentiality: 'restricted' } });
    const request = fetcher.mock.calls[0][0] as Request;
    expect(request.url).toContain('page=2'); expect(request.url).toContain('sort=registered_at'); expect(request.url).toContain('direction=desc'); expect(request.url).toContain('filter.registered_at_from=2026-01-01'); expect(request.url).toContain('filter.confidentiality=restricted');
  });
  it('uses the version endpoint for an auditable document revision', async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ id: 'v2' }), { headers: { 'content-type': 'application/json' } }));
    await createRegistraturaApi(fetcher, '/api').createVersion('doc-1', { subject: 'Actualizat', status: 'INCOMING', assigned_to: '', confidentiality: 'normal', summary: '', change_notes: 'Corectare subiect' });
    const request = requestAt(fetcher);
    expect(request.url).toContain('/documents/doc-1/versions');
    expect((await request.json() as { change_notes: string }).change_notes).toBe('Corectare subiect');
  });
  it('parses document print and register export as PDF blobs', async () => {
    const fetcher = vi.fn().mockImplementation(() => Promise.resolve(new Response(new Blob(['%PDF-1.7'], { type: 'application/pdf' }), { status: 200, headers: { 'content-type': 'application/pdf' } })));
    const api = createRegistraturaApi(fetcher, '/api');
    await expect(api.print('doc-1')).resolves.toBeInstanceOf(Blob);
    expect(requestAt(fetcher).headers.get('accept')).toBe('application/pdf');
    await expect(api.exportPdf({ registru_id: 7, start_date: '2026-09-01', end_date: '2026-09-08' })).resolves.toBeInstanceOf(Blob);
    expect(requestAt(fetcher, 1).headers.get('accept')).toBe('application/pdf');
    expect(await requestAt(fetcher, 1).json()).toEqual({ registru_id: 7, start_date: '2026-09-01', end_date: '2026-09-08' });
  });
  it('keeps tenant user enumeration on the admin endpoint and assignments on Registratură endpoints', async () => {
    const fetcher = vi.fn().mockImplementation(() => Promise.resolve(new Response(JSON.stringify({ items: [] }), { headers: { 'content-type': 'application/json' } })));
    const api = createRegistraturaApi(fetcher, '/api');
    await api.adminUsers();
    expect(requestAt(fetcher).url).toBe('http://localhost:3000/api/admin/users?page=1&pageSize=100&sort=name&direction=asc');
    await api.saveUserAssignments('user-1', { department_ids: ['dept-1'], primary_department_id: 'dept-1', organization_id: null });
    expect(requestAt(fetcher, 1).url).toContain('/api/registratura/admin/users/user-1/assignments');
    expect(requestAt(fetcher, 1).method).toBe('PUT');
  });
});
