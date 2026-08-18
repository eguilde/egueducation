import { describe, expect, it } from 'vitest';
import { calendarDateLabel, canonicalStatus, directionLabel, permittedActions, statusLabel } from './workflow';
describe('Registratură workflow rules', () => {
  it('normalizes legacy statuses and gates actions', () => {
    expect(canonicalStatus('allocated_department')).toBe('ALOCAT_COMPARTIMENT');
    expect(statusLabel('allocated_department')).toBe('Alocat compartiment');
    expect(permittedActions('ALOCAT_COMPARTIMENT')).toEqual(['assign_user', 'claim']);
    expect(permittedActions('finalized')).toEqual([]);
  });

  it('uses readable type and calendar-only date labels in the table', () => {
    expect(directionLabel({ document_type: 'DOCUMENT', direction: 'intrare' })).toBe('Intrare');
    expect(directionLabel({ document_type: 'MULTIPLU', direction: 'intrare' })).toBe('Multiplu');
    expect(calendarDateLabel('2026-08-18T21:30:25Z')).toBe('18.08.2026');
    expect(calendarDateLabel(null)).toBe('—');
  });
});
