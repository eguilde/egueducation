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

/**
 * Most Education domains intentionally expose their own record schema.  Keeping
 * a small lossless envelope here lets the UI render every authorised domain
 * without pretending that differently shaped dossiers are interchangeable.
 */
export interface EducationRecord {
  id: string;
  [key: string]: unknown;
}

export type EducationRecordsDomain = Exclude<EducationArea["id"], "overview" | "governance">;
export type EducationPdfRecordsDomain = "managerial" | "evaluations" | "mobility" | "merit" | "portfolios";

export interface EducationRecordInput {
  [key: string]: string | number | boolean | undefined;
}

export interface EducationApi {
  governanceDashboard(): Promise<GovernanceDashboard>;
  directorCockpit(): Promise<DirectorCockpit>;
  governanceMeetings(input?: EducationListQuery): Promise<EducationPage<GovernanceMeeting>>;
  governanceMeetingDetail(id: string): Promise<GovernanceMeeting>;
  saveGovernanceMeeting(input: EducationRecordInput, id?: string): Promise<GovernanceMeeting>;
  deleteGovernanceMeeting(id: string): Promise<void>;
  records(domain: EducationRecordsDomain, input?: EducationListQuery): Promise<EducationPage<EducationRecord>>;
  recordDetail(domain: EducationRecordsDomain, id: string): Promise<EducationRecord>;
  saveRecord(domain: EducationRecordsDomain, input: EducationRecordInput, id?: string): Promise<EducationRecord>;
  deleteRecord(domain: EducationRecordsDomain, id: string): Promise<void>;
  recordPdf(domain: EducationPdfRecordsDomain, id: string): Promise<Blob>;
  relatedRecords(path: string, input?: EducationListQuery): Promise<EducationPage<EducationRecord>>;
  relatedDetail(path: string, id: string): Promise<EducationRecord>;
  saveRelated(path: string, input: EducationRecordInput, id?: string): Promise<EducationRecord>;
  deleteRelated(path: string, id: string): Promise<void>;
  relatedPdf(path: string, id: string): Promise<Blob>;
  exportFile(format: "pdf" | "csv"): Promise<Blob>;
  metadata(path: string): Promise<Record<string, unknown>>;
  command(path: string): Promise<void>;
}

export interface EducationListQuery {
  page?: number;
  pageSize?: number;
  sort?: string;
  direction?: "asc" | "desc";
  q?: string;
  filters?: Record<string, string | undefined>;
}
