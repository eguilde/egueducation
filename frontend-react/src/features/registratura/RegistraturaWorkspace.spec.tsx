import { fireEvent, render, screen, waitFor } from "@testing-library/react";
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

describe("Registratura request ordering", () => {
  it("keeps the newest document list when request A resolves after B", async () => {
    const first = deferred<{ items: ReturnType<typeof documentItem>[]; total: number; page: number; pageSize: number }>();
    const second = deferred<{ items: ReturnType<typeof documentItem>[]; total: number; page: number; pageSize: number }>();
    const transport = apiForRace({ documents: vi.fn().mockReturnValueOnce(first.promise).mockReturnValueOnce(second.promise) });
    render(<PrimeReactProvider {...primeTheme}><RegistraturaWorkspace api={transport} tenantKey="race-list" canManage /></PrimeReactProvider>);
    await waitFor(() => expect(transport.documents).toHaveBeenCalledTimes(1));
    fireEvent.click(screen.getByRole("button", { name: /Aplică filtre/ }));
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
    fireEvent.click(screen.getByRole("button", { name: "Deschide A" }));
    fireEvent.click(screen.getByRole("button", { name: "Deschide B" }));
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
    fireEvent.click(screen.getByRole("button", { name: "Deschide LINKED" }));
    await screen.findByRole("dialog", { name: /Document LINKED/ });
    expect(screen.queryByText("Legături documente")).not.toBeInTheDocument();
    expect(transport.links).not.toHaveBeenCalled();
  });

  it("allows reading links but not creating them without links.manage", async () => {
    const transport = linkApi();
    render(<PrimeReactProvider {...primeTheme}><RegistraturaWorkspace api={transport} tenantKey="links-read" canManage canReadLinks canManageLinks={false} /></PrimeReactProvider>);
    await screen.findByText("Document legat");
    fireEvent.click(screen.getByRole("button", { name: "Deschide LINKED" }));
    await screen.findByText("Legături documente");
    fireEvent.change(screen.getByLabelText("Modul sursă legătură"), { target: { value: "education" } });
    fireEvent.change(screen.getByLabelText("ID înregistrare sursă legătură"), { target: { value: "source-1" } });
    expect(screen.queryByRole("button", { name: "Adaugă legătură" })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Încarcă legături" }));
    await waitFor(() => expect(transport.links).toHaveBeenCalledWith("education", "source-1"));
    expect(transport.createLink).not.toHaveBeenCalled();
  });
});
