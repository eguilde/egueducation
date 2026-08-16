export type Fetcher = (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>;

export interface Page<T> { items: T[]; total: number; page: number; pageSize: number }
export interface WorkflowTask {
  id: string; definition_code: string; definition_name: string; title: string; document_number: string;
  source_module: string; source_record_id?: string | null; status: string; priority: string; assigned_to: string;
  current_step: string; due_at?: string | null; started_at: string; updated_at: string; summary: string;
  linked_documents_count: number; dossier_ready: boolean; missing_relations: string[]; available_actions: string[];
}
export interface WorkflowDefinition { code: string; name: string; category: string; initial_step: string; sla_hours: number; active: boolean }
export interface WorkflowFilters { statuses: string[]; priorities: string[]; assignees: string[] }
export interface WorkflowDashboard { stats: { active_tasks: number; overdue_tasks: number; waiting_approval: number; active_definitions: number; ready_dossiers: number; blocked_dossiers: number } }
export interface CreateTaskInput { definition_code: string; title: string; document_number: string; source_module: string; source_record_id: string; priority: string; assigned_to: string; due_date?: string | null; summary: string }
export interface WorkflowApi {
  dashboard(): Promise<WorkflowDashboard>; definitions(): Promise<WorkflowDefinition[]>; filters(): Promise<WorkflowFilters>;
  tasks(query?: Record<string, string>): Promise<Page<WorkflowTask>>; create(input: CreateTaskInput): Promise<WorkflowTask>;
  transition(id: string, input: { action: string }): Promise<WorkflowTask>;
}

const toPage = <T,>(value: T[] | Partial<Page<T>>): Page<T> => Array.isArray(value)
  ? { items: value, total: value.length, page: 1, pageSize: value.length }
  : { items: value.items ?? [], total: value.total ?? 0, page: value.page ?? 1, pageSize: value.pageSize ?? 25 };

export function createWorkflowApi(fetcher: Fetcher = fetch, apiBase = '/api'): WorkflowApi {
  const request = async <T,>(path: string, init?: RequestInit) => {
    const response = await fetcher(`${apiBase}${path}`, { credentials: 'include', ...init, headers: { Accept: 'application/json', ...(init?.headers ?? {}) } });
    if (!response.ok) throw new Error(`Flux documente: ${response.status}`);
    return response.json() as Promise<T>;
  };
  return {
    dashboard: () => request('/workflow/dashboard'), definitions: () => request('/workflow/definitions'), filters: () => request('/workflow/tasks/filters'),
    async tasks(query = {}) { const params = new URLSearchParams({ page: '1', pageSize: '25', ...query }); return toPage(await request<WorkflowTask[] | Partial<Page<WorkflowTask>>>(`/workflow/tasks?${params}`)); },
    create: (input) => request('/workflow/tasks', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(input) }),
    transition: (id, input) => request(`/workflow/tasks/${encodeURIComponent(id)}/transition`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(input) })
  };
}
