import { render, screen, fireEvent, waitFor, within, act } from "@testing-library/react";
import { PrimeReactProvider } from "@primereact/core/config";
import { describe, expect, it, vi } from "vitest";
import { primeTheme } from "../../components/ThemeMenu";
import { ArchiveRetentionWorkspace } from "./ArchiveRetentionWorkspace";

const rule = { id: "rule-1", taxonomy_node_id: "series-1", source_id: "source-1", effective_from: "2026-01-01", effective_to: null, anchor_kind: "intake_received_at", duration_model: "minimum_days", minimum_retention_days: 365, status: "proposed", expected_version: 1, proposed_by_subject: "author" };
const api = () => ({ list: vi.fn().mockResolvedValue({ items: [rule], total: 1, page: 1, pageSize: 20 }), taxonomy: vi.fn().mockResolvedValue([{ id: "series-1", label: "Cataloage școlare" }]), sources: vi.fn().mockResolvedValue([{ id: "source-1", label: "Legea educației · Minister" }]), propose: vi.fn().mockResolvedValue(rule), approve: vi.fn().mockResolvedValue(rule), retire: vi.fn().mockResolvedValue(rule) });
const show = (transport = api(), permissions = { manage: true, approve: true }, actorSubject?: string) => { render(<PrimeReactProvider {...primeTheme}><ArchiveRetentionWorkspace api={transport as any} canManage={permissions.manage} canApprove={permissions.approve} actorSubject={actorSubject}/></PrimeReactProvider>); return transport; };

describe("ArchiveRetentionWorkspace", () => {
  it("does not replace the current context with an older list response", async () => {
    const oldTransport = api();
    let resolveOld!: (value: { items: typeof rule[]; total: number; page: number; pageSize: number }) => void;
    oldTransport.list.mockImplementation(() => new Promise(resolve => { resolveOld = resolve; }));
    const nextTransport = api();
    nextTransport.list.mockResolvedValue({ items: [{ ...rule, id: "next-rule", effective_from: "2027-01-01" }], total: 1, page: 1, pageSize: 20 });
    const view = render(<PrimeReactProvider {...primeTheme}><ArchiveRetentionWorkspace api={oldTransport as any} canManage={false} canApprove={false} /></PrimeReactProvider>);
    await waitFor(() => expect(oldTransport.list).toHaveBeenCalled());
    view.rerender(<PrimeReactProvider {...primeTheme}><ArchiveRetentionWorkspace api={nextTransport as any} canManage={false} canApprove={false} /></PrimeReactProvider>);
    expect(await screen.findByText("2027-01-01")).toBeInTheDocument();
    await act(async () => { resolveOld({ items: [rule], total: 1, page: 1, pageSize: 20 }); });
    expect(screen.getByText("2027-01-01")).toBeInTheDocument();
    expect(screen.queryByText("2026-01-01")).not.toBeInTheDocument();
  });
  it("gates mutation actions and hides self approval", async () => { const transport = show(undefined, { manage: false, approve: true }, "author"); await waitFor(() => expect(transport.list).toHaveBeenCalled()); expect(screen.queryByRole("button", { name: "Adaugă regulă" })).not.toBeInTheDocument(); expect(screen.queryByRole("button", { name: "Aprobă" })).not.toBeInTheDocument(); });
  it("opens a reason dialog before retiring and sends the entered reason", async () => { const transport = api(); transport.list.mockResolvedValue({ items: [{ ...rule, status: "active" }], total: 1, page: 1, pageSize: 20 }); show(transport); await waitFor(() => expect(screen.getByRole("button", { name: "Retrage" })).toBeInTheDocument()); fireEvent.click(screen.getByRole("button", { name: "Retrage" })); fireEvent.change(screen.getByLabelText("Motiv retragere"), { target: { value: "Sursă înlocuită" } }); fireEvent.click(screen.getByRole("button", { name: "Confirmă retragerea" })); await waitFor(() => expect(transport.retire).toHaveBeenCalledWith("rule-1", 1, "Sursă înlocuită")); });
  it("sends the exact initial-anchor proposal DTO selected through human labels", async () => {
    const transport = show();
    fireEvent.click(await screen.findByRole("button", { name: "Adaugă regulă" }));
    const dialog = await screen.findByRole("dialog", { name: "Regulă de retenție arhivă" });
    const choose = async (label: string, option: string) => {
      fireEvent.click(within(dialog).getByRole("combobox", { name: label }));
      fireEvent.click(await screen.findByRole("option", { name: option }));
    };
    expect(within(dialog).getByRole("button", { name: "Propune regula" })).toBeDisabled();
    await choose("Serie arhivistică", "Cataloage școlare");
    await choose("Sursă documentată", "Legea educației · Minister");
    fireEvent.change(within(dialog).getByLabelText("Dată intrare în vigoare"), { target: { value: "2026-09-12" } });
    fireEvent.change(within(dialog).getByLabelText("Zile retenție"), { target: { value: "365" } });
    fireEvent.click(within(dialog).getByRole("button", { name: "Propune regula" }));
    await waitFor(() => expect(transport.propose).toHaveBeenCalledWith({
      taxonomy_node_id: "series-1", source_id: "source-1", effective_from: "2026-09-12", effective_to: undefined,
      anchor_kind: "intake_received_at", duration_model: "minimum_days", minimum_retention_days: 365,
    }));
  });

  it("keeps add and filters in the table header and sends server sort and page changes", async () => {
    const transport = api();
    transport.list.mockResolvedValue({ items: [rule], total: 41, page: 1, pageSize: 20 });
    show(transport);
    const add = await screen.findByRole("button", { name: "Adaugă regulă" });
    expect(add.closest("th")).toHaveTextContent("Acțiuni");
    const filter = screen.getByRole("combobox", { name: "Filtru stare" });
    expect(filter.closest("thead")).not.toBeNull();
    expect(screen.getByRole("combobox", { name: "Filtru serie arhivistică" }).closest("thead")).not.toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Stare" }));
    await waitFor(() => expect(transport.list).toHaveBeenLastCalledWith(expect.objectContaining({ sort: "status", direction: "asc", page: 1 })));
    fireEvent.click(await screen.findByRole("button", { name: "Următor" }));
    await waitFor(() => expect(transport.list).toHaveBeenLastCalledWith(expect.objectContaining({ sort: "status", direction: "asc", page: 2 })));
    fireEvent.click(await screen.findByRole("combobox", { name: "Filtru stare" }));
    fireEvent.click(await screen.findByRole("option", { name: "Activă" }));
    await waitFor(() => expect(transport.list).toHaveBeenLastCalledWith(expect.objectContaining({ status: "active", page: 1 })));
  });
});
