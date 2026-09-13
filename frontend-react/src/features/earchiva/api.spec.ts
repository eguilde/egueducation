import { describe, expect, it, vi } from 'vitest';
import { createArchiveApi } from './api';

const json = (value: unknown, status = 200) => new Response(JSON.stringify(value), { status, headers: { 'content-type': 'application/json' } });
const requestAt = (fetcher: ReturnType<typeof vi.fn>, index = 0) => fetcher.mock.calls[index][0] as Request;

describe('eArhivă API adapter', () => {
  it('uses tenant-derived multipart upload without institution or storage fields', async () => {
    const fetcher = vi.fn().mockImplementation(() => Promise.resolve(json({ items: [] })));
    const api = createArchiveApi(fetcher);
    await api.documents({ q: 'catalog' });
    await api.upload({ file: new File(['x'], 'a.pdf', { type: 'application/pdf' }), title: 'Catalog', source_kind: 'scan', taxonomy_parent: 'FOND' });
    expect(requestAt(fetcher).url).toContain('/api/earchiva/documents?');
    const request = requestAt(fetcher, 1);
    expect(new URL(request.url).pathname).toBe('/api/earchiva/documents');
    expect(request.credentials).toBe('include');
    expect(request.headers.get('X-Institution-ID')).toBeNull();
    expect(request.headers.get('content-type')).toMatch(/^multipart\/form-data;\s*boundary=/i);
  });

  it('uses only the eArhivă administration contract paths', async () => {
    const fetcher = vi.fn().mockImplementation(() => Promise.resolve(json({ items: [] })));
    const api = createArchiveApi(fetcher);
    await api.adminHealth(); await api.adminStats(); await api.adminJobs(); await api.retryJob('job/one');
    expect(fetcher.mock.calls.map((call) => new URL((call[0] as Request).url).pathname)).toEqual([
      '/api/earchiva/admin/health', '/api/earchiva/admin/stats', '/api/earchiva/admin/jobs', '/api/earchiva/admin/jobs/job%2Fone/retry'
    ]);
    expect(requestAt(fetcher, 3).method).toBe('POST');
  });

  it('uses the scoped custody-recovery contracts without client tenant or storage identity', async () => {
    const recoveryResult = { operation_id: '33333333-3333-4333-8333-333333333333', intent_id: '11111111-1111-4111-8111-111111111111', portfolio_id: '22222222-2222-4222-8222-222222222222', status: 'queued', disposition: 'teacher_access', reason: 'recuperare documentată', title: 'Plan', original_file_name: 'plan.pdf', created_at: '2026-09-12T08:00:00Z', updated_at: '2026-09-12T08:00:00Z' };
    const fetcher = vi.fn()
      .mockResolvedValueOnce(json({ items: [], total: 0, page: 1, pageSize: 20 }))
      .mockResolvedValueOnce(json(recoveryResult, 202))
      .mockResolvedValueOnce(json({ ...recoveryResult, status: 'committed' }));
    const api = createArchiveApi(fetcher);
    await api.portfolioCustodyRecoveries({ page: 2, pageSize: 20, status: 'stored', disposition: 'teacher_access', title: 'plan', original_file_name: 'plan.pdf', sort: 'created_at', direction: 'desc' });
    await api.reconcilePortfolioCustody('intent/one', { reason: 'recuperare documentată', disposition: 'teacher_access', title: 'Plan', original_file_name: 'plan.pdf' });
    await api.portfolioCustodyRecovery('intent/one', 'op/one');

    const list = new URL(requestAt(fetcher).url);
    expect(list.pathname).toBe('/api/earchiva/admin/portfolio-custody-intents');
    expect(Object.fromEntries(list.searchParams)).toEqual({ page: '2', pageSize: '20', status: 'stored', disposition: 'teacher_access', title: 'plan', original_file_name: 'plan.pdf', sort: 'created_at', direction: 'desc' });
    expect(list.searchParams.has('institution_id')).toBe(false);
    const command = requestAt(fetcher, 1);
    expect(new URL(command.url).pathname).toBe('/api/earchiva/admin/portfolio-custody-intents/intent%2Fone/reconcile');
    expect(await command.json()).toEqual({ reason: 'recuperare documentată', disposition: 'teacher_access', title: 'Plan', original_file_name: 'plan.pdf' });
    expect(new URL(requestAt(fetcher, 2).url).pathname).toBe('/api/earchiva/admin/portfolio-custody-intents/intent%2Fone/recovery-operations/op%2Fone');
    expect(command.headers.get('X-Institution-ID')).toBeNull();
  });

  it('downloads original content only through the protected document route', async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(new Blob(['pdf'], { type: 'application/pdf' }), { status: 200, headers: { 'content-type': 'application/pdf' } }));
    const api = createArchiveApi(fetcher);
    await api.download('doc/one');
    const request = requestAt(fetcher);
    expect(new URL(request.url).pathname).toBe('/api/earchiva/documents/doc%2Fone/content');
    expect(request.credentials).toBe('include');
    expect(request.headers.get('accept')).toBe('application/pdf');
  });

  it('sends revision-bound structured classification decisions only to review routes', async () => {
    const fetcher = vi.fn().mockImplementation(() => Promise.resolve(json({})));
    const api = createArchiveApi(fetcher);
    await api.classificationReviews({ state: 'needs_review', page: '2', pageSize: '10' });
    await api.approveClassificationReview('review/one', { revision: 3, note: '' });
    await api.correctClassificationReview('review/two', { revision: 4, note: 'corectat', classification: { category: 'Școlar', fond: 'Școala', series: 'Cataloage', document_type: 'Catalog', document_date: '', document_number: '' } });
    expect(requestAt(fetcher).url).toBe('http://localhost:3000/api/earchiva/classification-reviews?page=2&pageSize=10&state=needs_review');
    expect(new URL(requestAt(fetcher, 1).url).pathname).toBe('/api/earchiva/classification-reviews/review%2Fone/approve');
    expect(await requestAt(fetcher, 1).json()).toEqual({ revision: 3, note: '' });
    expect(new URL(requestAt(fetcher, 2).url).pathname).toBe('/api/earchiva/classification-reviews/review%2Ftwo/correct');
    expect(await requestAt(fetcher, 2).json()).toEqual({ revision: 4, note: 'corectat', classification: { category: 'Școlar', fond: 'Școala', series: 'Cataloage', document_type: 'Catalog', document_date: '', document_number: '' } });
  });
});
