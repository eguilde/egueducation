import { describe, expect, it, vi } from 'vitest';
import { createWorkflowApi } from './api';

describe('Workflow API adapter', () => {
  it('uses authenticated fetch contracts, filter paging, creation and transition routes', async () => {
    const fetcher = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ items: [], total: 0, page: 2, pageSize: 25 }) });
    const api = createWorkflowApi(fetcher);
    await api.tasks({ page: '2', status: 'new' });
    await api.create({ definition_code: 'approval', title: 'Test', document_number: '', source_module: 'education', source_record_id: '', priority: 'medium', assigned_to: '', summary: '' });
    await api.transition('task/a', { action: 'submit' });
    expect(fetcher.mock.calls[0][0]).toContain('/api/workflow/tasks?');
    expect(fetcher.mock.calls[0][0]).toContain('status=new');
    expect(fetcher.mock.calls[1][0]).toBe('/api/workflow/tasks');
    expect(fetcher.mock.calls[1][1].method).toBe('POST');
    expect(fetcher.mock.calls[2][0]).toBe('/api/workflow/tasks/task%2Fa/transition');
  });
});
