import { describe, expect, it, vi } from "vitest";
import type { ContractClient } from "../../../api/client";
import { createSchoolWizardApi } from "../school-wizard-api";
import { buildWizardPayload, wizardDefinitions } from "./index";

const response = (data: object) => ({ data, response: new Response(null, { status: 201 }) });

const contractClient = () => {
  const get = vi.fn().mockResolvedValue({
    data: { items: [{ id: "user-1", name: "Director" }] },
    response: new Response(null, { status: 200 }),
  });
  const post = vi.fn().mockResolvedValue(response({ id: "record-1" }));
  return {
    get,
    post,
    contract: { GET: get, POST: post } as Pick<ContractClient, "GET" | "POST">,
  };
};

describe("education wizard contracts", () => {
  it.each([
    ["caMeeting", "caMeeting", "school_year"],
    ["personnel", "personnel", "full_name"],
    ["evaluation", "evaluation", "employee_code"],
    ["declaration", "declaration", "declaration_type"],
    ["mobility", "mobility", "request_type"],
    ["merit", "merit", "committee_name"],
    ["portfolio", "portfolio", "owner_name"],
  ] as const)("maps %s to a typed wizard operation and fields", (key, kind, field) => {
    const definition = wizardDefinitions[key];
    expect(definition.kind).toBe(kind);
    expect(definition.fields.some(item => item.key === field)).toBe(true);
    expect(definition.steps).toHaveLength(4);
  });
  it("trims strings and preserves typed values", () => expect(buildWizardPayload({ name: " Ana ", score: 8, funded: false })).toEqual({ name: "Ana", score: 8, funded: false }));
  it("declares manage versus self-manage RBAC", () => {
    expect(wizardDefinitions.caMeeting.permission).toBe("manage");
    expect(wizardDefinitions.portfolio.permission).toBe("manage");
    expect(wizardDefinitions.personnel.validate({ full_name: "", role_title: "", school_year: "" })).toHaveLength(3);
  });
  it("uses one immutable owner selector and never exposes raw pairing or server-controlled retention", () => {
    expect(wizardDefinitions.portfolio.initial).toHaveProperty("owner_user_id");
    expect(wizardDefinitions.portfolio.initial).toHaveProperty("owner_personnel_id");
    expect(wizardDefinitions.portfolio.initial).not.toHaveProperty("retention_until");
    expect(wizardDefinitions.portfolio.fields.some((field) => field.key === "owner_selection" && field.required)).toBe(true);
    expect(wizardDefinitions.portfolio.fields.some((field) => field.key === "owner_user_id" || field.key === "owner_personnel_id")).toBe(false);
    expect(wizardDefinitions.portfolio.fields.some((field) => field.key === "retention_until")).toBe(false);
  });
  it("uses the exact governance enums accepted by the backend", () => {
    const field = (definition: "minute" | "vote" | "resolution", key: string) =>
      wizardDefinitions[definition].fields.find((item) => item.key === key);

    expect(wizardDefinitions.minute.initial.follow_up_status).toBe("de_stabilit");
    expect(field("minute", "follow_up_status")?.options?.map((item) => item.value)).toEqual([
      "de_stabilit", "in_urmarire", "realizat", "amanat", "inchis",
    ]);
    expect(wizardDefinitions.vote.initial).toMatchObject({
      decision_type: "hotarare",
      outcome: "adoptat",
    });
    expect(field("vote", "decision_type")?.options?.map((item) => item.value)).toEqual([
      "hotarare", "aviz", "informare", "delegare", "aprobare",
    ]);
    expect(field("vote", "outcome")?.options?.map((item) => item.value)).toEqual([
      "adoptat", "respins", "amanat",
    ]);
    expect(wizardDefinitions.resolution.initial).toMatchObject({
      publication_status: "intern",
      anonymization_state: "nu_este_necesara",
    });
    expect(field("resolution", "publication_status")?.options?.map((item) => item.value)).toEqual([
      "intern", "publicat", "pregatit_publicare",
    ]);
    expect(field("resolution", "anonymization_state")?.options?.map((item) => item.value)).toEqual([
      "necesara", "finalizata", "nu_este_necesara",
    ]);
  });

  it("uses the exact personnel declaration taxonomy accepted by the backend", () => {
    const declarationType = wizardDefinitions.declaration.fields.find((item) => item.key === "declaration_type");
    expect(wizardDefinitions.declaration.initial.declaration_type).toBe("authenticity");
    expect(declarationType?.options?.map((item) => item.value)).toEqual([
      "interests", "assets", "gdpr", "authenticity",
    ]);
  });

  it("uses a literal generated personnel operation and DTO", async () => {
    const mock = contractClient();
    await createSchoolWizardApi(mock.contract).create("personnel", {
      full_name: " Ana Pop ", role_title: "Profesor", employment_type: "titular",
      status: "active", evaluation_status: "not_started", mobility_stage: "none",
      school_year: "2026-2027", assigned_unit: "Gimnaziu", phone: "", email: "",
      has_portfolio: false, notes: "",
    });
    expect(mock.post).toHaveBeenCalledWith("/api/education/personnel/records", {
      body: expect.objectContaining({ full_name: "Ana Pop", school_year: "2026-2027" }),
    });
  });

  it("uses the meeting ID solely as a literal generated path parameter", async () => {
    const mock = contractClient();
    await createSchoolWizardApi(mock.contract).create("minute", {
      meeting_id: "meeting-1", agenda_order: 1, topic_title: "Buget",
      discussion_summary: "Discuție", decision_summary: "Decizie", responsible_party: "",
      due_on: "", follow_up_status: "de_stabilit", requires_publication: false, notes: "",
    });
    expect(mock.post).toHaveBeenCalledWith("/api/education/governance/meetings/{meetingID}/minutes", {
      params: { path: { meetingID: "meeting-1" } },
      body: expect.not.objectContaining({ meeting_id: expect.anything() }),
    });
  });

  it("uses the literal generated eligible-governance-users contract", async () => {
    const mock = contractClient();
    await expect(createSchoolWizardApi(mock.contract).eligibleGovernanceUsers()).resolves.toEqual({ items: [{ id: "user-1", name: "Director" }], total: 1, page: 1, pageSize: 25 });
    expect(mock.get).toHaveBeenCalledWith("/api/education/governance/eligible-users", {
      params: { query: { page: 1, pageSize: 25, sort: "name", direction: "asc", "filter.name": undefined } },
    });
  });

  it("uses generated scoped selectors for meetings, votes and portfolio owner pairings", async () => {
    const mock = contractClient();
    mock.get
      .mockResolvedValueOnce({ data: { items: [{ id: "meeting-1", title: "CA", meeting_date: "2026-09-01", organism: "ca" }] }, response: new Response(null, { status: 200 }) })
      .mockResolvedValueOnce({ data: { items: [{ id: "vote-1", subject_title: "Buget", agenda_order: 1 }] }, response: new Response(null, { status: 200 }) })
      .mockResolvedValueOnce({ data: { items: [{ user_id: "user-1", personnel_id: "person-1", display_name: "Ana Pop", role_title: "Profesor", employment_status: "active" }] }, response: new Response(null, { status: 200 }) });
    const api = createSchoolWizardApi(mock.contract);
    await expect(api.eligibleGovernanceMeetings()).resolves.toMatchObject({ items: [expect.objectContaining({ id: "meeting-1" })], total: 1, page: 1, pageSize: 25 });
    await expect(api.governanceMeetingVotes("meeting-1")).resolves.toMatchObject({ items: [expect.objectContaining({ id: "vote-1" })], total: 1, page: 1, pageSize: 25 });
    await expect(api.eligiblePortfolioOwners()).resolves.toMatchObject({ items: [expect.objectContaining({ personnel_id: "person-1" })], total: 1, page: 1, pageSize: 25 });
    expect(mock.get).toHaveBeenNthCalledWith(1, "/api/education/governance/meetings", expect.anything());
    expect(mock.get).toHaveBeenNthCalledWith(2, "/api/education/governance/meetings/{meetingID}/votes", expect.objectContaining({ params: expect.objectContaining({ path: { meetingID: "meeting-1" } }) }));
    expect(mock.get).toHaveBeenNthCalledWith(3, "/api/education/portfolios/eligible-owners", expect.anything());
  });

  it("fails explicitly when the generated client reports an API error", async () => {
    const mock = contractClient();
    mock.post.mockResolvedValueOnce({ error: { code: "validation_failed" }, response: new Response(null, { status: 422 }) });
    await expect(createSchoolWizardApi(mock.contract).create("merit", {
      full_name: "Ana", role_title: "Profesor", school_year: "2026-2027", category: "predare",
      status: "draft", score: 0, committee_name: "Comisie", decision_date: "2026-01-01", funded: false, notes: "",
    })).rejects.toThrow("validation_failed");
  });
});

