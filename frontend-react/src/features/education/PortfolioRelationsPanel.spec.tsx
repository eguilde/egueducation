import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { PrimeReactProvider } from "@primereact/core/config";
import { describe, expect, it, vi } from "vitest";
import { PortfolioRelationsPanel } from "./EducationWorkspace";
import type { EducationApi } from "./types";

const page = (items: Array<Record<string, unknown>> = []) => Promise.resolve({ items, total: items.length, page: 1, pageSize: 20 });

function apiMock(): EducationApi {
  return {
    portfolioDocuments: vi.fn(() => page()), portfolioDocumentVersions: vi.fn(() => page()), portfolioChecklist: vi.fn(() => page()), portfolioOpis: vi.fn(() => page()), portfolioCustody: vi.fn(() => page()), portfolioReviews: vi.fn(() => page()), portfolioTransferHistory: vi.fn(() => page()), portfolioValorifications: vi.fn(() => page()),
    createPortfolioDocument: vi.fn(async () => ({ id: "doc-1" })), updatePortfolioDocument: vi.fn(async () => ({ id: "doc-1" })), deletePortfolioDocument: vi.fn(async () => undefined),
    createPortfolioChecklistItem: vi.fn(async () => ({ id: "check-1" })), updatePortfolioChecklistItem: vi.fn(async () => ({ id: "check-1" })), deletePortfolioChecklistItem: vi.fn(async () => undefined),
    createPortfolioOpisEntry: vi.fn(async () => ({ id: "opis-1" })), updatePortfolioOpisEntry: vi.fn(async () => ({ id: "opis-1" })), deletePortfolioOpisEntry: vi.fn(async () => undefined),
    createPortfolioCustodyEvent: vi.fn(async () => ({ id: "custody-1" })), updatePortfolioCustodyEvent: vi.fn(async () => ({ id: "custody-1" })), deletePortfolioCustodyEvent: vi.fn(async () => undefined),
  } as unknown as EducationApi;
}

function renderPanel(api: EducationApi, canManage = true) {
  return render(<PrimeReactProvider><PortfolioRelationsPanel api={api} recordID="portfolio-1" canManage={canManage} /></PrimeReactProvider>);
}
async function set(label: string, value: string) {
  fireEvent.change(await screen.findByLabelText(label), { target: { value } });
}
async function openAdd(label: string) {
  await screen.findByRole("button", { name: `Adaugă ${label}` });
  fireEvent.click(screen.getByRole("button", { name: `Adaugă ${label}` }));
}

describe("PortfolioRelationsPanel contractual managers", () => {
  it("sends an exact document DTO without server-owned output fields", async () => {
    const api = apiMock(); renderPanel(api);
    await openAdd("document");
    await set("Titlu", "Planificare"); await set("Descriere pedagogică", "Activitate didactică"); await set("An școlar", "2026-2027"); await set("Disciplina", "Matematică"); await set("Clasa aplicabilă", "IV A"); await set("Competențe (separate prin virgulă)", "C1, C2"); await set("Tip dovadă", "plan"); await set("Secțiune", "I"); await set("Componentă", "I.1"); await set("Domeniu sursă", "portofoliu"); await set("Autenticitate", "declared"); await set("Data emiterii", "2026-09-01"); await set("Data adăugării", "2026-09-02"); await set("Referință arhivă", "archive://doc-1");
    fireEvent.click(screen.getByRole("button", { name: "Salvează" }));
    await waitFor(() => expect(api.createPortfolioDocument).toHaveBeenCalledWith("portfolio-1", {
      added_on: "2026-09-02", applicable_class: "IV A", authenticity_status: "declared", chronological_index: undefined, competencies: ["C1", "C2"], component_code: "I.1", description: "Activitate didactică", document_title: "Planificare", evidence_type: "plan", file_reference: "archive://doc-1", issued_on: "2026-09-01", notes: undefined, school_year: "2026-2027", section_code: "I", sensitive_data: undefined, source_scope: "portofoliu", subject_discipline: "Matematică",
    }));
    expect(api.portfolioDocuments).toHaveBeenLastCalledWith("portfolio-1", expect.objectContaining({ page: 1, pageSize: 20, sectionCode: undefined }));
    expect(JSON.stringify(vi.mocked(api.createPortfolioDocument).mock.calls[0][1])).not.toContain("portfolio_id");
    expect(JSON.stringify(vi.mocked(api.createPortfolioDocument).mock.calls[0][1])).not.toContain("institution_id");
  });

  it.each([
    ["Checklist", "cerință", "Cod cerință *", "R-1", "Cerință *", "Cerință legală", "Secțiune *", "I", "Domeniu sursă *", "portofoliu", "Stare *", "complete", "Ultima verificare *", "2026-09-03", "createPortfolioChecklistItem", { requirement_code: "R-1", requirement_label: "Cerință legală", section_code: "I", source_scope: "portofoliu", status: "complete", last_checked_on: "2026-09-03", checked_by: undefined, document_count: undefined, mandatory: undefined, notes: undefined }],
    ["Opis", "poziție opis", "Secțiune *", "I", "Componentă *", "I.1", "Titlu *", "Opis anual", "Referință document *", "archive://d-1", "Domeniu sursă *", "portofoliu", "Data verificării *", "2026-09-03", "createPortfolioOpisEntry", { section_code: "I", component_code: "I.1", entry_title: "Opis anual", document_reference: "archive://d-1", source_scope: "portofoliu", checked_on: "2026-09-03", checked_by: undefined, chronological_index: undefined, included_in_transfer: undefined, notes: undefined }],
  ] as Array<unknown[]>)("builds the exact %s DTO", async (...args) => {
    const [_tab, addLabel, l1, v1, l2, v2, l3, v3, l4, v4, l5, v5, l6, v6, method, expected] = args as [string, string, string, string, string, string, string, string, string, string, string, string, string, string, keyof EducationApi, unknown];
    const api = apiMock(); renderPanel(api);
    fireEvent.click(screen.getByRole("button", { name: _tab }));
    await openAdd(addLabel);
    await set(l1, v1); await set(l2, v2); await set(l3, v3); await set(l4, v4); await set(l5, v5);
    if (l6) await set(l6, v6);
    fireEvent.click(screen.getByRole("button", { name: "Salvează" }));
    const fn = api[method] as unknown as ReturnType<typeof vi.fn>;
    await waitFor(() => expect(fn).toHaveBeenCalled());
    const call = fn.mock.calls[0] as unknown[];
    expect(call[0]).toBe("portfolio-1"); expect(call[1]).toEqual(expected);
  });

  it("keeps review history read-only even for an institutional manager", async () => {
    const api = apiMock(); renderPanel(api);
    fireEvent.click(screen.getByRole("button", { name: "Revizuiri" }));
    await waitFor(() => expect(screen.queryByRole("button", { name: "Adaugă revizuire" })).not.toBeInTheDocument());
    expect(screen.queryByRole("button", { name: "Editează" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Șterge" })).not.toBeInTheDocument();
  });

  it("builds the exact custody DTO", async () => {
    const api = apiMock(); renderPanel(api); fireEvent.click(screen.getByRole("button", { name: "Custodie" })); await openAdd("eveniment de custodie");
    await set("Tip eveniment *", "handover"); await set("Custode *", "Director"); await set("Rol custode *", "director"); await set("Locație *", "arhivă"); await set("Mod acces *", "read"); await set("Motiv acces *", "control"); await set("Început *", "2026-09-03");
    fireEvent.click(screen.getByRole("button", { name: "Salvează" }));
    await waitFor(() => expect(api.createPortfolioCustodyEvent).toHaveBeenCalledWith("portfolio-1", { event_type: "handover", holder_name: "Director", holder_role: "director", location_label: "arhivă", access_mode: "read", access_reason: "control", started_on: "2026-09-03", ended_on: undefined, sensitive_data_access: undefined, notes: undefined }));
  });

  it("does not expose writes without capability and confirms delete before calling the typed delete operation", async () => {
    const api = apiMock(); renderPanel(api, false);
    await waitFor(() => expect(screen.queryByRole("button", { name: "Adaugă document" })).not.toBeInTheDocument());
    const managing = apiMock();
    vi.mocked(managing.portfolioDocuments).mockResolvedValue({ items: [{
      id: "doc-1", portfolio_id: "portfolio-1", institution_id: "institution-1",
      document_title: "Plan", evidence_type: "plan", section_code: "I", component_code: "I.1",
      source_scope: "portofoliu", authenticity_status: "declared", issued_on: "2026-09-01",
      added_on: "2026-09-02", chronological_index: 1, file_reference: "archive://doc-1",
      sensitive_data: false, notes: "",
    } as never], total: 1, page: 1, pageSize: 20 });
    renderPanel(managing);
    fireEvent.click(await screen.findByRole("button", { name: "Acțiuni înregistrare" }));
    fireEvent.click(await screen.findByRole("button", { name: "Șterge" }));
    expect(managing.deletePortfolioDocument).not.toHaveBeenCalled();
    const confirmation = await screen.findByRole("dialog", { name: "Confirmă ștergerea" });
    fireEvent.click(within(confirmation).getByRole("button", { name: "Șterge" }));
    await waitFor(() => expect(managing.deletePortfolioDocument).toHaveBeenCalledWith("portfolio-1", "doc-1"));
  });

  it("closes the row action overlay before opening record details", async () => {
    const api = apiMock();
    vi.mocked(api.portfolioDocuments).mockResolvedValue({ items: [{
      id: "doc-1", portfolio_id: "portfolio-1", institution_id: "institution-1",
      document_title: "Plan", evidence_type: "plan", section_code: "I", component_code: "I.1",
      source_scope: "portofoliu", authenticity_status: "declared", issued_on: "2026-09-01",
      added_on: "2026-09-02", chronological_index: 1, file_reference: "archive://doc-1",
      sensitive_data: false, notes: "",
    } as never], total: 1, page: 1, pageSize: 20 });
    renderPanel(api);

    fireEvent.click(await screen.findByRole("button", { name: "Acțiuni înregistrare" }));
    fireEvent.click(await screen.findByRole("button", { name: "Detalii" }));

    expect(await screen.findByRole("dialog", { name: "Documente" })).toBeInTheDocument();
    await waitFor(() => expect(screen.queryByRole("menu")).not.toBeInTheDocument());
  });

  it("opens institutional document-version history through the server-paginated contract", async () => {
    const api = apiMock();
    vi.mocked(api.portfolioDocuments).mockResolvedValue({ items: [{ id: "doc-1", document_title: "Plan", description: "Planificare", school_year: "2026-2027", subject_discipline: "Matematică", applicable_class: "IV A", competencies: ["C1"], evidence_type: "plan", section_code: "I", component_code: "I.1", source_scope: "portofoliu", authenticity_status: "declared", issued_on: "2026-09-01", added_on: "2026-09-02", chronological_index: 1, file_reference: "archive://doc-1", sensitive_data: false, notes: "", archive_document_id: "archive-1", archive_version_id: "version-1", archive_version_no: 1, archive_sha256: "abc", portfolio_id: "portfolio-1", institution_id: "institution-1" }], total: 1, page: 1, pageSize: 20 });
    vi.mocked(api.portfolioDocumentVersions).mockResolvedValue({ items: [{ version_no: 1, change_type: "create", changed_by: "director", changed_at: "2026-09-02T10:00:00Z", reason: "inițial", snapshot: {} }], total: 1, page: 1, pageSize: 20 });
    renderPanel(api);
    fireEvent.click(await screen.findByRole("button", { name: "Acțiuni înregistrare" }));
    fireEvent.click(await screen.findByRole("button", { name: "Istoric versiuni" }));
    expect(await screen.findByRole("dialog", { name: "Istoric versiuni — Plan" })).toBeInTheDocument();
    await waitFor(() => expect(api.portfolioDocumentVersions).toHaveBeenCalledWith("portfolio-1", "doc-1", expect.objectContaining({ page: 1, pageSize: 20, sort: "version_no", direction: "desc" })));
    expect(screen.getByText("director")).toBeInTheDocument();
  });

  it.each([
    ["Documente", "Titlu", "portfolioDocuments", "document_title"],
    ["Checklist", "Cerință", "portfolioChecklist", "requirement_label"],
    ["Opis", "Titlu", "portfolioOpis", "entry_title"],
    ["Custodie", "Custode", "portfolioCustody", "holder_name"],
  ] as const)("sends server sort for %s only through its typed %s operation", async (tab, header, method, sort) => {
    const api = apiMock(); renderPanel(api);
    if (tab !== "Documente") fireEvent.click(screen.getByRole("button", { name: tab }));
    const sortButton = await screen.findByRole("button", { name: header });
    fireEvent.click(sortButton);
    await waitFor(() => expect(api[method]).toHaveBeenLastCalledWith("portfolio-1", expect.objectContaining({ page: 1, pageSize: 20, sort, direction: "asc" })));
  });

  it("maps every displayed document header filter to its named document query field", async () => {
    const api = apiMock(); renderPanel(api);
    fireEvent.change(await screen.findByLabelText("Filtru Titlu"), { target: { value: "Plan" } });
    fireEvent.change(await screen.findByLabelText("Filtru Descriere"), { target: { value: "didactică" } });
    fireEvent.change(await screen.findByLabelText("Filtru An școlar"), { target: { value: "2026-2027" } });
    fireEvent.change(await screen.findByLabelText("Filtru Disciplină"), { target: { value: "Matematică" } });
    fireEvent.change(await screen.findByLabelText("Filtru Clasă"), { target: { value: "IV A" } });
    fireEvent.change(await screen.findByLabelText("Filtru Competențe"), { target: { value: "C1" } });
    fireEvent.change(await screen.findByLabelText("Filtru Tip dovadă"), { target: { value: "plan" } });
    fireEvent.change(await screen.findByLabelText("Filtru Versiune arhivă"), { target: { value: "2" } });
    await waitFor(() => expect(api.portfolioDocuments).toHaveBeenLastCalledWith("portfolio-1", expect.objectContaining({ documentTitle: "Plan", description: "didactică", schoolYear: "2026-2027", subjectDiscipline: "Matematică", applicableClass: "IV A", competencies: "C1", evidenceType: "plan", archiveVersionNo: "2" })));
  });

  it("maps immutable transfer filters and exposes read-only details", async () => {
    const api = apiMock();
    vi.mocked(api.portfolioTransferHistory).mockResolvedValue({ items: [{ id: "transfer-1", transfer_code: "TR-1", transfer_type: "handover", status: "received", destination_institution: "Școala B", handover_on: "2026-09-01" } as never], total: 1, page: 1, pageSize: 20 });
    renderPanel(api); fireEvent.click(screen.getByRole("button", { name: "Istoric transferuri" }));
    fireEvent.change(await screen.findByLabelText("Filtru Tip"), { target: { value: "handover" } });
    fireEvent.change(await screen.findByLabelText("Filtru Instituție destinație"), { target: { value: "Școala B" } });
    await waitFor(() => expect(api.portfolioTransferHistory).toHaveBeenLastCalledWith("portfolio-1", expect.objectContaining({ transferType: "handover", destinationInstitution: "Școala B" })));
    fireEvent.click(await screen.findByRole("button", { name: "Acțiuni înregistrare" }));
    fireEvent.click(await screen.findByRole("button", { name: "Detalii" }));
    expect(await screen.findByRole("dialog", { name: "Detalii transfer" })).toHaveTextContent("TR-1");
  });
});
