import { describe, expect, it } from "vitest";
import { createContractClient } from "../../api/client";
import { createInstitutionPolicyApi, type CreateEducationOfferingInput, type CreateOfferingAuthorizationInput, type CreateSchoolLocationInput, type PutRegulatoryProfileInput, type UpdateEducationOfferingInput, type UpdateSchoolLocationInput } from "./api";

describe("institution policy contract adapter", () => {
  it("uses only the generated institution operations and never adds scope to the body", async () => {
    const calls: Array<{ method: string; path: string; body: unknown }> = [];
    const responseProfile = { id: "profile-1", tenant_code: "tenant-a", institution_id: "inst-a", version: 1, status: "active", school_legal_form: "private", regulatory_profile: "ro.private.preuniversity", authorization_status: "accredited", accreditation_reference: "", authorized_levels: [], has_legal_personality: true, tax_identifier: "", founder_name: "", funder_name: "", budget_authority_name: "", is_contracting_authority: false, accounting_profile: "", procurement_profile: "", payroll_profile: "", vat_profile: "not_registered", treasury_required: false, public_funding: false, program_codes: [], effective_from: "2026-09-11", effective_to: null, source_reference: "approval-1", approved_by_subject: "admin", approved_at: "2026-09-11T00:00:00Z", created_by_subject: "admin", created_at: "2026-09-11T00:00:00Z", updated_by_subject: "admin", updated_at: "2026-09-11T00:00:00Z" };
    const client = createContractClient(async (request) => {
      calls.push({ method: request.method, path: new URL(request.url).pathname, body: request.method === "PUT" ? await request.clone().json() : undefined });
      const path = new URL(request.url).pathname;
      const body = path.endsWith("/capabilities")
        ? { tenant_code: "tenant-a", institution_id: "inst-a", evaluated_at: "2026-09-11T00:00:00Z", evaluation_id: null, profile_id: "profile-1", profile_version: 1, profile_status: "active", school_legal_form: "private", blocked: false, block_reason: "", warnings: [], effective_policies: [], capabilities: [] }
        : path.endsWith("/policy-cutover-preflight")
          ? { tenant_code: "tenant-a", institution_id: "inst-a", phase: "legacy", legacy_profiles: 1, unmapped_profiles: 0, legacy_assignments: 0, unmapped_assignments: 0, legacy_overrides: 0, unmapped_overrides: 0, legacy_evaluations: 0, unmapped_evaluations: 0, missing_pack_provenance: 0, inputs_without_effective_date: 0, inputs_with_multiple_decisions: 0, consumer_provenance_mismatches: 0, open_blocking_issues: 0, structurally_ready_for_dual: true }
          : responseProfile;
      return new Response(JSON.stringify(body), { status: 200, headers: { "content-type": "application/json" } });
    }, "https://example.test/api");
    const api = createInstitutionPolicyApi(client);
    await api.profile(); await api.capabilities(); await api.cutoverPreflight?.();
    const input = {
      expected_version: 1, status: "approved", school_legal_form: "private", regulatory_profile: "ro.private.preuniversity",
      authorization_status: "accredited", authorized_levels: [], has_legal_personality: true,
      program_codes: [], effective_from: "2026-09-11", source: { source_kind: "authorization", citation: "approval-1", article_reference: "", issuer: "ARACIP", source_url: "https://example.test/approval-1", checksum_sha256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" },
    } satisfies PutRegulatoryProfileInput;
    await api.saveProfile(input);
    expect(calls.map(({ method, path }) => `${method} ${path}`)).toEqual([
      "GET /api/institution/regulatory-profile", "GET /api/institution/capabilities", "GET /api/institution/policy-cutover-preflight", "PUT /api/institution/regulatory-profile",
    ]);
    expect(calls[3].body).toEqual(input);
    expect(calls[3].body).not.toHaveProperty("tenant_code");
    expect(calls[3].body).not.toHaveProperty("institution_id");
    expect(calls[3].body).not.toHaveProperty("public_funding");
    expect(calls[3].body).not.toHaveProperty("is_contracting_authority");
    expect(calls[3].body).not.toHaveProperty("treasury_required");
  });

  it("fails closed when a successful response violates the runtime contract", async () => {
    const client = createContractClient(async () => new Response(JSON.stringify({ tenant_code: "tenant-a", institution_id: "inst-a", capabilities: "forged" }), { status: 200, headers: { "content-type": "application/json" } }), "https://example.test/api");
    await expect(createInstitutionPolicyApi(client).capabilities()).rejects.toThrow("institution_policy_contract_mismatch");
  });

  it("uses generated scope-free contracts for locations, offerings and authorization history", async () => {
    const calls: Array<{ method: string; path: string; body?: Record<string, unknown> }> = [];
    const location = { id: "00000000-0000-4000-8000-000000000001", code: "sediu", name: "Sediu", address: "Balotești", active: true, effective_from: "2026-09-11", effective_to: null, expected_version: 1 };
    const offering = { id: "00000000-0000-4000-8000-000000000002", code: "gimnazial", education_level: "gimnazial", specialization_code: "", language_code: "ro", title: "Gimnaziu", active: true, effective_from: "2026-09-11", effective_to: null, expected_version: 1 };
    const authorization = { id: "00000000-0000-4000-8000-000000000003", offering_id: offering.id, offering_code: offering.code, offering_title: offering.title, location_id: location.id, location_code: location.code, location_name: location.name, status: "accredited", authority_name: "ARACIP", decision_reference: "H-1", capacity: 120, capacity_unit: "students", shift: "day", effective_from: "2026-09-11", effective_to: null, expected_version: 1, source_citation: "Hotărârea H-1", source_url: "https://example.test/h-1", replaces_authorization_id: null };
    const client = createContractClient(async (request) => {
      const path = new URL(request.url).pathname;
      const method = request.method;
      const body = method === "POST" || method === "PATCH" ? await request.clone().json() as Record<string, unknown> : undefined;
      calls.push({ method, path, body });
      const item = path.includes("/locations") ? location : path.includes("/education-offerings") ? offering : authorization;
      return new Response(JSON.stringify(method === "GET" ? { items: [item], total: 1, page: 1, pageSize: 10 } : item), { status: method === "POST" ? 201 : 200, headers: { "content-type": "application/json" } });
    }, "https://example.test/api");
    const api = createInstitutionPolicyApi(client);
    const locationInput = { code: "sediu", name: "Sediu", address: "Balotești", active: true, effective_from: "2026-09-11", effective_to: null, idempotency_key: "location-key" } satisfies CreateSchoolLocationInput;
    const offeringInput = { code: "gimnazial", education_level: "gimnazial", specialization_code: "", language_code: "ro", title: "Gimnaziu", active: true, effective_from: "2026-09-11", effective_to: null, idempotency_key: "offering-key" } satisfies CreateEducationOfferingInput;
    const authorizationInput = { offering_id: offering.id, location_id: location.id, status: "accredited", authority_name: "ARACIP", decision_reference: "H-1", capacity: 120, capacity_unit: "students", shift: "day", effective_from: "2026-09-11", effective_to: null, source: { source_kind: "accreditation", citation: "Hotărârea H-1", article_reference: "", issuer: "ARACIP", source_url: "https://example.test/h-1", published_on: null, consolidated_on: null, checksum_sha256: "a".repeat(64) }, replaces_authorization_id: null, expected_version: null, idempotency_key: "authorization-key" } satisfies CreateOfferingAuthorizationInput;
    const locationUpdate = { expected_version: 1, name: "Sediu actualizat", address: "Balotești", active: true, effective_to: null } satisfies UpdateSchoolLocationInput;
    const offeringUpdate = { expected_version: 1, title: "Gimnaziu actualizat", active: true, effective_to: null } satisfies UpdateEducationOfferingInput;
    await api.locations?.({ page: 1, pageSize: 10, filters: { code: "sediu" } });
    await api.createLocation?.(locationInput);
    await api.updateLocation?.(location.id, locationUpdate);
    await api.offerings?.({ page: 1, pageSize: 10, filters: { education_level: "gimnazial" } });
    await api.createOffering?.(offeringInput);
    await api.updateOffering?.(offering.id, offeringUpdate);
    await api.authorizations?.({ page: 1, pageSize: 10, filters: { status: "accredited" } });
    await api.createAuthorization?.(authorizationInput);
    expect(calls.map(({ method, path }) => `${method} ${path}`)).toEqual([
      "GET /api/institution/locations", "POST /api/institution/locations", "PATCH /api/institution/locations/00000000-0000-4000-8000-000000000001",
      "GET /api/institution/education-offerings", "POST /api/institution/education-offerings", "PATCH /api/institution/education-offerings/00000000-0000-4000-8000-000000000002",
      "GET /api/institution/offering-authorizations", "POST /api/institution/offering-authorizations",
    ]);
    for (const call of calls.filter((entry) => entry.body)) {
      expect(call.body).not.toHaveProperty("tenant_code");
      expect(call.body).not.toHaveProperty("institution_id");
      expect(call.body).not.toHaveProperty("actor_subject");
    }
  });
});
