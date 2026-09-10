import { describe, expect, expectTypeOf, it, vi } from "vitest";
import type { components } from "../../api/generated";
import { createEducationApi } from "./api";
import type { EducationMetadataResultByResource, EducationRelatedCreateInputByResource, EducationRelatedRecordByResource, EducationRootCreateInputByDomain, EducationRootRecordByDomain, EducationRootUpdateInputByDomain, GovernanceMeetingInput } from "./types";

const requestAt = (fetcher: ReturnType<typeof vi.fn>, index = 0) => fetcher.mock.calls[index][0] as Request;
const urlAt = (fetcher: ReturnType<typeof vi.fn>, index = 0) => requestAt(fetcher, index).url;

describe("Education API", () => {
  it("keeps all eleven root CRUD DTO maps aligned with generated OpenAPI", () => {
    expectTypeOf<EducationRootRecordByDomain["decisions"]>().toEqualTypeOf<components["schemas"]["GovernanceDecision"]>();
    expectTypeOf<EducationRootRecordByDomain["managerial"]>().toEqualTypeOf<components["schemas"]["ManagerialDossier"]>();
    expectTypeOf<EducationRootRecordByDomain["regulations"]>().toEqualTypeOf<components["schemas"]["RegulationRecord"]>();
    expectTypeOf<EducationRootRecordByDomain["committees"]>().toEqualTypeOf<components["schemas"]["CommitteeRecord"]>();
    expectTypeOf<EducationRootRecordByDomain["personnel"]>().toEqualTypeOf<components["schemas"]["PersonnelRecord"]>();
    expectTypeOf<EducationRootRecordByDomain["evaluations"]>().toEqualTypeOf<components["schemas"]["PersonnelEvaluation"]>();
    expectTypeOf<EducationRootRecordByDomain["declarations"]>().toEqualTypeOf<components["schemas"]["PersonnelDeclaration"]>();
    expectTypeOf<EducationRootRecordByDomain["mobility"]>().toEqualTypeOf<components["schemas"]["MobilityCase"]>();
    expectTypeOf<EducationRootRecordByDomain["merit"]>().toEqualTypeOf<components["schemas"]["MeritGrant"]>();
    expectTypeOf<EducationRootRecordByDomain["portfolios"]>().toEqualTypeOf<components["schemas"]["PortfolioRecord"]>();
    expectTypeOf<EducationRootRecordByDomain["compliance"]>().toEqualTypeOf<components["schemas"]["PublicationRecord"]>();
    expectTypeOf<EducationRootCreateInputByDomain["decisions"]>().toEqualTypeOf<components["schemas"]["CreateGovernanceDecisionRequest"]>();
    expectTypeOf<EducationRootCreateInputByDomain["compliance"]>().toEqualTypeOf<components["schemas"]["CreatePublicationRecordRequest"]>();
    expectTypeOf<EducationRootUpdateInputByDomain["portfolios"]>().toEqualTypeOf<components["schemas"]["UpdatePortfolioRecordRequest"]>();
  });
  it("keeps the related-resource contract closed and makes catalogue bodies read-only", () => {
    expectTypeOf<EducationRelatedRecordByResource["meeting-documents"]>().toEqualTypeOf<components["schemas"]["GovernanceMeetingDocument"]>();
    expectTypeOf<EducationRelatedCreateInputByResource["meeting-documents"]>().toEqualTypeOf<components["schemas"]["CreateGovernanceMeetingDocumentRequest"]>();
    expectTypeOf<EducationRelatedRecordByResource["personnel-access-events"]>().toEqualTypeOf<components["schemas"]["PersonnelPersonalAccessEvent"]>();
    expectTypeOf<EducationRelatedCreateInputByResource["mobility-documents"]>().toEqualTypeOf<components["schemas"]["CreateMobilityDocumentRequest"]>();
    expectTypeOf<EducationRelatedCreateInputByResource["mobility-scores"]>().toEqualTypeOf<components["schemas"]["CreateMobilityCriterionScoreRequest"]>();
    expectTypeOf<EducationRelatedCreateInputByResource["mobility-appeals"]>().toEqualTypeOf<components["schemas"]["CreateMobilityAppealRequest"]>();
    expectTypeOf<EducationRelatedCreateInputByResource["mobility-final-decisions"]>().toEqualTypeOf<components["schemas"]["CreateMobilityFinalDecisionRequest"]>();
    expectTypeOf<EducationRelatedCreateInputByResource["mobility-result-issues"]>().toEqualTypeOf<components["schemas"]["CreateMobilityResultIssueRequest"]>();
    expectTypeOf<EducationRelatedCreateInputByResource["merit-documents"]>().toEqualTypeOf<components["schemas"]["CreateMeritDocumentRequest"]>();
    expectTypeOf<EducationRelatedCreateInputByResource["merit-scores"]>().toEqualTypeOf<components["schemas"]["CreateMeritCriterionScoreRequest"]>();
    expectTypeOf<EducationRelatedCreateInputByResource["merit-appeals"]>().toEqualTypeOf<components["schemas"]["CreateMeritAppealRequest"]>();
    expectTypeOf<EducationRelatedCreateInputByResource["merit-final-decisions"]>().toEqualTypeOf<components["schemas"]["CreateMeritFinalDecisionRequest"]>();
    expectTypeOf<EducationRelatedCreateInputByResource["merit-result-issues"]>().toEqualTypeOf<components["schemas"]["CreateMeritResultIssueRequest"]>();
    expectTypeOf<EducationRelatedCreateInputByResource["governance-bodies"]>().toEqualTypeOf<never>();
  });
  it("keeps governance meeting and every metadata response aligned to generated OpenAPI", () => {
    expectTypeOf<GovernanceMeetingInput>().toEqualTypeOf<components["schemas"]["CreateGovernanceMeetingRequest"]>();
    expectTypeOf<EducationMetadataResultByResource["director-cockpit"]>().toEqualTypeOf<components["schemas"]["DirectorCockpitResponse"]>();
    expectTypeOf<EducationMetadataResultByResource["governance-meeting-finalization"]>().toEqualTypeOf<components["schemas"]["GovernanceMeetingFinalizationSummary"]>();
    expectTypeOf<EducationMetadataResultByResource["committee-completeness"]>().toEqualTypeOf<components["schemas"]["EducationCommitteeCompletenessResponse"]>();
    expectTypeOf<EducationMetadataResultByResource["portfolio-transfer-summary"]>().toEqualTypeOf<components["schemas"]["EducationPortfolioTransferSummaryResponse"]>();
  });
  it("serializes server page, exact sort field and field filters", async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ items: [], total: 0, page: 3, pageSize: 10 }), { headers: { "content-type": "application/json" } }));
    const result = await createEducationApi(fetcher).records("personnel", { page: 3, pageSize: 10, sort: "full_name", direction: "asc", filters: { status: "active", school_year: "2025-2026" } });
    expect(result).toMatchObject({ page: 3, pageSize: 10 });
    const url = urlAt(fetcher);
    expect(url).toContain("page=3");
    expect(url).toContain("pageSize=10");
    expect(url).toContain("sort=full_name");
    expect(url).toContain("direction=asc");
    expect(url).toContain("filter.status=active");
    expect(url).toContain("filter.school_year=2025-2026");
  });
  it("uses the authenticated fetcher and server pagination without a client-selected tenant header", async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ items: [], total: 0, page: 1, pageSize: 50 }), { headers: { "content-type": "application/json" } }));
    const api = createEducationApi(fetcher, "/api");
    await api.governanceMeetings({ q: "consiliu", filters: { status: "scheduled" }, sort: "meeting_date", direction: "desc" });
    const request = requestAt(fetcher);
    const url = request.url;
    expect(url).toContain("/api/education/governance/meetings?");
    expect(url).toContain("q=consiliu");
    expect(url).toContain("filter.status=scheduled");
    expect(request.headers.get("accept")).toBe("application/json");
    expect(request.headers.get("X-Institution-ID")).toBeNull();
  });

  it("returns the generated server page contract for governance meetings", async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ items: [{ id: "meeting-1", title: "Ședință" }], total: 1, page: 1, pageSize: 50 })));
    const result = await createEducationApi(fetcher).governanceMeetings();
    expect(result).toMatchObject({ total: 1, items: [{ id: "meeting-1" }] });
  });

  it("preserves the backend taxonomy groups", async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      items: {
        governance: [{ id: "t1", code: "ca", label_ro: "Consiliu de administrație" }],
        portfolios: [{ id: "t2", code: "opis", label_ro: "Opis" }],
      },
    })));
    const result = await createEducationApi(fetcher).taxonomyCatalog();
    expect(result).toMatchObject({ items: { governance: [{ id: "t1" }], portfolios: [{ id: "t2" }] } });
  });

  it("sends eligible-governance-user search and paging through the generated query contract", async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      items: [{ id: "user-1", name: "Director Test" }],
      total: 1,
      page: 1,
      pageSize: 100,
    }), { headers: { "content-type": "application/json" } }));

    const users = await createEducationApi(fetcher).eligibleGovernanceUsers({
      q: "Director Test",
      page: 1,
      pageSize: 100,
    });

    expect(users).toEqual([{ id: "user-1", name: "Director Test" }]);
    const url = new URL(urlAt(fetcher));
    expect(url.pathname).toBe("/api/education/governance/eligible-users");
    expect(url.searchParams.get("filter.name")).toBe("Director Test");
    expect(url.searchParams.get("page")).toBe("1");
    expect(url.searchParams.get("pageSize")).toBe("100");
    expect(url.searchParams.get("sort")).toBe("name");
    expect(url.searchParams.get("direction")).toBe("asc");
  });

  it("uses literal generated routes for governance related lists and keeps bodies catalogue read-only", async () => {
    const fetcher = vi.fn().mockImplementation(() => Promise.resolve(new Response(JSON.stringify({ items: [], total: 0, page: 1, pageSize: 20 }), { headers: { "content-type": "application/json" } })));
    const api = createEducationApi(fetcher);
    await api.relatedRecords("meeting-documents", "meeting-1", { page: 1, pageSize: 20, sort: "registered_on", direction: "desc" });
    expect(new URL(urlAt(fetcher)).pathname).toBe("/api/education/governance/meetings/meeting-1/documents");
    expect(new URL(urlAt(fetcher)).searchParams.get("sort")).toBe("registered_on");
    await api.relatedRecords("governance-bodies", undefined, { page: 2, pageSize: 10 });
    expect(new URL(urlAt(fetcher, 1)).pathname).toBe("/api/education/governance/bodies");
    expect(requestAt(fetcher, 1).method).toBe("GET");
  });

  it("maps governance forms to their exact request DTOs and sends only documented relation filters", async () => {
    const fetcher = vi.fn().mockImplementation((request: Request) => Promise.resolve(
      request.method === "GET"
        ? new Response(JSON.stringify({ items: [], total: 0, page: 2, pageSize: 10 }), { headers: { "content-type": "application/json" } })
        : new Response(JSON.stringify({ id: "related-1" }), { headers: { "content-type": "application/json" } }),
    ));
    const api = createEducationApi(fetcher);

    await api.relatedRecords("meeting-participants", "meeting-1", { page: 2, pageSize: 10, sort: "attendance_status", direction: "desc", q: "ignored", filters: { full_name: "Ana", attendance_status: "prezent", unknown: "never" } });
    await api.saveRelated("meeting-participants", "meeting-1", { full_name: "Ana Pop", role_name: "Profesor", member_type: "membru", attendance_status: "prezent", voting_right: true, signature_present: false, id: "server-owned" });
    await api.saveRelated("regulation-versions", "regulation-1", { version_label: "1.0", version_status: "draft", prepared_by: "Director", effective_from: "2026-09-15", change_summary: "Actualizare anuală", approved_on: "2026-09-10", record_id: "server-owned" });
    await api.saveRelated("regulation-workflow", "regulation-1", { phase_order: 1, phase_type: "consultare", audience: "personal", started_on: "2026-09-01", due_on: "2026-09-10", status: "open", feedback_count: 2, assigned_to: "server-owned", outcome_note: "server-owned" });
    await api.saveRelated("managerial-documents", "dossier-1", { document_category: "plan", title: "Plan managerial", document_status: "draft", version_label: "v1", registered_on: "2026-09-01", mandatory: true, document_code: "server-owned" });

    const participantList = new URL(urlAt(fetcher, 0));
    expect(participantList.searchParams.get("filter.full_name")).toBe("Ana");
    expect(participantList.searchParams.get("filter.attendance_status")).toBe("prezent");
    expect(participantList.searchParams.has("q")).toBe(false);
    expect(participantList.searchParams.has("filter.unknown")).toBe(false);
    await expect(requestAt(fetcher, 1).clone().json()).resolves.toEqual({ full_name: "Ana Pop", role_name: "Profesor", member_type: "membru", attendance_status: "prezent", voting_right: true, signature_present: false });
    await expect(requestAt(fetcher, 2).clone().json()).resolves.toEqual({ version_label: "1.0", version_status: "draft", prepared_by: "Director", effective_from: "2026-09-15", change_summary: "Actualizare anuală", approved_on: "2026-09-10" });
    await expect(requestAt(fetcher, 3).clone().json()).resolves.toEqual({ phase_order: 1, phase_type: "consultare", audience: "personal", started_on: "2026-09-01", due_on: "2026-09-10", status: "open", feedback_count: 2 });
    await expect(requestAt(fetcher, 4).clone().json()).resolves.toEqual({ document_category: "plan", title: "Plan managerial", document_status: "draft", version_label: "v1", registered_on: "2026-09-01", mandatory: true });
  });

  it("uses literal generated personnel and evaluation routes with allow-listed bodies", async () => {
    const fetcher = vi.fn().mockImplementation((request: Request) => Promise.resolve(
      request.headers.get("accept") === "application/pdf"
        ? new Response(new Blob(["pdf"], { type: "application/pdf" }), { status: 200 })
        : request.method === "DELETE"
          ? new Response(null, { status: 204 })
          : new Response(JSON.stringify(request.method === "GET" ? { items: [], total: 0, page: 1, pageSize: 20 } : { id: "related-1" }), { status: 200, headers: { "content-type": "application/json" } }),
    ));
    const api = createEducationApi(fetcher);
    await api.relatedRecords("personnel-assignments", "person-1", { sort: "assigned_on", direction: "desc", filters: { assignment_code: "A-1", status: "ignored" } });
    await api.saveRelated("personnel-assignments", "person-1", { assigned_on: "2026-09-01", assignment_title: "Profesor", assignment_type: "titular", status: "active", assignment_code: "server-owned" });
    await api.saveRelated("personnel-file-documents", "person-1", { confidentiality_level: "restricted", document_category: "contract", document_title: "Contract", file_scope: "personnel", issued_on: "2026-09-01", document_code: "server-owned" });
    await api.saveRelated("personnel-disciplinary-cases", "person-1", { case_type: "review", reported_on: "2026-09-01", status: "open", case_code: "server-owned" });
    await api.saveRelated("personnel-access-events", "person-1", { access_channel: "ui", accessed_on: "2026-09-01", actor_name: "Administrator", actor_role: "admin", event_type: "view", purpose: "audit", institution_id: "server-owned" });
    await api.saveRelated("evaluation-self-reviews", "evaluation-1", { completed_on: "2026-09-01", narrative_type: "annual", section_title: "Activitate", status: "draft", review_code: "server-owned" });
    await api.saveRelated("evaluation-criteria", "evaluation-1", { criterion_category: "teaching", criterion_label: "Calitate", max_score: 10, status: "draft", criterion_code: "server-owned" });
    await api.saveRelated("evaluation-appeals", "evaluation-1", { grounds: "motiv", status: "submitted", submitted_by: "Ana", submitted_on: "2026-09-01", appeal_code: "server-owned" });
    await api.saveRelated("evaluation-result-issues", "evaluation-1", { delivery_channel: "email", delivery_status: "sent", document_type: "result", issued_on: "2026-09-01", recipient_name: "Ana", issue_code: "server-owned" });
    await api.relatedPdf("evaluation-appeals", "evaluation-1", "appeal-1");
    await api.relatedPdf("evaluation-result-issues", "evaluation-1", "issue-1");
    await api.deleteRelated("personnel-access-events", "person-1", "event-1");

    expect(new URL(urlAt(fetcher, 0)).pathname).toBe("/api/education/personnel/records/person-1/assignments");
    const listUrl = new URL(urlAt(fetcher, 0));
    expect(listUrl.searchParams.get("filter.assignment_code")).toBe("A-1");
    expect(listUrl.searchParams.has("filter.status")).toBe(false);
    expect(new URL(urlAt(fetcher, 1)).pathname).toBe("/api/education/personnel/records/person-1/assignments");
    await expect(requestAt(fetcher, 1).clone().json()).resolves.toEqual({ assigned_on: "2026-09-01", assignment_title: "Profesor", assignment_type: "titular", status: "active" });
    expect(new URL(urlAt(fetcher, 8)).pathname).toBe("/api/education/evaluations/records/evaluation-1/result-issues");
    await expect(requestAt(fetcher, 8).clone().json()).resolves.not.toHaveProperty("issue_code");
    expect(new URL(urlAt(fetcher, 9)).pathname).toBe("/api/education/evaluations/records/evaluation-1/appeals/appeal-1/pdf");
    expect(requestAt(fetcher, 9).headers.get("accept")).toBe("application/pdf");
    expect(new URL(urlAt(fetcher, 10)).pathname).toBe("/api/education/evaluations/records/evaluation-1/result-issues/issue-1/pdf");
    expect(requestAt(fetcher, 11).method).toBe("DELETE");
    expect(new URL(urlAt(fetcher, 11)).pathname).toBe("/api/education/personnel/records/person-1/access-events/event-1");
  });

  it("uses literal generated mobility and merit routes, their sole documented filters, allow-listed DTO bodies and protected PDFs", async () => {
    const fetcher = vi.fn().mockImplementation((request: Request) => Promise.resolve(
      request.headers.get("accept") === "application/pdf"
        ? new Response(new Blob(["pdf"], { type: "application/pdf" }), { status: 200 })
        : request.method === "DELETE"
          ? new Response(null, { status: 204 })
          : new Response(JSON.stringify(request.method === "GET" ? { items: [], total: 0, page: 1, pageSize: 20 } : { id: "related-1" }), { status: 200, headers: { "content-type": "application/json" } }),
    ));
    const api = createEducationApi(fetcher);
    const resources = [
      { resource: "mobility-documents", filter: "document_code", path: "/api/education/mobility/records/mobility-1/documents", input: { document_title: "Cerere", document_type: "request", registered_on: "2026-09-01", stage_scope: "transfer", validation_status: "accepted", document_code: "server-owned" } },
      { resource: "mobility-scores", filter: "criterion_code", path: "/api/education/mobility/records/mobility-1/scores", input: { criterion_category: "seniority", criterion_code: "C1", criterion_label: "Vechime", max_score: 10, criterion_score_id: "server-owned" } },
      { resource: "mobility-appeals", filter: "appeal_code", path: "/api/education/mobility/records/mobility-1/appeals", input: { grounds: "motiv", status: "submitted", submitted_by: "Ana", submitted_on: "2026-09-01", appeal_code: "server-owned" } },
      { resource: "mobility-final-decisions", filter: "decision_code", path: "/api/education/mobility/records/mobility-1/final-decisions", input: { approved_on: "2026-09-01", decision_type: "transfer", effective_from: "2026-09-02", outcome: "approved", panel_name: "Comisie", decision_code: "server-owned" } },
      { resource: "mobility-result-issues", filter: "issue_code", path: "/api/education/mobility/records/mobility-1/result-issues", input: { delivery_channel: "email", delivery_status: "sent", document_type: "result", issued_on: "2026-09-01", recipient_name: "Ana", issue_code: "server-owned" } },
      { resource: "merit-documents", filter: "document_code", path: "/api/education/gradatii/records/merit-1/documents", input: { document_title: "Dosar", document_type: "dossier", registered_on: "2026-09-01", validation_status: "accepted", document_code: "server-owned" } },
      { resource: "merit-scores", filter: "criterion_code", path: "/api/education/gradatii/records/merit-1/scores", input: { criterion_category: "results", criterion_code: "C1", criterion_label: "Rezultate", max_score: 10, panel_stage: "evaluation", criterion_score_id: "server-owned" } },
      { resource: "merit-appeals", filter: "appeal_code", path: "/api/education/gradatii/records/merit-1/appeals", input: { grounds: "motiv", status: "submitted", submitted_by: "Ana", submitted_on: "2026-09-01", appeal_code: "server-owned" } },
      { resource: "merit-final-decisions", filter: "decision_code", path: "/api/education/gradatii/records/merit-1/final-decisions", input: { approved_on: "2026-09-01", decision_stage: "final", effective_from: "2026-09-02", outcome: "approved", panel_name: "Comisie", decision_code: "server-owned" } },
      { resource: "merit-result-issues", filter: "issue_code", path: "/api/education/gradatii/records/merit-1/result-issues", input: { delivery_channel: "email", delivery_status: "sent", document_type: "result", issued_on: "2026-09-01", recipient_name: "Ana", issue_code: "server-owned" } },
    ] as const;
    for (const item of resources) {
      const parentID = item.resource.startsWith("mobility-") ? "mobility-1" : "merit-1";
      await api.relatedRecords(item.resource, parentID, { page: 2, pageSize: 10, sort: "created_at", direction: "desc", filters: { [item.filter]: "F-1", ignored: "never-send" } });
      await api.saveRelated(item.resource, parentID, item.input);
      await api.saveRelated(item.resource, parentID, item.input, "item-1");
      await api.deleteRelated(item.resource, parentID, "item-1");
    }
    for (const resource of ["mobility-appeals", "mobility-final-decisions", "mobility-result-issues", "merit-appeals", "merit-final-decisions", "merit-result-issues"] as const) {
      await api.relatedPdf(resource, resource.startsWith("mobility-") ? "mobility-1" : "merit-1", "item-1");
    }
    const requests = fetcher.mock.calls.map(([,], index) => requestAt(fetcher, index));
    resources.forEach((item, index) => {
      const start = index * 4;
      const list = new URL(requests[start].url);
      expect(list.pathname).toBe(item.path);
      expect(list.searchParams.get(`filter.${item.filter}`)).toBe("F-1");
      expect(list.searchParams.has("filter.ignored")).toBe(false);
      expect(new URL(requests[start + 1].url).pathname).toBe(item.path);
      expect(requests[start + 1].method).toBe("POST");
      expect(new URL(requests[start + 2].url).pathname).toBe(`${item.path}/item-1`);
      expect(requests[start + 2].method).toBe("PATCH");
      expect(requests[start + 3].method).toBe("DELETE");
      expect(new URL(requests[start + 3].url).pathname).toBe(`${item.path}/item-1`);
    });
    const postedBodies = await Promise.all(resources.map((_, index) => requests[index * 4 + 1].clone().json()));
    postedBodies.forEach((body) => {
      expect(body).not.toHaveProperty("document_code");
      expect(body).not.toHaveProperty("appeal_code");
      expect(body).not.toHaveProperty("decision_code");
      expect(body).not.toHaveProperty("issue_code");
      expect(body).not.toHaveProperty("criterion_score_id");
    });
    const pdfRequests = requests.slice(resources.length * 4);
    expect(pdfRequests).toHaveLength(6);
    pdfRequests.forEach((request) => {
      expect(request.headers.get("accept")).toBe("application/pdf");
      expect(new URL(request.url).pathname).toMatch(/\/(appeals|final-decisions|result-issues)\/item-1\/pdf$/);
    });
  });

  it("uses named Portfolio document transport with its sole documented filter", async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ items: [], total: 0, page: 1, pageSize: 20 }), { headers: { "content-type": "application/json" } }));
    const api = createEducationApi(fetcher);
    await api.portfolioDocuments("portfolio-1", { page: 1, pageSize: 20, sort: "issued_on", direction: "desc", sectionCode: "S1" });
    const url = new URL(urlAt(fetcher));
    expect(url.pathname).toBe("/api/education/portfolios/records/portfolio-1/documents");
    expect(url.searchParams.get("filter.section_code")).toBe("S1");
    expect(url.searchParams.has("filter.status")).toBe(false);
  });

  it("keeps transfer creation out of the history adapter", async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ items: [], total: 0, page: 1, pageSize: 20 }), { headers: { "content-type": "application/json" } }));
    await createEducationApi(fetcher).portfolioTransferHistory("portfolio-1", { transferCode: "TR-1" });
    expect(requestAt(fetcher).method).toBe("GET");
    expect(new URL(urlAt(fetcher)).pathname).toBe("/api/education/portfolios/records/portfolio-1/transfers");
    expect(new URL(urlAt(fetcher)).searchParams.get("filter.transfer_code")).toBe("TR-1");
  });

  it("routes each non-governance catalogue domain to its backend records endpoint", async () => {
    const fetcher = vi.fn().mockImplementation(() => Promise.resolve(new Response(JSON.stringify([]))));
    const api = createEducationApi(fetcher);
    await api.records("portfolios", { q: "opis" });
    expect(urlAt(fetcher)).toContain("/api/education/portfolios/records?");
    expect(urlAt(fetcher)).toContain("q=opis");
    expect(requestAt(fetcher).headers.get("X-Institution-ID")).toBeNull();
  });

  it("has a concrete list route for every non-placeholder School domain", async () => {
    const fetcher = vi.fn().mockImplementation(() => Promise.resolve(new Response(JSON.stringify([]))));
    const api = createEducationApi(fetcher);
    const domains = ["decisions", "managerial", "regulations", "committees", "personnel", "evaluations", "declarations", "mobility", "merit", "portfolios", "compliance"] as const;
    await Promise.all(domains.map((domain) => api.records(domain)));
    expect(fetcher).toHaveBeenCalledTimes(domains.length);
    expect(fetcher.mock.calls.map(([,], index) => urlAt(fetcher, index))).toEqual(expect.arrayContaining([
      expect.stringContaining("/education/decisions/records?"),
      expect.stringContaining("/education/compliance/publications?"),
      expect.stringContaining("/education/gradatii/records?"),
    ]));
  });

  it("uses the documented CRUD route and JSON method for a dossier record", async () => {
    const fetcher = vi.fn().mockImplementation(() => Promise.resolve(new Response(JSON.stringify({ id: "p1", full_name: "Ana" }), { status: 200 })));
    const api = createEducationApi(fetcher);
    const personnel = { full_name: "Ana", status: "active", employment_type: "permanent", evaluation_status: "current", mobility_stage: "none", role_title: "Profesor", school_year: "2026-2027" };
    await api.createRecord("personnel", personnel);
    await api.updateRecord("personnel", "p1", personnel);
    await api.deleteRecord("personnel", "p1");
    expect(new URL(urlAt(fetcher)).pathname).toBe("/api/education/personnel/records");
    expect(requestAt(fetcher).method).toBe("POST");
    expect(requestAt(fetcher).headers.get("content-type")).toBe("application/json");
    expect(new URL(urlAt(fetcher, 1)).pathname).toBe("/api/education/personnel/records/p1");
    expect(requestAt(fetcher, 1).method).toBe("PATCH");
    expect(requestAt(fetcher, 2).method).toBe("DELETE");
  });

  it("retrieves protected dossier PDFs through the authenticated fetcher", async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(new Blob(["pdf"], { type: "application/pdf" }), { status: 200 }));
    await expect(createEducationApi(fetcher).recordPdf("portfolios", "p 1")).resolves.toBeInstanceOf(Blob);
    expect(new URL(urlAt(fetcher)).pathname).toBe("/api/education/portfolios/records/p%201/pdf");
    expect(requestAt(fetcher).headers.get("accept")).toBe("application/pdf");
  });

  it("loads documented dashboard and filter metadata through the same authenticated client", async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ statuses: ["draft"] }), { status: 200 }));
    await expect(createEducationApi(fetcher).metadata("mobility-filters")).resolves.toEqual({ statuses: ["draft"] });
    expect(new URL(urlAt(fetcher)).pathname).toBe("/api/education/mobility/records/filters");
  });

  it("uses literal generated metadata paths and their required path keys", async () => {
    const fetcher = vi.fn().mockImplementation(() => Promise.resolve(new Response(JSON.stringify({}), { status: 200, headers: { "content-type": "application/json" } })));
    const api = createEducationApi(fetcher);
    await api.metadata("director-cockpit");
    await api.metadata("governance-meeting-filters");
    await api.metadata("governance-meeting-finalization", "meeting-1");
    await api.metadata("governance-body-completeness", "body-1");
    await api.metadata("decisions-dashboard");
    await api.metadata("decisions-filters");
    await api.metadata("managerial-dashboard");
    await api.metadata("managerial-filters");
    await api.metadata("regulations-dashboard");
    await api.metadata("regulations-filters");
    await api.metadata("personnel-dashboard");
    await api.metadata("personnel-filters");
    await api.metadata("evaluations-dashboard");
    await api.metadata("evaluations-filters");
    await api.metadata("declarations-dashboard");
    await api.metadata("declarations-filters");
    await api.metadata("mobility-dashboard");
    await api.metadata("mobility-filters");
    await api.metadata("merit-dashboard");
    await api.metadata("merit-filters");
    await api.metadata("portfolios-dashboard");
    await api.metadata("portfolios-filters");
    await api.metadata("committee-completeness", "committee-1");
    await api.metadata("managerial-portfolio-summary", "managerial-1");
    await api.metadata("personnel-portfolio-dossier-summary", "personnel-1");
    await api.metadata("portfolio-transfer-summary", "portfolio-1");
    await api.metadata("regulation-procedural-summary", "regulation-1");
    expect(fetcher.mock.calls.map((_, index) => new URL(urlAt(fetcher, index)).pathname)).toEqual([
      "/api/education/director/cockpit", "/api/education/governance/meetings/filters", "/api/education/governance/meetings/meeting-1/finalization-summary", "/api/education/governance/bodies/body-1/completeness-summary",
      "/api/education/decisions/dashboard", "/api/education/decisions/records/filters", "/api/education/managerial/dashboard", "/api/education/managerial/records/filters", "/api/education/regulations/dashboard", "/api/education/regulations/records/filters", "/api/education/personnel/dashboard", "/api/education/personnel/records/filters", "/api/education/evaluations/dashboard", "/api/education/evaluations/records/filters", "/api/education/declarations/dashboard", "/api/education/declarations/records/filters", "/api/education/mobility/dashboard", "/api/education/mobility/records/filters", "/api/education/gradatii/dashboard", "/api/education/gradatii/records/filters", "/api/education/portfolios/dashboard", "/api/education/portfolios/records/filters", "/api/education/committees/records/committee-1/completeness-summary", "/api/education/managerial/records/managerial-1/portfolio-summary", "/api/education/personnel/records/personnel-1/portfolio-dossier-summary", "/api/education/portfolios/records/portfolio-1/transfer-summary", "/api/education/regulations/records/regulation-1/procedural-summary",
    ]);
  });

  it("uses literal governance meeting CRUD paths and the exact generated request body", async () => {
    const fetcher = vi.fn().mockImplementation(() => Promise.resolve(new Response(JSON.stringify({ id: "meeting-1" }), { status: 200, headers: { "content-type": "application/json" } })));
    const input: GovernanceMeetingInput = { chairperson_user_id: "11111111-1111-4111-8111-111111111111", secretary_user_id: "22222222-2222-4222-8222-222222222222", meeting_date: "2026-09-10", meeting_type: "ordinary", organism: "CA", school_year: "2026-2027", status: "draft", title: "Ședință" };
    const api = createEducationApi(fetcher);
    await api.saveGovernanceMeeting(input);
    await api.saveGovernanceMeeting(input, "meeting-1");
    await api.deleteGovernanceMeeting("meeting-1");
    expect(new URL(urlAt(fetcher, 0)).pathname).toBe("/api/education/governance/meetings");
    expect(requestAt(fetcher, 0).method).toBe("POST");
    await expect(requestAt(fetcher, 0).clone().json()).resolves.toEqual(input);
    expect(new URL(urlAt(fetcher, 1)).pathname).toBe("/api/education/governance/meetings/meeting-1");
    expect(requestAt(fetcher, 1).method).toBe("PATCH");
    expect(requestAt(fetcher, 2).method).toBe("DELETE");
  });

  it("executes documented portfolio commands with an authenticated POST", async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
    await createEducationApi(fetcher).command("portfolio-opis-regenerate", "p1");
    expect(new URL(urlAt(fetcher)).pathname).toBe("/api/education/portfolios/records/p1/opis/regenerate");
    expect(requestAt(fetcher).method).toBe("POST");
  });

  it("sends the affirmative declaration acknowledgement through the generated transport", async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      id: "ack-1",
      declaration_type: "authenticity",
    }), { headers: { "content-type": "application/json" } }));
    const api = createEducationApi(fetcher);

    await api.acknowledgeOwnPortfolioDeclaration("portfolio-1", "authenticity");

    const request = requestAt(fetcher);
    expect(request.method).toBe("POST");
    expect(request.url).toContain("/api/education/portfolios/me/portfolio-1/declarations/authenticity/acknowledgements");
    await expect(request.clone().json()).resolves.toEqual({ confirmed: true });
  });

  it("uses only the owner-scoped portfolio contract for teacher self-service", async () => {
    const fetcher = vi.fn().mockImplementation((request: Request) => {
      const pathname = new URL(request.url).pathname;
      const isPage = request.method === "GET" && (pathname.endsWith("/portfolios/me") || pathname.endsWith("/opis") || pathname.endsWith("/archive-documents"));
      return new Response(JSON.stringify(isPage ? { items: [{ id: "own-1" }], total: 1, page: 1, pageSize: 1 } : { id: "own-1" }), { status: 200 });
    });
    const api = createEducationApi(fetcher);
    await api.ownPortfolios();
    await api.ownPortfolio("own-1");
    await api.createOwnPortfolio({ school_year: "2026-2027", last_updated_on: "2026-09-01", notes: "" });
    await api.updateOwnPortfolio("own-1", { school_year: "2026-2027", last_updated_on: "2026-09-02", notes: "actualizat" });
    await api.submitOwnPortfolio("own-1");
    await api.ownPortfolioRelated("own-1", "opis");
    await api.regenerateOwnPortfolioOpis("own-1");
    await api.createOwnPortfolioDocument("own-1", { section_code: "S1", component_code: "C1", document_title: "Planificare", evidence_type: "document", issued_on: "2026-09-01", added_on: "2026-09-01", chronological_index: 1, sensitive_data: false, file_reference: "archive://document-1", notes: "" });
    await api.deleteOwnPortfolioDocument("own-1", "document-1");
    await api.ownPortfolioArchiveDocuments();
    expect(new URL(urlAt(fetcher)).pathname).toBe("/api/education/portfolios/me");
    expect(new URL(urlAt(fetcher, 1)).pathname).toBe("/api/education/portfolios/me/own-1");
    expect(requestAt(fetcher, 2).method).toBe("POST");
    expect(requestAt(fetcher, 3).method).toBe("PATCH");
    expect(new URL(urlAt(fetcher, 4)).pathname).toBe("/api/education/portfolios/me/own-1/submit");
    expect(new URL(urlAt(fetcher, 5)).pathname).toBe("/api/education/portfolios/me/own-1/opis");
    expect(new URL(urlAt(fetcher, 6)).pathname).toBe("/api/education/portfolios/me/own-1/opis/regenerate");
    expect(requestAt(fetcher, 6).method).toBe("POST");
    expect(new URL(urlAt(fetcher, 7)).pathname).toBe("/api/education/portfolios/me/own-1/documents");
    expect(requestAt(fetcher, 7).method).toBe("POST");
    expect(new URL(urlAt(fetcher, 8)).pathname).toBe("/api/education/portfolios/me/own-1/documents/document-1");
    expect(requestAt(fetcher, 8).method).toBe("DELETE");
    expect(new URL(urlAt(fetcher, 9)).pathname).toBe("/api/education/portfolios/me/archive-documents");
    expect(urlAt(fetcher, 9)).toContain("sort=title");
    expect(fetcher.mock.calls.map((call) => String(call[0])).join(" ")).not.toContain("/records/own-1");
  });

  it("uses the dedicated tenant-scoped archive grant administration contracts", async () => {
    const fetcher = vi.fn().mockImplementation((request: Request) => Promise.resolve(
      request.method === "DELETE"
        ? new Response(null, { status: 204 })
        : new Response(
          JSON.stringify({ items: [], total: 0, page: 1, pageSize: 50 }),
          { status: 200, headers: { "content-type": "application/json" } },
        ),
    ));
    const api = createEducationApi(fetcher);
    await api.attachmentGrants();
    await api.eligibleAttachmentDocuments();
    await api.eligibleAttachmentUsers();
    await api.createAttachmentGrant({ archive_document_id: "11111111-1111-4111-8111-111111111111", grantee_user_id: "22222222-2222-4222-8222-222222222222" });
    await api.deleteAttachmentGrant("33333333-3333-4333-8333-333333333333");
    expect(new URL(urlAt(fetcher, 0)).pathname).toBe("/api/education/portfolios/archive-attachment-grants");
    expect(new URL(urlAt(fetcher, 1)).pathname).toBe("/api/education/portfolios/archive-attachment-grants/eligible-documents");
    expect(new URL(urlAt(fetcher, 2)).pathname).toBe("/api/education/portfolios/archive-attachment-grants/eligible-users");
    expect(requestAt(fetcher, 3).method).toBe("POST");
    await expect(requestAt(fetcher, 3).clone().json()).resolves.toEqual({ archive_document_id: "11111111-1111-4111-8111-111111111111", grantee_user_id: "22222222-2222-4222-8222-222222222222" });
    expect(requestAt(fetcher, 4).method).toBe("DELETE");
  });
});
