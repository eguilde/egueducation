import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { PrimeReactProvider } from '@primereact/core/config';
import { describe, expect, it, vi } from 'vitest';
import { primeTheme } from '../../components/ThemeMenu';
import { PortfolioCustodyRecoveryPanel } from './PortfolioCustodyRecoveryPanel';
import type { ArchiveApi, PortfolioCustodyRecoveryOperation } from './api';

const stored: PortfolioCustodyRecoveryOperation = {
  intent_id: '11111111-1111-4111-8111-111111111111',
  portfolio_id: '22222222-2222-4222-8222-222222222222',
  status: 'stored',
  created_at: '2026-09-12T08:00:00Z',
  updated_at: '2026-09-12T08:00:00Z',
};
const operation: PortfolioCustodyRecoveryOperation = {
  ...stored,
  operation_id: '33333333-3333-4333-8333-333333333333',
  disposition: 'teacher_access',
  status: 'queued',
  title: 'Planificare anuală',
  original_file_name: 'planificare.pdf',
  reason: 'Recuperare controlată pentru test',
};

const api = (): ArchiveApi => ({
  documents: vi.fn(), document: vi.fn(), versions: vi.fn(), download: vi.fn(), taxonomy: vi.fn(), upload: vi.fn(),
  adminHealth: vi.fn(), adminStats: vi.fn(), adminJobs: vi.fn(), retryJob: vi.fn(),
  portfolioCustodyRecoveries: vi.fn().mockResolvedValue({ items: [stored], total: 1, page: 1, pageSize: 20 }),
  reconcilePortfolioCustody: vi.fn().mockResolvedValue(operation),
  portfolioCustodyRecovery: vi.fn().mockResolvedValue({ ...operation, status: 'committed' }),
  classificationReviews: vi.fn(), approveClassificationReview: vi.fn(), correctClassificationReview: vi.fn(),
});

const mount = (transport: ArchiveApi) => render(<PrimeReactProvider {...primeTheme}><PortfolioCustodyRecoveryPanel api={transport} /></PrimeReactProvider>);

describe('PortfolioCustodyRecoveryPanel', () => {
  it('uses server-side query state for header filtering, stable sorting and paging', async () => {
    const transport = api();
    mount(transport);
    await screen.findByText(stored.intent_id);
    expect(transport.portfolioCustodyRecoveries).toHaveBeenCalledWith(expect.objectContaining({ page: 1, pageSize: 20, sort: 'created_at', direction: 'desc' }));

    fireEvent.change(screen.getByRole('textbox', { name: 'Filtru titlu recuperare' }), { target: { value: 'planificare' } });
    await waitFor(() => expect(transport.portfolioCustodyRecoveries).toHaveBeenLastCalledWith(expect.objectContaining({ title: 'planificare', page: 1 })));
    fireEvent.change(screen.getByLabelText('Creat de la'), { target: { value: '2026-09-01' } });
    await waitFor(() => expect(transport.portfolioCustodyRecoveries).toHaveBeenLastCalledWith(expect.objectContaining({ created_from: '2026-09-01T00:00:00.000Z', page: 1 })));
    fireEvent.click(screen.getByRole('button', { name: 'Sortează după Titlu' }));
    await waitFor(() => expect(transport.portfolioCustodyRecoveries).toHaveBeenLastCalledWith(expect.objectContaining({ sort: 'title', direction: 'asc', page: 1 })));
  });

  it('treats HTTP 202 as queued, polls the durable operation and only then reports completion', async () => {
    const transport = api();
    mount(transport);
    await screen.findByText(stored.intent_id);
    fireEvent.click(screen.getByRole('button', { name: `Recuperează intent ${stored.intent_id}` }));
    const dialog = screen.getByRole('dialog', { name: 'Recuperare verificată a custodiei' });
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Titlul inițial al încărcării' }), { target: { value: 'Planificare anuală' } });
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Numele original al fișierului' }), { target: { value: 'planificare.pdf' } });
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Motiv recuperare custodie' }), { target: { value: 'Recuperare controlată pentru test' } });
    fireEvent.click(within(dialog).getByRole('button', { name: 'Trimite pentru recuperare' }));

    await waitFor(() => expect(transport.reconcilePortfolioCustody).toHaveBeenCalledWith(stored.intent_id, {
      reason: 'Recuperare controlată pentru test', disposition: 'teacher_access', title: 'Planificare anuală', original_file_name: 'planificare.pdf', document_date: undefined,
    }));
    await screen.findByText(/Cererea a fost acceptată în coada durabilă/);
    await waitFor(() => expect(transport.portfolioCustodyRecovery).toHaveBeenCalledWith(stored.intent_id, operation.operation_id));
    await screen.findByText(/Recuperarea a fost finalizată și verificată de server/);
  });

  it('cancels operation polling on unmount', async () => {
    let resolveStatus: ((value: PortfolioCustodyRecoveryOperation) => void) | undefined;
    const transport = api();
    (transport.portfolioCustodyRecovery as ReturnType<typeof vi.fn>).mockImplementation(() => new Promise((resolve) => { resolveStatus = resolve; }));
    const view = mount(transport);
    await screen.findByText(stored.intent_id);
    fireEvent.click(screen.getByRole('button', { name: `Recuperează intent ${stored.intent_id}` }));
    const dialog = screen.getByRole('dialog', { name: 'Recuperare verificată a custodiei' });
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Titlul inițial al încărcării' }), { target: { value: 'Planificare anuală' } });
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Numele original al fișierului' }), { target: { value: 'planificare.pdf' } });
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Motiv recuperare custodie' }), { target: { value: 'Recuperare controlată pentru test' } });
    fireEvent.click(within(dialog).getByRole('button', { name: 'Trimite pentru recuperare' }));
    await waitFor(() => expect(transport.portfolioCustodyRecovery).toHaveBeenCalled());
    view.unmount();
    resolveStatus?.({ ...operation, status: 'committed' });
    await Promise.resolve();
    expect(transport.portfolioCustodyRecovery).toHaveBeenCalledTimes(1);
  });

  it('loads the durable operation before showing its documented reason and safe error code', async () => {
    const transport = api();
    (transport.portfolioCustodyRecoveries as ReturnType<typeof vi.fn>).mockResolvedValue({ items: [{ ...operation, status: 'blocked', last_error_code: 'recovery_metadata_mismatch' }], total: 1, page: 1, pageSize: 20 });
    (transport.portfolioCustodyRecovery as ReturnType<typeof vi.fn>).mockResolvedValue({ ...operation, status: 'blocked', reason: 'Justificare persistentă verificată', last_error_code: 'recovery_metadata_mismatch' });
    mount(transport);
    await screen.findByText(operation.intent_id);
    fireEvent.click(screen.getByRole('button', { name: `Urmărește operația ${operation.operation_id}` }));
    const dialog = await screen.findByRole('dialog', { name: 'Operație de recuperare' });
    await waitFor(() => expect(transport.portfolioCustodyRecovery).toHaveBeenCalledWith(operation.intent_id, operation.operation_id));
    expect(within(dialog).getByText('Justificare persistentă verificată')).toBeInTheDocument();
    expect(within(dialog).getByText('recovery_metadata_mismatch')).toBeInTheDocument();
  });
});
