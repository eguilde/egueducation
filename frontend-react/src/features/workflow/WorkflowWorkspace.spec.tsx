import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { PrimeReactProvider } from '@primereact/core/config';
import { primeTheme } from '../../components/ThemeMenu';
import { WorkflowWorkspace } from './WorkflowWorkspace';
import type { WorkflowApi } from './api';

const document = { id: 'doc-1', registry_number: 'REG-1', entry_at: '2026-09-08T10:00:00Z', subject: 'Cerere avizare', correspondent: 'Inspectorat', department_name: 'Secretariat', document_type: 'DOCUMENT', status: 'FLUX_APROBARE', workflow_version: 7, rejection_count: 0, target_approver_name: 'Director' };
function apiFor(overrides: Partial<WorkflowApi> = {}): WorkflowApi {
  return {
    queue: vi.fn().mockResolvedValue({ items: [document], total: 42, page: 1, pageSize: 20 }),
    mapa: vi.fn().mockResolvedValue({ items: [document], total: 1, page: 1, pageSize: 20 }),
    pipeline: vi.fn().mockResolvedValue({ items: [document], total: 1, page: 1, pageSize: 20 }),
    pipelineStats: vi.fn().mockResolvedValue([{ status: 'FLUX_APROBARE', count: 1 }, { status: 'FINALIZAT', count: 2 }]),
    assignees: vi.fn().mockResolvedValue({ departments: [{ id: 'dept-1', name: 'Secretariat' }], users: [{ id: 'user-2', name: 'Director' }] }),
    transition: vi.fn().mockResolvedValue(undefined), ...overrides,
  };
}
const renderWorkspace = (api: WorkflowApi, props = {}) => render(<PrimeReactProvider {...primeTheme}><WorkflowWorkspace api={api} {...props} /></PrimeReactProvider>);

describe('Flux documente Costești views', () => {
  it('renders the server-paged queue with column filters and date-only display', async () => {
    const api = apiFor(); renderWorkspace(api);
    await screen.findByText('Cerere avizare');
    expect(screen.getByText('08.09.2026')).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText('Filtru Nr. document'), { target: { value: 'REG-1' } });
    await waitFor(() => expect(api.queue).toHaveBeenLastCalledWith(expect.objectContaining({ nr_doc: 'REG-1', page: 1 })));
    fireEvent.click(screen.getByRole('button', { name: 'Sortează după Nr. document' }));
    await waitFor(() => expect(api.queue).toHaveBeenLastCalledWith(expect.objectContaining({ sort: 'registry_number', direction: 'asc' })));
    expect(screen.getByLabelText('Paginare Coada mea')).toHaveTextContent('1–20 din 42');
  });

  it('exposes Mapă and Pipeline only to the matching permissions and keeps their server filters distinct', async () => {
    const api = apiFor(); renderWorkspace(api, { canTransition: true, canManage: true });
    await screen.findByRole('tab', { name: 'Mapă semnături' });
    fireEvent.click(screen.getByRole('tab', { name: 'Mapă semnături' }));
    fireEvent.click(await screen.findByRole('button', { name: 'De la colegi' }));
    await waitFor(() => expect(api.mapa).toHaveBeenLastCalledWith(expect.objectContaining({ mapa_filter: 'peers' })));
    fireEvent.click(screen.getByRole('tab', { name: 'Evidență completă' }));
    await waitFor(() => expect(api.pipelineStats).toHaveBeenCalled());
    fireEvent.click(await screen.findByRole('button', { name: 'Filtrează status În aprobare' }));
    await waitFor(() => expect(api.pipeline).toHaveBeenLastCalledWith(expect.objectContaining({ status: 'FLUX_APROBARE' })));
  });

  it('applies a permitted action through the backend contract and enforces rejection rationale in UI', async () => {
    const api = apiFor(); renderWorkspace(api, { canTransition: true });
    await screen.findByText('Cerere avizare');
    fireEvent.click(screen.getByRole('button', { name: 'Deschide REG-1' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Respinge' }));
    const apply = screen.getAllByRole('button', { name: 'Respinge' }).at(-1)!;
    expect(apply).toBeDisabled();
    fireEvent.change(screen.getByLabelText('Motiv respingere'), { target: { value: 'Motiv suficient de clar' } });
    fireEvent.click(apply);
    await waitFor(() => expect(api.transition).toHaveBeenCalledWith('doc-1', { action: 'reject', expected_version: 7, note: 'Motiv suficient de clar' }));
  });

  it('keeps a reader in a consultation-only queue', async () => {
    const api = apiFor(); renderWorkspace(api);
    await screen.findByText('Cerere avizare');
    expect(screen.queryByRole('tab', { name: 'Mapă semnături' })).not.toBeInTheDocument();
    expect(screen.queryByRole('tab', { name: 'Evidență completă' })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Deschide REG-1' }));
    expect(await screen.findByText(/Nu aveți dreptul de a aplica tranziții/)).toBeInTheDocument();
  });
});
