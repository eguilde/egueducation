import { describe, expect, it, vi } from "vitest";
import type { ContractClient } from "../../api/client";
import { createAdmissionApi } from "./api";

const ok = (data: unknown, status = 200) => Promise.resolve({
  data,
  response: new Response(JSON.stringify(data), { status }),
});

describe("createAdmissionApi", () => {
  it("uses generated admission paths and server-side query parameters", async () => {
    const campaign = {
      id: "campaign-1", source_id: "source-1", code: "2026", title: "Admitere", school_year: "2026-2027",
      offering_id: "offering-1", location_id: "location-1", authorization_id: "authorization-1",
      class_offering_context_id: "context-1", capacity_limit: 25, capacity_unit: "students",
      student_place_limit: 25, capacity_basis: {}, shift: "day", opens_on: "2026-03-01",
      closes_on: "2026-03-31", decision_due_on: null, status: "draft", expected_version: 1,
    };
    const GET = vi.fn().mockImplementation(() => ok({ items: [campaign], total: 1, page: 1, pageSize: 20 }));
    const api = createAdmissionApi({ GET } as unknown as ContractClient);

    const result = await api.listCampaigns({ page: 1, pageSize: 20, sort: "opens_on", direction: "desc", filters: { title: "Admitere" } });

    expect(result.items[0]).toMatchObject({ id: "campaign-1", capacity_basis: {} });
    expect(GET).toHaveBeenCalledWith("/api/admissions/campaigns", expect.objectContaining({ params: { query: expect.objectContaining({ page: 1, pageSize: 20, sort: "opens_on", direction: "desc", "filter.title": "Admitere" }) } }));
  });

  it("sends a caller idempotency key for every admission command", async () => {
    const POST = vi.fn().mockImplementation(() => ok({ id: "application-1", status: "draft", expected_version: 1 }, 201));
    const api = createAdmissionApi({ POST } as unknown as ContractClient);

    await api.createApplication({ campaign_id: "campaign-1", application_no: "A-1", candidate_party_id: "party-1", consent_snapshot: { confirmed: true } });

    expect(POST).toHaveBeenCalledWith("/api/admissions/applications", expect.objectContaining({
      params: { header: { "Idempotency-Key": expect.any(String) } },
      body: expect.objectContaining({ consent_snapshot: { confirmed: true } }),
    }));
  });

  it("uses generated retention-authority contracts for propose, second-actor approval and activation", async () => {
    const policy = { id: "policy-1", status: "active", rule_version_id: "rule-1", minimum_retention_days: 3650, effective_from: "2026-09-12", source_id: "source-1" };
    const rule = { id: "rule-1", artifact_kind: "admission_dss", status: "proposed", minimum_retention_days: 3650, effective_from: "2026-09-12", effective_to: null, source_id: "source-1", source_checksum_sha256: "a".repeat(64), proposed_by_subject: "director-a" };
    const GET = vi.fn().mockImplementation(() => ok(policy));
    const POST = vi.fn().mockImplementation((path: string) => {
      if (path === "/api/admissions/retention-rule-versions") return ok(rule, 201);
      if (path === "/api/admissions/retention-rule-versions/approve") return ok({ ...rule, status: "active", approved_by_subject: "director-b" });
      return ok({ ...policy, replayed: false }, 201);
    });
    const api = createAdmissionApi({ GET, POST } as unknown as ContractClient);

    await expect(api.currentDSSRetentionPolicy()).resolves.toMatchObject(policy);
    await expect(api.proposeRetentionRule({ artifact_kind: "admission_dss", minimum_retention_days: 3650, effective_from: "2026-09-12", effective_to: null, source_id: "source-1" })).resolves.toMatchObject(rule);
    await expect(api.approveRetentionRule({ rule_version_id: "rule-1" })).resolves.toMatchObject({ ...rule, status: "active" });
    await expect(api.configureDSSRetentionPolicy({ rule_version_id: "rule-1", effective_from: "2026-09-12" })).resolves.toMatchObject(policy);
    expect(GET).toHaveBeenCalledWith("/api/admissions/dss-retention-policies/current");
    expect(POST).toHaveBeenCalledWith("/api/admissions/retention-rule-versions", expect.objectContaining({
      params: { header: { "Idempotency-Key": expect.any(String) } },
      body: { artifact_kind: "admission_dss", minimum_retention_days: 3650, effective_from: "2026-09-12", effective_to: null, source_id: "source-1" },
    }));
    expect(POST).toHaveBeenCalledWith("/api/admissions/retention-rule-versions/approve", expect.objectContaining({
      body: { rule_version_id: "rule-1" },
    }));
    expect(POST).toHaveBeenCalledWith("/api/admissions/dss-retention-policies", expect.objectContaining({
      body: { rule_version_id: "rule-1", effective_from: "2026-09-12" },
    }));
  });

  it("keeps legal prepare, signed-WORM finalize and signer authorization on the contract client", async () => {
    const preparation = { retention_policy_id: "11111111-1111-4111-8111-111111111111", retention_rule_version_id: "22222222-2222-4222-8222-222222222222", retention_source_id: "33333333-3333-4333-8333-333333333333", minimum_retention_days: 30, retention_anchor_at: "2026-09-12T10:15:00Z", required_retention_until: "2026-10-12T10:15:00Z", id: "prep-1", artifact_kind: "decision", artifact_id: "application-1", application_id: "application-1", policy_evaluation_v2_id: "policy-1", canonical_payload: { application_id: "application-1" }, canonical_payload_base64: "eyJhcHBsaWNhdGlvbl9pZCI6ImFwcGxpY2F0aW9uLTEifQ==", canonical_payload_sha256: "a".repeat(64), prepared_by_subject: "director-a", prepared_at: "2026-09-12T10:00:00Z", expires_at: "2026-09-12T10:15:00Z", status: "prepared" };
    const authorization = { id: "auth-1", certificate_sha256: "b".repeat(64), user_id: "user-1", actor_subject: "director-a", permission_code: "education.admissions.decide", valid_until: "2027-09-12", proposed_by_subject: "director-b", status: "proposed", expected_version: 2 };
    const GET = vi.fn().mockImplementation(() => ok({ items: [authorization], total: 1, page: 1, pageSize: 20 }));
    const POST = vi.fn().mockImplementation((path: string) => {
      if (path.endsWith("decision-preparations")) return ok(preparation, 201);
      if (path.endsWith("legal-preparations/finalize")) return ok({ id: "decision-1", status: "admitted" }, 201);
      if (path.endsWith("/revoke")) return ok({ id: "auth-1", status: "revoked" }, 201);
      if (path.endsWith("signer-authorizations/approve")) return ok({ ...authorization, status: "active" });
      return ok(authorization, 201);
    });
    const api = createAdmissionApi({ GET, POST } as unknown as ContractClient);

    await api.prepareDecision("application-1", { decision_no: "D-1", outcome: "admitted", rationale: "criterii îndeplinite", expected_version: 3 });
    await api.finalizeDecision("application-1", { preparation_id: "prep-1", archive: { document_id: "doc-1", version_id: "ver-1" } });
    await api.listSignerAuthorizations({ page: 1, pageSize: 20, sort: "valid_until", direction: "asc", filters: { status: "active" } });
    await api.proposeSignerAuthorization({ certificate_sha256: "b".repeat(64), user_id: "user-1", permission_code: "education.admissions.decide", valid_until: "2027-09-12" });
    await api.approveSignerAuthorization({ proposal_id: "auth-1" });
    await api.revokeSignerAuthorization("auth-1", { expected_version: 2, reason: "certificat compromis" });

    expect(POST).toHaveBeenCalledWith("/api/admissions/applications/{applicationID}/decision-preparations", expect.objectContaining({ params: expect.objectContaining({ path: { applicationID: "application-1" } }), body: expect.objectContaining({ decision_no: "D-1" }) }));
    expect(POST).toHaveBeenCalledWith("/api/admissions/legal-preparations/finalize", expect.objectContaining({ body: { preparation_id: "prep-1", archive: { document_id: "doc-1", version_id: "ver-1" } } }));
    expect(GET).toHaveBeenCalledWith("/api/admissions/signer-authorizations", expect.objectContaining({ params: { query: expect.objectContaining({ "filter.status": "active" }) } }));
    expect(POST).toHaveBeenCalledWith("/api/admissions/signer-authorizations/approve", expect.objectContaining({ body: { proposal_id: "auth-1" } }));
    expect(POST).toHaveBeenCalledWith("/api/admissions/signer-authorizations/{authorizationID}/revoke", expect.objectContaining({ params: expect.objectContaining({ path: { authorizationID: "auth-1" } }), body: { expected_version: 2, reason: "certificat compromis" } }));
    const { required_retention_until: _deadline, ...missingDeadline } = preparation;
    for (const invalid of [missingDeadline, { ...preparation, required_retention_until: "not-a-date" }, { ...preparation, minimum_retention_days: 0 }]) {
      POST.mockImplementationOnce(() => ok(invalid, 201));
      await expect(api.prepareDecision("application-1", { decision_no: "D-1", outcome: "admitted", rationale: "criterii îndeplinite", expected_version: 3 })).rejects.toThrow("admission_contract_mismatch");
    }
  });

  it("uses the preparation-bound multipart contract, preserves caller retry keys, and treats an absent slot as recovery state", async () => {
    const artifact = { intent_id: "11111111-1111-4111-8111-111111111111", preparation_id: "22222222-2222-4222-8222-222222222222", artifact_slot: "primary", retention_until: "2027-09-12T10:00:00Z", replayed: false, document: { id: "doc-1", institution_id: "institution-1", title: "Decizie", original_file_name: "signed.pdf", mime_type: "application/pdf", source_kind: "admission_legal_preparation", source_system: "admission", external_reference: "prep-1", status: "ready", current_version_no: 1, received_at: "2026-09-12T10:00:00Z", created_at: "2026-09-12T10:00:00Z", updated_at: "2026-09-12T10:00:00Z" }, version: { id: "version-1", document_id: "doc-1", version_no: 1, source_sha256: "a".repeat(64), source_size_bytes: 12, page_count: 1, text_status: "queued", created_at: "2026-09-12T10:00:00Z" } };
    const GET = vi.fn().mockResolvedValue({ data: undefined, response: new Response(null, { status: 404 }) });
    const POST = vi.fn().mockResolvedValue(ok(artifact, 201));
    const api = createAdmissionApi({ GET, POST } as unknown as ContractClient);
    const file = new File(["%PDF-1.7"], "signed.pdf", { type: "application/pdf" });

    await expect(api.getLegalPreparationArtifact("prep-1", "primary")).resolves.toBeUndefined();
    await expect(api.uploadLegalPreparationArtifact("prep-1", "primary", file, "retry-key-1")).resolves.toMatchObject({ document: { id: "doc-1" } });
    expect(GET).toHaveBeenCalledWith("/api/admissions/legal-preparations/{preparationID}/artifacts/{artifactSlot}", expect.objectContaining({ params: { path: { preparationID: "prep-1", artifactSlot: "primary" } } }));
    const call = POST.mock.calls[0][1];
    expect(call.params).toMatchObject({ path: { preparationID: "prep-1", artifactSlot: "primary" }, header: { "Idempotency-Key": "retry-key-1" } });
    expect(call.body).toBeInstanceOf(FormData);
    expect((call.body as FormData).get("file")).toMatchObject({ name: "signed.pdf", type: "application/pdf" });
  });

  it("rejects a response that violates the generated runtime contract", async () => {
    const GET = vi.fn().mockImplementation(() => ok({ items: [{ id: "incomplete" }], total: 1, page: 1, pageSize: 20 }));
    const api = createAdmissionApi({ GET } as unknown as ContractClient);

    await expect(api.listApplications({ page: 1, pageSize: 20, sort: "submitted_at", direction: "desc", filters: {} })).rejects.toThrow("admission_contract_mismatch");
  });
});
