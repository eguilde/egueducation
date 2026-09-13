import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { PrimeReactProvider } from "@primereact/core/config";
import { describe, expect, it, vi } from "vitest";
import { PortfolioLifecycleHistory } from "./PortfolioLifecycleHistory";
import type { EducationApi, PortfolioLifecyclePage, PortfolioLifecycleResult } from "./types";

const operation: PortfolioLifecycleResult["operation"] = { id: "op-1", portfolio_id: "p1", type: "cessation_retention", status: "blocked", requested_at: "2026-09-12T12:00:00Z", total_versions: 2, completed_versions: 0, blocked_versions: 2 };
const page = (): PortfolioLifecyclePage => ({ items: [operation], total: 21, page: 1, pageSize: 20 });
const api = () => ({ portfolioLifecycleOperations: vi.fn().mockResolvedValue(page()), portfolioLifecycleOperation: vi.fn().mockResolvedValue({ operation, portfolio: { id: "p1" } }), retryPortfolioLifecycleOperation: vi.fn() }) as unknown as EducationApi;
const view = (client: EducationApi, portfolioID = "p1", canRetry = false) => <PrimeReactProvider><PortfolioLifecycleHistory api={client} portfolioID={portfolioID} canRetry={canRetry} onClose={vi.fn()} /></PrimeReactProvider>;

describe("PortfolioLifecycleHistory", () => {
  it("loads persisted history on each opening and sends sorting, date filters and pagination to the server", async () => {
    const client = api();
    const mounted = render(view(client));
    await screen.findByText("21 operații");
    expect(client.portfolioLifecycleOperations).toHaveBeenLastCalledWith("p1", { page: 1, pageSize: 20, sort: "requested_at", direction: "desc" });
    fireEvent.click(screen.getByRole("button", { name: "Următor" }));
    await waitFor(() => expect(client.portfolioLifecycleOperations).toHaveBeenLastCalledWith("p1", expect.objectContaining({ page: 2 })));
    fireEvent.click(screen.getByRole("button", { name: "Sortează starea operației" }));
    await waitFor(() => expect(client.portfolioLifecycleOperations).toHaveBeenLastCalledWith("p1", expect.objectContaining({ page: 1, sort: "status", direction: "asc" })));
    fireEvent.change(screen.getByLabelText("Filtru data solicitării UTC"), { target: { value: "2026-09-12" } });
    await waitFor(() => expect(client.portfolioLifecycleOperations).toHaveBeenLastCalledWith("p1", expect.objectContaining({ "filter.requested_at": "2026-09-12" })));
    mounted.unmount();
    render(view(client));
    await waitFor(() => expect(client.portfolioLifecycleOperations).toHaveBeenLastCalledWith("p1", { page: 1, pageSize: 20, sort: "requested_at", direction: "desc" }));
  });

  it("lets an owner inspect status but offers retry only with the management permission", async () => {
    const client = api();
    const mounted = render(view(client));
    fireEvent.click(await screen.findByRole("button", { name: "Stare operație op-1" }));
    await screen.findByText(/Blocată — necesită verificare/);
    expect(client.portfolioLifecycleOperation).toHaveBeenCalledWith("p1", "op-1");
    expect(screen.queryByRole("button", { name: "Reia verificarea" })).not.toBeInTheDocument();
    mounted.rerender(view(client, "p1", true));
    expect(screen.getByRole("button", { name: "Reia verificarea" })).toBeDisabled();
    expect(client.retryPortfolioLifecycleOperation).not.toHaveBeenCalled();
  });

  it("refreshes the server list when the selected operation status is read", async () => {
    const client = api();
    render(view(client));
    fireEvent.click(await screen.findByRole("button", { name: "Stare operație op-1" }));
    await waitFor(() => expect(client.portfolioLifecycleOperations).toHaveBeenCalledTimes(2));
  });

  it("discards a previous portfolio response after switching", async () => {
    const client = api();
    let resolve!: (result: PortfolioLifecyclePage) => void;
    vi.mocked(client.portfolioLifecycleOperations).mockImplementationOnce(() => new Promise(done => { resolve = done; })).mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 20 });
    const mounted = render(view(client));
    mounted.rerender(view(client, "p2"));
    await screen.findByText("0 operații");
    await act(async () => resolve(page()));
    expect(screen.queryByRole("button", { name: "Stare operație op-1" })).not.toBeInTheDocument();
    expect(screen.getByText("0 operații")).toBeInTheDocument();
  });

  it("reports a denied history without retaining rows from a previous success", async () => {
    const client = api();
    render(view(client));
    await screen.findByText("21 operații");
    vi.mocked(client.portfolioLifecycleOperations).mockRejectedValue(new Error("forbidden"));
    fireEvent.click(screen.getByRole("button", { name: "Reîncarcă istoricul" }));
    await screen.findByText("Istoricul operațiilor nu a putut fi încărcat.");
    expect(screen.queryByRole("button", { name: "Stare operație op-1" })).not.toBeInTheDocument();
  });
});
