import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { PrimeReactProvider } from "@primereact/core";
import { describe, expect, it, vi } from "vitest";
import { PortfolioAppliedProcedurePanel, type OwnPortfolioAppliedProcedure } from "./PortfolioAppliedProcedurePanel";

const applied = (title = "Procedura portofoliului") : OwnPortfolioAppliedProcedure => ({
  procedure: { id: "procedure-1", procedure_code: "PROC-PORT", title, source_ref: "Decizia CA nr. 12/2026", lifecycle_status: "published", version_no: 3, effective_from: "2026-09-01", effective_to: undefined, calendar_rules: { submission: { deadline: "2026-10-01" } }, access_rules: { teacher: ["read", "write"] }, accepted_formats: { extensions: ["pdf", "docx"] }, retention_rules: { years: 5 }, transfer_rules: { permitted: false }, created_at: "2026-08-01T10:00:00Z", created_by_user_id: "user-1", institution_id: "institution-1", updated_at: "2026-08-02T10:00:00Z" },
  rules: [{ id: "rule-1", procedure_id: "procedure-1", section_code: "I", label_ro: "Proiectare didactică", label_en: "Planning", required: true, active: true, sort_order: 1, source_catalog_version: "2026.1" }],
});

function deferred<T>() { let resolve!: (value: T) => void; const promise = new Promise<T>((resolvePromise) => { resolve = resolvePromise; }); return { promise, resolve }; }

describe("PortfolioAppliedProcedurePanel", () => {
  it("does not open a procedure while the portfolio selection is being saved", () => {
    const load = vi.fn();
    render(<PrimeReactProvider><PortfolioAppliedProcedurePanel portfolioID="portfolio-1" load={load} disabled /></PrimeReactProvider>);
    const button = screen.getByRole("button", { name: "Procedura aplicată" });
    expect(button).toBeDisabled();
    fireEvent.click(button);
    expect(load).not.toHaveBeenCalled();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("loads and renders the applied procedure, semantic rule content, and immutable section rules", async () => {
    const load = vi.fn().mockResolvedValue(applied());
    render(<PrimeReactProvider><PortfolioAppliedProcedurePanel portfolioID="portfolio-1" load={load} /></PrimeReactProvider>);
    fireEvent.click(screen.getByRole("button", { name: "Procedura aplicată" }));
    expect(await screen.findByRole("dialog", { name: "Procedura aplicată portofoliului" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Închide procedura aplicată" }).querySelector("svg")).not.toBeNull();
    expect(await screen.findByText("Decizia CA nr. 12/2026")).toBeInTheDocument();
    expect(screen.getByText("Calendar")).toBeInTheDocument();
    expect(screen.getByText("Formate acceptate")).toBeInTheDocument();
    expect(screen.getByText("Proiectare didactică")).toBeInTheDocument();
    expect(screen.getByText("Secțiunile prevăzute de procedura școlii.")).toBeInTheDocument();
    expect(load).toHaveBeenCalledWith("portfolio-1");
  });

  it("explains when an older portfolio has no applied procedure", async () => {
    const load = vi.fn().mockRejectedValue(new Error("education_request_404", { cause: "education_own_portfolio_procedure_not_applied" }));
    render(<PrimeReactProvider><PortfolioAppliedProcedurePanel portfolioID="portfolio-legacy" load={load} /></PrimeReactProvider>);
    fireEvent.click(screen.getByRole("button", { name: "Procedura aplicată" }));
    expect(await screen.findByText("Acest portofoliu mai vechi nu are încă o procedură aplicată.")).toBeInTheDocument();
  });

  it("ignores a stale procedure response after the selected portfolio changes", async () => {
    const first = deferred<OwnPortfolioAppliedProcedure>();
    const second = deferred<OwnPortfolioAppliedProcedure>();
    const load = vi.fn((portfolioID: string) => portfolioID === "portfolio-1" ? first.promise : second.promise);
    const view = render(<PrimeReactProvider><PortfolioAppliedProcedurePanel portfolioID="portfolio-1" load={load} /></PrimeReactProvider>);
    fireEvent.click(screen.getByRole("button", { name: "Procedura aplicată" }));
    view.rerender(<PrimeReactProvider><PortfolioAppliedProcedurePanel portfolioID="portfolio-2" load={load} /></PrimeReactProvider>);
    second.resolve(applied("Procedura nouă"));
    expect(await screen.findByText("Procedura nouă")).toBeInTheDocument();
    first.resolve(applied("Procedura veche"));
    await waitFor(() => expect(screen.queryByText("Procedura veche")).not.toBeInTheDocument());
  });
});
