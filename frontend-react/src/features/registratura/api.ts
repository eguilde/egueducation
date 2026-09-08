import type { AdminUser, BatchCreateInput, CreateDocumentInput, DocumentAttachment, DocumentFilterOptions, DocumentFilters, DocumentLookup, DocumentVersion, LinkedDocument, OrganizationChartNode, Page, Party, Registry, RegistryAdminRecord, RegistryDocument, UserAssignment, WorkflowAction, WorkflowAssignees, WorkflowHistoryEntry } from './types';
import { createContractClient } from '../../api/client';
import type { paths } from '../../api/generated';
import { validateGetApiRegistraturaDocumentsResponse, validatePostApiRegistraturaDocumentsResponse } from '../../api/runtime-validators';

export interface RegistraturaApi {
  registries(): Promise<Registry[]>; filters(): Promise<DocumentFilterOptions>; documents(input: { registryId: number; page: number; pageSize: number; filters: DocumentFilters; sort?: string; direction?: 'asc' | 'desc' }): Promise<Page<RegistryDocument>>;
  document(id: string): Promise<RegistryDocument>; versions(id: string): Promise<DocumentVersion[]>; attachments(id: string): Promise<DocumentAttachment[]>; workflowHistory(id: string): Promise<WorkflowHistoryEntry[]>; assignees(): Promise<WorkflowAssignees>;
  create(input: CreateDocumentInput): Promise<RegistryDocument>; update(id: string, input: CreateDocumentInput & { change_notes?: string; expected_workflow_version?: number | null }): Promise<RegistryDocument>; createVersion(id: string, input: Pick<CreateDocumentInput, 'subject' | 'status' | 'assigned_to' | 'confidentiality' | 'summary' | 'due_date'> & { change_notes: string }): Promise<DocumentVersion>; cancel(id: string, reason: string): Promise<RegistryDocument>; createBatch(input: BatchCreateInput): Promise<RegistryDocument[]>;
  workflow(id: string, input: { action: WorkflowAction; expected_version: number | null; note: string | null; department_id?: string | null; user_id?: string | null }): Promise<RegistryDocument>;
  upload(id: string, file: File, category: string): Promise<DocumentAttachment>; download(id: string, attachmentId: string): Promise<Blob>; print(id: string): Promise<Blob>; exportPdf(input: { registru_id: number; start_date?: string; end_date?: string }): Promise<Blob>;
  parties(query?: string): Promise<Page<Party>>; createParty(input: Partial<Party> & Pick<Party, 'party_type' | 'display_name'>): Promise<Party>; updateParty(id: string, input: Partial<Party>): Promise<Party>; deleteParty(id: string): Promise<void>;
  admin(resource: 'departments' | 'organizations' | 'registries'): Promise<Page<RegistryAdminRecord>>; createAdmin(resource: 'departments' | 'organizations' | 'registries', input: Record<string, unknown>): Promise<RegistryAdminRecord>; updateAdmin(resource: 'departments' | 'organizations' | 'registries', id: string | number, input: Record<string, unknown>): Promise<RegistryAdminRecord>; deleteAdmin(resource: 'departments' | 'organizations' | 'registries', id: string | number): Promise<void>;
  chart(): Promise<OrganizationChartNode[]>; adminUsers(): Promise<Page<AdminUser>>; userAssignments(userId: string): Promise<UserAssignment>; saveUserAssignments(userId: string, input: Omit<UserAssignment, 'user_id'>): Promise<UserAssignment>; lookupDocuments(query: string): Promise<DocumentLookup[]>; links(sourceModule: string, sourceRecordId: string): Promise<LinkedDocument[]>; createLink(input: { document_id: string; source_module: string; source_record_id: string; relation_type: string }): Promise<LinkedDocument>; deleteLink(linkId: string): Promise<void>;
}
type Fetcher = (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>;
const page = <T,>(value: T[] | Partial<Page<T>>): Page<T> => Array.isArray(value) ? { items: value, total: value.length, page: 1, pageSize: value.length } : { items: value.items ?? [], total: value.total ?? 0, page: value.page ?? 1, pageSize: value.pageSize ?? 20 };

export function createRegistraturaApi(fetcher: Fetcher = fetch, apiBase = '/api'): RegistraturaApi {
	const contractClient = createContractClient((request) => fetcher(request), apiBase);
  const response = async (path: string, init?: RequestInit) => { const result = await fetcher(`${apiBase}${path}`, { credentials: 'include', ...init, headers: { Accept: 'application/json', ...(init?.headers ?? {}) } }); if (!result.ok) throw new Error(`Registratură: ${result.status}`); return result; };
  const request = async <T,>(path: string, init?: RequestInit): Promise<T> => (await response(path, init)).json() as Promise<T>;
  const json = (value: unknown): RequestInit => ({ method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(value) });
  const encode = encodeURIComponent;
  return {
    async registries() { return page(await request<Registry[] | Page<Registry>>('/registratura/registre')).items; }, filters: () => request('/registratura/documents/filters'),
    async documents({ registryId, page: pageNo, pageSize, filters, sort, direction }) {
      type Query = NonNullable<paths['/api/registratura/documents']['get']['parameters']['query']>;
      const query: Query = { page: pageNo, pageSize, 'filter.registru_id': registryId };
      if (sort) query.sort = sort as Query['sort'];
      if (direction) query.direction = direction;
      Object.entries(filters).forEach(([name, value]) => {
        const trimmed = value?.trim();
        if (!trimmed) return;
        (query as Record<string, string | number>)[name === 'q' ? 'q' : `filter.${name}`] = trimmed;
      });
      const result = await contractClient.GET('/api/registratura/documents', { params: { query } });
      if (!result.response.ok || !result.data) throw new Error(`Registratură: ${result.response.status}`);
      if (!validateGetApiRegistraturaDocumentsResponse(result.data)) throw new Error('Răspuns listă Registratură invalid conform OpenAPI.');
      return page(result.data as Page<RegistryDocument>);
    },
    document: (id) => request(`/registratura/documents/${encode(id)}`), versions: (id) => request(`/registratura/documents/${encode(id)}/versions`), attachments: (id) => request(`/registratura/documents/${encode(id)}/attachments`), workflowHistory: (id) => request(`/registratura/documents/${encode(id)}/workflow-history`), assignees: () => request('/registratura/workflow-assignees'),
    async create(input) {
      type CreateBody = paths['/api/registratura/documents']['post']['requestBody']['content']['application/json'];
      const body: CreateBody = {
        ...input,
        activity: input.activity ?? undefined,
        external_number: input.external_number ?? undefined,
      };
      const result = await contractClient.POST('/api/registratura/documents', { body });
      if (!result.response.ok || !result.data) throw new Error(`Registratură: ${result.response.status}`);
      if (!validatePostApiRegistraturaDocumentsResponse(result.data)) throw new Error('Răspuns creare Registratură invalid conform OpenAPI.');
      return result.data as RegistryDocument;
    }, update: (id, input) => request(`/registratura/documents/${encode(id)}`, { ...json(input), method: 'PATCH' }), createVersion: (id, input) => request(`/registratura/documents/${encode(id)}/versions`, json(input)), cancel: (id, reason) => request(`/registratura/documents/${encode(id)}/cancel`, json({ reason })), createBatch: (input) => request('/registratura/documents/batch', json(input)), workflow: (id, input) => request(`/registratura/documents/${encode(id)}/workflow-actions`, json(input)),
    async upload(id, file, category) { const body = new FormData(); body.set('file', file, file.name); body.set('category', category); return request(`/registratura/documents/${encode(id)}/attachments/upload`, { method: 'POST', body }); },
    async download(id, attachmentId) { return (await response(`/registratura/documents/${encode(id)}/attachments/${encode(attachmentId)}/download`)).blob(); }, async print(id) { return (await response(`/registratura/documents/${encode(id)}/print-pdf`)).blob(); }, async exportPdf(input) { return (await response('/registratura/documents/export-pdf', json(input))).blob(); },
    async parties(query) { const params = new URLSearchParams({ page: '1', pageSize: '50' }); if (query?.trim()) params.set('filter.query', query.trim()); return page(await request<Party[] | Page<Party>>(`/registratura/parties?${params}`)); }, createParty: (input) => request('/registratura/parties', json(input)), updateParty: (id, input) => request(`/registratura/parties/${encode(id)}`, { ...json(input), method: 'PATCH' }), async deleteParty(id) { await response(`/registratura/parties/${encode(id)}`, { method: 'DELETE' }); },
    async admin(resource) { return page(await request<RegistryAdminRecord[] | Page<RegistryAdminRecord>>(`/registratura/admin/${resource}?page=1&pageSize=50`)); }, createAdmin: (resource, input) => request(`/registratura/admin/${resource}`, json(input)), updateAdmin: (resource, id, input) => request(`/registratura/admin/${resource}/${encode(String(id))}`, { ...json(input), method: 'PATCH' }), async deleteAdmin(resource, id) { await response(`/registratura/admin/${resource}/${encode(String(id))}`, { method: 'DELETE' }); },
    chart: () => request('/registratura/admin/organization-chart'), async adminUsers() { return page(await request<AdminUser[] | Page<AdminUser>>('/admin/users?page=1&pageSize=100&sort=name&direction=asc')); }, userAssignments: (id) => request(`/registratura/admin/users/${encode(id)}/assignments`), saveUserAssignments: (id, input) => request(`/registratura/admin/users/${encode(id)}/assignments`, { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(input) }), lookupDocuments: (query) => request(`/registratura/documents/lookup?query=${encode(query)}`), links: (sourceModule, sourceRecordId) => request(`/registratura/document-links?source_module=${encode(sourceModule)}&source_record_id=${encode(sourceRecordId)}`), createLink: (input) => request('/registratura/document-links', json(input)), async deleteLink(linkId) { await response(`/registratura/document-links/${encode(linkId)}`, { method: 'DELETE' }); }
  };
}
