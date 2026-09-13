import type { ContractClient } from "../../api/client";
import type { components } from "../../api/generated";
import { validateGetApiRegulatorySourcesResponse, validateGetApiEarchivaRetentionRulesResponse, validateGetApiEarchivaTaxonomyResponse, validatePostApiEarchivaRetentionRulesResponse, validatePostApiEarchivaRetentionRulesRuleidApproveResponse, validatePostApiEarchivaRetentionRulesRuleidRetireResponse } from "../../api/runtime-validators";

export type ArchiveSeriesRetentionRule = components["schemas"]["get_api_earchiva_retention_rules_item"];
export type RetentionPage = { items: ArchiveSeriesRetentionRule[]; total: number; page: number; pageSize: number };
export type RetentionSort = "effective_from" | "effective_to" | "status" | "minimum_retention_days" | "created_at" | "updated_at";
export type RetentionQuery = { page: number; pageSize: number; taxonomyNodeID?: string; status?: "proposed" | "active" | "retired" | "revoked"; sourceID?: string; anchorKind?: "intake_received_at"; sort: RetentionSort; direction: "asc" | "desc" };
export type RetentionSource = { id: string; label: string };
export interface ArchiveRetentionApi {
  list(query: RetentionQuery): Promise<RetentionPage>;
  sources(query: string): Promise<RetentionSource[]>;
  taxonomy(): Promise<Array<{ id: string; label: string }>>;
  propose(input: { taxonomy_node_id: string; source_id: string; effective_from: string; effective_to?: string; anchor_kind: "intake_received_at"; duration_model: "minimum_days"; minimum_retention_days: number }): Promise<ArchiveSeriesRetentionRule>;
  approve(ruleID: string, expectedVersion: number): Promise<ArchiveSeriesRetentionRule>;
  retire(ruleID: string, expectedVersion: number, reason: string): Promise<ArchiveSeriesRetentionRule>;
}
type Result = { data?: unknown; error?: unknown; response: Response };
const pendingIdempotencyKeys = new Map<string, string>();
const idempotencyKey = (operation: string, payload: unknown) => {
  const fingerprint = `${operation}:${JSON.stringify(payload)}`;
  let value = pendingIdempotencyKeys.get(fingerprint);
  if (!value) { value = globalThis.crypto.randomUUID(); pendingIdempotencyKeys.set(fingerprint, value); }
  return { fingerprint, header: { "Idempotency-Key": value } };
};
async function mutation<T>(operation: string, payload: unknown, request: (header: { "Idempotency-Key": string }) => Promise<T>): Promise<T> {
  const key = idempotencyKey(operation, payload);
  try { const result = await request(key.header); pendingIdempotencyKeys.delete(key.fingerprint); return result; }
  catch (reason) { throw reason; }
}
async function checked<T>(request: Promise<Result>, validate: (data: unknown) => boolean): Promise<T> { const result = await request; if (!result.response.ok || result.error || result.data === undefined) throw new Error(`archive_retention_${result.response.status}`); if (!validate(result.data)) throw new Error("archive_retention_contract_mismatch"); return result.data as T; }

export function createArchiveRetentionApi(client: ContractClient): ArchiveRetentionApi {
  return {
    list: query => checked(client.GET("/api/earchiva/retention-rules", { params: { query: { page: query.page, pageSize: query.pageSize, taxonomy_node_id: query.taxonomyNodeID, status: query.status, source_id: query.sourceID, anchor_kind: query.anchorKind, sort: query.sort, direction: query.direction } } }), validateGetApiEarchivaRetentionRulesResponse),
    sources: async citation => {
      const result: RetentionSource[] = [];
      let page = 1;
      let total = 0;
      do {
        const response = await checked<{ items: Array<{ id: string; citation: string; issuer: string; activation_evidence_id: string | null }>; total: number }>(client.GET("/api/regulatory-sources", { params: { query: { page, pageSize: 100, sort: "citation", direction: "asc", status: "active", citation } } }), validateGetApiRegulatorySourcesResponse);
        total = response.total;
        result.push(...response.items.filter(item => item.activation_evidence_id !== null).map(item => ({ id: item.id, label: `${item.citation} · ${item.issuer}` })));
        if (response.items.length === 0) break;
        page += 1;
      } while ((page - 1) * 100 < total);
      return result;
    },
    taxonomy: async () => { const nodes = await checked<Array<{ id: string; path?: string; label: string }>>(client.GET("/api/earchiva/taxonomy"), validateGetApiEarchivaTaxonomyResponse); return nodes.map(node => ({ id: node.id, label: node.path || node.label })); },
    propose: body => mutation("propose", body, header => checked(client.POST("/api/earchiva/retention-rules", { params: { header }, body }), validatePostApiEarchivaRetentionRulesResponse)),
    approve: (ruleID, expected_version) => mutation(`approve:${ruleID}`, { expected_version }, header => checked(client.POST("/api/earchiva/retention-rules/{ruleID}/approve", { params: { path: { ruleID }, header }, body: { expected_version } }), validatePostApiEarchivaRetentionRulesRuleidApproveResponse)),
    retire: (ruleID, expected_version, reason) => mutation(`retire:${ruleID}`, { expected_version, reason }, header => checked(client.POST("/api/earchiva/retention-rules/{ruleID}/retire", { params: { path: { ruleID }, header }, body: { expected_version, reason } }), validatePostApiEarchivaRetentionRulesRuleidRetireResponse)),
  };
}
