import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { PrimeReactProvider } from "@primereact/core/config";
import { describe, expect, it, vi } from "vitest";
import { PortfolioOwnUpload } from "./PortfolioOwnUpload";
import type { OwnPortfolioArchiveUpload } from "./types";
import uploadResponse from "./portfolio-upload-response.fixture.json";

describe("PortfolioOwnUpload", () => {
  it("discards a previous portfolio upload response and resets the form on navigation", async () => {
    let resolve!: (value: OwnPortfolioArchiveUpload) => void;
    const uploadOwnPortfolioArchiveDocument = vi.fn(() => new Promise<OwnPortfolioArchiveUpload>(done => { resolve = done; }));
    const ownPortfolioArchiveDocuments = vi.fn().mockResolvedValue({ items: [], total: 0 });
    const api = { uploadOwnPortfolioArchiveDocument, ownPortfolioArchiveDocuments };
    const onReady = vi.fn();
    const view = render(<PrimeReactProvider><PortfolioOwnUpload portfolioID="portfolio-1" api={api} onReady={onReady} /></PrimeReactProvider>);
    fireEvent.change(view.container.querySelector('input[type="file"]')!, { target: { files: [new File(["%PDF"], "Old.pdf", { type: "application/pdf" })] } });
    fireEvent.click(screen.getByRole("button", { name: "Încarcă PDF propriu" }));
    view.rerender(<PrimeReactProvider><PortfolioOwnUpload portfolioID="portfolio-2" api={api} onReady={onReady} /></PrimeReactProvider>);
    await act(async () => resolve({ ...uploadResponse, id: "old-archive" }));
    expect(screen.getByLabelText("Titlu PDF încărcat")).toHaveValue("");
    expect(screen.getByRole("button", { name: "Încarcă PDF propriu" })).toBeDisabled();
    expect(screen.queryByText(/PDF primit/)).not.toBeInTheDocument();
    expect(ownPortfolioArchiveDocuments).not.toHaveBeenCalled();
    expect(onReady).not.toHaveBeenCalled();
  });

  it.each([
    ["archive_idempotency_conflict", /Datele diferă de încărcarea inițială/],
    ["portfolio_upload_authority_changed", /Nu mai aveți dreptul/],
  ])("explains %s without claiming the document was uploaded", async (code, message) => {
    const uploadOwnPortfolioArchiveDocument = vi.fn().mockRejectedValue(new Error("education_request_409", { cause: code }));
    const ownPortfolioArchiveDocuments = vi.fn();
    const { container } = render(<PrimeReactProvider><PortfolioOwnUpload portfolioID="portfolio-1" api={{ uploadOwnPortfolioArchiveDocument, ownPortfolioArchiveDocuments }} onReady={vi.fn()} /></PrimeReactProvider>);
    fireEvent.change(container.querySelector('input[type="file"]')!, { target: { files: [new File(["%PDF"], "Plan.pdf", { type: "application/pdf" })] } });
    fireEvent.click(screen.getByRole("button", { name: "Încarcă PDF propriu" }));
    expect(await screen.findByText(message)).toBeInTheDocument();
    expect(ownPortfolioArchiveDocuments).not.toHaveBeenCalled();
    if (code === "portfolio_upload_authority_changed") {
      const blocked = screen.getByRole("button", { name: "Încărcare indisponibilă" });
      expect(blocked).toBeDisabled();
      fireEvent.click(blocked);
      expect(uploadOwnPortfolioArchiveDocument).toHaveBeenCalledTimes(1);
    }
  });

  it("keeps the retry key and waits for processing before exposing the document", async () => {
    const uploadOwnPortfolioArchiveDocument = vi.fn()
      .mockRejectedValueOnce(new Error("connection lost"))
      .mockResolvedValue({ id: "archive-1" });
    const ownPortfolioArchiveDocuments = vi.fn().mockResolvedValue({ items: [], total: 0 });
    const onReady = vi.fn();
    const { container } = render(<PrimeReactProvider><PortfolioOwnUpload portfolioID="portfolio-1" api={{ uploadOwnPortfolioArchiveDocument, ownPortfolioArchiveDocuments }} onReady={onReady} /></PrimeReactProvider>);
    const input = container.querySelector('input[type="file"]');
    expect(input).not.toBeNull();
    const file = new File(["%PDF-1.7"], "Planificare.pdf", { type: "application/pdf" });
    fireEvent.change(input!, { target: { files: [file] } });
    fireEvent.change(screen.getByLabelText("Titlu PDF încărcat"), { target: { value: "  Planificare  " } });
    fireEvent.click(screen.getByRole("button", { name: "Încarcă PDF propriu" }));
    fireEvent.click(await screen.findByRole("button", { name: "Reîncearcă încărcarea PDF" }));
    await screen.findByText(/PDF primit/);
    expect(uploadOwnPortfolioArchiveDocument).toHaveBeenCalledTimes(2);
    const first = uploadOwnPortfolioArchiveDocument.mock.calls[0];
    expect(first[0]).toBe("portfolio-1");
    expect(first[1]).toEqual({ file, title: "Planificare" });
    expect(first[2]).toBeTruthy();
    expect(uploadOwnPortfolioArchiveDocument.mock.calls[1][2]).toBe(first[2]);
    expect(onReady).not.toHaveBeenCalled();
    await waitFor(() => expect(ownPortfolioArchiveDocuments).toHaveBeenCalledWith(expect.objectContaining({ filters: { title: "Planificare" } })));
    const document = { id: "archive-1", title: "Planificare", current_version_no: 1 };
    ownPortfolioArchiveDocuments.mockResolvedValue({ items: [document], total: 1 });
    fireEvent.click(screen.getByRole("button", { name: "Verifică starea" }));
    await screen.findByText(/PDF-ul este disponibil/);
    expect(onReady).toHaveBeenCalledWith([document]);
    expect(uploadOwnPortfolioArchiveDocument).toHaveBeenCalledTimes(2);
  });
});
