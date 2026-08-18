import type { RegistryDocument, WorkflowAction } from './types';

const aliases: Record<string, string> = { REGISTERED: 'INCOMING', DRAFT: 'INCOMING', ALLOCATED_DEPARTMENT: 'ALOCAT_COMPARTIMENT', IN_WORKFLOW: 'IN_LUCRU', FINALIZED: 'FINALIZAT', CANCELLED: 'ANULAT', CANCELED: 'ANULAT' };
const actions: Record<string, WorkflowAction[]> = { INCOMING: ['assign_department'], ALOCAT_COMPARTIMENT: ['assign_user', 'claim'], IN_LUCRU: ['assign_user', 'send_for_approval'], FLUX_APROBARE: ['approve', 'reject'] };
const statusLabels: Record<string, string> = {
  INCOMING: 'Înregistrat',
  ALOCAT_COMPARTIMENT: 'Alocat compartiment',
  IN_LUCRU: 'În lucru',
  FLUX_APROBARE: 'În aprobare',
  FINALIZAT: 'Finalizat',
  ANULAT: 'Anulat',
};
export const canonicalStatus = (status: string) => aliases[status.trim().toUpperCase()] ?? status.trim().toUpperCase();
export const statusLabel = (status: string) => {
  const canonical = canonicalStatus(status);
  return statusLabels[canonical] ?? canonical.toLocaleLowerCase('ro-RO').replaceAll('_', ' ').replace(/^./, (letter) => letter.toLocaleUpperCase('ro-RO'));
};
export const permittedActions = (status: string): WorkflowAction[] => actions[canonicalStatus(status)] ?? [];
export const isTerminalStatus = (status: string) => ['ANULAT', 'FINALIZAT'].includes(canonicalStatus(status));
export const directionLabel = (document: Pick<RegistryDocument, 'document_type' | 'direction'>) => document.document_type.toUpperCase() === 'MULTIPLU' ? 'Multiplu' : document.direction === 'iesire' ? 'Ieșire' : 'Intrare';
export const calendarDateLabel = (value?: string | null) => {
  const match = value?.trim().match(/^(\d{4})-(\d{2})-(\d{2})/);
  return match ? `${match[3]}.${match[2]}.${match[1]}` : '—';
};
