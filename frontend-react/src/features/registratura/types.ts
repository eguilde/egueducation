/** Contract-shaped models for the Registratură endpoints. Replace imports with
 * generated `paths` types once `src/api/generated.ts` is emitted in the root app. */
export interface Registry { id: number; nume: string; prefix_nr: string; tip_registru: string; isDefault: boolean }
export interface RegistryDocument {
  id: string; registru_id?: number | null; registry_number: string; subject: string; document_type: string;
  direction: 'intrare' | 'iesire' | string; status: string; correspondent: string; assigned_to: string;
  registered_at: string; due_date?: string | null; workflow_version?: number | null;
  confidentiality?: string; summary?: string; external_number?: string | null;
  external_number_date?: string | null; entry_at?: string | null; exit_at?: string | null;
  activity?: string | null; record_kind?: 'document' | 'dosar' | string | null;
  department_ids?: string[]; department_names?: string[]; cancellation_reason?: string | null;
  workflow_assignment?: WorkflowAssignment | null;
}
export interface Page<T> { items: T[]; total: number; page: number; pageSize: number }
export interface DocumentFilters { registry_number?: string; subject?: string; document_type?: string; direction?: string; status?: string; correspondent?: string; assigned_to?: string; confidentiality?: string; registered_at_from?: string; registered_at_to?: string; due_date_from?: string; due_date_to?: string; q?: string }
export interface DocumentFilterOptions { document_types: string[]; directions: string[]; statuses: string[]; confidentialities: string[] }
export type WorkflowAction = 'assign_department' | 'assign_user' | 'claim' | 'send_for_approval' | 'approve' | 'reject';
export interface WorkflowAssignment { department_id?: string | null; department_name?: string | null; user_id?: string | null; user_name?: string | null; approver_id?: string | null; approver_name?: string | null }
export interface WorkflowHistoryEntry { id: string; document_id: string; action: string; from_status?: string | null; to_status: string; note?: string | null; actor_name?: string | null; created_at: string }
export interface WorkflowAssignees { departments: AssigneeOption[]; users: AssigneeOption[] }
export interface AssigneeOption { id: string; name: string; department_ids?: string[] }
export interface DocumentVersion { id: string; document_id: string; version_no: number; subject: string; document_type: string; direction: string; status: string; correspondent: string; assigned_to: string; confidentiality: string; summary: string; due_date?: string | null; change_notes: string; created_by: string; created_at: string }
export interface DocumentAttachment { id: string; document_id: string; title: string; file_name: string; mime_type: string; storage_key: string; size_bytes: number; category: string; status: string; uploaded_by: string; uploaded_at: string }
export interface CreateDocumentInput { registru_id: number; subject: string; document_type: string; direction: 'intrare' | 'iesire'; status: string; correspondent: string; assigned_to: string; confidentiality: string; summary: string; due_date?: string | null; external_number?: string | null; external_number_date?: string | null; entry_at?: string | null; exit_at?: string | null; activity?: string | null; record_kind?: 'document' | 'dosar'; department_ids?: string[]; correspondent_party_id?: string | null; assigned_party_id?: string | null }
export interface BatchCreateInput extends CreateDocumentInput { count: number }
export interface Party { id: string; party_type: 'physical' | 'legal' | 'institution' | string; display_name: string; first_name?: string; last_name?: string; legal_name?: string; identifier_code?: string; tax_id?: string; email?: string; phone_number?: string; address_line1?: string; locality?: string; county?: string; country?: string; notes?: string; active?: boolean; is_default_organization?: boolean; birth_date?: string | null; birth_place?: string; trade_register_number?: string; legal_representative?: string; legal_form?: string; institution_type?: string; website?: string }
export interface RegistryAdminRecord { id: string | number; name?: string; nume?: string; code?: string; active?: boolean; [key: string]: unknown }
export interface LinkedDocument { link_id: string; document_id: string; registry_number: string; subject: string; document_type: string; status: string; relation_type: string; registered_at: string; confidentiality: string }
export interface DocumentLookup { id: string; registry_number: string; subject: string; document_type: string; status: string }
export interface OrganizationChartNode { id: string; name: string; description?: string; parent_id?: string | null; role_tag?: string; user_count: number; users: { id: string; name: string; email?: string }[]; children: OrganizationChartNode[] }
export interface UserAssignment { user_id: string; department_ids: string[]; primary_department_id?: string | null; organization_id?: string | null }
export interface AdminUser { id: string; name: string; email: string; position: string; status: string; locale: string }
