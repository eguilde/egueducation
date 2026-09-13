import { describe, expect, it } from "vitest";
import { createContractClient } from "../../api/client";
import { createSchoolOperationsApi } from "./api";

describe("school operations OpenAPI adapter", () => {
  it("uses only generated contracts endpoints for list, lifecycle and obligations", async () => {
    const calls: string[] = [];
    const client = createContractClient(async request => {
      const path = new URL(request.url).pathname;
      calls.push(`${request.method} ${path}`);
      const body = path.endsWith("/suppliers") ? { items: [{ id: "supplier-1", display_name: "Furnizor Energie", code: "FE", tax_id: "RO1" }], total: 1, page: 1, pageSize: 20 }
        : path.endsWith("/obligations") && request.method === "GET" ? { items: [], total: 0, page: 1, pageSize: 20 }
        : path.endsWith("/contracts") && request.method === "GET" ? { items: [], total: 0, page: 1, pageSize: 20 }
          : path.endsWith("/transition") ? { id: "contract-1", expected_version: 2, lifecycle_status: "active" }
            : path.endsWith("/obligations") ? { id: "obligation-1", expected_version: 1 }
              : path.endsWith("contract-1") && request.method === "GET" ? { id: "contract-1", expected_version: 1, archive_status: "pending", category: "utilities", contract_number: "C-1", currency: "RON", lifecycle_status: "draft", title: "Electricitate", total_value: 100 }
                : { id: "contract-1", expected_version: 1, archive_status: "pending" };
      return new Response(JSON.stringify(body), { status: 200, headers: { "content-type": "application/json" } });
    }, "https://example.test/api");
    const api = createSchoolOperationsApi(client);
    await api.list({ page: 1, pageSize: 20, filters: { title: "electric" } });
    await api.suppliers({ page: 1, pageSize: 20, q: "elect" });
    await api.create({ contract_number: "C-1", title: "Electricitate", category: "utilities", supplier_party_id: "supplier-1", starts_on: "2026-01-01", ends_on: "2026-12-31", total_value: 100, currency: "RON", idempotency_key: "key-1" });
    await api.detail("contract-1");
    await api.amend("contract-1", { title: "Electricitate", category: "utilities", starts_on: "2026-01-01", ends_on: "2026-12-31", total_value: 100, currency: "RON", expected_version: 1 });
    await api.obligations("contract-1", { page: 1, pageSize: 20 });
    await api.addObligation("contract-1", { title: "Raport lunar", due_on: "2026-02-01", sla_hours: 24, guarantee_value: null });
    await api.transition("contract-1", { status: "active", expected_version: 1 });
    expect(calls).toEqual([
      "GET /api/school-operations/contracts",
      "GET /api/school-operations/suppliers",
      "POST /api/school-operations/contracts",
      "GET /api/school-operations/contracts/contract-1",
      "PATCH /api/school-operations/contracts/contract-1",
      "GET /api/school-operations/contracts/contract-1/obligations",
      "POST /api/school-operations/contracts/contract-1/obligations",
      "POST /api/school-operations/contracts/contract-1/transition",
    ]);
  });
});
