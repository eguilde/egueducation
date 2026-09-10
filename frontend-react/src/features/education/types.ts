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

/** Literal generated governance contracts; no locally widened meeting DTO. */
export type GovernanceMeeting = components["schemas"]["GovernanceMeeting"];
export type GovernanceMeetingInput = components["schemas"]["CreateGovernanceMeetingRequest"];
export type GovernanceDashboard = components["schemas"]["GovernanceDashboardResponse"];
export type DirectorCockpit = components["schemas"]["DirectorCockpitResponse"];
export type EligibleGovernanceUser = components["schemas"]["EligibleGovernanceUser"];

/**
 * Most Education domains intentionally expose their own record schema.  Keeping
 * a small lossless envelope here lets the UI render every authorised domain
 * without pretending that differently shaped dossiers are interchangeable.
 */
/**
 * Projection used only by generic related-resource/table UI. Root education
 * records deliberately use the generated DTO map below.
 */
export interface EducationRecord {
  id: string;
  [key: string]: unknown;
}

/** The closed set of independently versioned root CRUD contracts. */
export type EducationRecordsDomain =
  | "decisions"
  | "managerial"
  | "regulations"
  | "committees"
  | "personnel"
  | "evaluations"
  | "declarations"
  | "mobility"
  | "merit"
  | "portfolios"
  | "compliance";

/**
 * The root CRUD boundary is contract-first.  A domain cannot accidentally
 * borrow another domain's request or response DTO merely because both happen
 * to have an `id` property.
 */
export interface EducationRootRecordByDomain {
  decisions: components["schemas"]["GovernanceDecision"];
  managerial: components["schemas"]["ManagerialDossier"];
  regulations: components["schemas"]["RegulationRecord"];
  committees: components["schemas"]["CommitteeRecord"];
  personnel: components["schemas"]["PersonnelRecord"];
  evaluations: components["schemas"]["PersonnelEvaluation"];
  declarations: components["schemas"]["PersonnelDeclaration"];
  mobility: components["schemas"]["MobilityCase"];
  merit: components["schemas"]["MeritGrant"];
  portfolios: components["schemas"]["PortfolioRecord"];
  compliance: components["schemas"]["PublicationRecord"];
}

export interface EducationRootCreateInputByDomain {
  decisions: components["schemas"]["CreateGovernanceDecisionRequest"];
  managerial: components["schemas"]["CreateManagerialDossierRequest"];
  regulations: components["schemas"]["CreateRegulationRecordRequest"];
  committees: components["schemas"]["CreateCommitteeRecordRequest"];
  personnel: components["schemas"]["CreatePersonnelRecordRequest"];
  evaluations: components["schemas"]["CreatePersonnelEvaluationRequest"];
  declarations: components["schemas"]["CreatePersonnelDeclarationRequest"];
  mobility: components["schemas"]["CreateMobilityCaseRequest"];
  merit: components["schemas"]["CreateMeritGrantRequest"];
  portfolios: components["schemas"]["CreatePortfolioRecordRequest"];
  compliance: components["schemas"]["CreatePublicationRecordRequest"];
}

/** PATCH contracts are distinct where the OpenAPI operation says so. */
export interface EducationRootUpdateInputByDomain {
  decisions: components["schemas"]["CreateGovernanceDecisionRequest"];
  managerial: components["schemas"]["CreateManagerialDossierRequest"];
  regulations: components["schemas"]["CreateRegulationRecordRequest"];
  committees: components["schemas"]["CreateCommitteeRecordRequest"];
  personnel: components["schemas"]["CreatePersonnelRecordRequest"];
  evaluations: components["schemas"]["CreatePersonnelEvaluationRequest"];
  declarations: components["schemas"]["CreatePersonnelDeclarationRequest"];
  mobility: components["schemas"]["CreateMobilityCaseRequest"];
  merit: components["schemas"]["CreateMeritGrantRequest"];
  portfolios: components["schemas"]["UpdatePortfolioRecordRequest"];
  compliance: components["schemas"]["CreatePublicationRecordRequest"];
}

/** @deprecated Use the create/update maps at the root CRUD boundary. */
export type EducationRootInputByDomain = EducationRootCreateInputByDomain;
export type EducationRootRecord = EducationRootRecordByDomain[EducationRecordsDomain];
export type EducationRootRecordInput = EducationRootCreateInputByDomain[EducationRecordsDomain];
export type EducationPdfRecordsDomain =
  "managerial" | "evaluations" | "mobility" | "merit" | "portfolios";

/**
 * Closed frontend resource identifiers. These deliberately model the OpenAPI
 * surface, instead of allowing a UI component to manufacture a URL.  A
 * resource can have a parent identifier, but the route itself is selected by
 * this discriminant in the contract adapter.
 */
export type EducationRelatedResource =
  | "governance-memberships"
  | "governance-bodies"
  | "meeting-participants" | "meeting-documents" | "meeting-votes" | "meeting-minutes" | "meeting-resolutions"
  | "decision-issuances" | "decision-publication-steps"
  | "regulation-versions" | "regulation-workflow"
  | "committee-members"
  | "managerial-documents" | "managerial-workflow"
  | "personnel-assignments" | "personnel-file-documents" | "personnel-disciplinary-cases" | "personnel-access-events"
  | "evaluation-self-reviews" | "evaluation-criteria" | "evaluation-appeals" | "evaluation-result-issues"
  | "mobility-documents" | "mobility-scores" | "mobility-appeals" | "mobility-final-decisions" | "mobility-result-issues"
  | "merit-documents" | "merit-scores" | "merit-appeals" | "merit-final-decisions" | "merit-result-issues";

/**
 * Every related collection has its own OpenAPI request and response schema.
 * These maps are intentionally closed: a caller cannot accidentally post a
 * personnel document to an evaluation route (or submit server-owned output
 * fields through the generic relation editor).
 */
export interface EducationRelatedRecordByResource {
  "governance-memberships": components["schemas"]["GovernanceMembership"];
  "governance-bodies": components["schemas"]["GovernanceBodyRecord"];
  "meeting-participants": components["schemas"]["GovernanceMeetingParticipant"];
  "meeting-documents": components["schemas"]["GovernanceMeetingDocument"];
  "meeting-votes": components["schemas"]["GovernanceMeetingVote"];
  "meeting-minutes": components["schemas"]["GovernanceMinuteItem"];
  "meeting-resolutions": components["schemas"]["GovernanceResolution"];
  "decision-issuances": components["schemas"]["DecisionIssuance"];
  "decision-publication-steps": components["schemas"]["DecisionPublicationStep"];
  "regulation-versions": components["schemas"]["RegulationVersion"];
  "regulation-workflow": components["schemas"]["RegulationWorkflowStep"];
  "committee-members": components["schemas"]["CommitteeMember"];
  "managerial-documents": components["schemas"]["ManagerialDocument"];
  "managerial-workflow": components["schemas"]["ManagerialWorkflowStep"];
  "personnel-assignments": components["schemas"]["PersonnelAssignment"];
  "personnel-file-documents": components["schemas"]["PersonnelPersonalFileDocument"];
  "personnel-disciplinary-cases": components["schemas"]["PersonnelDisciplinaryCase"];
  "personnel-access-events": components["schemas"]["PersonnelPersonalAccessEvent"];
  "evaluation-self-reviews": components["schemas"]["PersonnelEvaluationSelfReview"];
  "evaluation-criteria": components["schemas"]["PersonnelEvaluationCriterion"];
  "evaluation-appeals": components["schemas"]["PersonnelEvaluationAppeal"];
  "evaluation-result-issues": components["schemas"]["PersonnelEvaluationResultIssue"];
  "mobility-documents": components["schemas"]["MobilityDocument"];
  "mobility-scores": components["schemas"]["MobilityCriterionScore"];
  "mobility-appeals": components["schemas"]["MobilityAppeal"];
  "mobility-final-decisions": components["schemas"]["MobilityFinalDecision"];
  "mobility-result-issues": components["schemas"]["MobilityResultIssue"];
  "merit-documents": components["schemas"]["MeritDocument"];
  "merit-scores": components["schemas"]["MeritCriterionScore"];
  "merit-appeals": components["schemas"]["MeritAppeal"];
  "merit-final-decisions": components["schemas"]["MeritFinalDecision"];
  "merit-result-issues": components["schemas"]["MeritResultIssue"];
}

export interface EducationRelatedCreateInputByResource {
  "governance-memberships": components["schemas"]["CreateGovernanceMembershipRequest"];
  "governance-bodies": never; // catalogue endpoint: no create operation
  "meeting-participants": components["schemas"]["CreateGovernanceMeetingParticipantRequest"];
  "meeting-documents": components["schemas"]["CreateGovernanceMeetingDocumentRequest"];
  "meeting-votes": components["schemas"]["CreateGovernanceMeetingVoteRequest"];
  "meeting-minutes": components["schemas"]["CreateGovernanceMinuteItemRequest"];
  "meeting-resolutions": components["schemas"]["CreateGovernanceResolutionRequest"];
  "decision-issuances": components["schemas"]["CreateDecisionIssuanceRequest"];
  "decision-publication-steps": components["schemas"]["CreateDecisionPublicationStepRequest"];
  "regulation-versions": components["schemas"]["CreateRegulationVersionRequest"];
  "regulation-workflow": components["schemas"]["CreateRegulationWorkflowStepRequest"];
  "committee-members": components["schemas"]["CreateCommitteeMemberRequest"];
  "managerial-documents": components["schemas"]["CreateManagerialDocumentRequest"];
  "managerial-workflow": components["schemas"]["CreateManagerialWorkflowStepRequest"];
  "personnel-assignments": components["schemas"]["CreatePersonnelAssignmentRequest"];
  "personnel-file-documents": components["schemas"]["CreatePersonnelPersonalFileDocumentRequest"];
  "personnel-disciplinary-cases": components["schemas"]["CreatePersonnelDisciplinaryCaseRequest"];
  "personnel-access-events": components["schemas"]["CreatePersonnelPersonalAccessEventRequest"];
  "evaluation-self-reviews": components["schemas"]["CreatePersonnelEvaluationSelfReviewRequest"];
  "evaluation-criteria": components["schemas"]["CreatePersonnelEvaluationCriterionRequest"];
  "evaluation-appeals": components["schemas"]["CreatePersonnelEvaluationAppealRequest"];
  "evaluation-result-issues": components["schemas"]["CreatePersonnelEvaluationResultIssueRequest"];
  "mobility-documents": components["schemas"]["CreateMobilityDocumentRequest"];
  "mobility-scores": components["schemas"]["CreateMobilityCriterionScoreRequest"];
  "mobility-appeals": components["schemas"]["CreateMobilityAppealRequest"];
  "mobility-final-decisions": components["schemas"]["CreateMobilityFinalDecisionRequest"];
  "mobility-result-issues": components["schemas"]["CreateMobilityResultIssueRequest"];
  "merit-documents": components["schemas"]["CreateMeritDocumentRequest"];
  "merit-scores": components["schemas"]["CreateMeritCriterionScoreRequest"];
  "merit-appeals": components["schemas"]["CreateMeritAppealRequest"];
  "merit-final-decisions": components["schemas"]["CreateMeritFinalDecisionRequest"];
  "merit-result-issues": components["schemas"]["CreateMeritResultIssueRequest"];
}

/** All mutable related operations use the same generated create/update body. */
export type EducationRelatedUpdateInputByResource = EducationRelatedCreateInputByResource;

export type EducationMetadataResource =
  | "director-cockpit" | "governance-meeting-filters" | "governance-meeting-finalization"
  | "governance-body-completeness"
  | "decisions-dashboard" | "decisions-filters" | "managerial-dashboard" | "managerial-filters"
  | "regulations-dashboard" | "regulations-filters" | "personnel-dashboard" | "personnel-filters"
  | "evaluations-dashboard" | "evaluations-filters" | "declarations-dashboard" | "declarations-filters"
  | "mobility-dashboard" | "mobility-filters" | "merit-dashboard" | "merit-filters"
  | "portfolios-dashboard" | "portfolios-filters"
  | "committee-completeness" | "managerial-portfolio-summary" | "personnel-portfolio-dossier-summary"
  | "portfolio-transfer-summary" | "regulation-procedural-summary";

/**
 * Closed response map for every metadata endpoint. This makes a new endpoint
 * an intentional contract addition rather than an untyped Record response.
 */
export interface EducationMetadataResultByResource {
  "director-cockpit": components["schemas"]["DirectorCockpitResponse"];
  "governance-meeting-filters": components["schemas"]["GovernanceFiltersResponse"];
  "governance-meeting-finalization": components["schemas"]["GovernanceMeetingFinalizationSummary"];
  "governance-body-completeness": components["schemas"]["EducationGovernanceBodyCompletenessSummaryResponse"];
  "decisions-dashboard": components["schemas"]["GovernanceDecisionDashboardResponse"];
  "decisions-filters": components["schemas"]["GovernanceDecisionFiltersResponse"];
  "managerial-dashboard": components["schemas"]["ManagerialDossierDashboardResponse"];
  "managerial-filters": components["schemas"]["ManagerialDossierFiltersResponse"];
  "regulations-dashboard": components["schemas"]["RegulationDashboardResponse"];
  "regulations-filters": components["schemas"]["RegulationFiltersResponse"];
  "personnel-dashboard": components["schemas"]["PersonnelDashboardResponse"];
  "personnel-filters": components["schemas"]["PersonnelFiltersResponse"];
  "evaluations-dashboard": components["schemas"]["PersonnelEvaluationDashboardResponse"];
  "evaluations-filters": components["schemas"]["PersonnelEvaluationFiltersResponse"];
  "declarations-dashboard": components["schemas"]["PersonnelDeclarationDashboardResponse"];
  "declarations-filters": components["schemas"]["PersonnelDeclarationFiltersResponse"];
  "mobility-dashboard": components["schemas"]["MobilityDashboardResponse"];
  "mobility-filters": components["schemas"]["MobilityFiltersResponse"];
  "merit-dashboard": components["schemas"]["MeritGrantDashboardResponse"];
  "merit-filters": components["schemas"]["MeritGrantFiltersResponse"];
  "portfolios-dashboard": components["schemas"]["PortfolioDashboardResponse"];
  "portfolios-filters": components["schemas"]["PortfolioFiltersResponse"];
  "committee-completeness": components["schemas"]["EducationCommitteeCompletenessResponse"];
  "managerial-portfolio-summary": components["schemas"]["EducationManagerialPortfolioSummaryResponse"];
  "personnel-portfolio-dossier-summary": components["schemas"]["EducationPersonnelPortfolioDossierSummaryResponse"];
  "portfolio-transfer-summary": components["schemas"]["EducationPortfolioTransferSummaryResponse"];
  "regulation-procedural-summary": components["schemas"]["EducationRegulationProceduralSummaryResponse"];
}

export type EducationCommand =
  | "portfolio-opis-regenerate" | "portfolio-return" | "portfolio-verify";

export interface EducationRecordInput {
  [key: string]: string | number | boolean | undefined;
}

/**
 * Portfolio subresources have intentionally separate generated contracts.
 * They must not pass through the generic related-record adapter: their
 * request bodies, filters and lifecycle permissions are not interchangeable.
 */
export type PortfolioDocument = components["schemas"]["PortfolioDocument"];
export type PortfolioChecklistItem = components["schemas"]["PortfolioChecklistItem"];
export type PortfolioOpisEntry = components["schemas"]["PortfolioOpisEntry"];
export type PortfolioCustodyEvent = components["schemas"]["PortfolioCustodyEvent"];
export type PortfolioReviewEvent = components["schemas"]["PortfolioReviewEvent"];
export type PortfolioTransferEvent = components["schemas"]["PortfolioTransferEvent"];
export type PortfolioValorificationEvent = components["schemas"]["PortfolioValorificationEvent"];
export type PortfolioSection = components["schemas"]["PortfolioSection"];
export type EducationRequirement = components["schemas"]["EducationRequirement"];
export type TaxonomyCatalog = components["schemas"]["TaxonomyCatalogResponse"];

export type CreatePortfolioDocumentInput = components["schemas"]["CreatePortfolioDocumentRequest"];
export type CreatePortfolioChecklistItemInput = components["schemas"]["CreatePortfolioChecklistItemRequest"];
export type CreatePortfolioOpisEntryInput = components["schemas"]["CreatePortfolioOpisEntryRequest"];
export type CreatePortfolioCustodyEventInput = components["schemas"]["CreatePortfolioCustodyEventRequest"];
export type CreatePortfolioReviewEventInput = components["schemas"]["CreatePortfolioReviewEventRequest"];
export type CreatePortfolioValorificationEventInput = components["schemas"]["CreatePortfolioValorificationEventRequest"];

export interface PortfolioDocumentListQuery { page?: number; pageSize?: number; sort?: string; direction?: "asc" | "desc"; sectionCode?: string; }
export interface PortfolioChecklistListQuery { page?: number; pageSize?: number; sort?: string; direction?: "asc" | "desc"; requirementCode?: string; }
export interface PortfolioOpisListQuery { page?: number; pageSize?: number; sort?: string; direction?: "asc" | "desc"; sectionCode?: string; }
export interface PortfolioCustodyListQuery { page?: number; pageSize?: number; sort?: string; direction?: "asc" | "desc"; eventType?: string; }
export interface PortfolioReviewListQuery { page?: number; pageSize?: number; sort?: string; direction?: "asc" | "desc"; reviewCode?: string; }
export interface PortfolioTransferHistoryQuery { page?: number; pageSize?: number; sort?: string; direction?: "asc" | "desc"; transferCode?: string; }
export interface PortfolioValorificationListQuery { page?: number; pageSize?: number; sort?: string; direction?: "asc" | "desc"; valorificationCode?: string; }
export interface PortfolioSectionListQuery { page?: number; pageSize?: number; sort?: string; direction?: "asc" | "desc"; sectionCode?: string; }
export interface EducationRequirementListQuery { page?: number; pageSize?: number; sort?: string; direction?: "asc" | "desc"; domain?: string; }
export interface TaxonomyCatalogQuery { domains?: string; }

/**
 * Deliberately narrow self-service contract for a teacher's professional
 * portfolio.  Identity, tenancy, custody and lifecycle ownership are never
 * client supplied: the active authenticated principal is authoritative.
 */
export type OwnPortfolioInput = components["schemas"]["OwnPortfolioRequest"];
export type OwnPortfolio = components["schemas"]["PortfolioRecord"];

export type PortfolioOpisRegeneration = components["schemas"]["EducationRegeneratePortfolioOpisResponse"];
export type PortfolioCessationInput = components["schemas"]["PortfolioCessationRequest"];
export type PortfolioLegalHoldInput = components["schemas"]["PortfolioLegalHoldRequest"];
export type PortfolioDeclarationType = "gdpr_information" | "authenticity";
export interface PortfolioDeclarationTemplate {
  declaration_type: PortfolioDeclarationType;
  declaration_version: string;
  declaration_text: string;
  source_ref: string;
  effective_from: string;
}
export type PortfolioDeclarationAcknowledgement = components["schemas"]["PortfolioDeclarationAcknowledgement"];
export type PortfolioDeclarationEvidence = components["schemas"]["PortfolioDeclarationEvidenceResponse"];
/** Generated OpenAPI procedure contracts; UI must not manufacture legal defaults. */
export type PortfolioProcedure = components["schemas"]["PortfolioProcedure"];
export type PortfolioProcedureRule = components["schemas"]["PortfolioProcedureSectionRule"];
export type PortfolioProcedureCreateInput = components["schemas"]["CreatePortfolioProcedureRequest"];
export type PortfolioProcedureUpdateInput = components["schemas"]["UpdatePortfolioProcedureRequest"];

export type OwnPortfolioDocumentInput = components["schemas"]["OwnPortfolioDocumentRequest"];
export type OwnPortfolioArchiveDocument = components["schemas"]["PortfolioArchiveAttachment"];
export type PortfolioAttachmentGrant = components["schemas"]["PortfolioArchiveAttachmentGrant"];
export type CreatePortfolioAttachmentGrant = components["schemas"]["CreatePortfolioArchiveAttachmentGrantRequest"];

export type PortfolioEvidenceManifestDocument = components["schemas"]["PortfolioExportManifestDocument"];
export type PortfolioEvidenceManifest = components["schemas"]["PortfolioExportManifest"];
export type PortfolioEvidenceManifestResponse = components["schemas"]["PortfolioExportManifestResponse"];

export interface EducationApi {
  governanceDashboard(): Promise<GovernanceDashboard>;
  directorCockpit(): Promise<DirectorCockpit>;
  eligibleGovernanceUsers(input?: { q?: string; page?: number; pageSize?: number }): Promise<EligibleGovernanceUser[]>;
  governanceMeetings(
    input?: EducationListQuery,
  ): Promise<EducationPage<GovernanceMeeting>>;
  governanceMeetingDetail(id: string): Promise<GovernanceMeeting>;
  saveGovernanceMeeting(
    input: GovernanceMeetingInput,
    id?: string,
  ): Promise<GovernanceMeeting>;
  deleteGovernanceMeeting(id: string): Promise<void>;
  records<D extends EducationRecordsDomain>(
    domain: D,
    input?: EducationListQuery,
  ): Promise<EducationPage<EducationRootRecordByDomain[D]>>;
  recordDetail<D extends EducationRecordsDomain>(
    domain: D,
    id: string,
  ): Promise<EducationRootRecordByDomain[D]>;
  createRecord<D extends EducationRecordsDomain>(
    domain: D,
    input: EducationRootCreateInputByDomain[D],
  ): Promise<EducationRootRecordByDomain[D]>;
  updateRecord<D extends EducationRecordsDomain>(
    domain: D,
    id: string,
    input: EducationRootUpdateInputByDomain[D],
  ): Promise<EducationRootRecordByDomain[D]>;
  deleteRecord(domain: EducationRecordsDomain, id: string): Promise<void>;
  recordPdf(domain: EducationPdfRecordsDomain, id: string): Promise<Blob>;
  relatedRecords(
    resource: EducationRelatedResource,
    parentID: string | undefined,
    input?: EducationListQuery,
  ): Promise<EducationPage<EducationRecord>>;
  relatedDetail(resource: EducationRelatedResource, parentID: string | undefined, id: string): Promise<EducationRecord>;
  saveRelated(
    resource: EducationRelatedResource,
    parentID: string | undefined,
    input: EducationRecordInput,
    id?: string,
  ): Promise<EducationRecord>;
  deleteRelated(resource: EducationRelatedResource, parentID: string | undefined, id: string): Promise<void>;
  relatedPdf(resource: EducationRelatedResource, parentID: string | undefined, id: string): Promise<Blob>;
  portfolioDocuments(recordID: string, input?: PortfolioDocumentListQuery): Promise<EducationPage<PortfolioDocument>>;
  portfolioDocument(recordID: string, documentID: string): Promise<PortfolioDocument>;
  createPortfolioDocument(recordID: string, input: CreatePortfolioDocumentInput): Promise<PortfolioDocument>;
  updatePortfolioDocument(recordID: string, documentID: string, input: CreatePortfolioDocumentInput): Promise<PortfolioDocument>;
  deletePortfolioDocument(recordID: string, documentID: string): Promise<void>;
  portfolioChecklist(recordID: string, input?: PortfolioChecklistListQuery): Promise<EducationPage<PortfolioChecklistItem>>;
  portfolioChecklistItem(recordID: string, itemID: string): Promise<PortfolioChecklistItem>;
  createPortfolioChecklistItem(recordID: string, input: CreatePortfolioChecklistItemInput): Promise<PortfolioChecklistItem>;
  updatePortfolioChecklistItem(recordID: string, itemID: string, input: CreatePortfolioChecklistItemInput): Promise<PortfolioChecklistItem>;
  deletePortfolioChecklistItem(recordID: string, itemID: string): Promise<void>;
  portfolioOpis(recordID: string, input?: PortfolioOpisListQuery): Promise<EducationPage<PortfolioOpisEntry>>;
  portfolioOpisEntry(recordID: string, itemID: string): Promise<PortfolioOpisEntry>;
  createPortfolioOpisEntry(recordID: string, input: CreatePortfolioOpisEntryInput): Promise<PortfolioOpisEntry>;
  updatePortfolioOpisEntry(recordID: string, itemID: string, input: CreatePortfolioOpisEntryInput): Promise<PortfolioOpisEntry>;
  deletePortfolioOpisEntry(recordID: string, itemID: string): Promise<void>;
  portfolioCustody(recordID: string, input?: PortfolioCustodyListQuery): Promise<EducationPage<PortfolioCustodyEvent>>;
  portfolioCustodyEvent(recordID: string, itemID: string): Promise<PortfolioCustodyEvent>;
  createPortfolioCustodyEvent(recordID: string, input: CreatePortfolioCustodyEventInput): Promise<PortfolioCustodyEvent>;
  updatePortfolioCustodyEvent(recordID: string, itemID: string, input: CreatePortfolioCustodyEventInput): Promise<PortfolioCustodyEvent>;
  deletePortfolioCustodyEvent(recordID: string, itemID: string): Promise<void>;
  portfolioReviews(recordID: string, input?: PortfolioReviewListQuery): Promise<EducationPage<PortfolioReviewEvent>>;
  portfolioReview(recordID: string, itemID: string): Promise<PortfolioReviewEvent>;
  createPortfolioReview(recordID: string, input: CreatePortfolioReviewEventInput): Promise<PortfolioReviewEvent>;
  updatePortfolioReview(recordID: string, itemID: string, input: CreatePortfolioReviewEventInput): Promise<PortfolioReviewEvent>;
  deletePortfolioReview(recordID: string, itemID: string): Promise<void>;
  portfolioTransferHistory(recordID: string, input?: PortfolioTransferHistoryQuery): Promise<EducationPage<PortfolioTransferEvent>>;
  portfolioValorifications(recordID: string, input?: PortfolioValorificationListQuery): Promise<EducationPage<PortfolioValorificationEvent>>;
  portfolioValorification(recordID: string, itemID: string): Promise<PortfolioValorificationEvent>;
  createPortfolioValorification(recordID: string, input: CreatePortfolioValorificationEventInput): Promise<PortfolioValorificationEvent>;
  updatePortfolioValorification(recordID: string, itemID: string, input: CreatePortfolioValorificationEventInput): Promise<PortfolioValorificationEvent>;
  deletePortfolioValorification(recordID: string, itemID: string): Promise<void>;
  portfolioSections(input?: PortfolioSectionListQuery): Promise<EducationPage<PortfolioSection>>;
  educationRequirements(input?: EducationRequirementListQuery): Promise<EducationPage<EducationRequirement>>;
  taxonomyCatalog(input?: TaxonomyCatalogQuery): Promise<TaxonomyCatalog>;
  metadata<R extends EducationMetadataResource>(resource: R, parentID?: string, itemID?: string): Promise<EducationMetadataResultByResource[R]>;
  command(command: EducationCommand, portfolioID: string): Promise<OwnPortfolio | PortfolioOpisRegeneration>;
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
  portfolioProcedureRules(id: string, input?: PortfolioProcedureRuleListQuery): Promise<EducationPage<PortfolioProcedureRule>>;
  replacePortfolioProcedureRules(id: string, input: { expected_updated_at: string; rules: PortfolioProcedureRule[] }): Promise<void>;
  transitionPortfolioProcedure(id: string, transition: "approve" | "publish" | "supersede" | "withdraw", input: { expected_updated_at: string; evidence: Record<string, unknown> }): Promise<PortfolioProcedure>;
  ownPortfolioRelated(id: string, resource: "documents" | "checklist" | "opis" | "reviews", input?: EducationListQuery): Promise<EducationPage<EducationRecord>>;
  regenerateOwnPortfolioOpis(id: string): Promise<PortfolioOpisRegeneration>;
  recordPortfolioCessation(id: string, input: PortfolioCessationInput): Promise<OwnPortfolio>;
  setPortfolioLegalHold(id: string, input: PortfolioLegalHoldInput): Promise<OwnPortfolio>;
  createOwnPortfolioDocument(id: string, input: OwnPortfolioDocumentInput): Promise<PortfolioDocument>;
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

/** Query contract declared by the procedure-section-rules OpenAPI operation. */
export interface PortfolioProcedureRuleListQuery extends EducationListQuery {
  sort?: "section_code" | "label_ro" | "sort_order";
  filters?: Partial<Record<"section_code" | "label_ro" | "sort_order", string>>;
}
