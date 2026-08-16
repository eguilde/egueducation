import type { RegistryDocument, WorkflowAction } from './types';

const aliases: Record<string, string> = { REGISTERED: 'INCOMING', DRAFT: 'INCOMING', ALLOCATED_DEPARTMENT: 'ALOCAT_COMPARTIMENT', IN_WORKFLOW: 'IN_LUCRU', FINALIZED: 'FINALIZAT', CANCELLED: 'ANULAT', CANCELED: 'ANULAT' };
const actions: Record<string, WorkflowAction[]> = { INCOMING: ['assign_department'], ALOCAT_COMPARTIMENT: ['assign_user', 'claim'], IN_LUCRU: ['assign_user', 'send_for_approval'], FLUX_APROBARE: ['approve', 'reject'] };
export const canonicalStatus = (status: string) => aliases[status.trim().toUpperCase()] ?? status.trim().toUpperCase();
export const permittedActions = (status: string): WorkflowAction[] => actions[canonicalStatus(status)] ?? [];
export const isTerminalStatus = (status: string) => ['ANULAT', 'FINALIZAT'].includes(canonicalStatus(status));
export const directionLabel = (document: Pick<RegistryDocument, 'document_type' | 'direction'>) => document.document_type.toUpperCase() === 'MULTIPLU' ? 'MULTIPLU' : document.direction === 'iesire' ? 'IEȘIRE' : 'INTRARE';
