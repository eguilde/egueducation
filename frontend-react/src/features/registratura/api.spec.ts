import { describe, expect, it, vi } from 'vitest';
import { createRegistraturaApi } from './api';

describe('Registratura API adapter', () => {
  it('scopes document requests to the selected registry and sends typed filters', async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ items: [], total: 0, page: 1, pageSize: 50 }), { headers: { 'content-type': 'application/json' } }));
    await createRegistraturaApi(fetcher, '/api').documents({ registryId: 7, page: 1, pageSize: 50, filters: { q: 'cerere', direction: 'intrare' } });
    expect(String(fetcher.mock.calls[0][0])).toContain('filter.registru_id=7'); expect(String(fetcher.mock.calls[0][0])).toContain('filter.direction=intrare'); expect(String(fetcher.mock.calls[0][0])).toContain('q=cerere');
  });
  it('uploads the scanned file as multipart without a client-controlled tenant header', async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ id: 'a1' }), { status: 201, headers: { 'content-type': 'application/json' } }));
    await createRegistraturaApi(fetcher, '/api').upload('doc 1', new File(['safe'], 'scan.pdf', { type: 'application/pdf' }), 'primary');
    const [url, init] = fetcher.mock.calls[0];
    expect(String(url)).toContain('/documents/doc%201/attachments/upload');
    expect(init.body).toBeInstanceOf(FormData);
    expect(init.headers['X-Institution-ID']).toBeUndefined();
  });
  it('sends expected workflow version and assignment only to the action endpoint', async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ id: 'doc-1' }), { headers: { 'content-type': 'application/json' } }));
    await createRegistraturaApi(fetcher, '/api').workflow('doc-1', { action: 'assign_department', department_id: 'department-1', expected_version: 4, note: null });
    expect(String(fetcher.mock.calls[0][0])).toContain('/workflow-actions');
    expect(JSON.parse(fetcher.mock.calls[0][1].body)).toEqual({ action: 'assign_department', department_id: 'department-1', expected_version: 4, note: null });
  });
  it('uses server pagination, safe sort and advanced date filters rather than filtering client-side', async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ items: [], total: 92, page: 2, pageSize: 50 }), { headers: { 'content-type': 'application/json' } }));
    await createRegistraturaApi(fetcher, '/api').documents({ registryId: 7, page: 2, pageSize: 50, sort: 'registered_at', direction: 'desc', filters: { registered_at_from: '2026-01-01', confidentiality: 'restricted' } });
    const request = String(fetcher.mock.calls[0][0]);
    expect(request).toContain('page=2'); expect(request).toContain('sort=registered_at'); expect(request).toContain('direction=desc'); expect(request).toContain('filter.registered_at_from=2026-01-01'); expect(request).toContain('filter.confidentiality=restricted');
  });
  it('uses the version endpoint for an auditable document revision', async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ id: 'v2' }), { headers: { 'content-type': 'application/json' } }));
    await createRegistraturaApi(fetcher, '/api').createVersion('doc-1', { subject: 'Actualizat', status: 'INCOMING', assigned_to: '', confidentiality: 'normal', summary: '', change_notes: 'Corectare subiect' });
    expect(String(fetcher.mock.calls[0][0])).toContain('/documents/doc-1/versions');
    expect(JSON.parse(fetcher.mock.calls[0][1].body).change_notes).toBe('Corectare subiect');
  });
  it('keeps tenant user enumeration on the admin endpoint and assignments on Registratură endpoints', async () => {
    const fetcher = vi.fn().mockImplementation(() => Promise.resolve(new Response(JSON.stringify({ items: [] }), { headers: { 'content-type': 'application/json' } })));
    const api = createRegistraturaApi(fetcher, '/api');
    await api.adminUsers();
    expect(String(fetcher.mock.calls[0][0])).toBe('/api/admin/users?page=1&pageSize=100&sort=name&direction=asc');
    await api.saveUserAssignments('user-1', { department_ids: ['dept-1'], primary_department_id: 'dept-1', organization_id: null });
    expect(String(fetcher.mock.calls[1][0])).toContain('/api/registratura/admin/users/user-1/assignments');
    expect(fetcher.mock.calls[1][1].method).toBe('PUT');
  });
});
