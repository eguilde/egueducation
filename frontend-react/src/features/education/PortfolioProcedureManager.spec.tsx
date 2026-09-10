import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { PrimeReactProvider } from "@primereact/core/config";
import { describe, expect, it, vi } from "vitest";
import { PortfolioProcedureManager, type PortfolioProcedureApi } from "./PortfolioProcedureManager";

const procedure = { id: "p-1", code: "PP-2026", title: "Procedura portofoliului", status: "draft", version: 1, updated_at: "2026-09-09T10:00:00Z" };
const rule = { id: "r-1", legal_section_code: "I.1", label: "Proiectare", label_en: "Planning", source_catalog_version: "2026.1", active: true, required: true, sort_order: 1 };
function api(rows = [procedure]): PortfolioProcedureApi { return { list: vi.fn().mockResolvedValue({ items: rows, total: rows.length, page: 1, pageSize: 20 }), detail: vi.fn().mockResolvedValue({ ...procedure, updated_at: "2026-09-10T10:00:00Z" }), create: vi.fn().mockResolvedValue(procedure), update: vi.fn().mockResolvedValue(procedure), rules: vi.fn().mockResolvedValue({ items: [rule], total: 1, page: 1, pageSize: 20 }), replaceRules: vi.fn().mockResolvedValue(undefined), transition: vi.fn().mockResolvedValue({ ...procedure, status: "approved" }) }; }
function view(client: PortfolioProcedureApi) { return render(<PrimeReactProvider><PortfolioProcedureManager api={client} /></PrimeReactProvider>); }
describe("PortfolioProcedureManager", () => {
  it("exposes its page title as a semantic heading", async () => { const client=api();view(client);expect(await screen.findByRole("heading",{name:"Proceduri instituționale pentru portofolii"})).toBeVisible(); });
  it("uses a server query and retains Add in the action header when empty", async () => { const client = api([]); view(client); const add = await screen.findByRole("button", { name: "Adaugă procedură" }); await waitFor(() => expect(client.list).toHaveBeenCalledWith(expect.objectContaining({ page: 1, pageSize: 20, filters: {} }))); fireEvent.click(add); expect(await screen.findByRole("dialog")).toHaveTextContent("Adaugă procedură"); });
  it("sends a debounced header filter to the server", async () => { const client = api(); view(client); await screen.findByText(procedure.title); fireEvent.change(screen.getByLabelText("Filtru Titlu"), { target: { value: "portofoliu" } }); await waitFor(() => expect(client.list).toHaveBeenLastCalledWith(expect.objectContaining({ filters: expect.objectContaining({ title: "portofoliu" }) })), { timeout: 1200 }); });
  it("persists a new procedure through the explicit contract", async () => { const client = api([]); view(client); fireEvent.click(await screen.findByRole("button", { name: "Adaugă procedură" })); const dialog = await screen.findByRole("dialog"); fireEvent.change(dialog.querySelectorAll("input")[0], { target: { value: "PP-2026" } }); fireEvent.change(dialog.querySelectorAll("input")[1], { target: { value: "Procedura portofoliului" } }); fireEvent.click(screen.getByRole("button", { name: "Salvează" })); await waitFor(() => expect(client.create).toHaveBeenCalledWith(expect.objectContaining({ code: "PP-2026", title: "Procedura portofoliului" }))); });
  it("loads every server page before atomically replacing editable draft rules", async () => {
    const client = api();
    client.rules = vi.fn().mockImplementation((_id, query) => Promise.resolve(query.page === 1 ? { items: [rule], total: 2, page: 1, pageSize: 100 } : { items: [{ ...rule, id: "r-2", legal_section_code: "I.2", sort_order: 2 }], total: 2, page: 2, pageSize: 100 }));
    view(client);
    fireEvent.click(await screen.findByRole("button", { name: `Detalii și reguli pentru ${procedure.title}` }));
    fireEvent.click(await screen.findByRole("button", { name: "Editează reguli" }));
    await waitFor(() => expect(client.rules).toHaveBeenCalledWith(procedure.id, expect.objectContaining({ page: 2, pageSize: 100, filters: {} })));
    fireEvent.click(screen.getByRole("button", { name: "Elimină regula I.2" }));
    fireEvent.click(screen.getByRole("button", { name: "Salvează reguli" }));
    await waitFor(() => expect(client.replaceRules).toHaveBeenCalledWith(procedure.id, expect.objectContaining({ expected_updated_at: procedure.updated_at, rules: [expect.objectContaining({ legal_section_code: "I.1", source_catalog_version: "2026.1" })] })));
  });
  it("rejects duplicate section codes before replacing rules and exposes labelled rule checkboxes", async () => {
    const client = api();
    view(client);
    fireEvent.click(await screen.findByRole("button", { name: `Detalii și reguli pentru ${procedure.title}` }));
    fireEvent.click(await screen.findByRole("button", { name: "Editează reguli" }));
    await screen.findByText("Toate paginile regulilor au fost încărcate; salvarea înlocuiește atomic setul complet al versiunii draft.");
    fireEvent.click(screen.getByRole("button", { name: "Adaugă regulă" }));
    const ruleDialog = await screen.findByRole("dialog", { name: "Adaugă regulă" });
    const rule = within(ruleDialog);
    expect(rule.getByRole("checkbox", { name: "Obligatorie" })).toBeChecked();
    expect(rule.getByRole("checkbox", { name: "Activă" })).toBeChecked();
    fireEvent.change(rule.getByLabelText("Cod secțiune"), { target: { value: " i.1 " } });
    fireEvent.change(rule.getByLabelText("Ordine"), { target: { value: "2" } });
    fireEvent.change(rule.getByLabelText("Denumire română"), { target: { value: "Duplicat" } });
    fireEvent.change(rule.getByLabelText("Versiunea catalogului"), { target: { value: "2026.1" } });
    fireEvent.click(rule.getByRole("button", { name: "Aplică" }));
    await waitFor(() => expect(screen.queryByRole("dialog", { name: "Adaugă regulă" })).not.toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Salvează reguli" }));
    expect(await screen.findByText("Codurile de secțiune trebuie să fie unice în procedură.")).toBeVisible();
    expect(screen.getByRole("dialog", { name: `Reguli procedură: ${procedure.title}` })).toBeVisible();
    expect(client.replaceRules).not.toHaveBeenCalled();
  });
  it("keeps the rules editor open and reports a stale replace conflict", async () => {
    const client = api();
    client.replaceRules = vi.fn().mockRejectedValue(new Error("stale update"));
    view(client);
    fireEvent.click(await screen.findByRole("button", { name: `Detalii și reguli pentru ${procedure.title}` }));
    fireEvent.click(await screen.findByRole("button", { name: "Editează reguli" }));
    await screen.findByText("Toate paginile regulilor au fost încărcate; salvarea înlocuiește atomic setul complet al versiunii draft.");
    fireEvent.click(screen.getByRole("button", { name: "Salvează reguli" }));
    expect(await screen.findByText("Salvarea atomică a regulilor nu a reușit. Reîncărcați procedura și verificați versiunea curentă.")).toBeVisible();
    expect(screen.getByRole("dialog", { name: `Reguli procedură: ${procedure.title}` })).toBeVisible();
    expect(client.replaceRules).toHaveBeenCalledWith(procedure.id, expect.objectContaining({ expected_updated_at: procedure.updated_at }));
  });
});
