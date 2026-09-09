import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { PrimeReactProvider } from "@primereact/core/config";
import { describe, expect, it, vi } from "vitest";
import { TeacherPortfolioWorkspace } from "./TeacherPortfolioWorkspace";
import type { EducationApi } from "./types";

const portfolio = { id: "own-1", portfolio_code: "PORT-CD-1", owner_name: "Ana Pop", owner_role: "Profesor", school_year: "2026-2027", status: "draft", section_count: 2, last_updated_on: "2026-09-01", authenticity_declared: true, consent_captured: true, notes: "" };
const api = (): EducationApi => ({
  governanceDashboard: vi.fn(), directorCockpit: vi.fn(), eligibleGovernanceUsers: vi.fn(), dashboardAt: vi.fn(), governanceMeetings: vi.fn(), governanceMeetingDetail: vi.fn(), saveGovernanceMeeting: vi.fn(), deleteGovernanceMeeting: vi.fn(), records: vi.fn(), recordDetail: vi.fn(), saveRecord: vi.fn(), deleteRecord: vi.fn(), recordPdf: vi.fn(), relatedRecords: vi.fn(), relatedDetail: vi.fn(), saveRelated: vi.fn(), deleteRelated: vi.fn(), relatedPdf: vi.fn(), exportFile: vi.fn(), metadata: vi.fn(), command: vi.fn(),
  ownPortfolios: vi.fn().mockResolvedValue({ items: [portfolio], total: 1, page: 1, pageSize: 1 }), ownPortfolio: vi.fn().mockResolvedValue(portfolio), createOwnPortfolio: vi.fn(), updateOwnPortfolio: vi.fn().mockResolvedValue(portfolio), submitOwnPortfolio: vi.fn().mockResolvedValue({ ...portfolio, status: "submitted" }), ownPortfolioRelated: vi.fn().mockResolvedValue({ items: [{ id: "document-1", document_title: "Planificare", section_code: "S1", file_reference: "archive://document-1" }], total: 1, page: 1, pageSize: 1 }), regenerateOwnPortfolioOpis: vi.fn().mockResolvedValue(undefined), createOwnPortfolioDocument: vi.fn().mockResolvedValue({ id: "document-1" }), deleteOwnPortfolioDocument: vi.fn().mockResolvedValue(undefined), ownPortfolioArchiveDocuments: vi.fn().mockResolvedValue({ items: [{ id: "11111111-1111-4111-8111-111111111111", title: "Dovadă eArhivă", current_version_no: 1 }], total: 1, page: 1, pageSize: 1 }), attachmentGrants: vi.fn(), eligibleAttachmentDocuments: vi.fn(), eligibleAttachmentUsers: vi.fn(), createAttachmentGrant: vi.fn(), deleteAttachmentGrant: vi.fn(),
});

describe("TeacherPortfolioWorkspace", () => {
  it("shows only the authenticated teacher's portfolio and uses owner-scoped actions", async () => {
    const transport = api();
    render(<PrimeReactProvider><TeacherPortfolioWorkspace api={transport} canManageOwn /></PrimeReactProvider>);
    expect(await screen.findByText("PORT-CD-1")).toBeInTheDocument();
    expect(screen.getByText(/Ana Pop/)).toBeInTheDocument();
    expect((await screen.findAllByText("Planificare")).length).toBeGreaterThan(0);
    fireEvent.click(screen.getByRole("button", { name: "Regenerează opisul" }));
    await waitFor(() => expect(transport.regenerateOwnPortfolioOpis).toHaveBeenCalledWith("own-1"));
    fireEvent.click(screen.getByRole("button", { name: "Adaugă document" }));
    const dialog = screen.getByRole("dialog", { name: "Adaugă document în portofoliu" });
    expect(screen.getByText(/serverul verifică existența/i)).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Secțiune *"), { target: { value: "I" } });
    fireEvent.change(screen.getByLabelText("Componentă *"), { target: { value: "I.1" } });
    fireEvent.change(screen.getByLabelText("Titlu *"), { target: { value: "Dovadă" } });
    fireEvent.change(screen.getByLabelText("Tip dovadă *"), { target: { value: "adeverință" } });
    fireEvent.click(screen.getByRole("combobox", { name: "Document eArhivă autorizat" }));
    fireEvent.click(await screen.findByRole("option", { name: /Dovadă eArhivă/ }));
    fireEvent.click(within(dialog).getByRole("button", { name: "Adaugă document" }));
    await waitFor(() => expect(transport.createOwnPortfolioDocument).toHaveBeenCalledWith("own-1", expect.objectContaining({ file_reference: "archive://11111111-1111-4111-8111-111111111111" })));
    fireEvent.click(screen.getByRole("button", { name: "Trimite spre verificare" }));
    await waitFor(() => expect(transport.submitOwnPortfolio).toHaveBeenCalledWith("own-1"));
    expect(transport.records).not.toHaveBeenCalled();
  });
});
