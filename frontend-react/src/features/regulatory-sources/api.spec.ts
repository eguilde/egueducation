import { describe, expect, it } from "vitest";
import { createContractClient } from "../../api/client";
import { createRegulatorySourcesApi, type RegulatorySource } from "./api";

const source: RegulatorySource = {
  id: "00000000-0000-4000-8000-000000000001", citation: "Legea educației", publisher_url: "https://publisher.example/lege",
  issuer: "Minister", source_kind: "law", applicable_from: "2026-01-01", applicable_until: null, status: "draft",
  expected_version: 2, created_by_subject: "creator", created_at: "2026-01-01T00:00:00Z", updated_at: "2026-01-01T00:00:00Z",
  latest_evidence_id: null, latest_evidence_sha256: null, latest_evidence_retrieved_at: null, activation_evidence_id: null,
  activated_by_subject: null, activated_at: null, assessment: null,
};
const response = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status, headers: { "content-type": "application/json" } });
const register = { citation: source.citation, publisher_url: source.publisher_url, issuer: source.issuer, source_kind: source.source_kind, applicable_from: "2026-01-01", applicable_until: null } as const;

describe("regulatory source contract adapter", () => {
  it("uses generated validation and sends the exact register DTO", async () => {
    const requests: Request[] = [];
    const api = createRegulatorySourcesApi(createContractClient(async request => { requests.push(request); return response(source, 201); }, "https://example.test/api"));
    await expect(api.register(register)).resolves.toEqual(source);
    expect(new URL(requests[0].url).pathname).toBe("/api/regulatory-sources");
    expect(await requests[0].clone().json()).toEqual(register);
  });

  it("reuses an uncertain key only in the same API instance and changes it when the payload changes", async () => {
    const requests: Request[] = []; let calls = 0;
    const fetcher = async (request: Request) => { requests.push(request); calls += 1; if (calls === 1) throw new TypeError("network lost"); return response(source, 201); };
    const client = createContractClient(fetcher, "https://example.test/api"); const first = createRegulatorySourcesApi(client); const second = createRegulatorySourcesApi(client);
    await expect(first.register(register)).rejects.toThrow("network lost"); await first.register(register); await first.register({ ...register, citation: "Legea actualizată" }); await second.register(register);
    expect(requests[0].headers.get("Idempotency-Key")).toBe(requests[1].headers.get("Idempotency-Key"));
    expect(requests[1].headers.get("Idempotency-Key")).not.toBe(requests[2].headers.get("Idempotency-Key"));
    expect(requests[1].headers.get("Idempotency-Key")).not.toBe(requests[3].headers.get("Idempotency-Key"));
  });

  it("uses dedicated command routes and rejects an invalid generated list response", async () => {
    const requests: Request[] = [];
    const api = createRegulatorySourcesApi(createContractClient(async request => { requests.push(request); if (request.method === "GET") return response({ items: "bad" }); return response(source); }, "https://example.test/api"));
    await expect(api.list({ page: 1, pageSize: 20, sort: "citation", direction: "asc", citation: "lege" })).rejects.toThrow("regulatory_sources_contract_mismatch");
    await api.verify(source.id, source.expected_version); await api.activate(source.id, { expected_version: source.expected_version, evidence_id: "00000000-0000-4000-8000-000000000002", assessment: "Aplicabilă instituției." });
    expect(new URL(requests[1].url).pathname).toBe(`/api/regulatory-sources/${source.id}/verify`);
    expect(new URL(requests[2].url).pathname).toBe(`/api/regulatory-sources/${source.id}/activate`);
    expect(await requests[1].clone().json()).toEqual({ expected_version: 2 });
  });
});
