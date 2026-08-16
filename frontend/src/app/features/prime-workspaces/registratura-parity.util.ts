import { RegistraturaDocument } from '../../core/api/api.types';
import { RegistraturaWorkflowActionRequest } from '../../core/api/api.types';

export const REGISTRATURA_BATCH_MIN = 1;
export const REGISTRATURA_BATCH_MAX = 20;

export function isValidRegistraturaBatchCount(value: unknown): boolean {
  const count = Number(value);
  return Number.isInteger(count) && count >= REGISTRATURA_BATCH_MIN && count <= REGISTRATURA_BATCH_MAX;
}

export function isValidCancellationReason(reason: string | null | undefined): boolean {
  return (reason?.trim().length ?? 0) >= 10;
}

export function registraturaTypeLabel(document: Pick<RegistraturaDocument, 'document_type' | 'direction'>): string {
  if (document.document_type.toUpperCase() === 'MULTIPLU') return 'MULTIPLU';
  return document.direction === 'iesire' ? 'IEȘIRE' : 'INTRARE';
}

export function isTerminalRegistraturaStatus(status: string): boolean {
  return ['anulat', 'cancelled', 'canceled', 'finalizat', 'finalized'].includes(status.toLowerCase());
}

export function canonicalRegistraturaWorkflowStatus(status: string): string {
  const normalized = status.trim().toUpperCase();
  const aliases: Record<string, string> = {
    REGISTERED: 'INCOMING', DRAFT: 'INCOMING', ALLOCATED_DEPARTMENT: 'ALOCAT_COMPARTIMENT',
    IN_WORKFLOW: 'IN_LUCRU', FINALIZED: 'FINALIZAT', CANCELLED: 'ANULAT', CANCELED: 'ANULAT',
  };
  return aliases[normalized] ?? normalized;
}

export function permittedRegistraturaWorkflowActions(status: string): RegistraturaWorkflowActionRequest['action'][] {
  const actions: Record<string, RegistraturaWorkflowActionRequest['action'][]> = {
    INCOMING: ['assign_department'],
    ALOCAT_COMPARTIMENT: ['assign_user', 'claim'],
    IN_LUCRU: ['assign_user', 'send_for_approval'],
    FLUX_APROBARE: ['approve', 'reject'],
  };
  return actions[canonicalRegistraturaWorkflowStatus(status)] ?? [];
}

export function registraturaWorkflowPayload(action: RegistraturaWorkflowActionRequest['action'], expectedVersion: number | null, note: string, departmentId?: string | null, userId?: string | null): RegistraturaWorkflowActionRequest {
  const payload: RegistraturaWorkflowActionRequest = { action, expected_version: expectedVersion, note: note.trim() || null };
  if (action === 'assign_department') payload.department_id = departmentId ?? null;
  if (action === 'assign_user') payload.user_id = userId ?? null;
  return payload;
}
