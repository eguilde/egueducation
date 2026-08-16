import { describe, expect, it, vi } from 'vitest';
import { createArchiveApi } from './api';

describe('eArhivă API adapter', () => {
  it('uses multipart upload and does not attach a client-selected institution header', async () => {
    const fetcher = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ items: [] }) });
    const api = createArchiveApi(fetcher);
    await api.documents({ q: 'catalog' });
    await api.upload({ file: new File(['x'], 'a.pdf', { type: 'application/pdf' }), title: 'Catalog', source_kind: 'scan' });
    expect(fetcher.mock.calls[0][0]).toContain('/api/earchiva/documents?');
    expect(fetcher.mock.calls[1][0]).toBe('/api/earchiva/documents');
    expect(fetcher.mock.calls[1][1].body).toBeInstanceOf(FormData);
    expect(fetcher.mock.calls[1][1].headers['X-Institution-ID']).toBeUndefined();
  });
});
