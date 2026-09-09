import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { PrimeReactProvider } from "@primereact/core/config";
import { describe, expect, it, vi } from "vitest";
import { PortfolioArchiveGrantManager } from "./PortfolioArchiveGrantManager";
import type { EducationApi } from "./types";

const grant = {
  id: "33333333-3333-4333-8333-333333333333",
  archive_document_id: "11111111-1111-4111-8111-111111111111",
  document_title: "CV profesor",
  grantee_user_id: "22222222-2222-4222-8222-222222222222",
  grantee_name: "Ana Pop",
  created_at: "2026-09-09T09:30:00Z",
};

function makeApi({ grants = [grant] }: { grants?: typeof grant[] } = {}) {
  return {
    eligibleAttachmentDocuments: vi.fn().mockResolvedValue({ items: [{ id: grant.archive_document_id, title: "CV profesor", current_version_no: 2 }], total: 1, page: 1, pageSize: 20 }),
    eligibleAttachmentUsers: vi.fn().mockResolvedValue({ items: [{ id: grant.grantee_user_id, name: "Ana Pop" }], total: 1, page: 1, pageSize: 20 }),
    attachmentGrants: vi.fn().mockResolvedValue({ items: grants, total: grants.length, page: 1, pageSize: 20 }),
    createAttachmentGrant: vi.fn().mockResolvedValue(grant),
    deleteAttachmentGrant: vi.fn().mockResolvedValue(undefined),
  } as unknown as EducationApi;
}

function renderManager(api: EducationApi) {
  return render(<PrimeReactProvider><PortfolioArchiveGrantManager api={api} /></PrimeReactProvider>);
}

describe("PortfolioArchiveGrantManager", () => {
  it("loads the grant list with an explicit server query and sends header filters to the API", async () => {
    const api = makeApi();
    renderManager(api);
    await screen.findByText("CV profesor");
    await waitFor(() => expect(api.attachmentGrants).toHaveBeenCalledWith(expect.objectContaining({ page: 1, pageSize: 20, filters: {} })));
    fireEvent.change(screen.getByLabelText("Filtru Beneficiar"), { target: { value: "Ana" } });
    await waitFor(() => expect(api.attachmentGrants).toHaveBeenLastCalledWith(expect.objectContaining({ page: 1, filters: expect.objectContaining({ grantee_name: "Ana" }) })), { timeout: 1200 });
  });

  it("keeps the Add action available in the action header when there are no grants", async () => {
    const api = makeApi({ grants: [] });
    renderManager(api);
    const add = await screen.findByRole("button", { name: "Adaugă drept de atașare" });
    fireEvent.click(add);
    expect(await screen.findByText("Acordă acces la document eArhivă")).toBeInTheDocument();
    await waitFor(() => expect(api.eligibleAttachmentDocuments).toHaveBeenCalledWith(expect.objectContaining({ page: 1, pageSize: 20 })));
    await waitFor(() => expect(api.eligibleAttachmentUsers).toHaveBeenCalledWith(expect.objectContaining({ page: 1, pageSize: 20 })));
  });

  it("requires confirmation before revoking an active grant", async () => {
    const api = makeApi();
    renderManager(api);
    await screen.findByText("Ana Pop");
    fireEvent.click(screen.getByRole("button", { name: "Revocă dreptul pentru Ana Pop" }));
    expect(await screen.findByText("Confirmați revocarea")).toBeInTheDocument();
    expect(api.deleteAttachmentGrant).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Revocă dreptul" }));
    await waitFor(() => expect(api.deleteAttachmentGrant).toHaveBeenCalledWith(grant.id));
    expect(await screen.findByText("Dreptul de atașare a fost revocat.")).toBeInTheDocument();
  });
});
