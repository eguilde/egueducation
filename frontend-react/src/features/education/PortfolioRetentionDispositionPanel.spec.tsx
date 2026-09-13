import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { PrimeReactProvider } from "@primereact/core/config";
import { describe, expect, it, vi } from "vitest";
import { PortfolioRetentionDispositionPanel } from "./PortfolioRetentionDispositionPanel";
import type { EducationApi, PortfolioLifecycleResult } from "./types";

const transition: PortfolioLifecycleResult["transitions"][number] = {
  id: "11111111-1111-4111-8111-111111111111",
  status: "blocked",
  last_error: "portfolio_retention_expired_review_required",
  required_retention_until: "2026-09-01T00:00:00Z",
};

const api = () => ({
  portfolioRetentionDispositions: vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 10 }),
  submitPortfolioRetentionDisposition: vi.fn().mockResolvedValue({ id: "request-1", status: "submitted" }),
  decidePortfolioRetentionDisposition: vi.fn().mockResolvedValue({ id: "request-1", status: "approved", decision: "approved", outcome: "retained" }),
}) satisfies Pick<EducationApi, "portfolioRetentionDispositions" | "submitPortfolioRetentionDisposition" | "decidePortfolioRetentionDisposition">;

describe("PortfolioRetentionDispositionPanel", () => {
  it("submits owner evidence for the exact server-exposed expired transition", async () => {
    const client = api();
    render(<PrimeReactProvider><PortfolioRetentionDispositionPanel api={client} portfolioID="portfolio-1" operationID="operation-1" transitions={[transition]} canSubmit /></PrimeReactProvider>);
    fireEvent.click(await screen.findByRole("button", { name: "Solicită analizarea expirării" }));
    fireEvent.change(screen.getByLabelText("Justificare și dovezi *"), { target: { value: "  Termen împlinit și verificat  " } });
    fireEvent.change(screen.getByLabelText("Referință document"), { target: { value: "  PV-2026-09  " } });
    fireEvent.click(screen.getByRole("button", { name: "Trimite pentru aprobare" }));
    await waitFor(() => expect(client.submitPortfolioRetentionDisposition).toHaveBeenCalledWith("portfolio-1", "operation-1", transition.id, {
      evidence: { statement: "Termen împlinit și verificat", reference: "PV-2026-09" },
    }));
    expect(await screen.findByText(/decizie independentă/)).toBeInTheDocument();
  });

  it("does not offer release review for a non-expired transition", async () => {
    render(<PrimeReactProvider><PortfolioRetentionDispositionPanel api={api()} portfolioID="portfolio-1" operationID="operation-1" transitions={[{ ...transition, status: "completed", last_error: "" }]} canSubmit /></PrimeReactProvider>);
    await screen.findByText("0 solicitări");
    expect(screen.queryByRole("button", { name: "Solicită analizarea expirării" })).not.toBeInTheDocument();
  });

  it("records an independent administrator decision through the typed contract", async () => {
    const client = api();
    client.portfolioRetentionDispositions.mockResolvedValue({
      items: [{ id: "request-1", transition_id: transition.id, portfolio_id: "portfolio-1", status: "submitted", evidence: { statement: "Termen împlinit", reference: "PV-2026-09" }, requested_by_subject: "owner-sub", requested_at: "2026-09-12T10:00:00Z", operation_attempts: 0 }],
      total: 1, page: 1, pageSize: 10,
    });
    render(<PrimeReactProvider><PortfolioRetentionDispositionPanel api={client} portfolioID="portfolio-1" operationID="operation-1" transitions={[]} canSubmit={false} canDecide /></PrimeReactProvider>);
    fireEvent.click(await screen.findByRole("button", { name: "Înregistrează decizia" }));
    const dialog = screen.getByRole("dialog", { name: /^Decizie independentă de retenție$/ });
    const evidence = within(dialog).getByRole("group", { name: "Dovezile solicitării" });
    expect(evidence).toHaveTextContent("owner-sub");
    expect(evidence).toHaveTextContent("Termen împlinit");
    expect(evidence).toHaveTextContent("PV-2026-09");
    fireEvent.change(within(dialog).getByLabelText("Motivul deciziei *"), { target: { value: "  Dovezi verificate independent  " } });
    fireEvent.click(within(dialog).getByRole("button", { name: "Înregistrează decizia" }));
    await waitFor(() => expect(client.decidePortfolioRetentionDisposition).toHaveBeenCalledWith("portfolio-1", "request-1", { approve: true, reason: "Dovezi verificate independent" }));
  });
});
