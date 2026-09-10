import { createContractClient, type ContractClient } from "../../api/client";
import type { components, paths } from "../../api/generated";
import type { CreatePortfolioChecklistItemInput, CreatePortfolioCustodyEventInput, CreatePortfolioDocumentInput, CreatePortfolioOpisEntryInput, CreatePortfolioReviewEventInput, CreatePortfolioValorificationEventInput, DirectorCockpit, EducationApi, EducationCommand, EducationListQuery, EducationMetadataResource, EducationMetadataResultByResource, EducationPage, EducationPdfRecordsDomain, EducationRecord, EducationRecordInput, EducationRecordsDomain, EducationRelatedCreateInputByResource, EducationRelatedResource, EducationRequirementListQuery, EducationRootCreateInputByDomain, EducationRootRecordByDomain, EducationRootUpdateInputByDomain, EligibleGovernanceUser, GovernanceDashboard, GovernanceMeeting, GovernanceMeetingInput, OwnPortfolio, OwnPortfolioArchiveDocument, PortfolioAttachmentGrant, PortfolioChecklistListQuery, PortfolioCustodyListQuery, PortfolioDeclarationAcknowledgement, PortfolioDeclarationEvidence, PortfolioDocument, PortfolioDocumentListQuery, PortfolioEvidenceManifestResponse, PortfolioOpisListQuery, PortfolioOpisRegeneration, PortfolioProcedure, PortfolioProcedureRule, PortfolioReviewListQuery, PortfolioSectionListQuery, PortfolioTransferHistoryQuery, PortfolioValorificationListQuery, TaxonomyCatalogQuery } from "./types";

export type AuthenticatedFetcher = (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>;
type Result<T> = { data?: T; error?: unknown; response: Response };
type Parameters = Record<string, string>;
type Query = Record<string, string | number | undefined>;

function object(value: unknown): Record<string, unknown> { if (!value || typeof value !== "object" || Array.isArray(value)) throw new Error("education_contract_response_invalid"); return value as Record<string, unknown>; }
function record(value: unknown): EducationRecord { const item = object(value); if (typeof item.id !== "string") throw new Error("education_contract_record_id_missing"); return { ...item, id: item.id }; }
function page(value: unknown): EducationPage<EducationRecord> {
  if (Array.isArray(value)) return { items: value.map(record), total: value.length, page: 1, pageSize: value.length || 1 };
  const result = object(value); const source = result.items;
  const items = Array.isArray(source) ? source : source && typeof source === "object" ? Object.values(source).flatMap((group) => Array.isArray(group) ? group : []) : [];
  return { items: items.map(record), total: typeof result.total === "number" ? result.total : items.length, page: typeof result.page === "number" ? result.page : 1, pageSize: typeof result.pageSize === "number" ? result.pageSize : Math.max(items.length, 1) };
}
function typedRecord<T extends { id: string }>(value: unknown): T { return record(value) as T; }
function typedPage<T extends { id: string }>(value: unknown): EducationPage<T> { const result = page(value); return { ...result, items: result.items.map((item) => item as T) }; }
function typedObject<T>(value: unknown): T { return object(value) as T; }
async function unwrap<T>(promise: Promise<Result<T>>): Promise<T> { const result = await promise; if (!result.response.ok || result.error) throw new Error(`education_request_${result.response.status}`); return result.data as T; }
function query(input: EducationListQuery = {}): Query { const result: Query = { page: input.page ?? 1, pageSize: input.pageSize ?? 50, sort: input.sort, direction: input.direction, q: input.q?.trim() || undefined }; Object.entries(input.filters ?? {}).forEach(([key, value]) => { if (value?.trim()) result[`filter.${key}`] = value.trim(); }); return result; }

const portfolioPageParams = <Sort extends string>(input: { page?: number; pageSize?: number; sort?: Sort; direction?: "asc" | "desc" } = {}) => ({ page: input.page ?? 1, pageSize: input.pageSize ?? 50, sort: input.sort, direction: input.direction });
const portfolioDocumentParams = (input: PortfolioDocumentListQuery = {}) => ({ page: input.page ?? 1, pageSize: input.pageSize ?? 50, sort: input.sort === "section_code" ? "section_code" : input.sort === "document_title" ? "document_title" : input.sort === "evidence_type" ? "evidence_type" : input.sort === "authenticity_status" ? "authenticity_status" : input.sort === "issued_on" ? "issued_on" : undefined, direction: input.direction, "filter.section_code": input.sectionCode } as const);
const portfolioChecklistParams = (input: PortfolioChecklistListQuery = {}) => ({ page: input.page ?? 1, pageSize: input.pageSize ?? 50, sort: input.sort === "requirement_code" ? "requirement_code" : input.sort === "requirement_label" ? "requirement_label" : input.sort === "section_code" ? "section_code" : input.sort === "status" ? "status" : input.sort === "document_count" ? "document_count" : undefined, direction: input.direction, "filter.requirement_code": input.requirementCode } as const);
const portfolioOpisParams = (input: PortfolioOpisListQuery = {}) => ({ page: input.page ?? 1, pageSize: input.pageSize ?? 50, sort: input.sort === "section_code" ? "section_code" : input.sort === "component_code" ? "component_code" : input.sort === "entry_title" ? "entry_title" : input.sort === "chronological_index" ? "chronological_index" : input.sort === "document_reference" ? "document_reference" : undefined, direction: input.direction, "filter.section_code": input.sectionCode } as const);
const portfolioCustodyParams = (input: PortfolioCustodyListQuery = {}) => ({ page: input.page ?? 1, pageSize: input.pageSize ?? 50, sort: input.sort === "event_type" ? "event_type" : input.sort === "holder_name" ? "holder_name" : input.sort === "holder_role" ? "holder_role" : input.sort === "started_on" ? "started_on" : input.sort === "ended_on" ? "ended_on" : undefined, direction: input.direction, "filter.event_type": input.eventType } as const);
const portfolioReviewParams = (input: PortfolioReviewListQuery = {}) => ({ page: input.page ?? 1, pageSize: input.pageSize ?? 50, sort: input.sort === "review_code" ? "review_code" : input.sort === "review_stage" ? "review_stage" : input.sort === "outcome" ? "outcome" : input.sort === "reviewer_name" ? "reviewer_name" : input.sort === "reviewed_on" ? "reviewed_on" : undefined, direction: input.direction, "filter.review_code": input.reviewCode } as const);
const portfolioTransferParams = (input: PortfolioTransferHistoryQuery = {}) => ({ ...portfolioPageParams(input), "filter.transfer_code": input.transferCode });
const portfolioValorificationParams = (input: PortfolioValorificationListQuery = {}) => ({ ...portfolioPageParams(input), "filter.valorification_code": input.valorificationCode });
const portfolioSectionParams = (input: PortfolioSectionListQuery = {}) => ({ ...portfolioPageParams(input), "filter.section_code": input.sectionCode });
const requirementParams = (input: EducationRequirementListQuery = {}) => ({ ...portfolioPageParams(input), "filter.domain": input.domain });

/**
 * Root list transport remains server-side: pagination, sort and header-row
 * filters are serialized exactly once, then sent through the domain's literal
 * generated operation below. The backend owns validation of domain filters.
 */
function rootListQuery(input: EducationListQuery): Query { return query(input); }

async function rootRecords<D extends EducationRecordsDomain>(client: ContractClient, domain: D, input: EducationListQuery): Promise<EducationPage<EducationRootRecordByDomain[D]>> {
  const params = { query: rootListQuery(input) };
  switch (domain) {
    case "decisions": return unwrap(client.GET("/api/education/decisions/records", { params })) as Promise<EducationPage<EducationRootRecordByDomain[D]>>;
    case "managerial": return unwrap(client.GET("/api/education/managerial/records", { params })) as Promise<EducationPage<EducationRootRecordByDomain[D]>>;
    case "regulations": return unwrap(client.GET("/api/education/regulations/records", { params })) as Promise<EducationPage<EducationRootRecordByDomain[D]>>;
    case "committees": return unwrap(client.GET("/api/education/committees/records", { params })) as Promise<EducationPage<EducationRootRecordByDomain[D]>>;
    case "personnel": return unwrap(client.GET("/api/education/personnel/records", { params })) as Promise<EducationPage<EducationRootRecordByDomain[D]>>;
    case "evaluations": return unwrap(client.GET("/api/education/evaluations/records", { params })) as Promise<EducationPage<EducationRootRecordByDomain[D]>>;
    case "declarations": return unwrap(client.GET("/api/education/declarations/records", { params })) as Promise<EducationPage<EducationRootRecordByDomain[D]>>;
    case "mobility": return unwrap(client.GET("/api/education/mobility/records", { params })) as Promise<EducationPage<EducationRootRecordByDomain[D]>>;
    case "merit": return unwrap(client.GET("/api/education/gradatii/records", { params })) as Promise<EducationPage<EducationRootRecordByDomain[D]>>;
    case "portfolios": return unwrap(client.GET("/api/education/portfolios/records", { params })) as Promise<EducationPage<EducationRootRecordByDomain[D]>>;
    case "compliance": return unwrap(client.GET("/api/education/compliance/publications", { params })) as Promise<EducationPage<EducationRootRecordByDomain[D]>>;
  }
}

async function rootDetail<D extends EducationRecordsDomain>(client: ContractClient, domain: D, id: string): Promise<EducationRootRecordByDomain[D]> {
  switch (domain) {
    case "decisions": return unwrap(client.GET("/api/education/decisions/records/{decisionID}", { params: { path: { decisionID: id } } })) as Promise<EducationRootRecordByDomain[D]>;
    case "managerial": return unwrap(client.GET("/api/education/managerial/records/{recordID}", { params: { path: { recordID: id } } })) as Promise<EducationRootRecordByDomain[D]>;
    case "regulations": return unwrap(client.GET("/api/education/regulations/records/{recordID}", { params: { path: { recordID: id } } })) as Promise<EducationRootRecordByDomain[D]>;
    case "committees": return unwrap(client.GET("/api/education/committees/records/{recordID}", { params: { path: { recordID: id } } })) as Promise<EducationRootRecordByDomain[D]>;
    case "personnel": return unwrap(client.GET("/api/education/personnel/records/{recordID}", { params: { path: { recordID: id } } })) as Promise<EducationRootRecordByDomain[D]>;
    case "evaluations": return unwrap(client.GET("/api/education/evaluations/records/{recordID}", { params: { path: { recordID: id } } })) as Promise<EducationRootRecordByDomain[D]>;
    case "declarations": return unwrap(client.GET("/api/education/declarations/records/{recordID}", { params: { path: { recordID: id } } })) as Promise<EducationRootRecordByDomain[D]>;
    case "mobility": return unwrap(client.GET("/api/education/mobility/records/{recordID}", { params: { path: { recordID: id } } })) as Promise<EducationRootRecordByDomain[D]>;
    case "merit": return unwrap(client.GET("/api/education/gradatii/records/{recordID}", { params: { path: { recordID: id } } })) as Promise<EducationRootRecordByDomain[D]>;
    case "portfolios": return unwrap(client.GET("/api/education/portfolios/records/{recordID}", { params: { path: { recordID: id } } })) as Promise<EducationRootRecordByDomain[D]>;
    case "compliance": return unwrap(client.GET("/api/education/compliance/publications/{recordID}", { params: { path: { recordID: id } } })) as Promise<EducationRootRecordByDomain[D]>;
  }
}

async function saveRootRecord<D extends EducationRecordsDomain>(client: ContractClient, domain: D, input: EducationRootCreateInputByDomain[D] | EducationRootUpdateInputByDomain[D], id?: string): Promise<EducationRootRecordByDomain[D]> {
  switch (domain) {
    case "decisions": return unwrap(id ? client.PATCH("/api/education/decisions/records/{decisionID}", { params: { path: { decisionID: id } }, body: input as components["schemas"]["CreateGovernanceDecisionRequest"] }) : client.POST("/api/education/decisions/records", { body: input as components["schemas"]["CreateGovernanceDecisionRequest"] })) as Promise<EducationRootRecordByDomain[D]>;
    case "managerial": return unwrap(id ? client.PATCH("/api/education/managerial/records/{recordID}", { params: { path: { recordID: id } }, body: input as components["schemas"]["CreateManagerialDossierRequest"] }) : client.POST("/api/education/managerial/records", { body: input as components["schemas"]["CreateManagerialDossierRequest"] })) as Promise<EducationRootRecordByDomain[D]>;
    case "regulations": return unwrap(id ? client.PATCH("/api/education/regulations/records/{recordID}", { params: { path: { recordID: id } }, body: input as components["schemas"]["CreateRegulationRecordRequest"] }) : client.POST("/api/education/regulations/records", { body: input as components["schemas"]["CreateRegulationRecordRequest"] })) as Promise<EducationRootRecordByDomain[D]>;
    case "committees": return unwrap(id ? client.PATCH("/api/education/committees/records/{recordID}", { params: { path: { recordID: id } }, body: input as components["schemas"]["CreateCommitteeRecordRequest"] }) : client.POST("/api/education/committees/records", { body: input as components["schemas"]["CreateCommitteeRecordRequest"] })) as Promise<EducationRootRecordByDomain[D]>;
    case "personnel": return unwrap(id ? client.PATCH("/api/education/personnel/records/{recordID}", { params: { path: { recordID: id } }, body: input as components["schemas"]["CreatePersonnelRecordRequest"] }) : client.POST("/api/education/personnel/records", { body: input as components["schemas"]["CreatePersonnelRecordRequest"] })) as Promise<EducationRootRecordByDomain[D]>;
    case "evaluations": return unwrap(id ? client.PATCH("/api/education/evaluations/records/{recordID}", { params: { path: { recordID: id } }, body: input as components["schemas"]["CreatePersonnelEvaluationRequest"] }) : client.POST("/api/education/evaluations/records", { body: input as components["schemas"]["CreatePersonnelEvaluationRequest"] })) as Promise<EducationRootRecordByDomain[D]>;
    case "declarations": return unwrap(id ? client.PATCH("/api/education/declarations/records/{recordID}", { params: { path: { recordID: id } }, body: input as components["schemas"]["CreatePersonnelDeclarationRequest"] }) : client.POST("/api/education/declarations/records", { body: input as components["schemas"]["CreatePersonnelDeclarationRequest"] })) as Promise<EducationRootRecordByDomain[D]>;
    case "mobility": return unwrap(id ? client.PATCH("/api/education/mobility/records/{recordID}", { params: { path: { recordID: id } }, body: input as components["schemas"]["CreateMobilityCaseRequest"] }) : client.POST("/api/education/mobility/records", { body: input as components["schemas"]["CreateMobilityCaseRequest"] })) as Promise<EducationRootRecordByDomain[D]>;
    case "merit": return unwrap(id ? client.PATCH("/api/education/gradatii/records/{recordID}", { params: { path: { recordID: id } }, body: input as components["schemas"]["CreateMeritGrantRequest"] }) : client.POST("/api/education/gradatii/records", { body: input as components["schemas"]["CreateMeritGrantRequest"] })) as Promise<EducationRootRecordByDomain[D]>;
    case "portfolios": return unwrap(id ? client.PATCH("/api/education/portfolios/records/{recordID}", { params: { path: { recordID: id } }, body: input as components["schemas"]["UpdatePortfolioRecordRequest"] }) : client.POST("/api/education/portfolios/records", { body: input as components["schemas"]["CreatePortfolioRecordRequest"] })) as Promise<EducationRootRecordByDomain[D]>;
    case "compliance": return unwrap(id ? client.PATCH("/api/education/compliance/publications/{recordID}", { params: { path: { recordID: id } }, body: input as components["schemas"]["CreatePublicationRecordRequest"] }) : client.POST("/api/education/compliance/publications", { body: input as components["schemas"]["CreatePublicationRecordRequest"] })) as Promise<EducationRootRecordByDomain[D]>;
  }
}

async function deleteRootRecord(client: ContractClient, domain: EducationRecordsDomain, id: string): Promise<void> {
  switch (domain) {
    case "decisions": await unwrap(client.DELETE("/api/education/decisions/records/{decisionID}", { params: { path: { decisionID: id } } })); return;
    case "managerial": await unwrap(client.DELETE("/api/education/managerial/records/{recordID}", { params: { path: { recordID: id } } })); return;
    case "regulations": await unwrap(client.DELETE("/api/education/regulations/records/{recordID}", { params: { path: { recordID: id } } })); return;
    case "committees": await unwrap(client.DELETE("/api/education/committees/records/{recordID}", { params: { path: { recordID: id } } })); return;
    case "personnel": await unwrap(client.DELETE("/api/education/personnel/records/{recordID}", { params: { path: { recordID: id } } })); return;
    case "evaluations": await unwrap(client.DELETE("/api/education/evaluations/records/{recordID}", { params: { path: { recordID: id } } })); return;
    case "declarations": await unwrap(client.DELETE("/api/education/declarations/records/{recordID}", { params: { path: { recordID: id } } })); return;
    case "mobility": await unwrap(client.DELETE("/api/education/mobility/records/{recordID}", { params: { path: { recordID: id } } })); return;
    case "merit": await unwrap(client.DELETE("/api/education/gradatii/records/{recordID}", { params: { path: { recordID: id } } })); return;
    case "portfolios": await unwrap(client.DELETE("/api/education/portfolios/records/{recordID}", { params: { path: { recordID: id } } })); return;
    case "compliance": await unwrap(client.DELETE("/api/education/compliance/publications/{recordID}", { params: { path: { recordID: id } } })); return;
  }
}

async function rootPdf(client: ContractClient, domain: EducationPdfRecordsDomain, id: string): Promise<Blob> {
  const options = { parseAs: "blob" as const, headers: { Accept: "application/pdf" } };
  switch (domain) {
    case "managerial": return unwrap(client.GET("/api/education/managerial/records/{recordID}/pdf", { ...options, params: { path: { recordID: id } } }));
    case "evaluations": return unwrap(client.GET("/api/education/evaluations/records/{recordID}/pdf", { ...options, params: { path: { recordID: id } } }));
    case "mobility": return unwrap(client.GET("/api/education/mobility/records/{recordID}/pdf", { ...options, params: { path: { recordID: id } } }));
    case "merit": return unwrap(client.GET("/api/education/gradatii/records/{recordID}/pdf", { ...options, params: { path: { recordID: id } } }));
    case "portfolios": return unwrap(client.GET("/api/education/portfolios/records/{recordID}/pdf", { ...options, params: { path: { recordID: id } } }));
  }
}
type Related = { list: keyof paths; detail?: keyof paths; pdf?: keyof paths; parentKey?: string; itemKey?: string };
const related: Record<EducationRelatedResource, Related> = {
  "governance-memberships":{list:"/api/education/governance/memberships",detail:"/api/education/governance/memberships/{recordID}",itemKey:"recordID"},"governance-bodies":{list:"/api/education/governance/bodies",detail:"/api/education/governance/bodies/{bodyID}",itemKey:"bodyID"},
  "meeting-participants":{list:"/api/education/governance/meetings/{meetingID}/participants",detail:"/api/education/governance/meetings/{meetingID}/participants/{participantID}",parentKey:"meetingID",itemKey:"participantID"},"meeting-documents":{list:"/api/education/governance/meetings/{meetingID}/documents",detail:"/api/education/governance/meetings/{meetingID}/documents/{documentID}",pdf:"/api/education/governance/meetings/{meetingID}/documents/{documentID}/pdf",parentKey:"meetingID",itemKey:"documentID"},"meeting-votes":{list:"/api/education/governance/meetings/{meetingID}/votes",detail:"/api/education/governance/meetings/{meetingID}/votes/{voteID}",parentKey:"meetingID",itemKey:"voteID"},"meeting-minutes":{list:"/api/education/governance/meetings/{meetingID}/minutes",detail:"/api/education/governance/meetings/{meetingID}/minutes/{recordID}",pdf:"/api/education/governance/meetings/{meetingID}/minutes/{recordID}/pdf",parentKey:"meetingID",itemKey:"recordID"},"meeting-resolutions":{list:"/api/education/governance/meetings/{meetingID}/resolutions",detail:"/api/education/governance/meetings/{meetingID}/resolutions/{recordID}",pdf:"/api/education/governance/meetings/{meetingID}/resolutions/{recordID}/pdf",parentKey:"meetingID",itemKey:"recordID"},
  "decision-issuances":{list:"/api/education/decisions/records/{decisionID}/issuances",detail:"/api/education/decisions/records/{decisionID}/issuances/{itemID}",parentKey:"decisionID",itemKey:"itemID"},"decision-publication-steps":{list:"/api/education/decisions/records/{decisionID}/publication-steps",detail:"/api/education/decisions/records/{decisionID}/publication-steps/{itemID}",parentKey:"decisionID",itemKey:"itemID"},"regulation-versions":{list:"/api/education/regulations/records/{recordID}/versions",detail:"/api/education/regulations/records/{recordID}/versions/{versionID}",parentKey:"recordID",itemKey:"versionID"},"regulation-workflow":{list:"/api/education/regulations/records/{recordID}/workflow",detail:"/api/education/regulations/records/{recordID}/workflow/{stepID}",parentKey:"recordID",itemKey:"stepID"},"committee-members":{list:"/api/education/committees/records/{recordID}/members",detail:"/api/education/committees/records/{recordID}/members/{itemID}",parentKey:"recordID",itemKey:"itemID"},
  "managerial-documents":{list:"/api/education/managerial/records/{recordID}/documents",detail:"/api/education/managerial/records/{recordID}/documents/{documentID}",pdf:"/api/education/managerial/records/{recordID}/documents/{documentID}/pdf",parentKey:"recordID",itemKey:"documentID"},"managerial-workflow":{list:"/api/education/managerial/records/{recordID}/workflow",detail:"/api/education/managerial/records/{recordID}/workflow/{stepID}",parentKey:"recordID",itemKey:"stepID"},
  "personnel-assignments":{list:"/api/education/personnel/records/{recordID}/assignments",detail:"/api/education/personnel/records/{recordID}/assignments/{itemID}",parentKey:"recordID",itemKey:"itemID"},"personnel-file-documents":{list:"/api/education/personnel/records/{recordID}/file-documents",detail:"/api/education/personnel/records/{recordID}/file-documents/{documentID}",parentKey:"recordID",itemKey:"documentID"},"personnel-disciplinary-cases":{list:"/api/education/personnel/records/{recordID}/disciplinary-cases",detail:"/api/education/personnel/records/{recordID}/disciplinary-cases/{itemID}",parentKey:"recordID",itemKey:"itemID"},"personnel-access-events":{list:"/api/education/personnel/records/{recordID}/access-events",detail:"/api/education/personnel/records/{recordID}/access-events/{eventID}",parentKey:"recordID",itemKey:"eventID"},
  "evaluation-self-reviews":{list:"/api/education/evaluations/records/{recordID}/self-reviews",detail:"/api/education/evaluations/records/{recordID}/self-reviews/{itemID}",parentKey:"recordID",itemKey:"itemID"},"evaluation-criteria":{list:"/api/education/evaluations/records/{recordID}/criteria",detail:"/api/education/evaluations/records/{recordID}/criteria/{itemID}",parentKey:"recordID",itemKey:"itemID"},"evaluation-appeals":{list:"/api/education/evaluations/records/{recordID}/appeals",detail:"/api/education/evaluations/records/{recordID}/appeals/{appealID}",pdf:"/api/education/evaluations/records/{recordID}/appeals/{appealID}/pdf",parentKey:"recordID",itemKey:"appealID"},"evaluation-result-issues":{list:"/api/education/evaluations/records/{recordID}/result-issues",detail:"/api/education/evaluations/records/{recordID}/result-issues/{itemID}",pdf:"/api/education/evaluations/records/{recordID}/result-issues/{itemID}/pdf",parentKey:"recordID",itemKey:"itemID"},
  "mobility-documents":{list:"/api/education/mobility/records/{recordID}/documents",detail:"/api/education/mobility/records/{recordID}/documents/{itemID}",parentKey:"recordID",itemKey:"itemID"},"mobility-scores":{list:"/api/education/mobility/records/{recordID}/scores",detail:"/api/education/mobility/records/{recordID}/scores/{itemID}",parentKey:"recordID",itemKey:"itemID"},"mobility-appeals":{list:"/api/education/mobility/records/{recordID}/appeals",detail:"/api/education/mobility/records/{recordID}/appeals/{itemID}",pdf:"/api/education/mobility/records/{recordID}/appeals/{itemID}/pdf",parentKey:"recordID",itemKey:"itemID"},"mobility-final-decisions":{list:"/api/education/mobility/records/{recordID}/final-decisions",detail:"/api/education/mobility/records/{recordID}/final-decisions/{itemID}",pdf:"/api/education/mobility/records/{recordID}/final-decisions/{itemID}/pdf",parentKey:"recordID",itemKey:"itemID"},"mobility-result-issues":{list:"/api/education/mobility/records/{recordID}/result-issues",detail:"/api/education/mobility/records/{recordID}/result-issues/{itemID}",pdf:"/api/education/mobility/records/{recordID}/result-issues/{itemID}/pdf",parentKey:"recordID",itemKey:"itemID"},
  "merit-documents":{list:"/api/education/gradatii/records/{recordID}/documents",detail:"/api/education/gradatii/records/{recordID}/documents/{itemID}",parentKey:"recordID",itemKey:"itemID"},"merit-scores":{list:"/api/education/gradatii/records/{recordID}/scores",detail:"/api/education/gradatii/records/{recordID}/scores/{itemID}",parentKey:"recordID",itemKey:"itemID"},"merit-appeals":{list:"/api/education/gradatii/records/{recordID}/appeals",detail:"/api/education/gradatii/records/{recordID}/appeals/{itemID}",pdf:"/api/education/gradatii/records/{recordID}/appeals/{itemID}/pdf",parentKey:"recordID",itemKey:"itemID"},"merit-final-decisions":{list:"/api/education/gradatii/records/{recordID}/final-decisions",detail:"/api/education/gradatii/records/{recordID}/final-decisions/{itemID}",pdf:"/api/education/gradatii/records/{recordID}/final-decisions/{itemID}/pdf",parentKey:"recordID",itemKey:"itemID"},"merit-result-issues":{list:"/api/education/gradatii/records/{recordID}/result-issues",detail:"/api/education/gradatii/records/{recordID}/result-issues/{itemID}",pdf:"/api/education/gradatii/records/{recordID}/result-issues/{itemID}/pdf",parentKey:"recordID",itemKey:"itemID"},
};
function routePath(route: Related, parentID?: string, itemID?: string): Parameters { const result: Parameters = {}; if (route.parentKey) { if (!parentID) throw new Error("education_parent_id_required"); result[route.parentKey] = parentID; } if (route.itemKey && itemID) result[route.itemKey] = itemID; return result; }

type GovernanceRelatedResource = Extract<EducationRelatedResource,
  "governance-memberships" | "governance-bodies" | "meeting-participants" | "meeting-documents" | "meeting-votes" | "meeting-minutes" | "meeting-resolutions" | "decision-issuances" | "decision-publication-steps" | "regulation-versions" | "regulation-workflow" | "committee-members" | "managerial-documents" | "managerial-workflow">;
const isGovernanceRelated = (resource: EducationRelatedResource): resource is GovernanceRelatedResource => ["governance-memberships","governance-bodies","meeting-participants","meeting-documents","meeting-votes","meeting-minutes","meeting-resolutions","decision-issuances","decision-publication-steps","regulation-versions","regulation-workflow","committee-members","managerial-documents","managerial-workflow"].includes(resource);
const requiredParent = (value: string | undefined) => { if (!value) throw new Error("education_parent_id_required"); return value; };
/**
 * Governance handlers use httpx.ParsePageQuery.  They deliberately do not
 * implement a free-text `q` parameter: only their declared header filters are
 * consumed.  Do not reuse the generic root-query serialiser here, otherwise
 * the client would silently advertise/filter by a parameter the server drops.
 */
const governanceListParams = (input: EducationListQuery, filterKeys: readonly string[]): Query => {
  const params: Query = {
    page: input.page ?? 1,
    pageSize: input.pageSize ?? 50,
    sort: input.sort,
    direction: input.direction,
  };
  for (const key of filterKeys) {
    const value = input.filters?.[key]?.trim();
    if (value) params[`filter.${key}`] = value;
  }
  return params;
};
const relatedList = (client: ContractClient, resource: GovernanceRelatedResource, parentID: string | undefined, input: EducationListQuery) => {
  switch (resource) {
    case "governance-memberships": return unwrap(client.GET("/api/education/governance/memberships", { params: { query: governanceListParams(input, ["school_year", "organism", "full_name", "role_name", "status"]) } }));
    case "governance-bodies": return unwrap(client.GET("/api/education/governance/bodies", { params: { query: governanceListParams(input, ["school_year", "organism"]) } }));
    case "meeting-participants": return unwrap(client.GET("/api/education/governance/meetings/{meetingID}/participants", { params: { path: { meetingID: requiredParent(parentID) }, query: governanceListParams(input, ["full_name", "role_name", "member_type", "attendance_status", "signature_present", "voting_right"]) } }));
    case "meeting-documents": return unwrap(client.GET("/api/education/governance/meetings/{meetingID}/documents", { params: { path: { meetingID: requiredParent(parentID) }, query: governanceListParams(input, ["document_type", "title", "document_number", "registry_number", "publication_status", "issued_on", "custody_owner"]) } }));
    case "meeting-votes": return unwrap(client.GET("/api/education/governance/meetings/{meetingID}/votes", { params: { path: { meetingID: requiredParent(parentID) }, query: governanceListParams(input, ["subject_title", "agenda_order", "decision_type", "outcome", "requires_follow_up"]) } }));
    case "meeting-minutes": return unwrap(client.GET("/api/education/governance/meetings/{meetingID}/minutes", { params: { path: { meetingID: requiredParent(parentID) }, query: governanceListParams(input, ["agenda_order", "topic_title", "discussion_summary", "decision_summary", "follow_up_status", "responsible_party", "due_on", "requires_publication", "notes"]) } }));
    case "meeting-resolutions": return unwrap(client.GET("/api/education/governance/meetings/{meetingID}/resolutions", { params: { path: { meetingID: requiredParent(parentID) }, query: governanceListParams(input, ["resolution_code", "title", "resolution_type", "publication_status", "anonymization_state"]) } }));
    case "decision-issuances": return unwrap(client.GET("/api/education/decisions/records/{decisionID}/issuances", { params: { path: { decisionID: requiredParent(parentID) }, query: governanceListParams(input, ["issuance_code", "document_type", "recipient_name", "recipient_role", "delivery_channel", "delivery_status", "signed_on", "delivered_on"]) } }));
    case "decision-publication-steps": return unwrap(client.GET("/api/education/decisions/records/{decisionID}/publication-steps", { params: { path: { decisionID: requiredParent(parentID) }, query: governanceListParams(input, ["step_order", "step_type", "status", "responsible_name", "publication_channel", "due_on", "completed_on"]) } }));
    case "regulation-versions": return unwrap(client.GET("/api/education/regulations/records/{recordID}/versions", { params: { path: { recordID: requiredParent(parentID) }, query: governanceListParams(input, ["version_label", "version_status", "prepared_by", "approved_on", "effective_from", "published_on"]) } }));
    case "regulation-workflow": return unwrap(client.GET("/api/education/regulations/records/{recordID}/workflow", { params: { path: { recordID: requiredParent(parentID) }, query: governanceListParams(input, ["phase_order", "phase_type", "status", "audience", "started_on", "due_on", "completed_on", "feedback_count"]) } }));
    case "committee-members": return unwrap(client.GET("/api/education/committees/records/{recordID}/members", { params: { path: { recordID: requiredParent(parentID) }, query: governanceListParams(input, ["full_name", "role_name", "member_type", "status", "appointed_on"]) } }));
    case "managerial-documents": return unwrap(client.GET("/api/education/managerial/records/{recordID}/documents", { params: { path: { recordID: requiredParent(parentID) }, query: governanceListParams(input, ["document_code", "document_category", "title", "document_status", "version_label", "owner_name"]) } }));
    case "managerial-workflow": return unwrap(client.GET("/api/education/managerial/records/{recordID}/workflow", { params: { path: { recordID: requiredParent(parentID) }, query: governanceListParams(input, ["stage_order", "stage_type", "status", "assigned_to", "due_on", "completed_on"]) } }));
  }
};
const relatedDetail = (client: ContractClient, resource: GovernanceRelatedResource, parentID: string | undefined, id: string) => {
  switch (resource) {
    case "governance-memberships": return unwrap(client.GET("/api/education/governance/memberships/{recordID}", { params: { path: { recordID: id } } }));
    case "governance-bodies": return unwrap(client.GET("/api/education/governance/bodies/{bodyID}", { params: { path: { bodyID: id } } }));
    case "meeting-participants": return unwrap(client.GET("/api/education/governance/meetings/{meetingID}/participants/{participantID}", { params: { path: { meetingID: requiredParent(parentID), participantID: id } } }));
    case "meeting-documents": return unwrap(client.GET("/api/education/governance/meetings/{meetingID}/documents/{documentID}", { params: { path: { meetingID: requiredParent(parentID), documentID: id } } }));
    case "meeting-votes": return unwrap(client.GET("/api/education/governance/meetings/{meetingID}/votes/{voteID}", { params: { path: { meetingID: requiredParent(parentID), voteID: id } } }));
    case "meeting-minutes": return unwrap(client.GET("/api/education/governance/meetings/{meetingID}/minutes/{recordID}", { params: { path: { meetingID: requiredParent(parentID), recordID: id } } }));
    case "meeting-resolutions": return unwrap(client.GET("/api/education/governance/meetings/{meetingID}/resolutions/{recordID}", { params: { path: { meetingID: requiredParent(parentID), recordID: id } } }));
    case "decision-issuances": return unwrap(client.GET("/api/education/decisions/records/{decisionID}/issuances/{itemID}", { params: { path: { decisionID: requiredParent(parentID), itemID: id } } }));
    case "decision-publication-steps": return unwrap(client.GET("/api/education/decisions/records/{decisionID}/publication-steps/{itemID}", { params: { path: { decisionID: requiredParent(parentID), itemID: id } } }));
    case "regulation-versions": return unwrap(client.GET("/api/education/regulations/records/{recordID}/versions/{versionID}", { params: { path: { recordID: requiredParent(parentID), versionID: id } } }));
    case "regulation-workflow": return unwrap(client.GET("/api/education/regulations/records/{recordID}/workflow/{stepID}", { params: { path: { recordID: requiredParent(parentID), stepID: id } } }));
    case "committee-members": return unwrap(client.GET("/api/education/committees/records/{recordID}/members/{itemID}", { params: { path: { recordID: requiredParent(parentID), itemID: id } } }));
    case "managerial-documents": return unwrap(client.GET("/api/education/managerial/records/{recordID}/documents/{documentID}", { params: { path: { recordID: requiredParent(parentID), documentID: id } } }));
    case "managerial-workflow": return unwrap(client.GET("/api/education/managerial/records/{recordID}/workflow/{stepID}", { params: { path: { recordID: requiredParent(parentID), stepID: id } } }));
  }
};
/** Closed, discriminated request mappers for all governance relations.  A
 * relation list record is not a request DTO: each branch enforces the exact
 * generated request required fields and excludes IDs, codes and server fields.
 */
function governanceBody(resource: "governance-memberships", input: EducationRecordInput): components["schemas"]["CreateGovernanceMembershipRequest"];
function governanceBody(resource: "meeting-participants", input: EducationRecordInput): components["schemas"]["CreateGovernanceMeetingParticipantRequest"];
function governanceBody(resource: "meeting-documents", input: EducationRecordInput): components["schemas"]["CreateGovernanceMeetingDocumentRequest"];
function governanceBody(resource: "meeting-votes", input: EducationRecordInput): components["schemas"]["CreateGovernanceMeetingVoteRequest"];
function governanceBody(resource: "meeting-minutes", input: EducationRecordInput): components["schemas"]["CreateGovernanceMinuteItemRequest"];
function governanceBody(resource: "meeting-resolutions", input: EducationRecordInput): components["schemas"]["CreateGovernanceResolutionRequest"];
function governanceBody(resource: "decision-issuances", input: EducationRecordInput): components["schemas"]["CreateDecisionIssuanceRequest"];
function governanceBody(resource: "decision-publication-steps", input: EducationRecordInput): components["schemas"]["CreateDecisionPublicationStepRequest"];
function governanceBody(resource: "regulation-versions", input: EducationRecordInput): components["schemas"]["CreateRegulationVersionRequest"];
function governanceBody(resource: "regulation-workflow", input: EducationRecordInput): components["schemas"]["CreateRegulationWorkflowStepRequest"];
function governanceBody(resource: "committee-members", input: EducationRecordInput): components["schemas"]["CreateCommitteeMemberRequest"];
function governanceBody(resource: "managerial-documents", input: EducationRecordInput): components["schemas"]["CreateManagerialDocumentRequest"];
function governanceBody(resource: "managerial-workflow", input: EducationRecordInput): components["schemas"]["CreateManagerialWorkflowStepRequest"];
function governanceBody(resource: Exclude<GovernanceRelatedResource, "governance-bodies">, input: EducationRecordInput): EducationRelatedCreateInputByResource[Exclude<GovernanceRelatedResource, "governance-bodies">] {
  switch (resource) {
    case "governance-memberships": return { school_year: requiredText(input, "school_year"), organism: requiredText(input, "organism"), app_user_id: requiredText(input, "app_user_id"), role_name: requiredText(input, "role_name"), mandate_from: requiredText(input, "mandate_from"), mandate_to: requiredText(input, "mandate_to"), status: requiredText(input, "status"), full_name: optionalText(input, "full_name"), voting_right: optionalBoolean(input, "voting_right"), notes: optionalText(input, "notes") } satisfies components["schemas"]["CreateGovernanceMembershipRequest"];
    case "meeting-participants": return { full_name: requiredText(input, "full_name"), role_name: requiredText(input, "role_name"), member_type: requiredText(input, "member_type"), attendance_status: requiredText(input, "attendance_status"), voting_right: optionalBoolean(input, "voting_right"), signature_present: optionalBoolean(input, "signature_present"), notes: optionalText(input, "notes") } satisfies components["schemas"]["CreateGovernanceMeetingParticipantRequest"];
    case "meeting-documents": return { document_type: requiredText(input, "document_type"), title: requiredText(input, "title"), publication_status: requiredText(input, "publication_status"), issued_on: requiredText(input, "issued_on"), custody_owner: optionalText(input, "custody_owner"), document_number: optionalText(input, "document_number"), registry_number: optionalText(input, "registry_number"), signed_by: optionalText(input, "signed_by"), summary: optionalText(input, "summary") } satisfies components["schemas"]["CreateGovernanceMeetingDocumentRequest"];
    case "meeting-votes": return { agenda_order: requiredNumber(input, "agenda_order"), subject_title: requiredText(input, "subject_title"), decision_type: requiredText(input, "decision_type"), outcome: requiredText(input, "outcome"), abstentions: optionalNumber(input, "abstentions"), legal_basis: optionalText(input, "legal_basis"), notes: optionalText(input, "notes"), requires_follow_up: optionalBoolean(input, "requires_follow_up"), votes_against: optionalNumber(input, "votes_against"), votes_for: optionalNumber(input, "votes_for") } satisfies components["schemas"]["CreateGovernanceMeetingVoteRequest"];
    case "meeting-minutes": return { agenda_order: requiredNumber(input, "agenda_order"), topic_title: requiredText(input, "topic_title"), discussion_summary: requiredText(input, "discussion_summary"), decision_summary: requiredText(input, "decision_summary"), follow_up_status: requiredText(input, "follow_up_status"), due_on: optionalText(input, "due_on"), notes: optionalText(input, "notes"), requires_publication: optionalBoolean(input, "requires_publication"), responsible_party: optionalText(input, "responsible_party") } satisfies components["schemas"]["CreateGovernanceMinuteItemRequest"];
    case "meeting-resolutions": return { vote_id: requiredText(input, "vote_id"), title: requiredText(input, "title"), resolution_type: requiredText(input, "resolution_type"), publication_status: requiredText(input, "publication_status"), anonymization_state: requiredText(input, "anonymization_state"), issued_on: requiredText(input, "issued_on"), notes: optionalText(input, "notes"), signed_by: optionalText(input, "signed_by") } satisfies components["schemas"]["CreateGovernanceResolutionRequest"];
    case "decision-issuances": return { document_type: requiredText(input, "document_type"), recipient_name: requiredText(input, "recipient_name"), delivery_channel: requiredText(input, "delivery_channel"), delivery_status: requiredText(input, "delivery_status"), acknowledged_on: optionalText(input, "acknowledged_on"), delivered_on: optionalText(input, "delivered_on"), file_reference: optionalText(input, "file_reference"), notes: optionalText(input, "notes"), recipient_role: optionalText(input, "recipient_role"), signed_on: optionalText(input, "signed_on") } satisfies components["schemas"]["CreateDecisionIssuanceRequest"];
    case "decision-publication-steps": return { step_order: requiredNumber(input, "step_order"), step_type: requiredText(input, "step_type"), status: requiredText(input, "status"), responsible_name: requiredText(input, "responsible_name"), due_on: requiredText(input, "due_on"), completed_on: optionalText(input, "completed_on"), notes: optionalText(input, "notes"), publication_channel: optionalText(input, "publication_channel"), publication_reference: optionalText(input, "publication_reference") } satisfies components["schemas"]["CreateDecisionPublicationStepRequest"];
    case "regulation-versions": return { version_label: requiredText(input, "version_label"), version_status: requiredText(input, "version_status"), prepared_by: requiredText(input, "prepared_by"), effective_from: requiredText(input, "effective_from"), change_summary: requiredText(input, "change_summary"), approved_on: optionalText(input, "approved_on"), file_reference: optionalText(input, "file_reference"), notes: optionalText(input, "notes"), published_on: optionalText(input, "published_on") } satisfies components["schemas"]["CreateRegulationVersionRequest"];
    case "regulation-workflow": return { phase_order: requiredNumber(input, "phase_order"), phase_type: requiredText(input, "phase_type"), audience: requiredText(input, "audience"), started_on: requiredText(input, "started_on"), due_on: requiredText(input, "due_on"), status: requiredText(input, "status"), completed_on: optionalText(input, "completed_on"), decision_reference: optionalText(input, "decision_reference"), feedback_count: optionalNumber(input, "feedback_count"), notes: optionalText(input, "notes") } satisfies components["schemas"]["CreateRegulationWorkflowStepRequest"];
    case "committee-members": return { full_name: requiredText(input, "full_name"), role_name: requiredText(input, "role_name"), member_type: requiredText(input, "member_type"), status: requiredText(input, "status"), appointed_on: requiredText(input, "appointed_on"), notes: optionalText(input, "notes"), released_on: optionalText(input, "released_on"), voting_right: optionalBoolean(input, "voting_right") } satisfies components["schemas"]["CreateCommitteeMemberRequest"];
    case "managerial-documents": return managerialDocumentBody(input);
    case "managerial-workflow": return { stage_order: requiredNumber(input, "stage_order"), stage_type: requiredText(input, "stage_type"), status: requiredText(input, "status"), assigned_to: requiredText(input, "assigned_to"), due_on: requiredText(input, "due_on"), completed_on: optionalText(input, "completed_on"), decision_reference: optionalText(input, "decision_reference"), outcome_note: optionalText(input, "outcome_note"), requires_signature: optionalBoolean(input, "requires_signature") } satisfies components["schemas"]["CreateManagerialWorkflowStepRequest"];
  }
}
const relatedSave = (client: ContractClient, resource: Exclude<GovernanceRelatedResource, "governance-bodies">, parentID: string | undefined, input: EducationRecordInput, id?: string) => {
  switch(resource) {
    case "governance-memberships": { const body=governanceBody(resource,input); return id?unwrap(client.PATCH("/api/education/governance/memberships/{recordID}",{params:{path:{recordID:id}},body})):unwrap(client.POST("/api/education/governance/memberships",{body})); }
    case "meeting-participants": { const body=governanceBody(resource,input),meetingID=requiredParent(parentID); return id?unwrap(client.PATCH("/api/education/governance/meetings/{meetingID}/participants/{participantID}",{params:{path:{meetingID,participantID:id}},body})):unwrap(client.POST("/api/education/governance/meetings/{meetingID}/participants",{params:{path:{meetingID}},body})); }
    case "meeting-documents": { const body=governanceBody(resource,input),meetingID=requiredParent(parentID); return id?unwrap(client.PATCH("/api/education/governance/meetings/{meetingID}/documents/{documentID}",{params:{path:{meetingID,documentID:id}},body})):unwrap(client.POST("/api/education/governance/meetings/{meetingID}/documents",{params:{path:{meetingID}},body})); }
    case "meeting-votes": { const body=governanceBody(resource,input),meetingID=requiredParent(parentID); return id?unwrap(client.PATCH("/api/education/governance/meetings/{meetingID}/votes/{voteID}",{params:{path:{meetingID,voteID:id}},body})):unwrap(client.POST("/api/education/governance/meetings/{meetingID}/votes",{params:{path:{meetingID}},body})); }
    case "meeting-minutes": { const body=governanceBody(resource,input),meetingID=requiredParent(parentID); return id?unwrap(client.PATCH("/api/education/governance/meetings/{meetingID}/minutes/{recordID}",{params:{path:{meetingID,recordID:id}},body})):unwrap(client.POST("/api/education/governance/meetings/{meetingID}/minutes",{params:{path:{meetingID}},body})); }
    case "meeting-resolutions": { const body=governanceBody(resource,input),meetingID=requiredParent(parentID); return id?unwrap(client.PATCH("/api/education/governance/meetings/{meetingID}/resolutions/{recordID}",{params:{path:{meetingID,recordID:id}},body})):unwrap(client.POST("/api/education/governance/meetings/{meetingID}/resolutions",{params:{path:{meetingID}},body})); }
    case "decision-issuances": { const body=governanceBody(resource,input),decisionID=requiredParent(parentID); return id?unwrap(client.PATCH("/api/education/decisions/records/{decisionID}/issuances/{itemID}",{params:{path:{decisionID,itemID:id}},body})):unwrap(client.POST("/api/education/decisions/records/{decisionID}/issuances",{params:{path:{decisionID}},body})); }
    case "decision-publication-steps": { const body=governanceBody(resource,input),decisionID=requiredParent(parentID); return id?unwrap(client.PATCH("/api/education/decisions/records/{decisionID}/publication-steps/{itemID}",{params:{path:{decisionID,itemID:id}},body})):unwrap(client.POST("/api/education/decisions/records/{decisionID}/publication-steps",{params:{path:{decisionID}},body})); }
    case "regulation-versions": { const body=governanceBody(resource,input),recordID=requiredParent(parentID); return id?unwrap(client.PATCH("/api/education/regulations/records/{recordID}/versions/{versionID}",{params:{path:{recordID,versionID:id}},body})):unwrap(client.POST("/api/education/regulations/records/{recordID}/versions",{params:{path:{recordID}},body})); }
    case "regulation-workflow": { const body=governanceBody(resource,input),recordID=requiredParent(parentID); return id?unwrap(client.PATCH("/api/education/regulations/records/{recordID}/workflow/{stepID}",{params:{path:{recordID,stepID:id}},body})):unwrap(client.POST("/api/education/regulations/records/{recordID}/workflow",{params:{path:{recordID}},body})); }
    case "committee-members": { const body=governanceBody(resource,input),recordID=requiredParent(parentID); return id?unwrap(client.PATCH("/api/education/committees/records/{recordID}/members/{itemID}",{params:{path:{recordID,itemID:id}},body})):unwrap(client.POST("/api/education/committees/records/{recordID}/members",{params:{path:{recordID}},body})); }
    case "managerial-documents": { const body=governanceBody(resource,input),recordID=requiredParent(parentID); return id?unwrap(client.PATCH("/api/education/managerial/records/{recordID}/documents/{documentID}",{params:{path:{recordID,documentID:id}},body})):unwrap(client.POST("/api/education/managerial/records/{recordID}/documents",{params:{path:{recordID}},body})); }
    case "managerial-workflow": { const body=governanceBody(resource,input),recordID=requiredParent(parentID); return id?unwrap(client.PATCH("/api/education/managerial/records/{recordID}/workflow/{stepID}",{params:{path:{recordID,stepID:id}},body})):unwrap(client.POST("/api/education/managerial/records/{recordID}/workflow",{params:{path:{recordID}},body})); }
  }
};
const relatedDelete = async (client: ContractClient, resource: Exclude<GovernanceRelatedResource, "governance-bodies">, parentID: string | undefined, id: string): Promise<void> => {
  switch(resource) {
    case "governance-memberships": await unwrap(client.DELETE("/api/education/governance/memberships/{recordID}",{params:{path:{recordID:id}}})); return;
    case "meeting-participants": await unwrap(client.DELETE("/api/education/governance/meetings/{meetingID}/participants/{participantID}",{params:{path:{meetingID:requiredParent(parentID),participantID:id}}})); return;
    case "meeting-documents": await unwrap(client.DELETE("/api/education/governance/meetings/{meetingID}/documents/{documentID}",{params:{path:{meetingID:requiredParent(parentID),documentID:id}}})); return;
    case "meeting-votes": await unwrap(client.DELETE("/api/education/governance/meetings/{meetingID}/votes/{voteID}",{params:{path:{meetingID:requiredParent(parentID),voteID:id}}})); return;
    case "meeting-minutes": await unwrap(client.DELETE("/api/education/governance/meetings/{meetingID}/minutes/{recordID}",{params:{path:{meetingID:requiredParent(parentID),recordID:id}}})); return;
    case "meeting-resolutions": await unwrap(client.DELETE("/api/education/governance/meetings/{meetingID}/resolutions/{recordID}",{params:{path:{meetingID:requiredParent(parentID),recordID:id}}})); return;
    case "decision-issuances": await unwrap(client.DELETE("/api/education/decisions/records/{decisionID}/issuances/{itemID}",{params:{path:{decisionID:requiredParent(parentID),itemID:id}}})); return;
    case "decision-publication-steps": await unwrap(client.DELETE("/api/education/decisions/records/{decisionID}/publication-steps/{itemID}",{params:{path:{decisionID:requiredParent(parentID),itemID:id}}})); return;
    case "regulation-versions": await unwrap(client.DELETE("/api/education/regulations/records/{recordID}/versions/{versionID}",{params:{path:{recordID:requiredParent(parentID),versionID:id}}})); return;
    case "regulation-workflow": await unwrap(client.DELETE("/api/education/regulations/records/{recordID}/workflow/{stepID}",{params:{path:{recordID:requiredParent(parentID),stepID:id}}})); return;
    case "committee-members": await unwrap(client.DELETE("/api/education/committees/records/{recordID}/members/{itemID}",{params:{path:{recordID:requiredParent(parentID),itemID:id}}})); return;
    case "managerial-documents": await unwrap(client.DELETE("/api/education/managerial/records/{recordID}/documents/{documentID}",{params:{path:{recordID:requiredParent(parentID),documentID:id}}})); return;
    case "managerial-workflow": await unwrap(client.DELETE("/api/education/managerial/records/{recordID}/workflow/{stepID}",{params:{path:{recordID:requiredParent(parentID),stepID:id}}})); return;
  }
};
const relatedPdf = (client: ContractClient, resource: Extract<GovernanceRelatedResource, "meeting-documents" | "meeting-minutes" | "meeting-resolutions" | "managerial-documents">, parentID: string | undefined, id: string): Promise<Blob> => {
  const options={parseAs:"blob" as const,headers:{Accept:"application/pdf"}};
  switch(resource) {
    case "meeting-documents": return unwrap(client.GET("/api/education/governance/meetings/{meetingID}/documents/{documentID}/pdf",{...options,params:{path:{meetingID:requiredParent(parentID),documentID:id}}}));
    case "meeting-minutes": return unwrap(client.GET("/api/education/governance/meetings/{meetingID}/minutes/{recordID}/pdf",{...options,params:{path:{meetingID:requiredParent(parentID),recordID:id}}}));
    case "meeting-resolutions": return unwrap(client.GET("/api/education/governance/meetings/{meetingID}/resolutions/{recordID}/pdf",{...options,params:{path:{meetingID:requiredParent(parentID),recordID:id}}}));
    case "managerial-documents": return unwrap(client.GET("/api/education/managerial/records/{recordID}/documents/{documentID}/pdf",{...options,params:{path:{recordID:requiredParent(parentID),documentID:id}}}));
  }
};
/**
 * Personnel and evaluation relations have independent generated request and
 * response contracts.  They intentionally do not use the residual auxiliary
 * adapter: an accidental route/body mismatch here could expose personnel data
 * or cause the UI to submit server-owned identifiers.
 */
type PersonnelEvaluationRelatedResource = Extract<EducationRelatedResource,
  "personnel-assignments" | "personnel-file-documents" | "personnel-disciplinary-cases" | "personnel-access-events" |
  "evaluation-self-reviews" | "evaluation-criteria" | "evaluation-appeals" | "evaluation-result-issues">;
const isPersonnelEvaluationRelated = (resource: EducationRelatedResource): resource is PersonnelEvaluationRelatedResource => [
  "personnel-assignments", "personnel-file-documents", "personnel-disciplinary-cases", "personnel-access-events",
  "evaluation-self-reviews", "evaluation-criteria", "evaluation-appeals", "evaluation-result-issues",
].includes(resource);
const requiredText = (input: EducationRecordInput, key: string): string => {
  const value = input[key];
  if (typeof value !== "string" || !value.trim()) throw new Error(`education_required_${key}`);
  return value.trim();
};
const managerialDocumentCategories = ["diagnoza", "prognoza", "evidenta", "planificare", "raport", "anexa", "hotarare", "procedura"] as const;
const managerialDocumentStatuses = ["draft", "in_review", "approved", "published", "archived"] as const;
const requiredEnum = <T extends readonly string[]>(input: EducationRecordInput, key: string, allowed: T): T[number] => {
  const value = requiredText(input, key);
  if (!allowed.includes(value)) throw new Error(`education_enum_${key}`);
  return value as T[number];
};
function managerialDocumentBody(input: EducationRecordInput): components["schemas"]["CreateManagerialDocumentRequest"] {
  const documentStatus = requiredEnum(input, "document_status", managerialDocumentStatuses);
  const approvedOn = optionalText(input, "approved_on");
  if (["approved", "published", "archived"].includes(documentStatus) && !approvedOn) {
    throw new Error("education_required_approved_on");
  }
  return {
    document_category: requiredEnum(input, "document_category", managerialDocumentCategories),
    title: requiredText(input, "title"),
    document_status: documentStatus,
    version_label: requiredText(input, "version_label"),
    registered_on: requiredText(input, "registered_on"),
    approved_on: approvedOn,
    file_reference: optionalText(input, "file_reference"),
    mandatory: optionalBoolean(input, "mandatory"),
    notes: optionalText(input, "notes"),
    owner_name: optionalText(input, "owner_name"),
    publication_required: optionalBoolean(input, "publication_required"),
  };
}
const optionalText = (input: EducationRecordInput, key: string): string | undefined => {
  const value = input[key];
  return typeof value === "string" && value.trim() ? value.trim() : undefined;
};
const optionalNumber = (input: EducationRecordInput, key: string): number | undefined => {
  const value = input[key];
  if (value === undefined || value === "") return undefined;
  if (typeof value !== "number" || !Number.isFinite(value)) throw new Error(`education_number_${key}`);
  return value;
};
const requiredNumber = (input: EducationRecordInput, key: string): number => {
  const value = optionalNumber(input, key);
  if (value === undefined) throw new Error(`education_required_${key}`);
  return value;
};
const optionalBoolean = (input: EducationRecordInput, key: string): boolean | undefined => {
  const value = input[key];
  return typeof value === "boolean" ? value : undefined;
};
const personnelEvaluationListParams = (input: EducationListQuery, filterKey: string) => ({
  page: input.page ?? 1,
  pageSize: input.pageSize ?? 50,
  sort: input.sort,
  direction: input.direction,
  [`filter.${filterKey}`]: input.filters?.[filterKey]?.trim() || undefined,
});
function personnelEvaluationBody(resource: "personnel-assignments", input: EducationRecordInput): components["schemas"]["CreatePersonnelAssignmentRequest"];
function personnelEvaluationBody(resource: "personnel-file-documents", input: EducationRecordInput): components["schemas"]["CreatePersonnelPersonalFileDocumentRequest"];
function personnelEvaluationBody(resource: "personnel-disciplinary-cases", input: EducationRecordInput): components["schemas"]["CreatePersonnelDisciplinaryCaseRequest"];
function personnelEvaluationBody(resource: "personnel-access-events", input: EducationRecordInput): components["schemas"]["CreatePersonnelPersonalAccessEventRequest"];
function personnelEvaluationBody(resource: "evaluation-self-reviews", input: EducationRecordInput): components["schemas"]["CreatePersonnelEvaluationSelfReviewRequest"];
function personnelEvaluationBody(resource: "evaluation-criteria", input: EducationRecordInput): components["schemas"]["CreatePersonnelEvaluationCriterionRequest"];
function personnelEvaluationBody(resource: "evaluation-appeals", input: EducationRecordInput): components["schemas"]["CreatePersonnelEvaluationAppealRequest"];
function personnelEvaluationBody(resource: "evaluation-result-issues", input: EducationRecordInput): components["schemas"]["CreatePersonnelEvaluationResultIssueRequest"];
function personnelEvaluationBody(resource: PersonnelEvaluationRelatedResource, input: EducationRecordInput): EducationRelatedCreateInputByResource[PersonnelEvaluationRelatedResource] {
  switch (resource) {
    case "personnel-assignments": return {
      assigned_on: requiredText(input, "assigned_on"), assignment_title: requiredText(input, "assignment_title"), assignment_type: requiredText(input, "assignment_type"), status: requiredText(input, "status"),
      decision_reference: optionalText(input, "decision_reference"), ended_on: optionalText(input, "ended_on"), notes: optionalText(input, "notes"), weekly_hours: optionalNumber(input, "weekly_hours"),
    } satisfies components["schemas"]["CreatePersonnelAssignmentRequest"];
    case "personnel-file-documents": return {
      confidentiality_level: requiredText(input, "confidentiality_level"), document_category: requiredText(input, "document_category"), document_title: requiredText(input, "document_title"), file_scope: requiredText(input, "file_scope"), issued_on: requiredText(input, "issued_on"),
      expires_on: optionalText(input, "expires_on"), file_reference: optionalText(input, "file_reference"), included_in_portfolio: optionalBoolean(input, "included_in_portfolio"), notes: optionalText(input, "notes"), sensitive_data: optionalBoolean(input, "sensitive_data"),
    } satisfies components["schemas"]["CreatePersonnelPersonalFileDocumentRequest"];
    case "personnel-disciplinary-cases": return {
      case_type: requiredText(input, "case_type"), reported_on: requiredText(input, "reported_on"), status: requiredText(input, "status"),
      committee_name: optionalText(input, "committee_name"), hearing_on: optionalText(input, "hearing_on"), legal_basis: optionalText(input, "legal_basis"), notes: optionalText(input, "notes"), resolved_on: optionalText(input, "resolved_on"), sanction: optionalText(input, "sanction"),
    } satisfies components["schemas"]["CreatePersonnelDisciplinaryCaseRequest"];
    case "personnel-access-events": return {
      access_channel: requiredText(input, "access_channel"), accessed_on: requiredText(input, "accessed_on"), actor_name: requiredText(input, "actor_name"), actor_role: requiredText(input, "actor_role"), event_type: requiredText(input, "event_type"), purpose: requiredText(input, "purpose"),
      closed_on: optionalText(input, "closed_on"), notes: optionalText(input, "notes"), sensitive_scope: optionalBoolean(input, "sensitive_scope"),
    } satisfies components["schemas"]["CreatePersonnelPersonalAccessEventRequest"];
    case "evaluation-self-reviews": return {
      completed_on: requiredText(input, "completed_on"), narrative_type: requiredText(input, "narrative_type"), section_title: requiredText(input, "section_title"), status: requiredText(input, "status"),
      assumed_score: optionalNumber(input, "assumed_score"), evidence_summary: optionalText(input, "evidence_summary"), improvement_needs: optionalText(input, "improvement_needs"), notes: optionalText(input, "notes"), strengths: optionalText(input, "strengths"),
    } satisfies components["schemas"]["CreatePersonnelEvaluationSelfReviewRequest"];
    case "evaluation-criteria": return {
      criterion_category: requiredText(input, "criterion_category"), criterion_label: requiredText(input, "criterion_label"), max_score: requiredNumber(input, "max_score"), status: requiredText(input, "status"),
      evidence_summary: optionalText(input, "evidence_summary"), final_score: optionalNumber(input, "final_score"), notes: optionalText(input, "notes"), reviewer_score: optionalNumber(input, "reviewer_score"), self_score: optionalNumber(input, "self_score"),
    } satisfies components["schemas"]["CreatePersonnelEvaluationCriterionRequest"];
    case "evaluation-appeals": return {
      grounds: requiredText(input, "grounds"), status: requiredText(input, "status"), submitted_by: requiredText(input, "submitted_by"), submitted_on: requiredText(input, "submitted_on"),
      attached_to_personnel_file: optionalBoolean(input, "attached_to_personnel_file"), committee_note: optionalText(input, "committee_note"), decision_summary: optionalText(input, "decision_summary"), hearing_on: optionalText(input, "hearing_on"), resolved_on: optionalText(input, "resolved_on"),
    } satisfies components["schemas"]["CreatePersonnelEvaluationAppealRequest"];
    case "evaluation-result-issues": return {
      delivery_channel: requiredText(input, "delivery_channel"), delivery_status: requiredText(input, "delivery_status"), document_type: requiredText(input, "document_type"), issued_on: requiredText(input, "issued_on"), recipient_name: requiredText(input, "recipient_name"),
      acknowledged_on: optionalText(input, "acknowledged_on"), attached_to_personnel_file: optionalBoolean(input, "attached_to_personnel_file"), delivered_on: optionalText(input, "delivered_on"), notes: optionalText(input, "notes"), recipient_role: optionalText(input, "recipient_role"), registry_reference: optionalText(input, "registry_reference"),
    } satisfies components["schemas"]["CreatePersonnelEvaluationResultIssueRequest"];
  }
}
const personnelEvaluationRelatedList = (client: ContractClient, resource: PersonnelEvaluationRelatedResource, parentID: string | undefined, input: EducationListQuery) => {
  const recordID = requiredParent(parentID);
  switch (resource) {
    case "personnel-assignments": return unwrap(client.GET("/api/education/personnel/records/{recordID}/assignments", { params: { path: { recordID }, query: personnelEvaluationListParams(input, "assignment_code") } }));
    case "personnel-file-documents": return unwrap(client.GET("/api/education/personnel/records/{recordID}/file-documents", { params: { path: { recordID }, query: personnelEvaluationListParams(input, "document_code") } }));
    case "personnel-disciplinary-cases": return unwrap(client.GET("/api/education/personnel/records/{recordID}/disciplinary-cases", { params: { path: { recordID }, query: personnelEvaluationListParams(input, "case_code") } }));
    case "personnel-access-events": return unwrap(client.GET("/api/education/personnel/records/{recordID}/access-events", { params: { path: { recordID }, query: personnelEvaluationListParams(input, "event_type") } }));
    case "evaluation-self-reviews": return unwrap(client.GET("/api/education/evaluations/records/{recordID}/self-reviews", { params: { path: { recordID }, query: personnelEvaluationListParams(input, "review_code") } }));
    case "evaluation-criteria": return unwrap(client.GET("/api/education/evaluations/records/{recordID}/criteria", { params: { path: { recordID }, query: personnelEvaluationListParams(input, "criterion_code") } }));
    case "evaluation-appeals": return unwrap(client.GET("/api/education/evaluations/records/{recordID}/appeals", { params: { path: { recordID }, query: personnelEvaluationListParams(input, "appeal_code") } }));
    case "evaluation-result-issues": return unwrap(client.GET("/api/education/evaluations/records/{recordID}/result-issues", { params: { path: { recordID }, query: personnelEvaluationListParams(input, "issue_code") } }));
  }
};
const personnelEvaluationRelatedDetail = (client: ContractClient, resource: PersonnelEvaluationRelatedResource, parentID: string | undefined, id: string) => {
  const recordID = requiredParent(parentID);
  switch (resource) {
    case "personnel-assignments": return unwrap(client.GET("/api/education/personnel/records/{recordID}/assignments/{itemID}", { params: { path: { recordID, itemID: id } } }));
    case "personnel-file-documents": return unwrap(client.GET("/api/education/personnel/records/{recordID}/file-documents/{documentID}", { params: { path: { recordID, documentID: id } } }));
    case "personnel-disciplinary-cases": return unwrap(client.GET("/api/education/personnel/records/{recordID}/disciplinary-cases/{itemID}", { params: { path: { recordID, itemID: id } } }));
    case "personnel-access-events": return unwrap(client.GET("/api/education/personnel/records/{recordID}/access-events/{eventID}", { params: { path: { recordID, eventID: id } } }));
    case "evaluation-self-reviews": return unwrap(client.GET("/api/education/evaluations/records/{recordID}/self-reviews/{itemID}", { params: { path: { recordID, itemID: id } } }));
    case "evaluation-criteria": return unwrap(client.GET("/api/education/evaluations/records/{recordID}/criteria/{itemID}", { params: { path: { recordID, itemID: id } } }));
    case "evaluation-appeals": return unwrap(client.GET("/api/education/evaluations/records/{recordID}/appeals/{appealID}", { params: { path: { recordID, appealID: id } } }));
    case "evaluation-result-issues": return unwrap(client.GET("/api/education/evaluations/records/{recordID}/result-issues/{itemID}", { params: { path: { recordID, itemID: id } } }));
  }
};
const personnelEvaluationRelatedSave = (client: ContractClient, resource: PersonnelEvaluationRelatedResource, parentID: string | undefined, input: EducationRecordInput, id?: string) => {
  const recordID = requiredParent(parentID);
  switch (resource) {
    case "personnel-assignments": { const body = personnelEvaluationBody(resource, input); return id ? unwrap(client.PATCH("/api/education/personnel/records/{recordID}/assignments/{itemID}", { params: { path: { recordID, itemID: id } }, body })) : unwrap(client.POST("/api/education/personnel/records/{recordID}/assignments", { params: { path: { recordID } }, body })); }
    case "personnel-file-documents": { const body = personnelEvaluationBody(resource, input); return id ? unwrap(client.PATCH("/api/education/personnel/records/{recordID}/file-documents/{documentID}", { params: { path: { recordID, documentID: id } }, body })) : unwrap(client.POST("/api/education/personnel/records/{recordID}/file-documents", { params: { path: { recordID } }, body })); }
    case "personnel-disciplinary-cases": { const body = personnelEvaluationBody(resource, input); return id ? unwrap(client.PATCH("/api/education/personnel/records/{recordID}/disciplinary-cases/{itemID}", { params: { path: { recordID, itemID: id } }, body })) : unwrap(client.POST("/api/education/personnel/records/{recordID}/disciplinary-cases", { params: { path: { recordID } }, body })); }
    case "personnel-access-events": { const body = personnelEvaluationBody(resource, input); return id ? unwrap(client.PATCH("/api/education/personnel/records/{recordID}/access-events/{eventID}", { params: { path: { recordID, eventID: id } }, body })) : unwrap(client.POST("/api/education/personnel/records/{recordID}/access-events", { params: { path: { recordID } }, body })); }
    case "evaluation-self-reviews": { const body = personnelEvaluationBody(resource, input); return id ? unwrap(client.PATCH("/api/education/evaluations/records/{recordID}/self-reviews/{itemID}", { params: { path: { recordID, itemID: id } }, body })) : unwrap(client.POST("/api/education/evaluations/records/{recordID}/self-reviews", { params: { path: { recordID } }, body })); }
    case "evaluation-criteria": { const body = personnelEvaluationBody(resource, input); return id ? unwrap(client.PATCH("/api/education/evaluations/records/{recordID}/criteria/{itemID}", { params: { path: { recordID, itemID: id } }, body })) : unwrap(client.POST("/api/education/evaluations/records/{recordID}/criteria", { params: { path: { recordID } }, body })); }
    case "evaluation-appeals": { const body = personnelEvaluationBody(resource, input); return id ? unwrap(client.PATCH("/api/education/evaluations/records/{recordID}/appeals/{appealID}", { params: { path: { recordID, appealID: id } }, body })) : unwrap(client.POST("/api/education/evaluations/records/{recordID}/appeals", { params: { path: { recordID } }, body })); }
    case "evaluation-result-issues": { const body = personnelEvaluationBody(resource, input); return id ? unwrap(client.PATCH("/api/education/evaluations/records/{recordID}/result-issues/{itemID}", { params: { path: { recordID, itemID: id } }, body })) : unwrap(client.POST("/api/education/evaluations/records/{recordID}/result-issues", { params: { path: { recordID } }, body })); }
  }
};
const personnelEvaluationRelatedDelete = async (client: ContractClient, resource: PersonnelEvaluationRelatedResource, parentID: string | undefined, id: string): Promise<void> => {
  const recordID = requiredParent(parentID);
  switch (resource) {
    case "personnel-assignments": await unwrap(client.DELETE("/api/education/personnel/records/{recordID}/assignments/{itemID}", { params: { path: { recordID, itemID: id } } })); return;
    case "personnel-file-documents": await unwrap(client.DELETE("/api/education/personnel/records/{recordID}/file-documents/{documentID}", { params: { path: { recordID, documentID: id } } })); return;
    case "personnel-disciplinary-cases": await unwrap(client.DELETE("/api/education/personnel/records/{recordID}/disciplinary-cases/{itemID}", { params: { path: { recordID, itemID: id } } })); return;
    case "personnel-access-events": await unwrap(client.DELETE("/api/education/personnel/records/{recordID}/access-events/{eventID}", { params: { path: { recordID, eventID: id } } })); return;
    case "evaluation-self-reviews": await unwrap(client.DELETE("/api/education/evaluations/records/{recordID}/self-reviews/{itemID}", { params: { path: { recordID, itemID: id } } })); return;
    case "evaluation-criteria": await unwrap(client.DELETE("/api/education/evaluations/records/{recordID}/criteria/{itemID}", { params: { path: { recordID, itemID: id } } })); return;
    case "evaluation-appeals": await unwrap(client.DELETE("/api/education/evaluations/records/{recordID}/appeals/{appealID}", { params: { path: { recordID, appealID: id } } })); return;
    case "evaluation-result-issues": await unwrap(client.DELETE("/api/education/evaluations/records/{recordID}/result-issues/{itemID}", { params: { path: { recordID, itemID: id } } })); return;
  }
};
const personnelEvaluationRelatedPdf = (client: ContractClient, resource: Extract<PersonnelEvaluationRelatedResource, "evaluation-appeals" | "evaluation-result-issues">, parentID: string | undefined, id: string): Promise<Blob> => {
  const recordID = requiredParent(parentID); const options = { parseAs: "blob" as const, headers: { Accept: "application/pdf" } };
  return resource === "evaluation-appeals"
    ? unwrap(client.GET("/api/education/evaluations/records/{recordID}/appeals/{appealID}/pdf", { ...options, params: { path: { recordID, appealID: id } } }))
    : unwrap(client.GET("/api/education/evaluations/records/{recordID}/result-issues/{itemID}/pdf", { ...options, params: { path: { recordID, itemID: id } } }));
};

/**
 * Mobility and merit have ten individually versioned related contracts.  They
 * use their own generated operation: each route,
 * path key, documented filter and request DTO is selected below at compile
 * time.  This also ensures server-owned codes can never leak back into a
 * create/PATCH body from the generic table projection.
 */
type MobilityMeritRelatedResource = Extract<EducationRelatedResource,
  "mobility-documents" | "mobility-scores" | "mobility-appeals" | "mobility-final-decisions" | "mobility-result-issues" |
  "merit-documents" | "merit-scores" | "merit-appeals" | "merit-final-decisions" | "merit-result-issues">;
const isMobilityMeritRelated = (resource: EducationRelatedResource): resource is MobilityMeritRelatedResource => [
  "mobility-documents", "mobility-scores", "mobility-appeals", "mobility-final-decisions", "mobility-result-issues",
  "merit-documents", "merit-scores", "merit-appeals", "merit-final-decisions", "merit-result-issues",
].includes(resource);
const mobilityMeritListParams = (input: EducationListQuery, filterKey: "document_code" | "criterion_code" | "appeal_code" | "decision_code" | "issue_code") => ({
  page: input.page ?? 1,
  pageSize: input.pageSize ?? 50,
  sort: input.sort,
  direction: input.direction,
  [`filter.${filterKey}`]: input.filters?.[filterKey]?.trim() || undefined,
});
function mobilityMeritBody(resource: "mobility-documents", input: EducationRecordInput): components["schemas"]["CreateMobilityDocumentRequest"];
function mobilityMeritBody(resource: "mobility-scores", input: EducationRecordInput): components["schemas"]["CreateMobilityCriterionScoreRequest"];
function mobilityMeritBody(resource: "mobility-appeals", input: EducationRecordInput): components["schemas"]["CreateMobilityAppealRequest"];
function mobilityMeritBody(resource: "mobility-final-decisions", input: EducationRecordInput): components["schemas"]["CreateMobilityFinalDecisionRequest"];
function mobilityMeritBody(resource: "mobility-result-issues", input: EducationRecordInput): components["schemas"]["CreateMobilityResultIssueRequest"];
function mobilityMeritBody(resource: "merit-documents", input: EducationRecordInput): components["schemas"]["CreateMeritDocumentRequest"];
function mobilityMeritBody(resource: "merit-scores", input: EducationRecordInput): components["schemas"]["CreateMeritCriterionScoreRequest"];
function mobilityMeritBody(resource: "merit-appeals", input: EducationRecordInput): components["schemas"]["CreateMeritAppealRequest"];
function mobilityMeritBody(resource: "merit-final-decisions", input: EducationRecordInput): components["schemas"]["CreateMeritFinalDecisionRequest"];
function mobilityMeritBody(resource: "merit-result-issues", input: EducationRecordInput): components["schemas"]["CreateMeritResultIssueRequest"];
function mobilityMeritBody(resource: MobilityMeritRelatedResource, input: EducationRecordInput): EducationRelatedCreateInputByResource[MobilityMeritRelatedResource] {
  switch (resource) {
    case "mobility-documents": return {
      document_title: requiredText(input, "document_title"), document_type: requiredText(input, "document_type"), registered_on: requiredText(input, "registered_on"), stage_scope: requiredText(input, "stage_scope"), validation_status: requiredText(input, "validation_status"),
      mandatory: optionalBoolean(input, "mandatory"), notes: optionalText(input, "notes"), submitted_by: optionalText(input, "submitted_by"), verified_by: optionalText(input, "verified_by"),
    } satisfies components["schemas"]["CreateMobilityDocumentRequest"];
    case "mobility-scores": return {
      criterion_category: requiredText(input, "criterion_category"), criterion_code: requiredText(input, "criterion_code"), criterion_label: requiredText(input, "criterion_label"), max_score: requiredNumber(input, "max_score"),
      awarded_score: optionalNumber(input, "awarded_score"), contested: optionalBoolean(input, "contested"), evidence_reference: optionalText(input, "evidence_reference"), notes: optionalText(input, "notes"), validated_by: optionalText(input, "validated_by"),
    } satisfies components["schemas"]["CreateMobilityCriterionScoreRequest"];
    case "mobility-appeals": return {
      grounds: requiredText(input, "grounds"), status: requiredText(input, "status"), submitted_by: requiredText(input, "submitted_by"), submitted_on: requiredText(input, "submitted_on"),
      decision_summary: optionalText(input, "decision_summary"), hearing_on: optionalText(input, "hearing_on"), notes: optionalText(input, "notes"), resolved_on: optionalText(input, "resolved_on"),
    } satisfies components["schemas"]["CreateMobilityAppealRequest"];
    case "mobility-final-decisions": return {
      approved_on: requiredText(input, "approved_on"), decision_type: requiredText(input, "decision_type"), effective_from: requiredText(input, "effective_from"), outcome: requiredText(input, "outcome"), panel_name: requiredText(input, "panel_name"),
      destination_unit: optionalText(input, "destination_unit"), legal_basis: optionalText(input, "legal_basis"), notes: optionalText(input, "notes"),
    } satisfies components["schemas"]["CreateMobilityFinalDecisionRequest"];
    case "mobility-result-issues": return {
      delivery_channel: requiredText(input, "delivery_channel"), delivery_status: requiredText(input, "delivery_status"), document_type: requiredText(input, "document_type"), issued_on: requiredText(input, "issued_on"), recipient_name: requiredText(input, "recipient_name"),
      delivered_on: optionalText(input, "delivered_on"), notes: optionalText(input, "notes"), recipient_role: optionalText(input, "recipient_role"), registry_reference: optionalText(input, "registry_reference"),
    } satisfies components["schemas"]["CreateMobilityResultIssueRequest"];
    case "merit-documents": return {
      document_title: requiredText(input, "document_title"), document_type: requiredText(input, "document_type"), registered_on: requiredText(input, "registered_on"), validation_status: requiredText(input, "validation_status"),
      mandatory: optionalBoolean(input, "mandatory"), notes: optionalText(input, "notes"), submitted_by: optionalText(input, "submitted_by"),
    } satisfies components["schemas"]["CreateMeritDocumentRequest"];
    case "merit-scores": return {
      criterion_category: requiredText(input, "criterion_category"), criterion_code: requiredText(input, "criterion_code"), criterion_label: requiredText(input, "criterion_label"), max_score: requiredNumber(input, "max_score"), panel_stage: requiredText(input, "panel_stage"),
      awarded_score: optionalNumber(input, "awarded_score"), contested: optionalBoolean(input, "contested"), evidence_reference: optionalText(input, "evidence_reference"), notes: optionalText(input, "notes"), reviewer_name: optionalText(input, "reviewer_name"),
    } satisfies components["schemas"]["CreateMeritCriterionScoreRequest"];
    case "merit-appeals": return {
      grounds: requiredText(input, "grounds"), status: requiredText(input, "status"), submitted_by: requiredText(input, "submitted_by"), submitted_on: requiredText(input, "submitted_on"),
      decision_summary: optionalText(input, "decision_summary"), notes: optionalText(input, "notes"), resolved_on: optionalText(input, "resolved_on"),
    } satisfies components["schemas"]["CreateMeritAppealRequest"];
    case "merit-final-decisions": return {
      approved_on: requiredText(input, "approved_on"), decision_stage: requiredText(input, "decision_stage"), effective_from: requiredText(input, "effective_from"), outcome: requiredText(input, "outcome"), panel_name: requiredText(input, "panel_name"),
      funded: optionalBoolean(input, "funded"), legal_basis: optionalText(input, "legal_basis"), notes: optionalText(input, "notes"),
    } satisfies components["schemas"]["CreateMeritFinalDecisionRequest"];
    case "merit-result-issues": return {
      delivery_channel: requiredText(input, "delivery_channel"), delivery_status: requiredText(input, "delivery_status"), document_type: requiredText(input, "document_type"), issued_on: requiredText(input, "issued_on"), recipient_name: requiredText(input, "recipient_name"),
      delivered_on: optionalText(input, "delivered_on"), notes: optionalText(input, "notes"), recipient_role: optionalText(input, "recipient_role"), registry_reference: optionalText(input, "registry_reference"),
    } satisfies components["schemas"]["CreateMeritResultIssueRequest"];
  }
}
const mobilityMeritRelatedList = (client: ContractClient, resource: MobilityMeritRelatedResource, parentID: string | undefined, input: EducationListQuery) => {
  const recordID = requiredParent(parentID);
  switch (resource) {
    case "mobility-documents": return unwrap(client.GET("/api/education/mobility/records/{recordID}/documents", { params: { path: { recordID }, query: mobilityMeritListParams(input, "document_code") } }));
    case "mobility-scores": return unwrap(client.GET("/api/education/mobility/records/{recordID}/scores", { params: { path: { recordID }, query: mobilityMeritListParams(input, "criterion_code") } }));
    case "mobility-appeals": return unwrap(client.GET("/api/education/mobility/records/{recordID}/appeals", { params: { path: { recordID }, query: mobilityMeritListParams(input, "appeal_code") } }));
    case "mobility-final-decisions": return unwrap(client.GET("/api/education/mobility/records/{recordID}/final-decisions", { params: { path: { recordID }, query: mobilityMeritListParams(input, "decision_code") } }));
    case "mobility-result-issues": return unwrap(client.GET("/api/education/mobility/records/{recordID}/result-issues", { params: { path: { recordID }, query: mobilityMeritListParams(input, "issue_code") } }));
    case "merit-documents": return unwrap(client.GET("/api/education/gradatii/records/{recordID}/documents", { params: { path: { recordID }, query: mobilityMeritListParams(input, "document_code") } }));
    case "merit-scores": return unwrap(client.GET("/api/education/gradatii/records/{recordID}/scores", { params: { path: { recordID }, query: mobilityMeritListParams(input, "criterion_code") } }));
    case "merit-appeals": return unwrap(client.GET("/api/education/gradatii/records/{recordID}/appeals", { params: { path: { recordID }, query: mobilityMeritListParams(input, "appeal_code") } }));
    case "merit-final-decisions": return unwrap(client.GET("/api/education/gradatii/records/{recordID}/final-decisions", { params: { path: { recordID }, query: mobilityMeritListParams(input, "decision_code") } }));
    case "merit-result-issues": return unwrap(client.GET("/api/education/gradatii/records/{recordID}/result-issues", { params: { path: { recordID }, query: mobilityMeritListParams(input, "issue_code") } }));
  }
};
const mobilityMeritRelatedDetail = (client: ContractClient, resource: MobilityMeritRelatedResource, parentID: string | undefined, id: string) => {
  const recordID = requiredParent(parentID);
  switch (resource) {
    case "mobility-documents": return unwrap(client.GET("/api/education/mobility/records/{recordID}/documents/{itemID}", { params: { path: { recordID, itemID: id } } }));
    case "mobility-scores": return unwrap(client.GET("/api/education/mobility/records/{recordID}/scores/{itemID}", { params: { path: { recordID, itemID: id } } }));
    case "mobility-appeals": return unwrap(client.GET("/api/education/mobility/records/{recordID}/appeals/{itemID}", { params: { path: { recordID, itemID: id } } }));
    case "mobility-final-decisions": return unwrap(client.GET("/api/education/mobility/records/{recordID}/final-decisions/{itemID}", { params: { path: { recordID, itemID: id } } }));
    case "mobility-result-issues": return unwrap(client.GET("/api/education/mobility/records/{recordID}/result-issues/{itemID}", { params: { path: { recordID, itemID: id } } }));
    case "merit-documents": return unwrap(client.GET("/api/education/gradatii/records/{recordID}/documents/{itemID}", { params: { path: { recordID, itemID: id } } }));
    case "merit-scores": return unwrap(client.GET("/api/education/gradatii/records/{recordID}/scores/{itemID}", { params: { path: { recordID, itemID: id } } }));
    case "merit-appeals": return unwrap(client.GET("/api/education/gradatii/records/{recordID}/appeals/{itemID}", { params: { path: { recordID, itemID: id } } }));
    case "merit-final-decisions": return unwrap(client.GET("/api/education/gradatii/records/{recordID}/final-decisions/{itemID}", { params: { path: { recordID, itemID: id } } }));
    case "merit-result-issues": return unwrap(client.GET("/api/education/gradatii/records/{recordID}/result-issues/{itemID}", { params: { path: { recordID, itemID: id } } }));
  }
};
const mobilityMeritRelatedSave = (client: ContractClient, resource: MobilityMeritRelatedResource, parentID: string | undefined, input: EducationRecordInput, id?: string) => {
  const recordID = requiredParent(parentID);
  switch (resource) {
    case "mobility-documents": { const body = mobilityMeritBody(resource, input); return id ? unwrap(client.PATCH("/api/education/mobility/records/{recordID}/documents/{itemID}", { params: { path: { recordID, itemID: id } }, body })) : unwrap(client.POST("/api/education/mobility/records/{recordID}/documents", { params: { path: { recordID } }, body })); }
    case "mobility-scores": { const body = mobilityMeritBody(resource, input); return id ? unwrap(client.PATCH("/api/education/mobility/records/{recordID}/scores/{itemID}", { params: { path: { recordID, itemID: id } }, body })) : unwrap(client.POST("/api/education/mobility/records/{recordID}/scores", { params: { path: { recordID } }, body })); }
    case "mobility-appeals": { const body = mobilityMeritBody(resource, input); return id ? unwrap(client.PATCH("/api/education/mobility/records/{recordID}/appeals/{itemID}", { params: { path: { recordID, itemID: id } }, body })) : unwrap(client.POST("/api/education/mobility/records/{recordID}/appeals", { params: { path: { recordID } }, body })); }
    case "mobility-final-decisions": { const body = mobilityMeritBody(resource, input); return id ? unwrap(client.PATCH("/api/education/mobility/records/{recordID}/final-decisions/{itemID}", { params: { path: { recordID, itemID: id } }, body })) : unwrap(client.POST("/api/education/mobility/records/{recordID}/final-decisions", { params: { path: { recordID } }, body })); }
    case "mobility-result-issues": { const body = mobilityMeritBody(resource, input); return id ? unwrap(client.PATCH("/api/education/mobility/records/{recordID}/result-issues/{itemID}", { params: { path: { recordID, itemID: id } }, body })) : unwrap(client.POST("/api/education/mobility/records/{recordID}/result-issues", { params: { path: { recordID } }, body })); }
    case "merit-documents": { const body = mobilityMeritBody(resource, input); return id ? unwrap(client.PATCH("/api/education/gradatii/records/{recordID}/documents/{itemID}", { params: { path: { recordID, itemID: id } }, body })) : unwrap(client.POST("/api/education/gradatii/records/{recordID}/documents", { params: { path: { recordID } }, body })); }
    case "merit-scores": { const body = mobilityMeritBody(resource, input); return id ? unwrap(client.PATCH("/api/education/gradatii/records/{recordID}/scores/{itemID}", { params: { path: { recordID, itemID: id } }, body })) : unwrap(client.POST("/api/education/gradatii/records/{recordID}/scores", { params: { path: { recordID } }, body })); }
    case "merit-appeals": { const body = mobilityMeritBody(resource, input); return id ? unwrap(client.PATCH("/api/education/gradatii/records/{recordID}/appeals/{itemID}", { params: { path: { recordID, itemID: id } }, body })) : unwrap(client.POST("/api/education/gradatii/records/{recordID}/appeals", { params: { path: { recordID } }, body })); }
    case "merit-final-decisions": { const body = mobilityMeritBody(resource, input); return id ? unwrap(client.PATCH("/api/education/gradatii/records/{recordID}/final-decisions/{itemID}", { params: { path: { recordID, itemID: id } }, body })) : unwrap(client.POST("/api/education/gradatii/records/{recordID}/final-decisions", { params: { path: { recordID } }, body })); }
    case "merit-result-issues": { const body = mobilityMeritBody(resource, input); return id ? unwrap(client.PATCH("/api/education/gradatii/records/{recordID}/result-issues/{itemID}", { params: { path: { recordID, itemID: id } }, body })) : unwrap(client.POST("/api/education/gradatii/records/{recordID}/result-issues", { params: { path: { recordID } }, body })); }
  }
};
const mobilityMeritRelatedDelete = async (client: ContractClient, resource: MobilityMeritRelatedResource, parentID: string | undefined, id: string): Promise<void> => {
  const recordID = requiredParent(parentID);
  switch (resource) {
    case "mobility-documents": await unwrap(client.DELETE("/api/education/mobility/records/{recordID}/documents/{itemID}", { params: { path: { recordID, itemID: id } } })); return;
    case "mobility-scores": await unwrap(client.DELETE("/api/education/mobility/records/{recordID}/scores/{itemID}", { params: { path: { recordID, itemID: id } } })); return;
    case "mobility-appeals": await unwrap(client.DELETE("/api/education/mobility/records/{recordID}/appeals/{itemID}", { params: { path: { recordID, itemID: id } } })); return;
    case "mobility-final-decisions": await unwrap(client.DELETE("/api/education/mobility/records/{recordID}/final-decisions/{itemID}", { params: { path: { recordID, itemID: id } } })); return;
    case "mobility-result-issues": await unwrap(client.DELETE("/api/education/mobility/records/{recordID}/result-issues/{itemID}", { params: { path: { recordID, itemID: id } } })); return;
    case "merit-documents": await unwrap(client.DELETE("/api/education/gradatii/records/{recordID}/documents/{itemID}", { params: { path: { recordID, itemID: id } } })); return;
    case "merit-scores": await unwrap(client.DELETE("/api/education/gradatii/records/{recordID}/scores/{itemID}", { params: { path: { recordID, itemID: id } } })); return;
    case "merit-appeals": await unwrap(client.DELETE("/api/education/gradatii/records/{recordID}/appeals/{itemID}", { params: { path: { recordID, itemID: id } } })); return;
    case "merit-final-decisions": await unwrap(client.DELETE("/api/education/gradatii/records/{recordID}/final-decisions/{itemID}", { params: { path: { recordID, itemID: id } } })); return;
    case "merit-result-issues": await unwrap(client.DELETE("/api/education/gradatii/records/{recordID}/result-issues/{itemID}", { params: { path: { recordID, itemID: id } } })); return;
  }
};
const mobilityMeritRelatedPdf = (client: ContractClient, resource: Extract<MobilityMeritRelatedResource, "mobility-appeals" | "mobility-final-decisions" | "mobility-result-issues" | "merit-appeals" | "merit-final-decisions" | "merit-result-issues">, parentID: string | undefined, id: string): Promise<Blob> => {
  const recordID = requiredParent(parentID); const options = { parseAs: "blob" as const, headers: { Accept: "application/pdf" } };
  switch (resource) {
    case "mobility-appeals": return unwrap(client.GET("/api/education/mobility/records/{recordID}/appeals/{itemID}/pdf", { ...options, params: { path: { recordID, itemID: id } } }));
    case "mobility-final-decisions": return unwrap(client.GET("/api/education/mobility/records/{recordID}/final-decisions/{itemID}/pdf", { ...options, params: { path: { recordID, itemID: id } } }));
    case "mobility-result-issues": return unwrap(client.GET("/api/education/mobility/records/{recordID}/result-issues/{itemID}/pdf", { ...options, params: { path: { recordID, itemID: id } } }));
    case "merit-appeals": return unwrap(client.GET("/api/education/gradatii/records/{recordID}/appeals/{itemID}/pdf", { ...options, params: { path: { recordID, itemID: id } } }));
    case "merit-final-decisions": return unwrap(client.GET("/api/education/gradatii/records/{recordID}/final-decisions/{itemID}/pdf", { ...options, params: { path: { recordID, itemID: id } } }));
    case "merit-result-issues": return unwrap(client.GET("/api/education/gradatii/records/{recordID}/result-issues/{itemID}/pdf", { ...options, params: { path: { recordID, itemID: id } } }));
  }
};
/** Every metadata path is an explicit generated operation with its response map. */
function metadataResult<R extends EducationMetadataResource>(client: ContractClient, resource: R, parentID?: string): Promise<EducationMetadataResultByResource[R]> {
  const parent = (): string => {
    if (!parentID) throw new Error("education_metadata_parent_required");
    return parentID;
  };
  switch (resource) {
    case "director-cockpit": return unwrap(client.GET("/api/education/director/cockpit")) as Promise<EducationMetadataResultByResource[R]>;
    case "governance-meeting-filters": return unwrap(client.GET("/api/education/governance/meetings/filters")) as Promise<EducationMetadataResultByResource[R]>;
    case "governance-meeting-finalization": return unwrap(client.GET("/api/education/governance/meetings/{meetingID}/finalization-summary", { params: { path: { meetingID: parent() } } })) as Promise<EducationMetadataResultByResource[R]>;
    case "governance-body-completeness": return unwrap(client.GET("/api/education/governance/bodies/{bodyID}/completeness-summary", { params: { path: { bodyID: parent() } } })) as Promise<EducationMetadataResultByResource[R]>;
    case "decisions-dashboard": return unwrap(client.GET("/api/education/decisions/dashboard")) as Promise<EducationMetadataResultByResource[R]>;
    case "decisions-filters": return unwrap(client.GET("/api/education/decisions/records/filters")) as Promise<EducationMetadataResultByResource[R]>;
    case "managerial-dashboard": return unwrap(client.GET("/api/education/managerial/dashboard")) as Promise<EducationMetadataResultByResource[R]>;
    case "managerial-filters": return unwrap(client.GET("/api/education/managerial/records/filters")) as Promise<EducationMetadataResultByResource[R]>;
    case "regulations-dashboard": return unwrap(client.GET("/api/education/regulations/dashboard")) as Promise<EducationMetadataResultByResource[R]>;
    case "regulations-filters": return unwrap(client.GET("/api/education/regulations/records/filters")) as Promise<EducationMetadataResultByResource[R]>;
    case "personnel-dashboard": return unwrap(client.GET("/api/education/personnel/dashboard")) as Promise<EducationMetadataResultByResource[R]>;
    case "personnel-filters": return unwrap(client.GET("/api/education/personnel/records/filters")) as Promise<EducationMetadataResultByResource[R]>;
    case "evaluations-dashboard": return unwrap(client.GET("/api/education/evaluations/dashboard")) as Promise<EducationMetadataResultByResource[R]>;
    case "evaluations-filters": return unwrap(client.GET("/api/education/evaluations/records/filters")) as Promise<EducationMetadataResultByResource[R]>;
    case "declarations-dashboard": return unwrap(client.GET("/api/education/declarations/dashboard")) as Promise<EducationMetadataResultByResource[R]>;
    case "declarations-filters": return unwrap(client.GET("/api/education/declarations/records/filters")) as Promise<EducationMetadataResultByResource[R]>;
    case "mobility-dashboard": return unwrap(client.GET("/api/education/mobility/dashboard")) as Promise<EducationMetadataResultByResource[R]>;
    case "mobility-filters": return unwrap(client.GET("/api/education/mobility/records/filters")) as Promise<EducationMetadataResultByResource[R]>;
    case "merit-dashboard": return unwrap(client.GET("/api/education/gradatii/dashboard")) as Promise<EducationMetadataResultByResource[R]>;
    case "merit-filters": return unwrap(client.GET("/api/education/gradatii/records/filters")) as Promise<EducationMetadataResultByResource[R]>;
    case "portfolios-dashboard": return unwrap(client.GET("/api/education/portfolios/dashboard")) as Promise<EducationMetadataResultByResource[R]>;
    case "portfolios-filters": return unwrap(client.GET("/api/education/portfolios/records/filters")) as Promise<EducationMetadataResultByResource[R]>;
    case "committee-completeness": return unwrap(client.GET("/api/education/committees/records/{recordID}/completeness-summary", { params: { path: { recordID: parent() } } })) as Promise<EducationMetadataResultByResource[R]>;
    case "managerial-portfolio-summary": return unwrap(client.GET("/api/education/managerial/records/{recordID}/portfolio-summary", { params: { path: { recordID: parent() } } })) as Promise<EducationMetadataResultByResource[R]>;
    case "personnel-portfolio-dossier-summary": return unwrap(client.GET("/api/education/personnel/records/{recordID}/portfolio-dossier-summary", { params: { path: { recordID: parent() } } })) as Promise<EducationMetadataResultByResource[R]>;
    case "portfolio-transfer-summary": return unwrap(client.GET("/api/education/portfolios/records/{recordID}/transfer-summary", { params: { path: { recordID: parent() } } })) as Promise<EducationMetadataResultByResource[R]>;
    case "regulation-procedural-summary": return unwrap(client.GET("/api/education/regulations/records/{recordID}/procedural-summary", { params: { path: { recordID: parent() } } })) as Promise<EducationMetadataResultByResource[R]>;
  }
}

export function createEducationApi(fetcher: AuthenticatedFetcher, apiBase = "/api"): EducationApi {
 const client=createContractClient((request)=>fetcher(request),apiBase);
 return {
  async governanceDashboard(): Promise<GovernanceDashboard> { return unwrap(client.GET("/api/education/dashboard")); },
  async directorCockpit(): Promise<DirectorCockpit> { return unwrap(client.GET("/api/education/director/cockpit")); },
  async eligibleGovernanceUsers(input = {}): Promise<EligibleGovernanceUser[]> { return (await unwrap(client.GET("/api/education/governance/eligible-users", { params: { query: { page: input.page ?? 1, pageSize: input.pageSize ?? 100, sort: "name", direction: "asc", "filter.name": input.q?.trim() || undefined } } }))).items; },
  async governanceMeetings(input = {}): Promise<EducationPage<GovernanceMeeting>> { return unwrap(client.GET("/api/education/governance/meetings", { params: { query: query(input) }, headers: { Accept: "application/json" } })); },
  async governanceMeetingDetail(id: string): Promise<GovernanceMeeting> { return unwrap(client.GET("/api/education/governance/meetings/{meetingID}", { params: { path: { meetingID: id } } })); },
  async saveGovernanceMeeting(input: GovernanceMeetingInput, id?: string): Promise<GovernanceMeeting> {
    return id
      ? unwrap(client.PATCH("/api/education/governance/meetings/{meetingID}", { params: { path: { meetingID: id } }, body: input }))
      : unwrap(client.POST("/api/education/governance/meetings", { body: input }));
  },
  async deleteGovernanceMeeting(id: string): Promise<void> { await unwrap(client.DELETE("/api/education/governance/meetings/{meetingID}", { params: { path: { meetingID: id } } })); },
  async records(domain,input={}){return rootRecords(client,domain,input);},async recordDetail(domain,id){return rootDetail(client,domain,id);},async createRecord(domain,input){return saveRootRecord(client,domain,input);},async updateRecord(domain,id,input){return saveRootRecord(client,domain,input,id);},async deleteRecord(domain,id){await deleteRootRecord(client,domain,id);},async recordPdf(domain:EducationPdfRecordsDomain,id){return rootPdf(client,domain,id);},
  async relatedRecords(resource,parentID,input={}){if(isGovernanceRelated(resource))return page(await relatedList(client,resource,parentID,input));if(isPersonnelEvaluationRelated(resource))return page(await personnelEvaluationRelatedList(client,resource,parentID,input));return page(await mobilityMeritRelatedList(client,resource,parentID,input));},async relatedDetail(resource,parentID,id){if(isGovernanceRelated(resource))return record(await relatedDetail(client,resource,parentID,id));if(isPersonnelEvaluationRelated(resource))return record(await personnelEvaluationRelatedDetail(client,resource,parentID,id));return record(await mobilityMeritRelatedDetail(client,resource,parentID,id));},async saveRelated(resource,parentID,input,id){if(isGovernanceRelated(resource)){if(resource==="governance-bodies")throw new Error("education_related_write_unsupported");return record(await relatedSave(client,resource,parentID,input,id));}if(isPersonnelEvaluationRelated(resource))return record(await personnelEvaluationRelatedSave(client,resource,parentID,input,id));return record(await mobilityMeritRelatedSave(client,resource,parentID,input,id));},async deleteRelated(resource,parentID,id){if(isGovernanceRelated(resource)){if(resource==="governance-bodies")throw new Error("education_related_delete_unsupported");await relatedDelete(client,resource,parentID,id);return;}if(isPersonnelEvaluationRelated(resource)){await personnelEvaluationRelatedDelete(client,resource,parentID,id);return;}await mobilityMeritRelatedDelete(client,resource,parentID,id);},async relatedPdf(resource,parentID,id){if(resource==="meeting-documents"||resource==="meeting-minutes"||resource==="meeting-resolutions"||resource==="managerial-documents")return relatedPdf(client,resource,parentID,id);if(resource==="evaluation-appeals"||resource==="evaluation-result-issues")return personnelEvaluationRelatedPdf(client,resource,parentID,id);if(resource==="mobility-appeals"||resource==="mobility-final-decisions"||resource==="mobility-result-issues"||resource==="merit-appeals"||resource==="merit-final-decisions"||resource==="merit-result-issues")return mobilityMeritRelatedPdf(client,resource,parentID,id);throw new Error("education_related_pdf_unsupported");},
  async portfolioDocuments(recordID,input={}) { return unwrap(client.GET("/api/education/portfolios/records/{recordID}/documents", { params: { path: { recordID }, query: portfolioDocumentParams(input) } })); },
  async portfolioDocument(recordID,documentID) { return unwrap(client.GET("/api/education/portfolios/records/{recordID}/documents/{documentID}", { params: { path: { recordID, documentID } } })); },
  async createPortfolioDocument(recordID,input: CreatePortfolioDocumentInput) { return unwrap(client.POST("/api/education/portfolios/records/{recordID}/documents", { params: { path: { recordID } }, body: input })); },
  async updatePortfolioDocument(recordID,documentID,input: CreatePortfolioDocumentInput) { return unwrap(client.PATCH("/api/education/portfolios/records/{recordID}/documents/{documentID}", { params: { path: { recordID, documentID } }, body: input })); },
  async deletePortfolioDocument(recordID,documentID) { await unwrap(client.DELETE("/api/education/portfolios/records/{recordID}/documents/{documentID}", { params: { path: { recordID, documentID } } })); },
  async portfolioChecklist(recordID,input={}) { return unwrap(client.GET("/api/education/portfolios/records/{recordID}/checklist", { params: { path: { recordID }, query: portfolioChecklistParams(input) } })); },
  async portfolioChecklistItem(recordID,itemID) { return unwrap(client.GET("/api/education/portfolios/records/{recordID}/checklist/{itemID}", { params: { path: { recordID, itemID } } })); },
  async createPortfolioChecklistItem(recordID,input: CreatePortfolioChecklistItemInput) { return unwrap(client.POST("/api/education/portfolios/records/{recordID}/checklist", { params: { path: { recordID } }, body: input })); },
  async updatePortfolioChecklistItem(recordID,itemID,input: CreatePortfolioChecklistItemInput) { return unwrap(client.PATCH("/api/education/portfolios/records/{recordID}/checklist/{itemID}", { params: { path: { recordID, itemID } }, body: input })); },
  async deletePortfolioChecklistItem(recordID,itemID) { await unwrap(client.DELETE("/api/education/portfolios/records/{recordID}/checklist/{itemID}", { params: { path: { recordID, itemID } } })); },
  async portfolioOpis(recordID,input={}) { return unwrap(client.GET("/api/education/portfolios/records/{recordID}/opis", { params: { path: { recordID }, query: portfolioOpisParams(input) } })); },
  async portfolioOpisEntry(recordID,itemID) { return unwrap(client.GET("/api/education/portfolios/records/{recordID}/opis/{itemID}", { params: { path: { recordID, itemID } } })); },
  async createPortfolioOpisEntry(recordID,input: CreatePortfolioOpisEntryInput) { return unwrap(client.POST("/api/education/portfolios/records/{recordID}/opis", { params: { path: { recordID } }, body: input })); },
  async updatePortfolioOpisEntry(recordID,itemID,input: CreatePortfolioOpisEntryInput) { return unwrap(client.PATCH("/api/education/portfolios/records/{recordID}/opis/{itemID}", { params: { path: { recordID, itemID } }, body: input })); },
  async deletePortfolioOpisEntry(recordID,itemID) { await unwrap(client.DELETE("/api/education/portfolios/records/{recordID}/opis/{itemID}", { params: { path: { recordID, itemID } } })); },
  async portfolioCustody(recordID,input={}) { return unwrap(client.GET("/api/education/portfolios/records/{recordID}/custody", { params: { path: { recordID }, query: portfolioCustodyParams(input) } })); },
  async portfolioCustodyEvent(recordID,itemID) { return unwrap(client.GET("/api/education/portfolios/records/{recordID}/custody/{itemID}", { params: { path: { recordID, itemID } } })); },
  async createPortfolioCustodyEvent(recordID,input: CreatePortfolioCustodyEventInput) { return unwrap(client.POST("/api/education/portfolios/records/{recordID}/custody", { params: { path: { recordID } }, body: input })); },
  async updatePortfolioCustodyEvent(recordID,itemID,input: CreatePortfolioCustodyEventInput) { return unwrap(client.PATCH("/api/education/portfolios/records/{recordID}/custody/{itemID}", { params: { path: { recordID, itemID } }, body: input })); },
  async deletePortfolioCustodyEvent(recordID,itemID) { await unwrap(client.DELETE("/api/education/portfolios/records/{recordID}/custody/{itemID}", { params: { path: { recordID, itemID } } })); },
  async portfolioReviews(recordID,input={}) { return unwrap(client.GET("/api/education/portfolios/records/{recordID}/reviews", { params: { path: { recordID }, query: portfolioReviewParams(input) } })); },
  async portfolioReview(recordID,itemID) { return unwrap(client.GET("/api/education/portfolios/records/{recordID}/reviews/{itemID}", { params: { path: { recordID, itemID } } })); },
  async createPortfolioReview(recordID,input: CreatePortfolioReviewEventInput) { return unwrap(client.POST("/api/education/portfolios/records/{recordID}/reviews", { params: { path: { recordID } }, body: input })); },
  async updatePortfolioReview(recordID,itemID,input: CreatePortfolioReviewEventInput) { return unwrap(client.PATCH("/api/education/portfolios/records/{recordID}/reviews/{itemID}", { params: { path: { recordID, itemID } }, body: input })); },
  async deletePortfolioReview(recordID,itemID) { await unwrap(client.DELETE("/api/education/portfolios/records/{recordID}/reviews/{itemID}", { params: { path: { recordID, itemID } } })); },
  async portfolioTransferHistory(recordID,input={}) { return unwrap(client.GET("/api/education/portfolios/records/{recordID}/transfers", { params: { path: { recordID }, query: portfolioTransferParams(input) } })); },
  async portfolioValorifications(recordID,input={}) { return unwrap(client.GET("/api/education/portfolios/records/{recordID}/valorifications", { params: { path: { recordID }, query: portfolioValorificationParams(input) } })); },
  async portfolioValorification(recordID,itemID) { return unwrap(client.GET("/api/education/portfolios/records/{recordID}/valorifications/{itemID}", { params: { path: { recordID, itemID } } })); },
  async createPortfolioValorification(recordID,input: CreatePortfolioValorificationEventInput) { return unwrap(client.POST("/api/education/portfolios/records/{recordID}/valorifications", { params: { path: { recordID } }, body: input })); },
  async updatePortfolioValorification(recordID,itemID,input: CreatePortfolioValorificationEventInput) { return unwrap(client.PATCH("/api/education/portfolios/records/{recordID}/valorifications/{itemID}", { params: { path: { recordID, itemID } }, body: input })); },
  async deletePortfolioValorification(recordID,itemID) { await unwrap(client.DELETE("/api/education/portfolios/records/{recordID}/valorifications/{itemID}", { params: { path: { recordID, itemID } } })); },
  async portfolioSections(input={}) { return unwrap(client.GET("/api/education/portfolios/sections", { params: { query: portfolioSectionParams(input) } })); },
  async educationRequirements(input={}) { return unwrap(client.GET("/api/education/requirements", { params: { query: requirementParams(input) } })); },
  async taxonomyCatalog(input: TaxonomyCatalogQuery = {}) { return unwrap(client.GET("/api/education/taxonomies", { params: { query: { domains: input.domains } } })); },
  async metadata<R extends EducationMetadataResource>(resource: R, parentID?: string): Promise<EducationMetadataResultByResource[R]> { return metadataResult(client, resource, parentID); },
  async command(command: EducationCommand, portfolioID: string) {
    switch (command) {
      case "portfolio-opis-regenerate": return unwrap(client.POST("/api/education/portfolios/records/{recordID}/opis/regenerate", { params: { path: { recordID: portfolioID } } }));
      case "portfolio-return": return unwrap(client.POST("/api/education/portfolios/records/{recordID}/return", { params: { path: { recordID: portfolioID } } }));
      case "portfolio-verify": return unwrap(client.POST("/api/education/portfolios/records/{recordID}/verify", { params: { path: { recordID: portfolioID } } }));
    }
  },
  async ownPortfolios(input = {}) { return unwrap(client.GET("/api/education/portfolios/me", { params: { query: { page: input.page ?? 1, pageSize: input.pageSize ?? 50, sort: input.sort === "school_year" ? "school_year" : input.sort === "status" ? "status" : input.sort === "updated_at" ? "updated_at" : undefined, direction: input.direction, "filter.school_year": input.filters?.school_year, "filter.status": input.filters?.status } } })); },
  async ownPortfolio(id) { return unwrap(client.GET("/api/education/portfolios/me/{recordID}", { params: { path: { recordID: id } } })); },
  async createOwnPortfolio(input) { return unwrap(client.POST("/api/education/portfolios/me", { body: input })); },
  async updateOwnPortfolio(id, input) { return unwrap(client.PATCH("/api/education/portfolios/me/{recordID}", { params: { path: { recordID: id } }, body: input })); },
  async submitOwnPortfolio(id) { return unwrap(client.POST("/api/education/portfolios/me/{recordID}/submit", { params: { path: { recordID: id } } })); },
  async ownPortfolioDeclarations(id) { return unwrap(client.GET("/api/education/portfolios/me/{recordID}/declarations", { params: { path: { recordID: id } } })); },
  async acknowledgeOwnPortfolioDeclaration(id, declarationType) { return unwrap(client.POST("/api/education/portfolios/me/{recordID}/declarations/{declarationType}/acknowledgements", { params: { path: { recordID: id, declarationType } }, body: { confirmed: true } })); },
  async portfolioProcedures(input = {}) { return unwrap(client.GET("/api/education/portfolios/procedures", { params: { query: { page: input.page ?? 1, pageSize: input.pageSize ?? 50, sort: input.sort === "procedure_code" ? "procedure_code" : input.sort === "title" ? "title" : input.sort === "lifecycle_status" ? "lifecycle_status" : undefined, direction: input.direction, "filter.procedure_code": input.filters?.procedure_code, "filter.title": input.filters?.title, "filter.lifecycle_status": input.filters?.lifecycle_status } } })); },
  async portfolioProcedure(id) { return unwrap(client.GET("/api/education/portfolios/procedures/{procedureID}", { params: { path: { procedureID: id } } })); },
  async createPortfolioProcedure(input) { return unwrap(client.POST("/api/education/portfolios/procedures", { body: input })); },
  async updatePortfolioProcedure(id, input) { return unwrap(client.PATCH("/api/education/portfolios/procedures/{procedureID}", { params: { path: { procedureID: id } }, body: input })); },
  async portfolioProcedureRules(id, input = {}) { return unwrap(client.GET("/api/education/portfolios/procedures/{procedureID}/section-rules", { params: { path: { procedureID: id }, query: query({ ...input, sort: input.sort ?? "sort_order", direction: input.direction ?? "asc" }) } })); },
  async replacePortfolioProcedureRules(id, input) { await unwrap(client.PUT("/api/education/portfolios/procedures/{procedureID}/section-rules", { params: { path: { procedureID: id } }, body: input })); },
  async transitionPortfolioProcedure(id, transition, input) {
    const params = { path: { procedureID: id } };
    switch (transition) {
      case "approve": return unwrap(client.POST("/api/education/portfolios/procedures/{procedureID}/approve", { params, body: input }));
      case "publish": return unwrap(client.POST("/api/education/portfolios/procedures/{procedureID}/publish", { params, body: input }));
      case "supersede": return unwrap(client.POST("/api/education/portfolios/procedures/{procedureID}/supersede", { params, body: input }));
      case "withdraw": return unwrap(client.POST("/api/education/portfolios/procedures/{procedureID}/withdraw", { params, body: input }));
    }
  },
  async ownPortfolioRelated(id, resource, input = {}) {
    const pageParams = { page: input.page ?? 1, pageSize: input.pageSize ?? 50, direction: input.direction };
    switch (resource) {
      case "documents": return page(await unwrap(client.GET("/api/education/portfolios/me/{recordID}/documents", { params: { path: { recordID: id }, query: { ...pageParams, sort: input.sort === "section_code" ? "section_code" : input.sort === "document_title" ? "document_title" : input.sort === "evidence_type" ? "evidence_type" : input.sort === "authenticity_status" ? "authenticity_status" : input.sort === "issued_on" ? "issued_on" : undefined, "filter.section_code": input.filters?.section_code } } })));
      case "checklist": return page(await unwrap(client.GET("/api/education/portfolios/me/{recordID}/checklist", { params: { path: { recordID: id }, query: { ...pageParams, sort: input.sort === "requirement_code" ? "requirement_code" : input.sort === "requirement_label" ? "requirement_label" : input.sort === "section_code" ? "section_code" : input.sort === "status" ? "status" : input.sort === "document_count" ? "document_count" : undefined, "filter.requirement_code": input.filters?.requirement_code } } })));
      case "opis": return page(await unwrap(client.GET("/api/education/portfolios/me/{recordID}/opis", { params: { path: { recordID: id }, query: { ...pageParams, sort: input.sort === "section_code" ? "section_code" : input.sort === "component_code" ? "component_code" : input.sort === "entry_title" ? "entry_title" : input.sort === "chronological_index" ? "chronological_index" : input.sort === "document_reference" ? "document_reference" : undefined, "filter.section_code": input.filters?.section_code } } })));
      case "reviews": return page(await unwrap(client.GET("/api/education/portfolios/me/{recordID}/reviews", { params: { path: { recordID: id }, query: { ...pageParams, sort: input.sort === "review_code" ? "review_code" : input.sort === "review_stage" ? "review_stage" : input.sort === "outcome" ? "outcome" : input.sort === "reviewer_name" ? "reviewer_name" : input.sort === "reviewed_on" ? "reviewed_on" : undefined, "filter.review_code": input.filters?.review_code } } })));
    }
  },
  async regenerateOwnPortfolioOpis(id) { return unwrap(client.POST("/api/education/portfolios/me/{recordID}/opis/regenerate", { params: { path: { recordID: id } } })); },
  async recordPortfolioCessation(id, input) { return unwrap(client.POST("/api/education/portfolios/records/{recordID}/activity-cessation", { params: { path: { recordID: id } }, body: input })); },
  async setPortfolioLegalHold(id, input) { return unwrap(client.POST("/api/education/portfolios/records/{recordID}/legal-hold", { params: { path: { recordID: id } }, body: input })); },
  async createOwnPortfolioDocument(id, input): Promise<PortfolioDocument> { return unwrap(client.POST("/api/education/portfolios/me/{recordID}/documents", { params: { path: { recordID: id } }, body: input })); },
  async deleteOwnPortfolioDocument(portfolioID, documentID) { await unwrap(client.DELETE("/api/education/portfolios/me/{recordID}/documents/{documentID}", { params: { path: { recordID: portfolioID, documentID } } })); },
  async ownPortfolioArchiveDocuments(input = {}) { return unwrap(client.GET("/api/education/portfolios/me/archive-documents", { params: { query: { page: input.page ?? 1, pageSize: input.pageSize ?? 25, sort: input.sort === "current_version_no" ? "current_version_no" : "title", direction: input.direction ?? "asc", "filter.title": input.filters?.title } } })); },
  async attachmentGrants(input = {}) { return unwrap(client.GET("/api/education/portfolios/archive-attachment-grants", { params: { query: { page: input.page ?? 1, pageSize: input.pageSize ?? 50, sort: input.sort === "document_title" ? "document_title" : input.sort === "grantee_name" ? "grantee_name" : undefined, direction: input.direction, "filter.document_title": input.filters?.document_title, "filter.grantee_name": input.filters?.grantee_name } } })); },
  async eligibleAttachmentDocuments(input = {}) { return unwrap(client.GET("/api/education/portfolios/archive-attachment-grants/eligible-documents", { params: { query: { page: input.page ?? 1, pageSize: input.pageSize ?? 50, sort: input.sort === "title" ? "title" : input.sort === "current_version_no" ? "current_version_no" : undefined, direction: input.direction, "filter.title": input.filters?.title } } })); },
  async eligibleAttachmentUsers(input = {}) { return unwrap(client.GET("/api/education/portfolios/archive-attachment-grants/eligible-users", { params: { query: { page: input.page ?? 1, pageSize: input.pageSize ?? 100, sort: input.sort === "name" ? "name" : input.sort === "role_title" ? "role_title" : undefined, direction: input.direction, "filter.name": input.filters?.name } } })); },
  async createAttachmentGrant(input) { return unwrap(client.POST("/api/education/portfolios/archive-attachment-grants", { body: input })); },
  async deleteAttachmentGrant(id) { await unwrap(client.DELETE("/api/education/portfolios/archive-attachment-grants/{grantID}", { params: { path: { grantID: id } } })); },
  async createPortfolioExportManifest(id): Promise<PortfolioEvidenceManifestResponse> { return unwrap(client.POST("/api/education/portfolios/records/{recordID}/export-manifests", { params: { path: { recordID: id } } })); },
 };
}
