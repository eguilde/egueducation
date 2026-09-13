import { act, fireEvent, render, screen } from "@testing-library/react";
import { PrimeReactProvider } from "@primereact/core/config";
import { describe, expect, it, vi } from "vitest";
import { PortfolioLifecycleDialog, type PortfolioLifecycleDraft } from "./PortfolioLifecycleDialog";

const draft: PortfolioLifecycleDraft = { record: { id: "portfolio-1" }, kind: "cessation", date: "", reason: "", active: false };
describe("PortfolioLifecycleDialog", () => {
  it("requires an explicit cessation date and reason", async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    const onClose = vi.fn();
    render(<PrimeReactProvider><PortfolioLifecycleDialog value={draft} onSubmit={onSubmit} onClose={onClose} /></PrimeReactProvider>);
    expect(screen.getByLabelText("Data încetării *")).toHaveValue("");
    expect(screen.getByRole("button", { name: "Confirmă" })).toBeDisabled();
    fireEvent.change(screen.getByLabelText("Data încetării *"), { target: { value: "2026-09-12" } });
    fireEvent.change(screen.getByLabelText("Motiv *"), { target: { value: "  Decizie verificată  " } });
    await act(async () => fireEvent.click(screen.getByRole("button", { name: "Confirmă" })));
    expect(onSubmit).toHaveBeenCalledWith({ ...draft, date: "2026-09-12", reason: "Decizie verificată" });
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("requires a reason for releasing a legal hold too", () => {
    render(<PrimeReactProvider><PortfolioLifecycleDialog value={{ ...draft, kind: "legal_hold", active: false }} onSubmit={vi.fn()} onClose={vi.fn()} /></PrimeReactProvider>);
    const dialog = screen.getByRole("dialog", { name: "Ridică blocarea juridică" });
    expect(document.getElementById(dialog.getAttribute("aria-labelledby")!)).toHaveTextContent(/^Ridică blocarea juridică$/);
    expect(screen.queryByLabelText("Data încetării *")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Confirmă" })).toBeDisabled();
  });

  it("prevents duplicate submissions and keeps the dialog open on failure", async () => {
    let reject!: (error: Error) => void;
    const onSubmit = vi.fn(() => new Promise<void>((_resolve, fail) => { reject = fail; }));
    const onClose = vi.fn();
    render(<PrimeReactProvider><PortfolioLifecycleDialog value={{ ...draft, date: "2026-09-12", reason: "Decizie" }} onSubmit={onSubmit} onClose={onClose} /></PrimeReactProvider>);
    fireEvent.click(screen.getByRole("button", { name: "Confirmă" }));
    fireEvent.click(screen.getByRole("button", { name: "Se trimite solicitarea…" }));
    expect(onSubmit).toHaveBeenCalledOnce();
    expect(screen.getByRole("button", { name: "Renunță" })).toBeDisabled();
    await act(async () => reject(new Error("failed")));
    expect(screen.getByText(/Solicitarea nu a putut fi confirmată/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Confirmă" })).toBeEnabled();
    expect(onClose).not.toHaveBeenCalled();
  });
});
