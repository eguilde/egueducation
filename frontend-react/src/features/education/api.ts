import type {
  DirectorCockpit,
  EducationApi,
  EducationListQuery,
  EducationPage,
  GovernanceDashboard,
  GovernanceMeeting,
  EducationRecord,
  EducationRecordInput,
  EducationRecordsDomain,
} from "./types";

export type AuthenticatedFetcher = (
  input: RequestInfo | URL,
  init?: RequestInit,
) => Promise<Response>;

const toPage = <T,>(value: T[] | Partial<EducationPage<T>>): EducationPage<T> => {
  if (Array.isArray(value)) {
    return { items: value, total: value.length, page: 1, pageSize: value.length };
  }
  const rawItems = (value as { items?: unknown }).items;
  const items = Array.isArray(rawItems)
    ? rawItems as T[]
    : rawItems && typeof rawItems === "object"
      ? Object.values(rawItems).flatMap((group) => Array.isArray(group) ? group : []) as T[]
      : [];
  return {
    items,
    total: value.total ?? items.length,
    page: value.page ?? 1,
    pageSize: value.pageSize ?? Math.max(items.length, 1),
  };
};

export function createEducationApi(
  fetcher: AuthenticatedFetcher,
  apiBase = "/api",
): EducationApi {
  const request = async <T,>(path: string, init?: RequestInit): Promise<T> => {
    const response = await fetcher(`${apiBase}${path}`, {
      ...init,
      headers: {
        Accept: "application/json",
        ...(init?.headers ?? {}),
      },
    });
    if (!response.ok) {
      const body = await response.json().catch(() => undefined) as { code?: string } | undefined;
      throw new Error(body?.code ?? `education_request_${response.status}`);
    }
    if (response.status === 204) return undefined as T;
    return response.json() as Promise<T>;
  };
  const recordsPaths: Record<EducationRecordsDomain, string> = {
    decisions: "/education/decisions/records",
    managerial: "/education/managerial/records",
    regulations: "/education/regulations/records",
    committees: "/education/committees/records",
    personnel: "/education/personnel/records",
    evaluations: "/education/evaluations/records",
    declarations: "/education/declarations/records",
    mobility: "/education/mobility/records",
    merit: "/education/gradatii/records",
    portfolios: "/education/portfolios/records",
    compliance: "/education/compliance/publications",
  };
  const pdfPaths = {
    managerial: "/education/managerial/records",
    evaluations: "/education/evaluations/records",
    mobility: "/education/mobility/records",
    merit: "/education/gradatii/records",
    portfolios: "/education/portfolios/records",
  } as const;
  const listQuery = (input: EducationListQuery = {}) => {
    const query = new URLSearchParams({ page: String(input.page ?? 1), pageSize: String(input.pageSize ?? 50) });
    if (input.sort) query.set("sort", input.sort);
    if (input.direction) query.set("direction", input.direction);
    if (input.q?.trim()) query.set("q", input.q.trim());
    Object.entries(input.filters ?? {}).forEach(([name, value]) => { if (value?.trim()) query.set(`filter.${name}`, value.trim()); });
    return query;
  };

  return {
    governanceDashboard: () => request<GovernanceDashboard>("/education/dashboard"),
    directorCockpit: () => request<DirectorCockpit>("/education/director/cockpit"),
    async governanceMeetings(input: EducationListQuery = {}) {
      const query = listQuery(input);
      return toPage(await request<GovernanceMeeting[] | EducationPage<GovernanceMeeting>>(`/education/governance/meetings?${query}`));
    },
    governanceMeetingDetail: (id) => request<GovernanceMeeting>(`/education/governance/meetings/${encodeURIComponent(id)}`),
    saveGovernanceMeeting: (input, id?) => request<GovernanceMeeting>(id ? `/education/governance/meetings/${encodeURIComponent(id)}` : "/education/governance/meetings", { method: id ? "PUT" : "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(input) }),
    deleteGovernanceMeeting: (id) => request<void>(`/education/governance/meetings/${encodeURIComponent(id)}`, { method: "DELETE" }),
    records: async (domain, input = {}) => toPage(await request<EducationRecord[] | EducationPage<EducationRecord>>(`${recordsPaths[domain]}?${listQuery(input)}`)),
    recordDetail: (domain, id) => request<EducationRecord>(`${recordsPaths[domain]}/${encodeURIComponent(id)}`),
    saveRecord: (domain, input: EducationRecordInput, id?) => request<EducationRecord>(id ? `${recordsPaths[domain]}/${encodeURIComponent(id)}` : recordsPaths[domain], {
      method: id ? "PUT" : "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(input),
    }),
    deleteRecord: (domain, id) => request<void>(`${recordsPaths[domain]}/${encodeURIComponent(id)}`, { method: "DELETE" }),
    async recordPdf(domain, id) {
      const response = await fetcher(`${apiBase}${pdfPaths[domain]}/${encodeURIComponent(id)}/pdf`, { headers: { Accept: "application/pdf" } });
      if (!response.ok) throw new Error(`education_pdf_${response.status}`);
      return response.blob();
    },
    relatedRecords: async (path, input = {}) => toPage(await request<EducationRecord[] | EducationPage<EducationRecord>>(`${path}?${listQuery(input)}`)),
    relatedDetail: (path, id) => request<EducationRecord>(`${path}/${encodeURIComponent(id)}`),
    saveRelated: (path, input, id?) => request<EducationRecord>(id ? `${path}/${encodeURIComponent(id)}` : path, { method: id ? "PUT" : "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(input) }),
    deleteRelated: (path, id) => request<void>(`${path}/${encodeURIComponent(id)}`, { method: "DELETE" }),
    async relatedPdf(path, id) {
      const response = await fetcher(`${apiBase}${path}/${encodeURIComponent(id)}/pdf`, { headers: { Accept: "application/pdf" } });
      if (!response.ok) throw new Error(`education_related_pdf_${response.status}`);
      return response.blob();
    },
    async exportFile(format) {
      const response = await fetcher(`${apiBase}/education/exports/${format}`, { headers: { Accept: format === "pdf" ? "application/pdf" : "text/csv" } });
      if (!response.ok) throw new Error(`education_export_${response.status}`);
      return response.blob();
    },
    metadata: (path) => request<Record<string, unknown>>(path),
    command: (path) => request<void>(path, { method: "POST" }),
  };
}
