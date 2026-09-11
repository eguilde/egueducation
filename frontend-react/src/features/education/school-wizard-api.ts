import type { ContractClient } from "../../api/client";
import type { components } from "../../api/generated";
import type { WizardKind, WizardValues } from "./wizards";

type EligibleGovernanceUser = components["schemas"]["EligibleGovernanceUser"];
type GovernanceMeeting = components["schemas"]["GovernanceMeeting"];
type GovernanceMeetingVote = components["schemas"]["GovernanceMeetingVote"];
type EligiblePortfolioOwner = components["schemas"]["EligiblePortfolioOwner"];
type WizardResult =
  | components["schemas"]["GovernanceMeeting"]
  | components["schemas"]["GovernanceMinuteItem"]
  | components["schemas"]["GovernanceMeetingVote"]
  | components["schemas"]["GovernanceResolution"]
  | components["schemas"]["ManagerialDossier"]
  | components["schemas"]["PersonnelRecord"]
  | components["schemas"]["PersonnelEvaluation"]
  | components["schemas"]["PersonnelDeclaration"]
  | components["schemas"]["MobilityCase"]
  | components["schemas"]["MeritGrant"]
  | components["schemas"]["PortfolioRecord"];

export interface SchoolWizardApi {
  create(kind: WizardKind, payload: WizardValues): Promise<WizardResult>;
  eligibleGovernanceUsers(query?: SchoolSelectorQuery): Promise<SchoolSelectorPage<EligibleGovernanceUser>>;
  eligibleGovernanceMeetings(query?: SchoolSelectorQuery): Promise<SchoolSelectorPage<GovernanceMeeting>>;
  governanceMeetingDetail(id: string): Promise<GovernanceMeeting>;
  governanceMeetingVotes(meetingID: string, query?: SchoolSelectorQuery): Promise<SchoolSelectorPage<GovernanceMeetingVote>>;
  eligiblePortfolioOwners(query?: SchoolSelectorQuery): Promise<SchoolSelectorPage<EligiblePortfolioOwner>>;
}

/** Server-side selector contract. The caller must never silently truncate canonical identities. */
export type SchoolSelectorQuery = { page?: number; pageSize?: number; q?: string };
export type SchoolSelectorPage<T> = { items: T[]; total: number; page: number; pageSize: number };

type ApiResult<T> = { data?: T; error?: unknown; response: Response };

function resultOrThrow<T>(result: ApiResult<T>): T {
  if (result.response.ok && result.data !== undefined) return result.data;
  const code = typeof result.error === "object" && result.error !== null && "code" in result.error
    ? String(result.error.code)
    : `education_wizard_${result.response.status}`;
  throw new Error(code);
}

function selectorPage<T>(result: ApiResult<{ items: T[]; total?: number; page?: number; pageSize?: number }>, fallback: SchoolSelectorQuery): SchoolSelectorPage<T> {
  const page = resultOrThrow(result);
  return { items: page.items, total: page.total ?? page.items.length, page: page.page ?? fallback.page ?? 1, pageSize: page.pageSize ?? fallback.pageSize ?? 25 };
}

const text = (values: WizardValues, key: string) => String(values[key] ?? "").trim();
const number = (values: WizardValues, key: string) => {
  const value = values[key];
  return typeof value === "number" && Number.isFinite(value) ? value : 0;
};
const flag = (values: WizardValues, key: string) => values[key] === true;
const enumText = <const T extends readonly string[]>(values: WizardValues, key: string, allowed: T): T[number] => {
  const value = text(values, key);
  if (!(allowed as readonly string[]).includes(value)) throw new Error(`education_wizard_invalid_${key}`);
  return value as T[number];
};

const meeting = (values: WizardValues): components["schemas"]["CreateGovernanceMeetingRequest"] => ({
  school_year: text(values, "school_year"),
  organism: text(values, "organism"),
  title: text(values, "title"),
  meeting_type: text(values, "meeting_type"),
  status: text(values, "status"),
  quorum_required: number(values, "quorum_required"),
  participants_count: number(values, "participants_count"),
  meeting_date: text(values, "meeting_date"),
  location: text(values, "location") || undefined,
  chairperson_user_id: text(values, "chairperson_user_id"),
  secretary_user_id: text(values, "secretary_user_id"),
  summary: text(values, "summary") || undefined,
});

const minute = (values: WizardValues): components["schemas"]["CreateGovernanceMinuteItemRequest"] => ({
  agenda_order: number(values, "agenda_order"),
  topic_title: text(values, "topic_title"),
  discussion_summary: text(values, "discussion_summary"),
  decision_summary: text(values, "decision_summary"),
  responsible_party: text(values, "responsible_party") || undefined,
  due_on: text(values, "due_on") || undefined,
  follow_up_status: text(values, "follow_up_status"),
  requires_publication: flag(values, "requires_publication"),
  notes: text(values, "notes") || undefined,
});

const vote = (values: WizardValues): components["schemas"]["CreateGovernanceMeetingVoteRequest"] => ({
  subject_title: text(values, "subject_title"),
  agenda_order: number(values, "agenda_order"),
  decision_type: text(values, "decision_type"),
  votes_for: number(values, "votes_for"),
  votes_against: number(values, "votes_against"),
  abstentions: number(values, "abstentions"),
  outcome: text(values, "outcome"),
  requires_follow_up: flag(values, "requires_follow_up"),
  legal_basis: text(values, "legal_basis") || undefined,
  notes: text(values, "notes") || undefined,
});

const resolution = (values: WizardValues): components["schemas"]["CreateGovernanceResolutionRequest"] => ({
  vote_id: text(values, "vote_id"),
  title: text(values, "title"),
  resolution_type: text(values, "resolution_type"),
  publication_status: text(values, "publication_status"),
  anonymization_state: text(values, "anonymization_state"),
  issued_on: text(values, "issued_on"),
  signed_by: text(values, "signed_by") || undefined,
  notes: text(values, "notes") || undefined,
});

const managerial = (values: WizardValues): components["schemas"]["CreateManagerialDossierRequest"] => ({
  school_year: text(values, "school_year"),
  dossier_type: text(values, "dossier_type"),
  title: text(values, "title"),
  status: text(values, "status"),
  owner_name: text(values, "owner_name"),
  due_on: text(values, "due_on"),
  publication_required: flag(values, "publication_required"),
  summary: text(values, "summary") || undefined,
});

const personnel = (values: WizardValues): components["schemas"]["CreatePersonnelRecordRequest"] => ({
  full_name: text(values, "full_name"),
  role_title: text(values, "role_title"),
  employment_type: enumText(values, "employment_type", ["titular", "suplinitor", "plata_cu_ora", "auxiliar"] as const),
  status: enumText(values, "status", ["active", "on_leave", "vacant", "inactive"] as const),
  evaluation_status: enumText(values, "evaluation_status", ["draft", "in_review", "finalized"] as const),
  mobility_stage: enumText(values, "mobility_stage", ["none", "transfer", "detasare", "restrangere"] as const),
  school_year: text(values, "school_year"),
  assigned_unit: text(values, "assigned_unit") || undefined,
  phone: text(values, "phone") || undefined,
  email: text(values, "email") || undefined,
  has_portfolio: flag(values, "has_portfolio"),
  notes: text(values, "notes") || undefined,
});

const evaluation = (values: WizardValues): components["schemas"]["CreatePersonnelEvaluationRequest"] => ({
  employee_code: text(values, "employee_code"),
  full_name: text(values, "full_name"),
  role_title: text(values, "role_title"),
  school_year: text(values, "school_year"),
  status: text(values, "status"),
  score: number(values, "score"),
  evaluator_name: text(values, "evaluator_name"),
  finalized_on: text(values, "finalized_on") || undefined,
  summary: text(values, "summary") || undefined,
});

const declaration = (values: WizardValues): components["schemas"]["CreatePersonnelDeclarationRequest"] => ({
  employee_code: text(values, "employee_code"),
  full_name: text(values, "full_name"),
  declaration_type: text(values, "declaration_type"),
  status: text(values, "status"),
  school_year: text(values, "school_year"),
  submitted_on: text(values, "submitted_on"),
  valid_until: text(values, "valid_until") || undefined,
  summary: text(values, "summary") || undefined,
});

const mobility = (values: WizardValues): components["schemas"]["CreateMobilityCaseRequest"] => ({
  employee_code: text(values, "employee_code"),
  full_name: text(values, "full_name"),
  school_year: text(values, "school_year"),
  request_type: text(values, "request_type"),
  stage: text(values, "stage"),
  status: text(values, "status"),
  source_school: text(values, "source_school"),
  destination_school: text(values, "destination_school") || undefined,
  submitted_on: text(values, "submitted_on"),
  reviewed_by: text(values, "reviewed_by") || undefined,
  notes: text(values, "notes") || undefined,
});

const merit = (values: WizardValues): components["schemas"]["CreateMeritGrantRequest"] => ({
  full_name: text(values, "full_name"),
  role_title: text(values, "role_title"),
  school_year: text(values, "school_year"),
  category: text(values, "category"),
  status: text(values, "status"),
  score: number(values, "score"),
  committee_name: text(values, "committee_name"),
  decision_date: text(values, "decision_date"),
  funded: flag(values, "funded"),
  notes: text(values, "notes") || undefined,
});

const portfolio = (values: WizardValues): components["schemas"]["CreatePortfolioRecordRequest"] => ({
  owner_user_id: text(values, "owner_user_id"),
  owner_personnel_id: text(values, "owner_personnel_id"),
  owner_name: text(values, "owner_name"),
  owner_role: text(values, "owner_role"),
  school_year: text(values, "school_year"),
  status: text(values, "status"),
  section_count: number(values, "section_count"),
  last_updated_on: text(values, "last_updated_on"),
  transfer_status: text(values, "transfer_status"),
  authenticity_declared: flag(values, "authenticity_declared"),
  consent_captured: flag(values, "consent_captured"),
  custodian: text(values, "custodian") || undefined,
  notes: text(values, "notes") || undefined,
});

export function createSchoolWizardApi(client: Pick<ContractClient, "GET" | "POST">): SchoolWizardApi {
  return {
    async eligibleGovernanceUsers(input = {}) {
      const result = await client.GET("/api/education/governance/eligible-users", {
        params: { query: { page: input.page ?? 1, pageSize: input.pageSize ?? 25, sort: "name", direction: "asc", "filter.name": input.q?.trim() || undefined } },
      });
      return selectorPage(result, input);
    },
    async eligibleGovernanceMeetings(input = {}) {
      const result = await client.GET("/api/education/governance/meetings", {
        params: { query: { page: input.page ?? 1, pageSize: input.pageSize ?? 25, sort: "meeting_date", direction: "desc", "filter.title": input.q?.trim() || undefined } },
      });
      return selectorPage(result, input);
    },
    async governanceMeetingDetail(id) {
      return resultOrThrow(await client.GET("/api/education/governance/meetings/{meetingID}", { params: { path: { meetingID: id } } }));
    },
    async governanceMeetingVotes(meetingID, input = {}) {
      const result = await client.GET("/api/education/governance/meetings/{meetingID}/votes", {
        params: { path: { meetingID }, query: { page: input.page ?? 1, pageSize: input.pageSize ?? 25, sort: "agenda_order", direction: "asc", "filter.subject_title": input.q?.trim() || undefined } },
      });
      return selectorPage(result, input);
    },
    async eligiblePortfolioOwners(input = {}) {
      const result = await client.GET("/api/education/portfolios/eligible-owners", {
        params: { query: { page: input.page ?? 1, pageSize: input.pageSize ?? 25, sort: "display_name", direction: "asc", "filter.display_name": input.q?.trim() || undefined } },
      });
      return selectorPage(result, input);
    },
    async create(kind, values) {
      switch (kind) {
        case "caMeeting":
          return resultOrThrow(await client.POST("/api/education/governance/meetings", { body: meeting(values) }));
        case "minute":
          return resultOrThrow(await client.POST("/api/education/governance/meetings/{meetingID}/minutes", { params: { path: { meetingID: text(values, "meeting_id") } }, body: minute(values) }));
        case "vote":
          return resultOrThrow(await client.POST("/api/education/governance/meetings/{meetingID}/votes", { params: { path: { meetingID: text(values, "meeting_id") } }, body: vote(values) }));
        case "resolution":
          return resultOrThrow(await client.POST("/api/education/governance/meetings/{meetingID}/resolutions", { params: { path: { meetingID: text(values, "meeting_id") } }, body: resolution(values) }));
        case "managerial":
          return resultOrThrow(await client.POST("/api/education/managerial/records", { body: managerial(values) }));
        case "personnel":
          return resultOrThrow(await client.POST("/api/education/personnel/records", { body: personnel(values) }));
        case "evaluation":
          return resultOrThrow(await client.POST("/api/education/evaluations/records", { body: evaluation(values) }));
        case "declaration":
          return resultOrThrow(await client.POST("/api/education/declarations/records", { body: declaration(values) }));
        case "mobility":
          return resultOrThrow(await client.POST("/api/education/mobility/records", { body: mobility(values) }));
        case "merit":
          return resultOrThrow(await client.POST("/api/education/gradatii/records", { body: merit(values) }));
        case "portfolio":
          return resultOrThrow(await client.POST("/api/education/portfolios/records", { body: portfolio(values) }));
      }
    },
  };
}
