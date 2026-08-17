import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { PrimeReactProvider } from '@primereact/core/config';
import { describe, expect, it, vi } from 'vitest';
import { primeTheme } from '../../components/ThemeMenu';
import { ArchiveWorkspace, archiveUploadOutcome } from './ArchiveWorkspace';
import type { ArchiveApi } from './api';

const api = (): ArchiveApi => ({
  documents: vi.fn().mockResolvedValue({ items: [{ id: 'doc-1', title: 'Catalog', original_file_name: 'catalog.pdf', mime_type: 'application/pdf', source_kind: 'scan', source_system: '', external_reference: '', status: 'queued', current_version_no: 1, received_at: '2026-01-01', created_at: '2026-01-01', updated_at: '2026-01-01' }], total: 1, page: 1, pageSize: 25 }),
  taxonomy: vi.fn().mockResolvedValue([]), document: vi.fn(), versions: vi.fn(), download: vi.fn(), upload: vi.fn(), adminHealth: vi.fn(), adminStats: vi.fn(), adminJobs: vi.fn(), retryJob: vi.fn(), classificationReviews: vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 25 }), approveClassificationReview: vi.fn(), correctClassificationReview: vi.fn(),
});

describe('ArchiveWorkspace authorization', () => {
  it('never presents a failed OCR job as processed', () => {
    expect(archiveUploadOutcome('failed')).toBe('failed');
    expect(archiveUploadOutcome('ready')).toBe('processed');
    expect(archiveUploadOutcome('processing')).toBeUndefined();
  });
  it('does not render upload or administration controls for a read-only archivist', async () => {
    const transport = api();
    render(<PrimeReactProvider {...primeTheme}><ArchiveWorkspace api={transport} canManage={false} canReview={false} /></PrimeReactProvider>);
    await screen.findByText('Catalog');
    expect(screen.queryByRole('button', { name: /Încarcă PDF-uri/ })).not.toBeInTheDocument();
    expect(screen.queryByText('Administrare eArhivă')).not.toBeInTheDocument();
    expect(screen.queryByText('Revizuire clasificări OCR')).not.toBeInTheDocument();
    expect(transport.classificationReviews).not.toHaveBeenCalled();
  });

  it('loads tenant-scoped documents without client tenant parameters', async () => {
    const transport = api();
    render(<PrimeReactProvider {...primeTheme}><ArchiveWorkspace api={transport} canManage /></PrimeReactProvider>);
    await waitFor(() => expect(transport.documents).toHaveBeenCalled());
    const calls = (transport.documents as ReturnType<typeof vi.fn>).mock.calls;
    expect(calls[0][0]).not.toHaveProperty('institution_id');
    expect(calls[0][0]).not.toHaveProperty('tenant_id');
  });

  it('approves an OCR suggestion with its current server revision', async () => {
    const transport = api();
    (transport.classificationReviews as ReturnType<typeof vi.fn>).mockResolvedValue({ items: [{ id: 'review-1', document_id: 'doc-1', version_id: 'version-1', state: 'pending_review', revision: 7, suggestion_confidence: 0.91, suggestion_source: 'rules-v1', requires_human_review: true, generated_at: '2026-01-01T00:00:00Z', suggestion: { category: { value: 'Școlar', source: 'ocr', confidence: 1 }, fond: { value: 'Școala', source: 'ocr', confidence: 1 }, series: { value: 'Cataloage', source: 'ocr', confidence: 1 }, document_type: { value: 'Catalog', source: 'ocr', confidence: 1 }, document_date: { value: '2026-01-01', source: 'ocr', confidence: 1 }, document_number: { value: '1', source: 'ocr', confidence: 1 } } }], total: 1, page: 1, pageSize: 25 });
    (transport.approveClassificationReview as ReturnType<typeof vi.fn>).mockResolvedValue({});
    render(<PrimeReactProvider {...primeTheme}><ArchiveWorkspace api={transport} canReview /></PrimeReactProvider>);
    await screen.findByText('Revizuire clasificări OCR');
    fireEvent.click(screen.getByRole('button', { name: 'Aprobă propunerea' }));
    await waitFor(() => expect(transport.approveClassificationReview).toHaveBeenCalledWith('review-1', { revision: 7, note: '' }));
  });
});
