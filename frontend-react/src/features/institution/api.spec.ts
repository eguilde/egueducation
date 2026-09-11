import { describe, expect, it } from "vitest";
import { createContractClient } from "../../api/client";
import { createInstitutionPolicyApi, type PutRegulatoryProfileInput } from "./api";

describe("institution policy contract adapter", () => {
  it("uses only the three generated institution operations and never adds scope to the body", async () => {
    const calls: Array<{ method: string; path: string; body: unknown }> = [];
    const responseProfile = { id: "profile-1", tenant_code: "tenant-a", institution_id: "inst-a", version: 1, status: "active", school_legal_form: "private", regulatory_profile: "ro.private.preuniversity", authorization_status: "accredited", accreditation_reference: "", authorized_levels: [], has_legal_personality: true, tax_identifier: "", founder_name: "", funder_name: "", budget_authority_name: "", is_contracting_authority: false, accounting_profile: "", procurement_profile: "", payroll_profile: "", vat_profile: "not_registered", treasury_required: false, public_funding: false, program_codes: [], effective_from: "2026-09-11", effective_to: null, source_reference: "approval-1", approved_by_subject: "admin", approved_at: "2026-09-11T00:00:00Z", created_by_subject: "admin", created_at: "2026-09-11T00:00:00Z", updated_by_subject: "admin", updated_at: "2026-09-11T00:00:00Z" };
    const client = createContractClient(async (request) => {
      calls.push({ method: request.method, path: new URL(request.url).pathname, body: request.method === "PUT" ? await request.clone().json() : undefined });
      const body = new URL(request.url).pathname.endsWith("/capabilities")
        ? { tenant_code: "tenant-a", institution_id: "inst-a", evaluated_at: "2026-09-11T00:00:00Z", evaluation_id: null, profile_id: "profile-1", profile_version: 1, profile_status: "active", school_legal_form: "private", blocked: false, block_reason: "", warnings: [], effective_policies: [], capabilities: [] }
        : responseProfile;
      return new Response(JSON.stringify(body), { status: 200, headers: { "content-type": "application/json" } });
    }, "https://example.test/api");
    const api = createInstitutionPolicyApi(client);
    await api.profile(); await api.capabilities();
    const input = {
      expected_version: 1, status: "approved", school_legal_form: "private", regulatory_profile: "ro.private.preuniversity",
      authorization_status: "accredited", authorized_levels: [], has_legal_personality: true, is_contracting_authority: false,
      treasury_required: false, public_funding: false, program_codes: [], effective_from: "2026-09-11", source_reference: "approval-1",
    } satisfies PutRegulatoryProfileInput;
    await api.saveProfile(input);
    expect(calls.map(({ method, path }) => `${method} ${path}`)).toEqual([
      "GET /api/institution/regulatory-profile", "GET /api/institution/capabilities", "PUT /api/institution/regulatory-profile",
    ]);
    expect(calls[2].body).toEqual(input);
    expect(calls[2].body).not.toHaveProperty("tenant_code");
    expect(calls[2].body).not.toHaveProperty("institution_id");
  });

  it("fails closed when a successful response violates the runtime contract", async () => {
    const client = createContractClient(async () => new Response(JSON.stringify({ tenant_code: "tenant-a", institution_id: "inst-a", capabilities: "forged" }), { status: 200, headers: { "content-type": "application/json" } }), "https://example.test/api");
    await expect(createInstitutionPolicyApi(client).capabilities()).rejects.toThrow("institution_policy_contract_mismatch");
  });
});
