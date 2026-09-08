import { describe, expect, it, vi } from 'vitest';
import { contractServerBaseUrl, createOpenApiTransport } from './client';

describe('generated OpenAPI transport', () => {
  it('normalizes the API base and includes browser credentials', async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ items: [] }), { headers: { 'content-type': 'application/json' } }));
    await createOpenApiTransport(fetcher, '/api').request('GET', '/api/admin/users?page=1');
    const request = fetcher.mock.calls[0][0] as Request;
    expect(contractServerBaseUrl('/api', 'https://tenant.eguilde.cloud')).toBe('https://tenant.eguilde.cloud');
    expect(request.url).toBe('http://localhost:3000/api/admin/users?page=1');
    expect(request.credentials).toBe('include');
    expect(request.headers.get('X-Institution-ID')).toBeNull();
  });

  it('serializes snake_case JSON and preserves structured API errors', async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ code: 'version_conflict' }), { status: 409, headers: { 'content-type': 'application/json' } }));
    const result = await createOpenApiTransport(fetcher).request('POST', '/api/registratura/documents/doc-1/workflow-actions', {
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ action: 'assign_department', department_id: 'dept-1', expected_version: 3, note: null }),
    });
    const request = fetcher.mock.calls[0][0] as Request;
    expect(await request.json()).toEqual({ action: 'assign_department', department_id: 'dept-1', expected_version: 3, note: null });
    expect(result.response.status).toBe(409);
    expect(result.error).toEqual({ code: 'version_conflict' });
  });
});
