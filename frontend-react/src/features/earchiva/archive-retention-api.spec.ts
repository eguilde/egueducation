import { describe, expect, it, vi } from "vitest";
import { createContractClient } from "../../api/client";
import { createArchiveRetentionApi } from "./archive-retention-api";

const item = { id: "00000000-0000-4000-8000-000000000001", taxonomy_node_id: "00000000-0000-4000-8000-000000000002", source_id: "00000000-0000-4000-8000-000000000003", source_checksum_sha256: "a".repeat(64), anchor_kind: "intake_received_at", duration_model: "minimum_days", effective_from: "2026-01-01", effective_to: null, minimum_retention_days: 365, status: "proposed", expected_version: 1, proposed_by_subject: "author", proposed_at: "2026-01-01T00:00:00Z", approved_by_subject: "", approved_at: null, retired_by_subject: "", retired_at: null, retirement_reason: "", created_at: "2026-01-01T00:00:00Z", updated_at: "2026-01-01T00:00:00Z" };
const json = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status, headers: { "content-type": "application/json" } });

describe("archive retention contract adapter", () => {
  it("selects only evidence-activated sources through the dedicated source permission boundary", async () => {
    const source = {
      id: item.source_id, citation: "Nomenclator aprobat", publisher_url: "https://publisher.example/source.pdf",
      issuer: "Instituție", source_kind: "other", applicable_from: "2026-01-01", applicable_until: null,
      status: "active", expected_version: 3, created_by_subject: "author", created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z", latest_evidence_id: item.id, latest_evidence_sha256: "a".repeat(64),
      latest_evidence_retrieved_at: "2026-01-01T00:00:00Z", activation_evidence_id: item.id,
      activated_by_subject: "approver", activated_at: "2026-01-01T00:00:00Z", assessment: "Evaluare documentată",
    };
    const requests: Request[] = [];
    const api = createArchiveRetentionApi(createContractClient(async request => {
      requests.push(request);
      return json({ items: [source, { ...source, id: item.taxonomy_node_id, activation_evidence_id: null }], total: 2, page: 1, pageSize: 100 });
    }, "https://example.test/api"));
    expect(await api.sources("Nomenclator")).toEqual([{ id: item.source_id, label: "Nomenclator aprobat · Instituție" }]);
    const url = new URL(requests[0].url);
    expect(url.pathname).toBe("/api/regulatory-sources");
    expect(url.searchParams.get("status")).toBe("active");
    expect(url.searchParams.get("citation")).toBe("Nomenclator");
  });
  it("reuses an uncertain proposal key, changes it for changed DTOs, and sends the exact body", async () => {
    const requests: Request[] = []; let attempts = 0;
    const client = createContractClient(async request => { requests.push(request); attempts += 1; if (attempts === 1) throw new TypeError("network lost"); return json(item, 201); }, "https://example.test/api");
    const api = createArchiveRetentionApi(client);
    const input = { taxonomy_node_id: item.taxonomy_node_id, source_id: item.source_id, effective_from: "2026-01-01", anchor_kind: "intake_received_at" as const, duration_model: "minimum_days" as const, minimum_retention_days: 365 };
    await expect(api.propose(input)).rejects.toThrow("network lost");
    await api.propose(input);
    await api.propose({ ...input, minimum_retention_days: 366 });
    expect(requests[0].headers.get("Idempotency-Key")).toBe(requests[1].headers.get("Idempotency-Key"));
    expect(requests[1].headers.get("Idempotency-Key")).not.toBe(requests[2].headers.get("Idempotency-Key"));
    expect(await requests[1].clone().json()).toEqual(input);
  });

  it("uses all server query names and fails closed on an invalid list response", async () => {
    const fetcher = vi.fn().mockResolvedValue(json({ items: "invalid" }));
    const api = createArchiveRetentionApi(createContractClient(fetcher, "https://example.test/api"));
    await expect(api.list({ page: 2, pageSize: 10, taxonomyNodeID: item.taxonomy_node_id, status: "active", sourceID: item.source_id, anchorKind: "intake_received_at", sort: "updated_at", direction: "asc" })).rejects.toThrow("archive_retention_contract_mismatch");
    const url = new URL((fetcher.mock.calls[0][0] as Request).url);
    expect(url.searchParams.get("page")).toBe("2"); expect(url.searchParams.get("pageSize")).toBe("10"); expect(url.searchParams.get("taxonomy_node_id")).toBe(item.taxonomy_node_id); expect(url.searchParams.get("source_id")).toBe(item.source_id); expect(url.searchParams.get("anchor_kind")).toBe("intake_received_at"); expect(url.searchParams.get("sort")).toBe("updated_at"); expect(url.searchParams.get("direction")).toBe("asc");
  });
});
