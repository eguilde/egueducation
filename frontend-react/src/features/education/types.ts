import type { components } from "../../api/generated";

export interface EducationModule {
  code: string;
  active: boolean;
}

export interface EducationArea {
  id: string;
  label: string;
  icon: string;
  permissions: string[];
  module?: string;
  description: string;
}

export interface EducationPage<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
}

export interface GovernanceMeeting {
  id: string;
  school_year: string;
  organism: string;
  title: string;
  meeting_type: string;
  status: string;
  meeting_date: string;
  location: string;
  chairperson: string;
  secretary_name: string;
  chairperson_user_id: string;
  secretary_user_id: string;
}

export interface GovernanceDashboard {
  stats: {
    total_meetings: number;
    scheduled_meetings: number;
    held_meetings: number;
    published_meetings: number;
  };
}

export interface DirectorCockpit {
  [key: string]: unknown;
}
export type EligibleGovernanceUser = components["schemas"]["EligibleGovernanceUser"];

/**
 * Most Education domains intentionally expose their own record schema.  Keeping
 * a small lossless envelope here lets the UI render every authorised domain
 * without pretending that differently shaped dossiers are interchangeable.
 */
export interface EducationRecord {
  id: string;
  [key: string]: unknown;
}

export type EducationRecordsDomain = Exclude<
  EducationArea["id"],
  "overview" | "governance"
>;
export type EducationPdfRecordsDomain =
  "managerial" | "evaluations" | "mobility" | "merit" | "portfolios";

export interface EducationRecordInput {
  [key: string]: string | number | boolean | undefined;
}

/**
 * Deliberately narrow self-service contract for a teacher's professional
 * portfolio.  Identity, tenancy, custody and lifecycle ownership are never
 * client supplied: the active authenticated principal is authoritative.
 */
export interface OwnPortfolioInput {
  school_year: string;
  last_updated_on: string;
  notes: string;
}

type GeneratedPortfolio = components["schemas"]["PortfolioRecord"];
export type OwnPortfolio = OwnPortfolioInput & Required<Pick<GeneratedPortfolio,
  "id" | "portfolio_code" | "owner_name" | "owner_role" | "status" | "transfer_status" |
  "section_count" | "authenticity_declared" | "consent_captured"
>>;

export type PortfolioOpisRegeneration = components["schemas"]["EducationRegeneratePortfolioOpisResponse"];
export interface PortfolioCessationInput { activity_ceased_on: string; reason: string }
export interface PortfolioLegalHoldInput { active: boolean; reason: string }
export type PortfolioDeclarationType = "gdpr_information" | "authenticity";
export interface PortfolioDeclarationTemplate {
  declaration_type: PortfolioDeclarationType;
  declaration_version: string;
  declaration_text: string;
  source_ref: string;
  effective_from: string;
}
export interface PortfolioDeclarationAcknowledgement {
  id: string;
  portfolio_id: string;
  declaration_type: PortfolioDeclarationType;
  declaration_version: string;
  declaration_text: string;
  accepted_at: string;
  accepted_by_user_id: string;
  attestation_method: string;
}
export interface PortfolioDeclarationEvidence {
  templates: PortfolioDeclarationTemplate[];
  acknowledgements: PortfolioDeclarationAcknowledgement[];
}
/** Generated OpenAPI procedure contracts; UI must not manufacture legal defaults. */
export type PortfolioProcedure = components["schemas"]["PortfolioProcedure"];
export type PortfolioProcedureRule = components["schemas"]["PortfolioProcedureSectionRule"];
export type PortfolioProcedureCreateInput = components["schemas"]["CreatePortfolioProcedureRequest"];
export type PortfolioProcedureUpdateInput = components["schemas"]["UpdatePortfolioProcedureRequest"];

type GeneratedOwnPortfolioDocumentInput = components["schemas"]["OwnPortfolioDocumentRequest"];
export type OwnPortfolioDocumentInput = Required<Pick<GeneratedOwnPortfolioDocumentInput,
  "section_code" | "component_code" | "document_title" | "evidence_type" |
  "issued_on" | "added_on" | "chronological_index" | "sensitive_data" | "file_reference" | "notes"
>>;
export type OwnPortfolioArchiveDocument = components["schemas"]["PortfolioArchiveAttachment"];
export type PortfolioAttachmentGrant = components["schemas"]["PortfolioArchiveAttachmentGrant"];
export type CreatePortfolioAttachmentGrant = components["schemas"]["CreatePortfolioArchiveAttachmentGrantRequest"];

export type PortfolioEvidenceManifestDocument = {
  evidence_record_id: string;
  section_code: string;
  component_code: string;
  document_title: string;
  chronological_no: number;
  issued_on: string;
  evidence_type: string;
  archive_document_id: string;
  archive_version_id: string;
  archive_version_no: number;
  source_bucket: string;
  source_object_key: string;
  source_sha256: string;
};

export type PortfolioEvidenceManifest = {
  manifest_version: string;
  hash_algorithm: string;
  manifest_sha256: string;
  tenant_code: string;
  institution_id: string;
  portfolio: {
    id: string;
    portfolio_code: string;
    school_year: string;
    status: string;
    applied_procedure_id?: string;
  };
  documents: PortfolioEvidenceManifestDocument[];
};

export type PortfolioEvidenceManifestResponse = {
  export_manifest_id: string;
  generated_at: string;
  manifest: PortfolioEvidenceManifest;
};

export interface EducationApi {
  governanceDashboard(): Promise<GovernanceDashboard>;
  directorCockpit(): Promise<DirectorCockpit>;
  eligibleGovernanceUsers(): Promise<EligibleGovernanceUser[]>;
  dashboardAt(path: string): Promise<Record<string, unknown>>;
  governanceMeetings(
    input?: EducationListQuery,
  ): Promise<EducationPage<GovernanceMeeting>>;
  governanceMeetingDetail(id: string): Promise<GovernanceMeeting>;
  saveGovernanceMeeting(
    input: EducationRecordInput,
    id?: string,
  ): Promise<GovernanceMeeting>;
  deleteGovernanceMeeting(id: string): Promise<void>;
  records(
    domain: EducationRecordsDomain,
    input?: EducationListQuery,
  ): Promise<EducationPage<EducationRecord>>;
  recordDetail(
    domain: EducationRecordsDomain,
    id: string,
  ): Promise<EducationRecord>;
  saveRecord(
    domain: EducationRecordsDomain,
    input: EducationRecordInput,
    id?: string,
  ): Promise<EducationRecord>;
  deleteRecord(domain: EducationRecordsDomain, id: string): Promise<void>;
  recordPdf(domain: EducationPdfRecordsDomain, id: string): Promise<Blob>;
  relatedRecords(
    path: string,
    input?: EducationListQuery,
  ): Promise<EducationPage<EducationRecord>>;
  relatedDetail(path: string, id: string): Promise<EducationRecord>;
  saveRelated(
    path: string,
    input: EducationRecordInput,
    id?: string,
  ): Promise<EducationRecord>;
  deleteRelated(path: string, id: string): Promise<void>;
  relatedPdf(path: string, id: string): Promise<Blob>;
  metadata(path: string): Promise<Record<string, unknown>>;
  command(path: string): Promise<void>;
  ownPortfolios(input?: EducationListQuery): Promise<EducationPage<OwnPortfolio>>;
  ownPortfolio(id: string): Promise<OwnPortfolio>;
  createOwnPortfolio(input: OwnPortfolioInput): Promise<OwnPortfolio>;
  updateOwnPortfolio(id: string, input: OwnPortfolioInput): Promise<OwnPortfolio>;
  submitOwnPortfolio(id: string): Promise<OwnPortfolio>;
  ownPortfolioDeclarations(id: string): Promise<PortfolioDeclarationEvidence>;
  acknowledgeOwnPortfolioDeclaration(id: string, declarationType: PortfolioDeclarationType): Promise<PortfolioDeclarationAcknowledgement>;
  portfolioProcedures(input?: EducationListQuery): Promise<EducationPage<PortfolioProcedure>>;
  portfolioProcedure(id: string): Promise<PortfolioProcedure>;
  createPortfolioProcedure(input: PortfolioProcedureCreateInput): Promise<PortfolioProcedure>;
  updatePortfolioProcedure(id: string, input: PortfolioProcedureUpdateInput): Promise<PortfolioProcedure>;
  portfolioProcedureRules(id: string): Promise<EducationPage<PortfolioProcedureRule>>;
  replacePortfolioProcedureRules(id: string, input: { expected_updated_at: string; rules: PortfolioProcedureRule[] }): Promise<void>;
  transitionPortfolioProcedure(id: string, transition: "approve" | "publish" | "supersede" | "withdraw", input: { expected_updated_at: string; evidence: Record<string, unknown> }): Promise<PortfolioProcedure>;
  ownPortfolioRelated(id: string, resource: "documents" | "checklist" | "opis" | "reviews", input?: EducationListQuery): Promise<EducationPage<EducationRecord>>;
  regenerateOwnPortfolioOpis(id: string): Promise<PortfolioOpisRegeneration>;
  recordPortfolioCessation(id: string, input: PortfolioCessationInput): Promise<OwnPortfolio>;
  setPortfolioLegalHold(id: string, input: PortfolioLegalHoldInput): Promise<OwnPortfolio>;
  createOwnPortfolioDocument(id: string, input: OwnPortfolioDocumentInput): Promise<EducationRecord>;
  deleteOwnPortfolioDocument(portfolioID: string, documentID: string): Promise<void>;
  ownPortfolioArchiveDocuments(input?: EducationListQuery): Promise<EducationPage<OwnPortfolioArchiveDocument>>;
  attachmentGrants(input?: EducationListQuery): Promise<EducationPage<PortfolioAttachmentGrant>>;
  eligibleAttachmentDocuments(input?: EducationListQuery): Promise<EducationPage<OwnPortfolioArchiveDocument>>;
  eligibleAttachmentUsers(input?: EducationListQuery): Promise<EducationPage<EligibleGovernanceUser>>;
  createAttachmentGrant(input: CreatePortfolioAttachmentGrant): Promise<PortfolioAttachmentGrant>;
  deleteAttachmentGrant(id: string): Promise<void>;
  createPortfolioExportManifest(id: string): Promise<PortfolioEvidenceManifestResponse>;
}

export interface EducationListQuery {
  page?: number;
  pageSize?: number;
  sort?: string;
  direction?: "asc" | "desc";
  q?: string;
  filters?: Record<string, string | undefined>;
}
