import { act, fireEvent, render, screen } from "@testing-library/react";
import { PrimeReactProvider } from "@primereact/core/config";
import { afterEach, describe, expect, it, vi } from "vitest";
import { PortfolioLifecycleStatus } from "./PortfolioLifecycleStatus";
import type { PortfolioLifecycleResult } from "./types";

const result = (status: PortfolioLifecycleResult["operation"]["status"]): PortfolioLifecycleResult => ({
  operation: { id: "operation-1", portfolio_id: "portfolio-1", type: "cessation_retention", status, requested_at: "2026-09-12T12:00:00Z", total_versions: 2, completed_versions: status === "completed" ? 2 : 0, blocked_versions: status === "blocked" ? 2 : 0 },
  portfolio: { id: "portfolio-1" } as PortfolioLifecycleResult["portfolio"],
  transitions: [],
});
afterEach(() => vi.useRealTimers());
describe("PortfolioLifecycleStatus", () => {
  it("does not claim accepted work is complete and polls until verified", async () => {
    vi.useFakeTimers();
    const api = { portfolioLifecycleOperation: vi.fn().mockResolvedValueOnce(result("processing")).mockResolvedValue(result("completed")), retryPortfolioLifecycleOperation: vi.fn(), portfolioRetentionDispositions: vi.fn(), submitPortfolioRetentionDisposition: vi.fn(), decidePortfolioRetentionDisposition: vi.fn() };
    render(<PrimeReactProvider><PortfolioLifecycleStatus api={api} initial={result("pending")} canRetry={false} /></PrimeReactProvider>);
    await act(async () => {});
    expect(screen.getByText(/nu este încă finalizată/)).toBeInTheDocument();
    await act(async () => vi.advanceTimersByTime(3000));
    expect(screen.getByText(/Finalizată. Versiuni verificate: 2 \/ 2/)).toBeInTheDocument();
    await act(async () => vi.advanceTimersByTime(9000));
    expect(api.portfolioLifecycleOperation).toHaveBeenCalledTimes(2);
  });
  it("requires a reason and a management permission to retry", async () => {
    const api = { portfolioLifecycleOperation: vi.fn().mockResolvedValue(result("blocked")), retryPortfolioLifecycleOperation: vi.fn().mockResolvedValue(result("pending")), portfolioRetentionDispositions: vi.fn(), submitPortfolioRetentionDisposition: vi.fn(), decidePortfolioRetentionDisposition: vi.fn() };
    const view = render(<PrimeReactProvider><PortfolioLifecycleStatus api={api} initial={result("blocked")} canRetry={false} /></PrimeReactProvider>);
    await act(async () => {});
    expect(screen.queryByRole("button", { name: "Reia verificarea" })).not.toBeInTheDocument();
    view.rerender(<PrimeReactProvider><PortfolioLifecycleStatus api={api} initial={result("blocked")} canRetry /></PrimeReactProvider>);
    expect(screen.getByRole("button", { name: "Reia verificarea" })).toBeDisabled();
    fireEvent.change(screen.getByLabelText("Motivul reluării *"), { target: { value: "  Conexiune restabilită  " } });
    api.portfolioLifecycleOperation.mockResolvedValue(result("completed"));
    await act(async () => fireEvent.click(screen.getByRole("button", { name: "Reia verificarea" })));
    expect(api.retryPortfolioLifecycleOperation).toHaveBeenCalledWith("portfolio-1", "operation-1", { reason: "Conexiune restabilită" });
  });
  it("stops automatic polling on a read failure and permits explicit refresh", async () => {
    const api = { portfolioLifecycleOperation: vi.fn().mockRejectedValueOnce(new Error("forbidden")).mockResolvedValue(result("completed")), retryPortfolioLifecycleOperation: vi.fn(), portfolioRetentionDispositions: vi.fn(), submitPortfolioRetentionDisposition: vi.fn(), decidePortfolioRetentionDisposition: vi.fn() };
    render(<PrimeReactProvider><PortfolioLifecycleStatus api={api} initial={result("pending")} canRetry={false} /></PrimeReactProvider>);
    await act(async () => {});
    expect(screen.getByText(/Starea nu poate fi verificată/)).toBeInTheDocument();
    await act(async () => fireEvent.click(screen.getByRole("button", { name: "Reîncarcă starea" })));
    expect(screen.getByText(/Finalizată. Versiuni verificate: 2 \/ 2/)).toBeInTheDocument();
  });
  it("discards an old request after switching operations", async () => {
    let resolve!: (value: PortfolioLifecycleResult) => void;
    const api = { portfolioLifecycleOperation: vi.fn().mockImplementationOnce(() => new Promise<PortfolioLifecycleResult>(done => { resolve = done; })).mockResolvedValue(result("blocked")), retryPortfolioLifecycleOperation: vi.fn(), portfolioRetentionDispositions: vi.fn(), submitPortfolioRetentionDisposition: vi.fn(), decidePortfolioRetentionDisposition: vi.fn() };
    const view = render(<PrimeReactProvider><PortfolioLifecycleStatus api={api} initial={result("pending")} canRetry={false} /></PrimeReactProvider>);
    const next = result("blocked"); next.operation.id = "operation-2";
    view.rerender(<PrimeReactProvider><PortfolioLifecycleStatus api={api} initial={next} canRetry={false} /></PrimeReactProvider>);
    await act(async () => {});
    await act(async () => resolve(result("completed")));
    expect(screen.getByText(/Blocată — necesită verificare/)).toBeInTheDocument();
    expect(screen.queryByText(/Finalizată/)).not.toBeInTheDocument();
  });
});
