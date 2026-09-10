import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { PrimeReactProvider } from "@primereact/core";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ContractClient } from "../../api/client";
import { SchoolReportsWorkspace, SignedArtifactEvidenceWorkspace } from "./SchoolTrustWorkspaces";

const response = <T,>(data: T, status = 200) => ({ data, response: new Response(null, { status }) });

function reportClient() {
  const GET = vi.fn((path: string) => {
    if (path === "/api/education/reports") return Promise.resolve(response({ items: [{ code: "portfolio-status", label: "Situația portofoliilor", description: "Starea portofoliilor.", formats: ["json", "csv", "pdf"], columns: [{ key: "owner_name", label: "Titular" }, { key: "status", label: "Stare" }] }] }));
    if (path.endsWith("/csv") || path.endsWith("/pdf")) return Promise.resolve(response(new Blob(["report"])));
    return Promise.resolve(response({ items: [{ portfolio_code: "PF-1", owner_name: "Ana Pop", owner_role: "profesor", school_year: "2026-2027", status: "active", transfer_status: "none", last_updated_on: "2026-09-10T08:00:00Z" }], total: 21, page: 1, pageSize: 20 }));
  });
  return { GET } as unknown as ContractClient & { GET: ReturnType<typeof vi.fn> };
}

function signatureClient() {
  const evidence = { id: "evidence-1", artifact_id: "artifact-1", artifact_type: "decision", document_sha256: "a".repeat(64), signature_format: "PAdES", signature_level: "qualified", signature_subject: "CN=Test", certificate_issuer: "Issuer", certificate_serial: "123", certificate_valid_from: "2026-01-01T00:00:00Z", certificate_valid_until: "2027-01-01T00:00:00Z", storage_document_id: "document-1", storage_version_id: "version-1", storage_bucket: "earhive", storage_object_key: "tenant/document.pdf", submitted_at: "2026-09-10T08:00:00Z", submitted_by_subject: "test-user", latest_validation: { id: "validation-1", findings: {}, trusted_list_provider: "not-configured", validated_at: "2026-09-10T08:00:01Z", validated_by_subject: "test-user", validation_status: "error" } };
  const GET = vi.fn((path: string) => {
    if (path.endsWith("eligible-artifacts")) return Promise.resolve(response({ items: [{ id: "artifact-1", type: "decision", label: "DEC-1 · Hotărâre" }], total: 1, page: 1, pageSize: 50 }));
    if (path.endsWith("eligible-archive-versions")) return Promise.resolve(response({ items: [{ document_id: "document-1", version_id: "version-1", title: "Hotărâre semnată", version_no: 2, document_sha256: "a".repeat(64), storage_bucket: "earhive", storage_object_key: "tenant/document.pdf" }], total: 1, page: 1, pageSize: 50 }));
    return Promise.resolve(response({ items: [evidence], total: 1, page: 1, pageSize: 20 }));
  });
  const POST = vi.fn((path: string) => Promise.resolve(path.endsWith("/revalidate") ? response(evidence.latest_validation, 201) : response(evidence, 201)));
  return { GET, POST } as unknown as ContractClient & { GET: ReturnType<typeof vi.fn>; POST: ReturnType<typeof vi.fn> };
}

describe("School trust workspaces", () => {
  beforeEach(() => {
    vi.stubGlobal("URL", { ...URL, createObjectURL: vi.fn(() => "blob:report"), revokeObjectURL: vi.fn() });
    vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => undefined);
  });
  afterEach(() => { cleanup(); vi.restoreAllMocks(); vi.unstubAllGlobals(); });

  it("loads, filters, sorts and paginates a report through literal server contracts", async () => {
    const client = reportClient();
    render(<PrimeReactProvider><SchoolReportsWorkspace client={client} canExport={false} /></PrimeReactProvider>);
    expect(await screen.findByText("Ana Pop")).toBeInTheDocument();
    expect(client.GET).toHaveBeenCalledWith("/api/education/reports");
    expect(client.GET).toHaveBeenCalledWith("/api/education/reports/{reportCode}", expect.objectContaining({ params: expect.objectContaining({ path: { reportCode: "portfolio-status" }, query: expect.objectContaining({ page: 1, pageSize: 20 }) }) }));
    fireEvent.change(screen.getByLabelText("Filtru Titular"), { target: { value: "Ana" } });
    await waitFor(() => expect(client.GET).toHaveBeenLastCalledWith("/api/education/reports/{reportCode}", expect.objectContaining({ params: expect.objectContaining({ query: expect.objectContaining({ "filter.owner_name": "Ana" }) }) })), { timeout: 1500 });
    fireEvent.click(screen.getByRole("button", { name: "Sortează după Titular" }));
    await waitFor(() => expect(client.GET).toHaveBeenLastCalledWith("/api/education/reports/{reportCode}", expect.objectContaining({ params: expect.objectContaining({ query: expect.objectContaining({ sort: "owner_name", direction: "asc" }) }) })), { timeout: 1500 });
    fireEvent.click(screen.getByRole("button", { name: "Următor" }));
    await waitFor(() => expect(client.GET).toHaveBeenLastCalledWith("/api/education/reports/{reportCode}", expect.objectContaining({ params: expect.objectContaining({ query: expect.objectContaining({ page: 2 }) }) })), { timeout: 1500 });
    expect(screen.getByRole("button", { name: "Exportă PDF" })).toBeDisabled();
  });

  it("exports only when the sensitive-export capability is present", async () => {
    const client = reportClient();
    render(<PrimeReactProvider><SchoolReportsWorkspace client={client} canExport /></PrimeReactProvider>);
    await screen.findByText("Ana Pop");
    fireEvent.click(screen.getByRole("button", { name: "Exportă PDF" }));
    await waitFor(() => expect(client.GET).toHaveBeenCalledWith("/api/education/reports/{reportCode}/pdf", expect.objectContaining({ parseAs: "blob" })));
  });

  it("submits signature evidence using server-selected artifact and archive provenance", async () => {
    const client = signatureClient();
    render(<PrimeReactProvider><SignedArtifactEvidenceWorkspace client={client} canManage canValidate /></PrimeReactProvider>);
    await screen.findByText("CN=Test");
    fireEvent.click(screen.getByRole("button", { name: "Înregistrează dovadă" }));
    const dialog = await screen.findByRole("dialog");
    await waitFor(() => expect(client.GET).toHaveBeenCalledWith("/api/education/signatures/eligible-artifacts", expect.anything()), { timeout: 1500 });
    await waitFor(() => expect(client.GET).toHaveBeenCalledWith("/api/education/signatures/eligible-archive-versions", expect.anything()), { timeout: 1500 });
    fireEvent.click(within(dialog).getByLabelText("Artefact eligibil *"));
    fireEvent.click(await screen.findByText("DEC-1 · Hotărâre"));
    fireEvent.click(within(dialog).getByLabelText("Versiune eArhivă *"));
    fireEvent.click(await screen.findByText("Hotărâre semnată · v2"));
    fireEvent.change(within(dialog).getByLabelText("Subiect certificat"), { target: { value: "CN=Director" } });
    fireEvent.change(within(dialog).getByLabelText("Emitent certificat"), { target: { value: "Qualified CA" } });
    fireEvent.change(within(dialog).getByLabelText("Serie certificat"), { target: { value: "SER-42" } });
    fireEvent.change(within(dialog).getByLabelText("Certificat valabil de la"), { target: { value: "2026-01-01T08:00" } });
    fireEvent.change(within(dialog).getByLabelText("Certificat valabil până la"), { target: { value: "2027-01-01T08:00" } });
    fireEvent.click(within(dialog).getByRole("button", { name: "Înregistrează dovada" }));
    await waitFor(() => expect(client.POST).toHaveBeenCalledWith("/api/education/signatures", { body: expect.objectContaining({ artifact_type: "decision", artifact_id: "artifact-1", storage_document_id: "document-1", storage_version_id: "version-1", signature_format: "PAdES", signature_level: "qualified", signature_subject: "CN=Director" }) }), { timeout: 1500 });
    const body = client.POST.mock.calls.find(([path]) => path === "/api/education/signatures")?.[1]?.body;
    expect(body).not.toHaveProperty("document_sha256");
    expect(body).not.toHaveProperty("storage_bucket");
    expect(body).not.toHaveProperty("storage_object_key");
  });

  it("hides mutation controls from read-only users", async () => {
    const client = signatureClient();
    render(<PrimeReactProvider><SignedArtifactEvidenceWorkspace client={client} canManage={false} canValidate={false} /></PrimeReactProvider>);
    await screen.findByText("CN=Test");
    expect(screen.queryByRole("button", { name: "Înregistrează dovadă" })).not.toBeInTheDocument();
    expect(screen.queryByText("Revalidează")).not.toBeInTheDocument();
  });
});
