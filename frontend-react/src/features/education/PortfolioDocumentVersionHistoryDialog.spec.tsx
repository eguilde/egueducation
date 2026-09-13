import { fireEvent, render, screen } from "@testing-library/react";
import { PrimeReactProvider } from "@primereact/core/config";
import { describe, expect, it, vi } from "vitest";
import { PortfolioDocumentVersionHistoryDialog } from "./PortfolioDocumentVersionHistoryDialog";

describe("PortfolioDocumentVersionHistoryDialog", () => {
  it("provides a labelled close control with a visible icon and closes the dialog", async () => {
    const onClose = vi.fn();
    const load = vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 20 });
    render(<PrimeReactProvider><PortfolioDocumentVersionHistoryDialog title="Istoric document" load={load} onClose={onClose} /></PrimeReactProvider>);
    await screen.findByText("Nu există versiuni pentru filtrul curent.");
    const close = screen.getByRole("button", { name: "Închide istoricul versiunilor" });
    expect(close.querySelector("svg")).not.toBeNull();
    fireEvent.click(close);
    expect(onClose).toHaveBeenCalledOnce();
  });
});
