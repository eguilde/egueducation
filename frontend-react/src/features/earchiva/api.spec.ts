import { describe, expect, it, vi } from 'vitest';
import { createArchiveApi } from './api';

describe('eArhivă API adapter', () => {
  it('uses tenant-derived multipart upload without institution or storage fields', async () => {
    const fetcher = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ items: [] }) });
    const api = createArchiveApi(fetcher);
    await api.documents({ q: 'catalog' });
    await api.upload({ file: new File(['x'], 'a.pdf', { type: 'application/pdf' }), title: 'Catalog', source_kind: 'scan', taxonomy_parent: 'FOND' });
    expect(fetcher.mock.calls[0][0]).toContain('/api/earchiva/documents?');
    expect(fetcher.mock.calls[1][0]).toBe('/api/earchiva/documents');
    expect(fetcher.mock.calls[1][1].body).toBeInstanceOf(FormData);
    expect(fetcher.mock.calls[1][1].headers['X-Institution-ID']).toBeUndefined();
    const body = fetcher.mock.calls[1][1].body as FormData;
    expect(body.get('institution_id')).toBeNull();
    expect(body.get('tenant_id')).toBeNull();
    expect(body.get('object_key')).toBeNull();
    expect(body.get('taxonomy_parent_code')).toBe('FOND');
  });

  it('uses only the eArhivă administration contract paths', async () => {
    const fetcher = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ items: [] }) });
    const api = createArchiveApi(fetcher);
    await api.adminHealth(); await api.adminStats(); await api.adminJobs(); await api.retryJob('job/one');
    expect(fetcher.mock.calls.map((call) => call[0])).toEqual([
      '/api/earchiva/admin/health', '/api/earchiva/admin/stats', '/api/earchiva/admin/jobs', '/api/earchiva/admin/jobs/job%2Fone/retry'
    ]);
    expect(fetcher.mock.calls[3][1].method).toBe('POST');
  });

  it('downloads original content only through the protected document route', async () => {
    const fetcher = vi.fn().mockResolvedValue({ ok: true, blob: async () => new Blob(['pdf'], { type: 'application/pdf' }) });
    const api = createArchiveApi(fetcher);
    await api.download('doc/one');
    expect(fetcher).toHaveBeenCalledWith('/api/earchiva/documents/doc%2Fone/content', expect.objectContaining({ credentials: 'include', headers: { Accept: 'application/pdf' } }));
  });

  it('sends revision-bound structured classification decisions only to review routes', async () => {
    const fetcher = vi.fn().mockResolvedValue({ ok: true, json: async () => ({}) });
    const api = createArchiveApi(fetcher);
    await api.classificationReviews({ state: 'needs_review', page: '2', pageSize: '10' });
    await api.approveClassificationReview('review/one', { revision: 3, note: '' });
    await api.correctClassificationReview('review/two', { revision: 4, note: 'corectat', classification: { category: 'Școlar', fond: 'Școala', series: 'Cataloage', document_type: 'Catalog', document_date: '', document_number: '' } });
    expect(fetcher.mock.calls[0][0]).toBe('/api/earchiva/classification-reviews?page=2&pageSize=10&state=needs_review');
    expect(fetcher.mock.calls[1][0]).toBe('/api/earchiva/classification-reviews/review%2Fone/approve');
    expect(JSON.parse(fetcher.mock.calls[1][1].body)).toEqual({ revision: 3, note: '' });
    expect(fetcher.mock.calls[2][0]).toBe('/api/earchiva/classification-reviews/review%2Ftwo/correct');
    expect(JSON.parse(fetcher.mock.calls[2][1].body)).toEqual({ revision: 4, note: 'corectat', classification: { category: 'Școlar', fond: 'Școala', series: 'Cataloage', document_type: 'Catalog', document_date: '', document_number: '' } });
  });
});
