import { act, fireEvent, render, screen } from "@testing-library/react";
import { PrimeReactProvider } from "@primereact/core/config";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { PortfolioDownloadPanel } from "./PortfolioDownloadPanel";

describe("PortfolioDownloadPanel", () => {
  const createObjectURL = vi.fn(() => "blob:portfolio");
  beforeEach(() => {
    createObjectURL.mockClear();
    vi.stubGlobal("URL", class extends URL {
      static createObjectURL = createObjectURL;
      static revokeObjectURL = vi.fn();
    });
  });
  afterEach(() => { vi.restoreAllMocks(); vi.unstubAllGlobals(); });

  it("downloads the own portfolio once using a safe filename and an abort signal", async () => {
    const blob = new Blob(["ZIP"], { type: "application/zip" });
    const download = vi.fn().mockResolvedValue(blob);
    let filename = "";
    vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(function (this: HTMLAnchorElement) { filename = this.download; });
    render(<PrimeReactProvider><PortfolioDownloadPanel portfolioID="p-1" portfolioCode="PORT/2026" canExport download={download} /></PrimeReactProvider>);
    fireEvent.click(screen.getByRole("button", { name: "Descarcă portofoliul" }));
    expect(await screen.findByText("Descărcarea a fost pornită.")).toBeInTheDocument();
    expect(download).toHaveBeenCalledWith("p-1", expect.any(AbortSignal));
    expect(createObjectURL).toHaveBeenCalledWith(blob);
    expect(filename).toBe("PORT_2026.zip");
  });

  it("aborts and discards a download when the portfolio changes", async () => {
    let resolve!: (blob: Blob) => void;
    const download = vi.fn((_id: string, _signal?: AbortSignal) => new Promise<Blob>((done) => { resolve = done; }));
    const view = render(<PrimeReactProvider><PortfolioDownloadPanel portfolioID="p-1" portfolioCode="P1" canExport download={download} /></PrimeReactProvider>);
    fireEvent.click(screen.getByRole("button", { name: "Descarcă portofoliul" }));
    const signal = download.mock.calls[0][1] as AbortSignal;
    view.rerender(<PrimeReactProvider><PortfolioDownloadPanel portfolioID="p-2" portfolioCode="P2" canExport download={download} /></PrimeReactProvider>);
    expect(signal.aborted).toBe(true);
    await act(async () => resolve(new Blob(["old"]))) ;
    expect(createObjectURL).not.toHaveBeenCalled();
    expect(screen.queryByText("Descărcarea a fost pornită.")).not.toBeInTheDocument();
  });

  it("explains an integrity failure without offering an incomplete download", async () => {
    const download = vi.fn().mockRejectedValue(new Error("education_request_422", { cause: "education_portfolio_export_provenance_incomplete" }));
    render(<PrimeReactProvider><PortfolioDownloadPanel portfolioID="p-1" portfolioCode="P1" canExport download={download} /></PrimeReactProvider>);
    fireEvent.click(screen.getByRole("button", { name: "Descarcă portofoliul" }));
    expect(await screen.findByText(/integritatea lor nu poate fi confirmată/)).toBeInTheDocument();
    expect(createObjectURL).not.toHaveBeenCalled();
  });

  it("does not offer export when the permission is absent", () => {
    render(<PrimeReactProvider><PortfolioDownloadPanel portfolioID="p-1" portfolioCode="P1" canExport={false} download={vi.fn()} /></PrimeReactProvider>);
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  it("discards an in-flight result when export permission is revoked", async () => {
    let resolve!: (blob: Blob) => void;
    const download = vi.fn((_id: string, _signal?: AbortSignal) => new Promise<Blob>(done => { resolve = done; }));
    const view = render(<PrimeReactProvider><PortfolioDownloadPanel portfolioID="p-1" portfolioCode="P1" canExport download={download} /></PrimeReactProvider>);
    fireEvent.click(screen.getByRole("button", { name: "Descarcă portofoliul" }));
    const signal = download.mock.calls[0][1]!;
    view.rerender(<PrimeReactProvider><PortfolioDownloadPanel portfolioID="p-1" portfolioCode="P1" canExport={false} download={download} /></PrimeReactProvider>);
    expect(signal.aborted).toBe(true);
    await act(async () => resolve(new Blob(["ZIP"])));
    expect(createObjectURL).not.toHaveBeenCalled();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  it("does not issue a request while the selected portfolio is loading", () => {
    const download = vi.fn();
    render(<PrimeReactProvider><PortfolioDownloadPanel portfolioID="p-1" portfolioCode="P1" canExport disabled download={download} /></PrimeReactProvider>);
    const button = screen.getByRole("button", { name: "Descarcă portofoliul" });
    expect(button).toBeDisabled();
    fireEvent.click(button);
    expect(download).not.toHaveBeenCalled();
  });

  it.each([
    ["education_request_413", "education_portfolio_export_bundle_limit_exceeded", /depășește limita/],
    ["education_request_503", "education_portfolio_export_storage_unavailable", /nu este disponibilă momentan/],
    ["education_request_403", undefined, /Nu aveți dreptul/],
  ])("shows a useful error for %s and enables retry", async (message, cause, expected) => {
    const download = vi.fn().mockRejectedValue(new Error(message, { cause }));
    render(<PrimeReactProvider><PortfolioDownloadPanel portfolioID="p-1" portfolioCode="P1" canExport download={download} /></PrimeReactProvider>);
    fireEvent.click(screen.getByRole("button", { name: "Descarcă portofoliul" }));
    expect(await screen.findByText(expected)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Descarcă portofoliul" })).toBeEnabled();
    expect(createObjectURL).not.toHaveBeenCalled();
  });

  it("aborts and discards a result after unmount", async () => {
    let resolve!: (blob: Blob) => void;
    const download = vi.fn((_id: string, _signal?: AbortSignal) => new Promise<Blob>(done => { resolve = done; }));
    const view = render(<PrimeReactProvider><PortfolioDownloadPanel portfolioID="p-1" portfolioCode="P1" canExport download={download} /></PrimeReactProvider>);
    fireEvent.click(screen.getByRole("button", { name: "Descarcă portofoliul" }));
    const signal = download.mock.calls[0][1]!;
    view.unmount();
    expect(signal.aborted).toBe(true);
    await act(async () => resolve(new Blob(["ZIP"])));
    expect(createObjectURL).not.toHaveBeenCalled();
  });
});
