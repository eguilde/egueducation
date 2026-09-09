import { describe, expect, it, vi } from "vitest";
import { createEducationApi } from "./api";

const requestAt = (fetcher: ReturnType<typeof vi.fn>, index = 0) => fetcher.mock.calls[index][0] as Request;
const urlAt = (fetcher: ReturnType<typeof vi.fn>, index = 0) => requestAt(fetcher, index).url;

describe("Education API", () => {
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

  it("normalizes a legacy array response without losing typed items", async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify([{ id: "meeting-1", title: "Ședință" }])));
    const result = await createEducationApi(fetcher).governanceMeetings();
    expect(result).toMatchObject({ total: 1, items: [{ id: "meeting-1" }] });
  });

  it("flattens the backend taxonomy groups before rendering them as table rows", async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      items: {
        governance: [{ id: "t1", code: "ca", label_ro: "Consiliu de administrație" }],
        portfolios: [{ id: "t2", code: "opis", label_ro: "Opis" }],
      },
    })));
    const result = await createEducationApi(fetcher).relatedRecords("/education/taxonomies");
    expect(result).toMatchObject({ total: 2, items: [{ id: "t1" }, { id: "t2" }] });
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
    await api.saveRecord("personnel", { full_name: "Ana", status: "active" });
    await api.saveRecord("personnel", { full_name: "Ana" }, "p1");
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
    await expect(createEducationApi(fetcher).metadata("/education/mobility/records/filters")).resolves.toEqual({ statuses: ["draft"] });
    expect(new URL(urlAt(fetcher)).pathname).toBe("/api/education/mobility/records/filters");
  });

  it("executes documented portfolio commands with an authenticated POST", async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
    await createEducationApi(fetcher).command("/education/portfolios/records/p1/opis/regenerate");
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
    const fetcher = vi.fn().mockImplementation(() => new Response(JSON.stringify({ items: [{ id: "own-1" }], total: 1, page: 1, pageSize: 1 }), { status: 200 }));
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
