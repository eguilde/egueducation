import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { PrimeReactProvider } from "@primereact/core/config";
import { describe, expect, it, vi } from "vitest";
import { TeacherPortfolioWorkspace } from "./TeacherPortfolioWorkspace";
import type { EducationApi, OwnPortfolio } from "./types";

const portfolio: OwnPortfolio = { id: "own-1", portfolio_code: "PORT-CD-1", owner_name: "Ana Pop", owner_role: "Profesor", school_year: "2026-2027", status: "draft", section_count: 2, last_updated_on: "2026-09-01", authenticity_declared: true, consent_captured: true, notes: "", custodian: "Școala", institution_id: "institution-1", legal_hold_active: false, retention_period_days: 1095, retention_until: "", transfer_status: "none" };
const document = { id: "document-1", section_code: "I", component_code: "I.1", document_title: "Planificare", description: "Planificare inițială", school_year: "2026-2027", subject_discipline: "Matematică", applicable_class: "IV A", competencies: ["C1"], evidence_type: "document", issued_on: "2026-09-01", added_on: "2026-09-01", chronological_index: 1, sensitive_data: false, file_reference: "archive://11111111-1111-4111-8111-111111111111", notes: "Observație inițială", archive_document_id: "11111111-1111-4111-8111-111111111111", archive_version_id: "version-1", archive_version_no: 1, archive_sha256: "sha256", authenticity_status: "verified", institution_id: "institution-1", portfolio_id: "own-1" };
const declarationEvidence = {
  templates: [
    { declaration_type: "gdpr_information" as const, declaration_version: "gdpr-v1", declaration_text: "Informare GDPR", source_ref: "Sursă", effective_from: "2026-09-01" },
    { declaration_type: "authenticity" as const, declaration_version: "auth-v1", declaration_text: "Declarație autenticitate", source_ref: "Sursă", effective_from: "2026-09-01" },
  ],
  acknowledgements: [
    { id: "ack-1", portfolio_id: "own-1", declaration_type: "gdpr_information" as const, declaration_version: "gdpr-v1", declaration_text: "Informare GDPR", accepted_at: "2026-09-01T10:00:00Z", accepted_by_user_id: "user-1", attestation_method: "authenticated_web_acknowledgement" },
    { id: "ack-2", portfolio_id: "own-1", declaration_type: "authenticity" as const, declaration_version: "auth-v1", declaration_text: "Declarație autenticitate", accepted_at: "2026-09-01T10:00:00Z", accepted_by_user_id: "user-1", attestation_method: "authenticated_web_acknowledgement" },
  ],
};
const api = (): EducationApi => ({
  portfolioLifecycleOperation: vi.fn(), retryPortfolioLifecycleOperation: vi.fn(),
  portfolioLifecycleOperations: vi.fn(),
  portfolioRetentionDispositions: vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 10 }), submitPortfolioRetentionDisposition: vi.fn(), decidePortfolioRetentionDisposition: vi.fn(),
  exportOwnPortfolio: vi.fn(),
  ownPortfolioAppliedProcedure: vi.fn(),
  uploadOwnPortfolioArchiveDocument: vi.fn(),
  governanceDashboard: vi.fn(), directorCockpit: vi.fn(), eligibleGovernanceUsers: vi.fn(), governanceMeetings: vi.fn(), governanceMeetingDetail: vi.fn(), saveGovernanceMeeting: vi.fn(), deleteGovernanceMeeting: vi.fn(), records: vi.fn(), recordDetail: vi.fn(), createRecord: vi.fn(), updateRecord: vi.fn(), deleteRecord: vi.fn(), recordPdf: vi.fn(), relatedRecords: vi.fn(), relatedDetail: vi.fn(), saveRelated: vi.fn(), deleteRelated: vi.fn(), relatedPdf: vi.fn(), portfolioDocuments: vi.fn(), portfolioDocument: vi.fn(), portfolioDocumentVersions: vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 20 }), createPortfolioDocument: vi.fn(), updatePortfolioDocument: vi.fn(), deletePortfolioDocument: vi.fn(), portfolioChecklist: vi.fn(), portfolioChecklistItem: vi.fn(), createPortfolioChecklistItem: vi.fn(), updatePortfolioChecklistItem: vi.fn(), deletePortfolioChecklistItem: vi.fn(), portfolioOpis: vi.fn(), portfolioOpisEntry: vi.fn(), createPortfolioOpisEntry: vi.fn(), updatePortfolioOpisEntry: vi.fn(), deletePortfolioOpisEntry: vi.fn(), portfolioCustody: vi.fn(), portfolioCustodyEvent: vi.fn(), createPortfolioCustodyEvent: vi.fn(), updatePortfolioCustodyEvent: vi.fn(), deletePortfolioCustodyEvent: vi.fn(), portfolioReviews: vi.fn(), portfolioReview: vi.fn(), returnPortfolioForCorrections: vi.fn(), recordPortfolioManagerialDecision: vi.fn(), portfolioTransferHistory: vi.fn(), portfolioValorifications: vi.fn(), portfolioValorification: vi.fn(), createPortfolioValorification: vi.fn(), updatePortfolioValorification: vi.fn(), deletePortfolioValorification: vi.fn(), portfolioSections: vi.fn().mockResolvedValue({ items: [{ id: "section-1", section_code: "I", component_code: "I.1", label_ro: "Proiectare didactică", sensitive_data: false, active: true, label_en: "", required: true, retention_rule: "", example_documents: [], sort_order: 1 }], total: 1, page: 1, pageSize: 100 }), educationRequirements: vi.fn(), taxonomyCatalog: vi.fn(), metadata: vi.fn(), command: vi.fn(),
  ownPortfolios: vi.fn().mockResolvedValue({ items: [portfolio], total: 1, page: 1, pageSize: 1 }), ownPortfolio: vi.fn().mockResolvedValue(portfolio), createOwnPortfolio: vi.fn(), updateOwnPortfolio: vi.fn().mockResolvedValue(portfolio), submitOwnPortfolio: vi.fn().mockResolvedValue({ ...portfolio, status: "submitted" }), ownPortfolioDeclarations: vi.fn().mockResolvedValue(declarationEvidence), acknowledgeOwnPortfolioDeclaration: vi.fn(), portfolioProcedures: vi.fn(), portfolioProcedure: vi.fn(), createPortfolioProcedure: vi.fn(), updatePortfolioProcedure: vi.fn(), portfolioProcedureRules: vi.fn(), replacePortfolioProcedureRules: vi.fn(), transitionPortfolioProcedure: vi.fn(), ownPortfolioRelated: vi.fn().mockResolvedValue({ items: [document], total: 1, page: 1, pageSize: 1 }), regenerateOwnPortfolioOpis: vi.fn().mockResolvedValue(undefined), recordPortfolioCessation: vi.fn(), setPortfolioLegalHold: vi.fn(), createOwnPortfolioDocument: vi.fn().mockResolvedValue(document), updateOwnPortfolioDocument: vi.fn().mockResolvedValue(document), ownPortfolioDocumentVersions: vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 20 }), deleteOwnPortfolioDocument: vi.fn().mockResolvedValue(undefined), ownPortfolioArchiveDocuments: vi.fn().mockResolvedValue({ items: [{ id: "11111111-1111-4111-8111-111111111111", title: "Dovadă eArhivă", current_version_no: 1 }], total: 1, page: 1, pageSize: 1 }), attachmentGrants: vi.fn(), eligibleAttachmentDocuments: vi.fn(), eligibleAttachmentUsers: vi.fn(), createAttachmentGrant: vi.fn(), deleteAttachmentGrant: vi.fn(), createPortfolioExportManifest: vi.fn(),
});

describe("TeacherPortfolioWorkspace", () => {
  it("keeps one procedure and download panel across repeated portfolio changes", async () => {
    const transport = api();
    const next = { ...portfolio, id: "own-2", school_year: "2027-2028", portfolio_code: "PORT-NEXT" };
    vi.mocked(transport.ownPortfolios).mockResolvedValue({ items: [portfolio, next], total: 2, page: 1, pageSize: 20 });
    vi.mocked(transport.ownPortfolio).mockImplementation(async id => id === next.id ? next : portfolio);
    render(<PrimeReactProvider><TeacherPortfolioWorkspace api={transport} canManageOwn canExportOwn /></PrimeReactProvider>);
    for (const year of [next.school_year, portfolio.school_year, next.school_year]) {
      fireEvent.click(await screen.findByRole("button", { name: new RegExp(year) }));
      await waitFor(() => expect(screen.getAllByRole("button", { name: "Procedura aplicată" })).toHaveLength(1));
      expect(screen.getAllByRole("button", { name: "Descarcă portofoliul" })).toHaveLength(1);
    }
  });
  it("allows an authorized read-only teacher to download without exposing modification controls", async () => {
    render(<PrimeReactProvider><TeacherPortfolioWorkspace api={api()} canManageOwn={false} canExportOwn /></PrimeReactProvider>);
    expect(await screen.findByRole("button", { name: "Descarcă portofoliul" })).toBeEnabled();
    expect(screen.queryByRole("button", { name: "Editează ciorna" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Adaugă document" })).not.toBeInTheDocument();
  });

  it("does not infer export authorization from permission to edit", async () => {
    render(<PrimeReactProvider><TeacherPortfolioWorkspace api={api()} canManageOwn canExportOwn={false} /></PrimeReactProvider>);
    await screen.findByRole("button", { name: "Editează ciorna" });
    expect(screen.queryByRole("button", { name: "Descarcă portofoliul" })).not.toBeInTheDocument();
  });
  it("renders portfolio status and declaration labels with the installed PrimeReact API", async () => {
    render(<PrimeReactProvider><TeacherPortfolioWorkspace api={api()} canManageOwn /></PrimeReactProvider>);
    expect((await screen.findAllByText("Ciornă")).length).toBeGreaterThan(0);
    expect((await screen.findAllByText("Confirmată")).length).toBe(2);
    for (const name of ["Adaugă document", "Istoric versiuni Planificare", "Editează Planificare", "Șterge Planificare"]) {
      expect((await screen.findByRole("button", { name })).querySelector("svg")).not.toBeNull();
    }
  });

  it("edits only owner content without resending server-owned record fields", async () => {
    const transport = api();
    render(<PrimeReactProvider><TeacherPortfolioWorkspace api={transport} canManageOwn /></PrimeReactProvider>);
    fireEvent.click(await screen.findByRole("button", { name: "Editează ciorna" }));
    await screen.findByRole("dialog", { name: "Portofoliu profesional" });
    fireEvent.change(screen.getByLabelText("Observații"), { target: { value: "Corectat" } });
    fireEvent.click(screen.getByRole("button", { name: "Salvează ciorna" }));
    await waitFor(() => expect(transport.updateOwnPortfolio).toHaveBeenCalledWith("own-1", {
      school_year: portfolio.school_year,
      last_updated_on: portfolio.last_updated_on,
      notes: "Corectat",
    }));
  });

  it("prefills and saves editable document metadata through the owner-scoped patch contract", async () => {
    const transport = api();
    render(<PrimeReactProvider><TeacherPortfolioWorkspace api={transport} canManageOwn /></PrimeReactProvider>);
    fireEvent.click(await screen.findByRole("button", { name: "Editează Planificare" }));
    expect(await screen.findByRole("dialog", { name: "Editează document în portofoliu" })).toBeInTheDocument();
    expect(screen.getByLabelText("Descriere pedagogică *")).toHaveValue("Planificare inițială");
    expect(screen.getByLabelText("Observații")).toHaveValue("Observație inițială");
    fireEvent.change(screen.getByLabelText("Descriere pedagogică *"), { target: { value: "Planificare corectată" } });
    fireEvent.change(screen.getByLabelText("Observații"), { target: { value: "Corectat după observațiile evaluatorului." } });
    fireEvent.click(screen.getByRole("button", { name: "Salvează modificările" }));
    await waitFor(() => expect(transport.updateOwnPortfolioDocument).toHaveBeenCalledWith("own-1", "document-1", {
      section_code: "I", component_code: "I.1", document_title: "Planificare", description: "Planificare corectată", school_year: "2026-2027", subject_discipline: "Matematică", applicable_class: "IV A", competencies: ["C1"], evidence_type: "document", issued_on: "2026-09-01", added_on: "2026-09-01", chronological_index: 1, sensitive_data: false, file_reference: "archive://11111111-1111-4111-8111-111111111111", notes: "Corectat după observațiile evaluatorului.",
    }));
  });

  it("hides document editing for a read-only portfolio state", async () => {
    const transport = api();
    transport.ownPortfolios = vi.fn().mockResolvedValue({ items: [{ ...portfolio, status: "submitted" }], total: 1, page: 1, pageSize: 1 });
    render(<PrimeReactProvider><TeacherPortfolioWorkspace api={transport} canManageOwn /></PrimeReactProvider>);
    await screen.findByText("Portofoliul este în verificare instituțională.");
    expect(screen.queryByRole("button", { name: "Editează Planificare" })).not.toBeInTheDocument();
  });

  it("explains the missing school procedure instead of blaming teacher input", async () => {
    const transport = api();
    transport.ownPortfolios = vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 20 });
    transport.createOwnPortfolio = vi.fn().mockRejectedValue(new Error("education_request_422", { cause: "education_portfolio_published_procedure_required" }));
    render(<PrimeReactProvider><TeacherPortfolioWorkspace api={transport} canManageOwn /></PrimeReactProvider>);
    fireEvent.click(await screen.findByRole("button", { name: "Portofoliu nou" }));
    fireEvent.change(screen.getByLabelText("An școlar *"), { target: { value: "2026-2027" } });
    fireEvent.click(screen.getByRole("button", { name: "Salvează ciorna" }));
    expect(await screen.findByText(/Școala nu are încă o procedură de portofoliu publicată/)).toBeInTheDocument();
  });

  it("renders only own-portfolio filter and sort controls that the server contract forwards", async () => {
    const transport = api();
    render(<PrimeReactProvider><TeacherPortfolioWorkspace api={transport} canManageOwn /></PrimeReactProvider>);
    await screen.findByText("PORT-CD-1");
    const documentsCard = screen.getByText("Documente").closest(".p-card") as HTMLElement | null;
    const checklistCard = screen.getByText("Checklist").closest(".p-card") as HTMLElement | null;
    if (!documentsCard || !checklistCard) throw new Error("missing related-record cards");

    // The owner document route forwards the same named pedagogical metadata
    // fields as the institutional document list.
    expect(within(documentsCard).getByLabelText("Filtru Secțiune")).toBeInTheDocument();
    expect(within(documentsCard).getByLabelText("Filtru Componentă")).toBeInTheDocument();
    expect(within(documentsCard).getByLabelText("Filtru Document")).toBeInTheDocument();
    expect(within(documentsCard).getByRole("button", { name: "Sortează după Componentă" })).toBeInTheDocument();
    expect(within(documentsCard).getByRole("button", { name: "Sortează după Document" })).toBeInTheDocument();

    // Checklist exposes only requirement_code filtering; mandatory and the
    // reviewing person are server output, not own-route query fields.
    expect(within(checklistCard).getByLabelText("Filtru Cod")).toBeInTheDocument();
    expect(within(checklistCard).queryByLabelText("Filtru Obligatoriu")).not.toBeInTheDocument();
    expect(within(checklistCard).queryByLabelText("Filtru Verificat de")).not.toBeInTheDocument();
    expect(within(checklistCard).queryByRole("button", { name: "Sortează după Obligatoriu" })).not.toBeInTheDocument();

    fireEvent.change(within(documentsCard).getByLabelText("Filtru Secțiune"), { target: { value: "S1" } });
    await waitFor(() => expect(transport.ownPortfolioRelated).toHaveBeenCalledWith("own-1", "documents", expect.objectContaining({ filters: { section_code: "S1" } })));
    fireEvent.click(within(documentsCard).getByRole("button", { name: "Sortează după Document" }));
    await waitFor(() => expect(transport.ownPortfolioRelated).toHaveBeenCalledWith("own-1", "documents", expect.objectContaining({ sort: "document_title", direction: "asc" })));
  }, 15_000);

  it("shows only the authenticated teacher's portfolio and uses owner-scoped actions", async () => {
    const transport = api();
    render(<PrimeReactProvider><TeacherPortfolioWorkspace api={transport} canManageOwn /></PrimeReactProvider>);
    expect(await screen.findByText("PORT-CD-1")).toBeInTheDocument();
    expect(screen.getByText(/Ana Pop/)).toBeInTheDocument();
    expect((await screen.findAllByText("Planificare")).length).toBeGreaterThan(0);
    expect(transport.ownPortfolios).toHaveBeenCalledWith(expect.objectContaining({ pageSize: 100, sort: "school_year" }));
    expect(transport.ownPortfolioRelated).toHaveBeenCalledWith("own-1", "documents", expect.objectContaining({ page: 1, pageSize: 10 }));
    vi.mocked(transport.ownPortfolioDocumentVersions).mockResolvedValue({ items: [{ version_no: 1, change_type: "create", changed_by: "Ana Pop", changed_at: "2026-09-01T10:00:00Z", reason: "inițial", snapshot: {} }], total: 1, page: 1, pageSize: 20 });
    fireEvent.click(screen.getByRole("button", { name: "Istoric versiuni Planificare" }));
    const history = await screen.findByRole("dialog", { name: "Istoric versiuni — Planificare" });
    await waitFor(() => expect(transport.ownPortfolioDocumentVersions).toHaveBeenCalledWith("own-1", "document-1", expect.objectContaining({ page: 1, pageSize: 20, sort: "version_no", direction: "desc" })));
    expect(within(history).getByText("Ana Pop")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Închide istoricul versiunilor" }));
    fireEvent.click(screen.getByRole("button", { name: "Șterge Planificare" }));
    expect(transport.deleteOwnPortfolioDocument).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Confirmă eliminarea" }));
    await waitFor(() => expect(transport.deleteOwnPortfolioDocument).toHaveBeenCalledWith("own-1", "document-1"));
    fireEvent.click(screen.getByRole("button", { name: "Regenerare opis" }));
    await waitFor(() => expect(transport.regenerateOwnPortfolioOpis).toHaveBeenCalledWith("own-1"));
    fireEvent.click(screen.getByRole("button", { name: "Adaugă document" }));
    const dialog = screen.getByRole("dialog", { name: "Adaugă document în portofoliu" });
    expect(screen.getByText(/serverul fixează proveniența/i)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("combobox", { name: "Componentă din catalog" }));
    fireEvent.click(await screen.findByRole("option", { name: /Proiectare didactică/ }));
    fireEvent.change(screen.getByLabelText("Titlu *"), { target: { value: "Dovadă" } });
    fireEvent.change(screen.getByLabelText("Tip dovadă *"), { target: { value: "adeverință" } });
    fireEvent.change(screen.getByLabelText("Descriere pedagogică *"), { target: { value: "Activitate la clasă" } });
    fireEvent.change(screen.getByLabelText("Disciplina *"), { target: { value: "Matematică" } });
    fireEvent.change(screen.getByLabelText("Clasa aplicabilă *"), { target: { value: "IV A" } });
    fireEvent.change(screen.getByLabelText("Competențe *"), { target: { value: "C1, C2" } });
    fireEvent.click(screen.getByRole("combobox", { name: "Document eArhivă autorizat" }));
    fireEvent.click(await screen.findByRole("option", { name: /Dovadă eArhivă/ }));
    fireEvent.click(within(dialog).getByRole("button", { name: "Adaugă document" }));
    await waitFor(() => expect(transport.createOwnPortfolioDocument).toHaveBeenCalledWith("own-1", expect.objectContaining({ file_reference: "archive://11111111-1111-4111-8111-111111111111", description: "Activitate la clasă", school_year: "2026-2027", subject_discipline: "Matematică", applicable_class: "IV A", competencies: ["C1", "C2"] })));
    fireEvent.click(screen.getByRole("button", { name: "Trimite spre verificare" }));
    fireEvent.click(screen.getByRole("button", { name: "Confirmă trimiterea" }));
    await waitFor(() => expect(transport.submitOwnPortfolio).toHaveBeenCalledWith("own-1"));
    expect(transport.records).not.toHaveBeenCalled();
  }, 15_000);

  it("shows the server review instruction and correction measures for a returned portfolio", async () => {
    const transport = api();
    const returned = { ...portfolio, status: "returned", transfer_status: "neinițiat" };
    const review = { id: "review-1", institution_id: "institution-1", portfolio_id: "own-1", review_code: "REV-1", review_stage: "completeness", outcome: "returned", reviewer_name: "Director", reviewed_on: "2026-10-01", notes: "Adăugați planificarea anuală și corectați opisul.", missing_documents: 2, compliance_score: 64 };
    transport.ownPortfolios = vi.fn().mockResolvedValue({ items: [returned], total: 1, page: 1, pageSize: 1 });
    transport.ownPortfolioRelated = vi.fn(async (_portfolioID: string, resource: Parameters<EducationApi["ownPortfolioRelated"]>[1]) => ({ items: resource === "reviews" ? [review] : [], total: resource === "reviews" ? 1 : 0, page: 1, pageSize: 10 }));
    render(<PrimeReactProvider><TeacherPortfolioWorkspace api={transport} canManageOwn /></PrimeReactProvider>);
    expect(await screen.findByText(/Consultați observațiile evaluatorului/i)).toBeInTheDocument();
    expect(await screen.findByText(/Adăugați planificarea anuală/i)).toBeInTheDocument();
    expect(screen.getByText(/Documente lipsă: 2/)).toBeInTheDocument();
    expect(screen.getByText(/Scor conformitate: 64/)).toBeInTheDocument();
    await waitFor(() => expect(transport.ownPortfolioRelated).toHaveBeenCalledWith("own-1", "reviews", expect.objectContaining({ page: 1, pageSize: 10 })));
  });
});
