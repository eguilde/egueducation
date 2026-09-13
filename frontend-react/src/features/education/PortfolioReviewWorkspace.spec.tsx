import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { PrimeReactProvider } from "@primereact/core";
import { PortfolioReviewWorkspace } from "./PortfolioReviewWorkspace";
import type { EducationApi } from "./types";

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((resolvePromise) => { resolve = resolvePromise; });
  return { promise, resolve };
}

const makeApi = () => ({
  recordDetail: vi.fn().mockImplementation(async (_domain: string, id: string) => ({ id })),
  records: vi.fn().mockResolvedValue({ items: [{ id: "p1", owner_name: "Ana Profesor", status: "submitted", school_year: "2026-2027" }], total: 1, page: 1, pageSize: 20 }),
  portfolioReviews: vi.fn().mockResolvedValue({ items: [{ id: "r1", review_stage: "verificare_secretariat", outcome: "completari", reviewer_name: "Secretariat", notes: "Lipsește dovada.", reviewed_on: "2026-09-11" }], total: 1, page: 1, pageSize: 20 }),
  returnPortfolioForCorrections: vi.fn().mockResolvedValue({ id: "p1" }),
  recordPortfolioManagerialDecision: vi.fn().mockResolvedValue({ id: "p1" }),
} as unknown as EducationApi);

describe("PortfolioReviewWorkspace", () => {
  it("opens persisted lifecycle history for a reader without enabling management", async () => {
    const api = makeApi();
    api.portfolioLifecycleOperations = vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 20 });
    render(<PrimeReactProvider><PortfolioReviewWorkspace api={api} canManage={false} canReturn={false} /></PrimeReactProvider>);
    fireEvent.click(await screen.findByRole("button", { name: "Detalii" }));
    fireEvent.click(await screen.findByRole("button", { name: "Istoric operații de protecție" }));
    expect(await screen.findByRole("dialog", { name: "Istoric operații de protecție" })).toBeInTheDocument();
    await screen.findByText("0 operații");
    expect(api.portfolioLifecycleOperations).toHaveBeenCalledWith("p1", expect.objectContaining({ page: 1 }));
    expect(screen.queryByRole("button", { name: "Reia verificarea" })).not.toBeInTheDocument();
  });
  it("does not offer lifecycle mutations when the authoritative detail failed to load", async () => {
    const api = makeApi();
    vi.mocked(api.recordDetail).mockRejectedValue(new Error("unavailable"));
    render(<PrimeReactProvider><PortfolioReviewWorkspace api={api} canManage canReturn canManageLifecycle /></PrimeReactProvider>);
    fireEvent.click(await screen.findByRole("button", { name: "Detalii" }));
    expect(await screen.findByText("Detaliul portofoliului sau istoricul verificării nu a putut fi încărcat.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Aplică blocare juridică" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Înregistrează încetarea activității" })).not.toBeInTheDocument();
  });

  it("loads legal protection from the selected detail, not the list summary", async () => {
    const api = makeApi();
    vi.mocked(api.recordDetail).mockResolvedValue({ id: "p1", portfolio_code: "PORT-1", owner_name: "Ana Profesor", owner_role: "Profesor", school_year: "2026-2027", status: "validated", section_count: 5, last_updated_on: "2026-09-12", authenticity_declared: true, consent_captured: true, notes: "", custodian: "Școala", institution_id: "institution-1", legal_hold_active: true, activity_ceased_on: "2026-09-12", retention_period_days: 1095, retention_until: "2029-09-11", transfer_status: "none" });
    render(<PrimeReactProvider><PortfolioReviewWorkspace api={api} canManage canReturn canManageLifecycle /></PrimeReactProvider>);
    fireEvent.click(await screen.findByRole("button", { name: "Detalii" }));
    expect(await screen.findByRole("button", { name: "Ridică blocarea juridică" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Înregistrează încetarea activității" })).toBeDisabled();
    expect(api.recordDetail).toHaveBeenCalledWith("portfolios", "p1");
  });
  it("requires the separate school management permission for lifecycle actions", async () => {
    const api = makeApi();
    const view = render(<PrimeReactProvider><PortfolioReviewWorkspace api={api} canManage canReturn /></PrimeReactProvider>);
    fireEvent.click(await screen.findByRole("button", { name: "Detalii" }));
    await screen.findByText("Lipsește dovada.");
    expect(screen.queryByRole("button", { name: "Înregistrează încetarea activității" })).not.toBeInTheDocument();
    view.rerender(<PrimeReactProvider><PortfolioReviewWorkspace api={api} canManage canReturn canManageLifecycle /></PrimeReactProvider>);
    fireEvent.click(screen.getByRole("button", { name: "Înregistrează încetarea activității" }));
    expect(await screen.findByRole("dialog", { name: "Înregistrează încetarea activității" })).toBeInTheDocument();
    expect(screen.getByLabelText("Data încetării *")).toHaveValue("");
    view.rerender(<PrimeReactProvider><PortfolioReviewWorkspace api={api} canManage canReturn canManageLifecycle={false} /></PrimeReactProvider>);
    await waitFor(() => expect(screen.queryByRole("dialog", { name: "Înregistrează încetarea activității" })).not.toBeInTheDocument());
  });

  it("loads the server-side list and renders review history as read-only", async () => {
    const api = makeApi();
    render(<PrimeReactProvider><PortfolioReviewWorkspace api={api} canManage canReturn /></PrimeReactProvider>);
    await screen.findByText("Ana Profesor");
    fireEvent.click(await screen.findByRole("button", { name: "Detalii" }));
    expect(await screen.findByText("Lipsește dovada.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Editează|Șterge|Adaugă/ })).not.toBeInTheDocument();
  });

  it("uses the dedicated correction command for a correction-only user", async () => {
    const api = makeApi();
    render(<PrimeReactProvider><PortfolioReviewWorkspace api={api} canManage={false} canReturn /></PrimeReactProvider>);
    await screen.findByText("Ana Profesor");
    fireEvent.click(await screen.findByRole("button", { name: "Detalii" }));
    fireEvent.click(await screen.findByRole("button", { name: "Returnează pentru completări" }));
    fireEvent.change(screen.getByRole("textbox", { name: "Observații" }), { target: { value: "Lipsește dovada." } });
    fireEvent.click(screen.getByRole("button", { name: "Returnează" }));
    await waitFor(() => expect(api.returnPortfolioForCorrections).toHaveBeenCalledWith("p1", expect.objectContaining({ notes: "Lipsește dovada." })));
    expect(screen.queryByRole("button", { name: "Decizie managerială" })).not.toBeInTheDocument();
  });

  it("uses the dedicated managerial decision command only for managers", async () => {
    const api = makeApi();
    render(<PrimeReactProvider><PortfolioReviewWorkspace api={api} canManage canReturn={false} /></PrimeReactProvider>);
    await screen.findByText("Ana Profesor");
    fireEvent.click(await screen.findByRole("button", { name: "Detalii" }));
    fireEvent.click(await screen.findByRole("button", { name: "Decizie managerială" }));
    fireEvent.change(screen.getByRole("textbox", { name: "Observații" }), { target: { value: "Dosar complet." } });
    fireEvent.click(screen.getByRole("button", { name: "Înregistrează decizia" }));
    await waitFor(() => expect(api.recordPortfolioManagerialDecision).toHaveBeenCalledWith("p1", expect.objectContaining({ outcome: "acceptat", notes: "Dosar complet." })));
    expect(screen.queryByRole("button", { name: "Returnează pentru completări" })).not.toBeInTheDocument();
  });

  it("sends header filters through the supported server filter contract", async () => {
    const api = makeApi();
    render(<PrimeReactProvider><PortfolioReviewWorkspace api={api} canManage canReturn /></PrimeReactProvider>);
    await screen.findByText("Ana Profesor");
    fireEvent.change(screen.getByRole("textbox", { name: "Filtrează Profesor" }), { target: { value: "Ana" } });
    await waitFor(() => expect(api.records).toHaveBeenLastCalledWith("portfolios", expect.objectContaining({ page: 1, filters: { owner_name: "Ana" } })));
    fireEvent.change(screen.getByRole("textbox", { name: "Filtrează Stare" }), { target: { value: "submitted" } });
    await waitFor(() => expect(api.records).toHaveBeenLastCalledWith("portfolios", expect.objectContaining({ filters: { owner_name: "Ana", status: "submitted" } })));
  });

  it("does not render reviews from a slower, previously selected portfolio", async () => {
    const firstReviews = deferred<Awaited<ReturnType<EducationApi["portfolioReviews"]>>>();
    const secondReviews = deferred<Awaited<ReturnType<EducationApi["portfolioReviews"]>>>();
    const api = {
      ...makeApi(),
      records: vi.fn().mockResolvedValue({ items: [
        { id: "p1", owner_name: "Ana Profesor", status: "submitted", school_year: "2026-2027" },
        { id: "p2", owner_name: "Bogdan Profesor", status: "submitted", school_year: "2026-2027" },
      ], total: 2, page: 1, pageSize: 20 }),
      portfolioReviews: vi.fn((id: string) => id === "p1" ? firstReviews.promise : secondReviews.promise),
    } as unknown as EducationApi;
    render(<PrimeReactProvider><PortfolioReviewWorkspace api={api} canManage canReturn /></PrimeReactProvider>);
    await screen.findByText("Bogdan Profesor");
    const detailButtons = screen.getAllByRole("button", { name: "Detalii" });
    fireEvent.click(detailButtons[0]);
    fireEvent.click(detailButtons[1]);
    secondReviews.resolve({ items: [{ id: "r2", portfolio_id: "p2", review_code: "REV-2", review_stage: "managerial", outcome: "acceptat", reviewer_name: "Director", notes: "Review B", reviewed_on: "2026-09-12", missing_documents: 0, compliance_score: 100, institution_id: "institution-1" }], total: 1, page: 1, pageSize: 20 });
    expect(await screen.findByText("Review B")).toBeInTheDocument();
    firstReviews.resolve({ items: [{ id: "r1", portfolio_id: "p1", review_code: "REV-1", review_stage: "verificare_secretariat", outcome: "completari", reviewer_name: "Secretariat", notes: "Review A", reviewed_on: "2026-09-11", missing_documents: 1, compliance_score: 75, institution_id: "institution-1" }], total: 1, page: 1, pageSize: 20 });
    await waitFor(() => expect(screen.queryByText("Review A")).not.toBeInTheDocument());
  });

  it("uses the transition response as the selected portfolio's authoritative state", async () => {
    const api = {
      ...makeApi(),
      returnPortfolioForCorrections: vi.fn().mockResolvedValue({ id: "p1", owner_name: "Ana Profesor", status: "returned", school_year: "2026-2027" }),
    } as unknown as EducationApi;
    render(<PrimeReactProvider><PortfolioReviewWorkspace api={api} canManage={false} canReturn /></PrimeReactProvider>);
    await screen.findByText("Ana Profesor");
    fireEvent.click(await screen.findByRole("button", { name: "Detalii" }));
    fireEvent.click(await screen.findByRole("button", { name: "Returnează pentru completări" }));
    fireEvent.change(screen.getByRole("textbox", { name: "Observații" }), { target: { value: "Lipsește dovada." } });
    fireEvent.click(screen.getByRole("button", { name: "Returnează" }));
    await waitFor(() => expect(api.returnPortfolioForCorrections).toHaveBeenCalled());
    await waitFor(() => expect(screen.getByLabelText("Stare portofoliu selectat: returned")).toBeInTheDocument());
  });
});
