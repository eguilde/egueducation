import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { PrimeReactProvider } from "@primereact/core/config";
import { primeTheme } from "../../components/ThemeMenu";
import { RegistraturaWorkspace } from "./RegistraturaWorkspace";
import type { RegistraturaApi } from "./api";

function apiForConfirmation(): RegistraturaApi {
  return {
    registries: vi.fn().mockResolvedValue([{ id: 1, nume: "Registru general", isDefault: true }]),
    filters: vi.fn().mockResolvedValue({}),
    documents: vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 50 }),
    parties: vi.fn().mockResolvedValue({ items: [{ id: "party-1", party_type: "physical", display_name: "Ana Pop" }], total: 1, page: 1, pageSize: 50 }),
    admin: vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 50 }),
    deleteParty: vi.fn().mockResolvedValue(undefined),
  } as unknown as RegistraturaApi;
}

const documentItem = (id: string, subject: string) => ({ id, registru_id: 1, registry_number: id.toUpperCase(), subject, document_type: "DOCUMENT", direction: "intrare", status: "INCOMING", correspondent: "", assigned_to: "", registered_at: "2026-08-16" });
const deferred = <T,>() => { let resolve!: (value: T) => void; const promise = new Promise<T>((done) => { resolve = done; }); return { promise, resolve }; };

function apiForRace(overrides: Partial<RegistraturaApi>): RegistraturaApi {
  return {
    registries: vi.fn().mockResolvedValue([{ id: 1, nume: "Registru general", isDefault: true }]),
    filters: vi.fn().mockResolvedValue({ directions: [], statuses: [], document_types: [], confidentialities: [] }),
    documents: vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 50 }),
    document: vi.fn(), versions: vi.fn().mockResolvedValue([]), attachments: vi.fn().mockResolvedValue([]), workflowHistory: vi.fn().mockResolvedValue([]), assignees: vi.fn().mockResolvedValue({ departments: [], users: [] }),
    ...overrides,
  } as unknown as RegistraturaApi;
}

describe("Registratura administration deletion", () => {
  it("does not send a deletion until the accessible PrimeReact confirmation dialog is accepted", async () => {
    const transport = apiForConfirmation();
    render(<PrimeReactProvider {...primeTheme}><RegistraturaWorkspace api={transport} tenantKey="test" canManage /></PrimeReactProvider>);

    await waitFor(() => expect(screen.getByRole("button", { name: /Administrare/ })).toBeEnabled());
    fireEvent.click(screen.getByRole("button", { name: /Administrare/ }));
    await screen.findByText("Ana Pop");
    fireEvent.click(screen.getByRole("button", { name: "Șterge" }));

    expect(screen.getByRole("dialog", { name: "Confirmați ștergerea" })).toBeInTheDocument();
    expect(transport.deleteParty).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Șterge definitiv" }));
    await waitFor(() => expect(transport.deleteParty).toHaveBeenCalledWith("party-1"));
  });
});

describe("Registratura search panel", () => {
  it("starts closed and is toggled by the magnifier button", async () => {
    const transport = apiForConfirmation();
    render(<PrimeReactProvider {...primeTheme}><RegistraturaWorkspace api={transport} tenantKey="search-panel" /></PrimeReactProvider>);

    const openSearch = await screen.findByRole("button", { name: "Deschide căutarea" });
    expect(openSearch).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByLabelText("Căutare documente")).not.toBeInTheDocument();

    fireEvent.click(openSearch);
    expect(screen.getByLabelText("Căutare documente")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Închide căutarea" })).toHaveAttribute("aria-expanded", "true");
  });

  it("sends the Costești filter fields to the server", async () => {
    const transport = apiForRace({ documents: vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 20 }) });
    render(<PrimeReactProvider {...primeTheme}><RegistraturaWorkspace api={transport} tenantKey="search-contract" /></PrimeReactProvider>);
    await waitFor(() => expect(transport.documents).toHaveBeenCalledTimes(1));
    fireEvent.click(screen.getByRole("button", { name: "Deschide căutarea" }));
    fireEvent.change(screen.getByLabelText("Nr. Extern"), { target: { value: "ABC-123" } });
    fireEvent.change(screen.getByLabelText("Emitent"), { target: { value: "Inspectorat" } });
    fireEvent.change(screen.getByLabelText("Destinatar"), { target: { value: "Școala" } });
    fireEvent.change(screen.getByLabelText("Data intrare de la"), { target: { value: "2026-08-01" } });
    fireEvent.click(screen.getByRole("button", { name: "Caută documente" }));
    await waitFor(() => expect(transport.documents).toHaveBeenLastCalledWith(expect.objectContaining({ page: 1, pageSize: 20, filters: expect.objectContaining({ external_number: "ABC-123", correspondent: "Inspectorat", assigned_to: "Școala", entry_at_from: "2026-08-01" }) })));
  });
});

describe("Registratura Costești table parity", () => {
  it("uses server pagination/sorting, row expansion and five explicit actions", async () => {
    const item = { ...documentItem("reg-20", "Document test"), external_number: "EXT-9", external_number_date: "2026-08-10", activity: "Control", department_names: ["Secretariat"] };
    const transport = apiForRace({
      documents: vi.fn().mockResolvedValue({ items: [item], total: 45, page: 1, pageSize: 20 }),
      document: vi.fn().mockResolvedValue(item),
      print: vi.fn().mockResolvedValue(new Blob(["pdf"], { type: "application/pdf" })),
    });
    render(<PrimeReactProvider {...primeTheme}><RegistraturaWorkspace api={transport} tenantKey="costesti-table" canManage canManageWorkflow /></PrimeReactProvider>);
    await screen.findByText("Document test");
    const documentRow = screen.getByRole("row", { name: /REG-20/ });
    expect(within(documentRow).getByText("Intrare")).toBeInTheDocument();
    expect(within(documentRow).getByText("Înregistrat")).toBeInTheDocument();
    expect(within(documentRow).getByText("16.08.2026")).toBeInTheDocument();
    expect(screen.queryByText("2026-08-16")).not.toBeInTheDocument();
    expect(transport.documents).toHaveBeenCalledWith(expect.objectContaining({ page: 1, pageSize: 20 }));
    for (const name of ["Istoric REG-20", "Editează REG-20", "Anulează REG-20", "PDF REG-20", "Flux REG-20"]) expect(screen.getByRole("button", { name })).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Extinde REG-20" }));
    await screen.findByText(/Secretariat/);
    expect(screen.getByText(/EXT-9/)).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Sortează după Nr. Doc" }));
    await waitFor(() => expect(transport.documents).toHaveBeenLastCalledWith(expect.objectContaining({ sort: "registry_number", direction: "asc" })));
    fireEvent.click(screen.getByRole("button", { name: "Pagina 2" }));
    await waitFor(() => expect(transport.documents).toHaveBeenLastCalledWith(expect.objectContaining({ page: 2, pageSize: 20 })));
  });
});

describe("Registratura request ordering", () => {
  it("keeps the newest document list when request A resolves after B", async () => {
    const first = deferred<{ items: ReturnType<typeof documentItem>[]; total: number; page: number; pageSize: number }>();
    const second = deferred<{ items: ReturnType<typeof documentItem>[]; total: number; page: number; pageSize: number }>();
    const transport = apiForRace({ documents: vi.fn().mockReturnValueOnce(first.promise).mockReturnValueOnce(second.promise) });
    render(<PrimeReactProvider {...primeTheme}><RegistraturaWorkspace api={transport} tenantKey="race-list" canManage /></PrimeReactProvider>);
    await waitFor(() => expect(transport.documents).toHaveBeenCalledTimes(1));
    fireEvent.click(screen.getByRole("button", { name: "Deschide căutarea" }));
    fireEvent.click(screen.getByRole("button", { name: "Caută documente" }));
    await waitFor(() => expect(transport.documents).toHaveBeenCalledTimes(2));
    second.resolve({ items: [documentItem("new", "Rezultatul B")], total: 1, page: 1, pageSize: 50 });
    await screen.findByText("Rezultatul B");
    first.resolve({ items: [documentItem("old", "Rezultatul A")], total: 1, page: 1, pageSize: 50 });
    await waitFor(() => expect(screen.queryByText("Rezultatul A")).not.toBeInTheDocument());
  });

  it("keeps the newest document detail when request A resolves after B", async () => {
    const first = deferred<ReturnType<typeof documentItem>>();
    const second = deferred<ReturnType<typeof documentItem>>();
    const transport = apiForRace({ documents: vi.fn().mockResolvedValue({ items: [documentItem("a", "A inițial"), documentItem("b", "B inițial")], total: 2, page: 1, pageSize: 50 }), document: vi.fn().mockImplementation((id: string) => id === "a" ? first.promise : second.promise) });
    render(<PrimeReactProvider {...primeTheme}><RegistraturaWorkspace api={transport} tenantKey="race-detail" canManage /></PrimeReactProvider>);
    await screen.findByText("A inițial");
    fireEvent.click(screen.getByRole("button", { name: "Istoric A" }));
    fireEvent.click(screen.getByRole("button", { name: "Istoric B" }));
    second.resolve(documentItem("b", "Detaliu B"));
    await screen.findByText("Detaliu B");
    first.resolve(documentItem("a", "Detaliu A"));
    await waitFor(() => expect(screen.queryByText("Detaliu A")).not.toBeInTheDocument());
  });
});

describe("Registratura document links RBAC", () => {
  const linkDocument = documentItem("linked", "Document legat");
  const linkApi = (overrides: Partial<RegistraturaApi> = {}) => apiForRace({
    documents: vi.fn().mockResolvedValue({ items: [linkDocument], total: 1, page: 1, pageSize: 50 }),
    document: vi.fn().mockResolvedValue(linkDocument),
    links: vi.fn().mockResolvedValue([]),
    createLink: vi.fn().mockResolvedValue({ link_id: "link-1" }),
    deleteLink: vi.fn().mockResolvedValue(undefined),
    ...overrides,
  });

  it("does not expose document-link controls without the separate links.read permission", async () => {
    const transport = linkApi();
    render(<PrimeReactProvider {...primeTheme}><RegistraturaWorkspace api={transport} tenantKey="links-none" canManage canReadLinks={false} canManageLinks={false} /></PrimeReactProvider>);
    await screen.findByText("Document legat");
    fireEvent.click(screen.getByRole("button", { name: "Istoric LINKED" }));
    await screen.findByRole("dialog", { name: /Istoricul documentului LINKED/ });
    expect(screen.queryByText("Legături documente")).not.toBeInTheDocument();
    expect(transport.links).not.toHaveBeenCalled();
  });

  it("allows reading links but not creating them without links.manage", async () => {
    const transport = linkApi();
    render(<PrimeReactProvider {...primeTheme}><RegistraturaWorkspace api={transport} tenantKey="links-read" canManage canReadLinks canManageLinks={false} /></PrimeReactProvider>);
    await screen.findByText("Document legat");
    fireEvent.click(screen.getByRole("button", { name: "Istoric LINKED" }));
    await screen.findByText("Legături documente");
    fireEvent.change(screen.getByLabelText("Modul sursă legătură"), { target: { value: "education" } });
    fireEvent.change(screen.getByLabelText("ID înregistrare sursă legătură"), { target: { value: "source-1" } });
    expect(screen.queryByRole("button", { name: "Adaugă legătură" })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Încarcă legături" }));
    await waitFor(() => expect(transport.links).toHaveBeenCalledWith("education", "source-1"));
    expect(transport.createLink).not.toHaveBeenCalled();
  });
});
