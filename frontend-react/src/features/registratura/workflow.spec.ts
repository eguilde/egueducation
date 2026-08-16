import { describe, expect, it } from 'vitest';
import { canonicalStatus, permittedActions } from './workflow';
describe('Registratură workflow rules', () => { it('normalizes legacy statuses and gates actions', () => { expect(canonicalStatus('allocated_department')).toBe('ALOCAT_COMPARTIMENT'); expect(permittedActions('ALOCAT_COMPARTIMENT')).toEqual(['assign_user', 'claim']); expect(permittedActions('finalized')).toEqual([]); }); });
