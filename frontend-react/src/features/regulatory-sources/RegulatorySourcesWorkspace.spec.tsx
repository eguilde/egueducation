import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { PrimeReactProvider } from "@primereact/core/config";
import { primeTheme } from "../../components/ThemeMenu";
import { RegulatorySourcesWorkspace } from "./RegulatorySourcesWorkspace";
import type { RegulatorySourcesApi } from "./api";

const source = { id: "source-1", citation: "Legea educației", publisher_url: "https://example.test/lege", issuer: "Minister", source_kind: "law" as const, applicable_from: null, applicable_until: null, status: "draft" as const, expected_version: 3, created_by_subject: "another", created_at: "2026-01-01T00:00:00Z", updated_at: "2026-01-02T00:00:00Z", latest_evidence_id: null, latest_evidence_sha256: null, latest_evidence_retrieved_at: null, activation_evidence_id: null, activated_by_subject: null, activated_at: null, assessment: null };
function api(): RegulatorySourcesApi { return { list: vi.fn().mockResolvedValue({ items: [source], total: 1, page: 1, pageSize: 20 }), register: vi.fn().mockResolvedValue(source), verify: vi.fn().mockResolvedValue({ ...source, status: "verified" }), activate: vi.fn(), evidence: vi.fn() }; }

describe("RegulatorySourcesWorkspace", () => {
  it("submits only the register contract fields through real PrimeReact controls", async () => {
    const subject = api(); render(<PrimeReactProvider {...primeTheme}><RegulatorySourcesWorkspace api={subject} capabilities={{ read: true, manage: true, verify: true, activate: true }} actorSubject="operator" /></PrimeReactProvider>);
    await screen.findByText("Legea educației"); fireEvent.click(screen.getByRole("button", { name: /Adaugă/i })); expect(screen.getByRole("dialog", { name: "Înregistrează sursă" })).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Citare"), { target: { value: "Ordin nou" } }); fireEvent.change(screen.getByLabelText("URL publicare"), { target: { value: "https://example.test/ordin" } }); fireEvent.change(screen.getByLabelText("Emitent"), { target: { value: "Minister" } }); fireEvent.change(screen.getByLabelText("Aplicabilă de la"), { target: { value: "2026-01-01" } });
    fireEvent.click(screen.getByRole("button", { name: "Înregistrează" }));
    await waitFor(() => expect(subject.register).toHaveBeenCalledWith(expect.objectContaining({ citation: "Ordin nou", publisher_url: "https://example.test/ordin", issuer: "Minister", source_kind: "law", applicable_from: "2026-01-01" })));
    expect(subject.register).toHaveBeenCalledWith(expect.not.objectContaining({ expected_version: expect.anything(), id: expect.anything(), latest_evidence_sha256: expect.anything() }));
  });

  it("shows verify only for a draft when the verifier capability is present", async () => {
    const subject = api(); render(<PrimeReactProvider {...primeTheme}><RegulatorySourcesWorkspace api={subject} capabilities={{ read: true, manage: false, verify: true, activate: false }} /></PrimeReactProvider>);
    await screen.findByText("Legea educației"); fireEvent.click(screen.getByRole("button", { name: /Verifică/i }));
    await waitFor(() => expect(subject.verify).toHaveBeenCalledWith("source-1", 3));
  });

  it("prefills immutable evidence and activates an eligible verified source", async () => {
    const verified = { ...source, status: "verified" as const, latest_evidence_id: "evidence-1" };
    const subject = api(); subject.list = vi.fn().mockResolvedValue({ items: [verified], total: 1, page: 1, pageSize: 20 }); subject.activate = vi.fn().mockResolvedValue({ ...verified, status: "active" });
    render(<PrimeReactProvider {...primeTheme}><RegulatorySourcesWorkspace api={subject} capabilities={{ read: true, manage: false, verify: false, activate: true }} actorSubject="approver" /></PrimeReactProvider>);
    await screen.findByText("Legea educației"); fireEvent.click(screen.getByRole("button", { name: "Activează" }));
    expect(screen.getByLabelText("ID evidență")).toHaveValue("evidence-1"); expect(screen.getByLabelText("ID evidență")).toHaveAttribute("readonly");
    fireEvent.change(screen.getByLabelText("Evaluare aplicabilitate"), { target: { value: "Aplicabilă instituției pentru anul curent." } }); fireEvent.click(screen.getAllByRole("button", { name: "Activează" })[1]);
    await waitFor(() => expect(subject.activate).toHaveBeenCalledWith("source-1", { expected_version: 3, evidence_id: "evidence-1", assessment: "Aplicabilă instituției pentru anul curent." }));
  });

  it("forwards the citation filter as a server query", async () => {
    const subject = api(); render(<PrimeReactProvider {...primeTheme}><RegulatorySourcesWorkspace api={subject} capabilities={{ read: true, manage: false, verify: false, activate: false }} actorSubject="operator" /></PrimeReactProvider>);
    await screen.findByText("Legea educației"); fireEvent.change(screen.getByLabelText("Filtru citare"), { target: { value: "ordin" } });
    await waitFor(() => expect(subject.list).toHaveBeenLastCalledWith(expect.objectContaining({ page: 1, citation: "ordin", sort: "updated_at", direction: "desc" })));
  });
});
