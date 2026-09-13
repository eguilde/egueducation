import type { ContractClient } from "../../api/client";
import type { components } from "../../api/generated";
import {
  validateGetApiAdmissionsAppealsResponse,
  validateGetApiAdmissionsApplicationsApplicationidResponse,
  validateGetApiAdmissionsApplicationsResponse,
  validateGetApiAdmissionsCampaignsCampaignidCriteriaResponse,
  validateGetApiAdmissionsCampaignsCampaignidDocumentRequirementsResponse,
  validateGetApiAdmissionsCampaignsResponse,
  validateGetApiAdmissionsCandidatePartiesResponse,
  validateGetApiAdmissionsClassOfferingContextsResponse,
  validateGetApiAdmissionsDecisionsResponse,
  validateGetApiAdmissionsDssRetentionPoliciesCurrentResponse,
  validateGetApiAdmissionsEligibleArchiveVersionsResponse,
  validateGetApiAdmissionsRegulatorySourcesResponse,
  validateGetApiAdmissionsStudentsResponse,
  validateGetApiEducationClassesResponse,
  validateGetApiInstitutionOfferingAuthorizationsResponse,
  validatePostApiAdmissionsApplicationsApplicationidAppealsResponse,
  validatePostApiAdmissionsApplicationsApplicationidAssessmentsResponse,
  validatePostApiAdmissionsApplicationsApplicationidDocumentsDocumentidResponse,
  validatePostApiAdmissionsApplicationsApplicationidEnrolmentResponse,
  validatePostApiAdmissionsApplicationsApplicationidTransitionsResponse,
  validatePostApiAdmissionsApplicationsResponse,
  validatePostApiAdmissionsCampaignsCampaignidCriteriaResponse,
  validatePostApiAdmissionsCampaignsCampaignidDocumentRequirementsResponse,
  validatePostApiAdmissionsCampaignsCampaignidTransitionsResponse,
  validatePostApiAdmissionsCampaignsResponse,
  validatePostApiAdmissionsClassOfferingContextsResponse,
  validatePostApiAdmissionsDssRetentionPoliciesResponse,
  validatePostApiAdmissionsRetentionRuleVersionsApproveResponse,
  validatePostApiAdmissionsRetentionRuleVersionsResponse,
  validateGetApiAdmissionsSignerAuthorizationsResponse,
  validatePostApiAdmissionsAppealsAppealidResolutionPreparationsResponse,
  validatePostApiAdmissionsApplicationsApplicationidDecisionPreparationsResponse,
  validatePostApiAdmissionsLegalPreparationsFinalizeResponse,
  validatePostApiAdmissionsLegalPreparationsPreparationidCancelResponse,
  validatePostApiAdmissionsSignerAuthorizationsApproveResponse,
  validatePostApiAdmissionsSignerAuthorizationsAuthorizationidRevokeResponse,
  validatePostApiAdmissionsSignerAuthorizationsResponse,
  validateGetApiAdmissionsLegalPreparationsPreparationidArtifactsArtifactslotResponse,
  validatePostApiAdmissionsLegalPreparationsPreparationidArtifactsArtifactslotResponse,
} from "../../api/runtime-validators";

/**
 * UI-facing admission API. The HTTP implementation uses generated OpenAPI
 * paths and runtime response validators; components inject this boundary for
 * focused tests. Response types should derive from their generated schemas.
 */
export type SortDirection = "asc" | "desc";
export type AdmissionQuery = { page: number; pageSize: number; sort: string; direction: SortDirection; filters: Record<string, string> };
export type AdmissionPage<T> = { items: T[]; total: number; page: number; pageSize: number };

export type AdmissionCampaign = {
  id: string; code: string; title: string; school_year: string; status: "draft" | "published" | "open" | "closed" | "cancelled" | "archived";
  source_id: string; offering_id: string; location_id: string; authorization_id: string; class_offering_context_id: string; capacity_limit: number; student_place_limit: number; capacity_unit: "students" | "study_groups"; capacity_basis: Record<string, unknown>; shift: "day" | "afternoon" | "evening"; opens_on: string; closes_on: string; decision_due_on?: string | null; expected_version: number;
};
export type AdmissionCriterion = { id: string; campaign_id: string; code: string; title: string; kind: "eligibility" | "priority" | "ranking" | "tie_breaker"; required: boolean; weight: number; ordinal: number; rule_snapshot: Record<string, unknown>; expected_version: number };
export type AdmissionDocumentRequirement = { id: string; code: string; title: string; required: boolean; allowed_mime_types: string[]; ordinal: number; expected_version: number };
export type AdmissionApplication = { id: string; application_no: string; campaign_id: string; candidate_party_id: string; student_id?: string | null; status: "draft" | "submitted" | "under_review" | "waitlisted" | "admitted" | "rejected" | "withdrawn" | "cancelled"; submitted_at?: string | null; consent_snapshot: Record<string, unknown>; expected_version: number; campaign_code: string; candidate_name: string; documents_complete: boolean; criteria_complete: boolean };
export type AdmissionDecision = { id: string; application_id: string; capacity_allocation_id?: string | null; policy_evaluation_v2_id: string; decision_no: string; outcome: "admitted" | "waitlisted" | "rejected" | "withdrawn" | "cancelled"; rationale: string; ranking_value?: number | null; decided_at: string; appeal_deadline?: string | null; application_no: string; candidate_name: string };
export type AdmissionAppeal = { id: string; application_id: string; decision_id: string; appeal_no: string; submitted_by_party_id?: string | null; status: "draft" | "submitted" | "under_review" | "resolved" | "withdrawn" | "dismissed" | "rejected_late"; submitted_at?: string | null; expected_version: number; application_no: string };
export type AdmissionCommandResult = { id: string; status: string; expected_version?: number; replayed?: boolean };
export type AdmissionDSSRetentionPolicy = { id: string; status: "active"; rule_version_id: string; minimum_retention_days: number; effective_from: string; source_id: string; replayed?: boolean };
export type AdmissionRetentionRuleVersion = { id: string; artifact_kind: "admission_dss"; status: "proposed" | "active" | "superseded" | "revoked"; minimum_retention_days: number; effective_from: string; effective_to?: string | null; source_id: string; source_checksum_sha256: string; proposed_by_subject: string; approved_by_subject?: string; replayed?: boolean };
export type AdmissionLookup = { id: string; label: string; disabled?: boolean };
export type AdmissionCampaignContext = AdmissionLookup & { class_id: string; school_year: string; effective_from: string; effective_to?: string | null; offering_id: string; location_id: string; authorization_id: string; class_offering_context_id: string; shift: "day" | "afternoon" | "evening" };
export type AdmissionAuthorizationOption = AdmissionLookup & { offering_id: string; location_id: string; shift: "day" | "afternoon" | "evening" };
export type CreateCampaignContextInput = { class_id: string; offering_id: string; location_id: string; authorization_id: string; school_year: string; shift: "day" | "afternoon" | "evening"; effective_from: string; effective_to?: string };
export type AdmissionArchiveVersion = AdmissionLookup & { document_id: string; version_id: string; version_no: number; original_file_name: string; mime_type: string; retention_until: string; sha256: string; eligible: boolean };
export type AdmissionApplicationDocument = { id: string; requirement_id?: string | null; document_kind: string; status: "requested" | "submitted" | "accepted" | "rejected" | "waived" | "withdrawn"; archive_document_id?: string | null; archive_version_id?: string | null; expected_version: number };
export type AdmissionCriterionAssessment = { id: string; criterion_id: string; code: string; title: string; required: boolean; outcome: "pending" | "met" | "not_met" | "not_applicable" | "indeterminate"; score?: number | null; rationale: string; evidence_snapshot: Record<string, unknown>; expected_version: number };
export type AdmissionApplicationDetail = { application: AdmissionApplication; assessments: AdmissionCriterionAssessment[]; documents: AdmissionApplicationDocument[]; decisions: AdmissionDecision[] };

export type CampaignInput = { code: string; title: string; school_year: string; source_id: string; capacity_limit: number; student_place_limit: number; capacity_unit: "students" | "study_groups"; capacity_basis: Record<string, unknown>; shift: "day" | "afternoon" | "evening"; opens_on: string; closes_on: string; decision_due_on?: string; offering_id: string; location_id: string; authorization_id: string; class_offering_context_id: string; criteria: Array<{ code: string; title: string; kind: AdmissionCriterion["kind"]; required: boolean; weight: number; ordinal: number; rule_snapshot: Record<string, unknown> }>; document_requirements: Array<{ code: string; title: string; required: boolean; allowed_mime_types: string[]; ordinal: number }> };
export type CriterionInput = { code: string; title: string; kind: AdmissionCriterion["kind"]; required: boolean; weight: number; ordinal: number; rule_snapshot: Record<string, unknown> };
export type DocumentRequirementInput = Omit<AdmissionDocumentRequirement, "id" | "expected_version">;
export type DecisionInput = { application_id: string; outcome: AdmissionDecision["outcome"]; rationale: string; ranking_value?: number; expected_version: number };
export type ArchiveVersionInput = Pick<AdmissionArchiveVersion, "document_id" | "version_id">;
export type ApplicationInput = { campaign_id: string; application_no: string; candidate_party_id: string; consent_snapshot: Record<string, unknown> };
export type FullDecisionInput = DecisionInput & { decision_no: string; appeal_deadline?: string; archive: ArchiveVersionInput };
// Derived from the backend DTO through OpenAPI, including immutable retention
// provenance and exact signing bytes; do not maintain a parallel hand-written shape.
export type AdmissionLegalPreparation = components["schemas"]["post_api_admissions_applications_applicationid_decision_preparations_response"];
/** The preparation-bound WORM record returned by both upload and recovery GET. */
export type AdmissionLegalPreparationArtifact = components["schemas"]["get_api_admissions_legal_preparations_preparationid_artifacts_artifactslot_response"];
export type AdmissionLegalArtifactSlot = "primary" | "resulting_decision";
export type AdmissionSignerAuthorization = {
  id: string;
  proposal_id?: string;
  certificate_sha256: string;
  user_id: string;
  actor_subject: string;
  permission_code: AdmissionSignerPermission;
  valid_until: string;
  proposed_by_subject: string;
  approved_by_subject?: string;
  status: "proposed" | "active" | "revoked" | "expired";
  expected_version: number;
  replayed?: boolean;
};
export type AdmissionSignerPermission = "education.admissions.decide" | "education.admissions.appeals.manage";
export type PrepareDecisionInput = Omit<FullDecisionInput, "application_id" | "archive">;
export type AdmissionResultingOutcome = "admitted" | "waitlisted" | "rejected" | "withdrawn" | "cancelled";
export type PrepareAppealResolutionInput = {
  outcome: "upheld" | "partially_upheld" | "dismissed" | "withdrawn";
  rationale: string;
  application_expected_version: number;
  expected_version: number;
  resulting_decision_no?: string;
  resulting_outcome?: AdmissionResultingOutcome;
};

export interface AdmissionApi {
  listCampaigns(query: AdmissionQuery): Promise<AdmissionPage<AdmissionCampaign>>;
  listCampaignContexts(query: { school_year?: string; q?: string }): Promise<AdmissionCampaignContext[]>;
  listClasses(query: { q?: string }): Promise<AdmissionLookup[]>;
  listAuthorizations(query: { q?: string }): Promise<AdmissionAuthorizationOption[]>;
  createCampaignContext(input: CreateCampaignContextInput): Promise<AdmissionCommandResult>;
  listRegulatorySources(query: { q?: string }): Promise<AdmissionLookup[]>;
  currentDSSRetentionPolicy(): Promise<AdmissionDSSRetentionPolicy>;
  proposeRetentionRule(input: { artifact_kind: "admission_dss"; minimum_retention_days: number; effective_from: string; effective_to: string | null; source_id: string }): Promise<AdmissionRetentionRuleVersion>;
  approveRetentionRule(input: { rule_version_id: string }): Promise<AdmissionRetentionRuleVersion>;
  configureDSSRetentionPolicy(input: { rule_version_id: string; effective_from: string }): Promise<AdmissionDSSRetentionPolicy>;
  createCampaign(input: CampaignInput): Promise<AdmissionCommandResult>;
  transitionCampaign(campaignID: string, input: { status: "published" | "open" | "closed" | "cancelled" | "archived"; expected_version: number }): Promise<AdmissionCommandResult>;
  listCriteria(campaignID: string, query: AdmissionQuery): Promise<AdmissionPage<AdmissionCriterion>>;
  addCriterion(campaignID: string, input: CriterionInput): Promise<AdmissionCommandResult>;
  listDocumentRequirements(campaignID: string, query: AdmissionQuery): Promise<AdmissionPage<AdmissionDocumentRequirement>>;
  addDocumentRequirement(campaignID: string, input: DocumentRequirementInput): Promise<AdmissionCommandResult>;
  listApplications(query: AdmissionQuery): Promise<AdmissionPage<AdmissionApplication>>;
  listCandidateParties(query: { q?: string }): Promise<AdmissionLookup[]>;
  listStudents(query: { q?: string }): Promise<AdmissionLookup[]>;
  createApplication(input: ApplicationInput): Promise<AdmissionCommandResult>;
  transitionApplication(applicationID: string, input: { status: "submitted" | "under_review"; expected_version: number }): Promise<AdmissionCommandResult>;
  getApplication(applicationID: string): Promise<AdmissionApplicationDetail>;
  listArchiveVersions(query: { q?: string; purpose: "application_document" | "decision" | "appeal" | "appeal_resolution" }): Promise<AdmissionArchiveVersion[]>;
  assessDocument(applicationID: string, documentID: string, input: { status: "accepted" | "rejected" | "waived"; review_note: string; expected_version: number; archive?: ArchiveVersionInput }): Promise<AdmissionCommandResult>;
  assessCriterion(applicationID: string, input: { criterion_id: string; outcome: Exclude<AdmissionCriterionAssessment["outcome"], "pending">; score?: number; rationale: string; evidence_snapshot: Record<string, unknown>; expected_version: number }): Promise<AdmissionCommandResult>;
  listDecisions(query: AdmissionQuery): Promise<AdmissionPage<AdmissionDecision>>;
  /** A legal decision is always prepared before a separately signed WORM version is finalized. */
  prepareDecision(applicationID: string, input: PrepareDecisionInput): Promise<AdmissionLegalPreparation>;
  finalizeDecision(applicationID: string, input: { preparation_id: string; archive: ArchiveVersionInput }): Promise<AdmissionCommandResult>;
  cancelLegalPreparation(preparationID: string): Promise<AdmissionCommandResult>;
  /** Returns undefined only when this preparation slot has not been committed yet. */
  getLegalPreparationArtifact(preparationID: string, artifactSlot: AdmissionLegalArtifactSlot, signal?: AbortSignal): Promise<AdmissionLegalPreparationArtifact | undefined>;
  /** Upload is idempotent per selected file; callers retain the key for retries. */
  uploadLegalPreparationArtifact(preparationID: string, artifactSlot: AdmissionLegalArtifactSlot, file: File, idempotencyKey: string): Promise<AdmissionLegalPreparationArtifact>;
  enrolApplication(applicationID: string, input: { student_code: string; enrolled_from: string; expected_version: number }): Promise<{ id: string; status: string; expected_version?: number }>;
  listAppeals(query: AdmissionQuery): Promise<AdmissionPage<AdmissionAppeal>>;
  createAppeal(applicationID: string, input: { decision_id: string; appeal_no: string; submitted_by_party_id: string; statement: string; archive?: ArchiveVersionInput }): Promise<AdmissionCommandResult>;
  prepareAppealResolution(appealID: string, input: PrepareAppealResolutionInput): Promise<AdmissionLegalPreparation>;
  finalizeAppealResolution(appealID: string, input: { preparation_id: string; archive: ArchiveVersionInput; resulting_decision_archive?: ArchiveVersionInput }): Promise<AdmissionCommandResult>;
  /** @deprecated Rendering no longer calls this legacy one-step operation. */
  resolveAppeal(appealID: string, input: { outcome: "upheld" | "partially_upheld" | "dismissed" | "withdrawn"; rationale: string; application_expected_version: number; resulting_decision_no?: string; resulting_outcome?: AdmissionResultingOutcome; expected_version: number; archive: ArchiveVersionInput }): Promise<AdmissionCommandResult>;
  listSignerAuthorizations(query: AdmissionQuery): Promise<AdmissionPage<AdmissionSignerAuthorization>>;
  proposeSignerAuthorization(input: { certificate_sha256: string; user_id: string; permission_code: AdmissionSignerPermission; valid_until: string }): Promise<AdmissionSignerAuthorization>;
  approveSignerAuthorization(input: { proposal_id: string }): Promise<AdmissionSignerAuthorization>;
  revokeSignerAuthorization(authorizationID: string, input: { expected_version: number; reason: string }): Promise<AdmissionCommandResult>;
}

type ContractResult = { data?: unknown; error?: unknown; response: Response };
type RuntimeValidator = (value: unknown) => boolean;

async function validated<T>(request: Promise<ContractResult>, validator: RuntimeValidator): Promise<T> {
  const result = await request;
  if (!result.response.ok || result.error || result.data === undefined) {
    const error = new Error(`admission_api_${result.response.status}`) as Error & { status?: number };
    error.status = result.response.status;
    throw error;
  }
  if (!validator(result.data)) throw new Error("admission_contract_mismatch");
  return result.data as T;
}

async function optionalValidated<T>(request: Promise<ContractResult>, validator: RuntimeValidator): Promise<T | undefined> {
  const result = await request;
  if (result.response.status === 404) return undefined;
  if (!result.response.ok || result.error || result.data === undefined) {
    const error = new Error(`admission_api_${result.response.status}`) as Error & { status?: number };
    error.status = result.response.status;
    throw error;
  }
  if (!validator(result.data)) throw new Error("admission_contract_mismatch");
  return result.data as T;
}

const clean = (value?: string) => value?.trim() || undefined;
const commandHeaders = () => ({ "Idempotency-Key": globalThis.crypto.randomUUID() });
const query = (value: AdmissionQuery) => ({
  page: value.page,
  pageSize: value.pageSize,
  sort: value.sort,
  direction: value.direction,
});

/** Generated OpenAPI is the only network boundary for Admission. */
export function createAdmissionApi(client: ContractClient): AdmissionApi {
  return {
    listCampaigns: (value) => validated(client.GET("/api/admissions/campaigns", { params: { query: { ...query(value), "filter.code": clean(value.filters.code), "filter.title": clean(value.filters.title), "filter.school_year": clean(value.filters.school_year), "filter.status": clean(value.filters.status) } } }), validateGetApiAdmissionsCampaignsResponse),
    listCampaignContexts: async (value) => {
      const page = await validated<AdmissionPage<{ id: string; class_id: string; offering_id: string; location_id: string; authorization_id: string; school_year: string; shift: AdmissionCampaignContext["shift"]; active: boolean; effective_from: string; effective_to?: string | null }>>(client.GET("/api/admissions/class-offering-contexts", { params: { query: { page: 1, pageSize: 100, sort: "school_year", direction: "desc", "filter.school_year": clean(value.school_year) } } }), validateGetApiAdmissionsClassOfferingContextsResponse);
      const needle = value.q?.trim().toLocaleLowerCase("ro") ?? "";
      return page.items.filter((item) => item.active && (!needle || `${item.school_year} ${item.shift} ${item.class_id}`.toLocaleLowerCase("ro").includes(needle))).map((item) => ({ ...item, class_offering_context_id: item.id, label: `${item.school_year} · ${item.shift} · ${item.class_id.slice(0, 8)}` }));
    },
    listClasses: async (value) => {
      const page = await validated<AdmissionPage<{ id: string; class_code: string; class_name: string; school_year: string; active: boolean }>>(client.GET("/api/education/classes", { params: { query: { page: 1, pageSize: 100, sort: "class_name", direction: "asc", "filter.class_name": clean(value.q) } } }), validateGetApiEducationClassesResponse);
      return page.items.filter((item) => item.active).map((item) => ({ id: item.id, label: `${item.class_code} · ${item.class_name} · ${item.school_year}` }));
    },
    listAuthorizations: async (value) => {
      const page = await validated<AdmissionPage<{ id: string; offering_id: string; offering_code: string; offering_title: string; location_id: string; location_code: string; location_name: string; status: string; shift: AdmissionAuthorizationOption["shift"] }>>(client.GET("/api/institution/offering-authorizations", { params: { query: { page: 1, pageSize: 100, sort: "effective_from", direction: "desc", "filter.offering_code": clean(value.q) } } }), validateGetApiInstitutionOfferingAuthorizationsResponse);
      return page.items.filter((item) => item.status === "provisional" || item.status === "accredited").map((item) => ({ id: item.id, offering_id: item.offering_id, location_id: item.location_id, shift: item.shift, label: `${item.offering_code} · ${item.offering_title} · ${item.location_code} — ${item.location_name}` }));
    },
    createCampaignContext: (body) => validated(client.POST("/api/admissions/class-offering-contexts", { params: { header: commandHeaders() }, body }), validatePostApiAdmissionsClassOfferingContextsResponse),
    listRegulatorySources: async (value) => {
      const page = await validated<AdmissionPage<{ id: string; citation: string; source_kind: string; issuer: string }>>(client.GET("/api/admissions/regulatory-sources", { params: { query: { page: 1, pageSize: 100, sort: "citation", direction: "asc", "filter.citation": clean(value.q) } } }), validateGetApiAdmissionsRegulatorySourcesResponse);
      return page.items.map((item) => ({ id: item.id, label: `${item.citation}${item.issuer ? ` · ${item.issuer}` : ""}` }));
    },
    currentDSSRetentionPolicy: () => validated(client.GET("/api/admissions/dss-retention-policies/current"), validateGetApiAdmissionsDssRetentionPoliciesCurrentResponse),
    proposeRetentionRule: (body) => validated(client.POST("/api/admissions/retention-rule-versions", { params: { header: commandHeaders() }, body }), validatePostApiAdmissionsRetentionRuleVersionsResponse),
    approveRetentionRule: (body) => validated(client.POST("/api/admissions/retention-rule-versions/approve", { params: { header: commandHeaders() }, body }), validatePostApiAdmissionsRetentionRuleVersionsApproveResponse),
    configureDSSRetentionPolicy: (body) => validated(client.POST("/api/admissions/dss-retention-policies", { params: { header: commandHeaders() }, body }), validatePostApiAdmissionsDssRetentionPoliciesResponse),
    createCampaign: (body) => validated(client.POST("/api/admissions/campaigns", { params: { header: commandHeaders() }, body }), validatePostApiAdmissionsCampaignsResponse),
    transitionCampaign: (campaignID, body) => validated(client.POST("/api/admissions/campaigns/{campaignID}/transitions", { params: { path: { campaignID }, header: commandHeaders() }, body }), validatePostApiAdmissionsCampaignsCampaignidTransitionsResponse),
    listCriteria: (campaignID, value) => validated(client.GET("/api/admissions/campaigns/{campaignID}/criteria", { params: { path: { campaignID }, query: { ...query(value), "filter.code": clean(value.filters.code), "filter.title": clean(value.filters.title), "filter.kind": clean(value.filters.kind), "filter.required": clean(value.filters.required) } } }), validateGetApiAdmissionsCampaignsCampaignidCriteriaResponse),
    addCriterion: (campaignID, body) => validated(client.POST("/api/admissions/campaigns/{campaignID}/criteria", { params: { path: { campaignID }, header: commandHeaders() }, body }), validatePostApiAdmissionsCampaignsCampaignidCriteriaResponse),
    listDocumentRequirements: (campaignID, value) => validated(client.GET("/api/admissions/campaigns/{campaignID}/document-requirements", { params: { path: { campaignID }, query: { ...query(value), "filter.code": clean(value.filters.code), "filter.title": clean(value.filters.title), "filter.required": clean(value.filters.required) } } }), validateGetApiAdmissionsCampaignsCampaignidDocumentRequirementsResponse),
    addDocumentRequirement: (campaignID, body) => validated(client.POST("/api/admissions/campaigns/{campaignID}/document-requirements", { params: { path: { campaignID }, header: commandHeaders() }, body }), validatePostApiAdmissionsCampaignsCampaignidDocumentRequirementsResponse),
    listApplications: (value) => validated(client.GET("/api/admissions/applications", { params: { query: { ...query(value), "filter.campaign_id": clean(value.filters.campaign_id), "filter.application_no": clean(value.filters.application_no), "filter.status": clean(value.filters.status) } } }), validateGetApiAdmissionsApplicationsResponse),
    listCandidateParties: async (value) => {
      const page = await validated<AdmissionPage<{ id: string; code: string; display_name: string }>>(client.GET("/api/admissions/candidate-parties", { params: { query: { page: 1, pageSize: 100, sort: "display_name", direction: "asc", "filter.display_name": clean(value.q) } } }), validateGetApiAdmissionsCandidatePartiesResponse);
      return page.items.map((item) => ({ id: item.id, label: `${item.display_name}${item.code ? ` · ${item.code}` : ""}` }));
    },
    listStudents: async (value) => {
      const page = await validated<AdmissionPage<{ id: string; student_code: string; first_name: string; last_name: string; status: string }>>(client.GET("/api/admissions/students", { params: { query: { page: 1, pageSize: 100, sort: "last_name", direction: "asc", "filter.last_name": clean(value.q) } } }), validateGetApiAdmissionsStudentsResponse);
      return page.items.map((item) => ({ id: item.id, label: `${item.last_name} ${item.first_name} · ${item.student_code}` }));
    },
    createApplication: (body) => validated(client.POST("/api/admissions/applications", { params: { header: commandHeaders() }, body }), validatePostApiAdmissionsApplicationsResponse),
    transitionApplication: (applicationID, body) => validated(client.POST("/api/admissions/applications/{applicationID}/transitions", { params: { path: { applicationID }, header: commandHeaders() }, body }), validatePostApiAdmissionsApplicationsApplicationidTransitionsResponse),
    getApplication: (applicationID) => validated(client.GET("/api/admissions/applications/{applicationID}", { params: { path: { applicationID } } }), validateGetApiAdmissionsApplicationsApplicationidResponse),
    listArchiveVersions: async (value) => {
      const page = await validated<AdmissionPage<{ document_id: string; version_id: string; version_no: number; title: string; original_file_name: string; mime_type: string; sha256: string; retention_until: string }>>(client.GET("/api/admissions/eligible-archive-versions", { params: { query: { purpose: value.purpose, q: clean(value.q), page: 1, pageSize: 100, sort: "title", direction: "asc" } } }), validateGetApiAdmissionsEligibleArchiveVersionsResponse);
      return page.items.map((item) => ({ ...item, id: item.version_id, label: `${item.title} · v${item.version_no}`, eligible: true }));
    },
    assessDocument: (applicationID, documentID, body) => validated(client.POST("/api/admissions/applications/{applicationID}/documents/{documentID}", { params: { path: { applicationID, documentID }, header: commandHeaders() }, body }), validatePostApiAdmissionsApplicationsApplicationidDocumentsDocumentidResponse),
    assessCriterion: (applicationID, body) => validated(client.POST("/api/admissions/applications/{applicationID}/assessments", { params: { path: { applicationID }, header: commandHeaders() }, body }), validatePostApiAdmissionsApplicationsApplicationidAssessmentsResponse),
    listDecisions: (value) => validated(client.GET("/api/admissions/decisions", { params: { query: { ...query(value), "filter.application_id": clean(value.filters.application_id), "filter.decision_no": clean(value.filters.decision_no), "filter.outcome": clean(value.filters.outcome) } } }), validateGetApiAdmissionsDecisionsResponse),
    prepareDecision: (applicationID, body) => validated(client.POST("/api/admissions/applications/{applicationID}/decision-preparations", { params: { path: { applicationID }, header: commandHeaders() }, body }), validatePostApiAdmissionsApplicationsApplicationidDecisionPreparationsResponse),
    finalizeDecision: (_applicationID, body) => validated(client.POST("/api/admissions/legal-preparations/finalize", { params: { header: commandHeaders() }, body }), validatePostApiAdmissionsLegalPreparationsFinalizeResponse),
    cancelLegalPreparation: (preparationID) => validated(client.POST("/api/admissions/legal-preparations/{preparationID}/cancel", { params: { path: { preparationID }, header: commandHeaders() } }), validatePostApiAdmissionsLegalPreparationsPreparationidCancelResponse),
    getLegalPreparationArtifact: (preparationID, artifactSlot, signal) => optionalValidated(client.GET("/api/admissions/legal-preparations/{preparationID}/artifacts/{artifactSlot}", { params: { path: { preparationID, artifactSlot } }, signal }), validateGetApiAdmissionsLegalPreparationsPreparationidArtifactsArtifactslotResponse),
    uploadLegalPreparationArtifact: (preparationID, artifactSlot, file, idempotencyKey) => {
      const body = new FormData();
      body.set("file", file, file.name);
      return validated(client.POST("/api/admissions/legal-preparations/{preparationID}/artifacts/{artifactSlot}", { params: { path: { preparationID, artifactSlot }, header: { "Idempotency-Key": idempotencyKey } }, body: body as never }), validatePostApiAdmissionsLegalPreparationsPreparationidArtifactsArtifactslotResponse);
    },
    enrolApplication: (applicationID, body) => validated(client.POST("/api/admissions/applications/{applicationID}/enrolment", { params: { path: { applicationID }, header: commandHeaders() }, body }), validatePostApiAdmissionsApplicationsApplicationidEnrolmentResponse),
    listAppeals: (value) => validated(client.GET("/api/admissions/appeals", { params: { query: { ...query(value), "filter.application_id": clean(value.filters.application_id), "filter.appeal_no": clean(value.filters.appeal_no), "filter.status": clean(value.filters.status) } } }), validateGetApiAdmissionsAppealsResponse),
    createAppeal: (applicationID, body) => validated(client.POST("/api/admissions/applications/{applicationID}/appeals", { params: { path: { applicationID }, header: commandHeaders() }, body }), validatePostApiAdmissionsApplicationsApplicationidAppealsResponse),
    prepareAppealResolution: (appealID, input) => validated(client.POST("/api/admissions/appeals/{appealID}/resolution-preparations", { params: { path: { appealID }, header: commandHeaders() }, body: input.resulting_decision_no && input.resulting_outcome ? input : { outcome: input.outcome, rationale: input.rationale, expected_version: input.expected_version, application_expected_version: input.application_expected_version } }), validatePostApiAdmissionsAppealsAppealidResolutionPreparationsResponse),
    finalizeAppealResolution: (_appealID, body) => validated(client.POST("/api/admissions/legal-preparations/finalize", { params: { header: commandHeaders() }, body }), validatePostApiAdmissionsLegalPreparationsFinalizeResponse),
    resolveAppeal: async () => { throw new Error("admission_legacy_one_step_resolution_disabled"); },
    listSignerAuthorizations: (value) => validated(client.GET("/api/admissions/signer-authorizations", { params: { query: { ...query(value), "filter.actor_subject": clean(value.filters.actor_subject), "filter.permission_code": clean(value.filters.permission_code), "filter.status": clean(value.filters.status) } } }), validateGetApiAdmissionsSignerAuthorizationsResponse),
    proposeSignerAuthorization: (body) => validated(client.POST("/api/admissions/signer-authorizations", { params: { header: commandHeaders() }, body }), validatePostApiAdmissionsSignerAuthorizationsResponse),
    approveSignerAuthorization: (body) => validated(client.POST("/api/admissions/signer-authorizations/approve", { params: { header: commandHeaders() }, body }), validatePostApiAdmissionsSignerAuthorizationsApproveResponse),
    revokeSignerAuthorization: (authorizationID, body) => validated(client.POST("/api/admissions/signer-authorizations/{authorizationID}/revoke", { params: { path: { authorizationID }, header: commandHeaders() }, body }), validatePostApiAdmissionsSignerAuthorizationsAuthorizationidRevokeResponse),
  };
}
