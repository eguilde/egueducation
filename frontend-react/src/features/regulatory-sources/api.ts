import type { ContractClient } from "../../api/client";
import type { components } from "../../api/generated";
import { validateGetApiRegulatorySourcesResponse, validatePostApiRegulatorySourcesResponse, validatePostApiRegulatorySourcesSourceidActivateResponse, validatePostApiRegulatorySourcesSourceidVerifyResponse } from "../../api/runtime-validators";

export type RegulatorySource = components["schemas"]["get_api_regulatory_sources_item"];
export type RegulatorySourcesPage = components["schemas"]["get_api_regulatory_sources_response"];
export type RegisterRegulatorySource = components["schemas"]["Request_post_api_regulatory_sources"];
export type SourceKind = RegisterRegulatorySource["source_kind"];
export type RegulatorySourceStatus = RegulatorySource["status"];
export const sourceKinds: readonly SourceKind[] = ["law", "government_decision", "ministerial_order", "authorization", "accreditation", "founder_decision", "contract", "other"];
export interface RegulatorySourcesQuery { page: number; pageSize: number; status?: RegulatorySourceStatus; source_kind?: SourceKind; citation?: string; sort: "citation" | "source_kind" | "status" | "created_at" | "updated_at"; direction: "asc" | "desc"; }
export interface RegulatorySourcesApi {
  list(query: RegulatorySourcesQuery): Promise<RegulatorySourcesPage>;
  register(input: RegisterRegulatorySource): Promise<RegulatorySource>;
  verify(sourceID: string, expectedVersion: number): Promise<RegulatorySource>;
  activate(sourceID: string, input: components["schemas"]["Request_post_api_regulatory_sources_sourceid_activate"]): Promise<RegulatorySource>;
  evidence(sourceID: string, evidenceID: string): Promise<Blob>;
}

type Result<T> = { data?: T; error?: unknown; response: Response };
async function checked<T>(request: Promise<Result<T>>, validate: (value: unknown) => boolean, operation: string): Promise<T> {
  const result = await request;
  if (!result.response.ok || result.error || result.data === undefined) throw new Error(`regulatory_sources_${operation}_${result.response.status}`);
  if (!validate(result.data)) throw new Error("regulatory_sources_contract_mismatch");
  return result.data as T;
}
export function createRegulatorySourcesApi(client: ContractClient): RegulatorySourcesApi {
  const pendingKeys = new Map<string, string>();
  const newKey = () => crypto.randomUUID();
  const command = async <T,>(operation: string, body: unknown, request: (header: { "Idempotency-Key": string }) => Promise<T>): Promise<T> => {
    const fingerprint = `${operation}:${JSON.stringify(body)}`;
    const key = pendingKeys.get(fingerprint) ?? newKey(); pendingKeys.set(fingerprint, key);
    try { const output = await request({ "Idempotency-Key": key }); pendingKeys.delete(fingerprint); return output; } catch (error) { throw error; }
  };
  return {
    list: async query => checked(client.GET("/api/regulatory-sources", { params: { query } }), validateGetApiRegulatorySourcesResponse, "list"),
    register: body => command("register", body, async header => checked(client.POST("/api/regulatory-sources", { params: { header }, body }), validatePostApiRegulatorySourcesResponse, "register")),
    verify: (sourceID, expected_version) => command(`verify:${sourceID}`, { expected_version }, async header => checked(client.POST("/api/regulatory-sources/{sourceID}/verify", { params: { path: { sourceID }, header }, body: { expected_version } }), validatePostApiRegulatorySourcesSourceidVerifyResponse, "verify")),
    activate: (sourceID, body) => command(`activate:${sourceID}`, body, async header => checked(client.POST("/api/regulatory-sources/{sourceID}/activate", { params: { path: { sourceID }, header }, body }), validatePostApiRegulatorySourcesSourceidActivateResponse, "activate")),
    evidence: async (sourceID, evidenceID) => {
      const result = await client.GET("/api/regulatory-sources/{sourceID}/evidence/{evidenceID}", { params: { path: { sourceID, evidenceID } }, parseAs: "blob" });
      if (!result.response.ok || result.error || !(result.data instanceof Blob)) throw new Error(`regulatory_sources_evidence_${result.response.status}`);
      return result.data;
    },
  };
}
