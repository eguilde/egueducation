import { describe, expect, it, vi } from "vitest";
import { createEducationApi } from "./api";

describe("Education API", () => {
  it("uses the authenticated fetcher and server pagination without a client-selected tenant header", async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ items: [], total: 0, page: 1, pageSize: 50 }), { headers: { "content-type": "application/json" } }));
    const api = createEducationApi(fetcher, "/api");
    await api.governanceMeetings({ q: "consiliu", filters: { status: "scheduled" }, sort: "meeting_date", direction: "desc" });
    const [url, init] = fetcher.mock.calls[0] as [string, RequestInit];
    expect(url).toContain("/api/education/governance/meetings?");
    expect(url).toContain("q=consiliu");
    expect(url).toContain("filter.status=scheduled");
    expect(init.headers).toMatchObject({ Accept: "application/json" });
    expect(init.headers).not.toHaveProperty("X-Institution-ID");
  });

  it("normalizes a legacy array response without losing typed items", async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify([{ id: "meeting-1", title: "Ședință" }])));
    const result = await createEducationApi(fetcher).governanceMeetings();
    expect(result).toMatchObject({ total: 1, items: [{ id: "meeting-1" }] });
  });

  it("routes each non-governance catalogue domain to its backend records endpoint", async () => {
    const fetcher = vi.fn().mockImplementation(() => Promise.resolve(new Response(JSON.stringify([]))));
    const api = createEducationApi(fetcher);
    await api.records("portfolios", { q: "opis" });
    expect(fetcher.mock.calls[0][0]).toContain("/api/education/portfolios/records?");
    expect(fetcher.mock.calls[0][0]).toContain("q=opis");
    expect(fetcher.mock.calls[0][1].headers).not.toHaveProperty("X-Institution-ID");
  });

  it("has a concrete list route for every non-placeholder School domain", async () => {
    const fetcher = vi.fn().mockImplementation(() => Promise.resolve(new Response(JSON.stringify([]))));
    const api = createEducationApi(fetcher);
    const domains = ["decisions", "managerial", "regulations", "committees", "personnel", "evaluations", "declarations", "mobility", "merit", "portfolios", "compliance"] as const;
    await Promise.all(domains.map((domain) => api.records(domain)));
    expect(fetcher).toHaveBeenCalledTimes(domains.length);
    expect(fetcher.mock.calls.map(([url]) => String(url))).toEqual(expect.arrayContaining([
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
    expect(fetcher.mock.calls[0][0]).toBe("/api/education/personnel/records");
    expect(fetcher.mock.calls[0][1]).toMatchObject({ method: "POST", headers: { "Content-Type": "application/json" } });
    expect(fetcher.mock.calls[1][0]).toBe("/api/education/personnel/records/p1");
    expect(fetcher.mock.calls[1][1]).toMatchObject({ method: "PUT" });
    expect(fetcher.mock.calls[2][1]).toMatchObject({ method: "DELETE" });
  });

  it("retrieves protected dossier PDFs through the authenticated fetcher", async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(new Blob(["pdf"], { type: "application/pdf" }), { status: 200 }));
    await expect(createEducationApi(fetcher).recordPdf("portfolios", "p 1")).resolves.toBeInstanceOf(Blob);
    expect(fetcher.mock.calls[0][0]).toBe("/api/education/portfolios/records/p%201/pdf");
    expect(fetcher.mock.calls[0][1].headers).toMatchObject({ Accept: "application/pdf" });
  });

  it("uses protected backend endpoints for the PDF and CSV exports", async () => {
    const fetcher = vi.fn().mockImplementation(() => Promise.resolve(new Response(new Blob(["data"]), { status: 200 })));
    const api = createEducationApi(fetcher);
    await api.exportFile("pdf"); await api.exportFile("csv");
    expect(fetcher.mock.calls[0][0]).toBe("/api/education/exports/pdf");
    expect(fetcher.mock.calls[1][0]).toBe("/api/education/exports/csv");
  });

  it("loads documented dashboard and filter metadata through the same authenticated client", async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ statuses: ["draft"] }), { status: 200 }));
    await expect(createEducationApi(fetcher).metadata("/education/mobility/records/filters")).resolves.toEqual({ statuses: ["draft"] });
    expect(fetcher.mock.calls[0][0]).toBe("/api/education/mobility/records/filters");
  });

  it("executes documented portfolio commands with an authenticated POST", async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
    await createEducationApi(fetcher).command("/education/portfolios/records/p1/opis/regenerate");
    expect(fetcher.mock.calls[0][0]).toBe("/api/education/portfolios/records/p1/opis/regenerate");
    expect(fetcher.mock.calls[0][1]).toMatchObject({ method: "POST" });
  });
});
