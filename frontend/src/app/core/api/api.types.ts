export interface PagedResponse<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
}

export interface AppMeta {
  name: string;
  default_locale: 'ro' | 'en';
  available_locales: Array<'ro' | 'en'>;
  theme: {
    family: string;
    brand: string;
  };
}

export interface AuthMethod {
  code: string;
  enabled: boolean;
  primary: boolean;
}

export interface AuthMethodsResponse {
  methods: AuthMethod[];
}

export interface AuthUiConfig {
  auth_flow: string;
  default_locale: 'ro' | 'en';
  available_locales: Array<'ro' | 'en'>;
  theme_family: string;
  theme_brand: string;
  oidc_issuer: string;
  oidc_client_id: string;
  desktop_client_id: string;
  sms_otp_enabled: boolean;
  passkey_enabled: boolean;
  eudi_wallet_enabled: boolean;
  gdpr_features_enabled: boolean;
}

export interface RoleCatalogItem {
  code: string;
  label: string;
  description: string;
  permissions?: string[];
  positions?: string[];
}

export interface RoleCatalogResponse {
  roles: RoleCatalogItem[];
}

export interface RolePositionItem {
  position_code: string;
  position_name: string;
  role_code: string;
  role_label: string;
}

export interface RolePositionResponse {
  items: RolePositionItem[];
}

export interface RequestSMSOTPRequest {
  phone_number: string;
  identifier?: string;
}

export interface VerifySMSOTPRequest {
  phone_number: string;
  identifier?: string;
  code: string;
}

export interface SMSOTPRequestResponse {
  status: string;
  channel: string;
  masked_phone: string;
}

export interface SMSOTPVerifyResponse {
  status: string;
  channel: string;
  session: SessionContext;
}

export interface PasskeyAuthenticationOptions {
  challenge: string;
  rp: {
    id: string;
    name: string;
  };
  allowCredentials?: Array<{
    type: 'public-key';
    id: string;
    transports?: string[];
  }>;
  timeout: number;
  userVerification: 'required' | 'preferred' | 'discouraged';
}

export interface BeginPasskeyAuthenticationResponse {
  status: string;
  options: PasskeyAuthenticationOptions;
}

export interface FinishPasskeyAuthenticationRequest {
  challenge: string;
  credential_id: string;
  response: Record<string, unknown>;
}

export interface AuthConsentScope {
  code: string;
  label: string;
  description?: string;
  required: boolean;
}

export interface AuthConsentRequestResponse {
  request_id: string;
  client_id: string;
  client_name: string;
  scopes: AuthConsentScope[];
  expires_at: string;
}

export interface AuthConsentDecisionRequest {
  request_id: string;
  decision: 'allow' | 'deny';
  granted_scopes: string[];
}

export interface AuthConsentDecisionResponse {
  status: string;
  redirect_to: string;
}

export interface ModuleStatus {
  code: string;
  active: boolean;
}

export interface AdminDashboardResponse {
  stats: {
    users: number;
    memberships: number;
    positions: number;
    permissions: number;
    workflows: number;
    archives: number;
    ready_dossiers: number;
    blocked_dossiers: number;
  };
  modules: ModuleStatus[];
  admin_sections: string[];
  warnings: string[];
}

export interface AdminUser {
  id: string;
  sub: string;
  name: string;
  email: string;
  phone: string;
  position: string;
  locale: string;
  status: string;
  email_verified: boolean;
  phone_verified: boolean;
  preferred_otp_channel: string;
  last_login_at: string;
}

export interface UpsertAdminUserRequest {
  id?: string;
  name: string;
  email: string;
  phone: string;
  locale: string;
  status: string;
  email_verified: boolean;
  phone_verified: boolean;
  preferred_otp_channel: string;
}

export interface AdminRole {
  code: string;
  label: string;
}

export interface UpsertAdminRoleRequest {
  code: string;
  label: string;
}

export interface AdminUserRoleAssignment {
  id: string;
  user_id: string;
  user_name: string;
  user_email: string;
  role_code: string;
  role_label: string;
}

export interface UpsertAdminUserRoleAssignmentRequest {
  user_id: string;
  role_code: string;
  assigned: boolean;
}

export interface AdminRolePermissionAssignment {
  id: string;
  role_code: string;
  role_label: string;
  permission_code: string;
  permission_label: string;
}

export interface AdminRolePermissionAssignmentFilters {
  roles: Array<{ code: string; label: string }>;
  permissions: Array<{ code: string; label: string }>;
}

export interface UpsertAdminRolePermissionAssignmentRequest {
  role_code: string;
  permission_code: string;
  assigned: boolean;
}

export interface AdminPositionRoleAssignment {
  id: string;
  position_code: string;
  position_name: string;
  role_code: string;
  role_label: string;
}

export interface AdminPositionRoleAssignmentFilters {
  positions: Array<{ code: string; name: string }>;
  roles: Array<{ code: string; label: string }>;
}

export interface UpsertAdminPositionRoleAssignmentRequest {
  position_code: string;
  role_code: string;
  assigned: boolean;
}

export interface AdminMembership {
  id: string;
  user_id: string;
  user_name: string;
  user_email: string;
  position_code: string;
  position_name: string;
  org_unit_code: string;
  organization_name: string;
  is_primary: boolean;
  active: boolean;
  start_date: string;
  end_date: string;
}

export interface UpsertAdminMembershipRequest {
  id?: string;
  user_id: string;
  position_code: string;
  org_unit_code: string;
  organization_name: string;
  is_primary: boolean;
  active: boolean;
  start_date: string;
  end_date?: string;
}

export interface AdminOrgUnit {
  code: string;
  name: string;
  parent_code: string;
  parent_name: string;
  active: boolean;
  sort_order: number;
}

export interface UpsertAdminOrgUnitRequest {
  code: string;
  name: string;
  parent_code: string;
  active: boolean;
  sort_order: number;
}

export interface AdminPosition {
  code: string;
  name: string;
  scope_module: string;
  active: boolean;
  sort_order: number;
}

export interface UpsertAdminPositionRequest {
  code: string;
  name: string;
  scope_module: string;
  active: boolean;
  sort_order: number;
}

export interface AdminPermission {
  code: string;
  label: string;
  user_count: number;
  role_count: number;
}

export interface AdminPermissionAssignment {
  id: string;
  permission_code: string;
  permission_label: string;
  position_code: string;
  position_name: string;
  scope_module: string;
}

export interface AdminPermissionAssignmentFilters {
  permissions: Array<{ code: string; label: string }>;
  positions: Array<{ code: string; name: string }>;
}

export interface UpsertAdminPermissionAssignmentRequest {
  permission_code: string;
  position_code: string;
  assigned: boolean;
}

export interface AdminUserFilters {
  positions: string[];
  statuses: string[];
  locales: string[];
}

export interface AdminDossierRequirement {
  id: string;
  source_module: string;
  relation_type: string;
  min_count: number;
  required_for_readiness: boolean;
  required_for_submit: boolean;
  required_for_approve: boolean;
}

export interface AdminDossierRequirementFilters {
  source_modules: string[];
  relation_types: string[];
}

export interface CreateAdminDossierRequirementRequest {
  source_module: string;
  relation_type: string;
  min_count: number;
  required_for_readiness: boolean;
  required_for_submit: boolean;
  required_for_approve: boolean;
}

export interface AdminEducationTaxonomy {
  id: string;
  domain: string;
  code: string;
  label_ro: string;
  label_en: string;
  active: boolean;
  sort_order: number;
}

export interface AdminEducationTaxonomyFilters {
  domains: string[];
}

export interface CreateAdminEducationTaxonomyRequest {
  domain: string;
  code: string;
  label_ro: string;
  label_en: string;
  active: boolean;
  sort_order: number;
}

export interface AdminWorkflowDefinition {
  code: string;
  name: string;
  category: string;
  initial_step: string;
  sla_hours: number;
  active: boolean;
}

export interface AdminWorkflowDefinitionFilters {
  categories: string[];
}

export interface CreateAdminWorkflowDefinitionRequest {
  code: string;
  name: string;
  category: string;
  initial_step: string;
  sla_hours: number;
  active: boolean;
}

export interface AdminNomenclature {
  id: string;
  domain: string;
  code: string;
  label_ro: string;
  label_en: string;
  active: boolean;
  sort_order: number;
}

export interface AdminNomenclatureFilters {
  domains: string[];
}

export interface CreateAdminNomenclatureRequest {
  domain: string;
  code: string;
  label_ro: string;
  label_en: string;
  active: boolean;
  sort_order: number;
}

export interface AdminAuthMethodSetting {
  code: string;
  enabled: boolean;
  primary_method: boolean;
  sort_order: number;
}

export interface UpdateAdminAuthMethodSettingRequest {
  code: string;
  enabled: boolean;
  primary_method: boolean;
  sort_order: number;
}

export interface AdminModuleSetting {
  code: string;
  active: boolean;
}

export interface UpdateAdminModuleSettingRequest {
  code: string;
  active: boolean;
}

export interface AdminOIDCClient {
  client_id: string;
  client_name: string;
  public_client: boolean;
  require_pkce: boolean;
  active: boolean;
  redirect_uris: string[];
  created_at: string;
}

export interface UpsertAdminOIDCClientRequest {
  client_id: string;
  client_name: string;
  public_client: boolean;
  require_pkce: boolean;
  active: boolean;
  redirect_uris: string[];
}

export interface AdminOIDCConsentGrant {
  id: string;
  client_id: string;
  client_name: string;
  subject: string;
  subject_name: string;
  subject_email: string;
  scope: string;
  granted_at: string;
}

export interface RevokeAdminOIDCConsentRequest {
  id: string;
}

export interface AdminOIDCSession {
  token_id: string;
  client_id: string;
  client_name: string;
  subject: string;
  subject_name: string;
  subject_email: string;
  scope: string;
  created_at: string;
  expires_at: string;
  revoked: boolean;
}

export interface RevokeAdminOIDCSessionRequest {
  token_id: string;
}

export interface AdminAuditEvent {
  id: string;
  actor_subject: string;
  domain: string;
  action: string;
  target_type: string;
  target_id: string;
  status: string;
  summary: string;
  created_at: string;
}

export interface AdminAuditFilters {
  domains: string[];
  target_types: string[];
  statuses: string[];
}

export interface AdminGdprSetting {
  code: string;
  value_type: 'text' | 'bool' | 'int';
  value_text: string;
  value_bool: boolean;
  value_int: number;
}

export interface UpdateAdminGdprSettingRequest {
  code: string;
  value_type: 'text' | 'bool' | 'int';
  value_text: string;
  value_bool: boolean;
  value_int: number;
}

export interface RegistraturaDocument {
  id: string;
  registru_id?: number | null;
  registry_number: string;
  subject: string;
  document_type: string;
  direction: string;
  status: string;
  correspondent: string;
  assigned_to: string;
  institution_id: string;
  confidentiality: string;
  summary: string;
  registered_at: string;
  due_date?: string | null;
}

export interface RegistraturaRegistry {
  id: number;
  nume: string;
  prefix_nr: string;
  nr_inceput: number;
  nr_curent: string;
  nr_urmator: string;
  data_resetare?: string | null;
  tip_registru: string;
  isDefault: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateRegistraturaRegistryRequest {
  nume: string;
  prefix_nr: string;
  nr_inceput: number;
  nr_curent?: string;
  nr_urmator?: string;
  data_resetare?: string | null;
  tip_registru: string;
  isDefault: boolean;
}

export interface UpdateRegistraturaRegistryRequest {
  nume?: string;
  prefix_nr?: string;
  nr_inceput?: number;
  nr_curent?: string;
  nr_urmator?: string;
  data_resetare?: string | null;
  tip_registru?: string;
  isDefault?: boolean;
}

export interface RegistraturaDocumentVersion {
  id: string;
  document_id: string;
  version_no: number;
  subject: string;
  document_type: string;
  direction: string;
  status: string;
  correspondent: string;
  assigned_to: string;
  confidentiality: string;
  summary: string;
  due_date?: string | null;
  change_notes: string;
  created_by: string;
  created_at: string;
}

export interface RegistraturaDocumentAttachment {
  id: string;
  document_id: string;
  title: string;
  file_name: string;
  mime_type: string;
  storage_key: string;
  size_bytes: number;
  category: string;
  status: string;
  uploaded_by: string;
  uploaded_at: string;
}

export interface RegistraturaDocumentFilters {
  document_types: string[];
  directions: string[];
  statuses: string[];
  confidentialities: string[];
}

export interface CreateRegistraturaDocumentRequest {
  registru_id?: number | null;
  subject: string;
  document_type: string;
  direction: string;
  status: string;
  correspondent: string;
  assigned_to: string;
  confidentiality: string;
  summary: string;
  due_date?: string | null;
}

export interface UpdateRegistraturaDocumentRequest {
  registru_id?: number | null;
  subject: string;
  document_type: string;
  direction: string;
  status: string;
  correspondent: string;
  assigned_to: string;
  confidentiality: string;
  summary: string;
  due_date?: string | null;
  change_notes?: string;
}

export interface CancelRegistraturaDocumentRequest {
  reason: string;
}

export interface BatchCreateRegistraturaDocumentRequest {
  registru_id: number;
  count: number;
  subject: string;
  document_type: string;
  direction: string;
  status: string;
  correspondent: string;
  assigned_to: string;
  confidentiality: string;
  summary: string;
  due_date?: string | null;
}

export interface ExportRegistraturaDocumentsRequest {
  registru_id?: number | null;
  start_date?: string | null;
  end_date?: string | null;
}

export interface CreateRegistraturaDocumentVersionRequest {
  subject: string;
  status: string;
  assigned_to: string;
  confidentiality: string;
  summary: string;
  due_date?: string | null;
  change_notes: string;
}

export interface CreateRegistraturaDocumentAttachmentRequest {
  title: string;
  file_name: string;
  mime_type: string;
  storage_key: string;
  size_bytes: number;
  category: string;
  status: string;
  uploaded_by: string;
}

export interface DocumentLookupItem {
  id: string;
  registry_number: string;
  subject: string;
  document_type: string;
  status: string;
}

export interface LinkedDocument {
  link_id: string;
  document_id: string;
  registry_number: string;
  subject: string;
  document_type: string;
  status: string;
  relation_type: string;
  registered_at: string;
  confidentiality: string;
}

export interface CreateDocumentLinkRequest {
  document_id: string;
  source_module: string;
  source_record_id: string;
  relation_type: string;
}

export interface WorkflowDefinition {
  code: string;
  name: string;
  category: string;
  initial_step: string;
  sla_hours: number;
  active: boolean;
}

export interface WorkflowTask {
  id: string;
  definition_code: string;
  definition_name: string;
  title: string;
  document_number: string;
  source_module: string;
  source_record_id?: string | null;
  source_label?: string;
  status: string;
  priority: string;
  assigned_to: string;
  current_step: string;
  due_at?: string | null;
  started_at: string;
  updated_at: string;
  institution_id: string;
  summary: string;
  linked_documents_count: number;
  dossier_ready: boolean;
  missing_relations: string[];
  available_actions: string[];
}

export interface WorkflowFilters {
  statuses: string[];
  priorities: string[];
  assignees: string[];
}

export interface WorkflowDashboardResponse {
  stats: {
    active_tasks: number;
    overdue_tasks: number;
    waiting_approval: number;
    active_definitions: number;
    ready_dossiers: number;
    blocked_dossiers: number;
  };
}

export interface CreateWorkflowTaskRequest {
  definition_code: string;
  title: string;
  document_number: string;
  source_module?: string;
  source_record_id?: string;
  priority: string;
  assigned_to: string;
  due_date?: string | null;
  summary: string;
}

export interface TransitionWorkflowTaskRequest {
  action: string;
}

export interface ArchiveRecord {
  id: string;
  record_number: string;
  title: string;
  fond: string;
  series: string;
  source_module: string;
  source_reference: string;
  status: string;
  retention_years: number;
  assigned_archivist: string;
  box_number: string;
  location_code: string;
  archived_at: string;
  institution_id: string;
  notes: string;
}

export interface ArchiveFilters {
  fonds: string[];
  series: string[];
  statuses: string[];
  source_modules: string[];
  archivists: string[];
}

export interface ArchiveDashboardResponse {
  stats: {
    total_records: number;
    validated_records: number;
    draft_records: number;
    unique_fonds: number;
  };
}

export interface CreateArchiveRecordRequest {
  title: string;
  fond: string;
  series: string;
  source_module: string;
  source_reference: string;
  status: string;
  retention_years: number;
  assigned_archivist: string;
  box_number: string;
  location_code: string;
  archived_at: string;
  notes: string;
}

export interface GovernanceMeeting {
  id: string;
  school_year: string;
  organism: string;
  title: string;
  meeting_type: string;
  status: string;
  quorum_required: number;
  participants_count: number;
  meeting_date: string;
  location: string;
  chairperson: string;
  secretary_name: string;
  institution_id: string;
  summary: string;
}

export interface EducationTaxonomyItem {
  id: string;
  domain: string;
  code: string;
  label_ro: string;
  label_en: string;
  active: boolean;
  sort_order: number;
}

export interface EducationTaxonomyCatalogResponse {
  items: Record<string, EducationTaxonomyItem[]>;
}

export interface GovernanceFilters {
  school_years: string[];
  organisms: string[];
  meeting_types: string[];
  statuses: string[];
}

export interface EducationDashboardResponse {
  stats: {
    total_meetings: number;
    scheduled_meetings: number;
    held_meetings: number;
    published_meetings: number;
  };
}

export interface CreateGovernanceMeetingRequest {
  school_year: string;
  organism: string;
  title: string;
  meeting_type: string;
  status: string;
  quorum_required: number;
  participants_count: number;
  meeting_date: string;
  location: string;
  chairperson: string;
  secretary_name: string;
  summary: string;
}

export interface GovernanceDecision {
  id: string;
  decision_code: string;
  school_year: string;
  organism: string;
  title: string;
  status: string;
  publication_status: string;
  decision_date: string;
  legal_basis: string;
  signed_by: string;
  institution_id: string;
  summary: string;
}

export interface GovernanceDecisionFilters {
  school_years: string[];
  organisms: string[];
  statuses: string[];
  publication_statuses: string[];
}

export interface GovernanceDecisionDashboardResponse {
  stats: {
    total_decisions: number;
    approved_decisions: number;
    published_decisions: number;
    pending_publication: number;
  };
}

export interface CreateGovernanceDecisionRequest {
  school_year: string;
  organism: string;
  title: string;
  status: string;
  publication_status: string;
  decision_date: string;
  legal_basis: string;
  signed_by: string;
  summary: string;
}

export interface ManagerialDossier {
  id: string;
  dossier_code: string;
  school_year: string;
  dossier_type: string;
  title: string;
  status: string;
  owner_name: string;
  due_on: string;
  publication_required: boolean;
  institution_id: string;
  summary: string;
}

export interface ManagerialDossierFilters {
  school_years: string[];
  dossier_types: string[];
  statuses: string[];
}

export interface ManagerialDossierDashboardResponse {
  stats: {
    total_dossiers: number;
    review_dossiers: number;
    published_dossiers: number;
    overdue_dossiers: number;
  };
}

export interface CreateManagerialDossierRequest {
  school_year: string;
  dossier_type: string;
  title: string;
  status: string;
  owner_name: string;
  due_on: string;
  publication_required: boolean;
  summary: string;
}

export interface RegulationRecord {
  id: string;
  regulation_code: string;
  school_year: string;
  regulation_type: string;
  title: string;
  status: string;
  approval_status: string;
  owner_name: string;
  review_due_on: string;
  approved_on: string;
  institution_id: string;
  summary: string;
}

export interface RegulationFilters {
  school_years: string[];
  regulation_types: string[];
  statuses: string[];
  approval_statuses: string[];
}

export interface RegulationDashboardResponse {
  stats: {
    total_regulations: number;
    consultation_items: number;
    approved_regulations: number;
    published_regulations: number;
  };
}

export interface CreateRegulationRecordRequest {
  school_year: string;
  regulation_type: string;
  title: string;
  status: string;
  approval_status: string;
  owner_name: string;
  review_due_on: string;
  approved_on: string;
  summary: string;
}

export interface PersonnelRecord {
  id: string;
  employee_code: string;
  full_name: string;
  role_title: string;
  employment_type: string;
  status: string;
  evaluation_status: string;
  mobility_stage: string;
  school_year: string;
  assigned_unit: string;
  phone: string;
  email: string;
  has_portfolio: boolean;
  institution_id: string;
  notes: string;
}

export interface PersonnelFilters {
  school_years: string[];
  employment_types: string[];
  statuses: string[];
  evaluation_statuses: string[];
  mobility_stages: string[];
}

export interface PersonnelDashboardResponse {
  stats: {
    total_records: number;
    active_records: number;
    portfolios_enabled: number;
    mobility_cases: number;
  };
}

export interface CreatePersonnelRecordRequest {
  full_name: string;
  role_title: string;
  employment_type: string;
  status: string;
  evaluation_status: string;
  mobility_stage: string;
  school_year: string;
  assigned_unit: string;
  phone: string;
  email: string;
  has_portfolio: boolean;
  notes: string;
}

export interface PersonnelEvaluation {
  id: string;
  evaluation_code: string;
  employee_code: string;
  full_name: string;
  role_title: string;
  school_year: string;
  status: string;
  score: number;
  evaluator_name: string;
  finalized_on: string;
  institution_id: string;
  summary: string;
}

export interface PersonnelEvaluationFilters {
  school_years: string[];
  statuses: string[];
}

export interface PersonnelEvaluationDashboardResponse {
  stats: {
    total_evaluations: number;
    submitted_evaluations: number;
    approved_evaluations: number;
    contested_evaluations: number;
  };
}

export interface CreatePersonnelEvaluationRequest {
  employee_code: string;
  full_name: string;
  role_title: string;
  school_year: string;
  status: string;
  score: number;
  evaluator_name: string;
  finalized_on: string;
  summary: string;
}

export interface PersonnelDeclaration {
  id: string;
  declaration_code: string;
  employee_code: string;
  full_name: string;
  declaration_type: string;
  status: string;
  school_year: string;
  submitted_on: string;
  valid_until: string;
  institution_id: string;
  summary: string;
}

export interface PersonnelDeclarationFilters {
  school_years: string[];
  declaration_types: string[];
  statuses: string[];
}

export interface PersonnelDeclarationDashboardResponse {
  stats: {
    total_declarations: number;
    submitted_declarations: number;
    validated_declarations: number;
    expired_declarations: number;
  };
}

export interface CreatePersonnelDeclarationRequest {
  employee_code: string;
  full_name: string;
  declaration_type: string;
  status: string;
  school_year: string;
  submitted_on: string;
  valid_until: string;
  summary: string;
}

export interface MobilityCase {
  id: string;
  case_code: string;
  employee_code: string;
  full_name: string;
  school_year: string;
  request_type: string;
  stage: string;
  status: string;
  source_school: string;
  destination_school: string;
  submitted_on: string;
  reviewed_by: string;
  institution_id: string;
  notes: string;
}

export interface MobilityFilters {
  school_years: string[];
  request_types: string[];
  stages: string[];
  statuses: string[];
}

export interface MobilityDashboardResponse {
  stats: {
    total_cases: number;
    open_cases: number;
    approved_cases: number;
    transfer_cases: number;
  };
}

export interface CreateMobilityCaseRequest {
  employee_code: string;
  full_name: string;
  school_year: string;
  request_type: string;
  stage: string;
  status: string;
  source_school: string;
  destination_school: string;
  submitted_on: string;
  reviewed_by: string;
  notes: string;
}

export interface MeritGrant {
  id: string;
  grant_code: string;
  full_name: string;
  role_title: string;
  school_year: string;
  category: string;
  status: string;
  score: number;
  committee_name: string;
  decision_date: string;
  funded: boolean;
  institution_id: string;
  notes: string;
}

export interface MeritGrantFilters {
  school_years: string[];
  categories: string[];
  statuses: string[];
}

export interface MeritGrantDashboardResponse {
  stats: {
    total_records: number;
    approved_records: number;
    funded_records: number;
    average_score: number;
  };
}

export interface CreateMeritGrantRequest {
  full_name: string;
  role_title: string;
  school_year: string;
  category: string;
  status: string;
  score: number;
  committee_name: string;
  decision_date: string;
  funded: boolean;
  notes: string;
}

export interface PortfolioRecord {
  id: string;
  portfolio_code: string;
  owner_name: string;
  owner_role: string;
  school_year: string;
  status: string;
  section_count: number;
  last_updated_on: string;
  retention_until: string;
  transfer_status: string;
  authenticity_declared: boolean;
  consent_captured: boolean;
  custodian: string;
  institution_id: string;
  notes: string;
}

export interface PortfolioFilters {
  school_years: string[];
  statuses: string[];
  transfer_statuses: string[];
}

export interface PortfolioDashboardResponse {
  stats: {
    total_portfolios: number;
    validated_portfolios: number;
    transfer_portfolios: number;
    declared_portfolios: number;
  };
}

export interface CreatePortfolioRecordRequest {
  owner_name: string;
  owner_role: string;
  school_year: string;
  status: string;
  section_count: number;
  last_updated_on: string;
  retention_until: string;
  transfer_status: string;
  authenticity_declared: boolean;
  consent_captured: boolean;
  custodian: string;
  notes: string;
}

export interface GdprRetentionPolicy {
  id: string;
  policy_code: string;
  domain_code: string;
  record_category: string;
  retention_years: number;
  legal_basis: string;
  status: string;
  review_due_on: string;
  owner_name: string;
  institution_id: string;
  notes: string;
}

export interface GdprSubjectRequest {
  id: string;
  request_code: string;
  subject_name: string;
  request_type: string;
  status: string;
  submitted_on: string;
  due_on: string;
  handled_by: string;
  source_module: string;
  anonymization_required: boolean;
  institution_id: string;
  notes: string;
}

export interface GdprSubjectExport {
  id: string;
  export_code: string;
  request_id: string;
  subject_name: string;
  source_module: string;
  status: string;
  export_format: string;
  approved_by: string;
  approved_on: string;
  generated_on: string;
  package_summary: string;
  institution_id: string;
  notes: string;
}

export interface GdprPublicationReview {
  id: string;
  review_code: string;
  source_module: string;
  source_record_id: string;
  source_label: string;
  anonymization_status: string;
  publication_status: string;
  reviewed_by: string;
  reviewed_on: string;
  legal_basis: string;
  institution_id: string;
  notes: string;
}

export interface GdprRetentionFilters {
  domains: string[];
  statuses: string[];
}

export interface GdprSubjectRequestFilters {
  request_types: string[];
  statuses: string[];
  source_modules: string[];
}

export interface GdprDashboardResponse {
  stats: {
    active_policies: number;
    pending_requests: number;
    overdue_requests: number;
    anonymization_cases: number;
  };
}

export interface GdprExportDashboardResponse {
  stats: {
    total_exports: number;
    pending_approval: number;
    generated_exports: number;
    delivered_exports: number;
  };
}

export interface GdprPublicationDashboardResponse {
  stats: {
    total_reviews: number;
    pending_anonymization: number;
    ready_for_publication: number;
    published_items: number;
  };
}

export interface GdprConfigResponse {
  settings: {
    publication_anonymization_required: boolean;
    subject_export_requires_approval: boolean;
    default_response_sla_days: number;
    retention_review_notice_days: number;
    portfolio_consent_required: boolean;
    portfolio_authenticity_required: boolean;
  };
  catalogs: {
    domains: string[];
    policy_status: string[];
    request_types: string[];
    request_status: string[];
    source_modules: string[];
  };
}

export interface GdprSubjectExportFilters {
  statuses: string[];
  export_formats: string[];
  source_modules: string[];
}

export interface GdprPublicationReviewFilters {
  source_modules: string[];
  anonymization_statuses: string[];
  publication_statuses: string[];
}

export interface CreateGdprRetentionPolicyRequest {
  domain_code: string;
  record_category: string;
  retention_years: number;
  legal_basis: string;
  status: string;
  review_due_on: string;
  owner_name: string;
  notes: string;
}

export interface CreateGdprSubjectRequestRequest {
  subject_name: string;
  request_type: string;
  status: string;
  submitted_on: string;
  due_on: string;
  handled_by: string;
  source_module: string;
  anonymization_required: boolean;
  notes: string;
}

export interface CreateGdprSubjectExportRequest {
  request_id: string;
  subject_name: string;
  source_module: string;
  status: string;
  export_format: string;
  approved_by: string;
  approved_on: string;
  generated_on: string;
  package_summary: string;
  notes: string;
}

export interface CreateGdprPublicationReviewRequest {
  source_module: string;
  source_record_id: string;
  source_label: string;
  anonymization_status: string;
  publication_status: string;
  reviewed_by: string;
  reviewed_on: string;
  legal_basis: string;
  notes: string;
}

export interface SessionContext {
  user: {
    id: string;
    sub: string;
    name: string;
    email: string;
    phone_number?: string;
    locale: 'ro' | 'en';
    roles: string[];
  };
  institution_id: string;
  institution_name: string;
  permissions: string[];
  modules: Array<{
    code: string;
    active: boolean;
  }>;
  authentication: string[];
  gdpr_capabilities: string[];
}

export interface TableQuery {
  page: number;
  pageSize: number;
  sort?: string;
  direction?: 'asc' | 'desc';
  filters?: Record<string, string>;
}
