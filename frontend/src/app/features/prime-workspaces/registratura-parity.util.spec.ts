import {
  isTerminalRegistraturaStatus,
  canonicalRegistraturaWorkflowStatus,
  permittedRegistraturaWorkflowActions,
  registraturaWorkflowPayload,
  isValidCancellationReason,
  isValidRegistraturaBatchCount,
  registraturaTypeLabel,
} from './registratura-parity.util';

describe('Registratură parity rules', () => {
  it('accepts only Costești-compatible batch counts', () => {
    expect(isValidRegistraturaBatchCount(1)).toBe(true);
    expect(isValidRegistraturaBatchCount(20)).toBe(true);
    expect(isValidRegistraturaBatchCount(0)).toBe(false);
    expect(isValidRegistraturaBatchCount(21)).toBe(false);
    expect(isValidRegistraturaBatchCount(1.5)).toBe(false);
  });

  it('requires an explicit cancellation reason', () => {
    expect(isValidCancellationReason('scurt')).toBe(false);
    expect(isValidCancellationReason('Înregistrare duplicată')).toBe(true);
  });

  it('uses Romanian Costești type labels and locks terminal states', () => {
    expect(registraturaTypeLabel({ document_type: 'MULTIPLU', direction: 'intrare' })).toBe('MULTIPLU');
    expect(registraturaTypeLabel({ document_type: 'cerere', direction: 'iesire' })).toBe('IEȘIRE');
    expect(isTerminalRegistraturaStatus('ANULAT')).toBe(true);
    expect(isTerminalRegistraturaStatus('IN_LUCRU')).toBe(false);
  });

  it('maps legacy values to canonical document workflow actions', () => {
    expect(canonicalRegistraturaWorkflowStatus('in_workflow')).toBe('IN_LUCRU');
    expect(permittedRegistraturaWorkflowActions('INCOMING')).toEqual(['assign_department']);
    expect(permittedRegistraturaWorkflowActions('FLUX_APROBARE')).toEqual(['approve', 'reject']);
    expect(permittedRegistraturaWorkflowActions('FINALIZAT')).toEqual([]);
  });

  it('never adds a user identifier to a claim payload', () => {
    expect(registraturaWorkflowPayload('claim', 4, '', 'department-id', 'forged-user')).toEqual({ action: 'claim', expected_version: 4, note: null });
    expect(registraturaWorkflowPayload('assign_user', 4, 'alocare', null, 'user-id')).toEqual({ action: 'assign_user', expected_version: 4, note: 'alocare', user_id: 'user-id' });
  });
});
