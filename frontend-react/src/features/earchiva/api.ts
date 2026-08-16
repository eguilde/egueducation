import type { Fetcher, Page } from '../workflow/api';

export interface ArchiveRecord { id: string; record_number: string; title: string; fond: string; series: string; source_module: string; source_reference: string; status: string; assigned_archivist: string; archived_at: string; retention_years: number; box_number: string; location_code: string; notes: string }
export interface ArchiveFilters { fonds: string[]; series: string[]; statuses: string[]; source_modules: string[]; archivists: string[] }
export interface ArchiveDashboard { stats: { total_records: number; validated_records: number; draft_records: number; unique_fonds: number } }
export interface CreateRecordInput { title: string; fond: string; series: string; source_module: string; source_reference: string; status: string; retention_years: number; assigned_archivist: string; box_number: string; location_code: string; archived_at: string; notes: string }
export interface ArchiveDocument { id: string; title: string; original_file_name: string; mime_type: string; source_kind: string; source_system: string; external_reference: string; taxonomy_code?: string | null; taxonomy_label?: string | null; status: string; document_date?: string | null; metadata?: Record<string, unknown>; current_version_no: number; received_at: string; created_at: string; updated_at: string; score?: number; snippet?: string }
export interface ArchiveDocumentDetail extends ArchiveDocument { latest_version?: ArchiveDocumentVersion }
export interface ArchiveDocumentVersion { id: string; document_id: string; version_no: number; source_sha256: string; source_size_bytes: number; page_count: number; text_status: string; created_by: string; created_at: string; chunk_count?: number }
export interface ArchiveTaxonomy { id: string; parent_id?: string | null; code: string; label: string; description: string; path: string; active: boolean; sort_order: number }
export interface UploadArchiveDocumentInput { file: File; title: string; source_kind: string; source_system?: string; external_reference?: string; taxonomy_code?: string; taxonomy_label?: string; taxonomy_parent?: string; document_date?: string; metadata?: Record<string, unknown> }
export interface ArchiveApi { dashboard(): Promise<ArchiveDashboard>; records(query?: Record<string, string>): Promise<Page<ArchiveRecord>>; filters(): Promise<ArchiveFilters>; create(input: CreateRecordInput): Promise<ArchiveRecord>; documents(query?: Record<string, string>): Promise<Page<ArchiveDocument>>; document(id: string): Promise<ArchiveDocumentDetail>; versions(id: string): Promise<ArchiveDocumentVersion[]>; taxonomy(query?: Record<string, string>): Promise<ArchiveTaxonomy[]>; upload(input: UploadArchiveDocumentInput): Promise<ArchiveDocumentDetail> }
const page = <T,>(value: T[] | Partial<Page<T>>): Page<T> => Array.isArray(value) ? { items: value, total: value.length, page: 1, pageSize: value.length } : { items: value.items ?? [], total: value.total ?? 0, page: value.page ?? 1, pageSize: value.pageSize ?? 25 };

export function createArchiveApi(fetcher: Fetcher = fetch, apiBase = '/api'): ArchiveApi {
  const request = async <T,>(path: string, init?: RequestInit) => { const response = await fetcher(`${apiBase}${path}`, { credentials: 'include', ...init, headers: { Accept: 'application/json', ...(init?.headers ?? {}) } }); if (!response.ok) throw new Error(`eArhivă: ${response.status}`); return response.json() as Promise<T>; };
  const query = (path: string, values: Record<string, string>) => `${path}?${new URLSearchParams(values)}`;
  return {
    dashboard: () => request('/earchiva/dashboard'), filters: () => request('/earchiva/records/filters'),
    async records(values = {}) { return page(await request<ArchiveRecord[] | Partial<Page<ArchiveRecord>>>(query('/earchiva/records', { page: '1', pageSize: '25', ...values }))); },
    create: (input) => request('/earchiva/records', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(input) }),
    async documents(values = {}) { return page(await request<ArchiveDocument[] | Partial<Page<ArchiveDocument>>>(query('/earchiva/documents', { page: '1', pageSize: '25', ...values }))); },
    document: (id) => request(`/earchiva/documents/${encodeURIComponent(id)}`), versions: (id) => request(`/earchiva/documents/${encodeURIComponent(id)}/versions`),
    taxonomy: (values = {}) => request(query('/earchiva/taxonomy', values)),
    async upload(input) { const data = new FormData(); data.set('file', input.file); data.set('title', input.title); data.set('source_kind', input.source_kind); if (input.source_system) data.set('source_system', input.source_system); if (input.external_reference) data.set('external_reference', input.external_reference); if (input.taxonomy_code) data.set('taxonomy_code', input.taxonomy_code); if (input.taxonomy_label) data.set('taxonomy_label', input.taxonomy_label); if (input.taxonomy_parent) data.set('taxonomy_parent', input.taxonomy_parent); if (input.document_date) data.set('document_date', input.document_date); if (input.metadata) data.set('metadata', JSON.stringify(input.metadata)); return request('/earchiva/documents', { method: 'POST', body: data }); }
  };
}
